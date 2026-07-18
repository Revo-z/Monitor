package config

import "time"

const (
	DefaultInterval = 0.5
	MinInterval     = 0.01
	MaxInterval     = 2.0
	Timeout         = 10 * time.Second
	EndMarker       = '#'
)

const (
	AppTitle      = "DevMonitor"
	DefaultWidth  = 1000
	DefaultHeight = 700
	FontName      = "Consolas"
	FontSize      = 14
)

const (
	Encoding     = "UTF-8"
	ErrorLogFile = "error.log"
)

type CommandShortcut struct {
	Label string
	Cmd   string
}

var ShortcutCommands = []CommandShortcut{
	{Label: "CPU", Cmd: "display cpu-usage"},
	{Label: "内存", Cmd: "display memory-usage"},
	{Label: "磁盘", Cmd: "display disk-usage"},
	{Label: "网络", Cmd: "display interface brief"},
	{Label: "系统", Cmd: "display device"},
}
