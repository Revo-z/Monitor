package display

import (
	"strings"
	"testing"
)

func TestNewDisplayEmpty(t *testing.T) {
	d := NewDisplay()
	if d.Text() != "" {
		t.Errorf("新建 Display 应为空，实际: %q", d.Text())
	}
}

func TestOverwriteMode(t *testing.T) {
	d := NewDisplay()
	d.Write("a")
	d.Write("b")
	if d.Text() != "b" {
		t.Errorf("覆盖模式: 期望 %q, 实际 %q", "b", d.Text())
	}
}

func TestAppendMode(t *testing.T) {
	d := NewDisplay()
	d.SetMode(false)
	d.Write("a")
	d.Write("b")
	if d.Text() != "a\nb" {
		t.Errorf("追加模式: 期望 %q, 实际 %q", "a\nb", d.Text())
	}
}

func TestAppendModeFirstWriteNoNewline(t *testing.T) {
	d := NewDisplay()
	d.SetMode(false)
	d.Write("a")
	if d.Text() != "a" {
		t.Errorf("追加模式首次写入不应有前置换行: %q", d.Text())
	}
}

func TestClear(t *testing.T) {
	d := NewDisplay()
	d.Write("hello")
	d.Clear()
	if d.Text() != "" {
		t.Errorf("Clear 后应为空，实际: %q", d.Text())
	}
}

func TestModeSwitch(t *testing.T) {
	d := NewDisplay()
	d.SetMode(false)
	d.Write("line1")
	d.Write("line2")
	d.SetMode(true)
	d.Write("overwritten")
	if d.Text() != "overwritten" {
		t.Errorf("切换为覆盖模式后: 期望 %q, 实际 %q", "overwritten", d.Text())
	}
}

func TestWriteEmpty(t *testing.T) {
	d := NewDisplay()
	d.Write("")
	if d.Text() != "" {
		t.Errorf("Write 空字符串: 期望空, 实际 %q", d.Text())
	}
}

func TestMultipleClear(t *testing.T) {
	d := NewDisplay()
	d.Write("data")
	d.Clear()
	d.Clear()
	if d.Text() != "" {
		t.Errorf("连续两次 Clear 后应为空, 实际: %q", d.Text())
	}
}

func TestSetModeTwice(t *testing.T) {
	d := NewDisplay()
	d.SetMode(false)
	d.SetMode(false)
	d.Write("a")
	d.Write("b")
	if !strings.Contains(d.Text(), "a") || !strings.Contains(d.Text(), "b") {
		t.Errorf("SetMode 重复调用不应改变行为: %q", d.Text())
	}
}

func TestConsecutiveWritesOverwrite(t *testing.T) {
	d := NewDisplay()
	writes := []string{"1", "2", "3", "4", "5"}
	for _, w := range writes {
		d.Write(w)
	}
	if d.Text() != "5" {
		t.Errorf("覆盖模式连续 5 次写入, 最终应为最后一次: %q", d.Text())
	}
}

func TestConsecutiveWritesAppend(t *testing.T) {
	d := NewDisplay()
	d.SetMode(false)
	writes := []string{"1", "2", "3", "4", "5"}
	for _, w := range writes {
		d.Write(w)
	}
	lines := strings.Split(d.Text(), "\n")
	if len(lines) != 5 {
		t.Errorf("追加模式连续 5 次写入, 期望 5 行, 实际 %d 行", len(lines))
	}
}
