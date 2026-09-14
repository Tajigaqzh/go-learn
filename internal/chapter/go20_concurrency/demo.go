// Package go20_concurrency 演示 Go 并发基础：goroutine、channel、sync 包。
package go20_concurrency

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Demo 是第 20 章的统一入口。
func Demo() {
	fmt.Println("========== go20_concurrency: 并发基础 ==========")
	fmt.Println("==========================================")
	fmt.Println()

	demoGoroutineBasics()
	demoWaitGroup()
	demoChannelBasics()
	demoChannelDirection()
	demoChannelClose()
	demoChannelSelect()
	demoMutex()
	demoRWMutex()
	demoOnce()
	demoAtomic()
	demoDataRace()

	fmt.Println()
	fmt.Println("========== 并发基础演示结束 ==========")
}

// --- 1. goroutine 基础 ---

func demoGoroutineBasics() {
	fmt.Println("--- 1. goroutine 基础 ---")

	// 1.1 最简单的 goroutine
	go func() {
		fmt.Println("Hello from goroutine")
	}()

	// 必须等待，否则主 goroutine 退出后子 goroutine 会被强制终止
	time.Sleep(100 * time.Millisecond)

	// 1.2 goroutine 与闭包
	for i := range 3 {
		// Go 1.22+ 每次循环都是新变量，直接捕获安全
		go func() {
			fmt.Printf("Loop %d\n", i)
		}()
	}
	time.Sleep(100 * time.Millisecond)

	fmt.Println("\ngoroutine 特点：")
	fmt.Println("• 轻量级：初始栈只有 2KB，按需增长")
	fmt.Println("• 调度：M:N 模型，Go 运行时调度到系统线程")
	fmt.Println("• 主 goroutine 退出时，所有子 goroutine 会被强制终止")
	fmt.Println()
}

// --- 2. sync.WaitGroup ---

func demoWaitGroup() {
	fmt.Println("--- 2. sync.WaitGroup：等待多个 goroutine 完成 ---")

	var wg sync.WaitGroup

	for i := range 3 {
		wg.Add(1) // 启动前 +1
		go func(id int) {
			defer wg.Done() // 完成时 -1
			time.Sleep(50 * time.Millisecond)
			fmt.Printf("Worker %d done\n", id)
		}(i)
	}

	wg.Wait() // 等待计数器归零
	fmt.Println("All workers finished")

	fmt.Println("\nWaitGroup 规则：")
	fmt.Println("• Add() 必须在启动 goroutine 前调用")
	fmt.Println("• Done() 等价于 Add(-1)")
	fmt.Println("• Wait() 阻塞直到计数器归零")
	fmt.Println("• 计数器不能为负（panic）")
	fmt.Println()
}

// --- 3. channel 基础 ---

func demoChannelBasics() {
	fmt.Println("--- 3. channel 基础 ---")

	// 3.1 无缓冲 channel：同步
	ch1 := make(chan int)
	go func() {
		ch1 <- 42 // 发送会阻塞，直到有人接收
	}()
	val := <-ch1 // 接收会阻塞，直到有人发送
	fmt.Printf("无缓冲 channel 收到：%d\n", val)

	// 3.2 有缓冲 channel：异步
	ch2 := make(chan string, 2)
	ch2 <- "hello"
	ch2 <- "world"
	// 缓冲区满之前不阻塞
	fmt.Printf("有缓冲 channel 收到：%s, %s\n", <-ch2, <-ch2)

	fmt.Println("\nchannel 特点：")
	fmt.Println("• 无缓冲 channel：发送方和接收方必须同时就绪（握手）")
	fmt.Println("• 有缓冲 channel：缓冲区满时发送阻塞，空时接收阻塞")
	fmt.Println("• 向 nil channel 收发会永久阻塞")
	fmt.Println("• 向已关闭 channel 发送会 panic")
	fmt.Println()
}

// --- 4. channel 方向 ---

func demoChannelDirection() {
	fmt.Println("--- 4. channel 方向：只发送、只接收 ---")

	ch := make(chan int, 1)

	// 只能发送的 channel
	go producer(ch)

	// 只能接收的 channel
	consumer(ch)

	fmt.Println("\nchannel 方向：")
	fmt.Println("• chan<- T：只能发送")
	fmt.Println("• <-chan T：只能接收")
	fmt.Println("• 双向 channel 可以隐式转换为单向")
	fmt.Println()
}

func producer(ch chan<- int) {
	ch <- 100
	close(ch) // 生产者关闭 channel
}

func consumer(ch <-chan int) {
	val := <-ch
	fmt.Printf("Consumer 收到：%d\n", val)
}

// --- 5. channel 关闭与 range ---

func demoChannelClose() {
	fmt.Println("--- 5. channel 关闭与 range 遍历 ---")

	ch := make(chan int, 3)

	// 生产者
	go func() {
		for i := range 3 {
			ch <- i * 10
		}
		close(ch) // 关闭后，range 会退出
	}()

	// 消费者：range 自动检测关闭
	for val := range ch {
		fmt.Printf("收到：%d\n", val)
	}

	// 从已关闭 channel 接收：返回零值 + false
	val, ok := <-ch
	fmt.Printf("从已关闭 channel 接收：val=%d, ok=%t\n", val, ok)

	fmt.Println("\n关闭规则：")
	fmt.Println("• 只有发送方可以关闭 channel")
	fmt.Println("• 关闭后不能再发送（panic），但可以接收")
	fmt.Println("• 接收时用 val, ok := <-ch 检测是否已关闭")
	fmt.Println("• range 会在 channel 关闭后自动退出")
	fmt.Println()
}

// --- 6. select 多路复用 ---

func demoChannelSelect() {
	fmt.Println("--- 6. select：多路复用 channel ---")

	ch1 := make(chan string, 1)
	ch2 := make(chan string, 1)

	ch1 <- "from ch1"
	ch2 <- "from ch2"

	// select 随机选择一个就绪的 case
	select {
	case msg := <-ch1:
		fmt.Println("收到", msg)
	case msg := <-ch2:
		fmt.Println("收到", msg)
	}

	// 超时控制
	ch3 := make(chan int)
	select {
	case val := <-ch3:
		fmt.Println("收到", val)
	case <-time.After(50 * time.Millisecond):
		fmt.Println("超时：50ms 内没有收到数据")
	}

	// 非阻塞接收
	select {
	case val := <-ch3:
		fmt.Println("收到", val)
	default:
		fmt.Println("没有数据，立即返回")
	}

	fmt.Println("\nselect 规则：")
	fmt.Println("• 多个 case 就绪时，随机选择一个")
	fmt.Println("• 都不就绪时阻塞，直到某个 case 就绪")
	fmt.Println("• default 使 select 变成非阻塞")
	fmt.Println("• select {} 会永久阻塞")
	fmt.Println()
}

// --- 7. sync.Mutex ---

func demoMutex() {
	fmt.Println("--- 7. sync.Mutex：互斥锁 ---")

	var (
		mu      sync.Mutex
		counter int
		wg      sync.WaitGroup
	)

	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			counter++
			mu.Unlock()
		}()
	}

	wg.Wait()
	fmt.Printf("最终计数：%d（期望 10）\n", counter)

	fmt.Println("\nMutex 规则：")
	fmt.Println("• Lock() 获取锁，Unlock() 释放锁")
	fmt.Println("• 同一 goroutine 不能重复 Lock（死锁）")
	fmt.Println("• Unlock() 必须由 Lock() 的 goroutine 调用")
	fmt.Println("• 习惯用法：defer mu.Unlock()")
	fmt.Println()
}

// --- 8. sync.RWMutex ---

func demoRWMutex() {
	fmt.Println("--- 8. sync.RWMutex：读写锁 ---")

	var (
		mu   sync.RWMutex
		data = map[string]int{"a": 1}
		wg   sync.WaitGroup
	)

	// 多个读者可以并发
	for range 3 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			mu.RLock()
			fmt.Printf("读者 %d：data=%v\n", id, data)
			mu.RUnlock()
		}(int(1 + 1))
	}

	// 写者独占
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		mu.Lock()
		data["b"] = 2
		fmt.Println("写者：data=", data)
		mu.Unlock()
	}()

	wg.Wait()

	fmt.Println("\nRWMutex 规则：")
	fmt.Println("• RLock() / RUnlock()：读锁，多个读者可以并发")
	fmt.Println("• Lock() / Unlock()：写锁，独占访问")
	fmt.Println("• 读多写少场景下性能优于 Mutex")
	fmt.Println()
}

// --- 9. sync.Once ---

func demoOnce() {
	fmt.Println("--- 9. sync.Once：确保只执行一次 ---")

	var (
		once sync.Once
		wg   sync.WaitGroup
	)

	init := func() {
		fmt.Println("初始化（只会打印一次）")
	}

	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			once.Do(init) // 只有第一次调用会执行 init
		}()
	}

	wg.Wait()

	fmt.Println("\nOnce 规则：")
	fmt.Println("• Do(f) 只会执行一次 f")
	fmt.Println("• 后续调用会阻塞，直到第一次执行完成")
	fmt.Println("• 常用于单例初始化、配置加载")
	fmt.Println()
}

// --- 10. sync/atomic ---

func demoAtomic() {
	fmt.Println("--- 10. sync/atomic：原子操作 ---")

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
	fmt.Printf("原子计数器：%d（期望 1000）\n", counter.Load())

	// 比较并交换（CAS）
	old := counter.Load()
	swapped := counter.CompareAndSwap(old, 2000)
	fmt.Printf("CAS：old=%d, swapped=%t, new=%d\n", old, swapped, counter.Load())

	fmt.Println("\natomic 规则：")
	fmt.Println("• 提供原子操作，无需锁")
	fmt.Println("• 适用于简单的计数器、标志位")
	fmt.Println("• Go 1.19+ 提供泛型类型：atomic.Int64, atomic.Pointer[T]")
	fmt.Println()
}

// --- 11. 数据竞争与 -race ---

func demoDataRace() {
	fmt.Println("--- 11. 数据竞争与 -race 检测 ---")

	var counter int
	var wg sync.WaitGroup

	// 故意制造数据竞争（仅用于演示）
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter++ // 非原子操作，有数据竞争
		}()
	}

	wg.Wait()
	fmt.Printf("数据竞争示例：counter=%d（结果不确定）\n", counter)

	fmt.Println("\n数据竞争：")
	fmt.Println("• 多个 goroutine 并发访问同一变量，至少一个是写操作")
	fmt.Println("• 使用 go test -race 或 go run -race 检测")
	fmt.Println("• 修复方法：mutex、channel、atomic")
	fmt.Println()

	fmt.Println("channel 与锁的取舍：")
	fmt.Println("• channel：适合传递所有权、流式处理、goroutine 间通信")
	fmt.Println("• mutex：适合保护共享状态、短临界区、缓存")
	fmt.Println("• \"Don't communicate by sharing memory; share memory by communicating.\"")
	fmt.Println()
}
