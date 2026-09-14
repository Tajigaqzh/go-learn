# 第 21 章 并发模式与陷阱

在第 20 章中，我们学习了 Go 并发的基础原语：goroutine、channel、sync 包和原子操作。本章将这些工具组合起来，讲解真实项目中常用的并发模式、如何避免常见陷阱、以及如何测试并发代码。

**本章配套代码**在 `internal/chapter/go21_concurrency_patterns/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

---

## 21.1 Worker Pool（工作池）

### 模式说明

**Worker Pool** 是最常见的并发模式：固定数量的 worker goroutine 从任务队列（channel）中取任务并处理，避免创建过多 goroutine 导致资源耗尽。

### 实现结构

```
任务生产者 → jobs channel → worker 1 → results channel
                          → worker 2 → results channel
                          → worker 3 → results channel
```

### 代码示例

```go
jobs := make(chan int, 10)
results := make(chan int, 10)

// 启动 3 个 worker
for w := 1; w <= 3; w++ {
    go worker(w, jobs, results)
}

// 发送 5 个任务
for j := 1; j <= 5; j++ {
    jobs <- j
}
close(jobs) // 关闭任务队列

// 收集结果
for a := 1; a <= 5; a++ {
    <-results
}

func worker(id int, jobs <-chan int, results chan<- int) {
    for j := range jobs {
        fmt.Printf("worker %d 处理 job %d\n", id, j)
        time.Sleep(10 * time.Millisecond)
        results <- j * 2
    }
}
```

**实测输出**：

```
Worker Pool 模式：固定数量的 worker goroutine 消费任务队列
  worker 1 处理 job 1
  worker 2 处理 job 2
  worker 3 处理 job 3
  worker 1 处理 job 4
  worker 2 处理 job 5
✓ Worker Pool 完成，所有任务已处理
```

### 使用场景

- **限制并发数**：避免创建过多 goroutine（如爬虫、批量 API 调用）
- **任务队列**：生产者-消费者模型，生产速度与消费速度解耦
- **资源复用**：每个 worker 可以持有连接、缓冲区等昂贵资源

### 注意事项

- 务必关闭 `jobs` channel，否则 worker 永远阻塞在 `range`
- `results` channel 容量要足够，或用单独的 goroutine 消费
- worker 数量通常设为 `runtime.NumCPU()` 或根据 I/O 密集度调整

---

## 21.2 生产者消费者（Producer-Consumer）

### 模式说明

**生产者**写入 channel，**消费者**读取并处理。生产者和消费者解耦，可以独立调整速度。

### 代码示例

```go
ch := make(chan int, 5)
var wg sync.WaitGroup

// 生产者
wg.Add(1)
go func() {
    defer wg.Done()
    defer close(ch) // 生产完毕后关闭 channel
    for i := 1; i <= 3; i++ {
        fmt.Printf("生产者：写入 %d\n", i)
        ch <- i
    }
}()

// 消费者
wg.Add(1)
go func() {
    defer wg.Done()
    for v := range ch { // 读到 channel 关闭为止
        fmt.Printf("消费者：读取 %d\n", v)
        time.Sleep(10 * time.Millisecond)
    }
}()

wg.Wait()
```

**实测输出**：

```
生产者消费者模式：生产者写入 channel，消费者读取并处理
  生产者：写入 1
  生产者：写入 2
  生产者：写入 3
  消费者：读取 1
  消费者：读取 2
  消费者：读取 3
✓ 生产者消费者完成
```

### 关键点

- **生产者关闭 channel**：消费者通过 `range` 自动检测关闭
- **缓冲区大小**：平衡生产和消费速度，避免生产者阻塞
- **多生产者多消费者**：可以启动多个生产者和消费者 goroutine

---

## 21.3 Fan-In / Fan-Out

### 模式说明

- **Fan-Out**：一个输入分发给多个 worker 并发处理
- **Fan-In**：多个 channel 的输出合并到一个 channel

### 代码示例

```go
// Fan-Out：分发任务给多个 worker
input := make(chan int, 5)
c1 := fanOutWorker(input)
c2 := fanOutWorker(input)

// 发送任务
go func() {
    for i := 1; i <= 4; i++ {
        input <- i
    }
    close(input)
}()

// Fan-In：合并结果
output := fanIn(c1, c2)
for v := range output {
    fmt.Println(v)
}

func fanOutWorker(input <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for v := range input {
            out <- v * 2
        }
    }()
    return out
}

func fanIn(channels ...<-chan int) <-chan int {
    out := make(chan int)
    var wg sync.WaitGroup
    for _, ch := range channels {
        wg.Add(1)
        go func(c <-chan int) {
            defer wg.Done()
            for v := range c {
                out <- v
            }
        }(ch)
    }
    go func() {
        wg.Wait()
        close(out)
    }()
    return out
}
```

**实测输出**：

```
Fan-Out：一个输入分发给多个 worker
Fan-In：多个 channel 合并到一个输出
  worker 处理 1
  worker 处理 2
  worker 处理 3
  worker 处理 4
✓ Fan-In / Fan-Out 完成
```

### 使用场景

- **并行处理**：多个 worker 同时处理任务（如图片处理、数据转换）
- **结果聚合**：多个数据源合并到一个输出（如多个 API 调用结果合并）

---

## 21.4 Pipeline（流水线）

### 模式说明

**Pipeline** 将数据处理分成多个阶段，每个阶段是一个 goroutine，通过 channel 串联。数据从一个阶段流向下一个阶段。

### 代码示例

```go
// 阶段1：生成数字
gen := func(nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for _, n := range nums {
            out <- n
        }
    }()
    return out
}

// 阶段2：平方
sq := func(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for n := range in {
            out <- n * n
        }
    }()
    return out
}

// 连接 pipeline
c := gen(2, 3, 4)
out := sq(c)

// 消费结果
for n := range out {
    fmt.Println(n) // 4, 9, 16
}
```

**实测输出**：

```
Pipeline 模式：多个阶段串联，每个阶段是一个 goroutine
  pipeline 输出：4
  pipeline 输出：9
  pipeline 输出：16
✓ Pipeline 完成
```

### 使用场景

- **数据转换链**：读取 → 解析 → 过滤 → 转换 → 写入
- **流式处理**：日志处理、ETL、数据清洗

### 注意事项

- 每个阶段必须正确关闭输出 channel
- 支持取消：传入 `context`，每个阶段检查 `ctx.Done()`

---

## 21.5 取消与超时组合

### 使用 context 实现取消传播

```go
ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
defer cancel()

result := make(chan string, 1)
go func() {
    time.Sleep(100 * time.Millisecond) // 模拟慢任务
    result <- "done"
}()

select {
case <-ctx.Done():
    fmt.Printf("超时：%v\n", ctx.Err())
case res := <-result:
    fmt.Printf("结果：%s\n", res)
}
```

**实测输出**：

```
使用 context 实现取消传播与超时控制
  超时：context deadline exceeded
✓ context 超时正确触发
```

### 取消传播模式

```go
func process(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err() // 取消信号
        default:
            // 执行任务
        }
    }
}
```

### 最佳实践

- **context 作为第一个参数**：`func DoWork(ctx context.Context, ...)`
- **取消链**：父 context 取消时，子 context 自动取消
- **超时组合**：`WithTimeout` 套 `WithCancel`，两者都能触发取消

---

## 21.6 errgroup：简化并发错误处理

### 标准库的限制

`sync.WaitGroup` 只能等待 goroutine 完成，不能处理错误。手动收集错误很繁琐。

### errgroup 的优势

`golang.org/x/sync/errgroup` 提供：

- **错误收集**：任意 goroutine 返回错误时，`Wait()` 返回第一个错误
- **取消传播**：`WithContext` 模式下，第一个错误后自动取消其他 goroutine
- **简化代码**：不需要手动管理 channel 和 WaitGroup

### 代码示例

```go
g := new(errgroup.Group)

g.Go(func() error {
    time.Sleep(10 * time.Millisecond)
    return nil // 成功
})

g.Go(func() error {
    time.Sleep(20 * time.Millisecond)
    return errors.New("task 2 failed") // 失败
})

if err := g.Wait(); err != nil {
    fmt.Printf("errgroup 返回第一个错误：%v\n", err)
}
```

**实测输出**：

```
golang.org/x/sync/errgroup：简化并发任务的错误处理
  任务1：成功
  任务2：失败
  任务3：成功
  errgroup 返回第一个错误：task 2 failed
✓ errgroup 演示完成
```

### 带取消的版本

```go
g, ctx := errgroup.WithContext(context.Background())

g.Go(func() error {
    select {
    case <-ctx.Done():
        return ctx.Err() // 被取消
    case <-time.After(100 * time.Millisecond):
        return nil
    }
})

g.Go(func() error {
    return errors.New("fail fast") // 立即失败，取消其他任务
})

err := g.Wait() // 返回 "fail fast"
```

### 使用场景

- **并发 API 调用**：调用多个外部服务，任意失败则整体失败
- **批量任务**：任意任务失败则快速失败
- **资源加载**：并发加载多个资源，任意失败则放弃

---

## 21.7 Goroutine 泄漏

### 泄漏的定义

**Goroutine 泄漏**：goroutine 启动后永远无法退出，占用内存和调度资源。

### 常见泄漏场景

#### 场景 1：channel 发送者永久阻塞

```go
ch := make(chan int)
go func() {
    ch <- 1 // 永久阻塞，因为没有接收者
    fmt.Println("发送成功") // 永远不会打印
}()
```

**修复**：使用带缓冲的 channel，或确保有接收者。

#### 场景 2：context 未取消

```go
go func() {
    for {
        // 无限循环，没有退出条件
    }
}()
```

**修复**：使用 context 传递取消信号。

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    for {
        select {
        case <-ctx.Done():
            return // 正确退出
        default:
            // 执行任务
        }
    }
}()
```

#### 场景 3：等待永远不会到来的事件

```go
go func() {
    <-neverClosedChannel // 永远阻塞
}()
```

**修复**：使用 `select` 组合超时或取消。

```go
select {
case <-ch:
    // 正常接收
case <-ctx.Done():
    return // 取消时退出
case <-time.After(timeout):
    return // 超时退出
}
```

### 排查工具

| 工具 | 用途 |
| --- | --- |
| `runtime.NumGoroutine()` | 查看当前 goroutine 数量 |
| `pprof goroutine profile` | 查看泄漏的 goroutine 栈 |
| `go test -race` | 检测数据竞争（间接发现泄漏） |

**实测输出**：

```
Goroutine 泄漏：常见泄漏场景与预防

场景1：channel 接收者退出，发送者永久阻塞
  ⚠ 发送者 goroutine 已泄漏（没有接收者）

场景2：正确做法 - 使用 context 取消
  ✓ goroutine 收到取消信号，正常退出

排查工具：
• runtime.NumGoroutine() 查看当前 goroutine 数量
• pprof goroutine profile 查看泄漏栈
• go test -race 检测数据竞争
```

---

## 21.8 死锁

### 死锁的定义

**死锁**：所有 goroutine 都在等待，程序无法继续执行。Go 运行时检测到死锁后会 panic：

```
fatal error: all goroutines are asleep - deadlock!
```

### 常见死锁场景

#### 场景 1：循环等待锁

```go
var mu1, mu2 sync.Mutex

// goroutine A
mu1.Lock()
mu2.Lock() // 等待 mu2

// goroutine B
mu2.Lock()
mu1.Lock() // 等待 mu1，形成环路
```

**修复**：按固定顺序获取锁（如按内存地址排序）。

#### 场景 2：channel 永久阻塞

```go
ch := make(chan int)
ch <- 1 // 死锁：无缓冲 channel，没有接收者
```

**修复**：使用带缓冲的 channel，或启动接收者 goroutine。

#### 场景 3：WaitGroup 计数错误

```go
var wg sync.WaitGroup
wg.Add(1)
go func() {
    // 忘记调用 wg.Done()
}()
wg.Wait() // 永远等待
```

**修复**：使用 `defer wg.Done()` 确保调用。

### 避免死锁的原则

- **按固定顺序获取锁**
- **使用 `select` + `default` 避免阻塞**
- **使用 context 超时**
- **代码审查时画出依赖图**

**实测输出**：

```
死锁：所有 goroutine 都在等待，程序无法继续

场景1：循环等待锁
  var mu1, mu2 sync.Mutex
  goroutine A: mu1.Lock(); mu2.Lock()
  goroutine B: mu2.Lock(); mu1.Lock()
  ⚠ 可能死锁

场景2：channel 永久阻塞
  ch := make(chan int)
  ch <- 1  // fatal error: all goroutines are asleep - deadlock!

避免死锁：
• 按固定顺序获取锁
• 使用带缓冲的 channel 或 select + default
• 使用 context 超时
• 代码审查时画出依赖图
```

---

## 21.9 Channel 关闭原则

### 核心原则

**只有发送者关闭 channel，接收者永远不关闭。**

### 原因

- **向已关闭的 channel 发送会 panic**
- **从已关闭的 channel 接收返回零值和 `false`**
- **关闭已关闭的 channel 会 panic**

### 代码示例

```go
ch := make(chan int, 2)
ch <- 1
ch <- 2
close(ch) // 发送者关闭

// 接收者 range 读取
for v := range ch {
    fmt.Println(v)
}

// 检测 channel 是否关闭
v, ok := <-ch
if !ok {
    fmt.Println("channel 已关闭，v =", v) // v = 0
}
```

### 多发送者场景

当有多个发送者时，不能简单地让任意发送者关闭 channel（会导致其他发送者 panic）。

**解决方案 1：使用 `sync.Once`**

```go
var once sync.Once
closeCh := func() {
    once.Do(func() {
        close(ch)
    })
}
```

**解决方案 2：使用额外的信号 channel**

```go
done := make(chan struct{})
go func() {
    for {
        select {
        case <-done:
            return
        case ch <- value:
        }
    }
}()

close(done) // 通知所有发送者停止
```

**实测输出**：

```
Channel 关闭原则：只有发送者关闭 channel，接收者永远不关闭

✓ 正确：发送者关闭
  接收者 range 读取： 1 2

⚠ 错误：向已关闭的 channel 发送
  // ch <- 3  // panic: send on closed channel

✓ 检测 channel 是否关闭
  channel 已关闭，v = 0

多发送者场景：使用 sync.Once 或 context
  ✓ channel 只关闭一次
```

---

## 21.10 并发代码测试

### 测试策略

#### 1. 使用 `go test -race` 检测数据竞争

```bash
go test -race ./...
```

#### 2. 使用 `sync.WaitGroup` 确保 goroutine 完成

```go
func TestWorkerPool(t *testing.T) {
    var wg sync.WaitGroup
    wg.Add(3)
    for w := 1; w <= 3; w++ {
        go func() {
            defer wg.Done()
            // 执行任务
        }()
    }
    wg.Wait()
}
```

#### 3. 使用 `context.WithTimeout` 避免测试永久阻塞

```go
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

select {
case <-ctx.Done():
    t.Fatal("test timed out")
case result := <-ch:
    // 验证结果
}
```

#### 4. 测试边界条件

- channel 关闭后的行为
- context 取消传播
- 超时触发
- goroutine 泄漏预防

#### 5. 使用 `t.Parallel()` 并发运行测试

```go
func TestConcurrent(t *testing.T) {
    t.Parallel() // 与其他 Parallel 测试并发运行
    // 测试逻辑
}
```

### 示例测试

```go
func TestWorkerPool(t *testing.T) {
    jobs := make(chan int, 10)
    results := make(chan int, 10)
    var wg sync.WaitGroup

    for w := 1; w <= 3; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := range jobs {
                results <- j * 2
            }
        }()
    }

    for j := 1; j <= 5; j++ {
        jobs <- j
    }
    close(jobs)

    wg.Wait()
    close(results)

    count := 0
    for range results {
        count++
    }
    if count != 5 {
        t.Errorf("expected 5, got %d", count)
    }
}
```

**实测输出**：

```
并发代码测试

测试策略：
• 使用 go test -race 检测数据竞争
• 使用 sync.WaitGroup 确保 goroutine 完成
• 使用 context.WithTimeout 避免测试永久阻塞
• 测试取消、超时、channel 关闭等边界条件
• 使用 t.Parallel() 并发运行测试用例

示例：测试 worker pool
  func TestWorkerPool(t *testing.T) {
      jobs := make(chan int, 10)
      results := make(chan int, 10)
      var wg sync.WaitGroup
      ...
  }
```

---

## 5 个真实报错怎么读

### 报错 1：死锁

```
fatal error: all goroutines are asleep - deadlock!

goroutine 1 [chan send]:
main.main()
	/path/to/file.go:10 +0x50
```

**原因**：无缓冲 channel 发送时没有接收者。

**修复**：使用带缓冲的 channel，或启动接收者 goroutine。

### 报错 2：向已关闭的 channel 发送

```
panic: send on closed channel

goroutine 2 [running]:
main.worker()
	/path/to/file.go:15 +0x30
```

**原因**：channel 已被关闭，仍尝试发送。

**修复**：使用 `done` channel 通知发送者停止，或使用 `sync.Once` 确保只关闭一次。

### 报错 3：关闭已关闭的 channel

```
panic: close of closed channel

goroutine 1 [running]:
main.main()
	/path/to/file.go:20 +0x40
```

**原因**：多次关闭同一个 channel。

**修复**：使用 `sync.Once` 确保只关闭一次。

### 报错 4：数据竞争

```
WARNING: DATA RACE
Write at 0x00c000014090 by goroutine 7:
  main.increment()
      /path/to/file.go:10 +0x3c

Previous read at 0x00c000014090 by goroutine 6:
  main.read()
      /path/to/file.go:15 +0x3c
```

**原因**：多个 goroutine 同时读写同一个变量，没有同步。

**修复**：使用 `sync.Mutex` 或 `sync/atomic`。

### 报错 5：context 已取消

```
context canceled
```

**原因**：context 被取消（`cancel()` 调用或父 context 取消）。

**修复**：检查 `ctx.Done()` 并正确处理取消信号。

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| goroutine 永远不退出 | channel 发送阻塞或无限循环 | 使用 context 或 `select` + 超时 |
| 死锁 panic | 所有 goroutine 都在等待 | 按固定顺序获取锁，使用带缓冲 channel |
| 向已关闭 channel 发送 panic | 发送者不知道 channel 已关闭 | 使用 `done` channel 通知发送者 |
| 关闭已关闭 channel panic | 多个发送者都尝试关闭 | 使用 `sync.Once` 确保只关闭一次 |
| 数据竞争 | 多个 goroutine 无同步读写 | 使用 `sync.Mutex` 或 `sync/atomic` |
| WaitGroup 永远等待 | 忘记调用 `Done()` 或 `Add/Done` 不匹配 | 使用 `defer wg.Done()` |
| context 取消不传播 | 没有传递 context 给子任务 | context 作为第一个参数传递 |
| channel 泄漏 | 发送者没有接收者，或接收者没有发送者 | 确保 channel 两端都存在，或使用 `select` + `default` |

---

## 练习

### 第 1 题

实现一个带并发上限的 Worker Pool，最多同时处理 N 个任务，剩余任务排队等待。

::: details 第 1 题参考答案

```go
func WorkerPoolWithLimit(tasks []int, workerCount int) []int {
    jobs := make(chan int, len(tasks))
    results := make(chan int, len(tasks))

    // 启动固定数量的 worker
    var wg sync.WaitGroup
    for w := 1; w <= workerCount; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := range jobs {
                results <- j * 2 // 处理任务
            }
        }()
    }

    // 发送所有任务
    for _, task := range tasks {
        jobs <- task
    }
    close(jobs)

    // 等待所有 worker 完成
    wg.Wait()
    close(results)

    // 收集结果
    var output []int
    for r := range results {
        output = append(output, r)
    }
    return output
}
```

**为什么这样写更好**：

- 使用固定数量的 worker 限制并发数
- 任务通过 channel 排队，避免创建过多 goroutine
- `WaitGroup` 确保所有任务完成后再收集结果

:::

### 第 2 题

实现一个支持取消的 Pipeline，使用 context 在任意阶段取消整个流水线。

::: details 第 2 题参考答案

```go
func CancellablePipeline(ctx context.Context, nums []int) <-chan int {
    gen := func(ctx context.Context, nums []int) <-chan int {
        out := make(chan int)
        go func() {
            defer close(out)
            for _, n := range nums {
                select {
                case <-ctx.Done():
                    return // 取消时退出
                case out <- n:
                }
            }
        }()
        return out
    }

    sq := func(ctx context.Context, in <-chan int) <-chan int {
        out := make(chan int)
        go func() {
            defer close(out)
            for n := range in {
                select {
                case <-ctx.Done():
                    return // 取消时退出
                case out <- n * n:
                }
            }
        }()
        return out
    }

    c := gen(ctx, nums)
    return sq(ctx, c)
}

// 使用示例
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

out := CancellablePipeline(ctx, []int{1, 2, 3, 4, 5})
cancel() // 取消整个 pipeline
```

**为什么这样写更好**：

- 每个阶段都检查 `ctx.Done()`
- 取消信号传播到所有阶段
- 避免 goroutine 泄漏

:::

### 第 3 题

使用 `errgroup` 并发调用 3 个 API，任意一个失败则快速失败并返回错误。

::: details 第 3 题参考答案

```go
func FetchAll(ctx context.Context) ([]string, error) {
    g, ctx := errgroup.WithContext(ctx)
    results := make([]string, 3)

    apis := []string{
        "https://api1.example.com",
        "https://api2.example.com",
        "https://api3.example.com",
    }

    for i, url := range apis {
        i, url := i, url // 捕获循环变量
        g.Go(func() error {
            select {
            case <-ctx.Done():
                return ctx.Err()
            default:
            }

            // 模拟 API 调用
            resp, err := http.Get(url)
            if err != nil {
                return fmt.Errorf("fetch %s: %w", url, err)
            }
            defer resp.Body.Close()

            body, err := io.ReadAll(resp.Body)
            if err != nil {
                return err
            }

            results[i] = string(body)
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return nil, err
    }
    return results, nil
}
```

**为什么这样写更好**：

- `errgroup.WithContext` 自动取消：一个失败后取消其他任务
- 避免等待所有任务完成才返回错误
- 代码简洁，不需要手动管理 channel 和 WaitGroup

:::

### 第 4 题

编写一个函数检测 goroutine 泄漏：启动 10 个 goroutine，5 秒后检查是否都已退出。

::: details 第 4 题参考答案

```go
func TestGoroutineLeak(t *testing.T) {
    before := runtime.NumGoroutine()

    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            select {
            case <-ctx.Done():
                return
            case <-time.After(1 * time.Second):
                // 任务完成
            }
        }()
    }

    wg.Wait()

    // 等待 goroutine 清理
    time.Sleep(50 * time.Millisecond)

    after := runtime.NumGoroutine()
    leaked := after - before

    if leaked > 0 {
        t.Errorf("leaked %d goroutines", leaked)
    }
}
```

**为什么这样写更好**：

- 使用 `runtime.NumGoroutine()` 检测泄漏
- context 确保所有 goroutine 能够退出
- `WaitGroup` 等待所有 goroutine 完成

:::

### 第 5 题

实现一个支持超时的 Worker Pool，如果任务执行超过指定时间则跳过。

::: details 第 5 题参考答案

```go
func WorkerPoolWithTimeout(tasks []int, workerCount int, timeout time.Duration) []int {
    jobs := make(chan int, len(tasks))
    results := make(chan int, len(tasks))

    var wg sync.WaitGroup
    for w := 1; w <= workerCount; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := range jobs {
                ctx, cancel := context.WithTimeout(context.Background(), timeout)
                result := make(chan int, 1)

                go func() {
                    time.Sleep(10 * time.Millisecond) // 模拟任务
                    result <- j * 2
                }()

                select {
                case r := <-result:
                    results <- r
                case <-ctx.Done():
                    fmt.Printf("task %d timed out\n", j)
                }
                cancel()
            }
        }()
    }

    for _, task := range tasks {
        jobs <- task
    }
    close(jobs)

    wg.Wait()
    close(results)

    var output []int
    for r := range results {
        output = append(output, r)
    }
    return output
}
```

**为什么这样写更好**：

- 每个任务独立设置超时
- 超时任务不阻塞 worker
- 其他任务继续执行

:::

### 第 6 题

解释为什么「只有发送者关闭 channel」原则很重要，并举例说明违反这个原则会导致什么问题。

::: details 第 6 题参考答案

**原则重要性**：

1. **向已关闭的 channel 发送会 panic**
2. **关闭已关闭的 channel 会 panic**
3. **接收者无法判断 channel 是否还会有新数据**

**违反原则的后果**：

```go
// 错误示例：接收者关闭 channel
ch := make(chan int)

// 接收者
go func() {
    for v := range ch {
        fmt.Println(v)
    }
    close(ch) // ⚠ 接收者关闭
}()

// 发送者
ch <- 1
ch <- 2 // ⚠ panic: send on closed channel
```

**正确做法**：

```go
ch := make(chan int)

// 发送者
go func() {
    ch <- 1
    ch <- 2
    close(ch) // ✓ 发送者关闭
}()

// 接收者
for v := range ch {
    fmt.Println(v)
}
```

**多发送者场景**：

当有多个发送者时，使用额外的信号 channel：

```go
done := make(chan struct{})

// 多个发送者
for i := 0; i < 3; i++ {
    go func() {
        for {
            select {
            case <-done:
                return // 收到停止信号
            case ch <- value:
            }
        }
    }()
}

// 主 goroutine 控制关闭
close(done) // 通知所有发送者停止
time.Sleep(time.Millisecond) // 等待发送者退出
close(ch) // 最后关闭 channel
```

**为什么更好**：

- 避免 panic
- 清晰的所有权语义
- 多发送者场景下更安全

:::

---

## 小结

- **Worker Pool** 限制并发数，避免 goroutine 爆炸
- **生产者消费者** 解耦生产和消费速度，生产者关闭 channel
- **Fan-In / Fan-Out** 并行处理和结果聚合
- **Pipeline** 流水线处理，每个阶段是一个 goroutine
- **context** 实现取消传播与超时控制
- **errgroup** 简化并发错误处理，支持快速失败
- **goroutine 泄漏**：使用 context 确保 goroutine 能够退出
- **死锁**：按固定顺序获取锁，使用带缓冲 channel 或超时
- **channel 关闭原则**：只有发送者关闭，接收者永远不关闭
- **并发测试**：`go test -race`、`WaitGroup`、context 超时、边界测试

下一章将讲解 **context 与生命周期管理**，深入理解 context 的设计哲学和使用原则。
<!-- CONTINUATION_MARKER_1 -->