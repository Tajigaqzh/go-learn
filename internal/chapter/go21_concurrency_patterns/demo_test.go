package go21_concurrency_patterns

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestWorkerPool 测试 Worker Pool 模式
func TestWorkerPool(t *testing.T) {
	jobs := make(chan int, 10)
	results := make(chan int, 10)
	var wg sync.WaitGroup

	// 启动 3 个 worker
	for w := 1; w <= 3; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				results <- j * 2
			}
		}()
	}

	// 发送 5 个任务
	for j := 1; j <= 5; j++ {
		jobs <- j
	}
	close(jobs)

	wg.Wait()
	close(results)

	// 验证结果数量
	count := 0
	for range results {
		count++
	}
	if count != 5 {
		t.Errorf("expected 5 results, got %d", count)
	}
}

// TestProducerConsumer 测试生产者消费者模式
func TestProducerConsumer(t *testing.T) {
	ch := make(chan int, 5)
	var wg sync.WaitGroup
	var mu sync.Mutex
	consumed := []int{}

	// 生产者
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(ch)
		for i := 1; i <= 3; i++ {
			ch <- i
		}
	}()

	// 消费者
	wg.Add(1)
	go func() {
		defer wg.Done()
		for v := range ch {
			mu.Lock()
			consumed = append(consumed, v)
			mu.Unlock()
		}
	}()

	wg.Wait()

	if len(consumed) != 3 {
		t.Errorf("expected 3 items consumed, got %d", len(consumed))
	}
}

// TestFanIn 测试 Fan-In 模式
func TestFanIn(t *testing.T) {
	c1 := make(chan int)
	c2 := make(chan int)

	go func() {
		c1 <- 1
		c1 <- 2
		close(c1)
	}()

	go func() {
		c2 <- 3
		c2 <- 4
		close(c2)
	}()

	out := fanIn(c1, c2)
	count := 0
	for range out {
		count++
	}

	if count != 4 {
		t.Errorf("expected 4 values from fan-in, got %d", count)
	}
}

// TestPipeline 测试 Pipeline 模式
func TestPipeline(t *testing.T) {
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

	c := gen(2, 3)
	out := sq(c)

	results := []int{}
	for n := range out {
		results = append(results, n)
	}

	expected := []int{4, 9}
	if len(results) != len(expected) {
		t.Fatalf("expected %d results, got %d", len(expected), len(results))
	}
	for i, v := range expected {
		if results[i] != v {
			t.Errorf("index %d: expected %d, got %d", i, v, results[i])
		}
	}
}

// TestContextTimeout 测试 context 超时
func TestContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	result := make(chan string, 1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		result <- "done"
	}()

	select {
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("expected DeadlineExceeded, got %v", ctx.Err())
		}
	case <-result:
		t.Error("should have timed out")
	}
}

// TestContextCancellation 测试 context 取消传播
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bool)

	go func() {
		select {
		case <-ctx.Done():
			done <- true
		case <-time.After(100 * time.Millisecond):
			done <- false
		}
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	if cancelled := <-done; !cancelled {
		t.Error("context should have been cancelled")
	}
}

// TestChannelClose 测试 channel 关闭检测
func TestChannelClose(t *testing.T) {
	ch := make(chan int, 2)
	ch <- 1
	ch <- 2
	close(ch)

	// 读取直到 channel 关闭
	count := 0
	for v := range ch {
		count++
		if v == 0 {
			t.Error("should not receive zero value before close")
		}
	}

	if count != 2 {
		t.Errorf("expected 2 values, got %d", count)
	}

	// 从已关闭的 channel 读取返回零值和 false
	v, ok := <-ch
	if ok {
		t.Error("ok should be false for closed channel")
	}
	if v != 0 {
		t.Errorf("expected zero value, got %d", v)
	}
}

// TestGoroutineLeakPrevention 测试 goroutine 泄漏预防
func TestGoroutineLeakPrevention(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	done := make(chan bool)
	go func() {
		select {
		case <-ctx.Done():
			done <- true
		case <-time.After(100 * time.Millisecond):
			done <- false
		}
	}()

	// 等待 goroutine 响应取消
	select {
	case leaked := <-done:
		if leaked {
			// goroutine 正确响应了 context 取消
		} else {
			t.Error("goroutine should have been cancelled by context")
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("goroutine did not respond to cancellation")
	}
}

// TestSyncOnce 测试 sync.Once 确保只执行一次
func TestSyncOnce(t *testing.T) {
	var once sync.Once
	count := 0
	var mu sync.Mutex

	increment := func() {
		mu.Lock()
		count++
		mu.Unlock()
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			once.Do(increment)
		}()
	}

	wg.Wait()

	if count != 1 {
		t.Errorf("expected count=1, got %d", count)
	}
}

// TestRaceCondition 测试数据竞争检测（用 go test -race 运行）
func TestRaceCondition(t *testing.T) {
	var counter int
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			counter++
			mu.Unlock()
		}()
	}

	wg.Wait()

	if counter != 10 {
		t.Errorf("expected counter=10, got %d", counter)
	}
}

// TestBufferedChannel 测试有缓冲 channel 的非阻塞发送
func TestBufferedChannel(t *testing.T) {
	ch := make(chan int, 2)

	// 发送到有缓冲的 channel 不会阻塞
	ch <- 1
	ch <- 2

	// 第三次发送会阻塞，除非有接收者
	select {
	case ch <- 3:
		t.Error("should not be able to send to full channel without blocking")
	default:
		// 正确：channel 已满
	}

	// 读取两个值
	if v := <-ch; v != 1 {
		t.Errorf("expected 1, got %d", v)
	}
	if v := <-ch; v != 2 {
		t.Errorf("expected 2, got %d", v)
	}
}

// TestSelectDefault 测试 select 的 default 分支
func TestSelectDefault(t *testing.T) {
	ch := make(chan int)

	select {
	case v := <-ch:
		t.Errorf("should not receive from empty channel, got %d", v)
	default:
		// 正确：没有值可读，执行 default
	}
}

// TestSelectTimeout 测试 select 超时模式
func TestSelectTimeout(t *testing.T) {
	ch := make(chan int)

	select {
	case <-ch:
		t.Error("should not receive from empty channel")
	case <-time.After(10 * time.Millisecond):
		// 正确：超时
	}
}

// BenchmarkChannelSend 基准测试：channel 发送接收性能
func BenchmarkChannelSend(b *testing.B) {
	ch := make(chan int, 1)
	go func() {
		for range ch {
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch <- i
	}
	close(ch)
}

// BenchmarkMutex 基准测试：mutex 加锁解锁性能
func BenchmarkMutex(b *testing.B) {
	var mu sync.Mutex
	var counter int

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		counter++
		mu.Unlock()
	}
}
