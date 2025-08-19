package cmd

import (
	"github.com/spf13/cobra"
	"os"
)

// 工具描述信息，执行命令即显示
var rootCmd = &cobra.Command{
	Use:   "xfirefly",
	Short: "XFirefly is a fingerprint recognition tool",
	Long: `    XFirefly is an efficient fingerprint recognition tool configured based on YAML
rules,capable of precise system identification of multiple protocols such as 
HTTP/HTTPS, TCP,and UDP.It supports large-scale target batch scanning, facilitating
asset discovery and security assessment.`,
}

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
	if err := rootCmd.Execute(); err != nil {
		// 打印错误命令信息
		// 注释,cobra默认打印报错信息
		//fmt.Printf("root command: %s\n", err)
		os.Exit(1)
	}
}
