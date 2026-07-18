package display

import "strings"

type Display struct {
	buf       strings.Builder
	overwrite bool
	hasData   bool
}

func NewDisplay() *Display {
	return &Display{overwrite: true}
}

func (d *Display) SetMode(overwrite bool) {
	d.overwrite = overwrite
}

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

func (d *Display) Clear() {
	d.buf.Reset()
	d.hasData = false
}

func (d *Display) Text() string {
	return d.buf.String()
}
