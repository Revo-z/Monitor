package main

import "strings"

// Display 管理回显缓冲区和显示模式。纯数据层，不依赖任何 UI 框架。
type Display struct {
	buf       strings.Builder
	overwrite bool // true=覆盖, false=追加
	hasData   bool // 追加模式下是否已有内容
}

// NewDisplay 创建显示缓冲区，默认覆盖模式。
func NewDisplay() *Display {
	return &Display{overwrite: true}
}

// SetMode 设置显示模式。overwrite=true 为覆盖，false 为追加。
func (d *Display) SetMode(overwrite bool) {
	d.overwrite = overwrite
}

// Write 按当前模式写入文本。覆盖模式先清空再写入；追加模式在末尾追加。
func (d *Display) Write(text string) {
	if d.overwrite {
		d.buf.Reset()
		d.buf.WriteString(text)
	} else {
		if d.hasData {
			d.buf.WriteByte('\n')
		}
		d.buf.WriteString(text)
		d.hasData = true
	}
}

// Clear 清空缓冲区。
func (d *Display) Clear() {
	d.buf.Reset()
	d.hasData = false
}

// Text 返回当前缓冲区内容。
func (d *Display) Text() string {
	return d.buf.String()
}
