package main

import (
	"Monitor/display"
	"Monitor/ui"
)

func main() {
	disp := display.NewDisplay()
	ui.RunUI(disp)
}
