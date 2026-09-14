# 第 25 章 · 网络编程基础与 TCP/UDP

前几章我们掌握了并发、context 和 runtime 机制，现在是时候将这些知识应用到实际的网络编程中了。Go 的 `net` 包提供了简洁而强大的网络编程接口，让我们能够轻松编写高性能的网络服务。本章将深入 TCP 和 UDP 编程：如何监听端口、接受连接、处理粘包问题、设置超时、实现自定义协议，以及如何用"每连接一个 goroutine"模式构建并发服务器。这些技能是编写 Web 服务、RPC 框架、消息队列客户端等网络应用的基础。

本章配套代码在 `internal/chapter/go25_net/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

## 25.1 net 包基础与 TCP Echo 服务

`net` 包是 Go 网络编程的核心，提供了跨平台的网络 I/O 接口。

**核心函数**：

- `net.Listen(network, address string)` - 监听网络地址，返回 `Listener`
- `net.Dial(network, address string)` - 连接到网络地址，返回 `Conn`
- `listener.Accept()` - 接受一个连接，阻塞直到有新连接
- `conn.Read([]byte)` / `conn.Write([]byte)` - 读写数据

**最简单的 TCP Echo 服务器**：

```go
// 服务器：监听端口
listener, err := net.Listen("tcp", "127.0.0.1:8080")
if err != nil {
    log.Fatal(err)
}
defer listener.Close()

// 接受连接
conn, err := listener.Accept()
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

// Echo：读取数据并原样返回
buf := make([]byte, 1024)
n, err := conn.Read(buf)
if err != nil {
    log.Fatal(err)
}
conn.Write(buf[:n])
```

**客户端**：

```go
// 连接服务器
conn, err := net.Dial("tcp", "127.0.0.1:8080")
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

// 发送数据
conn.Write([]byte("Hello, TCP!"))

// 接收响应
buf := make([]byte, 1024)
n, err := conn.Read(buf)
fmt.Println(string(buf[:n])) // 输出：Hello, TCP!
```

**输出示例**：

```
TCP Echo 服务器监听在 127.0.0.1:54321
发送：Hello, TCP!
接收：Hello, TCP!
```

**关键点**：

- `Listen` 的地址可以是 `:8080`（所有接口）、`127.0.0.1:8080`（本地）、或 `:0`（自动分配端口）
- `Conn` 实现了 `io.Reader` 和 `io.Writer`，可以用 `io.Copy` 等标准库函数
- `Accept` 和 `Read` 都是阻塞调用，返回 `error` 表示连接关闭或网络错误
- `defer conn.Close()` 确保连接在函数返回时关闭，避免资源泄漏

## 25.2 粘包问题与换行分隔协议

TCP 是**字节流协议**，没有消息边界。发送方连续发送多条消息，接收方可能一次读到多条（粘包），或一条消息被分成多次读（拆包）。

**错误示例（没有分隔符）**：

```go
// 客户端连续发送
conn.Write([]byte("消息1"))
conn.Write([]byte("消息2"))

// 服务器可能一次读到 "消息1消息2"，无法区分边界
buf := make([]byte, 1024)
n, _ := conn.Read(buf)
fmt.Println(string(buf[:n])) // 可能输出："消息1消息2"（粘包）
```

**解决方案 1：换行分隔协议**

每条消息以 `\n` 结尾，接收方用 `bufio.Scanner` 逐行读取。

```go
// 服务器：逐行读取
scanner := bufio.NewScanner(conn)
for scanner.Scan() {
    line := scanner.Text() // 不包含 \n
    fmt.Println("收到：", line)
    fmt.Fprintf(conn, "ECHO: %s\n", line)
}
```

```go
// 客户端：每条消息加 \n
fmt.Fprintf(conn, "第一行\n")
fmt.Fprintf(conn, "第二行\n")

// 逐行读取响应
scanner := bufio.NewScanner(conn)
for scanner.Scan() {
    fmt.Println(scanner.Text())
}
```

**输出示例**：

```
换行分隔协议服务器监听在 127.0.0.1:54322
  服务器收到：第一行
  服务器收到：第二行
  服务器收到：第三行
客户端收到：ECHO: 第一行
客户端收到：ECHO: 第二行
客户端收到：ECHO: 第三行
```

**注意事项**：

- `Scanner` 默认按行分割，也可以用 `SplitFunc` 自定义分隔规则
- 默认行长限制 64KB，超过会报 `bufio.ErrTooLong`，用 `scanner.Buffer()` 调整
- 适合文本协议（HTTP、SMTP、Redis RESP）

## 25.3 长度前缀协议

换行分隔不适合二进制数据（数据中可能包含 `\n`）。**长度前缀协议**是更通用的解决方案：每条消息前加固定长度的字段表示消息长度。

**协议格式**：`[4 字节长度（大端序）][N 字节数据]`

```go
// 发送消息
func sendMessage(conn net.Conn, data []byte) error {
    length := uint32(len(data))
    // 写入长度（4 字节大端序）
    if err := binary.Write(conn, binary.BigEndian, length); err != nil {
        return err
    }
    // 写入数据
    _, err := conn.Write(data)
    return err
}

// 接收消息
func receiveMessage(conn net.Conn) ([]byte, error) {
    // 读取长度
    var length uint32
    if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
        return nil, err
    }
    // 读取指定长度的数据
    data := make([]byte, length)
    _, err := io.ReadFull(conn, data) // ReadFull 保证读满
    return data, err
}
```

**服务器循环处理**：

```go
for {
    data, err := receiveMessage(conn)
    if err != nil {
        if err == io.EOF {
            break // 客户端关闭
        }
        log.Printf("读取失败：%v", err)
        return
    }
    fmt.Printf("收到消息（长度 %d）：%s\n", len(data), data)
    
    // 回显
    sendMessage(conn, data)
}
```

**输出示例**：

```
长度前缀协议服务器监听在 127.0.0.1:54323
  服务器收到（长度 9）：短消息
客户端收到（长度 9）：短消息
  服务器收到（长度 39）：这是一条稍微长一点的消息
客户端收到（长度 39）：这是一条稍微长一点的消息
```

**关键点**：

- `io.ReadFull` 确保读满指定字节数，避免部分读取
- 大端序（Big Endian）是网络字节序标准，跨平台兼容
- 适合二进制协议（Protobuf、Thrift、自定义 RPC）

## 25.4 Deadline 超时控制

网络操作可能无限期阻塞（对端不响应、网络故障）。`Deadline` 设置超时时间，到期后 `Read` / `Write` 返回超时错误。

**设置超时**：

```go
// 设置读超时：从现在起 5 秒内必须读到数据
conn.SetReadDeadline(time.Now().Add(5 * time.Second))

// 设置写超时
conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

// 设置读写超时（同时设置）
conn.SetDeadline(time.Now().Add(5 * time.Second))
```

**检测超时错误**：

```go
buf := make([]byte, 1024)
n, err := conn.Read(buf)
if err != nil {
    if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
        fmt.Println("读取超时")
    } else {
        fmt.Printf("读取失败：%v\n", err)
    }
}
```

**示例：客户端设置读超时**

```go
// 服务器延迟 200ms 才响应
go func() {
    conn, _ := listener.Accept()
    defer conn.Close()
    
    buf := make([]byte, 1024)
    n, _ := conn.Read(buf)
    
    time.Sleep(200 * time.Millisecond) // 模拟慢响应
    conn.Write(buf[:n])
}()

// 客户端：100ms 读超时
conn, _ := net.Dial("tcp", "127.0.0.1:8080")
defer conn.Close()

conn.Write([]byte("超时测试"))
conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

_, err := conn.Read(buf)
if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
    fmt.Println("读取超时（预期行为）") // 输出这一行
}
```

**输出示例**：

```
Deadline 演示服务器监听在 127.0.0.1:54324
读取超时（预期行为）
```

**关键点**：

- `Deadline` 是**绝对时间**（`time.Time`），不是相对时间
- 每次 `Read` / `Write` 前都要重新设置，否则第二次操作会立即超时
- `SetDeadline(time.Time{})` 清除超时（零值表示无超时）
- 超时后连接仍可用，可以继续读写（除非底层连接已断开）

## 25.5 UDP 通信

UDP 是**无连接、不可靠**的数据报协议，没有三次握手，不保证顺序和到达。

**UDP 服务器**：

```go
// 监听 UDP 端口
addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:8080")
conn, _ := net.ListenUDP("udp", addr)
defer conn.Close()

// 接收数据（阻塞）
buf := make([]byte, 1024)
n, clientAddr, err := conn.ReadFromUDP(buf)
fmt.Printf("收到来自 %s 的数据：%s\n", clientAddr, buf[:n])

// 回复客户端
conn.WriteToUDP(buf[:n], clientAddr)
```

**UDP 客户端**：

```go
// 连接服务器（实际不建立连接，只是关联地址）
serverAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:8080")
conn, _ := net.DialUDP("udp", nil, serverAddr)
defer conn.Close()

// 发送数据
conn.Write([]byte("Hello, UDP!"))

// 接收响应
buf := make([]byte, 1024)
n, _ := conn.Read(buf)
fmt.Println(string(buf[:n]))
```

**输出示例**：

```
UDP 服务器监听在 127.0.0.1:54325
  服务器收到来自 127.0.0.1:54326 的 UDP 数据：Hello, UDP!
客户端收到 UDP 响应：Hello, UDP!
```

**TCP vs UDP**：

| 特性 | TCP | UDP |
| --- | --- | --- |
| 连接 | 面向连接（三次握手） | 无连接 |
| 可靠性 | 可靠（重传、确认） | 不可靠（可能丢包、乱序） |
| 顺序 | 保证顺序 | 不保证顺序 |
| 速度 | 较慢（握手 + 确认） | 快（无握手） |
| 用途 | HTTP、SSH、数据库 | DNS、视频流、游戏 |

**UDP 使用场景**：

- 实时性要求高：视频直播、VoIP、在线游戏（丢包可接受）
- 请求-响应简单：DNS 查询（单包请求 + 单包响应）
- 广播/多播：局域网设备发现

## 25.6 DNS 解析与网络地址

**DNS 解析**：

```go
// 解析域名到 IP
ips, err := net.LookupIP("www.google.com")
for _, ip := range ips {
    fmt.Println(ip) // 输出多个 IP（IPv4 + IPv6）
}

// 解析主机名
host, err := net.LookupHost("localhost")
// 输出：["127.0.0.1", "::1"]

// 反向解析：IP 到主机名
names, err := net.LookupAddr("8.8.8.8")
fmt.Println(names) // 输出：["dns.google"]
```

**网络地址解析**：

```go
// 解析 TCP 地址
addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
fmt.Println(addr.IP)      // 127.0.0.1
fmt.Println(addr.Port)    // 8080
fmt.Println(addr.Network()) // "tcp"

// 解析 UDP 地址
udpAddr, _ := net.ResolveUDPAddr("udp", "8.8.8.8:53")
```

**本机网络接口**：

```go
// 获取所有网络接口
interfaces, _ := net.Interfaces()
for _, iface := range interfaces {
    fmt.Printf("%s: MTU %d\n", iface.Name, iface.MTU)
    
    // 获取接口的地址
    addrs, _ := iface.Addrs()
    for _, addr := range addrs {
        fmt.Println("  ", addr.String())
    }
}
```

**输出示例**：

```
localhost 解析结果：[127.0.0.1 ::1]
TCP 地址：127.0.0.1:8080（网络 tcp，IP 127.0.0.1，端口 8080）
本机网络接口数量：3
  第一个接口：lo（索引 1，MTU 65536）
```

## 25.7 连接关闭与半关闭

**正常关闭**：

```go
conn.Close() // 关闭读写，发送 FIN
```

**半关闭（TCP 专有）**：

只关闭写入，但仍可读取对端数据。

```go
if tcpConn, ok := conn.(*net.TCPConn); ok {
    tcpConn.CloseWrite() // 关闭写，发送 FIN，但仍可读
    tcpConn.CloseRead()  // 关闭读（不常用）
}
```

**应用场景**：

客户端发送完所有数据后 `CloseWrite()`，服务器收到 EOF 后处理完数据，再发送响应，客户端读取响应。

```go
// 客户端
conn.Write([]byte("请求数据"))
tcpConn.CloseWrite() // 告诉服务器：我不再发送了

// 服务器收到 EOF，知道请求结束
data, _ := io.ReadAll(conn) // 读到 EOF
fmt.Println("收到完整请求：", string(data))

// 服务器发送响应
conn.Write([]byte("响应数据"))
conn.Close()

// 客户端读取响应
response, _ := io.ReadAll(conn)
```

**输出示例**：

```
连接关闭演示服务器监听在 127.0.0.1:54327
客户端关闭写入（半关闭）
  服务器读取完毕，共 18 字节：数据1 数据2 数据3
客户端收到响应：数据1 数据2 数据3
```

**`io.EOF` 的含义**：

- 对端调用 `Close()` 或 `CloseWrite()`
- TCP 收到 FIN 包
- 不是错误，是正常的连接结束信号

## 25.8 并发服务器模式

**模式 1：每连接一个 goroutine**

这是 Go 网络编程的标准模式，简单高效。

```go
listener, _ := net.Listen("tcp", ":8080")
for {
    conn, err := listener.Accept()
    if err != nil {
        log.Println("Accept 失败：", err)
        continue
    }
    
    // 每个连接启动一个 goroutine
    go handleConnection(conn)
}

func handleConnection(conn net.Conn) {
    defer conn.Close()
    
    // 处理请求
    buf := make([]byte, 1024)
    n, err := conn.Read(buf)
    if err != nil {
        return
    }
    
    // 业务逻辑
    response := processRequest(buf[:n])
    conn.Write(response)
}
```

**输出示例**：

```
并发服务器监听在 127.0.0.1:54328
客户端 1 收到：客户端 1 的消息
客户端 2 收到：客户端 2 的消息
客户端 3 收到：客户端 3 的消息
```

**优点**：

- 简单：每个连接的处理逻辑独立，不需要手动管理状态
- 高效：goroutine 比线程轻量 1000 倍，支持百万并发连接
- 自然阻塞：`Read` / `Write` 阻塞当前 goroutine，不影响其他连接

**注意事项**：

- `defer conn.Close()` 确保连接关闭，避免 goroutine 泄漏
- 处理 panic：用 `defer recover()` 防止单个连接 panic 导致整个服务崩溃
- 限制并发数：如果连接数过多，用 channel 或 `semaphore` 限流

**模式 2：Worker Pool**

限制 goroutine 数量，适合 CPU 密集型任务。

```go
type Job struct {
    conn net.Conn
}

func workerPool(jobChan chan Job, workerCount int) {
    for i := 0; i < workerCount; i++ {
        go func() {
            for job := range jobChan {
                handleConnection(job.conn)
            }
        }()
    }
}

// 主循环
jobChan := make(chan Job, 100)
workerPool(jobChan, 10) // 10 个 worker

listener, _ := net.Listen("tcp", ":8080")
for {
    conn, _ := listener.Accept()
    jobChan <- Job{conn: conn}
}
```

## 真实报错案例

### 1. 地址已被占用

**错误代码**：

```go
listener, err := net.Listen("tcp", ":8080")
// 运行两次程序
```

**报错**：

```
listen tcp :8080: bind: address already in use
```

**原因**：端口已被占用（上次程序未关闭、或其他程序使用）。

**修复**：

- 检查端口占用：`lsof -i :8080`（macOS/Linux）或 `netstat -ano | findstr 8080`（Windows）
- 使用 `:0` 让系统自动分配端口
- `SO_REUSEADDR`（Go 默认开启）允许重启后立即绑定端口

### 2. 连接被重置

**错误代码**：

```go
conn.Write([]byte("数据"))
time.Sleep(1 * time.Second)
conn.Write([]byte("更多数据")) // 如果对端已关闭
```

**报错**：

```
write tcp 127.0.0.1:54321->127.0.0.1:8080: write: connection reset by peer
```

**原因**：对端已关闭连接，本地还在写入。

**修复**：

- 检查 `Write` 返回的 `error`
- 用心跳检测连接是否存活
- 设置 `SetKeepAlive(true)` 让 TCP 自动检测死连接

### 3. 读取超时但没检测

**错误代码**：

```go
conn.SetReadDeadline(time.Now().Add(1 * time.Second))
buf := make([]byte, 1024)
n, err := conn.Read(buf)
// 没有检测 err，直接使用 buf[:n]
fmt.Println(string(buf[:n])) // 可能 panic 或读到垃圾数据
```

**报错**：

```
read tcp 127.0.0.1:54321->127.0.0.1:8080: i/o timeout
```

**修复**：

```go
n, err := conn.Read(buf)
if err != nil {
    if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
        // 处理超时
    }
    return
}
// 使用 buf[:n]
```

### 4. 忘记 io.ReadFull 导致部分读取

**错误代码**：

```go
// 期望读取 100 字节
buf := make([]byte, 100)
n, _ := conn.Read(buf)
// n 可能小于 100（TCP 分包）
```

**修复**：

```go
buf := make([]byte, 100)
_, err := io.ReadFull(conn, buf) // 保证读满 100 字节
```

## 常见陷阱

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 客户端收到多条消息粘在一起 | TCP 字节流无消息边界 | 用换行分隔或长度前缀协议 |
| `Read` 一直阻塞 | 对端不发数据，也不关闭 | 设置 `SetReadDeadline` 超时 |
| 服务器连接数暴涨 | 每个连接一个 goroutine，没有限流 | 用 channel 或 worker pool 限制并发 |
| `Close` 后还能 `Read` | 对端还在发送数据 | 用 `CloseWrite` 半关闭，读取剩余数据 |
| UDP 数据丢包 | UDP 不可靠 | 应用层实现重传，或改用 TCP |
| `Dial` 超时很久 | 默认超时是操作系统决定（可能几分钟） | 用 `net.DialTimeout` 或 `context` |
| 端口释放后仍不能立即绑定 | TIME_WAIT 状态（TCP 四次挥手） | 等待 60 秒，或用 `SO_REUSEADDR`（Go 默认） |
| goroutine 泄漏 | 连接未关闭，goroutine 一直阻塞在 `Read` | `defer conn.Close()`，设置超时 |

## 练习

### 第 1 题

编写一个 TCP 服务器，支持多个客户端同时连接，每个客户端发送一个数字，服务器计算所有客户端发送数字的总和，并广播给所有客户端。

::: details 第 1 题参考答案

```go
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
)

var (
	mu      sync.Mutex
	sum     int
	clients []net.Conn
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	fmt.Println("求和服务器监听在 :8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Accept 失败：", err)
			continue
		}

		mu.Lock()
		clients = append(clients, conn)
		mu.Unlock()

		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		numStr := scanner.Text()
		num, err := strconv.Atoi(numStr)
		if err != nil {
			fmt.Fprintf(conn, "错误：%v\n", err)
			return
		}

		mu.Lock()
		sum += num
		currentSum := sum
		mu.Unlock()

		// 广播给所有客户端
		broadcast(fmt.Sprintf("当前总和：%d\n", currentSum))
	}
}

func broadcast(message string) {
	mu.Lock()
	defer mu.Unlock()

	for _, conn := range clients {
		fmt.Fprint(conn, message)
	}
}
```

**关键点**：用 `sync.Mutex` 保护共享变量 `sum` 和 `clients` 列表，避免数据竞争。广播时遍历所有连接。

:::

### 第 2 题

实现一个基于 UDP 的简单聊天室：服务器转发每个客户端的消息给其他所有客户端。

::: details 第 2 题参考答案

```go
package main

import (
	"fmt"
	"log"
	"net"
	"sync"
)

var (
	mu      sync.Mutex
	clients = make(map[string]*net.UDPAddr)
)

func main() {
	addr, _ := net.ResolveUDPAddr("udp", ":8080")
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println("UDP 聊天室监听在 :8080")

	buf := make([]byte, 1024)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println("读取失败：", err)
			continue
		}

		message := string(buf[:n])
		key := clientAddr.String()

		mu.Lock()
		if _, exists := clients[key]; !exists {
			clients[key] = clientAddr
			fmt.Printf("新客户端：%s\n", key)
		}
		mu.Unlock()

		// 广播给其他客户端
		broadcast(conn, message, clientAddr)
	}
}

func broadcast(conn *net.UDPConn, message string, sender *net.UDPAddr) {
	mu.Lock()
	defer mu.Unlock()

	senderKey := sender.String()
	for key, addr := range clients {
		if key != senderKey {
			conn.WriteToUDP([]byte(message), addr)
		}
	}
}
```

**关键点**：UDP 无连接，通过 `map` 记录所有客户端地址。广播时排除发送者本身。

:::

### 第 3 题

为长度前缀协议添加超时：如果客户端 5 秒内没有发送完整消息，服务器主动断开连接。

::: details 第 3 题参考答案

```go
func receiveMessageWithTimeout(conn net.Conn, timeout time.Duration) ([]byte, error) {
	// 设置读超时
	conn.SetReadDeadline(time.Now().Add(timeout))
	defer conn.SetReadDeadline(time.Time{}) // 清除超时

	// 读取长度
	var length uint32
	if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	// 读取数据（仍在超时范围内）
	conn.SetReadDeadline(time.Now().Add(timeout))
	data := make([]byte, length)
	_, err := io.ReadFull(conn, data)
	return data, err
}

// 使用示例
for {
	data, err := receiveMessageWithTimeout(conn, 5*time.Second)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Println("客户端超时，断开连接")
		}
		return
	}
	// 处理消息
	fmt.Println("收到：", string(data))
}
```

**关键点**：每次 `Read` 前重新设置 `Deadline`，读取完成后清除超时（`time.Time{}` 零值）。

:::

### 第 4 题

编写一个 TCP 服务器，支持优雅关闭：收到 `SIGINT`（Ctrl+C）后，停止接受新连接，等待现有连接处理完毕，再退出。

::: details 第 4 题参考答案

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("服务器监听在 :8080（Ctrl+C 优雅关闭）")

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	// 信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n收到信号，开始优雅关闭...")
		cancel()        // 通知所有 goroutine 退出
		listener.Close() // 停止接受新连接
	}()

	// Accept 循环
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if ctx.Err() != nil {
					return // 优雅关闭中
				}
				log.Println("Accept 失败：", err)
				continue
			}

			wg.Add(1)
			go handleConnection(ctx, conn, &wg)
		}
	}()

	<-ctx.Done()
	fmt.Println("等待现有连接处理完毕...")
	wg.Wait()
	fmt.Println("所有连接已关闭，服务器退出")
}

func handleConnection(ctx context.Context, conn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	defer conn.Close()

	// 模拟处理
	select {
	case <-ctx.Done():
		fmt.Println("连接被取消")
		return
	case <-time.After(2 * time.Second):
		conn.Write([]byte("处理完成\n"))
	}
}
```

**关键点**：用 `context.Context` 通知所有 goroutine 退出，`WaitGroup` 等待所有连接处理完毕。`listener.Close()` 让 `Accept()` 返回错误。

:::

### 第 5 题

实现一个 TCP 客户端连接池：维护 N 个连接，支持 `Get()` 获取连接、`Put()` 归还连接。如果池空了，`Get()` 阻塞等待。

::: details 第 5 题参考答案

```go
package main

import (
	"errors"
	"net"
	"sync"
)

type ConnPool struct {
	address string
	pool    chan net.Conn
	mu      sync.Mutex
	closed  bool
}

func NewConnPool(address string, size int) (*ConnPool, error) {
	p := &ConnPool{
		address: address,
		pool:    make(chan net.Conn, size),
	}

	// 预创建连接
	for i := 0; i < size; i++ {
		conn, err := net.Dial("tcp", address)
		if err != nil {
			p.Close()
			return nil, err
		}
		p.pool <- conn
	}

	return p, nil
}

func (p *ConnPool) Get() (net.Conn, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, errors.New("连接池已关闭")
	}
	p.mu.Unlock()

	conn := <-p.pool // 阻塞等待可用连接
	return conn, nil
}

func (p *ConnPool) Put(conn net.Conn) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		conn.Close()
		return errors.New("连接池已关闭")
	}
	p.mu.Unlock()

	p.pool <- conn
	return nil
}

func (p *ConnPool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	close(p.pool)
	for conn := range p.pool {
		conn.Close()
	}
}

// 使用示例
func main() {
	pool, _ := NewConnPool("127.0.0.1:8080", 5)
	defer pool.Close()

	conn, _ := pool.Get()
	conn.Write([]byte("请求\n"))
	pool.Put(conn)
}
```

**关键点**：用带缓冲的 channel 作为连接池，`Get()` 从 channel 读（阻塞），`Put()` 写回 channel。关闭时用 `mu` 保护 `closed` 标志。

:::

### 第 6 题

实现一个简单的 HTTP 服务器，不使用 `net/http` 包，手动解析 HTTP 请求行和响应。

::: details 第 6 题参考答案

```go
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	fmt.Println("HTTP 服务器监听在 :8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Accept 失败：", err)
			continue
		}
		go handleHTTP(conn)
	}
}

func handleHTTP(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// 读取请求行：GET /path HTTP/1.1
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	requestLine = strings.TrimSpace(requestLine)
	parts := strings.Split(requestLine, " ")
	if len(parts) < 3 {
		return
	}

	method := parts[0]
	path := parts[1]

	// 读取请求头（直到空行）
	for {
		line, _ := reader.ReadString('\n')
		if line == "\r\n" || line == "\n" {
			break
		}
	}

	// 构造响应
	body := fmt.Sprintf("<h1>Hello, HTTP!</h1><p>Method: %s</p><p>Path: %s</p>", method, path)
	response := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\n"+
			"Content-Type: text/html; charset=utf-8\r\n"+
			"Content-Length: %d\r\n"+
			"\r\n"+
			"%s",
		len(body), body,
	)

	conn.Write([]byte(response))
}
```

**测试**：在浏览器访问 `http://localhost:8080/test`，看到 "Hello, HTTP!"。

**关键点**：HTTP 协议是文本协议，请求行 + 请求头 + 空行 + 请求体。响应格式类似。这让我们理解 `net/http` 包的底层实现。

:::

## 小结

- **net 包基础**：`Listen` 监听端口，`Accept` 接受连接，`Dial` 连接服务器，`Conn` 实现 `io.Reader` 和 `io.Writer`
- **粘包问题**：TCP 字节流无消息边界，用换行分隔（`bufio.Scanner`）或长度前缀（`binary.Read` + `io.ReadFull`）
- **超时控制**：`SetReadDeadline` / `SetWriteDeadline` 设置绝对超时时间，`net.Error.Timeout()` 检测超时
- **UDP 通信**：无连接、不可靠，用 `ListenUDP` / `ReadFromUDP` / `WriteToUDP`，适合实时性要求高的场景
- **DNS 解析**：`LookupIP` / `LookupHost` / `LookupAddr` 解析域名和 IP，`ResolveTCPAddr` / `ResolveUDPAddr` 解析地址
- **连接关闭**：`Close()` 关闭读写，`CloseWrite()` 半关闭（只关写），`io.EOF` 表示对端关闭
- **并发服务器**：每连接一个 goroutine 是标准模式，简单高效，支持百万并发；用 `defer conn.Close()` 防止泄漏
- **常见陷阱**：忘记设置超时导致阻塞、`Read` 部分读取、goroutine 泄漏、端口占用、连接被重置未检测

下一章我们将学习 HTTP 服务端开发：`net/http` 包、路由增强、中间件链、JSON 响应、超时控制、优雅关闭，以及如何用 `httptest` 测试 HTTP 服务。
