// Package go22_context 演示 context 包的使用与生命周期管理。
package go22_context

import (
	"context"
	"fmt"
	"time"
)

// Demo 运行第 22 章所有演示。
func Demo() {
	fmt.Println("\n========== go22_context: context 与生命周期管理 ==========")
	demoBackgroundAndTODO()
	demoWithCancel()
	demoWithTimeout()
	demoWithDeadline()
	demoWithValue()
	demoCancellationPropagation()
	demoContextAsParameter()
	demoContextDontStore()
	demoValueTypedKey()
	demoTimeoutChain()
	fmt.Println("\n========== context 与生命周期管理演示结束 ==========")
}

// --- 1. Background 与 TODO ---

func demoBackgroundAndTODO() {
	fmt.Println("\n--- 1. context.Background() 与 context.TODO() ---")

	ctx1 := context.Background()
	fmt.Printf("Background context: %v\n", ctx1)

	ctx2 := context.TODO()
	fmt.Printf("TODO context: %v\n", ctx2)

	fmt.Println("\n使用场景：")
	fmt.Println("• Background：顶层 context，main 函数、初始化、测试")
	fmt.Println("• TODO：占位符，不确定用哪个 context 时先用 TODO")
}

// --- 2. WithCancel ---

func demoWithCancel() {
	fmt.Println("\n--- 2. context.WithCancel() ---")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 确保资源释放

	go func() {
		select {
		case <-time.After(100 * time.Millisecond):
			fmt.Println("  goroutine: 任务完成")
		case <-ctx.Done():
			fmt.Printf("  goroutine: 收到取消信号 (%v)\n", ctx.Err())
		}
	}()

	time.Sleep(20 * time.Millisecond)
	cancel() // 主动取消
	time.Sleep(20 * time.Millisecond)

	fmt.Println("\nWithCancel 特点：")
	fmt.Println("• 返回 (ctx, cancel)，调用 cancel() 触发取消")
	fmt.Println("• ctx.Done() 返回只读 channel，关闭时通知取消")
	fmt.Println("• ctx.Err() 返回取消原因：context.Canceled")
}

// --- 3. WithTimeout ---

func demoWithTimeout() {
	fmt.Println("\n--- 3. context.WithTimeout() ---")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	select {
	case <-time.After(100 * time.Millisecond):
		fmt.Println("  任务完成")
	case <-ctx.Done():
		fmt.Printf("  超时：%v\n", ctx.Err())
	}

	fmt.Println("\nWithTimeout 特点：")
	fmt.Println("• 超时自动触发取消")
	fmt.Println("• ctx.Err() 返回 context.DeadlineExceeded")
	fmt.Println("• 仍需调用 cancel() 释放资源（defer 模式）")
}

// --- 4. WithDeadline ---

func demoWithDeadline() {
	fmt.Println("\n--- 4. context.WithDeadline() ---")

	deadline := time.Now().Add(50 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	select {
	case <-time.After(100 * time.Millisecond):
		fmt.Println("  任务完成")
	case <-ctx.Done():
		fmt.Printf("  到达 deadline：%v\n", ctx.Err())
	}

	fmt.Println("\nWithDeadline vs WithTimeout：")
	fmt.Println("• WithDeadline：指定绝对时间点")
	fmt.Println("• WithTimeout：指定相对时长（内部调用 WithDeadline）")
}

// --- 5. WithValue ---

func demoWithValue() {
	fmt.Println("\n--- 5. context.WithValue() ---")

	type key string
	const requestIDKey key = "request-id"

	ctx := context.WithValue(context.Background(), requestIDKey, "abc-123")

	// 读取值
	if reqID, ok := ctx.Value(requestIDKey).(string); ok {
		fmt.Printf("  Request ID: %s\n", reqID)
	}

	// 读取不存在的键
	if val := ctx.Value(key("not-exist")); val == nil {
		fmt.Println("  不存在的键返回 nil")
	}

	fmt.Println("\nWithValue 注意事项：")
	fmt.Println("• 仅用于请求范围的数据（request ID、用户信息、trace ID）")
	fmt.Println("• 不要传递可选参数、函数内部状态")
	fmt.Println("• key 应该是私有类型，避免冲突")
	fmt.Println("• Value 是线程安全的，但不应频繁读取")
}

// --- 6. 取消传播 ---

func demoCancellationPropagation() {
	fmt.Println("\n--- 6. 取消传播 ---")

	parent, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 子 context
	child1, cancel1 := context.WithCancel(parent)
	defer cancel1()
	child2, cancel2 := context.WithCancel(parent)
	defer cancel2()

	go func() {
		<-child1.Done()
		fmt.Println("  child1: 收到取消")
	}()

	go func() {
		<-child2.Done()
		fmt.Println("  child2: 收到取消")
	}()

	time.Sleep(20 * time.Millisecond)
	cancel() // 取消父 context
	time.Sleep(20 * time.Millisecond)

	fmt.Println("\n取消传播规则：")
	fmt.Println("• 父 context 取消时，所有子 context 自动取消")
	fmt.Println("• 子 context 取消不影响父 context 和兄弟 context")
	fmt.Println("• 形成取消树结构")
}

// --- 7. context 作为第一个参数 ---

func demoContextAsParameter() {
	fmt.Println("\n--- 7. context 作为第一个参数 ---")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := doWork(ctx, "task-1")
	if err != nil {
		fmt.Printf("  错误：%v\n", err)
	} else {
		fmt.Printf("  结果：%s\n", result)
	}

	fmt.Println("\ncontext 参数约定：")
	fmt.Println("• context 总是第一个参数，命名为 ctx")
	fmt.Println("• func DoWork(ctx context.Context, ...) error")
	fmt.Println("• 不要传递 nil context，用 context.TODO()")
}

func doWork(ctx context.Context, taskID string) (string, error) {
	select {
	case <-time.After(100 * time.Millisecond):
		return fmt.Sprintf("task %s done", taskID), nil
	case <-ctx.Done():
		return "", fmt.Errorf("task %s canceled: %w", taskID, ctx.Err())
	}
}

// --- 8. 不要把 context 存进结构体 ---

func demoContextDontStore() {
	fmt.Println("\n--- 8. 不要把 context 存进结构体 ---")

	fmt.Println("\n❌ 错误做法：")
	fmt.Println("  type Server struct {")
	fmt.Println("      ctx context.Context  // 不要这样做")
	fmt.Println("  }")

	fmt.Println("\n✓ 正确做法：")
	fmt.Println("  type Server struct {")
	fmt.Println("      // 不存储 context")
	fmt.Println("  }")
	fmt.Println("  func (s *Server) Handle(ctx context.Context, req Request) {")
	fmt.Println("      // ctx 作为参数传入")
	fmt.Println("  }")

	fmt.Println("\n原因：")
	fmt.Println("• context 的生命周期绑定到请求/操作")
	fmt.Println("• 存储 context 会导致生命周期混乱")
	fmt.Println("• 例外：当整个结构体的生命周期就是一个请求时可以存储")
}

// --- 9. Value 的类型化 key ---

func demoValueTypedKey() {
	fmt.Println("\n--- 9. Value 的类型化 key ---")

	// 定义私有类型作为 key
	type contextKey string

	const (
		userIDKey    contextKey = "user-id"
		requestIDKey contextKey = "request-id"
	)

	ctx := context.WithValue(context.Background(), userIDKey, 42)
	ctx = context.WithValue(ctx, requestIDKey, "xyz-789")

	// 读取时类型断言
	if userID, ok := ctx.Value(userIDKey).(int); ok {
		fmt.Printf("  User ID: %d\n", userID)
	}

	if reqID, ok := ctx.Value(requestIDKey).(string); ok {
		fmt.Printf("  Request ID: %s\n", reqID)
	}

	fmt.Println("\n类型化 key 的好处：")
	fmt.Println("• 避免不同包之间的 key 冲突")
	fmt.Println("• 类型安全，编译时检查")
	fmt.Println("• 不要用 string 或 int 作为 key")
}

// --- 10. 超时链路 ---

func demoTimeoutChain() {
	fmt.Println("\n--- 10. 超时链路 ---")

	// 顶层 5 秒超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 子任务 100ms 超时
	childCtx, childCancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer childCancel()

	// 模拟子任务
	select {
	case <-time.After(200 * time.Millisecond):
		fmt.Println("  子任务完成")
	case <-childCtx.Done():
		fmt.Printf("  子任务超时：%v\n", childCtx.Err())
	}

	// 父 context 仍然有效
	select {
	case <-ctx.Done():
		fmt.Println("  父 context 已取消")
	default:
		fmt.Println("  ✓ 父 context 仍然有效")
	}

	fmt.Println("\n超时链路规则：")
	fmt.Println("• 子 context 的 deadline 不能晚于父 context")
	fmt.Println("• 如果子 deadline 晚于父，自动使用父 deadline")
	fmt.Println("• 子超时不影响父 context")
}
