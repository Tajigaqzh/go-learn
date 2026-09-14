// Package go21_concurrency_patterns 演示 Go 并发模式与常见陷阱。
package go21_concurrency_patterns

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Demo 运行第 21 章所有演示。
func Demo() {
	fmt.Println("========== go21_concurrency_patterns: 并发模式与陷阱 ==========")
	demoWorkerPool()
	demoProducerConsumer()
	demoFanInFanOut()
	demoPipeline()
	demoCancellationTimeout()
	demoErrGroup()
	demoGoroutineLeaks()
	demoDeadlock()
	demoChannelCloseRules()
	demoConcurrencyTesting()
	fmt.Println("\n========== 并发模式与陷阱演示结束 ==========")
}

// --- 1. Worker Pool ---

func demoWorkerPool() {
	fmt.Println("\n--- 1. Worker Pool ---")
	fmt.Println("Worker Pool 模式：固定数量的 worker goroutine 消费任务队列")

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
	close(jobs)

	// 收集结果
	for a := 1; a <= 5; a++ {
		<-results
	}

	fmt.Println("✓ Worker Pool 完成，所有任务已处理")
}

func worker(id int, jobs <-chan int, results chan<- int) {
	for j := range jobs {
		fmt.Printf("  worker %d 处理 job %d\n", id, j)
		time.Sleep(10 * time.Millisecond)
		results <- j * 2
	}
}

// --- 2. 生产者消费者 ---

func demoProducerConsumer() {
	fmt.Println("\n--- 2. 生产者消费者 ---")
	fmt.Println("生产者消费者模式：生产者写入 channel，消费者读取并处理")

	ch := make(chan int, 5)
	var wg sync.WaitGroup

	// 生产者
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(ch) // 生产完毕后关闭 channel
		for i := 1; i <= 3; i++ {
			fmt.Printf("  生产者：写入 %d\n", i)
			ch <- i
		}
	}()

	// 消费者
	wg.Add(1)
	go func() {
		defer wg.Done()
		for v := range ch { // 读到 channel 关闭为止
			fmt.Printf("  消费者：读取 %d\n", v)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	wg.Wait()
	fmt.Println("✓ 生产者消费者完成")
}

// --- 3. Fan-In / Fan-Out ---

func demoFanInFanOut() {
	fmt.Println("\n--- 3. Fan-In / Fan-Out ---")
	fmt.Println("Fan-Out：一个输入分发给多个 worker")
	fmt.Println("Fan-In：多个 channel 合并到一个输出")

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
	count := 0
	for range output {
		count++
		if count == 4 { // 4 个任务完成
			break
		}
	}
	fmt.Println("✓ Fan-In / Fan-Out 完成")
}

func fanOutWorker(input <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range input {
			fmt.Printf("  worker 处理 %d\n", v)
			time.Sleep(10 * time.Millisecond)
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

// --- 4. Pipeline ---

func demoPipeline() {
	fmt.Println("\n--- 4. Pipeline ---")
	fmt.Println("Pipeline 模式：多个阶段串联，每个阶段是一个 goroutine")

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
		fmt.Printf("  pipeline 输出：%d\n", n)
	}
	fmt.Println("✓ Pipeline 完成")
}

// --- 5. 取消与超时组合 ---

func demoCancellationTimeout() {
	fmt.Println("\n--- 5. 取消与超时组合 ---")
	fmt.Println("使用 context 实现取消传播与超时控制")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result := make(chan string, 1)
	go func() {
		time.Sleep(100 * time.Millisecond) // 模拟慢任务
		result <- "done"
	}()

	select {
	case <-ctx.Done():
		fmt.Printf("  超时：%v\n", ctx.Err())
	case res := <-result:
		fmt.Printf("  结果：%s\n", res)
	}

	fmt.Println("✓ context 超时正确触发")
}

// --- 6. errgroup ---

func demoErrGroup() {
	fmt.Println("\n--- 6. errgroup ---")
	fmt.Println("golang.org/x/sync/errgroup：简化并发任务的错误处理")

	g := new(errgroup.Group)

	// 任务1：成功
	g.Go(func() error {
		time.Sleep(10 * time.Millisecond)
		fmt.Println("  任务1：成功")
		return nil
	})

	// 任务2：失败
	g.Go(func() error {
		time.Sleep(20 * time.Millisecond)
		fmt.Println("  任务2：失败")
		return errors.New("task 2 failed")
	})

	// 任务3：会被取消（如果使用 WithContext）
	g.Go(func() error {
		time.Sleep(30 * time.Millisecond)
		fmt.Println("  任务3：成功")
		return nil
	})

	if err := g.Wait(); err != nil {
		fmt.Printf("  errgroup 返回第一个错误：%v\n", err)
	}
	fmt.Println("✓ errgroup 演示完成")
}

// --- 7. Goroutine 泄漏 ---

func demoGoroutineLeaks() {
	fmt.Println("\n--- 7. Goroutine 泄漏 ---")
	fmt.Println("常见泄漏场景：channel 永久阻塞、context 未取消")

	fmt.Println("\n场景1：channel 接收者退出，发送者永久阻塞")
	ch := make(chan int)
	go func() {
		ch <- 1               // 永久阻塞，因为没有接收者
		fmt.Println("  发送成功") // 永远不会打印
	}()
	time.Sleep(10 * time.Millisecond)
	fmt.Println("  ⚠ 发送者 goroutine 已泄漏（没有接收者）")

	fmt.Println("\n场景2：正确做法 - 使用 context 取消")
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-ctx.Done():
			fmt.Println("  ✓ goroutine 收到取消信号，正常退出")
		case <-time.After(1 * time.Second):
			fmt.Println("  任务完成")
		}
	}()
	time.Sleep(10 * time.Millisecond)
	cancel() // 主动取消，避免泄漏
	time.Sleep(10 * time.Millisecond)

	fmt.Println("\n排查工具：")
	fmt.Println("• runtime.NumGoroutine() 查看当前 goroutine 数量")
	fmt.Println("• pprof goroutine profile 查看泄漏栈")
	fmt.Println("• go test -race 检测数据竞争")
}

// --- 8. 死锁 ---

func demoDeadlock() {
	fmt.Println("\n--- 8. 死锁 ---")
	fmt.Println("死锁：所有 goroutine 都在等待，程序无法继续")

	fmt.Println("\n场景1：循环等待锁")
	fmt.Println("  var mu1, mu2 sync.Mutex")
	fmt.Println("  goroutine A: mu1.Lock(); mu2.Lock()")
	fmt.Println("  goroutine B: mu2.Lock(); mu1.Lock()")
	fmt.Println("  ⚠ 可能死锁")

	fmt.Println("\n场景2：channel 永久阻塞")
	// ch := make(chan int)
	// ch <- 1  // 死锁：无缓冲 channel，没有接收者

	fmt.Println("  ch := make(chan int)")
	fmt.Println("  ch <- 1  // fatal error: all goroutines are asleep - deadlock!")

	fmt.Println("\n避免死锁：")
	fmt.Println("• 按固定顺序获取锁")
	fmt.Println("• 使用带缓冲的 channel 或 select + default")
	fmt.Println("• 使用 context 超时")
	fmt.Println("• 代码审查时画出依赖图")
}

// --- 9. Channel 关闭原则 ---

func demoChannelCloseRules() {
	fmt.Println("\n--- 9. Channel 关闭原则 ---")
	fmt.Println("原则：只有发送者关闭 channel，接收者永远不关闭")

	fmt.Println("\n✓ 正确：发送者关闭")
	ch := make(chan int, 2)
	ch <- 1
	ch <- 2
	close(ch) // 发送者关闭
	fmt.Println("  接收者 range 读取：", <-ch, <-ch)

	fmt.Println("\n⚠ 错误：向已关闭的 channel 发送")
	// ch <- 3  // panic: send on closed channel

	fmt.Println("\n✓ 检测 channel 是否关闭")
	v, ok := <-ch
	if !ok {
		fmt.Println("  channel 已关闭，v =", v) // v = 0
	}

	fmt.Println("\n多发送者场景：使用 sync.Once 或 context")
	var once sync.Once
	ch2 := make(chan int, 1)
	closeCh := func() {
		once.Do(func() {
			close(ch2)
			fmt.Println("  ✓ channel 只关闭一次")
		})
	}
	closeCh()
	closeCh() // 不会 panic
}

// --- 10. 并发代码测试 ---

func demoConcurrencyTesting() {
	fmt.Println("\n--- 10. 并发代码测试 ---")
	fmt.Println("测试策略：")
	fmt.Println("• 使用 go test -race 检测数据竞争")
	fmt.Println("• 使用 sync.WaitGroup 确保 goroutine 完成")
	fmt.Println("• 使用 context.WithTimeout 避免测试永久阻塞")
	fmt.Println("• 测试取消、超时、channel 关闭等边界条件")
	fmt.Println("• 使用 t.Parallel() 并发运行测试用例")

	fmt.Println("\n示例：测试 worker pool")
	fmt.Println("  func TestWorkerPool(t *testing.T) {")
	fmt.Println("      jobs := make(chan int, 10)")
	fmt.Println("      results := make(chan int, 10)")
	fmt.Println("      var wg sync.WaitGroup")
	fmt.Println("      for w := 1; w <= 3; w++ {")
	fmt.Println("          wg.Add(1)")
	fmt.Println("          go func() {")
	fmt.Println("              defer wg.Done()")
	fmt.Println("              for j := range jobs {")
	fmt.Println("                  results <- j * 2")
	fmt.Println("              }")
	fmt.Println("          }()")
	fmt.Println("      }")
	fmt.Println("      for j := 1; j <= 5; j++ { jobs <- j }")
	fmt.Println("      close(jobs)")
	fmt.Println("      wg.Wait()")
	fmt.Println("      close(results)")
	fmt.Println("      count := 0")
	fmt.Println("      for range results { count++ }")
	fmt.Printf("      if count != 5 { t.Errorf(\"expected 5, got %%d\", count) }\n")
	fmt.Println("  }")
}
