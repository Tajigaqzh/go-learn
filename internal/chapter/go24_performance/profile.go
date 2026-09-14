package go24_performance

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sort"
	"strings"
	"sync"
)

// --- 24.7 pprof 与 go tool trace ---

// demoProfile 采集 CPU、堆、goroutine 三类 profile 和一个 trace 文件。
//
// 文件写到临时目录并在函数返回时删除，正文里的分析命令可以直接在真实
// 项目上复用。文件大小取决于采样内容，每次运行都不一样。
func demoProfile() {
	fmt.Printf("runtime/pprof 注册的 profile 类型: %s\n", strings.Join(profileNames(), ", "))
	fmt.Println("各自负责一类问题：goroutine 查泄漏，heap 查内存，block/mutex 查阻塞与锁竞争")

	dir, err := os.MkdirTemp("", "go24-profile-")
	if err != nil {
		fmt.Printf("创建临时目录失败: %v\n", err)
		return
	}
	defer os.RemoveAll(dir)

	writeProfile := func(name string, path string, size int64) {
		fmt.Printf("%s 写入 %s（%d 字节）\n", name, filepath.Base(path), size)
	}

	cpuPath, cpuSize := collectCPUProfile(dir)
	writeProfile("CPU profile", cpuPath, cpuSize)

	heapPath, heapSize := collectHeapProfile(dir)
	writeProfile("Heap profile", heapPath, heapSize)

	tracePath, traceSize := collectTrace(dir)
	writeProfile("Trace 文件", tracePath, traceSize)

	fmt.Println("文件大小随采样到的内容变化，每次运行都不一样")

	var buf bytes.Buffer
	if err := pprof.Lookup("goroutine").WriteTo(&buf, 1); err != nil {
		fmt.Printf("读取 goroutine profile 失败: %v\n", err)
	} else {
		summary := strings.SplitN(buf.String(), "\n", 2)[0]
		fmt.Printf("goroutine profile 首行: %s\n", summary)
	}

	fmt.Println("分析命令：")
	fmt.Println("  go tool pprof -http=:8080 cpu.pprof")
	fmt.Println("  go tool pprof -inuse_space heap.pprof")
	fmt.Println("  go tool trace trace.out")
}

// profileNames 返回运行时注册的 profile 类型名，排序后输出以便结果稳定。
func profileNames() []string {
	profiles := pprof.Profiles()
	names := make([]string, 0, len(profiles))
	for _, p := range profiles {
		names = append(names, p.Name())
	}
	sort.Strings(names)
	return names
}

// collectCPUProfile 采集一段 CPU profile，返回文件路径与字节数。
func collectCPUProfile(dir string) (string, int64) {
	path := filepath.Join(dir, "cpu.pprof")
	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("创建 CPU profile 文件失败: %v\n", err)
		return path, 0
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		fmt.Printf("启动 CPU profile 失败: %v\n", err)
		f.Close()
		return path, 0
	}

	// 采样率默认 100 Hz，跑够几十毫秒才能采到有效样本。
	sinkInt = busyWork(20_000_000)
	pprof.StopCPUProfile()

	if err := f.Close(); err != nil {
		fmt.Printf("关闭 CPU profile 文件失败: %v\n", err)
	}
	return path, fileSize(path)
}

// collectHeapProfile 采集一次堆快照，返回文件路径与字节数。
func collectHeapProfile(dir string) (string, int64) {
	path := filepath.Join(dir, "heap.pprof")
	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("创建 heap profile 文件失败: %v\n", err)
		return path, 0
	}

	// 留一块存活对象，否则堆快照里几乎只有运行时自身的分配。
	held := make([]byte, 1<<20)
	runtime.GC()
	runtime.KeepAlive(held)

	if err := pprof.WriteHeapProfile(f); err != nil {
		fmt.Printf("写入 heap profile 失败: %v\n", err)
		f.Close()
		return path, 0
	}
	if err := f.Close(); err != nil {
		fmt.Printf("关闭 heap profile 文件失败: %v\n", err)
	}
	return path, fileSize(path)
}

// collectTrace 采集一段执行 trace，返回文件路径与字节数。
func collectTrace(dir string) (string, int64) {
	path := filepath.Join(dir, "trace.out")
	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("创建 trace 文件失败: %v\n", err)
		return path, 0
	}

	if err := trace.Start(f); err != nil {
		fmt.Printf("启动 trace 失败: %v\n", err)
		f.Close()
		return path, 0
	}

	// 几个 goroutine 各自做一段确定量的运算，给时间线留下可观察的事件。
	results := make([]int, 4)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = busyWork(2_000_000)
		}(i)
	}
	wg.Wait()

	trace.Stop()
	if err := f.Close(); err != nil {
		fmt.Printf("关闭 trace 文件失败: %v\n", err)
	}
	return path, fileSize(path)
}

// busyWork 做一段确定数量的整数运算，给采样器和 trace 提供可观察的负载。
func busyWork(n int) int {
	sum := 0
	for i := 0; i < n; i++ {
		sum += i % 7
	}
	return sum
}

// fileSize 返回文件大小，出错时返回 0，避免为诊断代码引入额外错误分支。
func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
