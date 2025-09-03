package finger

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"xfirefly/pkg/utils/common"

	"gopkg.in/yaml.v2"
)

// FingerFile 配置poc文件目录
const FingerFile = "fingerprint"
const (
	HttpType = "http"
	TcpType  = "tcp"
	UdpType  = "udp"
	SslType  = "ssl"
	GoType   = "go"
)

var order = 0

type Finger struct {
	Id         string        `yaml:"id"`        //  脚本名称
	Transport  string        `yaml:"transport"` // 传输方式，该字段用于指定发送数据包的协议，该字段用于指定发送数据包的协议:①tcp ②udp ③http
	Set        yaml.MapSlice `yaml:"set"`       // 全局变量定义，该字段用于定义全局变量。比如随机数，反连平台等
	Payloads   Payloads      `yaml:"payloads"`
	Rules      RuleMapSlice  `yaml:"rules"`
	Expression string        `yaml:"expression"`
	Info       Info          `yaml:"info"`
	Gopoc      string        `yaml:"gopoc"` // Gopoc 脚本名称
}
type Payloads struct {
	Continue bool          `yaml:"continue"`
	Payloads yaml.MapSlice `yaml:"payloads"`
}

// RuleMap 用于帮助yaml解析，保证Rule有序
type RuleMap struct {
	Key   string
	Value Rule
}

// RuleMapSlice 用于帮助yaml解析，保证Rule有序
type RuleMapSlice []RuleMap

type Rule struct {
	Request        RuleRequest   `yaml:"request"`
	Expression     string        `yaml:"expression"`
	Expressions    []string      `yaml:"expressions"`
	Output         yaml.MapSlice `yaml:"output"`
	StopIfMatch    bool          `yaml:"stop_if_match"`
	StopIfMismatch bool          `yaml:"stop_if_mismatch"`
	BeforeSleep    int           `yaml:"before_sleep"`
	order          int
}
type RuleRequest struct {
	Type            string            `yaml:"type"`         // 传输方式，默认 http，可选：tcp,udp,ssl,go 等任意扩展
	Host            string            `yaml:"host"`         // tcp/udp 请求的主机名
	Data            string            `yaml:"data"`         // tcp/udp 发送的内容
	DataType        string            `yaml:"data-type"`    // tcp/udp 发送的数据类型，默认字符串
	ReadSize        int               `yaml:"read-size"`    // tcp/udp 读取内容的长度
	ReadTimeout     int               `yaml:"read-timeout"` // tcp/udp专用
	Raw             string            `yaml:"raw"`          // raw 专用
	Method          string            `yaml:"method"`
	Path            string            `yaml:"path"`
	Headers         map[string]string `yaml:"headers"`
	Body            string            `yaml:"body"`
	FollowRedirects bool              `yaml:"follow_redirects"` // 是否跟随重定向，默认跟随重定向
}

// Info 以下开始是 信息部分
type Info struct {
	Name           string         `yaml:"name"`
	Author         string         `yaml:"author"`
	Severity       string         `yaml:"severity"`
	Verified       bool           `yaml:"verified"`
	Description    string         `yaml:"description"`
	Reference      []string       `yaml:"reference"`
	Affected       string         `yaml:"affected"`  // 影响版本
	Solutions      string         `yaml:"solutions"` // 解决方案
	Tags           string         `yaml:"tags"`      // 标签
	Classification Classification `yaml:"classification"`
	Created        string         `yaml:"created"` // create time
}

type Classification struct {
	CvssMetrics string  `yaml:"cvss-metrics"`
	CvssScore   float64 `yaml:"cvss-score"`
	CveId       string  `yaml:"cve-id"`
	CweId       string  `yaml:"cwe-id"`
}

type ruleAlias struct {
	Request        RuleRequest   `yaml:"request"`
	Expression     string        `yaml:"expression"`
	Expressions    []string      `yaml:"expressions"`
	Output         yaml.MapSlice `yaml:"output"`
	StopIfMatch    bool          `yaml:"stop_if_match"`
	StopIfMismatch bool          `yaml:"stop_if_mismatch"`
	BeforeSleep    int           `yaml:"before_sleep"`
}

// Select 获取指定名字的yaml文件位置
func Select(pocPath string, pocName string) (string, error) {
	// 检查路径是否存在
	if pocPath == "" {
		return "", fmt.Errorf("指纹库路径为空")
	}

	// 检查路径是否存在
	if !common.Exists(pocPath) {
		return "", fmt.Errorf("指纹库路径 '%s' 不存在", pocPath)
	}

	var result string
	// 遍历目录中的所有文件和子目录
	err := filepath.WalkDir(pocPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err // 如果遇到错误，立即返回
		}

		// 检查文件是否是 YAML 文件
		if !d.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
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
	// 检查文件名是否已经包含finger/前缀，避免重复路径
	filePath := fileName
	if !strings.HasPrefix(fileName, "fingerprint/") {
		filePath = "fingerprint/" + fileName
	}

	yamlFile, err := Fingers.ReadFile(filePath)

	if err != nil {
		fmt.Printf("[-] load poc %s error1: %v\n", filePath, err)
		return nil, err
	}
	err = yaml.Unmarshal(yamlFile, p)
	if err != nil {
		fmt.Printf("[-] load poc %s error2: %v\n", filePath, err)
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

	if err := yaml.NewDecoder(file).Decode(&p); err != nil {
		return p, err
	}
	return p, nil
}

// ReadDir 获取特定目录下面所有yaml文件
func ReadDir(root string) ([]string, error) {
	var allPocs []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
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
