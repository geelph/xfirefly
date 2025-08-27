package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"xfirefly/pkg/types"

	"github.com/projectdiscovery/goflags"
)

// NewCmdOptions 创建并解析命令行选项
func NewCmdOptions() (*types.CmdOptionsType, error) {
	// 声明命令行参数类型
	options := &types.CmdOptionsType{}
	// 声明参数结构体
	flagSet := goflags.NewFlagSet()
	// 目标拆解
	flagSet.CreateGroup("target", "TARGET",
		flagSet.StringSliceVarP(&options.Target, "target", "u", nil, "target URLs/hosts to scan", goflags.NormalizedStringSliceOptions),
		flagSet.StringVarP(&options.TargetsFile, "list", "l", "", "path to file containing a list of target URLs/hosts to scan (one per line)"),
	)
	// 结果输出
	flagSet.CreateGroup("output", "OUTPUT",
		flagSet.StringVarP(&options.Output, "output", "o", "", "output file to write found result"),
		flagSet.BoolVar(&options.JSONOutput, "json", false, "write output in JSON format"),
		flagSet.StringVar(&options.SockOutput, "sock", "", "write output socket in JOSN format"),
	)
	// 优化参数
	flagSet.CreateGroup("optimizations", "OPTIMIZATIONS",
		flagSet.StringVarP(&options.Proxy, "proxy", "p", "", "string of http/socks5 proxy to use"),
		flagSet.IntVarP(&options.Threads, "threads", "t", 5, "maximum number of requests to send per second"),
		flagSet.IntVarP(&options.RuleThreads, "rulethreads", "rt", 200, "maximum number of fingers to send per second"),
		flagSet.IntVar(&options.Timeout, "timeout", 3, "time to wait in seconds before timeout (default 3)"),
		flagSet.IntVar(&options.Retries, "retrise", 1, "number of times to retry a failed request (default 1)"),
		flagSet.IntVar(&options.MaxRedirects, "max-redirects", 5, "max number of redirects to follow for http templates (default 5)"),
	)
	// 日志管理
	flagSet.CreateGroup("debug", "DEBUG",
		flagSet.BoolVar(&options.Debug, "debug", false, "show verbose output"),
		flagSet.BoolVarP(&options.NoTimestamp, "no-timestamp", "ntp", false, "Output without timestamp"),
		flagSet.BoolVar(&options.FileLog, "file-log", false, "Output the log to file"),
	)
	// 指纹参数
	flagSet.CreateGroup("finger", "FINGER",
		flagSet.StringSliceVarP(&options.FingerOptions.FingerYaml, "finger-file", "f", nil, "list of finger to run (comma-separated, file)", goflags.NormalizedStringSliceOptions),
		flagSet.StringVarP(&options.FingerOptions.FingerPath, "finger-path", "fp", "", "finger directory to run"),
		flagSet.BoolVarP(&options.Active, "active", "a", false, "enable active finger path"),
	)
	// 杂项
	flagSet.CreateGroup("misc", "MISC",
		flagSet.BoolVar(&options.InitConfig, "init", false, "init config file"),
		flagSet.BoolVar(&options.PrintPreset, "print", false, "print preset all preset config"),
		flagSet.StringVarP(&options.Config, "config", "c", "", "path to the xfirefly configuration file"),
		flagSet.BoolVarP(&options.Version, "version", "v", false, "print version and exit"),
	)

	// 实例化操作
	if err := flagSet.Parse(); err != nil {
		return options, fmt.Errorf("The flag cannot be parsed: %s", err)
	}
	// 验证必参数是否传入
	if err := verifyOptions(options); err != nil {
		return options, err
	}

	return options, nil
}

// verifyOptions 验证命令行选项
func verifyOptions(opt *types.CmdOptionsType) error {
	// 使用反射自动序列化命令行选项用于调试
	//optionsStr := fmt.Sprintf("%+v", *opt)
	//fmt.Println("命令行选项：", optionsStr)
	// 验证版本输入、初始化配置、打印内置配置参数
	if opt.Version || opt.InitConfig || opt.PrintPreset {
		return nil
	}

	// 验证目标输入
	if len(opt.Target) == 0 && opt.TargetsFile == "" {
		return fmt.Errorf("The `-u` or `-l` parameter must be set to specify the scanning target")
	}

	// 验证输出文件格式
	if opt.Output != "" && !opt.JSONOutput { // 如果启用了JSON格式输出，则不检查文件扩展名
		ext := strings.ToLower(filepath.Ext(opt.Output))
		if ext != ".txt" && ext != ".csv" {
			return fmt.Errorf("The output file format only supports.txt or.csv, or the -json parameter can be used to enable JSON format output")
		}
	}

	// 验证socket文件扩展名
	if opt.SockOutput != "" {
		ext := strings.ToLower(filepath.Ext(opt.SockOutput))
		if ext != ".sock" {
			return fmt.Errorf("The socket output file must use the.sock extension")
		}
	}

	// 验证线程数
	if opt.Threads <= 0 {
		fmt.Println("[-] The number of threads is invalid and the default value of 5 will be used")
		opt.Threads = 5
	}

	// 验证规则线程数
	if opt.RuleThreads < 0 {
		fmt.Println("[-] The rule thread count is invalid and the default calculated value will be used")
		opt.RuleThreads = 0
	} else if opt.RuleThreads > 50000 {
		fmt.Println("[-] The number of rule threads is too large and has been limited to a maximum of 50,000")
		opt.RuleThreads = 50000
	}

	// 验证超时时间
	if opt.Timeout <= 0 {
		fmt.Println("[-] The timeout period is invalid and the default value of 3 seconds will be used")
		opt.Timeout = 3
	}

	return nil
}
