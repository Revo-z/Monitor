package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// logPath 日志文件路径，默认使用 config 中定义的 ErrorLogFile。
// 测试中可覆盖此变量指向临时目录。
var logPath = ErrorLogFile

// LogError 追加一条带时间戳的错误日志。
// 如果文件打开或写入失败则静默返回，不中断主流程。
func LogError(msg string) {
	dir := filepath.Dir(logPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return
		}
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	line := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
	f.WriteString(line)
}
