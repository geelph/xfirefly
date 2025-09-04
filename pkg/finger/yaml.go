package finger

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"xfirefly/pkg/utils/common"

	"github.com/donnie4w/go-logger/logger"
	"gopkg.in/yaml.v2"
)

// FingerPath 配置poc文件目录
const FingerPath = "fingerprint"
const (
	HttpType = "http"
	TcpType  = "tcp"
	UdpType  = "udp"
	SslType  = "ssl"
	GoType   = "go"
)

var order = 0

type Finger struct {
	Id         string        `yaml:"id"`         //  脚本名称
	Transport  string        `yaml:"transport"`  // 传输方式，该字段用于指定发送数据包的协议，该字段用于指定发送数据包的协议:tcp、udp、http
	Set        yaml.MapSlice `yaml:"set"`        // 全局变量定义，该字段用于定义全局变量。比如随机数，反连平台等
	Payloads   Payloads      `yaml:"payloads"`   // 定义载荷
	Rules      RuleMapSlice  `yaml:"rules"`      // 定义规则
	Expression string        `yaml:"expression"` // 匹配规则
	Info       Info          `yaml:"info"`       // 信息
	Gopoc      string        `yaml:"gopoc"`      // Gopoc 脚本名称
}
type Payloads struct {
	Continue bool          `yaml:"continue"` // 是否继续执行
	Payloads yaml.MapSlice `yaml:"payloads"` // 载荷
}

// RuleMap 用于帮助yaml解析，保证Rule有序
type RuleMap struct {
	Key   string // 规则名称
	Value Rule   // 规则
}

// RuleMapSlice 用于帮助yaml解析，保证Rule有序
type RuleMapSlice []RuleMap

// Rule 类型
type Rule struct {
	Request        RuleRequest   `yaml:"request"`          // 请求
	Expression     string        `yaml:"expression"`       // 匹配规则
	Expressions    []string      `yaml:"expressions"`      // 匹配规则
	Output         yaml.MapSlice `yaml:"output"`           // 输出
	StopIfMatch    bool          `yaml:"stop_if_match"`    // 匹配成功时，是否停止继续匹配
	StopIfMismatch bool          `yaml:"stop_if_mismatch"` // 匹配失败时，是否停止继续匹配
	BeforeSleep    int           `yaml:"before_sleep"`     // 匹配成功时，等待的时间
	order          int           // 规则顺序
}

// RuleRequest 请求结构体
type RuleRequest struct {
	Type            string            `yaml:"type"`             // 传输方式，默认 http，可选：tcp,udp,ssl,go 等任意扩展
	Host            string            `yaml:"host"`             // tcp/udp 请求的主机名
	Data            string            `yaml:"data"`             // tcp/udp 发送的内容
	DataType        string            `yaml:"data-type"`        // tcp/udp 发送的数据类型，默认字符串
	ReadSize        int               `yaml:"read-size"`        // tcp/udp 读取内容的长度
	ReadTimeout     int               `yaml:"read-timeout"`     // tcp/udp专用
	Raw             string            `yaml:"raw"`              // raw 专用
	Method          string            `yaml:"method"`           // http 请求方式
	Path            string            `yaml:"path"`             // http 请求路径
	Headers         map[string]string `yaml:"headers"`          // http 请求头
	Body            string            `yaml:"body"`             // http 请求体
	FollowRedirects bool              `yaml:"follow_redirects"` // 是否跟随重定向，默认跟随重定向
}

// Info 以下开始是 信息部分
type Info struct {
	Name           string         `yaml:"name"`           // 名称
	Author         string         `yaml:"author"`         //  作者
	Severity       string         `yaml:"severity"`       // 漏洞等级
	Verified       bool           `yaml:"verified"`       // 是否验证
	Description    string         `yaml:"description"`    // 描述
	Reference      []string       `yaml:"reference"`      // 参考
	Affected       string         `yaml:"affected"`       // 影响版本
	Solutions      string         `yaml:"solutions"`      // 解决方案
	Tags           string         `yaml:"tags"`           // 标签
	Classification Classification `yaml:"classification"` // 分类
	Created        string         `yaml:"created"`        // 创建时间
}

// Classification 分类
type Classification struct {
	CvssMetrics string  `yaml:"cvss-metrics"` // cvss
	CvssScore   float64 `yaml:"cvss-score"`   // cvss分数
	CveId       string  `yaml:"cve-id"`       // cve
	CweId       string  `yaml:"cwe-id"`       // cwe
}

// ruleAlias 类型
type ruleAlias struct {
	Request        RuleRequest   `yaml:"request"`          // 请求
	Expression     string        `yaml:"expression"`       // 匹配规则
	Expressions    []string      `yaml:"expressions"`      // 匹配规则
	Output         yaml.MapSlice `yaml:"output"`           // 输出
	StopIfMatch    bool          `yaml:"stop_if_match"`    // 匹配成功时，是否停止继续匹配
	StopIfMismatch bool          `yaml:"stop_if_mismatch"` // 匹配失败时，是否停止继续匹配
	BeforeSleep    int           `yaml:"before_sleep"`     // 匹配成功时，等待的时间
}

// Select 获取指定名字的yaml文件位置
func Select(pocPath string, pocName string) (string, error) {
	// 检查路径是否存在
	if pocPath == "" {
		return "", fmt.Errorf("指纹库路径为空")
	}

	// 检查路径是否存在
	if !common.DirIsExist(pocPath) {
		return "", fmt.Errorf("指纹库路径 '%s' 不存在", pocPath)
	}

	var result string
	// 遍历目录中的所有文件和子目录
	err := filepath.WalkDir(pocPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err // 如果遇到错误，立即返回
		}

		// 检查文件是否是 YAML 文件
		if !d.IsDir() && common.IsYamlFile(path) {
			if strings.Contains(d.Name(), pocName) {
				result = path
				return filepath.SkipDir // 找到文件后停止遍历
			}
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if result == "" {
		return "", fmt.Errorf("未找到名为 '%s' 的指纹文件", pocName)
	}

	return result, nil
}

// Load 加载yaml文件
func Load(fileName string, Fingers embed.FS) (*Finger, error) {
	p := &Finger{}
	// 检查文件名是否已经包含fingerprint/前缀，避免重复路径
	filePath := fileName
	if !strings.HasPrefix(fileName, "fingerprint/") {
		filePath = "fingerprint/" + fileName
	}

	// 读取文件内容
	yamlFile, err := Fingers.ReadFile(filePath)

	if err != nil {
		logger.Errorf("读取指纹 %s 时发生错误：%v", filePath, err)
		return nil, err
	}
	// 反序列化yaml文件
	err = yaml.Unmarshal(yamlFile, p)
	if err != nil {
		logger.Errorf("反序列化指纹 %s 时发生错误：%v", filePath, err)
		return nil, err
	}
	return p, err
}

// Read 获取yaml文件内容
func Read(fileName string) (*Finger, error) {
	p := &Finger{}

	file, err := os.Open(fileName)
	if err != nil {
		return p, err
	}
	defer file.Close()

	// 解析yaml文件内容
	if err := yaml.NewDecoder(file).Decode(&p); err != nil {
		return p, err
	}
	return p, nil
}

// ReadDir 遍历指定目录及其子目录，收集所有YAML文件路径
// 参数:
//
//	root: 需要遍历的根目录路径
//
// 返回值:
//
//	[]string: 包含所有找到的YAML文件路径的切片
//	error: 遍历过程中可能发生的错误
func ReadDir(root string) ([]string, error) {
	var allPocs []string

	// 使用filepath.WalkDir递归遍历目录
	// 对于每个文件，检查是否为YAML文件，如果是则添加到结果切片中
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && common.IsYamlFile(path) {
			allPocs = append(allPocs, path)
		}
		return nil
	})

	return allPocs, err
}

// IsHTTPType 判断是否是http请求
func (finger *Finger) IsHTTPType() bool {
	for _, rule := range finger.Rules {
		reqType := rule.Value.Request.Type
		if len(reqType) == 0 || reqType == HttpType {
			return true
		}
	}
	return false
}

// UnmarshalYAML 解析yaml文件内容
func (r *Rule) UnmarshalYAML(unmarshal func(any) error) error {

	var tmp ruleAlias
	if err := unmarshal(&tmp); err != nil {
		return err
	}

	r.Request = tmp.Request
	r.Expression = tmp.Expression
	r.Expressions = append(r.Expressions, tmp.Expressions...)
	r.Output = tmp.Output
	r.StopIfMatch = tmp.StopIfMatch
	r.StopIfMismatch = tmp.StopIfMismatch
	r.BeforeSleep = tmp.BeforeSleep
	r.order = order

	order += 1
	return nil
}

// UnmarshalYAML 解析yaml文件内容
func (m *RuleMapSlice) UnmarshalYAML(unmarshal func(any) error) error {
	order = 0

	tempMap := make(map[string]Rule, 1)
	err := unmarshal(&tempMap)
	if err != nil {
		return err
	}

	newRuleSlice := make([]RuleMap, len(tempMap))
	for roleName, role := range tempMap {
		newRuleSlice[role.order] = RuleMap{
			Key:   roleName,
			Value: role,
		}
	}

	*m = newRuleSlice
	return nil
}
