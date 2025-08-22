package cmd

import (
	"os"
	"xfirefly/pkg/cli"

	"github.com/donnie4w/go-logger/logger"
)

var (
	xenv = "prod" // 环境标识，控制日志输出，{dev | prod}
)

// init
//
//	@Description: 工具入口，初始化函数
func init() {
	// 配置日志格式
	// 日志格式初始化
	logger.SetFormat(logger.FORMAT_TIME | logger.FORMAT_LEVELFLAG | logger.FORMAT_SHORTFILENAME)
	logger.SetFormatter("[{time}] {level} {message} [{file}]\n")
	logger.SetLevel(logger.LEVEL_INFO)

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

	// 配置日志级别
	if options.Debug {
		logger.SetLevel(logger.LEVEL_DEBUG)
		logger.Debug("设置日志级别为：DEBUG")
	}

}

// DisplayBanner
//
//	@Description: 打印 banner 信息
func DisplayBanner() {
	cli.DisplayBanner()
}
