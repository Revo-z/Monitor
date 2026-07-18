package main

import (
	"sync/atomic"
	"time"
)

// telnetConn 定义 Telnet 连接接口，Monitor 依赖此接口而非具体 *Conn。
type telnetConn interface {
	ReadBytes(delim byte) ([]byte, error)
	Write(cmd string) error
	Close() error
}

// 编译期检查 *Conn 实现了 telnetConn
var _ telnetConn = (*Conn)(nil)

// DisplayCallback 回显回调函数类型
type DisplayCallback func(text string)

// readResult 读 goroutine 发往主循环的数据
type readResult struct {
	data string
	err  error
}

// Monitor 管理监控循环。
type Monitor struct {
	conn     telnetConn
	cmd      string
	interval time.Duration
	timeout  time.Duration

	stopChan chan struct{}
	doneChan chan struct{}
	running  int32
}

// NewMonitor 创建监控器实例。
func NewMonitor(conn telnetConn, cmd string, intervalSec float64) *Monitor {
	return &Monitor{
		conn:     conn,
		cmd:      cmd,
		interval: time.Duration(intervalSec * float64(time.Second)),
		timeout:  Timeout,
	}
}

// Start 启动监控循环。若已在运行则无操作。
func (m *Monitor) Start(callback DisplayCallback) {
	if !atomic.CompareAndSwapInt32(&m.running, 0, 1) {
		return
	}

	m.stopChan = make(chan struct{})
	m.doneChan = make(chan struct{})

	readChan := make(chan readResult, 1)

	// 读 goroutine：持续读取 Telnet 直到检测到 #，通过 channel 发送完整块
	go func() {
		for {
			data, err := m.conn.ReadBytes(EndMarker)
			res := readResult{data: string(data), err: err}
			select {
			case readChan <- res:
			case <-m.stopChan:
				return
			}
			if err != nil {
				return
			}
		}
	}()

	// 监控主循环 goroutine
	go func() {
		defer close(m.doneChan)
		defer atomic.StoreInt32(&m.running, 0)
		defer m.conn.Close()

		// 立即发送第一条命令，不等间隔
		if err := m.conn.Write(m.cmd); err != nil {
			LogError("发送命令失败: " + err.Error())
			callback("[错误] 发送命令失败: " + err.Error())
			return
		}

		timeoutTimer := time.NewTimer(m.timeout)
		defer timeoutTimer.Stop()

		for {
			select {
			case res := <-readChan:
				if res.err != nil {
					LogError("读取错误: " + res.err.Error())
					callback("[错误] " + res.err.Error() + "，连接已断开")
					return
				}
				callback(res.data)

				// 重置超时 timer
				if !timeoutTimer.Stop() {
					<-timeoutTimer.C
				}
				timeoutTimer.Reset(m.timeout)

				// 等待间隔后发送下一条命令
				select {
				case <-time.After(m.interval):
					if err := m.conn.Write(m.cmd); err != nil {
						LogError("发送命令失败: " + err.Error())
						callback("[错误] 发送命令失败，连接已断开")
						return
					}
				case <-m.stopChan:
					return
				}

			case <-timeoutTimer.C:
				LogError("命令执行超时")
				callback("[错误] 命令执行超时，连接已断开")
				return

			case <-m.stopChan:
				return
			}
		}
	}()
}

// Stop 停止监控循环，断开连接，等待 goroutine 退出。
func (m *Monitor) Stop() {
	if !atomic.CompareAndSwapInt32(&m.running, 1, 2) {
		return
	}
	close(m.stopChan)
	m.conn.Close()
	<-m.doneChan
	atomic.StoreInt32(&m.running, 0)
}

// IsRunning 返回当前是否正在监控。
func (m *Monitor) IsRunning() bool {
	return atomic.LoadInt32(&m.running) == 1
}
