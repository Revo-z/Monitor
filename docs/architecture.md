# 架构文档

> 最后更新：2026-07-18 | 对应代码版本：`master`

---

## 概述

DevMonitor 是一个 Windows GUI 工具，通过 Telnet 连接远程设备，循环执行指定命令并将回显展示在界面上。

- 语言：Go 1.25，单模块 `Monitor`
- UI：walk（Win32 原生控件，本地补丁版位于 `third_party/walk/`）
- Telnet：标准库 `net` + 手写 IAC 协商
- 发布：`go build -ldflags "-H=windowsgui"` 单文件 exe

---

## 目录结构

```
Monitor/
├── main.go                 ← 入口，创建 display + 启动 UI
├── go.mod / go.sum         ← Go module 定义（含 replace 指向 third_party/walk）
├── .gitignore
├── AGENTS.md
├── docs/
│   ├── design.md           ← 产品设计规格
│   ├── plan.md             ← 开发步骤计划
│   └── architecture.md     ← 本文档
├── config/                 ← 所有可调常量 + 快捷命令定义
│   ├── config.go
│   └── config_test.go
├── display/                ← 显示缓冲区（纯数据，无 UI 依赖）
│   ├── display.go
│   └── display_test.go
├── logger/                 ← 错误日志，追加写 error.log
│   ├── logger.go
│   └── logger_test.go
├── telnet/                 ← Telnet 连接 + IAC 协商 + Connector 接口
│   ├── telnet.go
│   └── telnet_test.go
├── monitor/                ← 监控循环、超时控制、goroutine 管理
│   ├── monitor.go
│   └── monitor_test.go
├── ui/                     ← walk 窗口、控件、事件绑定
│   └── ui.go
└── third_party/walk/       ← walk 库本地副本（tooltip.go TTM_ADDTOOL 补丁）
```

---

## 架构分层

```
┌──────────────────────┐
│       main.go        │  入口：组装各模块
└────────┬─────────────┘
         │
┌────────▼─────────────┐
│     ui (ui/ui.go)    │  walk 窗口 + 回调转发
│     依赖: config       │
│     依赖: display      │
│     依赖: logger       │
│     依赖: telnet       │
│     依赖: monitor      │
└──┬────────┬─────┬────┘
   │        │     │
   │    ┌───▼──┐  │
   │    │monitor│  │      monitor 依赖 telnet.Connector 接口
   │    │       │  │      monitor 依赖 config.EndMarker/Timeout
   │    │       │  │      monitor 依赖 logger
   │    └───┬───┘  │
   │        │      │
   │   ┌────▼──┐   │
   │   │telnet │   │      telnet 定义 Connector 接口
   │   │       │   │      Conn 实现 Connector
   │   └───────┘   │
   │               │
   ├───────────────┤
   │   横向工具     │
   │  ┌─ config    │      所有可调参数集中定义
   │  ├─ display   │      纯数据，无任何依赖
   │  └─ logger    │      依赖 config.ErrorLogFile
   └───────────────┘
```

**依赖方向**：`main → ui → monitor → telnet`（单向），`config / display / logger` 为横向工具模块，被各层按需 import。

---

## 包设计

### `config` — 常量与配置

所有可调参数集中在此，无代码逻辑。

| 导出 | 说明 |
|---|---|
| `DefaultInterval`, `MinInterval`, `MaxInterval` | 刷新间隔（秒） |
| `Timeout` | 命令执行超时（10s） |
| `EndMarker` | 回显结束标志 `'#'` |
| `AppTitle`, `DefaultWidth`, `DefaultHeight` | 窗口属性 |
| `FontName`, `FontSize` | 显示区字体 |
| `ErrorLogFile` | 错误日志文件名 |
| `CommandShortcut` 结构体 | 快捷按钮定义 |
| `ShortcutCommands` 切片 | 5 条占位命令 |

### `display` — 显示缓冲区（纯数据层）

零外部依赖，仅依赖 `strings` 标准库。两种模式切换。

```
NewDisplay() *Display
  ├── SetMode(overwrite bool)     // true=覆盖, false=追加
  ├── Write(text string)          // 按模式写入
  ├── Clear()                     // 清空
  └── Text() string               // 读取当前内容
```

覆盖模式：每次 `Write` 先 Reset 再写入。
追加模式：`Write` 在末尾追加，首次不加前缀换行。

### `logger` — 错误日志

```
LogError(msg string)
```

格式：`[2006-01-02 15:04:05] 消息\n`
追加写入 `config.ErrorLogFile`（默认 `error.log`）。
文件打开/写入失败时静默返回。

日志路径通过 `logPath` 变量控制，测试中覆盖为 `t.TempDir()`。

### `telnet` — Telnet 协议层

```
Connector interface {                     // 供 monitor 依赖，便于 mock
    ReadBytes(delim byte) ([]byte, error)
    Write(cmd string) error
    Close() error
}

NewConn(host, port string) (*Conn, error) // 创建连接
Conn.ReadBytes(delim byte)                // 读取到 delim，过滤 IAC + 回复 WONT/DONT
Conn.Write(cmd string)                    // 发送命令，自动追加 \r\n
Conn.Close()                              // 断开
```

**IAC 协商处理**：逐字节读取。遇到 `IAC WILL` → 回复 `IAC DONT`；`IAC DO` → 回复 `IAC WONT`；`IAC WONT/DONT` → 忽略；`IAC SB ... IAC SE` → 整段丢弃；`IAC IAC` → 转为 0xFF 字面量。

内部拆分为两个函数：
- `filterTelnetWithReply`：面向真实连接，同时发送协商回复
- `filterTelnet`：纯过滤版本，`io.Reader` 输入，仅用于单元测试

### `monitor` — 监控循环

```
DisplayCallback func(text string)

NewMonitor(conn telnet.Connector, cmd string, intervalSec float64) *Monitor
Monitor.Start(callback DisplayCallback)   // 启动监控循环（非阻塞）
Monitor.Stop()                            // 停止循环 + 断开连接 + 等待 goroutine 退出
Monitor.IsRunning() bool
```

**生命周期状态机**：

```
running: 0 (idle) ──Start()──▶ 1 (running) ──Stop()──▶ 2 (stopping) ──▶ 0
                    CAS(0→1)              CAS(1→2)              goroutine defers
```

`Timeout` 为 unexported 字段，默认 `config.Timeout`，测试中可覆盖为短值。

### `ui` — 窗口与事件

```
RunUI(disp *display.Display) (int, error)
```

控件清单：IP 输入、端口输入、命令输入、间隔 NumberEdit（0.01–2，默认 0.5）、覆盖/追加 RadioButton、清空/开始/停止按钮、只读 TextEdit 显示区、5 个快捷按钮。

事件绑定：
- 开始 → 校验 → 创建 `telnet.Conn` → `monitor.Start` → 按钮置灰切换
- 停止 → `monitor.Stop` → 按钮置灰切换
- 清空 → `display.Clear` + UI 同步
- 模式切换 → `display.SetMode`
- 关闭 → `monitor.Stop` 确保释放
- 快捷按钮 → 填充命令输入框

Monitor 回调通过 `mw.Synchronize` 安全地更新 UI。

### `main` — 入口

```go
func main() {
    disp := display.NewDisplay()
    ui.RunUI(disp)
}
```

---

## goroutine 模型

```
Monitor.Start()
    │
    ├──→ readLoop goroutine      循环 conn.ReadBytes('#')
    │       │                     读到完整块 → readChan (chan readResult, buf=1)
    │       │                     读错误 → readChan (带 error)
    │       │                     收到 stopChan → 退出
    │
    └──→ monitorLoop goroutine   首次: 立即发送第一条命令
            │                     之后循环 select:
            ├── readChan ←─ 回显 → callback(data) → 重置 timeout timer
            │                                → 等待 interval → 发送下一条命令
            ├── readChan ←─ 错误 → LogError + callback(错误消息) + Close → 退出
            ├── timeoutTimer.C  → LogError + callback(超时消息) + Close → 退出
            └── stopChan       → 退出 (defer: Close + StoreInt32(0) + close(doneChan))
```

**超时**：每次发送命令后启动/重置 10s timer。回显到达时重置。超时触发则退出循环。

**停止**：`Stop()` → CAS(1→2) → close(stopChan) → conn.Close()（解除 readLoop 阻塞）→ <-doneChan（等待 monitorLoop 退出）。

---

## 接口设计

```
telnet.Connector  ←──  monitor.NewMonitor 的入参类型
    ▲
    │ 实现者
    ├── *telnet.Conn      （生产环境）
    └── *mockTelnetConn   （测试环境，channel 控制返回值）
```

`monitor` 包不依赖 `*telnet.Conn` 具体类型，只依赖 `telnet.Connector` 接口。测试用 `mockTelnetConn` 通过 buffered channel 模拟正常回显、读错误、超时阻塞等场景。

---

## 关键技术决策

| 决策 | 原因 |
|---|---|
| walk 本地补丁 | walk@v0.0.0-20210112085537 在当前 Windows 环境下 `TTM_ADDTOOL` 失败，导致全部控件创建失败。在 `third_party/walk/tooltip.go:addTool` 中跳过该错误 |
| `-ldflags "-H=windowsgui"` | walk 创建的是 GUI 窗口，不加此标志会出现控制台窗口 |
| 窗口居中 + `SetForegroundWindow` | 默认窗口位置可能不在可视区域 |
| `telnet.Connector` 接口 | 解耦 monitor 与具体连接实现，支持 mock 测试 |
| monitor timeout 为 struct field 而非 const | 便于测试中覆盖为短值（100ms），不影响生产代码 |
| UTF-8 编码（无 BOM） | Go 编译器在 `-cover` 等场景下会拒绝 BOM 文件 |
| 不回写配置 | 设计规格要求每次启动从零开始 |

---

## 构建与测试

```bash
# 构建 GUI exe
go build -ldflags "-H=windowsgui" -o DevMonitor.exe

# 全量测试 + 覆盖率
go test ./... -cover -count=1

# 静态检查
go vet ./...
```

覆盖率基线：`display` 100% / `monitor` ≥ 86% / `logger` ≥ 80% / `telnet` IAC 过滤全覆盖。
