# 第 22 章 context 与生命周期管理

`context` 包是 Go 并发编程中管理生命周期的核心工具。它提供了取消信号传播、超时控制、截止时间管理和请求范围值传递等功能。本章深入讲解 context 的设计哲学、使用原则和常见陷阱。

**本章配套代码**在 `internal/chapter/go22_context/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

---

## 22.1 context.Background() 与 context.TODO()

### 两个顶层 context

```go
ctx1 := context.Background()
ctx2 := context.TODO()
```

| Context | 用途 |
| --- | --- |
| `Background()` | 顶层 context，用于 main、初始化、测试 |
| `TODO()` | 占位符，不确定用哪个 context 时先用 TODO |

### 共同特点

- 永远不会被取消
- 没有截止时间
- 没有值
- 是所有 context 树的根节点

**实测输出**：

```
context.Background() 与 context.TODO()
Background context: context.Background
TODO context: context.TODO

使用场景：
• Background：顶层 context，main 函数、初始化、测试
• TODO：占位符，不确定用哪个 context 时先用 TODO
```

---

## 22.2 context.WithCancel()

### 手动取消

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel() // 确保资源释放

go func() {
    select {
    case <-ctx.Done():
        fmt.Println("收到取消信号:", ctx.Err())
    case <-time.After(time.Second):
        fmt.Println("任务完成")
    }
}()

time.Sleep(100 * time.Millisecond)
cancel() // 主动取消
```

### 关键方法

| 方法 | 作用 |
| --- | --- |
| `ctx.Done()` | 返回只读 channel，关闭时表示取消 |
| `ctx.Err()` | 返回取消原因：`context.Canceled` |
| `cancel()` | 触发取消，关闭 `Done()` channel |

### 注意事项

- **必须调用 `cancel()`**：释放资源，避免 goroutine 泄漏
- **多次调用 `cancel()` 是安全的**：第二次调用无副作用
- **使用 `defer cancel()`**：确保即使函数提前返回也能释放资源

**实测输出**：

```
context.WithCancel()
  goroutine: 收到取消信号 (context canceled)

WithCancel 特点：
• 返回 (ctx, cancel)，调用 cancel() 触发取消
• ctx.Done() 返回只读 channel，关闭时通知取消
• ctx.Err() 返回取消原因：context.Canceled
```

---

## 22.3 context.WithTimeout()

### 超时自动取消

```go
ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
defer cancel()

select {
case <-time.After(100 * time.Millisecond):
    fmt.Println("任务完成")
case <-ctx.Done():
    fmt.Println("超时:", ctx.Err())
}
```

### 与 WithCancel 的区别

- **自动取消**：超时后自动触发 `cancel()`
- **错误类型**：`ctx.Err()` 返回 `context.DeadlineExceeded`
- **仍需 defer cancel()**：提前完成时释放定时器资源

**实测输出**：

```
context.WithTimeout()
  超时：context deadline exceeded

WithTimeout 特点：
• 超时自动触发取消
• ctx.Err() 返回 context.DeadlineExceeded
• 仍需调用 cancel() 释放资源（defer 模式）
```

---

## 22.4 context.WithDeadline()

### 指定截止时间

```go
deadline := time.Now().Add(50 * time.Millisecond)
ctx, cancel := context.WithDeadline(context.Background(), deadline)
defer cancel()

<-ctx.Done()
fmt.Println("到达 deadline:", ctx.Err())
```

### WithDeadline vs WithTimeout

| 方法 | 参数 | 用途 |
| --- | --- | --- |
| `WithDeadline(parent, time.Time)` | 绝对时间点 | 需要在特定时刻取消 |
| `WithTimeout(parent, time.Duration)` | 相对时长 | 从现在起 N 秒后取消 |

**内部实现**：`WithTimeout` 内部调用 `WithDeadline`。

**实测输出**：

```
context.WithDeadline()
  到达 deadline：context deadline exceeded

WithDeadline vs WithTimeout：
• WithDeadline：指定绝对时间点
• WithTimeout：指定相对时长（内部调用 WithDeadline）
```

---

## 22.5 context.WithValue()

### 传递请求范围的数据

```go
type key string
const requestIDKey key = "request-id"

ctx := context.WithValue(context.Background(), requestIDKey, "abc-123")

// 读取值
if reqID, ok := ctx.Value(requestIDKey).(string); ok {
    fmt.Println("Request ID:", reqID)
}
```

### 使用原则

#### ✅ 应该用 WithValue 传递的

- **请求范围的数据**：request ID、trace ID、用户信息
- **跨越多个函数调用栈的数据**：避免每个函数都加参数
- **不影响业务逻辑的元数据**：监控、日志、调试信息

#### ❌ 不应该用 WithValue 传递的

- **函数的可选参数**：用结构体或可变参数
- **业务逻辑必需的数据**：用函数参数
- **配置信息**：用配置文件或全局变量

### key 的最佳实践

```go
// ❌ 错误：使用 string 作为 key（容易冲突）
ctx := context.WithValue(ctx, "user-id", 42)

// ✅ 正确：使用私有类型作为 key
type contextKey string
const userIDKey contextKey = "user-id"
ctx := context.WithValue(ctx, userIDKey, 42)
```

**实测输出**：

```
context.WithValue()
  Request ID: abc-123
  不存在的键返回 nil

WithValue 注意事项：
• 仅用于请求范围的数据（request ID、用户信息、trace ID）
• 不要传递可选参数、函数内部状态
• key 应该是私有类型，避免冲突
• Value 是线程安全的，但不应频繁读取
```

---

## 22.6 取消传播

### 取消树结构

```
parent context (取消)
    ├── child1 context (自动取消)
    └── child2 context (自动取消)
```

### 代码示例

```go
parent, cancel := context.WithCancel(context.Background())
defer cancel()

child1, _ := context.WithCancel(parent)
child2, _ := context.WithCancel(parent)

cancel() // 取消父 context，子 context 自动取消
```

### 传播规则

- **父取消 → 子取消**：父 context 取消时，所有子 context 自动取消
- **子取消 ≠ 父取消**：子 context 取消不影响父 context 和兄弟 context
- **形成树结构**：context 的取消传播形成一棵树

**实测输出**：

```
取消传播
  child1: 收到取消
  child2: 收到取消

取消传播规则：
• 父 context 取消时，所有子 context 自动取消
• 子 context 取消不影响父 context 和兄弟 context
• 形成取消树结构
```

---

## 22.7 context 作为第一个参数

### 函数签名约定

```go
// ✅ 正确：context 作为第一个参数，命名为 ctx
func DoWork(ctx context.Context, taskID string) error {
    select {
    case <-time.After(time.Second):
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

### 约定

- **总是第一个参数**：`func F(ctx context.Context, ...)`
- **命名为 `ctx`**：统一命名，易于识别
- **不要传递 nil**：用 `context.TODO()` 或 `context.Background()`

**实测输出**：

```
context 作为第一个参数
  错误：task task-1 canceled: context deadline exceeded

context 参数约定：
• context 总是第一个参数，命名为 ctx
• func DoWork(ctx context.Context, ...) error
• 不要传递 nil context，用 context.TODO()
```

---

## 22.8 不要把 context 存进结构体

### ❌ 错误做法

```go
type Server struct {
    ctx context.Context // 不要这样做
}

func (s *Server) Handle(req Request) {
    // 使用 s.ctx
}
```

### ✅ 正确做法

```go
type Server struct {
    // 不存储 context
}

func (s *Server) Handle(ctx context.Context, req Request) {
    // ctx 作为参数传入
}
```

### 原因

- **生命周期混乱**：context 的生命周期绑定到请求/操作，存储到结构体会导致生命周期不明确
- **并发安全问题**：多个请求共享同一个 context 可能导致错误
- **例外情况**：当整个结构体的生命周期就是一个请求时可以存储（如 HTTP handler）

**实测输出**：

```
不要把 context 存进结构体

❌ 错误做法：
  type Server struct {
      ctx context.Context  // 不要这样做
  }

✓ 正确做法：
  type Server struct {
      // 不存储 context
  }
  func (s *Server) Handle(ctx context.Context, req Request) {
      // ctx 作为参数传入
  }

原因：
• context 的生命周期绑定到请求/操作
• 存储 context 会导致生命周期混乱
• 例外：当整个结构体的生命周期就是一个请求时可以存储
```

---

## 22.9 Value 的类型化 key

### 避免 key 冲突

```go
// 定义私有类型作为 key
type contextKey string

const (
    userIDKey   contextKey = "user-id"
    requestIDKey contextKey = "request-id"
)

ctx := context.WithValue(context.Background(), userIDKey, 42)
ctx = context.WithValue(ctx, requestIDKey, "xyz-789")

// 读取时类型断言
if userID, ok := ctx.Value(userIDKey).(int); ok {
    fmt.Println("User ID:", userID)
}
```

### 为什么不用 string 或 int

```go
// ❌ 不同包可能使用相同的 string key，导致冲突
// package A
ctx = context.WithValue(ctx, "user-id", 123)

// package B
ctx = context.WithValue(ctx, "user-id", "bob") // 覆盖了 package A 的值
```

### 类型化 key 的好处

- **避免冲突**：私有类型只有本包能访问
- **类型安全**：编译时检查
- **文档清晰**：通过类型名就能知道用途

**实测输出**：

```
Value 的类型化 key
  User ID: 42
  Request ID: xyz-789

类型化 key 的好处：
• 避免不同包之间的 key 冲突
• 类型安全，编译时检查
• 不要用 string 或 int 作为 key
```

---

## 22.10 超时链路

### 父子 context 的 deadline 关系

```go
// 父 context：5 秒超时
parent, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// 子 context：100ms 超时
child, childCancel := context.WithTimeout(parent, 100*time.Millisecond)
defer childCancel()

<-child.Done() // 子 context 先超时
// 父 context 仍然有效
```

### deadline 继承规则

- **子 deadline ≤ 父 deadline**：子 context 的 deadline 不能晚于父 context
- **自动调整**：如果子设置的 deadline 晚于父，自动使用父 deadline
- **独立超时**：子超时不影响父 context

**实测输出**：

```
超时链路
  子任务超时：context deadline exceeded
  ✓ 父 context 仍然有效

超时链路规则：
• 子 context 的 deadline 不能晚于父 context
• 如果子 deadline 晚于父，自动使用父 deadline
• 子超时不影响父 context
```

---

## 5 个真实报错怎么读

### 报错 1：context 已取消

```
context canceled
```

**原因**：context 被 `cancel()` 调用取消。

**修复**：检查 `ctx.Done()` 并正确处理取消信号。

### 报错 2：context 超时

```
context deadline exceeded
```

**原因**：context 超时或到达 deadline。

**修复**：增加超时时间，或优化任务执行速度。

### 报错 3：传递了 nil context

```
panic: runtime error: invalid memory address or nil pointer dereference
```

**原因**：传递了 `nil` 作为 context。

**修复**：使用 `context.Background()` 或 `context.TODO()`。

### 报错 4：context.Value 返回 nil

```
// 没有报错，但 Value 返回 nil
val := ctx.Value(key)
if val == nil {
    // key 不存在
}
```

**原因**：key 不存在，或父 context 中没有设置该 key。

**修复**：检查 key 是否正确，确保在正确的 context 中设置值。

### 报错 5：goroutine 泄漏

```
// runtime.NumGoroutine() 持续增长
```

**原因**：没有调用 `cancel()`，goroutine 永远等待 `ctx.Done()`。

**修复**：使用 `defer cancel()` 确保资源释放。

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| goroutine 泄漏 | 忘记调用 `cancel()` | 使用 `defer cancel()` |
| context 超时不生效 | 没有检查 `ctx.Done()` | 在 select 中监听 `ctx.Done()` |
| Value 返回 nil | key 类型不匹配或不存在 | 使用类型化 key，检查类型断言 |
| 取消不传播 | 使用了错误的父 context | 确保子 context 从正确的父创建 |
| 传递 nil context | 参数为 nil | 用 `context.Background()` 或 `context.TODO()` |
| context 存储到结构体 | 生命周期混乱 | context 作为参数传递 |
| 子 deadline 晚于父 | 设置了过长的超时 | 子 deadline 会自动调整为父 deadline |
| 多次调用 cancel panic | 错误理解 cancel 行为 | 多次调用 cancel 是安全的 |

---

## 练习

### 第 1 题

实现一个带超时的 HTTP 请求函数，超时时取消请求。

::: details 第 1 题参考答案

```go
func FetchWithTimeout(url string, timeout time.Duration) ([]byte, error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    return io.ReadAll(resp.Body)
}

// 使用
data, err := FetchWithTimeout("https://example.com", 5*time.Second)
if errors.Is(err, context.DeadlineExceeded) {
    fmt.Println("请求超时")
}
```

**为什么这样写更好**：

- 使用 `http.NewRequestWithContext` 传递 context
- 超时自动取消请求
- `defer cancel()` 确保资源释放

:::

### 第 2 题

实现一个支持取消的 Pipeline，父 context 取消时所有阶段都停止。

::: details 第 2 题参考答案

```go
func Pipeline(ctx context.Context, nums []int) <-chan int {
    gen := func(ctx context.Context, nums []int) <-chan int {
        out := make(chan int)
        go func() {
            defer close(out)
            for _, n := range nums {
                select {
                case <-ctx.Done():
                    return
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
                    return
                case out <- n * n:
                }
            }
        }()
        return out
    }

    c := gen(ctx, nums)
    return sq(ctx, c)
}

// 使用
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

out := Pipeline(ctx, []int{1, 2, 3, 4, 5})
cancel() // 取消整个 pipeline
```

**为什么这样写更好**：

- 每个阶段都检查 `ctx.Done()`
- 取消信号传播到所有阶段
- 避免 goroutine 泄漏

:::

### 第 3 题

使用 context.WithValue 在 HTTP 中间件中传递 request ID。

::: details 第 3 题参考答案

```go
type contextKey string

const requestIDKey contextKey = "request-id"

// 中间件：生成 request ID 并存入 context
func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := generateRequestID() // 生成唯一 ID
        ctx := context.WithValue(r.Context(), requestIDKey, requestID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Handler：读取 request ID
func Handler(w http.ResponseWriter, r *http.Request) {
    requestID, ok := r.Context().Value(requestIDKey).(string)
    if !ok {
        requestID = "unknown"
    }
    fmt.Fprintf(w, "Request ID: %s", requestID)
}

func generateRequestID() string {
    return fmt.Sprintf("%d", time.Now().UnixNano())
}
```

**为什么这样写更好**：

- 使用私有类型作为 key，避免冲突
- request ID 跨越多个 handler
- 不需要每个函数都传递 request ID 参数

:::

### 第 4 题

解释为什么不应该把 context 存储到结构体中，并举例说明。

::: details 第 4 题参考答案

**原因**：

1. **生命周期混乱**：context 的生命周期绑定到单个请求/操作，存储到结构体会导致多个请求共享同一个 context
2. **并发安全问题**：多个 goroutine 访问同一个 context 可能导致错误
3. **违反设计原则**：context 应该在调用链中显式传递

**错误示例**：

```go
type Server struct {
    ctx context.Context // ❌ 不要这样做
}

func NewServer() *Server {
    ctx, cancel := context.WithCancel(context.Background())
    // cancel 什么时候调用？
    return &Server{ctx: ctx}
}

func (s *Server) HandleRequest1() {
    // 使用 s.ctx
}

func (s *Server) HandleRequest2() {
    // 也使用 s.ctx，但这是不同的请求
}
```

**问题**：

- 两个请求共享同一个 context
- cancel 一个会影响另一个
- 生命周期不清晰

**正确做法**：

```go
type Server struct {
    // 不存储 context
}

func (s *Server) HandleRequest(ctx context.Context, req Request) {
    // ctx 作为参数传入，每个请求独立
}
```

**例外情况**：

当整个结构体的生命周期就是一个请求时可以存储：

```go
type RequestHandler struct {
    ctx context.Context // 可以，因为整个结构体就是为一个请求服务
    req *http.Request
}

func NewRequestHandler(ctx context.Context, req *http.Request) *RequestHandler {
    return &RequestHandler{ctx: ctx, req: req}
}
```

:::

### 第 5 题

实现一个并发批量处理函数，使用 context 控制超时和取消。

::: details 第 5 题参考答案

```go
func ProcessBatch(ctx context.Context, items []int) ([]int, error) {
    results := make([]int, len(items))
    errCh := make(chan error, len(items))
    var wg sync.WaitGroup

    for i, item := range items {
        wg.Add(1)
        go func(index, value int) {
            defer wg.Done()

            select {
            case <-ctx.Done():
                errCh <- ctx.Err()
                return
            default:
            }

            // 模拟处理
            time.Sleep(10 * time.Millisecond)
            results[index] = value * 2
        }(i, item)
    }

    // 等待所有 goroutine 完成或 context 取消
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        close(errCh)
        if len(errCh) > 0 {
            return nil, <-errCh
        }
        return results, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}

// 使用
ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()

results, err := ProcessBatch(ctx, []int{1, 2, 3, 4, 5})
if errors.Is(err, context.DeadlineExceeded) {
    fmt.Println("批处理超时")
}
```

**为什么这样写更好**：

- 每个 goroutine 检查 context 取消
- 超时时所有 goroutine 收到取消信号
- 使用 WaitGroup 等待所有 goroutine 完成

:::

---

## 小结

- **Background() 与 TODO()**：顶层 context，永不取消
- **WithCancel()**：手动取消，返回 cancel 函数
- **WithTimeout() / WithDeadline()**：超时自动取消
- **WithValue()**：传递请求范围数据，使用类型化 key
- **取消传播**：父取消时子自动取消，形成树结构
- **context 作为第一个参数**：`func F(ctx context.Context, ...)`
- **不要存储到结构体**：context 生命周期绑定到请求
- **类型化 key**：避免冲突，类型安全
- **超时链路**：子 deadline 不能晚于父 deadline
- **defer cancel()**：确保资源释放，避免泄漏

下一章将讲解 **运行时、调度与内存模型**，深入理解 Go 的 GMP 调度模型和内存管理。
