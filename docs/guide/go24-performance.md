# 第 24 章 · 性能分析与优化

第 23 章讲清了 Go 运行时「为什么这样跑」：GMP 调度、GC、内存模型。知道底层机制之后，下一步是把它变成工程判断力——**怎么度量性能、怎么找到真正的瓶颈、怎么证明优化确实有效**。

性能优化只有一条铁律：**不要猜，要量**。Go 的标准库自带完整的测量工具链：`testing.B` 做基准测试、`runtime.MemStats` 数分配、`runtime/pprof` 采多维性能数据、`go tool trace` 看时间线。本章先讲清这些工具的用法和输出读法，再落到几个收益最明显的优化手段：预分配、`strings.Builder`、零拷贝转换、`sync.Pool`、锁竞争消除。

本章还有一个贯穿始终的约定：**正文里的数字都来自当前仓库的真实运行结果**。为了让这些数字可复现，示例度量的是「分配次数」而不是耗时——分配次数由代码路径决定，换台机器也基本一致；耗时随 CPU 频率、后台进程、GC 时机抖动，只适合同一台机器上的前后对比。

本章配套代码在 `internal/chapter/go24_performance/`，执行 `go run ./cmd/go-learn` 可以看到全部输出；耗时数字用 `go test -bench=. -benchmem ./internal/chapter/go24_performance/` 单独跑。

---

## 24.1 benchmark 的正确写法与分配度量

### 结论

基准测试函数写在 `_test.go` 里，签名固定为 `func BenchmarkXxx(b *testing.B)`。三条硬规则：循环次数用框架给的 `b.N`、初始化代码必须排除在计时之外、**被测结果必须真的被使用**——否则编译器会把整段代码优化掉，你会得到「一次操作 0.5 纳秒、零分配」这种漂亮的假数据。

### 最小可运行示例

```go
// BenchmarkAppendNoPrealloc 测量不预分配时的追加开销。
func BenchmarkAppendNoPrealloc(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkSlice = appendIntsNoPrealloc(1000) // 结果写进包级变量，防止被优化掉
	}
}

// BenchmarkAppendPrealloc 测量预分配容量后的追加开销。
func BenchmarkAppendPrealloc(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkSlice = appendIntsPrealloc(1000)
	}
}
```

```bash
go test -bench=. -benchmem ./internal/chapter/go24_performance/
```

### 实测输出

```text
goos: windows
goarch: amd64
pkg: go-learn/internal/chapter/go24_performance
cpu: Intel(R) Core(TM) i7-14700F
BenchmarkAppendNoPrealloc-28       	   89626	      3762 ns/op	   25208 B/op	      12 allocs/op
BenchmarkAppendPrealloc-28         	  299383	      1486 ns/op	    8192 B/op	       1 allocs/op
BenchmarkMapNoHint-28              	   10000	     34957 ns/op	   74457 B/op	      22 allocs/op
BenchmarkMapWithHint-28            	   44382	      9933 ns/op	   36992 B/op	       6 allocs/op
BenchmarkConcatPlus-28             	   18651	     17037 ns/op	   84440 B/op	     199 allocs/op
BenchmarkConcatBuilder-28          	 1239505	       357.4 ns/op	     896 B/op	       1 allocs/op
BenchmarkStringToBytesCopy-28      	17200424	        34.68 ns/op	      64 B/op	       1 allocs/op
BenchmarkStringToBytesUnsafe-28    	692891319	         0.5037 ns/op	       0 B/op	       0 allocs/op
BenchmarkCounterMutex-28           	 8054086	        44.26 ns/op	       0 B/op	       0 allocs/op
BenchmarkCounterAtomic-28          	30064136	        12.36 ns/op	       0 B/op	       0 allocs/op
BenchmarkCounterSharded-28         	15985364	        22.14 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	go-learn/internal/chapter/go24_performance	5.796s
```

每列的含义：`-28` 是运行时的 `GOMAXPROCS`（这台机器 28 逻辑核），`89626` 是 1 秒内执行的次数，`3762 ns/op` 是单次操作耗时（随机器负载波动），`25208 B/op` 与 `12 allocs/op` 是单次操作的堆分配字节数和次数。

预分配把追加 1000 个 int 的开销从 3762 ns 降到 1486 ns，分配次数从 12 次降到 1 次。这就是「先量再改」的价值：方向不需要猜。

### 用分配次数做可复现的度量

耗时数字写进文档就会过期，分配次数不会。本章的演示程序用 `runtime.ReadMemStats` 的差值来度量分配，跑出来的数字换台机器也基本一致：

```go
func measureAllocs(runs int, f func()) allocStats {
	f() // 预热：消化 map 桶、字符串缓冲这类懒初始化

	previous := debug.SetGCPercent(-1) // 测量期间关掉 GC，避免 GC 自身的分配混进计数
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
```

演示程序里同一组对比的输出（`--- 24.1 ---` 小节）：

```text
append 1000 个 int，不预分配:   9 次分配/op，25152 B/op
append 1000 个 int，预分配容量: 1 次分配/op，8192 B/op
预分配把「按倍数扩容 + 整体拷贝」压成一次分配，分配次数从十几次降到一次
```

### 边界与坑

上面的 9 次和 benchmark 表里的 12 次是同一段代码，为什么不一样？因为**分配次数的绝对值受编译器的内联决策和测量写法影响**。同一台机器上实测：把被测代码直接内联进循环，得到 12 次分配、25208 B；通过 `func` 值间接调用，得到 9 次分配、25152 B，差值 56 B 正好是前三次扩容的 8 B + 16 B + 32 B。两种结果都真实，但要记住两点：

- **对比必须同口径**：优化前后用同一个 benchmark 写法、同一组参数，不要拿演示程序里的计数去核对 `-benchmem` 的数字。
- **看数量级和字节数**：12 次降到 1 次、25208 B 降到 8192 B 才是结论，「少 3 次」这种精度没有意义。

另外两条经验：微秒级操作加 `-count=5` 多跑几轮，跨机器对比用 `benchstat`；不要用 `time.Now()` 给微秒级操作计时，`time.Since` 的分辨率和调度抖动会直接淹没信号。

---

## 24.2 预分配：切片与 map

### 结论

切片和 map 都能在创建时给出容量预期：`make([]T, 0, n)` 与 `make(map[K]V, n)`。切片省掉的是反复扩容和数据搬迁，map 省掉的是重新散列全部已有键。这是收益最确定、改动最小的一类优化——**前提是你事先知道大概要装多少**。

### 最小可运行示例

```go
// 慢：append 自己按倍数扩容，每次都要把旧内容拷到新数组
var s []int
for i := 0; i < 1000; i++ {
	s = append(s, i)
}

// 快：一次分配到位
s = make([]int, 0, 1000)
for i := 0; i < 1000; i++ {
	s = append(s, i)
}

// map 同理
m := make(map[int]int, 1000)
for i := 0; i < 1000; i++ {
	m[i] = i
}
```

### 实测输出

```text
写入 1000 个键值对，make(map[int]int):       22 次分配/op，74456 B/op
写入 1000 个键值对，make(map[int]int, 1000): 6 次分配/op，36992 B/op
map 扩容要把已有键重新散列到新桶，预分配省下的正是这批搬迁工作
```

对应的基准测试结论（见 24.1 的表）：map 写入从 34957 ns/op 降到 9933 ns/op，分配从 74457 B 降到 36992 B。

拆开看就清楚了：不预分配时，map 扩容要「分配新桶数组 → 重新散列迁移旧键值对 → 释放旧数组」，1000 个元素要经历十几轮这样的搬迁；预分配把桶数组一次建好，写入过程中不再搬迁。

### 边界与坑

| 写法 | 实测结果 | 说明 |
| --- | --- | --- |
| `make(map[int]int, n)`，`n` 是负变量 | 不 panic，得到空 map（实测输出 `0`） | 容量只是提示，运行时不会校验 |
| `make([]int, 0, n)`，`n` 是负变量 | `panic: runtime error: makeslice: cap out of range` | 切片的容量是硬约束 |
| `make(map[int]int, -1)` 直接写常量 | 编译错误：`invalid argument: index -1 (constant of type int) must not be negative` | 常量负数在编译期就被拦下 |

容量提示给多了同样有代价：`make([]int, 0, 1<<20)` 会立刻占用 8 MB，哪怕最后只放一个元素。**合理做法是按已知规模估计**（一批任务的数量、一次请求解析出的记录条数），而不是无脑给一个大数。

---

## 24.3 字符串构建与 string/[]byte 零拷贝转换

### 结论

两件事值得记住：字符串拼接用 `strings.Builder` 而不是 `+=`；`[]byte(s)` 和 `string(b)` 都会复制内存，只读场景可以用 `unsafe` 做零拷贝——但代价是你必须自己保证不写、且源数据活着。

### 最小可运行示例

```go
// 慢：每拼一次都可能重新分配并整体拷贝已有内容
func concatWithPlus(parts int) string {
	s := ""
	for i := 0; i < parts; i++ {
		s += "part"
	}
	return s
}

// 快：Builder 复用同一块缓冲区，只在扩容时拷贝
func concatWithBuilder(parts int) string {
	var b strings.Builder
	b.Grow(parts * len("part"))
	for i := 0; i < parts; i++ {
		b.WriteString("part")
	}
	return b.String()
}
```

### 实测输出

```text
拼接 200 段字符串，s += "part":      199 次分配/op，84440 B/op
拼接 200 段字符串，strings.Builder: 1 次分配/op，896 B/op
+= 每次都新建字符串并整体拷贝旧内容，Builder 只在扩容时拷贝
```

拼接 200 段字符串，`+=` 触发了 199 次分配、84440 B；Builder 只有 1 次分配、896 B。对应的基准测试也印证了这个量级：17037 ns/op 对 357.4 ns/op。

### string 与 []byte 的两种转换

```go
// stringToBytesSafe 把字符串复制成可写的字节切片，代价是一次分配和拷贝。
func stringToBytesSafe(s string) []byte {
	return []byte(s)
}

// stringToBytesUnsafe 零拷贝地把字符串看作字节切片。
//
// 契约（违反即为未定义行为）：返回的切片只读，绝不能写入；源字符串必须
// 在切片使用期间一直存活。需要可写副本时请用 stringToBytesSafe。
func stringToBytesUnsafe(s string) []byte {
	if len(s) == 0 {
		// 空字符串没有可取的底层数组，unsafe.StringData 会返回未定义的指针。
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
```

```text
[]byte(s) 转换:   1 次分配/op，64 B/op
unsafe 零拷贝:    0 次分配/op，0 B/op
零拷贝结果与源字符串共享底层数组: true
共享内存意味着返回的切片是只读的：写它会破坏 string 的不可变性
```

基准测试同样印证：34.68 ns/op 与 0.5037 ns/op。注意最后一行 `true` 是判断两个指针是否相等的实测结果——共享底层数组既是零拷贝的来源，也是「不能写」的原因：你写的就是那个字符串本身。字符串字面量位于只读内存段，改写它会直接让进程崩掉（见下文报错 5）。

### 边界与坑

- **需要修改就别用零拷贝**：`buf[0] = 'H'` 这类操作先用 `[]byte(s)` 拿到副本；从函数返回的视图也不能比源字符串活得更久。
- **空串要单独处理**：`unsafe.StringData("")` 返回未定义的指针，示例里用 `len(s) == 0` 提前返回 `nil`。
- **别对外暴露这个视图**：它是只读约定，一旦传给第三方库或交给可能写它的函数，约定就失效了。封装成不导出函数、只在自己包内使用。
- **先量再换**：转换次数不多时，`[]byte(s)` 的开销完全可以忽略，`unsafe` 带来的维护成本不划算。

---

## 24.4 sync.Pool 复用临时对象

### 结论

`sync.Pool` 缓存的是**可重建的临时对象**，价值在于降低分配次数和 GC 压力，而不是让单次操作变快。用之前先确认两件事：对象确实会被反复创建（比如每个请求一个 1 KB 缓冲区），以及对象能被安全重置。

### 最小可运行示例

```go
pool := &sync.Pool{
	New: func() any {
		return make([]byte, 1024)
	},
}

buf := pool.Get().([]byte)
buf = buf[:0] // 归还前清空脏数据
// ... 使用 buf ...
pool.Put(buf)
```

### 实测输出

```text
1000 次请求走 sync.Pool: 新建缓冲区 1 个
1000 次请求每次都 make([]byte, 1024): 新建缓冲区 1000 个，共 1024000 B
Get/Put 本身的装箱开销: 1 次分配/op，24 B/op
对象越大、分配越频繁，Pool 省下的越多；换来的是 24 字节的装箱开销
Pool 不保证对象持久，GC 会清空它，因此只能放可重建的临时对象
归还前记得重置对象内容，否则脏数据会流给下一个使用者
```

1000 次取还只触发了 1 次对象构造，省下了 1 MB 的分配。注意第三行：`Get`/`Put` 的参数类型是 `any`，每次取还把切片头装箱会产生一次 24 字节的分配——这是接口的代价，不是 Pool 的代价。对象越大、分配越频繁，这笔开销越值得。反过来说，如果池里放的是几个字节的小对象，装箱开销可能比省下的还多。

测量这段代码时演示程序临时关掉了 GC：

```go
previous := debug.SetGCPercent(-1) // Pool 里的对象每次 GC 都可能被清空
defer debug.SetGCPercent(previous)
```

这也正是 `sync.Pool` 最重要的边界：**它不是缓存**。运行时会在每次 GC 时清空池中的对象，所以池里的东西必须能随时重建，绝不能放数据库连接、文件句柄这类需要显式关闭的资源。

### 边界与坑

- **归还前重置**：`buf = buf[:0]` 或 `buf.Reset()`。忘了重置，上一个使用者遗留的数据会泄给下一个使用者，这类 bug 极难排查。
- **不要放长连接**：HTTP 连接、数据库连接有独立的连接池（`http.Transport`、`sql.DB`），`sync.Pool` 只管短命的内存对象。
- **`Get` 可能返回 `nil`**：没设 `New` 且池为空时会返回 `nil`，类型断言前先判断。另外 `Get`/`Put` 虽然是并发安全的，但对象优先回到当前 P 的本地池，跨 P 复用要等 GC 或偷取。
- **`-race` 下新建次数会变大**：竞态版本的 `sync.Pool.Put` 会随机丢弃约 1/4 的对象（`sync/pool.go` 里写着 `Randomly drop x on floor`），专门用来暴露「假设对象一定被复用」的代码。同一段循环在本仓库正常构建下 `New` 只调用 1 次，`go test -race` 下约 240 次，属于预期行为。

---

## 24.5 锁竞争与原子操作

### 结论

并发的瓶颈常常不是代码算得慢，而是 goroutine 排队等锁。三种常见解法按侵入性从低到高排列：**原子操作**（单个计数器）、**分段锁**（把一把锁拆成多把）、**本地累加后合并**（每个 goroutine 自己算，最后汇总）。

### 最小可运行示例

```go
// 方案一：原子操作，无锁
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

// 方案二：分段锁，把竞争分摊到 16 把锁上
type shardCounter struct {
	shards [16]struct {
		mu sync.Mutex
		n  int
	}
}

func (c *shardCounter) add(key int) {
	shard := &c.shards[key%len(c.shards)]
	shard.mu.Lock()
	shard.n++
	shard.mu.Unlock()
}
```

### 实测输出

```text
100 个 goroutine 各累加 100 次，结果都应当是 10000：
sync.Mutex 串行加锁:   10000
atomic.Int64 无锁自增: 10000
16 段分段锁:           10000
三种写法结果一致，差别只在竞争开销，耗时对比见 bench_test.go
```

三种写法都得到 10000，这是优化前的必要条件：**先保证结果一致，再谈快不快**。对应的基准测试（`b.RunParallel`，28 个并行执行者）：

| 写法 | ns/op | 相对开销 |
| --- | --- | --- |
| `sync.Mutex` | 44.26 | 基准 |
| `atomic.Int64` | 12.36 | 约 1/4 |
| 16 段分段锁 | 22.14 | 约 1/2 |

分段锁的优势取决于 key 是否被打散：命中不同分段的 goroutine 之间完全不阻塞；如果所有 key 都落到同一段，分段锁退化成一把普通锁，还多了一层取模和指针跳转。

### 边界与坑

- **原子操作只保一个变量**：`atomic.Int64` 能保证计数器正确，但没法让「改状态」和「写日志」这两步整体原子。多字段需要一起变，还是得上锁。
- **不要自己造无锁结构**：复杂场景直接用 `sync.Mutex`，先保证正确；分段、原子这类优化留到 pprof 或 `-race` 指出瓶颈之后再做。分段数也不要超过 CPU 核数，否则收益停止、只剩内存和汇总开销。
- **用 `-race` 兜底**：任何并发优化之后都跑一遍 `go test -race ./...`，正确性优先。

---

## 24.6 逃逸分析与内联

### 结论

Go 的内存分配位置由编译器决定：能在栈上放的在栈上放，会「逃出」当前函数的才放到堆上。堆分配的代价是多一次分配和一份 GC 压力，所以热路径上的意外逃逸值得关注。内联则是把小的函数调用展开成函数体，省掉调用开销，也让后续优化有更多空间。

### 最小可运行示例

```go
// 按值返回：调用方拿到一个副本，局部变量无需逃逸
func returnValue() int {
	return 42
}

// 按指针返回：局部变量必须活到调用方用完，只能放到堆上
func returnPointer() *int {
	v := 42
	return &v
}
```

### 实测输出

```text
返回 int 值:      0 次分配/op
返回 *int:        1 次分配/op
返回指针让局部变量必须活到调用方用完，编译器只能把它放到堆上
查看逃逸结论: go build -gcflags='-m' ./internal/chapter/go24_performance/
查看内联决策: go build -gcflags='-m=2' ./internal/chapter/go24_performance/
```

编译器给出的逐行结论（摘自当前仓库的真实输出，行号对应 `internal/chapter/go24_performance/demo.go`）：

```text
internal\chapter\go24_performance/demo.go:87:13: append escapes to heap
internal\chapter\go24_performance/demo.go:94:11: make([]int, 0, n) escapes to heap
internal\chapter\go24_performance/demo.go:200:16: ([]byte)(s) escapes to heap
internal\chapter\go24_performance/demo.go:211:6: can inline stringToBytesUnsafe
internal\chapter\go24_performance/demo.go:379:6: can inline returnValue
internal\chapter\go24_performance/demo.go:384:6: can inline returnPointer
internal\chapter\go24_performance/demo.go:385:2: moved to heap: v
```

读法很直接：`escapes to heap` / `moved to heap` 表示这个值最终分配在堆上；`can inline` 表示函数会被展开到调用点。同一个函数既 `can inline` 又让某个变量 `moved to heap` 并不矛盾——`returnPointer` 可以被内联，但局部变量 `v` 的地址被返回，仍然要逃逸。用 `-m=2` 还能看到「函数体过大」「包含闭包」「递归」这类没内联的原因。

### 边界与坑

- **逃逸不等于慢**：一次小对象分配约几十纳秒。只有在热路径上、量级达到每秒百万次时才值得动手，先看 pprof 再决定。
- **接口和闭包是常见触发点**：把值塞进 `any`（装箱）、闭包捕获了要在函数返回后使用的变量，都会让它逃逸。
- **`-m` 的输出不是承诺**：它是编译器的当前决策，会随版本变化，不要把它当 API 写进代码注释。
- **别为了消掉逃逸写出怪代码**：把 `*int` 改成返回值确实省了分配，但也要看调用方是否需要共享或修改。可读性优先，热点再优化。

---

## 24.7 pprof 与 go tool trace

### 结论

`pprof` 回答「时间花在谁身上」，`go tool trace` 回答「时间是怎么被耗掉的」。前者按函数维度聚合，后者按时间线展开调度、GC、阻塞事件。定位热点先用 pprof，看不出明显热点但程序整体很慢，再上 trace。

### 最小可运行示例

```go
f, err := os.Create("cpu.pprof")
if err != nil {
	return err
}
if err := pprof.StartCPUProfile(f); err != nil {
	return err
}
defer pprof.StopCPUProfile() // 保证一定会停止采样
defer f.Close()

// ... 被测的业务代码 ...
```

常驻服务更常用 HTTP 端点的方式，导入空包即可暴露默认路由：

```go
import _ "net/http/pprof"

go http.ListenAndServe("localhost:6060", nil)
// 采集 30 秒 CPU：curl -o cpu.pprof 'http://localhost:6060/debug/pprof/profile?seconds=30'
```

### 实测输出

```text
runtime/pprof 注册的 profile 类型: allocs, block, goroutine, goroutineleak, heap, mutex, threadcreate
各自负责一类问题：goroutine 查泄漏，heap 查内存，block/mutex 查阻塞与锁竞争
CPU profile 写入 cpu.pprof（867 字节）
Heap profile 写入 heap.pprof（5544 字节）
Trace 文件 写入 trace.out（6892 字节）
文件大小随采样到的内容变化，每次运行都不一样
goroutine profile 首行: goroutine profile: total 2
分析命令：
  go tool pprof -http=:8080 cpu.pprof
  go tool pprof -inuse_space heap.pprof
  go tool trace trace.out
```

profile 类型与用途对照：

| 类型 | 采集方式 | 用来回答 |
| --- | --- | --- |
| `cpu` | `pprof.StartCPUProfile` | CPU 时间花在哪些函数 |
| `heap` | `pprof.WriteHeapProfile` | 堆上活了多少对象、谁分配的 |
| `allocs` | `pprof.Lookup("allocs")` | 累计分配量（含已回收的） |
| `goroutine` | `pprof.Lookup("goroutine")` | 当前所有 goroutine 的调用栈，用于查泄漏 |
| `block` | 先 `runtime.SetBlockProfileRate(1)` | 阻塞在 channel、锁、select 上的时间 |
| `mutex` | 先 `runtime.SetMutexProfileFraction(1)` | 锁竞争的热点 |
| `goroutineleak` | `pprof.Lookup("goroutineleak")` | 已经泄漏、再也不会结束的 goroutine（`debug=2` 能拿到和 panic 一样的栈） |

拿真实数据跑一次 `top`，输出是这样的（采集自本章的 `busyWork` 基准程序）：

```text
File: go24perf.exe
Type: cpu
Duration: 403.82ms, Total samples = 390ms (96.58%)
Showing nodes accounting for 390ms, 100% of 390ms total
      flat  flat%   sum%        cum   cum%
     380ms 97.44% 97.44%      380ms 97.44%  go-learn/internal/chapter/go24_performance.busyWork (inline)
      10ms  2.56%   100%       10ms  2.56%  runtime.osPreemptExtEnter
```

`flat` 是本函数自身消耗的时间，`cum` 是包含被调用函数在内的累计时间。哪个函数 `flat` 占比高，优化就从它开始；`cum` 高但 `flat` 低，说明时间花在它调用的下游。**百分比和耗时每次运行都不一样，要看的是排序和量级。**

### 边界与坑

- **`block`/`mutex` 默认不采样**：不先调用 `runtime.SetBlockProfileRate` / `runtime.SetMutexProfileFraction`，采集到的文件里没有样本。生产环境用较大的分母（如 100）降低开销。
- **`pprof` 需要二进制才能解析符号**：`go tool pprof <二进制> <profile>`。只给 profile 文件，函数名可能解析不出来。
- **线上别裸奔暴露 pprof**：`http.ListenAndServe(":6060", nil)` 监听所有网卡，且 `/debug/pprof` 能拿到内存里的堆快照（可能包含敏感数据）。绑到 `localhost` 或用独立管理端口加鉴权。
- **采样不是精确统计**：CPU profile 默认 100 Hz，几毫秒的程序采不到样本；profile 文件是二进制，分析结果随采样内容变化，不要把某次的百分比写进文档当结论。

---

## 6 个真实报错怎么读

下面每一条都是在本仓库实测跑出来的输出。路径里的目录名取决于你把示例放在哪儿，行号则精确对应上面的最小示例。

### 报错 1：预分配之后忘了用 → 编译不过

```go
func main() {
	buf := make([]byte, 0, 1024) // 预分配了缓冲区，后面却忘了用它
	fmt.Println("请求处理完成")
}
```

```text
.tmp-errors\unused\main.go:6:2: declared and not used: buf
```

Go 把「声明了没使用」当编译错误，正是为了在优化过程中拦住这种半成品改动。**修复线索在第 6 行**：要么真的用上 `buf`，要么删掉这行预分配——`make` 本身分配了内存，白留一个 1 KB 的切片也是浪费。

### 报错 2：并发累加没同步 → 数据竞争

```go
for i := 0; i < 4; i++ {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 1000; j++ {
			counter++ // 没有同步，多个 goroutine 交错读写
		}
	}()
}
```

```bash
go run -race ./.tmp-errors/race
```

```text
==================
WARNING: DATA RACE
Read at 0x00c00010e068 by goroutine 9:
  main.main.func1()
      .../.tmp-errors/race/main.go:17 +0x99

Previous write at 0x00c00010e068 by goroutine 8:
  main.main.func1()
      .../.tmp-errors/race/main.go:17 +0xab
==================
counter = 4000
Found 2 data race(s)
exit status 66
```

读法：`WARNING: DATA RACE` 后面成对出现「这次是读/写」和「上一次是读/写」，两条栈都指向 `main.go:17`——就是 `counter++` 那一行。内存地址和 goroutine 编号每次都不同，唯一稳定的是文件与行号。顺带留意最后一行 `counter = 4000`：**结果看起来完全正确，竞争依然存在**，所以数据竞争不能靠肉眼验证，只能靠 `-race`。

### 报错 3：并发写 map → 致命错误

```go
m := make(map[int]int)
// 两个 goroutine 同时写同一个 map
m[base+j] = j
```

```text
fatal error: concurrent map writes

goroutine 20 [running]:
internal/runtime/maps.fatal({0x7ff6eda60435?, 0x0?})
	.../src/runtime/panic.go:1195 +0x18
main.main.func1(0xf4240)
	.../.tmp-errors/concurrentmap/main.go:14 +0x68
created by main.main in goroutine 1
	.../.tmp-errors/concurrentmap/main.go:11 +0x4d
exit status 2
```

`fatal error` 是运行时直接终止进程，**不能被 `recover` 捕获**。第 14 行是那次写操作，第 11 行是启动 goroutine 的位置。修复方向二选一：用 `sync.Mutex`/`sync.RWMutex` 保护 map，或换成 `sync.Map`（读多写少、键集合固定的场景）。注意 `-race` 能更早地报出这个问题，`fatal error` 只是一定概率被撞上。

### 报错 4：`sync.Pool` 类型断言写错 → panic

```go
pool := sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

buf := pool.Get().([]byte) // 池里放的是 *bytes.Buffer，断言成 []byte
```

```text
panic: interface conversion: interface {} is *bytes.Buffer, not []uint8

goroutine 1 [running]:
main.main()
	C:/Users/123/Desktop/demo-project/go-learn/.tmp-errors/poolassert/main.go:14 +0xbb
exit status 2
```

报错把「实际类型」和「断言类型」都写出来了：`*bytes.Buffer` 不是 `[]uint8`。类型断言 `x.(T)` 在失败时直接 panic；不确定时用带 `ok` 的形式 `buf, ok := pool.Get().([]byte)`。更稳妥的做法是把 `Get`/`Put` 包一层类型化的函数（`getBuffer()` / `putBuffer()`），让断言只出现在一处。

### 报错 5：用 unsafe 写字符串的底层字节 → 进程崩溃

```go
s := "hello"
b := unsafe.Slice(unsafe.StringData(s), len(s))

b[0] = 'H' // 字符串字面量位于只读内存段，这一行会直接崩溃
```

```text
unexpected fault address 0x7ff6fde261a9
fatal error: fault
[signal 0xc0000005 code=0x1 addr=0x7ff6fde261a9 pc=0x7ff6fde25226]

goroutine 1 [running]:
runtime.throw({0x7ff6fde2627b?, 0x26573edf8150?})
	.../src/runtime/panic.go:1243 +0x4d
runtime.sigpanic()
	.../src/runtime/signal_windows.go:398 +0xd0
main.main()
	.../.tmp-errors/unsafe/main.go:12 +0x26
exit status 2
```

`signal 0xc0000005` 是 Windows 的访问违例（Linux 上是 `SIGSEGV`），`main.go:12` 就是那行写操作。再看 24.3 的实测输出：`共享底层数组: true`——正因为共享，写它就等于写那个只读的字符串。**修复方式只有一个：需要写就先用 `[]byte(s)` 复制一份**，不要试图用 `recover` 兜住，`fatal error` 抓不住。

### 报错 6：CPU profile 重复启动 → 采样失败

```go
if err := pprof.StartCPUProfile(f); err != nil {
	panic(err)
}
defer pprof.StopCPUProfile()

// 忘了已经有一次采样在跑，又启动一次
if err := pprof.StartCPUProfile(f); err != nil {
	fmt.Println("第二次启动失败:", err)
}
```

```text
第二次启动失败: cpu profiling already in use
```

CPU profile 是进程级单例，同一时刻只能有一个在跑。常见触发场景是：HTTP 端点和手动采集同时进行，或者把 `StartCPUProfile` 写在会被多次调用的函数里。修复思路是保证「启动—停止」成对出现（用 `defer` 或显式同步），并且不要在每次请求里启动采样。

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| benchmark 跑出 `0.09 ns/op`、`0 allocs/op` | 被测结果没人使用，整段代码被编译器优化掉 | 把结果写进包级 sink 变量，或 `runtime.KeepAlive` |
| 同样代码两次测量分配次数不同 | 内联决策与调用形状不同，小对象分配路径随之变化 | 前后对比用同一套写法，看数量级和字节数 |
| 优化后反而更慢 | 没有先测量，凭直觉改 | 先 benchmark + pprof 定位热点，改完再量一次 |
| `[]byte(s)` 出现在热路径 | 每次转换都分配并拷贝 | 只读场景用零拷贝视图；否则复用缓冲区 |
| 用 unsafe 视图后程序崩溃 | 写入了字符串的只读内存段 | 需要写就用 `[]byte(s)` 拿副本 |
| `sync.Pool` 里的对象「消失」了，或取出时带着旧数据 | GC 会清空 Pool；归还前没有重置 | 池里只放可重建的临时对象，`buf = buf[:0]` / `buf.Reset()` 后再 `Put` |
| `fatal error: concurrent map writes` | 多个 goroutine 无同步地写 map | 加锁或换 `sync.Map`；`-race` 提前发现 |
| 计数器结果对但数据竞争警告不断 | 竞态导致的结果恰好正确 | 用 atomic、锁或 channel 消除竞争 |
| 高并发下 CPU 没跑满、耗时却很长 | goroutine 排队等同一把锁 | 原子操作、分段锁、本地累加后合并 |
| 堆内存持续增长 | 切片/map 引用未释放、goroutine 泄漏 | heap profile 对比快照 + goroutine profile |
| `go tool pprof` 看不到函数名 | 只传了 profile，没传二进制 | `go tool pprof <二进制> <profile>` |
| block/mutex profile 里没有样本 | 没有先开启采样率 | `runtime.SetBlockProfileRate` / `SetMutexProfileFraction` |

---

## 练习

### 第 1 题

下面的 benchmark 有两个问题，找出来并修复：

```go
func BenchmarkSum(b *testing.B) {
	data := make([]int, 10000)
	for i := range data {
		data[i] = i
	}

	sum := 0
	for i := 0; i < b.N; i++ {
		for _, v := range data {
			sum += v
		}
	}
}
```

::: details 第 1 题参考答案

```go
func BenchmarkSum(b *testing.B) {
	data := make([]int, 10000)
	for i := range data {
		data[i] = i
	}

	b.ResetTimer() // 问题一：初始化数据的耗时被算进了被测逻辑

	sum := 0
	for i := 0; i < b.N; i++ {
		for _, v := range data {
			sum += v
		}
	}
	b.StopTimer()
	sinkInt = sum // 问题二：结果要落到包级变量，否则整个循环可能被优化掉
}
```

**为什么这样写更好**：`b.ResetTimer()` 把构造 10000 个元素的耗时排除，测量结果才反映求和本身；把 `sum` 写入包级变量，编译器就不能假设这段循环没有副作用。两个问题都属于「测的不是你想测的东西」，前者让数字偏大，后者让数字偏小到接近 0。

:::

### 第 2 题

写两个 benchmark 对比 `map[string]int` 在 100 个 key 和 10000 个 key 时的查找性能，并用 `-benchmem` 观察分配差异。

::: details 第 2 题参考答案

```go
func BenchmarkMapLookupSmall(b *testing.B) {
	m := make(map[string]int, 100)
	for i := 0; i < 100; i++ {
		m[strconv.Itoa(i)] = i
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkInt = m["50"]
	}
}
```

大 map 版本把 `100` 换成 `10000`、查找键换成 `"5000"` 即可，其余保持一致——两个 benchmark 的差别只有 map 规模，跑出来的差异才能归因到规模上。`b.ReportAllocs()` 用来确认查找路径本身零分配：如果这里出现分配，说明查找过程中做了额外的事（字符串拼接、类型装箱），那才是真正的优化点。注意字符串作为 map key 时 `m["50"]` 不产生分配，但 `m[strconv.Itoa(i)]` 会。

:::

### 第 3 题

下面的函数在每次请求里被调用，输入大约 1000 行。找出可以优化的地方：

```go
func Process(lines []string) []string {
	var result []string
	for _, line := range lines {
		result = append(result, strings.ToUpper(line))
	}
	return result
}
```

::: details 第 3 题参考答案

```go
func Process(lines []string) []string {
	result := make([]string, 0, len(lines)) // 预分配：元素个数已知
	for _, line := range lines {
		result = append(result, strings.ToUpper(line))
	}
	return result
}
```

**为什么这样写更好**：输入长度 `len(lines)` 就是输出长度的上界，`make([]string, 0, len(lines))` 让 1000 次 `append` 只发生一次分配，而不是经历十几次扩容和元素搬迁。注意 `strings.ToUpper` 对每个元素仍会分配新字符串，这部分无法避免，属于「必要的分配」——优化的目标是消掉**不必要**的那部分，改动前先用 `-benchmem` 看清分配来自哪里。

:::

### 第 4 题

实现一个复用的 `bytes.Buffer` 对象池，要求：`GetBuffer` 返回的对象一定是干净的、`PutBuffer` 之后对象可以被安全复用、发生 panic 也能归还。

::: details 第 4 题参考答案

```go
var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func GetBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset() // 关键：清空上一个使用者留下的数据
	return buf
}

func PutBuffer(buf *bytes.Buffer) {
	buf.Reset() // 归还前也清一次，避免大缓冲区一直被引用
	bufferPool.Put(buf)
}

func WriteSomething(data []byte) string {
	buf := GetBuffer()
	defer PutBuffer(buf) // 即使中途 panic 也会归还

	buf.Write(data)
	return buf.String()
}
```

**为什么这样写更好**：`Reset()` 放在 `Get` 里保证调用方拿到的状态是确定的，放在 `Put` 里则能及时断开对大底层数组的引用。`defer PutBuffer(buf)` 也是必须的——手动归还时只要有一条错误分支提前返回，对象就永久丢失，池的命中率会下降。池里的对象类型统一为 `*bytes.Buffer`，所以断言只出现在 `GetBuffer` 一处。

:::

### 第 5 题

下面这个计数器在 1000 个 goroutine 并发调用时很慢。说明原因，给出两种优化方案，并说明怎么验证正确性。

```go
type Counter struct {
	mu    sync.Mutex
	count int
}

func (c *Counter) Inc() {
	c.mu.Lock()
	c.count++
	c.mu.Unlock()
}
```

::: details 第 5 题参考答案

**原因**：1000 个 goroutine 争抢同一把锁，同一时刻只有一个能进入临界区，其余全部排队。临界区里只做了一次自增，锁本身的开销远大于业务逻辑。

方案一：原子操作。

```go
type Counter struct {
	count atomic.Int64
}

func (c *Counter) Inc() {
	c.count.Add(1)
}
```

方案二：分片累加（`locals []atomic.Int64`，`Inc(shard int)` 只写 `locals[shard%n]`，读取时逐个 `Load` 求和）。每个 goroutine 写自己的槽位，几乎不存在共享写，适合「写多读少」；代价是需要一个稳定的分片依据，读时要汇总。

**怎么验证**：先写测试断言并发结束后总数等于期望值（比如 1000 个 goroutine 各加 100 次，必须得到 100000），再跑 `go test -race ./...` 确认没有数据竞争，最后用 benchmark 对比 `ns/op`。

**为什么这样写更好**：原子操作把「加锁—修改—解锁」压缩成一条原子指令，避免了排队；分片方案则把竞争分散到多个内存槽位上。两者都保持了「先把结果算对，再让结果变快」的顺序。

:::

## 小结

- **先测量再优化**：`go test -bench` 看耗时，`-benchmem` 看分配，`pprof` 定位热点，`trace` 看时间线。没有数据支撑的「优化」都是猜测。
- **benchmark 的正确性比数字更重要**：用 `b.N` 循环、用 `b.ResetTimer` 排除准备阶段、把结果写进包级变量防优化。耗时数字受内联和调用形状影响，对比时要保持同一口径；分配次数由代码路径决定，是可复现的指标。
- **预分配是最便宜的优化**：`make([]T, 0, n)` 与 `make(map[K]V, n)` 把多次扩容搬运压成一次分配，但提示值要看实际规模，给大了反而浪费内存。
- **字符串的两种优化各有代价**：拼接用 `strings.Builder`（实测 200 段从 199 次分配降到 1 次）；`unsafe.Slice(unsafe.StringData(s), len(s))` 零拷贝省掉一次分配，代价是「只读」和「源字符串必须活着」两条硬约束，违反会直接 `fatal error`。
- **`sync.Pool` 降低的是 GC 压力**：池里的对象随时可能被 GC 清空，只能放可重建的临时对象；归还前一定重置，注意 `Get`/`Put` 使用 `any` 带来的 24 字节装箱开销。
- **锁竞争从结构上解决**：原子操作用于单个计数器，分段锁用于热点集中的共享状态，分片累加适合写多读少。改完一定跑 `-race`。
- **逃逸分析解释分配从哪来**：`go build -gcflags='-m'` 给出逐行结论，内联用 `-m=2` 查看原因。逃逸不等于慢，值不值得改由 profile 决定。
- **pprof 的三种读法**：CPU 找热点、heap 找泄漏、goroutine 找卡住的协程；`block`/`mutex` 需要先开采样率，线上暴露 pprof 端点是安全事故。

下一章将讲解 **网络编程基础与 TCP/UDP**：从标准库 `net` 包出发，写一个能处理粘包、带超时控制和优雅关闭的 echo 服务。
