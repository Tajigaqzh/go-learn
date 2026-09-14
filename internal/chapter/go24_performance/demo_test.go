package go24_performance

import (
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"unsafe"
)

// TestAppendPreallocReducesAllocs 验证预分配能把扩容带来的多次分配压成一次。
func TestAppendPreallocReducesAllocs(t *testing.T) {
	const runs = 50

	without := measureAllocs(runs, func() {
		sinkSlice = appendIntsNoPrealloc(1000)
	})
	with := measureAllocs(runs, func() {
		sinkSlice = appendIntsPrealloc(1000)
	})

	if with.allocsPerRun >= without.allocsPerRun {
		t.Errorf("预分配应当减少分配次数：不预分配 %.1f 次/op，预分配 %.1f 次/op",
			without.allocsPerRun, with.allocsPerRun)
	}
	if with.bytesPerRun >= without.bytesPerRun {
		t.Errorf("预分配应当减少分配字节数：不预分配 %.0f B/op，预分配 %.0f B/op",
			without.bytesPerRun, with.bytesPerRun)
	}
}

// TestAppendResultsMatch 验证两种 append 写法产生的内容完全一致。
func TestAppendResultsMatch(t *testing.T) {
	without := appendIntsNoPrealloc(1000)
	with := appendIntsPrealloc(1000)

	if len(without) != len(with) {
		t.Fatalf("长度不一致：不预分配 %d，预分配 %d", len(without), len(with))
	}
	for i := range without {
		if without[i] != with[i] {
			t.Fatalf("下标 %d 的值不一致：不预分配 %d，预分配 %d", i, without[i], with[i])
		}
	}
}

// TestMapHintReducesAllocs 验证 map 预设容量能减少扩容次数。
func TestMapHintReducesAllocs(t *testing.T) {
	const runs = 50

	without := measureAllocs(runs, func() {
		sinkMap = mapWithoutHint(1000)
	})
	with := measureAllocs(runs, func() {
		sinkMap = mapWithHint(1000)
	})

	if with.allocsPerRun >= without.allocsPerRun {
		t.Errorf("预设容量应当减少分配次数：不预设 %.1f 次/op，预设 %.1f 次/op",
			without.allocsPerRun, with.allocsPerRun)
	}
}

// TestMapContentMatch 验证带不带容量提示写出的内容相同。
func TestMapContentMatch(t *testing.T) {
	without := mapWithoutHint(1000)
	with := mapWithHint(1000)

	if len(without) != len(with) {
		t.Fatalf("键数量不一致：不预设 %d，预设 %d", len(without), len(with))
	}
	for i := 0; i < 1000; i++ {
		if without[i] != i || with[i] != i {
			t.Fatalf("下标 %d 的值不对：不预设 %d，预设 %d", i, without[i], with[i])
		}
	}
}

// TestBuilderMatchesPlus 验证 strings.Builder 与 += 拼出的内容一致。
func TestBuilderMatchesPlus(t *testing.T) {
	for _, parts := range []int{0, 1, 17, 200} {
		got := concatWithBuilder(parts)
		want := concatWithPlus(parts)
		if got != want {
			t.Errorf("拼接 %d 段结果不一致：Builder %q，+= %q", parts, got, want)
		}
		if want != strings.Repeat("part", parts) {
			t.Errorf("拼接 %d 段内容不对：%q", parts, want)
		}
	}
}

// TestStringToBytesSafeCopies 验证安全转换得到的是独立副本，可以随便改。
func TestStringToBytesSafeCopies(t *testing.T) {
	const original = "copy me"

	buf := stringToBytesSafe(original)
	buf[0] = 'C'

	if original != "copy me" {
		t.Errorf("修改副本不应影响原字符串，实际 string 变成了 %q", original)
	}
	if string(buf) != "Copy me" {
		t.Errorf("副本内容不对：%q", string(buf))
	}
}

// TestStringToBytesUnsafeSharesMemory 验证零拷贝转换确实共享底层数组。
//
// 测试只读取结果，绝不会写入——写入是未定义行为，见正文的真实报错。
func TestStringToBytesUnsafeSharesMemory(t *testing.T) {
	const sample = "zero copy"

	view := stringToBytesUnsafe(sample)
	if string(view) != sample {
		t.Fatalf("零拷贝结果内容不对：%q", string(view))
	}

	shared := unsafe.Pointer(unsafe.StringData(sample)) == unsafe.Pointer(unsafe.SliceData(view))
	if !shared {
		t.Error("零拷贝转换应当与源字符串共享底层数组，实际拿到的是副本")
	}
	if got := stringToBytesUnsafe(""); got != nil {
		t.Errorf("空字符串应当返回 nil，实际返回 %v", got)
	}
}

// TestSyncPoolReusesObjects 验证 Pool 取还的对象会被复用，而不是每次新建。
//
// 这里数的是 New 被调用的次数，而不是堆分配次数：Get/Put 的参数类型是 any，
// 每次取还都会把切片头装箱，本身就会产生一次分配，用它衡量复用会看错重点。
func TestSyncPoolReusesObjects(t *testing.T) {
	// Pool 里的对象会被 GC 清空，测量期间关掉 GC 才能得到稳定的结果。
	previous := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(previous)

	// 对象会优先回到当前 P 的本地池，固定单 P 才能保证「只新建一个」。
	previousProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previousProcs)

	const runs = 1000
	created := 0
	pool := &sync.Pool{
		New: func() any {
			created++
			return make([]byte, 1024)
		},
	}

	for i := 0; i < runs; i++ {
		buf := pool.Get().([]byte)
		pool.Put(buf)
	}

	// 正常构建下 New 只会被调用一次；竞态构建（-race）下 sync.Pool.Put 会
	// 随机丢弃约 1/4 的对象（见 sync/pool.go 的 "Randomly drop x on floor"），
	// 所以这里只断言「远少于请求次数」。
	if created >= runs/2 {
		t.Errorf("%d 次取还应当复用绝大多数对象，实际新建 %d 个", runs, created)
	}
}

// TestCounterImplementationsAgree 验证三种计数写法得到同一个结果。
func TestCounterImplementationsAgree(t *testing.T) {
	const goroutines, perGoroutine = 32, 50
	const want = int64(goroutines * perGoroutine)

	if got := int64(countWithMutex(goroutines, perGoroutine)); got != want {
		t.Errorf("互斥锁结果不对：期望 %d，实际 %d", want, got)
	}
	if got := countWithAtomic(goroutines, perGoroutine); got != want {
		t.Errorf("原子操作结果不对：期望 %d，实际 %d", want, got)
	}
	if got := int64(countWithShards(goroutines, perGoroutine)); got != want {
		t.Errorf("分段锁结果不对：期望 %d，实际 %d", want, got)
	}
}

// TestShardCounterSplitsKeys 验证不同 key 落到不同分段，同一 key 稳定落到同一段。
func TestShardCounterSplitsKeys(t *testing.T) {
	var counter shardCounter
	for key := 0; key < len(counter.shards); key++ {
		counter.add(key)
		counter.add(key + len(counter.shards))
	}

	if got := counter.total(); got != 2*len(counter.shards) {
		t.Fatalf("总数不对：期望 %d，实际 %d", 2*len(counter.shards), got)
	}
	for i := range counter.shards {
		if counter.shards[i].n != 2 {
			t.Errorf("第 %d 段应当收到 2 次累加，实际 %d", i, counter.shards[i].n)
		}
	}
}

// TestEscapeReturnValueDoesNotAllocate 验证按值返回不产生堆分配，按指针返回必然分配。
func TestEscapeReturnValueDoesNotAllocate(t *testing.T) {
	value := measureAllocs(50, func() {
		sinkInt = returnValue()
	})
	if value.allocsPerRun != 0 {
		t.Errorf("按值返回不应分配，实际 %.1f 次/op", value.allocsPerRun)
	}

	pointer := measureAllocs(50, func() {
		sinkIntPtr = returnPointer()
	})
	if pointer.allocsPerRun < 1 {
		t.Errorf("返回指针应当逃逸到堆，实际 %.1f 次/op", pointer.allocsPerRun)
	}
	if *sinkIntPtr != 42 {
		t.Errorf("逃逸后的值不对：期望 42，实际 %d", *sinkIntPtr)
	}
}

// TestProfileNames 验证运行时至少注册了正文提到的那几种 profile。
func TestProfileNames(t *testing.T) {
	names := profileNames()
	want := []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"}

	for _, name := range want {
		found := false
		for _, got := range names {
			if got == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("运行时没有注册 %s profile，实际类型：%v", name, names)
		}
	}
}
