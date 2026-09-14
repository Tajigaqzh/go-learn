// Package go23_runtime 演示 Go 运行时、调度与内存模型。
//
// 本章覆盖：
//   - GMP 调度模型与 GOMAXPROCS
//   - goroutine 栈增长与调度追踪
//   - GC 三色标记、写屏障与 GC 参数
//   - 内存分配与逃逸分析
//   - 内存模型与 happens-before
//   - sync/atomic 的语义
//   - 构建标签与 GODEBUG
package go23_runtime

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// Demo 是本章的统一入口。
func Demo() {
	fmt.Println("========== go23_runtime: 运行时、调度与内存模型 ==========")

	fmt.Println("\n--- 23.1 GMP 调度模型与 GOMAXPROCS ---")
	demoGMP()

	fmt.Println("\n--- 23.2 goroutine 栈增长与调度信息 ---")
	demoStackGrowth()

	fmt.Println("\n--- 23.3 GC 基础与三色标记 ---")
	demoGC()

	fmt.Println("\n--- 23.4 GC 参数：GOGC 与 GOMEMLIMIT ---")
	demoGCParams()

	fmt.Println("\n--- 23.5 内存分配与逃逸分析 ---")
	demoEscape()

	fmt.Println("\n--- 23.6 内存模型与 happens-before ---")
	demoMemoryModel()

	fmt.Println("\n--- 23.7 sync/atomic 的语义 ---")
	demoAtomic()

	fmt.Println("\n--- 23.8 runtime 包常用函数 ---")
	demoRuntimeFuncs()

	fmt.Println("\n--- 23.9 构建标签与 GODEBUG ---")
	demoBuildTags()

	fmt.Println("\n========== 运行时、调度与内存模型演示结束 ==========")
}

// demoGMP 演示 GMP 调度模型与 GOMAXPROCS。
func demoGMP() {
	// Go 调度器使用 GMP 模型：
	// G（goroutine）：用户态轻量级线程
	// M（machine）：OS 线程
	// P（processor）：调度上下文，数量由 GOMAXPROCS 决定

	// 查看当前 GOMAXPROCS
	maxProcs := runtime.GOMAXPROCS(0) // 传 0 不修改，只查询
	fmt.Printf("GOMAXPROCS = %d（默认等于 CPU 核心数）\n", maxProcs)

	// 查看当前 goroutine 数量
	fmt.Printf("当前 goroutine 数量：%d\n", runtime.NumGoroutine())

	// 启动一些 goroutine
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// 短暂计算，模拟工作
			sum := 0
			for j := 0; j < 1000; j++ {
				sum += j
			}
		}(i)
	}

	fmt.Printf("启动 10 个 goroutine 后：%d\n", runtime.NumGoroutine())
	wg.Wait()
	fmt.Printf("等待完成后：%d\n", runtime.NumGoroutine())

	// 调度器会在多个 P 上调度 G，每个 P 绑定一个 M
	// 当 G 阻塞（如 syscall、channel）时，M 会解绑并尝试获取其他 P
}

// demoStackGrowth 演示 goroutine 栈增长。
func demoStackGrowth() {
	// Go 1.3+ 使用连续栈（contiguous stack）
	// 初始栈很小（2KB），按需增长到 1GB

	// 递归调用会触发栈增长
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	deepRecursion(0, 100)

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	fmt.Printf("栈内存增长：%d bytes\n", memAfter.StackInuse-memBefore.StackInuse)

	// 栈增长时，runtime 会分配新的更大的栈，复制数据，更新指针
	// 这对用户是透明的，但会有性能开销
}

// deepRecursion 递归调用，演示栈增长。
func deepRecursion(depth, max int) int {
	if depth >= max {
		return depth
	}
	// 在栈上分配一些空间
	var buf [128]byte
	_ = buf
	return deepRecursion(depth+1, max)
}

// demoGC 演示 GC 基础与三色标记。
func demoGC() {
	// Go 使用并发三色标记清除算法（tricolor mark-and-sweep）
	// 三色：白色（未扫描）、灰色（已扫描但子对象未扫描）、黑色（已扫描且子对象已扫描）

	// 查看 GC 统计
	var stats debug.GCStats
	debug.ReadGCStats(&stats)
	fmt.Printf("GC 次数：%d\n", stats.NumGC)
	if len(stats.Pause) > 0 {
		fmt.Printf("最近一次 GC 暂停：%v\n", stats.Pause[0])
	}

	// 手动触发 GC
	fmt.Println("手动触发 GC...")
	runtime.GC()

	debug.ReadGCStats(&stats)
	fmt.Printf("GC 次数（触发后）：%d\n", stats.NumGC)

	// 写屏障（write barrier）：在 GC 标记阶段，
	// 当修改指针时插入的代码，确保新指向的对象被标记
	// 用户代码通常感知不到，但会有轻微性能开销
}

// demoGCParams 演示 GC 参数：GOGC 与 GOMEMLIMIT。
func demoGCParams() {
	// GOGC：目标堆增长百分比（默认 100，即堆翻倍时触发 GC）
	// 可以通过环境变量 GOGC=200 或代码设置
	oldGOGC := debug.SetGCPercent(200) // 堆增长 200% 时触发 GC
	fmt.Printf("GOGC 从 %d 改为 200\n", oldGOGC)
	debug.SetGCPercent(oldGOGC) // 恢复

	// GOMEMLIMIT（Go 1.19+）：软内存限制
	// 环境变量 GOMEMLIMIT=2GiB 或代码设置
	// 注意：这是软限制，不是硬限制，GC 会尽量不超过但不保证
	oldLimit := debug.SetMemoryLimit(-1) // 传 -1 只查询
	fmt.Printf("当前内存限制：%d bytes（-1 表示无限制）\n", oldLimit)

	// 调整 GC 参数的场景：
	// - 高吞吐：提高 GOGC（如 200-300），减少 GC 频率，代价是更高内存
	// - 低延迟：降低 GOGC（如 50），更频繁 GC，减少每次暂停时间
	// - 容器环境：设置 GOMEMLIMIT 接近容器内存限制，避免 OOM
}

// demoEscape 演示内存分配与逃逸分析。
func demoEscape() {
	// 逃逸分析（escape analysis）：编译器决定变量分配在栈还是堆
	// 栈分配快、无 GC 压力；堆分配慢、需要 GC

	// 使用 go build -gcflags="-m" 查看逃逸分析结果

	// 不逃逸：局部变量，不会被外部引用
	x := 42
	fmt.Printf("局部变量 x = %d（通常在栈上）\n", x)

	// 逃逸：返回局部变量的指针
	p := makePointer()
	fmt.Printf("逃逸到堆的指针：%v\n", *p)

	// 逃逸：传递给 interface{} 或 fmt.Println
	// 因为编译器无法静态确定接口的具体类型
	var iface interface{} = 100
	fmt.Printf("interface{} 导致逃逸：%v\n", iface)

	// 逃逸：闭包捕获的变量如果在闭包返回后仍被访问
	f := makeCounter()
	fmt.Printf("闭包捕获的变量逃逸：%d\n", f())
}

// makePointer 返回局部变量的指针，导致逃逸。
func makePointer() *int {
	x := 42
	return &x // x 逃逸到堆
}

// makeCounter 返回闭包，捕获的变量逃逸。
func makeCounter() func() int {
	count := 0
	return func() int {
		count++ // count 逃逸到堆
		return count
	}
}

// demoMemoryModel 演示内存模型与 happens-before。
func demoMemoryModel() {
	// Go 内存模型定义了多个 goroutine 读写共享变量时的可见性保证
	// 核心概念：happens-before

	// 错误示例：没有同步的并发读写（数据竞争）
	// var x int
	// go func() { x = 1 }()
	// fmt.Println(x) // 可能看到 0 或 1，未定义行为

	// 正确示例 1：channel 提供 happens-before 保证
	var x int
	ch := make(chan struct{})
	go func() {
		x = 1
		ch <- struct{}{} // 发送 happens-before 接收
	}()
	<-ch
	fmt.Printf("通过 channel 同步，x = %d\n", x) // 保证看到 1

	// 正确示例 2：sync.Mutex 提供 happens-before 保证
	var mu sync.Mutex
	var y int
	mu.Lock()
	y = 2
	mu.Unlock() // 解锁 happens-before 下一次加锁
	mu.Lock()
	fmt.Printf("通过 Mutex 同步，y = %d\n", y)
	mu.Unlock()

	// 正确示例 3：sync.WaitGroup 保证
	var z int
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		z = 3
		wg.Done() // Done happens-before Wait 返回
	}()
	wg.Wait()
	fmt.Printf("通过 WaitGroup 同步，z = %d\n", z)

	// 错误的同步：用普通变量做 flag（没有 happens-before 保证）
	// var done bool
	// go func() {
	//     doWork()
	//     done = true // 可能被重排序或不可见
	// }()
	// for !done {} // 可能永远循环
}

// demoAtomic 演示 sync/atomic 的语义。
func demoAtomic() {
	// atomic 操作提供：
	// 1. 原子性（不可分割）
	// 2. happens-before 保证（类似加锁）

	var counter atomic.Int64
	var wg sync.WaitGroup

	// 多个 goroutine 并发自增
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Add(1) // 原子自增
		}()
	}
	wg.Wait()
	fmt.Printf("atomic 计数器：%d（预期 100）\n", counter.Load())

	// Load/Store 提供 acquire/release 语义
	var flag atomic.Bool
	var data int

	// goroutine 1：写入数据后设置标志
	go func() {
		data = 42
		flag.Store(true) // release：确保之前的写入对后续 Load 可见
	}()

	// goroutine 2：等待标志后读取数据
	for !flag.Load() { // acquire：确保看到 Store 之前的写入
		runtime.Gosched() // 让出 CPU
	}
	fmt.Printf("通过 atomic.Bool 同步，data = %d\n", data)

	// 注意：atomic 只保证单个变量的操作，
	// 多个变量的复合操作仍需要锁或 channel
}

// demoRuntimeFuncs 演示 runtime 包常用函数。
func demoRuntimeFuncs() {
	// NumCPU：逻辑 CPU 核心数
	fmt.Printf("CPU 核心数：%d\n", runtime.NumCPU())

	// NumGoroutine：当前 goroutine 数量
	fmt.Printf("当前 goroutine 数：%d\n", runtime.NumGoroutine())

	// Gosched：主动让出 CPU 时间片
	go func() {
		for i := 0; i < 3; i++ {
			fmt.Printf("  goroutine A: %d\n", i)
			runtime.Gosched() // 让出，让其他 goroutine 执行
		}
	}()
	for i := 0; i < 3; i++ {
		fmt.Printf("  main: %d\n", i)
		runtime.Gosched()
	}
	time.Sleep(10 * time.Millisecond) // 等待 goroutine 完成

	// GC：手动触发 GC
	// runtime.GC()

	// ReadMemStats：读取内存统计
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("堆内存分配：%d MB\n", m.Alloc/1024/1024)
	fmt.Printf("系统内存：%d MB\n", m.Sys/1024/1024)
	fmt.Printf("GC 次数：%d\n", m.NumGC)

	// Stack：返回当前 goroutine 的栈追踪
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, false)
	fmt.Printf("栈追踪（前 100 字节）：\n%s\n", buf[:min(n, 100)])
}

// demoBuildTags 演示构建标签与 GODEBUG。
func demoBuildTags() {
	// 构建标签（build tags）：条件编译
	// 写在文件顶部注释：//go:build linux && amd64
	// 用于平台特定代码、测试、可选功能

	// GODEBUG：运行时调试参数（环境变量）
	// 例如：
	// - GODEBUG=gctrace=1：打印 GC 信息
	// - GODEBUG=schedtrace=1000：每秒打印调度信息
	// - GODEBUG=madvdontneed=1：内存归还策略
	// - GODEBUG=http2debug=1：HTTP/2 调试

	// 查看 GODEBUG 的值（通常在环境变量中设置）
	// 在代码里无法直接读取，但可以观察其效果
	fmt.Println("GODEBUG 参数在环境变量中设置，影响运行时行为")
	fmt.Println("例如：GODEBUG=gctrace=1 go run main.go")

	// Compiler：返回编译器名称
	fmt.Printf("编译器：%s\n", runtime.Compiler) // 通常是 "gc"

	// Version：Go 版本
	fmt.Printf("Go 版本：%s\n", runtime.Version())

	// GOOS/GOARCH：目标操作系统和架构
	fmt.Printf("操作系统：%s\n", runtime.GOOS)
	fmt.Printf("架构：%s\n", runtime.GOARCH)
}
