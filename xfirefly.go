package main

import (
	"xfirefly/cmd"
)

func main() {

	// 绿色方式打印 banner
	cmd.DisplayBanner()
	//color.Greenp(cli.Banner)

	// 程序核心入口函数
	cmd.Execute()
}

func init() {

}
