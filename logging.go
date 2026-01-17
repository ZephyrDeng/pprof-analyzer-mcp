package main

import (
	"os"
)

// 全局日志配置
var (
	// 可通过环境变量 LOG_LEVEL 设置日志级别
	// 支持的级别: debug, info, warn, error
	logLevel = os.Getenv("LOG_LEVEL")
)

// initLogging 初始化日志系统
func initLogging() {
	// 设置默认日志级别
	if logLevel == "" {
		logLevel = "info"
	}
}

// getLogLevel 返回当前日志级别
func getLogLevel() string {
	return logLevel
}
