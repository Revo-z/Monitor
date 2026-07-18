package monitor

import (
	"sync/atomic"
	"time"

	"Monitor/config"
	"Monitor/logger"
	"Monitor/telnet"
)

type DisplayCallback func(text string)

type readResult struct {
	data string
	err  error
}

type Monitor struct {
	conn     telnet.Connector
	cmd      string
	interval time.Duration
	timeout  time.Duration

	stopChan chan struct{}
	doneChan chan struct{}
	running  int32
}

func NewMonitor(conn telnet.Connector, cmd string, intervalSec float64) *Monitor {
	return &Monitor{
		conn:     conn,
		cmd:      cmd,
		interval: time.Duration(intervalSec * float64(time.Second)),
		timeout:  config.Timeout,
	}
}

func (m *Monitor) Start(callback DisplayCallback) {
	if !atomic.CompareAndSwapInt32(&m.running, 0, 1) {
		return
	}
	m.stopChan = make(chan struct{})
	m.doneChan = make(chan struct{})
	readChan := make(chan readResult, 1)

	go func() {
		for {
			data, err := m.conn.ReadBytes(config.EndMarker)
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

	go func() {
		defer close(m.doneChan)
		defer atomic.StoreInt32(&m.running, 0)
		defer m.conn.Close()

		if err := m.conn.Write(m.cmd); err != nil {
			logger.LogError("发送命令失败: " + err.Error())
			callback("[错误] 发送命令失败: " + err.Error())
			return
		}

		timeoutTimer := time.NewTimer(m.timeout)
		defer timeoutTimer.Stop()

		for {
			select {
			case res := <-readChan:
				if res.err != nil {
					logger.LogError("读取错误: " + res.err.Error())
					callback("[错误] " + res.err.Error() + "，连接已断开")
					return
				}
				callback(res.data)

				if !timeoutTimer.Stop() {
					<-timeoutTimer.C
				}
				timeoutTimer.Reset(m.timeout)

				select {
				case <-time.After(m.interval):
					if err := m.conn.Write(m.cmd); err != nil {
						logger.LogError("发送命令失败: " + err.Error())
						callback("[错误] 发送命令失败，连接已断开")
						return
					}
				case <-m.stopChan:
					return
				}

			case <-timeoutTimer.C:
				logger.LogError("命令执行超时")
				callback("[错误] 命令执行超时，连接已断开")
				return

			case <-m.stopChan:
				return
			}
		}
	}()
}

func (m *Monitor) Stop() {
	if !atomic.CompareAndSwapInt32(&m.running, 1, 2) {
		return
	}
	close(m.stopChan)
	m.conn.Close()
	<-m.doneChan
	atomic.StoreInt32(&m.running, 0)
}

func (m *Monitor) IsRunning() bool {
	return atomic.LoadInt32(&m.running) == 1
}
