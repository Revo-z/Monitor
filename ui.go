package main

import (
	"fmt"
	"strconv"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

type uiState struct {
	disp         *Display
	mon          *Monitor

	mw           *walk.MainWindow
	ipEdit       *walk.LineEdit
	portEdit     *walk.LineEdit
	cmdEdit      *walk.LineEdit
	intervalEdit *walk.NumberEdit
	overwriteRB  *walk.RadioButton
	appendRB     *walk.RadioButton
	clearBtn     *walk.PushButton
	startBtn     *walk.PushButton
	stopBtn      *walk.PushButton
	displayEdit  *walk.TextEdit
}

func newLabel(parent walk.Container, text string) *walk.Label {
	lb, _ := walk.NewLabel(parent)
	lb.SetText(text)
	return lb
}

func newLineEdit(parent walk.Container) *walk.LineEdit {
	le, _ := walk.NewLineEdit(parent)
	return le
}

func newNumberEdit(parent walk.Container) *walk.NumberEdit {
	ne, _ := walk.NewNumberEdit(parent)
	return ne
}

func newPushButton(parent walk.Container, text string) *walk.PushButton {
	btn, _ := walk.NewPushButton(parent)
	btn.SetText(text)
	return btn
}

func newRadioButton(parent walk.Container, text string) *walk.RadioButton {
	rb, _ := walk.NewRadioButton(parent)
	rb.SetText(text)
	return rb
}

func newComposite(parent walk.Container) *walk.Composite {
	c, _ := walk.NewComposite(parent)
	return c
}

func RunUI(disp *Display) (int, error) {
	s := &uiState{disp: disp}

	mw, err := walk.NewMainWindow()
	if err != nil {
		return 1, err
	}
	s.mw = mw

	mw.SetTitle(AppTitle)
	mw.SetSize(walk.Size{Width: DefaultWidth, Height: DefaultHeight})

	screenW := int(win.GetSystemMetrics(win.SM_CXSCREEN))
	screenH := int(win.GetSystemMetrics(win.SM_CYSCREEN))
	mw.SetBounds(walk.Rectangle{
		X:      (screenW - DefaultWidth) / 2,
		Y:      (screenH - DefaultHeight) / 2,
		Width:  DefaultWidth,
		Height: DefaultHeight,
	})

	layout := walk.NewVBoxLayout()
	layout.SetMargins(walk.Margins{HNear: 8, VNear: 8, HFar: 8, VFar: 8})
	layout.SetSpacing(4)
	mw.SetLayout(layout)

	row1 := newComposite(mw)
	row1.SetLayout(walk.NewHBoxLayout())
	newLabel(row1, "IP:")
	s.ipEdit = newLineEdit(row1)
	newLabel(row1, "  端口:")
	s.portEdit = newLineEdit(row1)
	s.portEdit.SetMaxLength(5)

	row2 := newComposite(mw)
	row2.SetLayout(walk.NewHBoxLayout())
	newLabel(row2, "监控命令:")
	s.cmdEdit = newLineEdit(row2)

	row3 := newComposite(mw)
	row3.SetLayout(walk.NewHBoxLayout())
	newLabel(row3, "刷新间隔:")
	s.intervalEdit = newNumberEdit(row3)
	s.intervalEdit.SetDecimals(2)
	s.intervalEdit.SetValue(0.5)
	s.intervalEdit.SetRange(0.01, 2.0)
	newLabel(row3, "秒 （范围 0.01–2）")

	row4 := newComposite(mw)
	row4.SetLayout(walk.NewHBoxLayout())
	newLabel(row4, "显示模式:")
	s.overwriteRB = newRadioButton(row4, "覆盖")
	s.overwriteRB.SetChecked(true)
	s.appendRB = newRadioButton(row4, "追加")
	s.clearBtn = newPushButton(row4, "清空显示")

	row5 := newComposite(mw)
	row5.SetLayout(walk.NewHBoxLayout())
	s.startBtn = newPushButton(row5, "开始")
	s.stopBtn = newPushButton(row5, "停止")
	s.stopBtn.SetEnabled(false)

	s.displayEdit, _ = walk.NewTextEdit(mw)
	s.displayEdit.SetReadOnly(true)
	font, _ := walk.NewFont(FontName, FontSize, 0)
	s.displayEdit.SetFont(font)

	row6 := newComposite(mw)
	row6.SetLayout(walk.NewHBoxLayout())
	for _, sc := range ShortcutCommands {
		btn := newPushButton(row6, sc.Label)
		cmd := sc.Cmd
		btn.Clicked().Attach(func() {
			s.cmdEdit.SetText(cmd)
		})
	}

	s.bindEvents()

	mw.SetVisible(true)
	win.SetForegroundWindow(mw.Handle())

	return mw.Run(), nil
}

func (s *uiState) bindEvents() {
	s.startBtn.Clicked().Attach(func() {
		ip := s.ipEdit.Text()
		portStr := s.portEdit.Text()
		cmd := s.cmdEdit.Text()

		if ip == "" {
			walk.MsgBox(s.mw, "提示", "请输入 IP 地址", walk.MsgBoxIconInformation)
			return
		}
		portNum, err := strconv.Atoi(portStr)
		if err != nil || portNum < 1 || portNum > 65535 {
			walk.MsgBox(s.mw, "提示", "端口必须为 1–65535 的数字", walk.MsgBoxIconInformation)
			return
		}
		if cmd == "" {
			walk.MsgBox(s.mw, "提示", "请输入监控命令", walk.MsgBoxIconInformation)
			return
		}

		conn, err := NewConn(ip, portStr)
		if err != nil {
			LogError(fmt.Sprintf("连接失败: %s", err.Error()))
			walk.MsgBox(s.mw, "错误", fmt.Sprintf("连接失败: %s", err.Error()), walk.MsgBoxIconError)
			return
		}

		s.mon = NewMonitor(conn, cmd, s.intervalEdit.Value())
		s.mon.Start(func(text string) {
			s.disp.Write(text)
			s.mw.Synchronize(func() {
				s.displayEdit.SetText(s.disp.Text())
			})
		})

		s.setRunning(true)
	})

	s.stopBtn.Clicked().Attach(func() {
		if s.mon != nil {
			s.mon.Stop()
			s.mon = nil
		}
		s.setRunning(false)
	})

	s.clearBtn.Clicked().Attach(func() {
		s.disp.Clear()
		s.displayEdit.SetText("")
	})

	s.overwriteRB.Clicked().Attach(func() {
		s.disp.SetMode(true)
	})

	s.appendRB.Clicked().Attach(func() {
		s.disp.SetMode(false)
	})

	s.mw.Closing().Attach(func(cancel *bool, reason walk.CloseReason) {
		if s.mon != nil {
			s.mon.Stop()
			s.mon = nil
		}
	})
}

func (s *uiState) setRunning(running bool) {
	s.startBtn.SetEnabled(!running)
	s.stopBtn.SetEnabled(running)
	s.ipEdit.SetReadOnly(running)
	s.portEdit.SetReadOnly(running)
	s.cmdEdit.SetReadOnly(running)
	s.intervalEdit.SetReadOnly(running)
}
