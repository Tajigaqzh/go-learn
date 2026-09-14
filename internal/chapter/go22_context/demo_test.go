package go22_context

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestContextCancel 测试 WithCancel
func TestContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		<-ctx.Done()
		done <- true
	}()

	cancel()

	select {
	case <-done:
		// 成功收到取消信号
	case <-time.After(100 * time.Millisecond):
		t.Fatal("context cancellation not received")
	}

	if ctx.Err() != context.Canceled {
		t.Errorf("expected Canceled, got %v", ctx.Err())
	}
}

// TestContextTimeout 测试 WithTimeout
func TestContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	select {
	case <-time.After(50 * time.Millisecond):
		t.Fatal("should have timed out")
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("expected DeadlineExceeded, got %v", ctx.Err())
		}
	}
}

// TestContextDeadline 测试 WithDeadline
func TestContextDeadline(t *testing.T) {
	deadline := time.Now().Add(10 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	<-ctx.Done()

	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", ctx.Err())
	}
}

// TestContextValue 测试 WithValue
func TestContextValue(t *testing.T) {
	type key string
	const userIDKey key = "user-id"

	ctx := context.WithValue(context.Background(), userIDKey, 42)

	if val, ok := ctx.Value(userIDKey).(int); !ok || val != 42 {
		t.Errorf("expected 42, got %v", val)
	}

	// 不存在的键返回 nil
	if val := ctx.Value(key("not-exist")); val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}

// TestCancellationPropagation 测试取消传播
func TestCancellationPropagation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()

	child1, cancel1 := context.WithCancel(parent)
	defer cancel1()
	child2, cancel2 := context.WithCancel(parent)
	defer cancel2()

	cancel() // 取消父 context

	// 验证子 context 都被取消
	select {
	case <-child1.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("child1 should be canceled")
	}

	select {
	case <-child2.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("child2 should be canceled")
	}
}

// TestChildCancelDoesNotAffectParent 测试子取消不影响父
func TestChildCancelDoesNotAffectParent(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()

	child, childCancel := context.WithCancel(parent)

	childCancel() // 只取消子 context

	// 验证父 context 仍然有效
	select {
	case <-parent.Done():
		t.Fatal("parent should not be canceled")
	default:
		// 正确：父 context 未被取消
	}

	// 验证子 context 已取消
	select {
	case <-child.Done():
		// 正确：子 context 已取消
	default:
		t.Fatal("child should be canceled")
	}
}

// TestContextTimeoutChain 测试超时链
func TestContextTimeoutChain(t *testing.T) {
	parent, parentCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer parentCancel()

	child, childCancel := context.WithTimeout(parent, 10*time.Millisecond)
	defer childCancel()

	// 子 context 应该先超时
	<-child.Done()

	if child.Err() != context.DeadlineExceeded {
		t.Errorf("child expected DeadlineExceeded, got %v", child.Err())
	}

	// 父 context 仍然有效
	select {
	case <-parent.Done():
		t.Fatal("parent should not be canceled yet")
	default:
		// 正确
	}
}

// TestContextWithWork 测试 context 与实际工作函数
func TestContextWithWork(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := doWork(ctx, "test-task")

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

// TestContextValueTypedKey 测试类型化 key
func TestContextValueTypedKey(t *testing.T) {
	type contextKey string

	const (
		key1 contextKey = "key1"
		key2 contextKey = "key2"
	)

	ctx := context.WithValue(context.Background(), key1, "value1")
	ctx = context.WithValue(ctx, key2, 42)

	if val, ok := ctx.Value(key1).(string); !ok || val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}

	if val, ok := ctx.Value(key2).(int); !ok || val != 42 {
		t.Errorf("expected 42, got %v", val)
	}
}

// TestMultipleCancel 测试多次调用 cancel 是安全的
func TestMultipleCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cancel()
	cancel() // 第二次调用应该安全
	cancel() // 第三次调用应该安全

	if ctx.Err() != context.Canceled {
		t.Errorf("expected Canceled, got %v", ctx.Err())
	}
}

// TestBackgroundAndTODO 测试 Background 和 TODO
func TestBackgroundAndTODO(t *testing.T) {
	ctx1 := context.Background()
	ctx2 := context.TODO()

	// 两者都不应该为 nil
	if ctx1 == nil {
		t.Error("Background should not be nil")
	}
	if ctx2 == nil {
		t.Error("TODO should not be nil")
	}

	// 两者都不会取消
	select {
	case <-ctx1.Done():
		t.Error("Background should never be canceled")
	default:
	}

	select {
	case <-ctx2.Done():
		t.Error("TODO should never be canceled")
	default:
	}
}

// TestContextDeadlineInheritance 测试 deadline 继承
func TestContextDeadlineInheritance(t *testing.T) {
	// 父 context：100ms 超时
	parent, parentCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer parentCancel()

	// 子 context：尝试设置 200ms 超时（应该被父限制）
	child, childCancel := context.WithTimeout(parent, 200*time.Millisecond)
	defer childCancel()

	// 获取 deadline
	parentDeadline, parentOk := parent.Deadline()
	childDeadline, childOk := child.Deadline()

	if !parentOk || !childOk {
		t.Fatal("both should have deadlines")
	}

	// 子 deadline 不应该晚于父 deadline
	if childDeadline.After(parentDeadline) {
		t.Error("child deadline should not be after parent deadline")
	}
}

// BenchmarkContextCreation 基准测试：context 创建性能
func BenchmarkContextCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_ = ctx
	}
}

// BenchmarkContextValue 基准测试：context.Value 性能
func BenchmarkContextValue(b *testing.B) {
	type key string
	const testKey key = "test"

	ctx := context.WithValue(context.Background(), testKey, "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ctx.Value(testKey)
	}
}

// BenchmarkContextTimeout 基准测试：WithTimeout 性能
func BenchmarkContextTimeout(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		cancel()
		_ = ctx
	}
}
