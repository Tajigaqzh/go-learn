package go14_reflection

import (
	"encoding/json"
	"reflect"
	"testing"
)

// benchSink 接收基准结果，防止编译器把「没人用」的调用优化掉。
var (
	benchInt    int
	benchString string
	benchAny    any
)

// benchAccount 是基准测试共用的样例数据。
func benchAccount() Person {
	return Person{Name: "Ada", Age: 36, note: "hidden"}
}

// BenchmarkDirectField 直接读字段，作为其他几项的对照。
func BenchmarkDirectField(b *testing.B) {
	p := benchAccount()
	b.ReportAllocs()
	for b.Loop() {
		benchInt = p.Age
	}
}

// BenchmarkReflectFieldByIndex 用反射按字段下标取值。
func BenchmarkReflectFieldByIndex(b *testing.B) {
	v := reflect.ValueOf(benchAccount())
	b.ReportAllocs()
	for b.Loop() {
		benchInt = int(v.Field(1).Int())
	}
}

// BenchmarkReflectFieldByName 用反射按字段名字取值（比下标慢，因为要查名字）。
func BenchmarkReflectFieldByName(b *testing.B) {
	v := reflect.ValueOf(benchAccount())
	b.ReportAllocs()
	for b.Loop() {
		benchInt = int(v.FieldByName("Age").Int())
	}
}

// BenchmarkReflectInterface 把反射值还原成接口值。
func BenchmarkReflectInterface(b *testing.B) {
	v := reflect.ValueOf(benchAccount())
	b.ReportAllocs()
	for b.Loop() {
		benchAny = v.FieldByName("Name").Interface()
	}
}

// BenchmarkJSONMarshal 标准库的序列化。
func BenchmarkJSONMarshal(b *testing.B) {
	p := benchAccount()
	b.ReportAllocs()
	for b.Loop() {
		data, err := json.Marshal(p)
		if err != nil {
			b.Fatal(err)
		}
		benchString = string(data)
	}
}

// BenchmarkSimpleJSON 手写反射实现的序列化，用来对比「同样走反射」的开销。
func BenchmarkSimpleJSON(b *testing.B) {
	p := benchAccount()
	b.ReportAllocs()
	for b.Loop() {
		out, err := SimpleJSON(p)
		if err != nil {
			b.Fatal(err)
		}
		benchString = out
	}
}
