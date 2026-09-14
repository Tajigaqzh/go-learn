package go24_performance

import (
	"runtime"
	"runtime/debug"
)

// sink 系列变量用来接收被测函数的返回值。
//
// 如果计算结果没人使用，编译器会把整段代码优化掉，测量结果就会变成
// 「一次不到一纳秒、零分配」的假象。让结果落到包级变量上，被测代码
// 才会真的执行。这是性能测量里最常见的一个坑。
var (
	sinkSlice  []int
	sinkMap    map[int]int
	sinkString string
	sinkBytes  []byte
	sinkInt    int
	sinkIntPtr *int
)

// allocStats 是一次分配测量的结果。
type allocStats struct {
	// allocsPerRun 是每次调用触发的堆分配次数。
	allocsPerRun float64
	// bytesPerRun 是每次调用分配的字节数。
	bytesPerRun float64
}

// measureAllocs 统计 f 每次调用平均触发多少次堆分配、多少字节。
//
// 这里刻意不测墙上时钟：耗时随机器负载抖动，写进注释隔一天就对不上了；
// 分配次数由代码路径决定，同一段代码在多次运行之间是稳定的，所以更适合
// 写进正文对照。耗时对比交给 bench_test.go 里的 Benchmark，用
// go test -bench 运行。
//
// 两点细节：
//   - 先预热一次，把 map 桶、字符串缓冲这类懒初始化消化掉，避免第一次
//     调用的额外分配被摊进平均值。
//   - 测量期间关闭 GC。GC 自身会分配内存，混进计数里会让数字抖动；
//     关掉之后 sync.Pool 里的对象也不会在中途被清空。
func measureAllocs(runs int, f func()) allocStats {
	f()

	// 只留一个 P：调度器把 goroutine 在 P 之间搬来搬去时，sync.Pool
	// 这类带本地缓存的代码分配次数会跟着抖。标准库的 testing.AllocsPerRun
	// 同样做了这件事。
	previousProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previousProcs)

	previous := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(previous)

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < runs; i++ {
		f()
	}
	runtime.ReadMemStats(&after)

	runs64 := float64(runs)
	return allocStats{
		allocsPerRun: float64(after.Mallocs-before.Mallocs) / runs64,
		bytesPerRun:  float64(after.TotalAlloc-before.TotalAlloc) / runs64,
	}
}
