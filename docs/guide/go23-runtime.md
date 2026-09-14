# 第 23 章 · 运行时、调度与内存模型

前几章我们学习了并发基础、并发模式和 context 生命周期管理，掌握了如何在应用层使用 goroutine 和 channel。但要写出高性能、低延迟的 Go 程序，还需要理解 Go 运行时的调度机制、垃圾回收策略和内存模型保证。本章深入 Go 运行时的核心机制：GMP 调度模型如何在少量线程上调度海量 goroutine、GC 如何在不停服的前提下回收内存、编译器如何决定变量分配在栈还是堆、内存模型如何定义并发读写的可见性，以及如何通过 `runtime` 包和构建标签调试和优化程序。

本章配套代码在 `internal/chapter/go23_runtime/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

## 23.1 GMP 调度模型与 GOMAXPROCS

Go 的调度器使用 **GMP 模型**来管理 goroutine 的执行：

- **G (goroutine)**：用户态轻量级线程，栈初始只有 2KB，按需增长到 1GB
- **M (machine)**：操作系统线程，真正执行代码的载体
- **P (processor)**：调度上下文，持有本地 goroutine 队列，数量由 `GOMAXPROCS` 决定

**核心机制**：

1. 每个 P 绑定一个 M，从本地队列取 G 执行
2. P 数量默认等于 CPU 核心数，通过 `runtime.GOMAXPROCS()` 调整
3. 当 G 阻塞（syscall、channel）时，M 会解绑 P，P 可以绑定其他 M 继续调度
4. 工作窃取：空闲的 P 会从其他 P 的本地队列偷取一半 G

```go
// 查看当前 GOMAXPROCS
maxProcs := runtime.GOMAXPROCS(0) // 传 0 不修改，只查询
fmt.Printf("GOMAXPROCS = %d\n", maxProcs)

// 查看当前 goroutine 数量
fmt.Printf("当前 goroutine 数量：%d\n", runtime.NumGoroutine())

// 启动一些 goroutine
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        sum := 0
        for j := 0; j < 1000; j++ {
            sum += j
        }
    }(i)
}
wg.Wait()
```

**输出示例**：

```
GOMAXPROCS = 8（默认等于 CPU 核心数）
当前 goroutine 数量：1
启动 10 个 goroutine 后：11
等待完成后：1
```

**调优建议**：

- CPU 密集型任务：`GOMAXPROCS` 设为 CPU 核心数（默认值）
- IO 密集型任务：可以略高于核心数（如 1.5-2 倍），让阻塞时有更多 P 继续调度
- 容器环境：注意容器的 CPU 配额，Go 1.5+ 会自动识别 cgroup 限制

## 23.2 goroutine 栈增长与调度信息

Go 1.3+ 使用 **连续栈（contiguous stack）** 策略：

- 初始栈：2KB（相比 OS 线程的 MB 级栈非常轻量）
- 按需增长：函数调用时检测栈空间，不足时分配更大的栈（2 倍），复制旧数据，更新指针
- 栈收缩：GC 时如果栈使用率低于 1/4，收缩到 1/2 大小
- 最大栈：1GB（64 位系统）

```go
var memBefore runtime.MemStats
runtime.ReadMemStats(&memBefore)

deepRecursion(0, 100) // 递归 100 层

var memAfter runtime.MemStats
runtime.ReadMemStats(&memAfter)

fmt.Printf("栈内存增长：%d bytes\n", memAfter.StackInuse - memBefore.StackInuse)
```

**栈增长的代价**：

- 每次增长需要复制整个栈，更新所有指针（编译器插入栈检查代码）
- 深度递归或在栈上分配大数组会频繁触发增长
- 优化手段：预分配足够栈空间的 goroutine 池，避免热路径栈增长

**查看调度信息**：

```bash
GODEBUG=schedtrace=1000 go run main.go  # 每秒打印调度统计
```

输出示例：

```
SCHED 1000ms: gomaxprocs=8 idleprocs=6 threads=10 spinningthreads=0 idlethreads=4 runqueue=0 [0 0 0 0 0 0 0 0]
```

- `gomaxprocs=8`：P 的数量
- `idleprocs=6`：空闲的 P
- `threads=10`：M 的数量（包括空闲和执行中）
- `runqueue=0`：全局队列长度
- `[0 0 0 0 0 0 0 0]`：每个 P 的本地队列长度

## 23.3 GC 基础与三色标记

Go 使用 **并发三色标记清除（concurrent tricolor mark-and-sweep）** 算法：

**三色抽象**：

- **白色**：未扫描的对象（初始状态，扫描结束后仍为白色的会被回收）
- **灰色**：已扫描但其引用的对象未全部扫描（工作队列）
- **黑色**：已扫描且其引用的对象已全部扫描（确认存活）

**GC 流程**：

1. **STW（Stop-The-World）标记准备**：暂停所有 goroutine，启用写屏障，根对象标记为灰色（耗时 ~10-30μs）
2. **并发标记**：与用户代码并发执行，遍历灰色对象，标记其引用为灰色，自己变黑色
3. **STW 标记终止**：再次短暂暂停，处理标记队列剩余对象（耗时 ~10-30μs）
4. **并发清除**：回收白色对象

**写屏障（write barrier）**：

- 标记阶段修改指针时插入的代码，确保新指向的对象被标记（防止漏标）
- 有轻微性能开销（~5-10%），但换来了低暂停时间（<1ms）

```go
// 查看 GC 统计
var stats debug.GCStats
debug.ReadGCStats(&stats)
fmt.Printf("GC 次数：%d\n", stats.NumGC)
if len(stats.Pause) > 0 {
    fmt.Printf("最近一次 GC 暂停：%v\n", stats.Pause[0])
}

// 手动触发 GC
runtime.GC()
```

**输出示例**：

```
GC 次数：3
最近一次 GC 暂停：45.2µs
GC 次数（触发后）：4
```

## 23.4 GC 参数：GOGC 与 GOMEMLIMIT

**GOGC**：目标堆增长百分比（默认 100）

- `GOGC=100`：堆从 100MB 增长到 200MB 时触发 GC
- `GOGC=200`：堆增长 200% 时触发（更少 GC，更高内存）
- `GOGC=50`：堆增长 50% 时触发（更频繁 GC，更低内存）
- `GOGC=off`：禁用自动 GC（只能手动 `runtime.GC()`）

```go
oldGOGC := debug.SetGCPercent(200) // 代码中设置
fmt.Printf("GOGC 从 %d 改为 200\n", oldGOGC)
debug.SetGCPercent(oldGOGC) // 恢复
```

**GOMEMLIMIT**（Go 1.19+）：软内存限制

- 环境变量：`GOMEMLIMIT=2GiB`
- 代码设置：`debug.SetMemoryLimit(2 * 1024 * 1024 * 1024)`
- 作用：GC 会尽量保持堆内存不超过此值，超过后更频繁 GC
- 注意：这是**软限制**，不是硬限制（极端情况仍可能超过）

```go
oldLimit := debug.SetMemoryLimit(-1) // 传 -1 只查询
fmt.Printf("当前内存限制：%d bytes\n", oldLimit) // -1 表示无限制
```

**调优场景**：

| 场景 | GOGC | GOMEMLIMIT | 效果 |
| --- | --- | --- | --- |
| 高吞吐批处理 | 200-300 | 不设置 | 减少 GC 频率，用内存换吞吐 |
| 低延迟服务 | 50-100 | 容器内存 80% | 更频繁但更短的 GC，避免长尾延迟 |
| 容器环境 | 100 | 容器限制 90% | 防止 OOM，让 GC 提前介入 |
| 内存受限设备 | 50 | 设备内存 70% | 激进回收，牺牲吞吐保稳定 |

**查看 GC 详情**：

```bash
GODEBUG=gctrace=1 go run main.go
```

输出示例：

```
gc 1 @0.002s 3%: 0.018+0.45+0.003 ms clock, 0.14+0.23/0.40/0.63+0.029 ms cpu, 4->4->1 MB, 5 MB goal, 8 P
```

- `gc 1`：第 1 次 GC
- `0.002s`：程序启动后 2ms
- `3%`：GC 占用 CPU 时间百分比
- `0.018+0.45+0.003 ms`：STW 准备 + 并发标记 + STW 终止
- `4->4->1 MB`：GC 前堆大小 → GC 后堆大小 → 存活对象大小
- `5 MB goal`：下次 GC 的目标堆大小

## 23.5 内存分配与逃逸分析

**逃逸分析（escape analysis）**：编译器决定变量分配在栈还是堆

- **栈分配**：函数返回后自动释放，无 GC 压力，速度快
- **堆分配**：需要 GC 回收，速度慢，但生命周期可超出函数

**常见逃逸场景**：

1. **返回局部变量的指针**

```go
func makePointer() *int {
    x := 42
    return &x  // x 逃逸到堆
}
```

2. **传递给 interface{}**

```go
var iface interface{} = 100  // 100 逃逸到堆
fmt.Println(x)  // x 逃逸（fmt.Println 接受 interface{}）
```

3. **闭包捕获的变量在闭包返回后仍被访问**

```go
func makeCounter() func() int {
    count := 0
    return func() int {
        count++  // count 逃逸到堆
        return count
    }
}
```

4. **切片/map 超出容量**

```go
s := make([]int, 10, 100)  // 可能在栈上
s = append(s, 1)           // 如果超出容量，逃逸到堆
```

**查看逃逸分析**：

```bash
go build -gcflags="-m" ./internal/chapter/go23_runtime/
```

输出示例：

```
./demo.go:196:2: moved to heap: x
./demo.go:202:2: moved to heap: count
./demo.go:187:12: ... argument does not escape
./demo.go:187:13: iface escapes to heap
```

**优化建议**：

- 热路径避免返回指针，考虑值拷贝（小对象 <128 字节）
- 预分配容量，避免 append 触发逃逸
- 减少 `interface{}` 使用，泛型可以减少装箱
- 使用 `sync.Pool` 复用堆对象

## 23.6 内存模型与 happens-before

Go 内存模型定义了多个 goroutine 读写共享变量时的 **可见性保证**。核心概念是 **happens-before**：如果事件 A happens-before 事件 B，则 A 的效果对 B 可见。

**错误示例（数据竞争）**：

```go
var x int
go func() { x = 1 }()
fmt.Println(x)  // 可能看到 0 或 1，未定义行为！
```

用 `-race` 检测：

```bash
go run -race ./cmd/go-learn
```

输出：

```
==================
WARNING: DATA RACE
Write at 0x00c000012090 by goroutine 7:
  ...
Previous read at 0x00c000012090 by main goroutine:
  ...
==================
```

**正确的同步机制**：

### 1. Channel

```go
var x int
ch := make(chan struct{})
go func() {
    x = 1
    ch <- struct{}{}  // 发送 happens-before 接收
}()
<-ch
fmt.Println(x)  // 保证看到 1
```

### 2. sync.Mutex

```go
var mu sync.Mutex
var y int
mu.Lock()
y = 2
mu.Unlock()  // 解锁 happens-before 下一次加锁

mu.Lock()
fmt.Println(y)  // 保证看到 2
mu.Unlock()
```

### 3. sync.WaitGroup

```go
var z int
var wg sync.WaitGroup
wg.Add(1)
go func() {
    z = 3
    wg.Done()  // Done happens-before Wait 返回
}()
wg.Wait()
fmt.Println(z)  // 保证看到 3
```

### 4. sync.Once

```go
var once sync.Once
var config string
once.Do(func() {
    config = "loaded"  // Do 内的写入 happens-before Do 返回
})
fmt.Println(config)  // 保证看到 "loaded"
```

**错误的同步（没有 happens-before 保证）**：

```go
var done bool
go func() {
    doWork()
    done = true  // 可能被编译器或 CPU 重排序
}()
for !done {}  // 可能永远循环，编译器可能优化为 if !done { for {} }
```

正确做法：用 `atomic.Bool` 或 channel。

## 23.7 sync/atomic 的语义

`sync/atomic` 提供原子操作，同时提供 **happens-before 保证**：

- `Load`：acquire 语义（之后的读写不会被重排到它之前）
- `Store`：release 语义（之前的读写不会被重排到它之后）
- `Add`/`Swap`/`CompareAndSwap`：acquire + release 语义

**原子计数器**：

```go
var counter atomic.Int64
var wg sync.WaitGroup

for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        counter.Add(1)  // 原子自增
    }()
}
wg.Wait()
fmt.Printf("counter = %d\n", counter.Load())  // 输出：100
```

**通过 atomic 同步数据**：

```go
var flag atomic.Bool
var data int

// goroutine 1：写入数据后设置标志
go func() {
    data = 42
    flag.Store(true)  // release：确保之前的写入对后续 Load 可见
}()

// goroutine 2：等待标志后读取数据
for !flag.Load() {  // acquire：确保看到 Store 之前的写入
    runtime.Gosched()
}
fmt.Printf("data = %d\n", data)  // 保证看到 42
```

**atomic vs Mutex**：

| 场景 | 推荐 | 原因 |
| --- | --- | --- |
| 单个变量的简单读写 | atomic | 性能更好，无锁 |
| 多个变量的复合操作 | Mutex | atomic 无法保证多个变量的原子性 |
| 读多写少 | atomic（配合 `RWMutex`） | Load 无竞争 |
| 需要条件变量 | Mutex + `sync.Cond` | atomic 无法等待 |

## 23.8 runtime 包常用函数

### 系统信息

```go
runtime.NumCPU()        // 逻辑 CPU 核心数
runtime.NumGoroutine()  // 当前 goroutine 数量
runtime.GOMAXPROCS(0)   // 查询 GOMAXPROCS
runtime.Version()       // Go 版本（如 "go1.26.3"）
runtime.Compiler        // 编译器名称（通常是 "gc"）
runtime.GOOS            // 目标操作系统（如 "linux", "windows"）
runtime.GOARCH          // 目标架构（如 "amd64", "arm64"）
```

### 调度控制

```go
runtime.Gosched()  // 主动让出 CPU 时间片，让其他 goroutine 执行
runtime.Goexit()   // 终止当前 goroutine（不影响其他 goroutine）
```

示例：

```go
go func() {
    for i := 0; i < 3; i++ {
        fmt.Printf("A: %d\n", i)
        runtime.Gosched()  // 让出，增加交替执行的机会
    }
}()
for i := 0; i < 3; i++ {
    fmt.Printf("main: %d\n", i)
    runtime.Gosched()
}
```

### 内存统计

```go
var m runtime.MemStats
runtime.ReadMemStats(&m)
fmt.Printf("堆分配：%d MB\n", m.Alloc/1024/1024)
fmt.Printf("系统内存：%d MB\n", m.Sys/1024/1024)
fmt.Printf("GC 次数：%d\n", m.NumGC)
fmt.Printf("GC 暂停总时间：%v\n", time.Duration(m.PauseTotalNs))
```

**主要字段**：

- `Alloc`：当前堆上分配的字节数
- `TotalAlloc`：累计分配的字节数（包括已释放的）
- `Sys`：从操作系统申请的总内存
- `NumGC`：GC 次数
- `PauseTotalNs`：GC 暂停总时间（纳秒）

### GC 控制

```go
runtime.GC()                      // 手动触发 GC
debug.FreeOSMemory()              // 强制归还内存给操作系统
debug.SetGCPercent(200)           // 设置 GOGC
debug.SetMemoryLimit(4 << 30)     // 设置 GOMEMLIMIT（4GB）
```

### 栈追踪

```go
buf := make([]byte, 1024)
n := runtime.Stack(buf, false)  // false: 只当前 goroutine
fmt.Printf("栈追踪：\n%s\n", buf[:n])
```

输出示例：

```
goroutine 1 [running]:
go-learn/internal/chapter/go23_runtime.demoRuntimeFuncs()
    /path/to/demo.go:332 +0x123
...
```

## 23.9 构建标签与 GODEBUG

### 构建标签（Build Tags）

条件编译，写在文件顶部：

```go
//go:build linux && amd64
// +build linux,amd64  // Go 1.17 之前的旧语法
```

**常用场景**：

1. **平台特定代码**

```go
// file_unix.go
//go:build unix

package mypackage

func openFile() { /* Unix 实现 */ }
```

```go
// file_windows.go
//go:build windows

package mypackage

func openFile() { /* Windows 实现 */ }
```

2. **测试/调试代码**

```go
//go:build debug

package mypackage

func trace(msg string) {
    log.Println(msg)
}
```

编译时指定：

```bash
go build -tags debug ./cmd/myapp
```

3. **可选功能**

```go
//go:build cgo

package mypackage

// #include <stdlib.h>
import "C"
```

**构建约束语法**：

- `&&`：与（必须同时满足）
- `||`：或（满足任一即可）
- `!`：非（不满足）
- 例如：`//go:build (linux || darwin) && !cgo`

### GODEBUG 环境变量

运行时调试参数，格式：`GODEBUG=key1=val1,key2=val2`

**常用参数**：

| 参数 | 作用 | 示例 |
| --- | --- | --- |
| `gctrace=1` | 打印 GC 信息 | `GODEBUG=gctrace=1 go run main.go` |
| `schedtrace=1000` | 每 1000ms 打印调度信息 | `GODEBUG=schedtrace=1000 go run main.go` |
| `madvdontneed=1` | 内存归还策略（Linux） | Go 1.12+ 默认为 0（MADV_FREE） |
| `http2debug=1` | HTTP/2 调试日志 | `GODEBUG=http2debug=1 go run server.go` |
| `invalidptr=0` | 禁用无效指针检查 | 调试 cgo/unsafe 代码 |

**示例**：

```bash
# 查看 GC 详情
GODEBUG=gctrace=1 go run ./cmd/go-learn

# 查看调度统计
GODEBUG=schedtrace=1000 go run ./cmd/go-learn

# 组合使用
GODEBUG=gctrace=1,schedtrace=500 go run ./cmd/go-learn
```

## 真实报错案例

### 1. 数据竞争检测

**错误代码**：

```go
var counter int
for i := 0; i < 10; i++ {
    go func() {
        counter++  // 数据竞争
    }()
}
```

**编译通过，但运行时 `-race` 报错**：

```bash
$ go run -race main.go
==================
WARNING: DATA RACE
Write at 0x00c000126010 by goroutine 7:
  main.main.func1()
      /path/to/main.go:10 +0x38

Previous write at 0x00c000126010 by goroutine 6:
  main.main.func1()
      /path/to/main.go:10 +0x38

Goroutine 7 (running) created at:
  main.main()
      /path/to/main.go:9 +0x78
==================
```

**修复**：用 `atomic.Int64` 或 `sync.Mutex`。

### 2. 栈溢出

**错误代码**：

```go
func recursive() {
    var buf [1024 * 1024]byte  // 1MB 栈上数组
    recursive()
}
```

**运行时报错**：

```
runtime: goroutine stack exceeds 1000000000-byte limit
runtime: sp=0xc0200e0390 stack=[0xc0200e0000, 0xc0400e0000]
fatal error: stack overflow
```

**修复**：减小栈上分配，或用堆分配（`make([]byte, 1024*1024)`）。

### 3. 逃逸分析误判

**代码**：

```go
func process() {
    data := make([]int, 1000000)  // 期望在栈上
    // ... 使用 data
}
```

**逃逸分析**：

```bash
$ go build -gcflags="-m" main.go
./main.go:5:14: make([]int, 1000000) escapes to heap
```

**原因**：切片过大（>64KB），编译器强制堆分配。

**修复**：接受逃逸，或用 `sync.Pool` 复用。

## 常见陷阱

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| goroutine 数量暴涨 | 创建 goroutine 但不回收（如无限 accept + go handle） | 用 worker pool 限制并发数，配合 `context` 取消 |
| GC 暂停过长 | 堆太大或存活对象太多 | 降低 GOGC，设置 GOMEMLIMIT，减少堆上分配 |
| 内存持续增长 | goroutine 泄漏或大对象未释放 | `pprof` 分析堆，检查 channel 阻塞和闭包捕获 |
| 数据竞争但不崩溃 | 并发读写无同步，在某些平台表现正常 | 始终用 `-race` 测试，不依赖运气 |
| `for !flag {}` 死循环 | 编译器优化或 CPU 缓存，flag 更新不可见 | 用 `atomic.Bool` 或 channel |
| 性能下降 10% | 开启写屏障（GC 标记阶段） | 正常代价，减少堆分配可降低 GC 频率 |
| 容器 OOM | Go 不识别容器内存限制（老版本） | 设置 GOMEMLIMIT，或升级到 Go 1.19+ |
| 栈溢出 | 深度递归或栈上大数组 | 改为迭代，或用堆分配 |
| `GOMAXPROCS=1` 性能差 | 只有 1 个 P，无法利用多核 | 删除手动设置，用默认值 |

## 练习

### 第 1 题

编写程序测量创建 10000 个 goroutine 的内存开销，对比 10000 个 OS 线程的开销（提示：用 `runtime.ReadMemStats` 和 `pprof`）。

::: details 第 1 题参考答案

```go
package main

import (
    "fmt"
    "runtime"
    "sync"
)

func main() {
    var m1 runtime.MemStats
    runtime.ReadMemStats(&m1)

    var wg sync.WaitGroup
    for i := 0; i < 10000; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            var buf [128]byte
            _ = buf
            // 阻塞等待
            select {}
        }()
    }

    var m2 runtime.MemStats
    runtime.ReadMemStats(&m2)

    fmt.Printf("10000 个 goroutine 内存开销：%d MB\n", (m2.Sys-m1.Sys)/1024/1024)
    fmt.Printf("每个 goroutine 约 %d KB\n", (m2.StackInuse-m1.StackInuse)/10000/1024)

    // OS 线程对比（不实际运行，会耗尽系统资源）：
    // 每个线程约 2-8 MB（取决于平台），10000 个约 20-80 GB
    // goroutine 每个约 2-4 KB，10000 个约 20-40 MB
}
```

**关键点**：goroutine 比线程轻量 1000 倍以上，这是 Go 能支持百万并发的基础。

:::

### 第 2 题

写一个函数，用 `atomic.Bool` 实现一次性初始化（类似 `sync.Once`），要求线程安全且高性能。

::: details 第 2 题参考答案

```go
type AtomicOnce struct {
    done atomic.Bool
}

func (o *AtomicOnce) Do(f func()) {
    if o.done.Load() {  // 快速路径：已初始化
        return
    }
    // 竞争路径：尝试 CAS
    if o.done.CompareAndSwap(false, true) {
        f()  // 只有一个 goroutine 能执行
    } else {
        // 其他 goroutine 等待（简化版，生产代码需要自旋或等待）
        for !o.done.Load() {
            runtime.Gosched()
        }
    }
}
```

**关键点**：`CompareAndSwap` 保证只有一个 goroutine 执行 `f()`，Load 的 acquire 语义保证其他 goroutine 看到 `f()` 的效果。生产环境建议直接用 `sync.Once`，它有更好的等待机制。

:::

### 第 3 题

用 `-gcflags="-m -m"` 分析下面代码的逃逸行为，解释为什么 `x` 逃逸而 `y` 不逃逸：

```go
func foo() *int {
    x := 42
    return &x
}

func bar() int {
    y := 42
    return y
}
```

::: details 第 3 题参考答案

```bash
$ go build -gcflags="-m -m" main.go
./main.go:4:2: x escapes to heap:
./main.go:4:2:   flow: ~r0 = &x:
./main.go:4:2:     from &x (address-of) at ./main.go:5:9
./main.go:4:2:     from return &x (return) at ./main.go:5:2
./main.go:4:2: moved to heap: x

./main.go:9:2: y does not escape
```

**解释**：

- `foo` 返回 `&x`，`x` 的生命周期必须超出函数，所以逃逸到堆
- `bar` 返回 `y` 的值拷贝，调用方拿到的是副本，`y` 可以留在栈上

**更深层次**：编译器做逃逸分析的规则是"如果指针可能被外部持有，就逃逸"。返回指针、传递给 `interface{}`、闭包捕获后逃出作用域，都属于这种情况。

:::

### 第 4 题

容器环境下，设置 `GOMEMLIMIT=1GiB` 和 `GOGC=100`，堆当前是 800MB，下次 GC 会在什么时候触发？如果改为 `GOGC=200`，又会在什么时候触发？

::: details 第 4 题参考答案

**场景 1：`GOMEMLIMIT=1GiB`，`GOGC=100`，当前堆 800MB**

- `GOGC=100` 的目标：堆增长到 800MB * 2 = 1600MB 时触发
- `GOMEMLIMIT=1GiB` 的目标：堆接近 1024MB 时触发
- **实际触发**：约 1024MB（以 GOMEMLIMIT 为准）

**场景 2：`GOMEMLIMIT=1GiB`，`GOGC=200`，当前堆 800MB**

- `GOGC=200` 的目标：堆增长到 800MB * 3 = 2400MB 时触发
- `GOMEMLIMIT=1GiB` 的目标：堆接近 1024MB 时触发
- **实际触发**：约 1024MB（仍然以 GOMEMLIMIT 为准）

**结论**：`GOMEMLIMIT` 是硬上限（软限制但优先级高），`GOGC` 只在未达到 `GOMEMLIMIT` 时生效。容器环境建议设置 `GOMEMLIMIT` 为容器内存的 80-90%，避免 OOM。

:::

### 第 5 题

为什么下面的代码在 `-race` 下报数据竞争，但用 `atomic.Bool` 就不报？

```go
// 版本 A：数据竞争
var ready bool
var data int

go func() {
    data = 42
    ready = true
}()

for !ready {}
fmt.Println(data)
```

```go
// 版本 B：无数据竞争
var ready atomic.Bool
var data int

go func() {
    data = 42
    ready.Store(true)
}()

for !ready.Load() {}
fmt.Println(data)
```

::: details 第 5 题参考答案

**版本 A 的问题**：

1. **数据竞争**：goroutine 写 `ready`，main 读 `ready`，无同步机制
2. **可见性问题**：`ready = true` 和 `data = 42` 可能被编译器或 CPU 重排序，main 可能看到 `ready == true` 但 `data == 0`
3. **死循环问题**：编译器可能把 `for !ready {}` 优化为 `if !ready { for {} }`（因为循环内没有对 `ready` 的写入）

**版本 B 的解决**：

1. `atomic.Bool` 提供原子性，无数据竞争
2. `Store(true)` 有 release 语义，确保之前的写入（`data = 42`）对后续 `Load()` 可见
3. `Load()` 有 acquire 语义，确保看到 `Store()` 之前的所有写入
4. `Load()` 的 volatile 语义阻止编译器优化死循环

**关键点**：`atomic` 不仅提供原子性，还提供内存顺序保证（happens-before），这是 `bool` 做不到的。

:::

### 第 6 题

用 `runtime.Stack` 编写一个 panic 恢复函数，打印当前 goroutine 的栈追踪到日志，然后继续运行其他 goroutine。

::: details 第 6 题参考答案

```go
func safeLaunch(name string, f func()) {
    go func() {
        defer func() {
            if err := recover(); err != nil {
                buf := make([]byte, 4096)
                n := runtime.Stack(buf, false)
                log.Printf("goroutine %s panic: %v\nStack:\n%s", name, err, buf[:n])
            }
        }()
        f()
    }()
}

// 使用示例
func main() {
    safeLaunch("worker-1", func() {
        panic("oops")
    })

    safeLaunch("worker-2", func() {
        fmt.Println("worker-2 正常运行")
    })

    time.Sleep(100 * time.Millisecond)
}
```

**输出示例**：

```
2026/09/14 10:30:15 goroutine worker-1 panic: oops
Stack:
goroutine 5 [running]:
main.safeLaunch.func1.1()
    /path/to/main.go:8 +0x85
panic({0x1004a60, 0x1008f50})
    /usr/local/go/src/runtime/panic.go:884 +0x212
main.safeLaunch.func1.2()
    /path/to/main.go:14 +0x27
...

worker-2 正常运行
```

**关键点**：每个 goroutine 独立的 defer + recover，一个 goroutine 的 panic 不影响其他 goroutine。这是微服务和 worker pool 的常见模式。

:::

## 小结

- **GMP 调度模型**：G（goroutine）、M（OS 线程）、P（调度上下文），GOMAXPROCS 控制 P 的数量，默认等于 CPU 核心数；工作窃取让空闲 P 从其他 P 偷取任务
- **栈增长**：goroutine 初始栈 2KB，按需增长到 1GB，使用连续栈策略（分配新栈 + 复制 + 更新指针）
- **GC 三色标记**：并发标记清除，STW 时间 <1ms；写屏障确保标记阶段的指针修改不漏标；`gctrace=1` 查看 GC 详情
- **GC 参数**：`GOGC` 控制堆增长比例（默认 100），`GOMEMLIMIT`（Go 1.19+）设置软内存限制，容器环境建议设置为容器内存的 80-90%
- **逃逸分析**：编译器决定栈还是堆分配；返回指针、传给 `interface{}`、闭包捕获后逃出作用域会导致逃逸；用 `-gcflags="-m"` 查看分析结果
- **内存模型**：happens-before 定义可见性保证；channel、Mutex、WaitGroup、Once 提供同步；普通变量的并发读写是未定义行为，必须用 `-race` 检测
- **sync/atomic**：提供原子操作和 acquire/release 语义，`Load`/`Store` 保证 happens-before；适合单变量同步，多变量需要 Mutex
- **runtime 包**：`NumCPU()`、`NumGoroutine()`、`Gosched()`、`ReadMemStats()`、`Stack()` 等常用函数；`GODEBUG` 环境变量调试 GC 和调度
- **构建标签**：`//go:build` 条件编译，用于平台特定代码、可选功能、测试代码；`GODEBUG` 运行时调试参数，如 `gctrace=1`、`schedtrace=1000`

下一章我们将学习性能分析与优化：如何用 pprof 定位 CPU 和内存热点、如何用 trace 分析 goroutine 调度、如何通过预分配和 sync.Pool 减少 GC 压力、如何用 benchstat 对比优化效果，以及内联、逃逸分析和锁竞争的优化实战。
