package go18_serde_config

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"testing"
)

// benchSink 接收基准结果，防止编译器把「没人用」的调用优化掉。
var (
	benchBytes []byte
	benchMap   map[string]any
)

// benchBook 是基准测试共用的样例数据。
func benchBook() Book {
	return Book{Title: "Go 语言", Author: "Ada", Price: 42.5}
}

// BenchmarkJSONMarshal 标准库 JSON 序列化。
func BenchmarkJSONMarshal(b *testing.B) {
	book := benchBook()
	b.ReportAllocs()
	for b.Loop() {
		data, err := json.Marshal(book)
		if err != nil {
			b.Fatal(err)
		}
		benchBytes = data
	}
}

// BenchmarkGobEncode gob 二进制编码，作为「紧凑格式」的对照。
func BenchmarkGobEncode(b *testing.B) {
	book := benchBook()
	b.ReportAllocs()
	for b.Loop() {
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(book); err != nil {
			b.Fatal(err)
		}
		benchBytes = buf.Bytes()
	}
}

// BenchmarkParseSimpleYAML 手写 YAML 子集解析。
func BenchmarkParseSimpleYAML(b *testing.B) {
	const src = "app:\n  name: \"go-learn\"\n  debug: true\n  port: 8080\ndatabase:\n  host: 127.0.0.1\n  max_open: 20\n"
	b.ReportAllocs()
	for b.Loop() {
		parsed, err := ParseSimpleYAML(src)
		if err != nil {
			b.Fatal(err)
		}
		benchMap = parsed
	}
}

// BenchmarkLoadConfig 三层合并的开销（默认值 + 文件 + 环境变量）。
func BenchmarkLoadConfig(b *testing.B) {
	defaults := DefaultConfig()
	const fileJSON = `{"host":"0.0.0.0","port":9000,"log_level":"debug"}`
	env := func(key string) (string, bool) {
		if key == "APP_PORT" {
			return "7000", true
		}
		return "", false
	}
	b.ReportAllocs()
	for b.Loop() {
		cfg, err := LoadConfig(defaults, fileJSON, env)
		if err != nil {
			b.Fatal(err)
		}
		benchMap = map[string]any{"port": cfg.Port}
	}
}
