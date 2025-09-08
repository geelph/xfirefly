package cmd

import (
	"os"
	"time"
	"xfirefly/pkg/cli"
	"xfirefly/pkg/runner"
	"xfirefly/pkg/types"
	"xfirefly/pkg/utils/common"

	"github.com/donnie4w/go-logger/logger"
	"github.com/fatih/color"
)

// init
//
//	@Description: 工具入口，初始化函数
func init() {
	// 配置日志格式
	initLogConfig()
}

// Execute
//
//	@Description: 整个程序的入口
func Execute() {
	// 加载配置文件
	//logger.Info("使用以下位置的配置文件：xxx")
	//logger.Info("未能正确加载配置文件或配置文件不存在，使用默认配置")

	// 声明参数结构变量
	options, err := cli.NewCmdOptions()
	if err != nil {
		// 在初始化logger之前的错误使用默认logger
		//color.Red(fmt.Sprintf("[ERROR] %s", err.Error()))
		//fmt.Println(fmt.Sprintf("[ERROR] %s", err.Error()))
		logger.Error(err)
		// 异常退出
		os.Exit(1)
	}

	// 打印版本信息并退出
	if options.Version {
		cli.DisplayVersion()
		os.Exit(0)
	}

	// 初始化配置文件
	if options.InitConfig {
		logger.Info("正在初始化配置文件")
		os.Exit(0)
	}

	// 打印所有内置配置
	if options.PrintPreset {
		logger.Info("正在打印内置指纹信息")
		if err := runner.PrintPresetFinger(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// 日志时间戳设置
	if options.NoTimestamp {
		logger.SetFormat(logger.FORMAT_LEVELFLAG | logger.FORMAT_SHORTFILENAME)
		logger.SetFormatter("[{level}] {message} ({file})\n")
	}

	// 配置日志级别
	if options.Debug {
		common.LogLevel = logger.LEVEL_DEBUG
		logger.SetLevel(common.LogLevel)
		logger.Debug("DEBUG 模式已开启")
	}

	// 日志文件
	if options.FileLog {
		filename := "logs/app.log"
		// 日志写入文件
		logger.SetOption(&logger.Option{
			Level:      common.LogLevel,
			Console:    true,
			FileOption: &logger.FileTimeMode{Filename: filename, Maxbuckup: 10, IsCompress: true, Timemode: logger.MODE_HOUR},
		})
		logger.Debugf("文件日志记录功能已开启,日志文件位置:%s", filename)
	}

	// 代理选项配置
	if options.Proxy != "" {
		logger.Infof("代理参数已配置：%s", options.Proxy)
	}

	// 日志测试
	//logger.Debug("this is a debug message:", 1111111111111111111)
	//logger.Info("this is a info message:", 2222222222222222222)
	//logger.Warn("this is a warn message:", 33333333333333333)
	//logger.Error("this is a error message:", 4444444444444444444)
	//logger.Fatal("this is a fatal message:", 555555555555555555)

	// 输出文件设置
	// 确定输出文件格式
	// 初始化输出文件
	// 初始化socket文件输出（如果启用）
	// 延时匿名函数，关闭所有输出资源

	// 记录运行开始时间
	startTime := time.Now()

	// 运行核心程序
	run(options)

	// 运行结束时间
	// 计算并打印运行时间
	elapsedTime := time.Since(startTime)
	logger.Infof("程序运行时长: %v", elapsedTime)
	// 或者格式化输出，例如只显示秒
	//fmt.Printf("程序运行时间: %.2f 秒\n", elapsedTime.Seconds())
}

// DisplayBanner
//
//	@Description: 打印 banner 信息
func DisplayBanner() {
	cli.DisplayBanner()
}

// initLogConfig
//
//	@Description: 初始化日志配置，日志等级、输出类型、输出格式、等级颜色等
func initLogConfig() {
	// 日志格式初始化
	// 自定义日志格式
	attrFormat := &logger.AttrFormat{
		// 为日志等级设置颜色，但会影响文件日志显示美观性，目前对日志文件要求不高，可忽略
		SetLevelFmt: func(level logger.LEVELTYPE) string {
			switch level {
			case logger.LEVEL_DEBUG:
				//return fmt.Sprintf("%sDEBUG%s", common.ColorCyan, common.ColorReset)
				return color.CyanString("DEBUG")
			case logger.LEVEL_INFO:
				//return fmt.Sprintf("%sINFO %s", common.ColorGreen, common.ColorReset)
				return color.GreenString("INFO")
			case logger.LEVEL_WARN:
				return color.YellowString("WARN")
				//return fmt.Sprintf("%sWARN %s", common.ColorYellow, common.ColorReset)
			case logger.LEVEL_ERROR:
				return color.RedString("ERROR")
				//return fmt.Sprintf("%sERROR%s", common.ColorRed, common.ColorReset)
			case logger.LEVEL_FATAL:
				return color.HiRedString("FATAL")
				//return fmt.Sprintf("%sFATAL%s", common.ColorRed+common.ColorBold, common.ColorReset)
			default:
				return "UNKNOWN"
			}
		},
		//SetTimeFmt: func() (string, string, string) {
		//	now := time.Now().Format("2006-01-02 15:04:05")
		//	return now, "", ""
		//},
		//// 整行日志显示颜色
		//SetBodyFmt: func(level logger.LEVELTYPE, msg []byte) []byte {
		//	switch level {
		//	case logger.LEVEL_DEBUG:
		//		return append([]byte("\033[34m"), append(msg, '\033', '[', '0', 'm')...) // Blue for DEBUG
		//	case logger.LEVEL_INFO:
		//		return append([]byte("\033[32m"), append(msg, '\033', '[', '0', 'm')...) // Green for INFO
		//	case logger.LEVEL_WARN:
		//		return append([]byte("\033[33m"), append(msg, '\033', '[', '0', 'm')...) // Yellow for WARN
		//	case logger.LEVEL_ERROR:
		//		return append([]byte("\033[31m"), append(msg, '\033', '[', '0', 'm')...) // Red for ERROR
		//	case logger.LEVEL_FATAL:
		//		return append([]byte("\033[41m"), append(msg, '\033', '[', '0', 'm')...) // Red background for FATAL
		//	default:
		//		return msg
		//	}
		//},
	}
	// 日志选项设置
	logger.SetOption(&logger.Option{
		Level:      common.LogLevel,
		Console:    true,
		Format:     logger.FORMAT_TIME | logger.FORMAT_LEVELFLAG | logger.FORMAT_SHORTFILENAME,
		Formatter:  "[{time}] [{level}] {message} ({file})\n",
		AttrFormat: attrFormat,
	})
	// 设置日志等级显示格式
	// 启用后导致对应等级格式失效
	//logger.SetFormat(logger.FORMAT_TIME | logger.FORMAT_LEVELFLAG | logger.FORMAT_SHORTFILENAME)
	//logger.SetLevelOption(logger.LEVEL_DEBUG, &logger.LevelOption{Format: logger.FORMAT_TIME | logger.FORMAT_LEVELFLAG | logger.FORMAT_SHORTFILENAME})
	//logger.SetLevelOption(logger.LEVEL_INFO, &logger.LevelOption{Format: logger.FORMAT_TIME | logger.FORMAT_LEVELFLAG | logger.FORMAT_SHORTFILENAME})
	//logger.SetLevelOption(logger.LEVEL_WARN, &logger.LevelOption{Format: logger.FORMAT_TIME | logger.FORMAT_LEVELFLAG | logger.FORMAT_SHORTFILENAME | logger.FORMAT_FUNC})
	//logger.SetLevelOption(logger.LEVEL_ERROR, &logger.LevelOption{Format: logger.FORMAT_TIME | logger.FORMAT_LEVELFLAG | logger.FORMAT_SHORTFILENAME | logger.FORMAT_FUNC})
	//logger.SetLevelOption(logger.LEVEL_FATAL, &logger.LevelOption{Format: logger.FORMAT_TIME | logger.FORMAT_LEVELFLAG | logger.FORMAT_SHORTFILENAME | logger.FORMAT_FUNC})
	//// 设置显示格式
	//logger.SetFormatter("[{time}] {level} {message} [{file}]\n")
	//logger.SetLevel(common.LogLevel)
	//logger.SetLevel(logger.LEVEL_FATAL)

}

func run(options *types.CmdOptionsType) {
	// 开启内存监控
	runner.StartMemoryMonitor()
	// 停止内存监控，延时调用，后进先出
	defer runner.StopMemoryMonitor()
	// 声明一个新的Runner
	r := runner.NewRunner(options)
	// 运行扫描
	if err := r.Run(options); err != nil {
		// 错误已在Run函数内部记录，这里无需额外处理
		logger.Error(err)
		return
	}
}
