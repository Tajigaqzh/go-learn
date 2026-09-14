# 第 20 章 并发基础

Go 的并发模型建立在 CSP（Communicating Sequential Processes）理论之上，通过 goroutine 和 channel 提供了一套简洁而强大的并发原语。本章从 goroutine 的基本用法开始，逐步深入到 channel、select、sync 包和原子操作，配合实际示例与测试，帮你掌握 Go 并发编程的核心技能。

**本章配套代码**在 `internal/chapter/go20_concurrency/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

---

## 20.1 goroutine 基础

### 什么是 goroutine

**goroutine** 是 Go 运行时管理的轻量级线程：

- **初始栈**只有 2KB，按需增长（最大可达 1GB）
- **M:N 调度**：M 个 goroutine 映射到 N 个系统线程（`GOMAXPROCS` 控制）
- **协作式调度**：函数调用、channel 操作、系统调用时触发调度

### 启动 goroutine

```go
go func() {
    fmt.Println("Hello from goroutine")
}()
```

关键规则：

- **主 goroutine 退出时，所有子 goroutine 会被强制终止**
- 需要同步机制（`sync.WaitGroup`、channel）等待子 goroutine 完成

### 闭包捕获变量（Go 1.22+）

```go
for i := range 3 {
    go func() {
        fmt.Println(i) // Go 1.22+ 每次循环都是新变量，安全
    }()
}
```

**Go 1.21 及更早版本**：

```go
for i := range 3 {
    i := i // 必须重新声明，否则所有 goroutine 共享同一个 i
    go func() {
        fmt.Println(i)
    }()
}
```

**实测输出**：

```
Hello from goroutine
Loop 0
Loop 1
Loop 2

goroutine 特点：
• 轻量级：初始栈只有 2KB，按需增长
• 调度：M:N 模型，Go 运行时调度到系统线程
• 主 goroutine 退出时，所有子 goroutine 会被强制终止
```

---

## 20.2 sync.WaitGroup：等待多个 goroutine 完成

### 基本用法

```go
var wg sync.WaitGroup

for i := range 3 {
    wg.Add(1) // 启动前 +1
    go func(id int) {
        defer wg.Done() // 完成时 -1
        fmt.Printf("Worker %d done\n", id)
    }(i)
}

wg.Wait() // 等待计数器归零
```

### 规则

| 方法 | 作用 |
| --- | --- |
| `Add(delta int)` | 计数器 += delta（可以为负） |
| `Done()` | 等价于 `Add(-1)` |
| `Wait()` | 阻塞直到计数器归零 |

**注意事项**：

- `Add()` **必须在启动 goroutine 前调用**（否则 `Wait()` 可能提前返回）
- 计数器不能为负（panic）
- `WaitGroup` 不能复制（包含内部状态）

**实测输出**：

```
Worker 0 done
Worker 1 done
Worker 2 done
All workers finished

WaitGroup 规则：
• Add() 必须在启动 goroutine 前调用
• Done() 等价于 Add(-1)
• Wait() 阻塞直到计数器归零
• 计数器不能为负（panic）
```

---

## 20.3 channel 基础

### 无缓冲 channel：同步

```go
ch := make(chan int)

go func() {
    ch <- 42 // 发送会阻塞，直到有人接收
}()

val := <-ch // 接收会阻塞，直到有人发送
```

**同步语义**：发送方和接收方必须同时就绪（握手）。

### 有缓冲 channel：异步

```go
ch := make(chan string, 2)
ch <- "hello"
ch <- "world"
// 缓冲区满之前不阻塞

fmt.Println(<-ch) // hello
fmt.Println(<-ch) // world
```

### 操作汇总

| 操作 | 无缓冲 channel | 有缓冲 channel | nil channel | 已关闭 channel |
| --- | --- | --- | --- | --- |
| 发送 `ch <- v` | 阻塞直到接收 | 缓冲区满时阻塞 | 永久阻塞 | **panic** |
| 接收 `<-ch` | 阻塞直到发送 | 缓冲区空时阻塞 | 永久阻塞 | 返回零值 + false |
| 关闭 `close(ch)` | 通知接收者 | 通知接收者 | **panic** | **panic** |

**实测输出**：

```
无缓冲 channel 收到：42
有缓冲 channel 收到：hello, world

channel 特点：
• 无缓冲 channel：发送方和接收方必须同时就绪（握手）
• 有缓冲 channel：缓冲区满时发送阻塞，空时接收阻塞
• 向 nil channel 收发会永久阻塞
• 向已关闭 channel 发送会 panic
```

---

## 20.4 channel 方向：只发送、只接收

### 类型声明

```go
chan T      // 双向 channel（可发送、可接收）
chan<- T    // 只能发送
<-chan T    // 只能接收
```

### 用途：约束函数参数

```go
func producer(ch chan<- int) {
    ch <- 100
    close(ch)
}

func consumer(ch <-chan int) {
    val := <-ch
    fmt.Println(val)
}
```

**隐式转换**：

- 双向 channel 可以隐式转换为单向
- 单向 channel **不能**转换回双向

**实测输出**：

```
Consumer 收到：100

channel 方向：
• chan<- T：只能发送
• <-chan T：只能接收
• 双向 channel 可以隐式转换为单向
```

---

## 20.5 channel 关闭与 range 遍历

### 关闭规则

- **只有发送方可以关闭 channel**（接收方无法判断是否还有数据）
- 关闭后不能再发送（panic），但可以接收
- 从已关闭 channel 接收：返回零值 + `false`

### range 遍历

```go
ch := make(chan int, 3)

go func() {
    for i := range 3 {
        ch <- i * 10
    }
    close(ch) // 关闭后，range 会退出
}()

for val := range ch {
    fmt.Println(val)
}
```

### 检测关闭

```go
val, ok := <-ch
if !ok {
    fmt.Println("channel 已关闭")
}
```

**实测输出**：

```
收到：0
收到：10
收到：20
从已关闭 channel 接收：val=0, ok=false

关闭规则：
• 只有发送方可以关闭 channel
• 关闭后不能再发送（panic），但可以接收
• 接收时用 val, ok := <-ch 检测是否已关闭
• range 会在 channel 关闭后自动退出
```

---

## 20.6 select：多路复用 channel

### 基本用法

```go
select {
case msg := <-ch1:
    fmt.Println("从 ch1 收到", msg)
case msg := <-ch2:
    fmt.Println("从 ch2 收到", msg)
}
```

### 规则

- **多个 case 就绪时**，随机选择一个
- **都不就绪时**，阻塞直到某个 case 就绪
- **有 default**：select 变成非阻塞

### 超时控制

```go
select {
case val := <-ch:
    fmt.Println(val)
case <-time.After(50 * time.Millisecond):
    fmt.Println("超时")
}
```

### 非阻塞接收

```go
select {
case val := <-ch:
    fmt.Println(val)
default:
    fmt.Println("没有数据，立即返回")
}
```

### 空 select

```go
select {} // 永久阻塞
```

**实测输出**：

```
收到 from ch1
超时：50ms 内没有收到数据
没有数据，立即返回

select 规则：
• 多个 case 就绪时，随机选择一个
• 都不就绪时阻塞，直到某个 case 就绪
• default 使 select 变成非阻塞
• select {} 会永久阻塞
```

---

## 20.7 sync.Mutex：互斥锁

### 基本用法

```go
var (
    mu      sync.Mutex
    counter int
)

mu.Lock()
counter++
mu.Unlock()
```

### 习惯用法

```go
mu.Lock()
defer mu.Unlock()

// 临界区代码
counter++
```

### 规则

- `Lock()` 获取锁，`Unlock()` 释放锁
- **同一 goroutine 不能重复 Lock**（死锁）
- `Unlock()` 必须由 `Lock()` 的 goroutine 调用
- `Mutex` 不能复制（包含内部状态）

**实测输出**：

```
最终计数：10（期望 10）

Mutex 规则：
• Lock() 获取锁，Unlock() 释放锁
• 同一 goroutine 不能重复 Lock（死锁）
• Unlock() 必须由 Lock() 的 goroutine 调用
• 习惯用法：defer mu.Unlock()
```

---

## 20.8 sync.RWMutex：读写锁

### 基本用法

```go
var (
    mu   sync.RWMutex
    data map[string]int
)

// 读者
mu.RLock()
val := data["key"]
mu.RUnlock()

// 写者
mu.Lock()
data["key"] = 42
mu.Unlock()
```

### 规则

- **读锁（RLock）**：多个读者可以并发
- **写锁（Lock）**：独占访问，阻塞所有读者和写者
- 读多写少场景下性能优于 `Mutex`

### 何时使用

| 场景 | 推荐 |
| --- | --- |
| 读多写少 | `RWMutex` |
| 读写均衡或写多 | `Mutex`（更简单） |
| 纯读 | 不需要锁 |

**实测输出**：

```
读者 1：data=map[a:1]
读者 2：data=map[a:1]
读者 3：data=map[a:1]
写者：data= map[a:1 b:2]

RWMutex 规则：
• RLock() / RUnlock()：读锁，多个读者可以并发
• Lock() / Unlock()：写锁，独占访问
• 读多写少场景下性能优于 Mutex
```

---

## 20.9 sync.Once：确保只执行一次

### 基本用法

```go
var once sync.Once

once.Do(func() {
    fmt.Println("初始化（只会执行一次）")
})
```

### 规则

- `Do(f)` 只会执行一次 `f`
- 后续调用会阻塞，直到第一次执行完成
- 常用于**单例初始化**、**配置加载**

### 单例模式

```go
var (
    instance *Config
    once     sync.Once
)

func GetConfig() *Config {
    once.Do(func() {
        instance = &Config{/* ... */}
    })
    return instance
}
```

**实测输出**：

```
初始化（只会打印一次）

Once 规则：
• Do(f) 只会执行一次 f
• 后续调用会阻塞，直到第一次执行完成
• 常用于单例初始化、配置加载
```

---

## 20.10 sync/atomic：原子操作

### Go 1.19+ 泛型类型

```go
var counter atomic.Int64

counter.Add(1)
counter.Store(100)
val := counter.Load()
```

### 比较并交换（CAS）

```go
old := counter.Load()
swapped := counter.CompareAndSwap(old, 2000)
```

### 何时使用

| 场景 | 推荐 |
| --- | --- |
| 简单计数器、标志位 | `atomic` |
| 保护复杂数据结构 | `Mutex` / `RWMutex` |
| 通信、传递所有权 | `channel` |

**实测输出**：

```
原子计数器：1000（期望 1000）
CAS：old=1000, swapped=true, new=2000

atomic 规则：
• 提供原子操作，无需锁
• 适用于简单的计数器、标志位
• Go 1.19+ 提供泛型类型：atomic.Int64, atomic.Pointer[T]
```

---

## 20.11 数据竞争与 -race 检测

### 什么是数据竞争

**数据竞争（Data Race）**：多个 goroutine 并发访问同一变量，至少一个是写操作，且没有同步。

```go
var counter int

for range 10 {
    go func() {
        counter++ // 数据竞争
    }()
}
```

### 检测方法

```bash
go test -race
go run -race main.go
go build -race
```

### 修复方法

| 方法 | 适用场景 |
| --- | --- |
| `Mutex` / `RWMutex` | 保护共享状态 |
| `channel` | 传递所有权、流式处理 |
| `atomic` | 简单计数器、标志位 |

### channel 与锁的取舍

**Don't communicate by sharing memory; share memory by communicating.**

| 场景 | 推荐 |
| --- | --- |
| 传递所有权 | channel |
| 流式处理、pipeline | channel |
| goroutine 间通信 | channel |
| 保护共享状态 | Mutex |
| 短临界区 | Mutex |
| 缓存、配置 | RWMutex |

**实测输出**：

```
数据竞争示例：counter=10（结果不确定）

数据竞争：
• 多个 goroutine 并发访问同一变量，至少一个是写操作
• 使用 go test -race 或 go run -race 检测
• 修复方法：mutex、channel、atomic

channel 与锁的取舍：
• channel：适合传递所有权、流式处理、goroutine 间通信
• mutex：适合保护共享状态、短临界区、缓存
• "Don't communicate by sharing memory; share memory by communicating."
```

---

## 5 个真实报错怎么读

### 报错 1：向已关闭 channel 发送

```
panic: send on closed channel
```

**原因**：channel 已关闭，不能再发送。

**修复**：

- 只有发送方可以关闭 channel
- 确保发送方在关闭前完成所有发送

### 报错 2：WaitGroup 计数器为负

```
panic: sync: negative WaitGroup counter
```

**原因**：`Done()` 次数超过 `Add()` 次数。

**修复**：

- 确保每个 goroutine 只调用一次 `Done()`
- `Add()` 必须在启动 goroutine 前调用

### 报错 3：unlock of unlocked mutex

```
fatal error: sync: unlock of unlocked mutex
```

**原因**：`Unlock()` 调用了两次，或未配对。

**修复**：

- 检查 `defer mu.Unlock()` 是否重复
- 确保每个 `Lock()` 对应一个 `Unlock()`

### 报错 4：数据竞争（-race 检测）

```
WARNING: DATA RACE
Write at 0x00c000014098 by goroutine 7:
  main.main.func1()
      main.go:15 +0x3c
Previous write at 0x00c000014098 by goroutine 6:
  main.main.func1()
      main.go:15 +0x3c
```

**原因**：多个 goroutine 并发写同一变量，无同步。

**修复**：

- 用 `Mutex` 保护共享变量
- 或用 `atomic` 原子操作
- 或用 channel 传递所有权

### 报错 5：死锁检测

```
fatal error: all goroutines are asleep - deadlock!
```

**原因**：所有 goroutine 都在等待，无法继续。

**常见场景**：

- 无缓冲 channel，没有接收方
- `WaitGroup.Wait()` 但没人 `Done()`
- 循环等待（A 等 B，B 等 A）

**修复**：

- 检查 channel 是否有接收方
- 检查 `WaitGroup` 计数器是否正确
- 用 `select` + `default` 避免永久阻塞

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| goroutine 没执行完就退出 | 主 goroutine 提前退出 | 用 `WaitGroup` 或 channel 等待 |
| 循环变量捕获错误 | Go 1.21- 循环变量共享 | 在循环内重新声明 `i := i` |
| channel 死锁 | 无缓冲 channel 缺少接收方 | 用有缓冲 channel 或启动接收 goroutine |
| `WaitGroup` 提前返回 | `Add()` 在 goroutine 内调用 | `Add()` 必须在 `go func()` 前调用 |
| 数据竞争 | 并发写共享变量无同步 | 用 `Mutex`、`atomic` 或 channel |
| 关闭已关闭 channel | 多个 goroutine 都调用 `close()` | 只有一个发送方负责关闭 |
| `Mutex` 死锁 | 同一 goroutine 重复 Lock | 用 `defer` 确保 Unlock，不要嵌套 Lock |
| select 永久阻塞 | 所有 case 都不就绪，无 default | 加 `default` 或 `time.After` 超时 |

---

## 练习

### 第 1 题

用 goroutine 和 channel 实现一个并发求和：给定 `[]int{1, 2, 3, 4, 5}`，启动 2 个 worker 分别计算一部分，最后汇总结果。

::: details 第 1 题参考答案

```go
func concurrentSum(nums []int) int {
    mid := len(nums) / 2
    ch := make(chan int, 2)

    go func() {
        sum := 0
        for _, v := range nums[:mid] {
            sum += v
        }
        ch <- sum
    }()

    go func() {
        sum := 0
        for _, v := range nums[mid:] {
            sum += v
        }
        ch <- sum
    }()

    return <-ch + <-ch
}

func main() {
    nums := []int{1, 2, 3, 4, 5}
    fmt.Println(concurrentSum(nums)) // 15
}
```

**为什么这样写更好**：

- 有缓冲 channel（容量 2）避免死锁
- 每个 worker 负责一部分数据，主 goroutine 汇总结果
- 不需要 `WaitGroup`，channel 本身就是同步点

:::

### 第 2 题

解释为什么无缓冲 channel 可以用作同步信号，并给出示例。

::: details 第 2 题参考答案

**无缓冲 channel 的同步语义**：发送方和接收方必须同时就绪。

**示例**：等待 goroutine 完成

```go
done := make(chan struct{}) // 空结构体不占内存

go func() {
    // 执行任务
    time.Sleep(100 * time.Millisecond)
    done <- struct{}{} // 发送信号
}()

<-done // 等待信号
fmt.Println("任务完成")
```

**为什么这样写更好**：

- `chan struct{}` 不占内存，只传递信号
- 无缓冲 channel 保证发送和接收同步
- 比 `time.Sleep` 精确，不依赖时间估算

:::

### 第 3 题

用 `sync.Once` 实现一个线程安全的单例模式。

::: details 第 3 题参考答案

```go
type Config struct {
    Host string
    Port int
}

var (
    instance *Config
    once     sync.Once
)

func GetConfig() *Config {
    once.Do(func() {
        // 模拟从文件加载
        instance = &Config{
            Host: "localhost",
            Port: 8080,
        }
    })
    return instance
}

func main() {
    var wg sync.WaitGroup
    for range 10 {
        wg.Add(1)
        go func() {
            defer wg.Done()
            cfg := GetConfig()
            fmt.Println(cfg.Host)
        }()
    }
    wg.Wait()
}
```

**为什么这样写更好**：

- `Once.Do` 保证只执行一次，线程安全
- 后续调用会阻塞，直到第一次完成
- 比手动用 `Mutex` + bool 标志更简洁

:::

### 第 4 题

解释 `select` 的随机选择行为，并说明为什么需要这样设计。

::: details 第 4 题参考答案

**随机选择行为**：

```go
ch1 := make(chan int, 1)
ch2 := make(chan int, 1)

ch1 <- 1
ch2 <- 2

select {
case <-ch1:
    fmt.Println("ch1")
case <-ch2:
    fmt.Println("ch2")
}
// 输出不确定，每次运行可能不同
```

**为什么需要随机**：

- **避免饥饿**：如果总是优先选择第一个就绪的 case，后面的 case 可能永远得不到执行
- **公平调度**：所有就绪的 case 有相同的机会被选中
- **防止死锁**：随机化减少了循环等待的可能性

**实际应用**：

- 多个 worker 向同一个 channel 发送结果，select 随机选择避免某个 worker 被优待
- 负载均衡：多个后端服务 channel，随机选择避免总是选同一个

:::

### 第 5 题

用 `atomic.CompareAndSwap` 实现一个无锁的计数器自增，并解释 CAS 的工作原理。

::: details 第 5 题参考答案

```go
func atomicIncrement(counter *atomic.Int64) {
    for {
        old := counter.Load()       // 读取当前值
        new := old + 1
        if counter.CompareAndSwap(old, new) {
            return // 成功
        }
        // 失败，重试
    }
}

func main() {
    var counter atomic.Int64
    var wg sync.WaitGroup

    for range 1000 {
        wg.Add(1)
        go func() {
            defer wg.Done()
            atomicIncrement(&counter)
        }()
    }

    wg.Wait()
    fmt.Println(counter.Load()) // 1000
}
```

**CAS 工作原理**：

1. 读取当前值 `old`
2. 计算新值 `new = old + 1`
3. 尝试交换：如果当前值仍是 `old`，则更新为 `new`；否则失败
4. 失败则重试（有人先改了）

**为什么这样写更好**：

- 无锁，性能高于 `Mutex`
- 适合高竞争场景（多个 goroutine 同时修改）
- CPU 提供原子指令支持（x86 的 CMPXCHG）

**何时使用 CAS**：

- 简单的计数器、标志位
- 乐观锁（假设冲突少）
- 不适合保护复杂数据结构（用 `Mutex` 更简单）

:::

### 第 6 题

实现一个 worker pool：有 N 个 worker 从 jobs channel 取任务，执行后将结果发送到 results channel。

::: details 第 6 题参考答案

```go
func workerPool(numWorkers int, jobs <-chan int, results chan<- int) {
    var wg sync.WaitGroup

    // 启动 N 个 worker
    for range numWorkers {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for job := range jobs {
                // 模拟任务处理
                results <- job * 2
            }
        }()
    }

    // 等待所有 worker 完成，然后关闭 results
    go func() {
        wg.Wait()
        close(results)
    }()
}

func main() {
    jobs := make(chan int, 10)
    results := make(chan int, 10)

    // 启动 3 个 worker
    workerPool(3, jobs, results)

    // 发送 5 个任务
    for i := range 5 {
        jobs <- i
    }
    close(jobs)

    // 收集结果
    for result := range results {
        fmt.Println(result)
    }
}
```

**为什么这样写更好**：

- 固定数量的 worker，避免 goroutine 爆炸
- 用 channel 分发任务，自动负载均衡
- 关闭 `jobs` 后，worker 会自动退出（range 检测到关闭）
- 用 goroutine 等待所有 worker 完成后关闭 `results`

**实际应用**：

- 并发下载文件
- 批量处理数据库记录
- 限制并发请求数

:::

---

## 小结

- **goroutine** 是轻量级线程，初始栈 2KB，M:N 调度
- **channel** 是 goroutine 间通信的管道，分无缓冲（同步）和有缓冲（异步）
- **select** 多路复用 channel，支持超时和非阻塞
- **sync.WaitGroup** 等待多个 goroutine 完成
- **sync.Mutex** 保护共享状态，`defer mu.Unlock()` 是习惯用法
- **sync.RWMutex** 读写锁，读多写少场景性能更优
- **sync.Once** 确保只执行一次，常用于单例初始化
- **sync/atomic** 提供原子操作，无需锁，适合简单计数器
- **数据竞争**用 `go test -race` 检测，修复方法：Mutex、channel、atomic
- **channel 与锁的取舍**：channel 适合通信和传递所有权，Mutex 适合保护共享状态

下一章将讲解**并发模式**，学习常见的并发设计模式与最佳实践。
