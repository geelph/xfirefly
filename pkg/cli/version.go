package cli

import (
	"fmt"
	"runtime"
)

// 模板版本信息，使用 var 使其可以通过 ldflags 修改
var defaultVersion = "v0.0.1"
var defaultAuthor = "geelph"
var defaultBuildDate = "2025-08-20"
var defaultGitCommit = "none"

// 版本命令
func DisplayVersion() {
	// 自定义 version 显示
	fmt.Printf("  %s version information: \n", "xfirefly")
	fmt.Printf("  Version:\t%s\n", defaultVersion)
	fmt.Printf("  Git Commit:\t%s\n", defaultGitCommit)
	fmt.Printf("  Go Version:\t%s\n", runtime.Version())
	fmt.Printf("  OS/Arch:\t%s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  Build Time:\t%s\n", defaultBuildDate)

	return
}
