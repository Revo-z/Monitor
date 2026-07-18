package main

import (
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockTelnetConn 实现 telnetConn 接口，用于测试。
type mockTelnetConn struct {
	readResponses chan mockReadResponse
	writeCalls    []string
	closeCalled   bool
	mu            sync.Mutex
}

type mockReadResponse struct {
	data string
	err  error
}

func newMockTelnetConn() *mockTelnetConn {
	return &mockTelnetConn{
		readResponses: make(chan mockReadResponse, 10),
	}
}

func (m *mockTelnetConn) ReadBytes(delim byte) ([]byte, error) {
	resp, ok := <-m.readResponses
	if !ok {
		return nil, io.EOF
	}
	return []byte(resp.data), resp.err
}

func (m *mockTelnetConn) Write(cmd string) error {
	m.mu.Lock()
	m.writeCalls = append(m.writeCalls, cmd)
	m.mu.Unlock()
	return nil
}

func (m *mockTelnetConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closeCalled {
		m.closeCalled = true
		close(m.readResponses)
	}
	return nil
}

func (m *mockTelnetConn) writeCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.writeCalls)
}

func (m *mockTelnetConn) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeCalled
}

func TestMonitorStartSendsFirstCommandImmediately(t *testing.T) {
	mock := newMockTelnetConn()
	mock.readResponses <- mockReadResponse{data: "data"}

	mon := NewMonitor(mock, "test-cmd", 0.5)
	mon.Start(func(text string) {})

	time.Sleep(50 * time.Millisecond)

	if mock.writeCallCount() < 1 {
		t.Errorf("期望 Write 被调用，实际 %d 次", mock.writeCallCount())
	}

	mon.Stop()
}

func TestMonitorReceivesResponseAndCallsCallback(t *testing.T) {
	mock := newMockTelnetConn()
	mock.readResponses <- mockReadResponse{data: "hello"}

	mon := NewMonitor(mock, "test-cmd", 0.5)

	cbChan := make(chan string, 1)
	mon.Start(func(text string) {
		cbChan <- text
	})

	select {
	case text := <-cbChan:
		if text != "hello" {
			t.Errorf("期望 hello, 实际 %q", text)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("等待回调超时")
	}

	mon.Stop()
}

func TestMonitorSendsSecondCommandAfterInterval(t *testing.T) {
	mock := newMockTelnetConn()
	mock.readResponses <- mockReadResponse{data: "r1"}
	mock.readResponses <- mockReadResponse{data: "r2"}

	mon := NewMonitor(mock, "test-cmd", 0.05)

	cbChan := make(chan string, 2)
	mon.Start(func(text string) {
		cbChan <- text
	})

	<-cbChan // 第一个
	select {
	case <-cbChan: // 第二个（间隔后到达）
	case <-time.After(500 * time.Millisecond):
		t.Fatal("等待第二个回调超时")
	}

	if mock.writeCallCount() < 2 {
		t.Errorf("期望 Write 调用 >= 2 次，实际 %d", mock.writeCallCount())
	}

	mon.Stop()
}

func TestMonitorTimeout(t *testing.T) {
	mock := newMockTelnetConn()

	mon := NewMonitor(mock, "test-cmd", 0.5)
	mon.timeout = 100 * time.Millisecond

	cbChan := make(chan string, 1)
	mon.Start(func(text string) {
		cbChan <- text
	})

	select {
	case text := <-cbChan:
		if !strings.Contains(text, "超时") {
			t.Errorf("期望超时错误消息，实际 %q", text)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("等待超时回调超时")
	}

	time.Sleep(100 * time.Millisecond) // 让 defer 执行

	if !mock.isClosed() {
		t.Error("超时后连接应被关闭")
	}
}

func TestMonitorStop(t *testing.T) {
	mock := newMockTelnetConn()
	mock.readResponses <- mockReadResponse{data: "data"}

	mon := NewMonitor(mock, "test-cmd", 0.5)

	cbChan := make(chan string, 1)
	mon.Start(func(text string) {
		cbChan <- text
	})

	<-cbChan
	mon.Stop()

	if !mock.isClosed() {
		t.Error("Stop 后连接应被关闭")
	}
}

func TestMonitorDoubleStart(t *testing.T) {
	mock := newMockTelnetConn()
	mock.readResponses <- mockReadResponse{data: "d1"}
	mock.readResponses <- mockReadResponse{data: "d2"}

	mon := NewMonitor(mock, "test-cmd", 0.1)

	firstCount := 0
	mon.Start(func(text string) { firstCount++ })

	// 第二次不应启动新 goroutine
	secondCount := 0
	mon.Start(func(text string) { secondCount++ })

	if !mon.IsRunning() {
		t.Error("第一次 Start 后 IsRunning 应为 true")
	}

	mon.Stop()
}

func TestMonitorStopWhenIdle(t *testing.T) {
	mock := newMockTelnetConn()
	mon := NewMonitor(mock, "test-cmd", 0.5)

	// 未启动时调用 Stop 不应 panic
	mon.Stop()
}

func TestMonitorReadError(t *testing.T) {
	mock := newMockTelnetConn()
	mock.readResponses <- mockReadResponse{err: io.ErrUnexpectedEOF}

	mon := NewMonitor(mock, "test-cmd", 0.5)

	cbChan := make(chan string, 1)
	mon.Start(func(text string) {
		cbChan <- text
	})

	select {
	case text := <-cbChan:
		if !strings.Contains(text, "[错误]") || !strings.Contains(text, "断开") {
			t.Errorf("期望错误消息，实际 %q", text)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("等待错误回调超时")
	}

	time.Sleep(50 * time.Millisecond)
	if !mock.isClosed() {
		t.Error("读错误后连接应被关闭")
	}
}

func TestMonitorIsRunning(t *testing.T) {
	mock := newMockTelnetConn()
	mock.readResponses <- mockReadResponse{data: "x"}

	mon := NewMonitor(mock, "test-cmd", 0.5)

	if mon.IsRunning() {
		t.Error("启动前 IsRunning 应为 false")
	}

	mon.Start(func(text string) {})
	if !mon.IsRunning() {
		t.Error("启动后 IsRunning 应为 true")
	}

	mon.Stop()
	if mon.IsRunning() {
		t.Error("停止后 IsRunning 应为 false")
	}
}

func TestMonitorCallbackOnMultipleResponses(t *testing.T) {
	mock := newMockTelnetConn()
	mock.readResponses <- mockReadResponse{data: "a"}
	mock.readResponses <- mockReadResponse{data: "b"}
	mock.readResponses <- mockReadResponse{data: "c"}

	mon := NewMonitor(mock, "test-cmd", 0.03)

	count := 0
	done := make(chan int, 1)
	mon.Start(func(text string) {
		count++
		if count == 3 {
			done <- count
		}
	})

	select {
	case c := <-done:
		if c != 3 {
			t.Errorf("期望 3 次回调，实际 %d", c)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("等待回调超时，已收到 %d 次", count)
	}

	mon.Stop()
}
