package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"xfirefly/pkg/types"

	"github.com/donnie4w/go-logger/logger"

	"github.com/spf13/pflag"
)

// NewCmdOptions 初始化并解析命令行参数，返回 CmdOptionsType 结构体实例和可能的错误。
// 该函数使用 pflag 包定义并解析命令行选项，支持多种扫描目标输入方式、输出配置、
// 并发控制、超时设置、代理配置等功能。
//
// 返回值：
//   - CmdOptionsType：包含解析后的命令行参数配置。
//   - error：解析过程中可能发生的错误（当前实现始终返回 nil）。
func NewCmdOptions() (types.CmdOptionsType, error) {
	// 声明命令行参数类型
	options := types.CmdOptionsType{}
	flagset := pflag.NewFlagSet("test", pflag.ExitOnError)

	// 定义命令行参数
	flagset.StringSliceVarP(&options.Target, "url", "u", []string{}, "扫描目标: 可以为URL/IP/域名/Host:Port等多种形式的混合输入")
	flagset.StringVarP(&options.TargetsList, "list", "l", "", "目标文件: 指定含有扫描目标的文本文件")
	flagset.StringVarP(&options.Output, "output", "o", "", "结果输出: 指定保存结果的文件路径（txt/csv，根据扩展名自动识别；也可配合 --json 输出JSON）")
	flagset.BoolVar(&options.JSONOutput, "json", false, "使用JSON格式输出结果到文件")
	flagset.StringVar(&options.SockOutput, "sock", "", "结果输出: 输出socket文件")
	flagset.StringVarP(&options.Proxy, "proxy", "p", "", "HTTP客户端代理: [http|https|socks5://][username[:password]@]host[:port]")
	flagset.IntVarP(&options.Threads, "threads", "t", 5, "URL并发线程数")
	flagset.IntVar(&options.RuleThreads, "rule-threads", 200, "指纹规则并发线程数")
	flagset.IntVar(&options.Timeout, "timeout", 5, "读超时: 从连接中读取数据的最大耗时")
	flagset.IntVar(&options.Retries, "retries", 2, "请求失败重试次数")
	flagset.IntVar(&options.MaxRedirects, "max-redirects", 5, "最大允许 HTTP 请求跳转次数")
	flagset.BoolVar(&options.Debug, "debug", false, "调试：打印debug日志")
	flagset.BoolVar(&options.NoTimestamp, "no-timestamp", false, "不显示时间戳")
	flagset.BoolVar(&options.FileLog, "file-log", false, "保存日志到文件")
	flagset.StringVar(&options.FingerOptions.FingerPath, "finger-path", "", "指纹路径")
	flagset.StringSliceVarP(&options.FingerOptions.FingerYaml, "finger", "f", []string{}, "指纹文件")
	flagset.BoolVarP(&options.Active, "active", "a", false, "启用主动指纹探测")
	flagset.BoolVar(&options.InitConfig, "init-config", false, "初始化配置文件")
	flagset.BoolVar(&options.PrintPreset, "print", false, "打印所有预置配置")
	flagset.StringVarP(&options.Config, "config", "c", "config.yaml", "配置文件路径")
	flagset.BoolVarP(&options.Version, "version", "v", false, "查看版本信息")

	// 禁止自动排序参数
	flagset.SortFlags = false

	// 自定义 Usage
	flagset.Usage = func() {
		fmt.Fprintf(pflag.CommandLine.Output(), "用法: %s [选项]\n", os.Args[0])
		fmt.Println("Web应用指纹识别工具")
		fmt.Println()
		fmt.Println("选项:")
		flagset.PrintDefaults()
		fmt.Println()
		fmt.Println("示例:")
		fmt.Println("  ", os.Args[0], "-t http://test.com")
	}

	// 解析命令行参数
	flagset.Parse(os.Args[1:])

	// 验证必参数是否传入
	if err := verifyOptions(options); err != nil {
		return options, err
	}

	return options, nil
}

// verifyOptions 验证命令行选项
func verifyOptions(opt types.CmdOptionsType) error {
	// 使用反射自动序列化命令行选项用于调试
	//optionsStr := fmt.Sprintf("%+v", *opt)
	//fmt.Println("命令行选项：", optionsStr)
	// 验证版本输入、初始化配置、打印内置配置参数
	if opt.Version || opt.InitConfig || opt.PrintPreset {
		return nil
	}

	// 验证目标输入
	if len(opt.Target) == 0 && opt.TargetsList == "" {
		return fmt.Errorf("必须使用`-u`或`-l`参数指定扫描目标")
	}

	// 验证输出文件格式
	if opt.Output != "" && !opt.JSONOutput { // 如果启用了JSON格式输出，则不检查文件扩展名
		ext := strings.ToLower(filepath.Ext(opt.Output))
		if ext != ".txt" && ext != ".csv" {
			return fmt.Errorf("输出文件格式仅支持.txt或.csv，也可以使用-json参数启用JSON格式输出")
		}
	}

	// 验证socket文件扩展名
	if opt.SockOutput != "" {
		ext := strings.ToLower(filepath.Ext(opt.SockOutput))
		if ext != ".sock" {
			return fmt.Errorf("socket输出文件扩展名必须是.sock")
		}
	}

	// 验证线程数
	if opt.Threads <= 0 {
		logger.Warn("指定线程数无效，将使用默认值5")
		opt.Threads = 5
	}

	// 验证规则线程数
	if opt.RuleThreads < 0 {
		logger.Warn("指定规则线程数无效，将使用默认值200")
		opt.RuleThreads = 200
	} else if opt.RuleThreads > 50000 {
		logger.Warn("指定规则线程数太大，将使用最大值50000")
		opt.RuleThreads = 50000
	}

	// 验证超时时间
	if opt.Timeout <= 0 {
		logger.Warn("指定超时时间不合法，将使用默认值3秒")
		opt.Timeout = 3
	}

	// 重试次数
	if opt.Retries < 0 {
		logger.Warn("指定重试次数不合法，将使用默认值1")
		opt.Retries = 1
	}

	// 最大跳转次数
	if opt.MaxRedirects < 0 {
		logger.Warn("指定最大跳转次数不合法，将使用默认值5")
		opt.MaxRedirects = 5
	}

	return nil
}
