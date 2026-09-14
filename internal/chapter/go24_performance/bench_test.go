package go24_performance

import (
	"sync"
	"sync/atomic"
	"testing"
)

// 这些基准测试统一把结果写入包级 sink 变量。
// 少了这一步，编译器会把整段被测代码优化掉，跑出「0.09 ns/op」这种假数据。

// BenchmarkAppendNoPrealloc 测量不预分配时的追加开销。
func BenchmarkAppendNoPrealloc(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkSlice = appendIntsNoPrealloc(1000)
	}
}

// BenchmarkAppendPrealloc 测量预分配容量后的追加开销。
func BenchmarkAppendPrealloc(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkSlice = appendIntsPrealloc(1000)
	}
}

// BenchmarkMapNoHint 测量不带容量提示的 map 写入。
func BenchmarkMapNoHint(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkMap = mapWithoutHint(1000)
	}
}

// BenchmarkMapWithHint 测量带容量提示的 map 写入。
func BenchmarkMapWithHint(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkMap = mapWithHint(1000)
	}
}

// BenchmarkConcatPlus 测量 += 拼接 200 段字符串的开销。
func BenchmarkConcatPlus(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkString = concatWithPlus(200)
	}
}

// BenchmarkConcatBuilder 测量 strings.Builder 拼接 200 段字符串的开销。
func BenchmarkConcatBuilder(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkString = concatWithBuilder(200)
	}
}

// BenchmarkStringToBytesCopy 测量 []byte(s) 的拷贝开销。
func BenchmarkStringToBytesCopy(b *testing.B) {
	const sample = "hello world, this is a benchmark string for conversion"

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkBytes = stringToBytesSafe(sample)
	}
}

// BenchmarkStringToBytesUnsafe 测量零拷贝转换的开销。
func BenchmarkStringToBytesUnsafe(b *testing.B) {
	const sample = "hello world, this is a benchmark string for conversion"

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sinkBytes = stringToBytesUnsafe(sample)
	}
}

// BenchmarkCounterMutex 测量所有 goroutine 争抢同一把锁的开销。
func BenchmarkCounterMutex(b *testing.B) {
	var mu sync.Mutex
	total := 0

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			total++
			mu.Unlock()
		}
	})
	if total == 0 {
		b.Fatal("计数没有推进")
	}
}

// BenchmarkCounterAtomic 测量原子自增的开销。
func BenchmarkCounterAtomic(b *testing.B) {
	var counter atomic.Int64

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			counter.Add(1)
		}
	})
}

// BenchmarkCounterSharded 测量分段锁的开销，每个 goroutine 固定使用一个分段。
func BenchmarkCounterSharded(b *testing.B) {
	var counter shardCounter
	var next atomic.Int64

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		key := int(next.Add(1)) - 1
		for pb.Next() {
			counter.add(key)
		}
	})
	if counter.total() == 0 {
		b.Fatal("计数没有推进")
	}
}
