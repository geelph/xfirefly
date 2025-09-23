package runner

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"xfirefly/pkg/output"
	"xfirefly/pkg/types"
	"xfirefly/pkg/utils/common"

	"github.com/donnie4w/go-logger/logger"
)

// getTargets 从命令行参数或文件中读取目标，并进行去重处理
func getTargets(options *types.CmdOptionsType) ([]string, error) {
	// 优先使用命令行直接指定的目标
	if len(options.Target) > 0 {
		// 记录原始目标数
		originalCount := len(options.Target)
		// 移除重复目标
		targets := common.RemoveDuplicateURLs(options.Target)
		// 计算重复目标数
		duplicateCount := originalCount - len(targets)
		logger.Info(fmt.Sprintf("原始目标数量：%v个，重复目标数量：%v个，去重后目标数量：%v个", originalCount, duplicateCount, len(targets)))
		return targets, nil
	}

	// 其次从文件读取（流式扫描，内存占用更低）
	if options.TargetsList == "" {
		return nil, fmt.Errorf("目标文件为空")
	}

	// 读取文件内容
	file, err := os.Open(options.TargetsList)
	if err != nil {
		//logger.Error(fmt.Sprintf("读取目标文件失败: %v", err))
		return nil, fmt.Errorf("读取目标文件失败: %v", err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	// 提升扫描缓存，避免异常长行导致的扫描失败
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	unique := make(map[string]struct{}, 1024)
	totalLines := 0
	for scanner.Scan() {
		// 移除字符串前后空白字符
		line := strings.TrimSpace(scanner.Text())
		// 空行处理
		if line == "" {
			continue
		}
		// 计数
		totalLines++
		if _, ok := unique[line]; !ok {
			unique[line] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Error(fmt.Sprintf("扫描目标文件出错: %v", err))
	}

	targets := make([]string, 0, len(unique))
	for t := range unique {
		targets = append(targets, t)
	}
	// 计算重复目标数量
	duplicateCount := totalLines - len(targets)
	logger.Info(fmt.Sprintf("原始目标数量：%v个，重复目标数量：%v个，去重后目标数量：%v个", totalLines, duplicateCount, len(targets)))

	return targets, nil
}

// ProcessURL 处理单个URL的所有指纹识别，获取目标基础信息并执行指纹识别
func ProcessURL(target string, proxy string, timeout int, _ int) (*TargetResult, error) {
	// 确保目标不为空
	if target == "" {
		return nil, fmt.Errorf("目标URL不能为空")
	}

	// 创建目标结果对象，提前预分配
	targetResult := &TargetResult{
		URL:        target,
		StatusCode: 0,
		Title:      "",
		Server:     types.EmptyServerInfo(),
		Matches:    make([]*FingerMatch, 0, 10), // 预分配容量
		Wappalyzer: nil,
	}

	// 获取目标基础信息
	baseInfoResp, err := GetBaseInfo(target, proxy, timeout)

	// 即使获取基础信息失败，也继续处理
	if err != nil {
		logger.Debug(fmt.Sprintf("获取目标 %s 基础信息失败: %v", target, err))
		return targetResult, nil
	}

	// 更新目标结果对象
	targetResult.StatusCode = baseInfoResp.StatusCode
	targetResult.Title = baseInfoResp.Title
	targetResult.Server = baseInfoResp.Server
	targetResult.Wappalyzer = baseInfoResp.Wappalyzer
	targetResult.URL = baseInfoResp.Url
	logger.Debug(fmt.Sprintf("初始URL：%s", targetResult.URL))

	// 初始化缓存和变量映射
	var variableMap = make(map[string]any, 4) // 预分配map容量
	lastResponse, lastRequest := initializeCache(baseInfoResp, proxy)
	if lastResponse == nil {
		// 如果无法获取响应，直接返回
		return targetResult, nil
	}

	variableMap["request"] = lastRequest
	variableMap["response"] = lastResponse

	targetResult.LastRequest = lastRequest
	targetResult.LastResponse = lastResponse

	UpdateTargetCache(variableMap, targetResult.URL, false)

	// 创建基础信息对象
	baseInfo := &BaseInfo{
		Title:      targetResult.Title,
		Server:     targetResult.Server,
		StatusCode: targetResult.StatusCode,
	}

	// 如果没有指纹规则，直接返回结果
	if len(AllFinger) == 0 {
		return targetResult, nil
	}

	// 执行指纹识别
	matches := runFingerDetection(baseInfoResp.Url, baseInfo, proxy, timeout)
	targetResult.Matches = matches

	// 指纹规则运行完成之后立即删除缓存，减少内存压力
	ClearTargetURLCache(targetResult.URL)

	return targetResult, nil
}

// runFingerDetection 执行指纹识别，使用全局规则池高效处理指纹识别任务
func runFingerDetection(target string, baseInfo *BaseInfo, proxy string, timeout int) []*FingerMatch {
	// 确保全局规则池已初始化
	if !IsRulePoolInitialized() {
		logger.Error("全局规则池未初始化")
		return []*FingerMatch{}
	}

	// 如果没有指纹规则，直接返回（基于快照）
	ruleCount := GetFingerCount()
	if ruleCount == 0 {
		return []*FingerMatch{}
	}

	// 复制快照，避免并发安全隐患
	localFingers := GetAllFingerSnapshot()
	ruleCount = len(localFingers)

	// 结果通道容量限制，避免为大规模规则集分配过大的缓冲
	chanCap := ruleCount
	if chanCap > 512 {
		chanCap = 512
	}
	if chanCap < 1 {
		chanCap = 1
	}
	resultChan := make(chan *FingerMatch, chanCap)

	// 创建等待组
	var wg sync.WaitGroup

	// 记录开始时间用于性能监控
	startTime := time.Now()

	// 统计实际提交的任务数
	submittedTasks := int64(0)

	// 提交所有指纹任务到全局规则池
	for _, fingerprint := range localFingers {
		wg.Add(1)

		task := &RuleTask{
			Target:     target,
			Finger:     fingerprint,
			BaseInfo:   baseInfo,
			Proxy:      proxy,
			Timeout:    timeout,
			ResultChan: resultChan,
			WaitGroup:  &wg,
		}

		// 简化重试机制，只在池满时重试一次
		submitErr := SubmitRuleTask(task)
		if submitErr != nil {
			time.Sleep(1 * time.Millisecond)
			submitErr = SubmitRuleTask(task)
		}

		if submitErr != nil {
			logger.Debug(fmt.Sprintf("提交指纹任务失败: %s, 错误: %v", fingerprint.Id, submitErr))
			wg.Done()
			continue
		}

		submittedTasks++
	}

	// 启动结果收集协程，避免阻塞主流程（仅由单协程写入，无需互斥）
	matches := make([]*FingerMatch, 0, ruleCount/4+1)
	resultDone := make(chan struct{})

	go func() {
		defer close(resultDone)
		for result := range resultChan {
			if result != nil && result.Result {
				matches = append(matches, result)
			}
		}
	}()

	// 等待所有指纹任务完成
	wg.Wait()
	close(resultChan)

	// 等待结果收集完成
	<-resultDone

	// 记录性能信息
	duration := time.Since(startTime)
	logger.Debug(fmt.Sprintf("目标 %s 指纹识别完成，耗时: %v, 匹配数量: %d/%d, 实际任务数: %d",
		target, duration, len(matches), ruleCount, submittedTasks))

	return matches
}

// handleMatchResults 处理匹配结果，将结果输出到终端和文件
func handleMatchResults(targetResult *TargetResult, options *types.CmdOptionsType, printResult func(string), outputFormat string) {
	output.HandleMatchResults(&output.TargetResult{
		URL:        targetResult.URL,
		StatusCode: targetResult.StatusCode,
		Title:      targetResult.Title,
		ServerInfo: targetResult.Server,
		Matches:    convertFingerMatches(targetResult.Matches),
		Wappalyzer: targetResult.Wappalyzer,
	}, options.Output, options.SockOutput, printResult, outputFormat, targetResult.LastResponse)
}

// convertFingerMatches 将pkg.FingerMatch切片转换为output.FingerMatch切片
func convertFingerMatches(matches []*FingerMatch) []*output.FingerMatch {
	result := make([]*output.FingerMatch, len(matches))
	for i, match := range matches {
		result[i] = &output.FingerMatch{
			Finger:   match.Finger,
			Result:   match.Result,
			Request:  match.Request,
			Response: match.Response,
		}
	}
	return result
}

// printSummary 打印汇总信息
func printSummary(targets []string, results map[string]*TargetResult) {
	// 将pkg.TargetResult映射转换为output.TargetResult映射
	outputResults := make(map[string]*output.TargetResult)
	for key, result := range results {
		outputResults[key] = &output.TargetResult{
			URL:        result.URL,
			StatusCode: result.StatusCode,
			Title:      result.Title,
			ServerInfo: result.Server,
			Matches:    convertFingerMatches(result.Matches),
			Wappalyzer: result.Wappalyzer,
		}
	}
	output.PrintSummary(targets, outputResults)
}
