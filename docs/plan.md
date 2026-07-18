# 开发计划

> 本计划供 AI coding agent 按序执行。每步完成后需 `go build` 编译通过再提交。

---

## 步骤 0：项目初始化

**操作：**
- 在项目根目录执行 `go mod init Monitor`
- 执行 `go get github.com/lxn/walk` 拉取 walk 依赖
- 执行 `go build` 确认空项目可编译（虽然还没代码，但 go mod tidy 后应无报错）

**验证：** `go.mod` 和 `go.sum` 生成，`go build` 返回成功（或仅 "no Go files" 警告）

---

## 步骤 1：config.go — 集中定义所有可调常量

**文件：** `config/config.go`

**内容：**
- 定义 `Config` 结构体（或直接定义 const 块），包含：
  - 默认刷新间隔：`0.5`（秒）
  - 最小间隔：`0.01`，最大间隔：`2.0`
  - 超时：`10` 秒
  - 回显结束标志：`'#'`
  - 窗口默认尺寸：`1000 × 700`
  - 窗口标题：`"DevMonitor"`
  - 显示区字体：`"Consolas"`，字号 `14`
  - 编码：`"UTF-8"`
  - 错误日志文件名：`"error.log"`
- 定义快捷命令列表：一个 `[]CommandShortcut` 切片，每个元素含 `Label`（按钮文本）和 `Cmd`（命令字符串）。占位内容：
  - CPU → `display cpu-usage`
  - 内存 → `display memory-usage`
  - 磁盘 → `display disk-usage`
  - 网络 → `display interface brief`
  - 系统 → `display device`

**验证：** `go build ./...`

---

## 步骤 2：logger.go — 错误日志写入

**文件：** `logger/logger.go`

**内容：**
- 导出函数 `LogError(msg string)`
- 以追加模式打开/创建程序同目录下的 `error.log`
- 写入格式：`[2006-01-02 15:04:05] 错误描述\n`
- 处理文件打开失败的情况（静默失败，不中断主流程）
- 每次调用都打开→写入→关闭（简单可靠，避免长生命周期文件句柄）

**验证：** `go build ./...`；可写一个临时 main 调用 `LogError("test")` 确认文件生成，然后删掉临时 main

---

## 步骤 3：display.go — 显示缓冲区管理

**文件：** `display/display.go`

**内容：**
- 导出 `Display` 结构体，内部字段不导出
- 导出构造 `NewDisplay() *Display`
- 方法：
  - `SetMode(overwrite bool)` — `true` 为覆盖模式，`false` 为追加模式
  - `Write(text string)` — 覆盖模式先清空再写入；追加模式在末尾追加（每次追加前加一个换行 `\n` 分隔，首次不加）
  - `Clear()` — 清空缓冲区
  - `Text() string` — 返回当前缓冲区内容
- 纯数据层，不依赖任何 UI 框架，不引入 `sync` 包（由调用方保证单 goroutine 访问）

**验证：** `go build ./...`

---

## 步骤 4：telnet.go — Telnet 协议层

**文件：** `telnet/telnet.go`

**内容：**
- 导出 `Conn` 结构体，封装 `net.Conn`
- 导出 `NewConn(host string, port string) (*Conn, error)`：
  - 拼接地址 `host:port`，调用 `net.DialTimeout("tcp", addr, 10*time.Second)` 建立连接
- 方法：
  - `ReadBytes(delim byte) ([]byte, error)` — 从连接读取字节直到遇到 delim，底层处理 IAC 协商：
    - 逐字节读取
    - 遇到 `IAC(0xFF)`：读下一个字节判断命令类型
      - `IAC WILL` → 回复 `IAC DONT <option>`
      - `IAC DO` → 回复 `IAC WONT <option>`
      - `IAC WONT` / `IAC DONT` → 忽略
      - `IAC SB` → 读取直到 `IAC SE`，忽略子协商内容
      - `IAC IAC` → 当作普通字节 `0xFF` 加入结果
    - 非 IAC 字节：追加到结果缓冲区
    - 遇到 delim 且非 IAC 前缀时返回累积的字节（不含 delim）
  - `Write(cmd string) error` — 发送命令，自动追加 `\r\n`
  - `Close() error` — 关闭连接
- IAC 常量集中定义在文件顶部：
  ```go
  const (
      iac  = 255
      will = 251
      wont = 252
      do   = 253
      dont = 254
      sb   = 250
      se   = 240
  )
  ```

**验证：** `go build ./...`

---

## 步骤 5：monitor.go — 监控循环与业务逻辑

**文件：** `monitor/monitor.go`

**内容：**
- 导出 `Monitor` 结构体（字段不导出）
- 导出 `NewMonitor(conn *Conn, cmd string, interval float64) *Monitor`
- 导出 `DisplayCallback func(text string)` 回调类型
- 方法：
  - `Start(callback DisplayCallback) error`：
    1. 将状态置为"运行中"
    2. 启动**读 goroutine**：循环调用 `conn.ReadBytes('#')`，读到完整块后通过 channel（`chan string`）发给主循环
    3. 启动**监控主循环 goroutine**：
       - 先立即发送第一条命令 `conn.Write(cmd)`
       - 然后进入循环：
         a. 用 `select` 同时监听：回显 channel、10 秒超时 timer、停止信号 channel
         b. 收到回显 → 调用 `callback(text)` → 重置超时 timer → 启动间隔等待（`time.After` 对应 interval）→ 间隔到期后发送下一条命令 → 回到 a
         c. 超时触发 → 调用 `LogError("命令执行超时")` → 调用 `callback("[错误] 命令执行超时，连接已断开")` → 断开连接 → 退出循环
         d. 收到停止信号 → 退出循环
  - `Stop()`：
    - 发送停止信号
    - 关闭 Telnet 连接
    - 等待 goroutine 退出（通过 done channel 或 WaitGroup）
    - 将状态置为"已停止"
  - `IsRunning() bool` — 返回当前是否正在监控
- 状态管理用 channel 通信，不加 mutex

**要点：**
- 读 goroutine 中 `ReadBytes` 返回的 error 直接走超时/断连逻辑（通过网络错误 channel 通知主循环）
- 间隔 timer 在每次收到完整回显后重置，避免多 timer 堆积
- 防重入：`Start` 调用时若 `IsRunning()` 为 true，直接 return（不重复启动）

**验证：** `go build ./...`

---

## 步骤 6：ui.go — 窗口布局与事件绑定

**文件：** `ui/ui.go`

**内容（使用 walk 库）：**

### 6.1 窗口创建
- 创建 `walk.MainWindow`，标题 `"DevMonitor"`，初始尺寸 `1000×700`，允许自由缩放
- 窗口关闭时调用 `monitor.Stop()` 确保资源释放

### 6.2 控件布局
- 使用 `walk.VBoxLayout` 或手动布局（建议用 Composite + GridLayout / 手动 SetBounds）
- 从上到下：
  1. **顶部参数区**（第一行）：IP 输入框 + 端口输入框
  2. **第二行**：监控命令输入框 + 刷新间隔 NumberEdit + "秒"标签
  3. **第三行**：显示模式 GroupBox（含两个 RadioButton：覆盖/追加，默认覆盖）+ "清空显示"按钮
  4. **第四行**："开始"按钮 + "停止"按钮
  5. **显示区**：多行只读 TextEdit，占窗口主体，VScroll，Consolas 14pt，白底黑字
  6. **底部快捷按钮栏**：根据 `config/config.go` 中的快捷命令列表动态生成 PushButton

### 6.3 控件属性
| 控件 | walk 类型 | 属性 |
|---|---|---|
| IP 输入 | `LineEdit` | 无默认值 |
| 端口输入 | `LineEdit` | 无默认值 |
| 命令输入 | `LineEdit` | 无默认值 |
| 间隔输入 | `NumberEdit` | 范围 0.01–2，默认 0.5，2 位小数 |
| 覆盖 Radio | `RadioButton` | 默认选中，文本"覆盖" |
| 追加 Radio | `RadioButton` | 文本"追加" |
| 清空按钮 | `PushButton` | 文本"清空显示" |
| 开始按钮 | `PushButton` | 文本"开始" |
| 停止按钮 | `PushButton` | 文本"停止"，初始置灰 |
| 显示区 | `TextEdit` | ReadOnly, VScroll, 等宽字体 |
| 快捷按钮 | `PushButton` × N | 文本为 Label |

### 6.4 事件绑定
- **"开始"按钮 clicked**：
  1. 校验 IP 非空、端口为合法数字（1–65535）、命令非空
  2. 校验失败 → 弹出 `walk.MsgBox` 提示
  3. 校验通过 → 创建 `telnet.NewConn(ip, port)` → 创建 `monitor.NewMonitor(conn, cmd, interval)`
  4. 定义 `DisplayCallback`：内部调用 `display.Write(text)`，然后通过 `walk.InvokeSync` 更新显示区内容
  5. 调用 `monitor.Start(callback)`
  6. 更新按钮状态：开始置灰、停止可用
  7. 连接失败时弹框提示错误并记日志
- **"停止"按钮 clicked**：
  1. 调用 `monitor.Stop()`
  2. 更新按钮状态：开始可用、停止置灰
- **"清空显示" clicked**：调用 `display.Clear()` + 更新显示区
- **显示模式 RadioButton clicked**：更新 `display.SetMode()`（覆盖=`true`，追加=`false`）
- **窗口关闭**：调用 `monitor.Stop()`，然后 `mainWindow.Close()`
- **快捷按钮 clicked**：将对应命令写入命令输入框

### 6.5 端口校验
- 点击"开始"时校验端口字符串：
  - 非空、纯数字、范围 1–65535
  - 不合法则弹框提示

**验证：** `go build ./...`

---

## 步骤 7：main.go — 入口组装

**文件：** `main.go`（根目录）

**内容：**
- `package main`
- `func main()`：
  - 创建 `Display` 实例
  - 调用 `ui.RunApp(display)` 启动 UI（`ui/ui.go` 导出 `RunApp` 函数，接收 `*Display`）
  - walk 的 `RunApp` 等价于创建窗口并进入消息循环

> 说明：`ui/ui.go` 负责创建窗口以及持有 `*Monitor`、`*Display` 等实例的引用，`main.go`（根目录） 仅做组装。

**验证：** `go build ./...`

---

## 步骤 8：集成验证

**操作：**
- 执行 `go build -o DevMonitor.exe`
- 检查生成的 `DevMonitor.exe` 文件
- 执行 `go vet ./...` 做静态检查
- 如有可测试的逻辑（如 `display/display.go` 的纯函数），编写 `display/display_test.go` 做单元测试

**验证：** `go build` 无报错，`go vet` 无告警

---

## 提交节奏

| 步骤 | commit message |
|---|---|
| 0 | `初始化: Go module 及 walk 依赖` |
| 1 | `config.go: 集中定义可调常量与快捷命令列表` |
| 2 | `logger.go: 实现错误日志追加写入` |
| 3 | `display.go: 实现显示缓冲区与覆盖/追加模式` |
| 4 | `telnet.go: 实现 Telnet 连接与 IAC 协商处理` |
| 5 | `monitor.go: 实现监控循环、超时控制与 goroutine 管理` |
| 6 | `ui.go: 实现窗口布局与事件绑定` |
| 7 | `main.go: 实现入口组装` |
| 8 | `集成: 最终编译验证与静态检查` |
