package cmd

import (
	"os"
	"xfirefly/pkg/cli"
	"xfirefly/pkg/runner"
	"xfirefly/pkg/types"
	"xfirefly/pkg/utils/common"

	"github.com/donnie4w/go-logger/logger"
)

// init
//
//	@Description: 工具入口，初始化函数
func init() {
	// 配置日志格式
	// 日志格式初始化
	logger.SetFormat(logger.FORMAT_TIME | logger.FORMAT_LEVELFLAG | logger.FORMAT_SHORTFILENAME)
	logger.SetFormatter("[{time}] {level} {message} [{file}]\n")
	logger.SetLevel(common.LogLevel)

}

// Execute
//
//	@Description: 整个程序的入口
func Execute() {
	// 加载配置文件
	logger.Info("使用以下位置的配置文件：xxx")
	logger.Info("未能正确加载配置文件或配置文件不存在，使用默认配置")

	// 声明参数结构变量
	options, err := cli.NewCmdOptions()
	if err != nil {
		// 在初始化logger之前的错误使用默认logger
		//color.Red(fmt.Sprintf("[ERROR] %s", err.Error()))
		//fmt.Println(fmt.Sprintf("[ERROR] %s", err.Error()))
		logger.Error(err.Error())
		os.Exit(1)
	}

	// 打印版本信息并退出
	if options.Version {
		cli.DisplayVersion()
		os.Exit(0)
	}

	// 日志时间戳设置
	if options.NoTimestamp {
		logger.SetFormat(logger.FORMAT_LEVELFLAG | logger.FORMAT_SHORTFILENAME)
		logger.SetFormatter("{level} {message} [{file}]\n")
	}

	// 配置日志级别
	if options.Debug {
		logger.SetLevel(logger.LEVEL_DEBUG)
		common.LogLevel = logger.LEVEL_DEBUG
		logger.Debug("DEBUG 模式已开启")
	}

	// 日志文件
	if options.FileLog {
		// 日志写入文件
		logger.SetOption(&logger.Option{
			Level:      common.LogLevel,
			Console:    true,
			FileOption: &logger.FileTimeMode{Filename: "logs/app.log", Maxbuckup: 10, IsCompress: true, Timemode: logger.MODE_MONTH},
		})
		logger.Debug("文件日志记录功能已开启")
	}

	// 输出文件设置
	// 确定输出文件格式
	// 初始化输出文件
	// 初始化socket文件输出（如果启用）
	// 延时匿名函数，关闭所有输出资源

	// 运行核心程序
	run(options)

}

// DisplayBanner
//
//	@Description: 打印 banner 信息
func DisplayBanner() {
	cli.DisplayBanner()
}

func initLogConfig() {

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
		return
	}
}
