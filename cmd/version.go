package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"runtime"
)

// 模板版本信息，使用 var 使其可以通过 ldflags 修改
var defaultVersion = "v0.0.1"
var defaultAuthor = "geelph"
var defaultBuildDate = "unknown"
var defaultGitCommit = "none"

// 版本命令
var cmdVersion = &cobra.Command{
	Use:   "version",
	Short: "Print version and exit",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		// 自定义 version 显示
		fmt.Printf("  %s version information: \n", rootCmd.Name())
		fmt.Printf("  Version:\t%s\n", rootCmd.Version)
		fmt.Printf("  Git Commit:\t%s\n", defaultGitCommit)
		fmt.Printf("  Go Version:\t%s\n", runtime.Version())
		fmt.Printf("  OS/Arch:\t%s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("  Build Time:\t%s\n", defaultBuildDate)
		return
	},
}

// init
//
//	@Description: 初始化函数，将添加version命令、设置程序版本
func init() {
	// 添加 version 命令
	rootCmd.AddCommand(cmdVersion)
	// 设置版本
	rootCmd.Version = defaultVersion
}
