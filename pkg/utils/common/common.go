package common

import "github.com/donnie4w/go-logger/logger"

// 全局变量
var (
	// 维护的日志等级变量，go-logger中没有提供GetLevel的函数
	LogLevel = logger.LEVEL_INFO
)
