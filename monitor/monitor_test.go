package monitor

import (
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

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
	return &mockTelnetConn{readResponses: make(chan mockReadResponse, 10)}
}
func (m *mockTelnetConn) ReadBytes(delim byte) ([]byte, error) {
	resp, ok := <-m.readResponses
	if !ok { return nil, io.EOF }
	return []byte(resp.data), resp.err
}
func (m *mockTelnetConn) Write(cmd string) error {
	m.mu.Lock(); m.writeCalls = append(m.writeCalls, cmd); m.mu.Unlock()
	return nil
}
func (m *mockTelnetConn) Close() error {
	m.mu.Lock(); defer m.mu.Unlock()
	if !m.closeCalled { m.closeCalled = true; close(m.readResponses) }
	return nil
}
func (m *mockTelnetConn) writeCallCount() int {
	m.mu.Lock(); defer m.mu.Unlock(); return len(m.writeCalls)
}
func (m *mockTelnetConn) isClosed() bool {
	m.mu.Lock(); defer m.mu.Unlock(); return m.closeCalled
}

func TestMonitorStartSendsFirstCommandImmediately(t *testing.T) {
	mock := newMockTelnetConn()
	mock.readResponses <- mockReadResponse{data: "data"}
	mon := NewMonitor(mock, "test-cmd", 0.5)
	mon.Start(func(text string) {})
	time.Sleep(50 * time.Millisecond)
	if mock.writeCallCount() < 1 {
		t.Errorf("want Write called, got %d", mock.writeCallCount())
	}
	mon.Stop()
}

func TestMonitorReceivesResponseAndCallsCallback(t *testing.T) {
	mock := newMockTelnetConn()
	mock.readResponses <- mockReadResponse{data: "hello"}
	mon := NewMonitor(mock, "test-cmd", 0.5)
	cbChan := make(chan string, 1)
	mon.Start(func(text string) { cbChan <- text })
	select {
	case text := <-cbChan:
		if text != "hello" { t.Errorf("want hello, got %q", text) }
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for callback")
	}
	mon.Stop()
}

func TestMonitorSendsSecondCommandAfterInterval(t *testing.T) {
	mock := newMockTelnetConn()
	mock.readResponses <- mockReadResponse{data: "r1"}
	mock.readResponses <- mockReadResponse{data: "r2"}
	mon := NewMonitor(mock, "test-cmd", 0.05)
	cbChan := make(chan string, 2)
	mon.Start(func(text string) { cbChan <- text })
	<-cbChan
	select {
	case <-cbChan:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for second callback")
	}
	if mock.writeCallCount() < 2 {
		t.Errorf("want Write called >= 2, got %d", mock.writeCallCount())
	}
	mon.Stop()
}

func TestMonitorTimeout(t *testing.T) {
	mock := newMockTelnetConn()
	mon := NewMonitor(mock, "test-cmd", 0.5)
	mon.timeout = 100 * time.Millisecond
	cbChan := make(chan string, 1)
	mon.Start(func(text string) { cbChan <- text })
	select {
	case text := <-cbChan:
		if !strings.Contains(text, "[") { t.Errorf("want error, got %q", text) }
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for timeout callback")
	}
	time.Sleep(100 * time.Millisecond)
	if !mock.isClosed() { t.Error("connection should be closed after timeout") }
}

func TestMonitorStop(t *testing.T) {
	mock := newMockTelnetConn()
	mock.readResponses <- mockReadResponse{data: "data"}
	mon := NewMonitor(mock, "test-cmd", 0.5)
	cbChan := make(chan string, 1)
	mon.Start(func(text string) { cbChan <- text })
	<-cbChan
	mon.Stop()
	if !mock.isClosed() { t.Error("connection should be closed after Stop") }
}

func TestMonitorDoubleStart(t *testing.T) {
	mock := newMockTelnetConn()
	mock.readResponses <- mockReadResponse{data: "d1"}
	mock.readResponses <- mockReadResponse{data: "d2"}
	mon := NewMonitor(mock, "test-cmd", 0.1)
	mon.Start(func(text string) {})
	mon.Start(func(text string) {})
	if !mon.IsRunning() { t.Error("IsRunning should be true after first Start") }
	mon.Stop()
}

func TestMonitorStopWhenIdle(t *testing.T) {
	mock := newMockTelnetConn()
	mon := NewMonitor(mock, "test-cmd", 0.5)
	mon.Stop()
}

func TestMonitorReadError(t *testing.T) {
	mock := newMockTelnetConn()
	mock.readResponses <- mockReadResponse{err: io.ErrUnexpectedEOF}
	mon := NewMonitor(mock, "test-cmd", 0.5)
	cbChan := make(chan string, 1)
	mon.Start(func(text string) { cbChan <- text })
	select {
	case text := <-cbChan:
		if !strings.Contains(text, "[") { t.Errorf("want error, got %q", text) }
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for error callback")
	}
	time.Sleep(50 * time.Millisecond)
	if !mock.isClosed() { t.Error("connection should be closed after read error") }
}

func TestMonitorIsRunning(t *testing.T) {
	mock := newMockTelnetConn()
	mock.readResponses <- mockReadResponse{data: "x"}
	mon := NewMonitor(mock, "test-cmd", 0.5)
	if mon.IsRunning() { t.Error("IsRunning should be false before Start") }
	mon.Start(func(text string) {})
	if !mon.IsRunning() { t.Error("IsRunning should be true after Start") }
	mon.Stop()
	if mon.IsRunning() { t.Error("IsRunning should be false after Stop") }
}

func TestMonitorCallbackOnMultipleResponses(t *testing.T) {
	mock := newMockTelnetConn()
	mock.readResponses <- mockReadResponse{data: "a"}
	mock.readResponses <- mockReadResponse{data: "b"}
	mock.readResponses <- mockReadResponse{data: "c"}
	mon := NewMonitor(mock, "test-cmd", 0.03)
	count := 0; done := make(chan int, 1)
	mon.Start(func(text string) { count++; if count == 3 { done <- count } })
	select {
	case c := <-done:
		if c != 3 { t.Errorf("want 3 callbacks, got %d", c) }
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout, got %d callbacks", count)
	}
	mon.Stop()
}
