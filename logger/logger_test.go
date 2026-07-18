package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogErrorCreatesFile(t *testing.T) {
	orig := logPath
	defer func() { logPath = orig }()
	logPath = filepath.Join(t.TempDir(), "error.log")
	LogError("test message")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatal("日志文件未创建")
	}
}

func TestLogErrorFormat(t *testing.T) {
	orig := logPath
	defer func() { logPath = orig }()
	logPath = filepath.Join(t.TempDir(), "error.log")
	LogError("hello world")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.HasPrefix(content, "[20") {
		t.Errorf("日志行未包含时间戳前缀 '[20': %s", content)
	}
	if !strings.Contains(content, "hello world") {
		t.Errorf("日志行未包含消息文本: %s", content)
	}
}

func TestLogErrorAppend(t *testing.T) {
	orig := logPath
	defer func() { logPath = orig }()
	logPath = filepath.Join(t.TempDir(), "error.log")
	LogError("first")
	LogError("second")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("期望 2 行，实际 %d 行: %v", len(lines), lines)
	}
	if !strings.Contains(lines[1], "second") {
		t.Errorf("第二行未包含 'second': %s", lines[1])
	}
}

func TestLogErrorEmptyMsg(t *testing.T) {
	orig := logPath
	defer func() { logPath = orig }()
	logPath = filepath.Join(t.TempDir(), "error.log")
	LogError("")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.HasPrefix(content, "[20") {
		t.Error("即使消息为空，也应输出时间戳")
	}
}
