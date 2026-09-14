package go20_concurrency

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestGoroutineExecution 验证 goroutine 能正常执行
func TestGoroutineExecution(t *testing.T) {
	done := make(chan bool)
	go func() {
		done <- true
	}()

	select {
	case <-done:
		// 成功接收
	case <-time.After(100 * time.Millisecond):
		t.Fatal("goroutine 没有执行")
	}
}

// TestChannelUnbuffered 验证无缓冲 channel 的同步语义
func TestChannelUnbuffered(t *testing.T) {
	ch := make(chan int)

	go func() {
		ch <- 42
	}()

	val := <-ch
	if val != 42 {
		t.Errorf("期望 42, 得到 %d", val)
	}
}

// TestChannelBuffered 验证有缓冲 channel
func TestChannelBuffered(t *testing.T) {
	ch := make(chan int, 2)

	// 缓冲区未满，不阻塞
	ch <- 1
	ch <- 2

	if len(ch) != 2 {
		t.Errorf("期望长度 2, 得到 %d", len(ch))
	}

	if cap(ch) != 2 {
		t.Errorf("期望容量 2, 得到 %d", cap(ch))
	}

	// 读取
	if v := <-ch; v != 1 {
		t.Errorf("期望 1, 得到 %d", v)
	}
	if v := <-ch; v != 2 {
		t.Errorf("期望 2, 得到 %d", v)
	}
}

// TestChannelClose 验证 channel 关闭行为
func TestChannelClose(t *testing.T) {
	ch := make(chan int, 2)
	ch <- 1
	ch <- 2
	close(ch)

	// 从已关闭 channel 接收
	if v := <-ch; v != 1 {
		t.Errorf("期望 1, 得到 %d", v)
	}
	if v := <-ch; v != 2 {
		t.Errorf("期望 2, 得到 %d", v)
	}

	// 接收零值和关闭标志
	v, ok := <-ch
	if v != 0 || ok != false {
		t.Errorf("期望 (0, false), 得到 (%d, %t)", v, ok)
	}
}

// TestChannelRange 验证 range 遍历 channel
func TestChannelRange(t *testing.T) {
	ch := make(chan int, 3)
	go func() {
		for i := range 3 {
			ch <- i
		}
		close(ch)
	}()

	var sum int
	for v := range ch {
		sum += v
	}

	if sum != 0+1+2 {
		t.Errorf("期望 3, 得到 %d", sum)
	}
}

// TestSelect 验证 select 多路复用
func TestSelect(t *testing.T) {
	ch1 := make(chan string, 1)
	ch2 := make(chan string, 1)

	ch1 <- "a"
	ch2 <- "b"

	received := 0
	for range 2 {
		select {
		case <-ch1:
			received++
		case <-ch2:
			received++
		}
	}

	if received != 2 {
		t.Errorf("期望接收 2 次, 得到 %d", received)
	}
}

// TestSelectTimeout 验证 select 超时
func TestSelectTimeout(t *testing.T) {
	ch := make(chan int)

	select {
	case <-ch:
		t.Fatal("不应该接收到数据")
	case <-time.After(10 * time.Millisecond):
		// 超时，测试通过
	}
}

// TestSelectDefault 验证 select 非阻塞
func TestSelectDefault(t *testing.T) {
	ch := make(chan int)

	select {
	case <-ch:
		t.Fatal("不应该接收到数据")
	default:
		// 立即返回，测试通过
	}
}

// TestWaitGroup 验证 WaitGroup 同步
func TestWaitGroup(t *testing.T) {
	var wg sync.WaitGroup
	var counter atomic.Int32

	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Add(1)
		}()
	}

	wg.Wait()

	if counter.Load() != 10 {
		t.Errorf("期望 10, 得到 %d", counter.Load())
	}
}

// TestMutex 验证 Mutex 互斥
func TestMutex(t *testing.T) {
	var (
		mu      sync.Mutex
		counter int
		wg      sync.WaitGroup
	)

	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			counter++
			mu.Unlock()
		}()
	}

	wg.Wait()

	if counter != 100 {
		t.Errorf("期望 100, 得到 %d", counter)
	}
}

// TestRWMutex 验证 RWMutex 读写锁
func TestRWMutex(t *testing.T) {
	var (
		mu   sync.RWMutex
		data int
		wg   sync.WaitGroup
	)

	// 10 个读者
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.RLock()
			_ = data
			mu.RUnlock()
		}()
	}

	// 1 个写者
	wg.Add(1)
	go func() {
		defer wg.Done()
		mu.Lock()
		data = 42
		mu.Unlock()
	}()

	wg.Wait()

	mu.RLock()
	if data != 42 {
		t.Errorf("期望 42, 得到 %d", data)
	}
	mu.RUnlock()
}

// TestOnce 验证 Once 只执行一次
func TestOnce(t *testing.T) {
	var (
		once    sync.Once
		counter atomic.Int32
		wg      sync.WaitGroup
	)

	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			once.Do(func() {
				counter.Add(1)
			})
		}()
	}

	wg.Wait()

	if counter.Load() != 1 {
		t.Errorf("期望 1, 得到 %d", counter.Load())
	}
}

// TestAtomic 验证原子操作
func TestAtomic(t *testing.T) {
	var (
		counter atomic.Int64
		wg      sync.WaitGroup
	)

	for range 1000 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Add(1)
		}()
	}

	wg.Wait()

	if counter.Load() != 1000 {
		t.Errorf("期望 1000, 得到 %d", counter.Load())
	}
}

// TestAtomicCAS 验证 CompareAndSwap
func TestAtomicCAS(t *testing.T) {
	var val atomic.Int32
	val.Store(10)

	// 成功的 CAS
	if !val.CompareAndSwap(10, 20) {
		t.Error("CAS 应该成功")
	}
	if val.Load() != 20 {
		t.Errorf("期望 20, 得到 %d", val.Load())
	}

	// 失败的 CAS
	if val.CompareAndSwap(10, 30) {
		t.Error("CAS 应该失败")
	}
	if val.Load() != 20 {
		t.Errorf("期望 20, 得到 %d", val.Load())
	}
}

// TestChannelAsSignal 验证 channel 作为信号
func TestChannelAsSignal(t *testing.T) {
	done := make(chan struct{})

	go func() {
		time.Sleep(10 * time.Millisecond)
		close(done) // 信号：工作完成
	}()

	<-done // 等待信号
}

// TestWorkerPool 验证简单的 worker pool
func TestWorkerPool(t *testing.T) {
	jobs := make(chan int, 10)
	results := make(chan int, 10)

	// 启动 3 个 worker
	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				results <- job * 2
			}
		}()
	}

	// 发送 5 个任务
	for i := range 5 {
		jobs <- i
	}
	close(jobs)

	// 等待 worker 完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	var sum int
	for result := range results {
		sum += result
	}

	// 0*2 + 1*2 + 2*2 + 3*2 + 4*2 = 20
	if sum != 20 {
		t.Errorf("期望 20, 得到 %d", sum)
	}
}
