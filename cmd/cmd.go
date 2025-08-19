package cmd

import (
	"fmt"
	"github.com/jessevdk/go-flags"
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
	var option cli.Option

	// 命令行参数设计
	parser := flags.NewParser(&option, flags.Default)

	// 用法说明
	parser.Usage = `

`

	// 参数解析,错误结束
	_, err := parser.Parse()
	if err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			fmt.Println(err.Error())
		}
		return
	}
}
