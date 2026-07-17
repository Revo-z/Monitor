package main

import "time"

// ── 监控参数 ──

const (
	// DefaultInterval 默认刷新间隔（秒）
	DefaultInterval = 0.5
	// MinInterval 最小刷新间隔（秒）
	MinInterval = 0.01
	// MaxInterval 最大刷新间隔（秒）
	MaxInterval = 2.0
	// Timeout 命令执行超时（秒）
	Timeout = 10 * time.Second
	// EndMarker 回显结束标志
	EndMarker = '#'
)

// ── 窗口属性 ──

const (
	// AppTitle 窗口标题
	AppTitle = "DevMonitor"
	// DefaultWidth 窗口默认宽度
	DefaultWidth = 1000
	// DefaultHeight 窗口默认高度
	DefaultHeight = 700
	// FontName 显示区字体
	FontName = "Consolas"
	// FontSize 显示区字号
	FontSize = 14
)

// ── 其他 ──

const (
	// Encoding 通信编码
	Encoding = "UTF-8"
	// ErrorLogFile 错误日志文件名
	ErrorLogFile = "error.log"
)

// ── 快捷命令 ──

// CommandShortcut 快捷命令按钮定义
type CommandShortcut struct {
	Label string // 按钮文本
	Cmd   string // 命令内容
}

// ShortcutCommands 快捷命令列表，用户可自行修改
var ShortcutCommands = []CommandShortcut{
	{Label: "CPU", Cmd: "display cpu-usage"},
	{Label: "内存", Cmd: "display memory-usage"},
	{Label: "磁盘", Cmd: "display disk-usage"},
	{Label: "网络", Cmd: "display interface brief"},
	{Label: "系统", Cmd: "display device"},
}
