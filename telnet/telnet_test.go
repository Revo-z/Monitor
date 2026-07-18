package telnet

import (
	"bytes"
	"testing"
)

func iacBytes(b ...byte) []byte {
	return b
}

func TestFilterTelnetPlainText(t *testing.T) {
	data, err := filterTelnet(bytes.NewBuffer(iacBytes('h', 'e', 'l', 'l', 'o', '#')), '#')
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Errorf("期望 hello, 实际 %q", string(data))
	}
}

func TestFilterTelnetIAC_WILL(t *testing.T) {
	input := iacBytes(0xFF, 0xFB, 0x01, 'd', 'a', 't', 'a', '#')
	data, err := filterTelnet(bytes.NewBuffer(input), '#')
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "data" {
		t.Errorf("期望 data, 实际 %q", string(data))
	}
}

func TestFilterTelnetIAC_DO(t *testing.T) {
	input := iacBytes(0xFF, 0xFD, 0x03, 'a', 'b', 'c', '#')
	data, err := filterTelnet(bytes.NewBuffer(input), '#')
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "abc" {
		t.Errorf("期望 abc, 实际 %q", string(data))
	}
}

func TestFilterTelnetIAC_WONT(t *testing.T) {
	input := iacBytes(0xFF, 0xFC, 0x18, 'x', '#')
	data, err := filterTelnet(bytes.NewBuffer(input), '#')
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "x" {
		t.Errorf("期望 x, 实际 %q", string(data))
	}
}

func TestFilterTelnetIAC_DONT(t *testing.T) {
	input := iacBytes(0xFF, 0xFE, 0x20, 'y', '#')
	data, err := filterTelnet(bytes.NewBuffer(input), '#')
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "y" {
		t.Errorf("期望 y, 实际 %q", string(data))
	}
}

func TestFilterTelnetIAC_Subnegotiation(t *testing.T) {
	input := iacBytes(0xFF, 0xFA, 0x18, 0x00, 0x4D, 0x53, 0x54, 0xFF, 0xF0, 'e', 'n', 'd', '#')
	data, err := filterTelnet(bytes.NewBuffer(input), '#')
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "end" {
		t.Errorf("期望 end, 实际 %q", string(data))
	}
}

func TestFilterTelnetIAC_Escaped(t *testing.T) {
	input := iacBytes('v', 'a', 'l', 0xFF, 0xFF, 'u', 'e', '#')
	data, err := filterTelnet(bytes.NewBuffer(input), '#')
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "val\xFFue" {
		t.Errorf("期望 val\\xFFue, 实际 %q", string(data))
	}
}

func TestFilterTelnetEmptyAfterIAC(t *testing.T) {
	input := iacBytes(0xFF, 0xFB, 0x01, '#')
	data, err := filterTelnet(bytes.NewBuffer(input), '#')
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "" {
		t.Errorf("期望空字符串, 实际 %q", string(data))
	}
}

func TestFilterTelnetNoDelimiter(t *testing.T) {
	_, err := filterTelnet(bytes.NewBuffer(iacBytes('h', 'e', 'l', 'l', 'o')), '#')
	if err == nil {
		t.Fatal("期望返回错误（无定界符），实际为 nil")
	}
}
