package go08_strings

import (
	"strings"
	"testing"
)

// benchSink 接收基准测试的结果，防止编译器把「没人用」的调用优化掉。
var benchSink string

// benchParts 生成四种拼接写法共用的输入，保证比较的是同一份工作。
func benchParts() []string {
	parts := make([]string, 0, 200)
	for i := 0; i < 200; i++ {
		parts = append(parts, string(rune('a'+i%26)))
	}
	return parts
}

// BenchmarkConcatPlus 衡量循环里用 + 累加的代价。
func BenchmarkConcatPlus(b *testing.B) {
	parts := benchParts()
	b.ReportAllocs()
	for b.Loop() {
		benchSink = concatWithPlus(parts)
	}
}

// BenchmarkConcatJoin 衡量 strings.Join 的代价。
func BenchmarkConcatJoin(b *testing.B) {
	parts := benchParts()
	b.ReportAllocs()
	for b.Loop() {
		benchSink = strings.Join(parts, "")
	}
}

// BenchmarkConcatBuilder 衡量预分配容量后 strings.Builder 的代价。
func BenchmarkConcatBuilder(b *testing.B) {
	parts := benchParts()
	b.ReportAllocs()
	for b.Loop() {
		benchSink = concatWithBuilder(parts)
	}
}

// BenchmarkConcatBuffer 衡量 bytes.Buffer 的代价。
func BenchmarkConcatBuffer(b *testing.B) {
	parts := benchParts()
	b.ReportAllocs()
	for b.Loop() {
		benchSink = concatWithBuffer(parts)
	}
}

// BenchmarkTruncateBytes 衡量安全截断的开销（最坏情况回退 3 个字节）。
func BenchmarkTruncateBytes(b *testing.B) {
	s := "Go 学习笔记：字符串、字节与 Unicode"
	b.ReportAllocs()
	for b.Loop() {
		benchSink = TruncateBytes(s, 12)
	}
}
