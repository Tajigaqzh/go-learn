// Package go24_performance 演示 Go 的性能分析与优化。
//
// 本章覆盖：
//   - benchmark 的正确写法与分配度量
//   - 预分配：切片与 map
//   - 字符串构建与 string / []byte 的零拷贝转换
//   - sync.Pool 复用临时对象
//   - 锁竞争：互斥锁、原子操作与分段锁
//   - 逃逸分析与内联
//   - pprof（CPU / heap / goroutine）与 go tool trace
//
// 演示里的数字都来自当前仓库的运行结果。为了让输出可复现，正文度量的是
// 「分配次数」而不是耗时：分配次数由代码路径决定，耗时则随机器负载抖动。
package go24_performance

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Demo 运行第 24 章所有演示。
func Demo() {
	fmt.Println("========== go24_performance: 性能分析与优化 ==========")

	fmt.Println("\n--- 24.1 benchmark 与分配度量 ---")
	demoBenchmarkAndAllocs()

	fmt.Println("\n--- 24.2 预分配：切片与 map ---")
	demoPreallocate()

	fmt.Println("\n--- 24.3 字符串构建与 string/[]byte 转换 ---")
	demoStrings()

	fmt.Println("\n--- 24.4 sync.Pool 复用临时对象 ---")
	demoSyncPool()

	fmt.Println("\n--- 24.5 锁竞争与原子操作 ---")
	demoLockContention()

	fmt.Println("\n--- 24.6 逃逸分析与内联 ---")
	demoEscapeInline()

	fmt.Println("\n--- 24.7 pprof 与 go tool trace ---")
	demoProfile()

	fmt.Println("\n========== 性能分析与优化演示结束 ==========")
}

// --- 24.1 benchmark 与分配度量 ---

// demoBenchmarkAndAllocs 讲清 benchmark 的写法，并用分配次数验证预分配的价值。
//
// 耗时数字交给 bench_test.go 里的 Benchmark，用 go test -bench 跑；
// 这里演示不依赖计时器的度量方式，输出的数字在多次运行之间保持一致。
func demoBenchmarkAndAllocs() {
	fmt.Println("benchmark 写在 _test.go 中，签名固定为 func BenchmarkXxx(b *testing.B)：")
	fmt.Println("• 循环次数交给框架（b.N），不要硬编码")
	fmt.Println("• 初始化代码放在循环外，用 b.ResetTimer() 排除它的耗时")
	fmt.Println("• 加上 -benchmem 才能看到 B/op 和 allocs/op")
	fmt.Println("• 被测结果必须被使用，否则整段代码会被编译器优化掉")
	fmt.Println()

	const runs = 200

	without := measureAllocs(runs, func() {
		sinkSlice = appendIntsNoPrealloc(1000)
	})
	with := measureAllocs(runs, func() {
		sinkSlice = appendIntsPrealloc(1000)
	})

	fmt.Printf("append 1000 个 int，不预分配:   %.0f 次分配/op，%.0f B/op\n",
		without.allocsPerRun, without.bytesPerRun)
	fmt.Printf("append 1000 个 int，预分配容量: %.0f 次分配/op，%.0f B/op\n",
		with.allocsPerRun, with.bytesPerRun)
	fmt.Println("预分配把「按倍数扩容 + 整体拷贝」压成一次分配，分配次数从十几次降到一次")
}

// appendIntsNoPrealloc 让 append 自己决定扩容时机，返回切片供调用方使用。
func appendIntsNoPrealloc(n int) []int {
	var s []int
	for i := 0; i < n; i++ {
		s = append(s, i)
	}
	return s
}

// appendIntsPrealloc 按已知长度预分配容量，避免中途扩容。
func appendIntsPrealloc(n int) []int {
	s := make([]int, 0, n)
	for i := 0; i < n; i++ {
		s = append(s, i)
	}
	return s
}

// --- 24.2 预分配：切片与 map ---

// demoPreallocate 对比 map 预设容量前后的分配次数。
func demoPreallocate() {
	const runs = 200

	without := measureAllocs(runs, func() {
		sinkMap = mapWithoutHint(1000)
	})
	with := measureAllocs(runs, func() {
		sinkMap = mapWithHint(1000)
	})

	fmt.Printf("写入 1000 个键值对，make(map[int]int):       %.0f 次分配/op，%.0f B/op\n",
		without.allocsPerRun, without.bytesPerRun)
	fmt.Printf("写入 1000 个键值对，make(map[int]int, 1000): %.0f 次分配/op，%.0f B/op\n",
		with.allocsPerRun, with.bytesPerRun)
	fmt.Println("map 扩容要把已有键重新散列到新桶，预分配省下的正是这批搬迁工作")
}

// mapWithoutHint 不预设容量，由运行时按需扩容。
func mapWithoutHint(n int) map[int]int {
	m := make(map[int]int)
	for i := 0; i < n; i++ {
		m[i] = i
	}
	return m
}

// mapWithHint 按预期元素个数预设容量。
func mapWithHint(n int) map[int]int {
	m := make(map[int]int, n)
	for i := 0; i < n; i++ {
		m[i] = i
	}
	return m
}

// --- 24.3 字符串构建与 string/[]byte 转换 ---

// demoStrings 对比字符串拼接方式，以及 string 与 []byte 的两种转换。
func demoStrings() {
	const parts = 200

	plus := measureAllocs(50, func() {
		sinkString = concatWithPlus(parts)
	})
	builder := measureAllocs(50, func() {
		sinkString = concatWithBuilder(parts)
	})

	fmt.Printf("拼接 200 段字符串，s += \"part\":      %.0f 次分配/op，%.0f B/op\n",
		plus.allocsPerRun, plus.bytesPerRun)
	fmt.Printf("拼接 200 段字符串，strings.Builder: %.0f 次分配/op，%.0f B/op\n",
		builder.allocsPerRun, builder.bytesPerRun)
	fmt.Println("+= 每次都新建字符串并整体拷贝旧内容，Builder 只在扩容时拷贝")
	fmt.Println()

	const sample = "hello world, this is a test string for conversion"

	copyStats := measureAllocs(1000, func() {
		sinkBytes = stringToBytesSafe(sample)
	})
	unsafeStats := measureAllocs(1000, func() {
		sinkBytes = stringToBytesUnsafe(sample)
	})

	fmt.Printf("[]byte(s) 转换:   %.0f 次分配/op，%.0f B/op\n",
		copyStats.allocsPerRun, copyStats.bytesPerRun)
	fmt.Printf("unsafe 零拷贝:    %.0f 次分配/op，%.0f B/op\n",
		unsafeStats.allocsPerRun, unsafeStats.bytesPerRun)

	view := stringToBytesUnsafe(sample)
	shared := unsafe.Pointer(unsafe.StringData(sample)) == unsafe.Pointer(unsafe.SliceData(view))
	fmt.Printf("零拷贝结果与源字符串共享底层数组: %t\n", shared)
	fmt.Println("共享内存意味着返回的切片是只读的：写它会破坏 string 的不可变性")
}

// concatWithPlus 用 += 逐段拼接，每拼一次都可能重新分配并拷贝已有内容。
func concatWithPlus(parts int) string {
	s := ""
	for i := 0; i < parts; i++ {
		s += "part"
	}
	return s
}

// concatWithBuilder 用 strings.Builder 复用同一块缓冲区。
func concatWithBuilder(parts int) string {
	var b strings.Builder
	b.Grow(parts * len("part"))
	for i := 0; i < parts; i++ {
		b.WriteString("part")
	}
	return b.String()
}

// stringToBytesSafe 把字符串复制成可写的字节切片，代价是一次分配和拷贝。
func stringToBytesSafe(s string) []byte {
	return []byte(s)
}

// stringToBytesUnsafe 零拷贝地把字符串看作字节切片。
//
// 契约（违反即为未定义行为，不是「可能出点小问题」）：
//   - 返回的切片是只读的，绝对不能写入。字符串字面量的底层字节位于只读
//     内存段，写入会直接触发 fatal error: unexpected fault address。
//   - 调用方必须保证源字符串在切片使用期间一直存活，否则切片会指向
//     已被回收的内存。
//   - 只在转换确实落在热路径上时使用；需要可写副本时请用 stringToBytesSafe。
func stringToBytesUnsafe(s string) []byte {
	if len(s) == 0 {
		// 空字符串没有可取的底层数组，unsafe.StringData 会返回未定义的指针。
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// --- 24.4 sync.Pool 复用临时对象 ---

// demoSyncPool 用「对象构造函数被调用了几次」衡量复用效果。
func demoSyncPool() {
	const runs = 1000

	// 只留一个 P：对象会优先回到当前 P 的本地池，多 P 之间迁移会让
	// 「新建了几个」这个数字随调度抖动。
	previousProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previousProcs)

	// sync.Pool 里的对象在每次 GC 时都可能被清空，不关掉 GC，这个数字
	// 就会随 GC 时机变化。测量期间关掉，测量结束立刻恢复。
	previous := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(previous)

	newCalls := 0
	pool := &sync.Pool{
		New: func() any {
			newCalls++
			return make([]byte, 1024)
		},
	}

	for i := 0; i < runs; i++ {
		buf := pool.Get().([]byte)
		buf[0] = byte(i)
		pool.Put(buf)
	}
	fmt.Printf("%d 次请求走 sync.Pool: 新建缓冲区 %d 个\n", runs, newCalls)

	fresh := measureAllocs(200, func() {
		sinkBytes = make([]byte, 1024)
	})
	fmt.Printf("%d 次请求每次都 make([]byte, 1024): 新建缓冲区 %.0f 个，共 %.0f B\n",
		runs, fresh.allocsPerRun*float64(runs), fresh.bytesPerRun*float64(runs))

	// Get/Put 的参数类型是 any，取还时把切片头装箱也要分配一次 24 字节。
	// 这是接口的代价，不是 Pool 的代价；池里省下的是整整 1 KB 的缓冲区。
	boxing := measureAllocs(200, func() {
		buf := pool.Get().([]byte)
		pool.Put(buf)
	})
	fmt.Printf("Get/Put 本身的装箱开销: %.0f 次分配/op，%.0f B/op\n",
		boxing.allocsPerRun, boxing.bytesPerRun)
	fmt.Println("对象越大、分配越频繁，Pool 省下的越多；换来的是 24 字节的装箱开销")
	fmt.Println("Pool 不保证对象持久，GC 会清空它，因此只能放可重建的临时对象")
	fmt.Println("归还前记得重置对象内容，否则脏数据会流给下一个使用者")
}

// --- 24.5 锁竞争与原子操作 ---

// demoLockContention 用三种写法累加同一个总数，验证优化不改变结果。
func demoLockContention() {
	const goroutines, perGoroutine = 100, 100
	const want = goroutines * perGoroutine

	fmt.Printf("%d 个 goroutine 各累加 %d 次，结果都应当是 %d：\n", goroutines, perGoroutine, want)
	fmt.Printf("sync.Mutex 串行加锁:   %d\n", countWithMutex(goroutines, perGoroutine))
	fmt.Printf("atomic.Int64 无锁自增: %d\n", countWithAtomic(goroutines, perGoroutine))
	fmt.Printf("16 段分段锁:           %d\n", countWithShards(goroutines, perGoroutine))
	fmt.Println("三种写法结果一致，差别只在竞争开销，耗时对比见 bench_test.go")
}

// countWithMutex 让所有 goroutine 争抢同一把锁。
func countWithMutex(goroutines, perGoroutine int) int {
	var mu sync.Mutex
	total := 0

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				mu.Lock()
				total++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return total
}

// countWithAtomic 用原子加法替代互斥锁。
func countWithAtomic(goroutines, perGoroutine int) int64 {
	var counter atomic.Int64

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				counter.Add(1)
			}
		}()
	}
	wg.Wait()
	return counter.Load()
}

// shardCounter 把计数分散到多把锁上，不同 goroutine 命中不同段时互不阻塞。
type shardCounter struct {
	shards [16]struct {
		mu sync.Mutex
		n  int
	}
}

// add 按 key 选定分段后累加。
func (c *shardCounter) add(key int) {
	shard := &c.shards[key%len(c.shards)]
	shard.mu.Lock()
	shard.n++
	shard.mu.Unlock()
}

// total 汇总所有分段，调用前必须确保没有并发的 add。
func (c *shardCounter) total() int {
	sum := 0
	for i := range c.shards {
		sum += c.shards[i].n
	}
	return sum
}

// countWithShards 用分段锁分摊竞争。
func countWithShards(goroutines, perGoroutine int) int {
	var counter shardCounter

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(key int) {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				counter.add(key)
			}
		}(i)
	}
	wg.Wait()
	return counter.total()
}

// --- 24.6 逃逸分析与内联 ---

// demoEscapeInline 用分配次数展示「返回值」和「返回指针」的差别。
func demoEscapeInline() {
	value := measureAllocs(200, func() {
		sinkInt = returnValue()
	})
	pointer := measureAllocs(200, func() {
		sinkIntPtr = returnPointer()
	})

	fmt.Printf("返回 int 值:      %.0f 次分配/op\n", value.allocsPerRun)
	fmt.Printf("返回 *int:        %.0f 次分配/op\n", pointer.allocsPerRun)
	fmt.Println("返回指针让局部变量必须活到调用方用完，编译器只能把它放到堆上")
	fmt.Println("查看逃逸结论: go build -gcflags='-m' ./internal/chapter/go24_performance/")
	fmt.Println("查看内联决策: go build -gcflags='-m=2' ./internal/chapter/go24_performance/")
}

// returnValue 直接返回整数，调用方拿到的是副本。
func returnValue() int {
	return 42
}

// returnPointer 返回局部变量的地址，该变量必须逃逸到堆上。
func returnPointer() *int {
	v := 42
	return &v
}
