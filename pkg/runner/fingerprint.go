package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	cel2 "xfirefly/pkg/cel"
	"xfirefly/pkg/finger"
	"xfirefly/pkg/types"
	"xfirefly/pkg/utils"
	"xfirefly/pkg/utils/common"
	"xfirefly/pkg/utils/proto"

	"github.com/donnie4w/go-logger/logger"
)

// AllFinger 全局指纹数据
var AllFinger []*finger.Finger

// 用于保护AllFinger的读写锁
var allFingerMutex sync.RWMutex

// GetAllFingerSnapshot 以读锁复制一份只读快照，避免并发读写竞态
func GetAllFingerSnapshot() []*finger.Finger {
	allFingerMutex.RLock()
	defer allFingerMutex.RUnlock()
	if len(AllFinger) == 0 {
		return nil
	}
	snapshot := make([]*finger.Finger, len(AllFinger))
	copy(snapshot, AllFinger)
	return snapshot
}

// LoadFingerprints 加载指纹规则文件，支持从默认嵌入指纹库、指定目录或单个YAML文件加载
func LoadFingerprints(options types.YamlFingerType) error {
	// 指纹数据锁
	allFingerMutex.Lock()
	defer allFingerMutex.Unlock()

	// 清空现有指纹规则
	AllFinger = AllFinger[:0]

	// 使用嵌入式指纹库
	if len(options.FingerYaml) == 0 && options.FingerPath == "" {
		logger.Info("使用默认指纹库")
		// 获取指纹规则
		fin, err := utils.GetFingerYaml()
		if err != nil {
			return err
		}
		AllFinger = fin
		return nil
	}

	// 从目录加载指纹文件
	if options.FingerPath != "" {
		logger.Info(fmt.Sprintf("加载yaml文件目录：%s", options.FingerPath))

		return filepath.WalkDir(options.FingerPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && common.IsYamlFile(path) {
				if poc, err := finger.Read(path); err == nil && poc != nil {
					AllFinger = append(AllFinger, poc)
				}
			}
			return nil
		})
	}

	// 加载单个指纹文件
	if len(options.FingerYaml) != 0 {
		logger.Info(fmt.Sprintf("加载yaml文件：%s", options.FingerYaml))

		for _, fyaml := range options.FingerYaml {
			if !common.IsYamlFile(fyaml) {
				return fmt.Errorf("%s 不是有效的yaml文件", fyaml)
			}

			poc, err := finger.Read(fyaml)
			if err != nil {
				return fmt.Errorf("读取yaml文件出错: %v", err)
			}

			if poc != nil {
				AllFinger = append(AllFinger, poc)
			}
		}
	}

	return nil
}

// GetFingerCount 获取指纹规则数量（线程安全）
func GetFingerCount() int {
	allFingerMutex.RLock()
	defer allFingerMutex.RUnlock()
	return len(AllFinger)
}

// evaluateFingerprintWithCache 使用缓存的基础信息评估指纹规则，执行单个指纹的识别逻辑，包括发送请求和规则评估
func evaluateFingerprintWithCache(fg *finger.Finger, target string, baseInfo *BaseInfo, proxy string, timeout int, fingerActive bool) (*FingerMatch, error) {
	customLib := cel2.NewCustomLib()

	// 初始化变量映射
	resultData := &FingerMatch{
		Finger: fg,
		Result: false, // 默认为false
	}
	varMap := make(map[string]any)

	logger.Debug(fmt.Sprintf("执行指纹识别：%s", fg.Id))

	// 设置基础变量容器（请求/响应会在缓存命中或首次请求后赋值）
	varMap["title"] = baseInfo.Title
	varMap["server"] = baseInfo.Server

	// 初始化响应对象
	varMap["response"] = &proto.Response{
		Status:      baseInfo.StatusCode,
		Headers:     map[string]string{},
		ContentType: "",
		Body:        []byte{},
		Raw:         []byte{},
		RawHeader:   []byte{},
		Url:         &proto.UrlType{},
		Latency:     0,
	}

	// 处理预设规则
	if len(fg.Set) > 0 {
		finger.IsFuzzSet(fg.Set, varMap, customLib)
	}
	if len(fg.Payloads.Payloads) > 0 {
		finger.IsFuzzSet(fg.Payloads.Payloads, varMap, customLib)
	}

	// 评估规则
	for _, rule := range fg.Rules {
		// 提前处理path
		rule.Value.Request.Path = finger.SetVariableMap(strings.TrimSpace(rule.Value.Request.Path), varMap)
		urlStr := common.ParseTarget(target, rule.Value.Request.Path)

		// 主动指纹识别规则区分，优化发包数量，通过参数控制主动发包行为
		if !fingerActive {
			if rule.Value.Request.Path != "" && rule.Value.Request.Path != "/" {
				logger.Info("主动发包的规则键为：%s", rule.Key)
				logger.Info("主动发包的路径为：%s", rule.Value.Request.Path)
				customLib.WriteRuleFunctionsROptions(rule.Key, false)
				continue
			}
		}
		// 检查是否可以使用缓存
		isCache, cache := ShouldUseCache(rule, urlStr)
		logger.Debug(fmt.Sprintf("%s 规则 %s 是否使用缓存：%t", target, rule.Key, isCache))

		if isCache && cache.Request != nil && cache.Response != nil {
			varMap["request"] = cache.Request
			varMap["response"] = cache.Response
		} else {
			// 发送新请求
			newVarMap, err := finger.SendRequest(target, rule.Value.Request, rule.Value, varMap, proxy, timeout)
			if err != nil {
				logger.Debug(fmt.Sprintf("规则 %s 请求失败: %v", rule.Key, err))
				customLib.WriteRuleFunctionsROptions(rule.Key, false)
				continue
			}

			// 更新变量映射
			if len(newVarMap) > 0 {
				varMap = newVarMap
				// 只有头部和body为空的请求才缓存
				if len(rule.Value.Request.Headers) == 0 {
					UpdateTargetCache(varMap, urlStr, rule.Value.Request.FollowRedirects)
				}
			}
		}

		// 安全调试输出（截断大包体，避免日志与内存压力）
		const maxDump = 4096
		if req, ok := varMap["request"].(*proto.Request); ok && req != nil {
			raw := req.Raw
			if len(raw) > maxDump {
				raw = raw[:maxDump]
			}
			logger.Debug(fmt.Sprintf("请求数据包(截断)：\n%s", raw))
		}
		if resp, ok := varMap["response"].(*proto.Response); ok && resp != nil {
			raw := resp.Raw
			if len(raw) > maxDump {
				raw = raw[:maxDump]
			}
			logger.Debug(fmt.Sprintf("响应数据包(截断)：\n%s", raw))
		}
		logger.Debug("开始CEL表达式匹配")

		// 执行规则评估
		result, err := customLib.Evaluate(rule.Value.Expression, varMap)
		if err != nil {
			logger.Debug(fmt.Sprintf("规则 %s CEL解析错误：%s", rule.Key, err.Error()))
			customLib.WriteRuleFunctionsROptions(rule.Key, false)
		} else {
			ruleBool := result.Value().(bool)
			logger.Debug(fmt.Sprintf("规则 %s 评估结果: %v", rule.Value.Expression, ruleBool))
			customLib.WriteRuleFunctionsROptions(rule.Key, ruleBool)
		}

		// 处理输出规则
		if len(rule.Value.Output) > 0 {
			finger.IsFuzzSet(rule.Value.Output, varMap, customLib)
		}
	}

	// 执行最终评估
	result, err := customLib.Evaluate(fg.Expression, varMap)
	if err != nil {
		return resultData, fmt.Errorf("最终表达式解析错误：%v", err)
	}

	resultData.Result = result.Value().(bool)

	// 如果匹配成功，存储请求和响应数据
	if resultData.Result {
		if req, ok := varMap["request"].(*proto.Request); ok {
			resultData.Request = req
		}
		if resp, ok := varMap["response"].(*proto.Response); ok {
			resultData.Response = resp
		}
	}

	logger.Debug(fmt.Sprintf("最终规则 %s 评估结果: %v", fg.Expression, resultData.Result))

	return resultData, nil
}
