package go23_runtime

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestGOMMAXPROCS 测试 GOMAXPROCS 查询和设置
func TestGOMMAXPROCS(t *testing.T) {
	// 保存原始值
	original := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(original)

	// 测试设置新值
	newValue := 2
	old := runtime.GOMAXPROCS(newValue)
	if old != original {
		t.Errorf("GOMAXPROCS(2) 返回值应该是 %d，实际是 %d", original, old)
	}

	// 测试查询当前值
	current := runtime.GOMAXPROCS(0)
	if current != newValue {
		t.Errorf("GOMAXPROCS(0) 应该返回 %d，实际是 %d", newValue, current)
	}
}

// TestNumGoroutine 测试 goroutine 数量统计
func TestNumGoroutine(t *testing.T) {
	before := runtime.NumGoroutine()

	var wg sync.WaitGroup
	count := 10
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
		}()
	}

	// 短暂等待让 goroutine 启动
	time.Sleep(5 * time.Millisecond)
	during := runtime.NumGoroutine()

	wg.Wait()
	after := runtime.NumGoroutine()

	// 运行期间应该有更多 goroutine
	if during <= before {
		t.Errorf("期望运行期间 goroutine 数量 (%d) > 启动前 (%d)", during, before)
	}

	// 完成后应该接近初始数量（允许少量误差，因为可能有其他系统 goroutine）
	if after > before+2 {
		t.Errorf("期望完成后 goroutine 数量 (%d) 接近启动前 (%d)", after, before)
	}
}

// TestGC 测试手动触发 GC
func TestGC(t *testing.T) {
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// 分配一些内存
	_ = make([]byte, 10*1024*1024) // 10MB

	// 手动触发 GC
	runtime.GC()

	runtime.ReadMemStats(&m2)

	// GC 次数应该增加
	if m2.NumGC <= m1.NumGC {
		t.Errorf("GC 次数应该增加：之前 %d，之后 %d", m1.NumGC, m2.NumGC)
	}
}

// TestAtomicInt64 测试原子计数器
func TestAtomicInt64(t *testing.T) {
	var counter atomic.Int64
	var wg sync.WaitGroup

	goroutines := 100
	increments := 1000

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				counter.Add(1)
			}
		}()
	}

	wg.Wait()

	expected := int64(goroutines * increments)
	actual := counter.Load()

	if actual != expected {
		t.Errorf("原子计数器结果不正确：期望 %d，实际 %d", expected, actual)
	}
}

// TestAtomicBool 测试原子布尔值同步
func TestAtomicBool(t *testing.T) {
	var flag atomic.Bool
	var data int
	var wg sync.WaitGroup

	wg.Add(2)

	// 写入 goroutine
	go func() {
		defer wg.Done()
		data = 42
		flag.Store(true) // release 语义
	}()

	// 读取 goroutine
	go func() {
		defer wg.Done()
		for !flag.Load() { // acquire 语义
			runtime.Gosched()
		}
		if data != 42 {
			t.Errorf("数据同步失败：期望 42，实际 %d", data)
		}
	}()

	// 设置超时防止死锁
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 测试通过
	case <-time.After(5 * time.Second):
		t.Fatal("测试超时：可能发生死锁")
	}
}

// TestStackGrowth 测试栈增长
func TestStackGrowth(t *testing.T) {
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// 递归调用以触发栈增长
	var recursiveFunc func(depth int)
	recursiveFunc = func(depth int) {
		if depth <= 0 {
			return
		}
		// 在栈上分配一些数据
		var buf [1024]byte
		_ = buf
		recursiveFunc(depth - 1)
	}

	recursiveFunc(100)

	runtime.ReadMemStats(&m2)

	// 栈内存使用应该有变化（可能增长或 GC 后收缩）
	// 这里只验证统计信息可以正常读取
	if m2.StackInuse == 0 && m1.StackInuse == 0 {
		t.Error("栈内存统计异常：两次都是 0")
	}
}

// TestChannelHappensBefore 测试 channel 的 happens-before 保证
func TestChannelHappensBefore(t *testing.T) {
	var x int
	ch := make(chan struct{})

	go func() {
		x = 1
		ch <- struct{}{} // 发送 happens-before 接收
	}()

	<-ch

	if x != 1 {
		t.Errorf("channel 同步失败：期望 x=1，实际 x=%d", x)
	}
}

// TestMutexHappensBefore 测试 Mutex 的 happens-before 保证
func TestMutexHappensBefore(t *testing.T) {
	var mu sync.Mutex
	var y int

	mu.Lock()
	y = 2
	mu.Unlock() // 解锁 happens-before 下一次加锁

	mu.Lock()
	if y != 2 {
		t.Errorf("Mutex 同步失败：期望 y=2，实际 y=%d", y)
	}
	mu.Unlock()
}

// TestWaitGroupHappensBefore 测试 WaitGroup 的 happens-before 保证
func TestWaitGroupHappensBefore(t *testing.T) {
	var z int
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		z = 3
		wg.Done() // Done happens-before Wait 返回
	}()

	wg.Wait()

	if z != 3 {
		t.Errorf("WaitGroup 同步失败：期望 z=3，实际 z=%d", z)
	}
}

// TestOnceHappensBefore 测试 sync.Once 的 happens-before 保证
func TestOnceHappensBefore(t *testing.T) {
	var once sync.Once
	var config string

	// 多个 goroutine 尝试初始化
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			once.Do(func() {
				config = "loaded" // Do 内的写入 happens-before Do 返回
			})
			// 所有 goroutine 都应该看到 "loaded"
			if config != "loaded" {
				t.Errorf("sync.Once 同步失败：期望 config='loaded'，实际 config='%s'", config)
			}
		}()
	}

	wg.Wait()
}

// TestGosched 测试主动让出 CPU
func TestGosched(t *testing.T) {
	// 这个测试只验证 Gosched 不会 panic
	for i := 0; i < 10; i++ {
		runtime.Gosched()
	}
}

// TestNumCPU 测试 CPU 核心数查询
func TestNumCPU(t *testing.T) {
	cpus := runtime.NumCPU()
	if cpus < 1 {
		t.Errorf("NumCPU 应该至少返回 1，实际返回 %d", cpus)
	}

	// GOMAXPROCS 默认应该等于 NumCPU
	maxProcs := runtime.GOMAXPROCS(0)
	if maxProcs != cpus {
		t.Logf("注意：GOMAXPROCS (%d) != NumCPU (%d)，可能被手动设置过", maxProcs, cpus)
	}
}

// TestMemStats 测试内存统计信息读取
func TestMemStats(t *testing.T) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 验证一些基本字段有合理的值
	if m.Sys == 0 {
		t.Error("MemStats.Sys 不应该是 0")
	}

	if m.Alloc > m.TotalAlloc {
		t.Errorf("当前分配 (%d) 不应该大于累计分配 (%d)", m.Alloc, m.TotalAlloc)
	}
}

// BenchmarkAtomicAdd 基准测试原子自增
func BenchmarkAtomicAdd(b *testing.B) {
	var counter atomic.Int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			counter.Add(1)
		}
	})
}

// BenchmarkMutexIncrement 基准测试互斥锁保护的自增
func BenchmarkMutexIncrement(b *testing.B) {
	var mu sync.Mutex
	var counter int64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			counter++
			mu.Unlock()
		}
	})
}

// BenchmarkGoroutineCreation 基准测试 goroutine 创建开销
func BenchmarkGoroutineCreation(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			done := make(chan struct{})
			go func() {
				close(done)
			}()
			<-done
		}
	})
}

// BenchmarkChannelSendRecv 基准测试 channel 发送接收
func BenchmarkChannelSendRecv(b *testing.B) {
	ch := make(chan int)
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
