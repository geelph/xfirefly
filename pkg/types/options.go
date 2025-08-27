package types

import (
	"github.com/projectdiscovery/goflags"
)

// YamlFingerType 指纹文件类型
type YamlFingerType struct {
	FingerPath string              // POC文件路径
	FingerYaml goflags.StringSlice // 单个POC yaml文件
}

// CmdOptionsType 命令行选项结构体
type CmdOptionsType struct {
	Target        goflags.StringSlice // 测试目标
	TargetsFile   string              // 测试目标文件
	Output        string              // 输出文件路径
	JSONOutput    bool                // 是否使用JSON格式输出结果
	SockOutput    string              // socket文件输出路径，启用后会以JSON格式输出到socket文件
	Proxy         string              // 代理地址
	Threads       int                 // 并发线程数
	RuleThreads   int                 // 指纹规则线程数
	Timeout       int                 // 超时时间，默认5秒
	Retries       int                 // 重试次数，默认1次
	MaxRedirects  int                 // 最大跳转次数，默认5次
	Debug         bool                // 设置debug模式
	NoTimestamp   bool                // 输出时间戳
	FileLog       bool                // 是否禁用文件日志，仅输出到控制台
	FingerOptions YamlFingerType      // Finger yaml文件配置
	Active        bool                // 主动指纹探测
	InitConfig    bool                // 初始化配置文件
	PrintPreset   bool                // 打印预配置
	Config        string              // 指定配置文件
	Version       bool                // 打印版本信息
}
