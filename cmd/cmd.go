package cmd

import (
	"fmt"
	"github.com/fatih/color"
	"os"
	"xfirefly/pkg/cli"
)

// init
//
//	@Description: 工具入口，初始化函数
func init() {
	// version 命令
	//rootCmd.AddCommand(cmdVersion)
}

// Execute
//
//	@Description: 整个程序的入口
func Execute() {
	// 声明参数结构变量
	options, err := cli.NewCmdOptions()
	if err != nil {
		// 在初始化logger之前的错误使用默认logger
		color.Red(fmt.Sprintf("[ERROR] %s", err.Error()))
		os.Exit(1)
	}

	// 配置日志级别
	options.Debug = true

}

func DisplayBanner() {
	cli.DisplayBanner()
}
