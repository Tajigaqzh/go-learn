# 第 19 章 · 测试、基准与代码质量

前面的章节把 Go 的语法和标准库讲完了，但工程代码除了能跑起来，还得「证明它是对的」。Go 把测试提到了工具链一等公民的位置：`go test` 内置在编译器旁边、测试文件和源码放在同一个目录、覆盖率和基准直接支持，不需要装额外的框架就能写出可维护的测试。

本章讲 `testing` 包的完整用法：表驱动测试、子测试、并发测试、基准、模糊测试、示例测试、覆盖率，以及怎么用 `httptest` 打桩 HTTP、怎么用接口隔离外部依赖。最后简单过一遍静态检查工具（`go vet`、`staticcheck`、`golangci-lint`），让你在提交前就能发现大部分低级错误。

本章配套代码在 `internal/chapter/go19_testing/`，执行 `go run ./cmd/go-learn` 可以看到全部输出；执行 `go test ./internal/chapter/go19_testing/` 可以跑所有测试。

## 19.1 测试基础与表驱动测试

Go 的测试文件命名为 `*_test.go`，和被测代码放在同一个包目录下。测试函数签名是 `func TestXxx(t *testing.T)`，以 `Test` 开头、驼峰命名。

**最小可运行示例**：

```go
// add.go
package math

func Add(a, b int) int {
	return a + b
}

// add_test.go
package math

import "testing"

func TestAdd(t *testing.T) {
	got := Add(2, 3)
	want := 5
	if got != want {
		t.Errorf("Add(2, 3) = %d, want %d", got, want)
	}
}
```

**实测输出**：

```
$ go test -v
=== RUN   TestAdd
--- PASS: TestAdd (0.00s)
PASS
ok      example/math    0.002s
```

`t.Errorf` 报告失败但继续执行后续测试；`t.Fatalf` 报告失败并立即停止。

### 表驱动测试（Table-Driven Tests）

一个函数通常有多组输入和期望输出。Go 习惯用「表驱动」模式：定义一个测试用例表，循环执行。

```go
func TestAdd(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"正数", 2, 3, 5},
		{"负数", -1, -2, -3},
		{"零", 0, 0, 0},
		{"混合", -5, 10, 5},
	}
	for _, tt := range tests {
		got := Add(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("%s: Add(%d, %d) = %d, want %d", tt.name, tt.a, tt.b, got, tt.want)
		}
	}
}
```

这样一个测试函数覆盖多个场景，加新用例只需要往表里加一行。

## 19.2 子测试与 `t.Run`

表驱动测试的缺点是：一旦某个用例失败，错误消息里要靠 `name` 字段区分。Go 1.7 引入了 **子测试（Subtests）**：

```go
func TestAdd(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"正数", 2, 3, 5},
		{"负数", -1, -2, -3},
		{"零", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Add(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("Add(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
```

**实测输出**：

```
$ go test -v
=== RUN   TestAdd
=== RUN   TestAdd/正数
=== RUN   TestAdd/负数
=== RUN   TestAdd/零
--- PASS: TestAdd (0.00s)
    --- PASS: TestAdd/正数 (0.00s)
    --- PASS: TestAdd/负数 (0.00s)
    --- PASS: TestAdd/零 (0.00s)
PASS
```

每个子测试在输出里单独一行，失败时能精确定位。还可以用 `-run` 只跑某个子测试：

```bash
go test -v -run TestAdd/正数
```

子测试的名字会做 URL 转义（空格变 `%20`），命令行传参时要加引号或转义。

## 19.3 并发测试与 `t.Parallel`

子测试默认串行执行，但很多测试之间没有依赖、可以并发跑来节省时间。调用 `t.Parallel()` 标记当前测试可以并发：

```go
func TestConcurrent(t *testing.T) {
	t.Run("A", func(t *testing.T) {
		t.Parallel()
		time.Sleep(100 * time.Millisecond)
		// 测试逻辑
	})
	t.Run("B", func(t *testing.T) {
		t.Parallel()
		time.Sleep(100 * time.Millisecond)
	})
	t.Run("C", func(t *testing.T) {
		t.Parallel()
		time.Sleep(100 * time.Millisecond)
	})
}
```

三个子测试会并发执行，总耗时约 100ms 而不是 300ms。

**注意**：

- `t.Parallel()` 只对同一个父测试下的子测试生效；不同顶层测试函数之间本来就会并发。
- 并发测试要避免共享状态（全局变量、文件、数据库），否则容易 data race。

## 19.4 `t.Helper`、`t.Cleanup`、`t.TempDir`

### `t.Helper`

把重复的断言逻辑抽成辅助函数时，调用 `t.Helper()` 可以让失败报告显示调用方的行号，而不是辅助函数内部：

```go
func assertEqual(t *testing.T, got, want int) {
	t.Helper() // 标记这是辅助函数
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestSomething(t *testing.T) {
	assertEqual(t, Add(2, 3), 5) // 失败时报告这一行，不是 assertEqual 内部
}
```

### `t.Cleanup`

注册一个函数，在测试结束时自动执行（类似 `defer`，但作用域是整个测试函数）：

```go
func TestWithCleanup(t *testing.T) {
	f, err := os.Create("temp.txt")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		f.Close()
		os.Remove("temp.txt")
	})
	// 使用 f 做测试...
}
```

即使测试失败或 panic，`Cleanup` 注册的函数也会执行。多个 `Cleanup` 按 LIFO 顺序调用（和 `defer` 一样）。

### `t.TempDir`

自动创建临时目录，测试结束时自动删除：

```go
func TestTempDir(t *testing.T) {
	dir := t.TempDir() // 返回路径，测试结束自动删除
	// 在 dir 里创建文件...
}
```

比 `os.MkdirTemp` + 手动删除方便，也不会忘记清理。

## 19.5 基准测试（Benchmark）

基准测试用来测量代码性能，函数签名是 `func BenchmarkXxx(b *testing.B)`，循环执行 `b.N` 次：

```go
func BenchmarkAdd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Add(2, 3)
	}
}
```

**实测输出**：

```
$ go test -bench=.
goos: linux
goarch: amd64
pkg: example/math
BenchmarkAdd-8   	1000000000	         0.25 ns/op
PASS
```

- `BenchmarkAdd-8`：函数名 + GOMAXPROCS
- `1000000000`：总共执行了 10 亿次
- `0.25 ns/op`：平均每次操作耗时 0.25 纳秒

Go 会自动调整 `b.N` 直到结果稳定（通常跑 1 秒以上）。

### `-benchmem`：查看内存分配

加 `-benchmem` 可以看每次操作分配了多少内存、分配了几次：

```
$ go test -bench=. -benchmem
BenchmarkAdd-8   	1000000000	         0.25 ns/op	       0 B/op	       0 allocs/op
```

- `0 B/op`：每次操作分配 0 字节
- `0 allocs/op`：每次操作分配 0 次

这对优化很有用：比如你把 `[]byte` 改成 `strings.Builder`，可以直接看到 `allocs/op` 是否降低。

### 排除准备时间：`b.ResetTimer`

如果基准前有准备逻辑（比如加载数据），不应计入耗时，用 `b.ResetTimer()` 重置计时器：

```go
func BenchmarkProcess(b *testing.B) {
	data := loadHugeData() // 准备数据
	b.ResetTimer()         // 重置计时器
	for i := 0; i < b.N; i++ {
		process(data)
	}
}
```

### 子基准测试

和 `t.Run` 类似，基准也可以用 `b.Run` 分组：

```go
func BenchmarkFib(b *testing.B) {
	for _, n := range []int{10, 20, 30} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				Fib(n)
			}
		})
	}
}
```

## 19.6 模糊测试（Fuzzing，Go 1.18+）

模糊测试（fuzz testing）会自动生成随机输入，找出代码的边界 bug。函数签名是 `func FuzzXxx(f *testing.F)`：

```go
func FuzzReverse(f *testing.F) {
	// 种子语料库（seed corpus）
	f.Add("hello")
	f.Add("世界")

	f.Fuzz(func(t *testing.T, s string) {
		rev := Reverse(s)
		revRev := Reverse(rev)
		if s != revRev {
			t.Errorf("Reverse(Reverse(%q)) != %q, got %q", s, s, revRev)
		}
	})
}
```

**运行模糊测试**：

```bash
go test -fuzz=FuzzReverse -fuzztime=10s
```

- `-fuzz=<pattern>`：指定要跑的模糊测试
- `-fuzztime=10s`：跑 10 秒（默认一直跑到发现问题或手动停止）

Go 会基于种子语料库生成大量随机输入，找到触发 panic 或断言失败的输入时会报告并保存到 `testdata/fuzz/` 目录，下次跑测试会自动回归这些 case。

**注意**：模糊测试只支持简单类型（`string`、`[]byte`、`int`、`bool` 等），不能直接用结构体。

## 19.7 示例测试（Example）

示例测试既是测试也是文档，函数签名是 `func ExampleXxx()`，用注释 `// Output:` 标记期望输出：

```go
func ExampleAdd() {
	fmt.Println(Add(2, 3))
	// Output: 5
}
```

运行 `go test` 时会检查实际输出是否和 `// Output:` 匹配；`go doc` 和 godoc 工具会把示例显示在文档里。

示例测试的命名规则：

- `Example()`：包级别示例
- `ExampleF()`：函数 `F` 的示例
- `ExampleT()`：类型 `T` 的示例
- `ExampleT_M()`：类型 `T` 的方法 `M` 的示例
- `ExampleF_suffix()`：同一个函数的多个示例，用后缀区分

## 19.8 覆盖率（Coverage）

覆盖率衡量测试执行了多少代码。Go 内置覆盖率统计：

```bash
go test -cover
```

**实测输出**：

```
PASS
coverage: 85.7% of statements
ok      example/math    0.003s
```

### 详细覆盖率报告

生成覆盖率 profile 并用浏览器查看哪些行未覆盖：

```bash
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

浏览器会打开一个 HTML 页面，绿色是执行过的行、红色是未执行的行。

### 跨包覆盖率

默认 `go test -cover` 只统计当前包，要统计所有依赖包：

```bash
go test -coverpkg=./... -coverprofile=coverage.out ./...
```

## 19.9 HTTP 测试：`httptest`

测试 HTTP handler 时不需要真的启动服务器，用 `httptest.NewRecorder` 模拟 `ResponseWriter`：

```go
func TestHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/hello", nil)
	w := httptest.NewRecorder()

	HelloHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Hello, World!" {
		t.Errorf("body = %q, want %q", body, "Hello, World!")
	}
}
```

### `httptest.NewServer`：模拟服务端

如果要测试 HTTP 客户端，用 `httptest.NewServer` 启动一个临时服务器：

```go
func TestClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "mock response")
	}))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "mock") {
		t.Errorf("unexpected body: %s", body)
	}
}
```

`ts.URL` 是临时服务器的地址（如 `http://127.0.0.1:xxxxx`），测试结束后 `defer ts.Close()` 会自动关闭。

## 19.10 接口打桩与依赖注入

真实代码通常依赖外部服务（数据库、API、文件系统）。测试时不应该真的调用它们，而是用 **打桩（stub/mock）** 替换。

Go 的习惯做法是 **面向接口编程 + 依赖注入**：

```go
// 定义接口
type UserStore interface {
	GetUser(id int) (*User, error)
}

// 真实实现
type DBUserStore struct {
	db *sql.DB
}

func (s *DBUserStore) GetUser(id int) (*User, error) {
	// 查询数据库...
}

// 业务逻辑依赖接口，不依赖具体实现
type UserService struct {
	store UserStore
}

func (s *UserService) GetUserName(id int) (string, error) {
	user, err := s.store.GetUser(id)
	if err != nil {
		return "", err
	}
	return user.Name, nil
}
```

测试时注入 mock 实现：

```go
// 测试用的 mock
type MockUserStore struct {
	users map[int]*User
}

func (m *MockUserStore) GetUser(id int) (*User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return user, nil
}

func TestUserService(t *testing.T) {
	mock := &MockUserStore{
		users: map[int]*User{
			1: {ID: 1, Name: "Alice"},
		},
	}
	svc := &UserService{store: mock}

	name, err := svc.GetUserName(1)
	if err != nil {
		t.Fatal(err)
	}
	if name != "Alice" {
		t.Errorf("name = %q, want Alice", name)
	}
}
```

这样测试不依赖真实数据库，跑得快、不需要环境准备、结果可预测。

## 19.11 静态检查工具

测试能发现运行时错误，但编译器无法发现所有逻辑错误。Go 社区有一套静态分析工具：

### `go vet`

内置工具，检查常见错误（未使用的赋值、`Printf` 格式不匹配、锁复制等）：

```bash
go vet ./...
```

### `staticcheck`

比 `go vet` 更全面的检查工具：

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```

能发现：未使用的代码、不必要的类型断言、字符串格式错误、并发问题等。

### `golangci-lint`

集成了几十种 linter 的工具，可配置启用哪些规则：

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
golangci-lint run
```

配置文件 `.golangci.yml`：

```yaml
linters:
  enable:
    - gofmt
    - goimports
    - govet
    - staticcheck
    - errcheck
    - unused
    - ineffassign
```

## 19.12 六个真实报错怎么读

### 报错 1：测试函数名不以 `Test` 开头

```go
func testAdd(t *testing.T) { // 小写开头
	// ...
}
```

**错误现象**：`go test` 不会执行这个函数，也不报错。

**原因**：测试函数必须以 `Test` 开头且大写，否则不会被识别为测试。

**修复**：改成 `func TestAdd(t *testing.T)`。

### 报错 2：表驱动测试中循环变量捕获问题

```go
for _, tt := range tests {
	t.Run(tt.name, func(t *testing.T) {
		t.Parallel()
		// 使用 tt.a, tt.b ...
	})
}
```

**错误现象**：Go 1.21 前所有并发子测试都看到最后一个 `tt` 的值。

**原因**：闭包捕获循环变量 `tt`，并发执行时循环已结束，所有子测试共享最后一个值。

**修复**：Go 1.22+ 修复了这个问题（循环变量每次迭代都是新的）；Go 1.21 及以下需要手动复制：

```go
for _, tt := range tests {
	tt := tt // 复制一份
	t.Run(tt.name, func(t *testing.T) {
		t.Parallel()
		// 使用 tt ...
	})
}
```

### 报错 3：基准测试中编译器优化掉了被测代码

```go
func BenchmarkAdd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Add(2, 3) // 结果未使用，可能被优化掉
	}
}
```

**错误现象**：耗时异常低（如 0.01 ns/op），不符合预期。

**原因**：编译器发现结果未使用，可能优化掉整个计算。

**修复**：把结果赋值给包级变量（防止优化）：

```go
var result int

func BenchmarkAdd(b *testing.B) {
	var r int
	for i := 0; i < b.N; i++ {
		r = Add(2, 3)
	}
	result = r // 防止优化
}
```

### 报错 4：`t.Fatal` 在 goroutine 中调用导致 panic

```go
func TestConcurrent(t *testing.T) {
	go func() {
		t.Fatal("error") // panic: testing: t.Fatal() called from non-test goroutine
	}()
}
```

**错误现象**：panic，提示 `t.Fatal()` 不能在非测试 goroutine 中调用。

**原因**：`t.Fatal` 调用 `runtime.Goexit()` 退出当前 goroutine，只能在测试主 goroutine 中安全调用。

**修复**：用 channel 或 `t.Error` + 等待：

```go
func TestConcurrent(t *testing.T) {
	errCh := make(chan error, 1)
	go func() {
		if err := doSomething(); err != nil {
			errCh <- err
		}
		close(errCh)
	}()
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}
```

### 报错 5：模糊测试中使用了不支持的类型

```go
func FuzzProcess(f *testing.F) {
	f.Fuzz(func(t *testing.T, data MyStruct) { // 不支持自定义结构体
		// ...
	})
}
```

**错误消息**：`unsupported type: MyStruct`

**原因**：模糊测试只支持 `string`、`[]byte`、整数、`bool` 等基本类型。

**修复**：用 `[]byte` 序列化结构体，或拆成多个基本类型参数。

### 报错 6：忘记调用 `b.ResetTimer` 导致基准包含准备时间

```go
func BenchmarkProcess(b *testing.B) {
	data := loadHugeData() // 耗时操作
	for i := 0; i < b.N; i++ {
		process(data)
	}
}
```

**错误现象**：第一次迭代耗时远超后续迭代，导致 `b.N` 被低估、总耗时不稳定。

**原因**：`loadHugeData()` 的耗时被计入每次操作的平均值。

**修复**：准备完数据后调用 `b.ResetTimer()`。

## 19.13 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 测试函数不执行 | 函数名不以 `Test` 开头或未导出 | 改成 `func TestXxx(t *testing.T)` |
| 并发子测试看到相同的循环变量值 | 闭包捕获循环变量（Go 1.21 及以下） | 在 `t.Run` 前加 `tt := tt` |
| 基准耗时异常低 | 编译器优化掉了未使用的结果 | 把结果赋值给包级变量 |
| `t.Fatal` 在 goroutine 中 panic | `t.Fatal` 只能在测试主 goroutine 调用 | 用 channel 传递错误，或用 `t.Error` |
| 模糊测试报 unsupported type | 参数类型不是基本类型 | 改用 `[]byte`、`string`、`int` 等 |
| 基准包含准备时间 | 未调用 `b.ResetTimer()` | 准备完数据后重置计时器 |
| 覆盖率不包含依赖包 | 默认只统计当前包 | `go test -coverpkg=./... ./...` |
| 示例测试不执行 | `// Output:` 注释缺失或格式错误 | 确保注释紧跟函数、格式完全匹配 |

## 19.14 练习

1. **表驱动测试练习**：为第 5 章的 `Fibonacci` 函数写表驱动测试，覆盖 `n=0, 1, 2, 10` 四种情况，用 `t.Run` 分成子测试。

2. **基准与内存分析**：为第 8 章的字符串拼接函数（`+`、`fmt.Sprintf`、`strings.Builder`）分别写基准测试，用 `-benchmem` 对比内存分配。

3. **HTTP 打桩**：写一个函数 `FetchUserName(url string) (string, error)`，从 HTTP API 获取用户名；用 `httptest.NewServer` 模拟服务端，写测试验证函数能正确解析 JSON 响应。

4. **接口打桩**：定义接口 `EmailSender interface { Send(to, body string) error }`，写一个 `NotifyUser` 函数依赖这个接口；测试时用 mock 实现验证 `Send` 被调用且参数正确。

5. **模糊测试**：为 `strings.Split` 写一个模糊测试，验证 `strings.Join(strings.Split(s, sep), sep)` 是否总能还原原始字符串（提示：会发现边界 case）。

6. **覆盖率实战**：为第 11 章的错误包装函数写测试，用 `-cover` 检查覆盖率，确保所有 `if err != nil` 分支都被覆盖。

::: details 第 1 题参考答案

```go
func TestFibonacci(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want int
	}{
		{"F(0)", 0, 0},
		{"F(1)", 1, 1},
		{"F(2)", 2, 1},
		{"F(10)", 10, 55},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Fibonacci(tt.n)
			if got != tt.want {
				t.Errorf("Fibonacci(%d) = %d, want %d", tt.n, got, tt.want)
			}
		})
	}
}
```

**为什么这样写更好**：`t.Run` 让每个用例的失败独立显示，`-run` 可以单独跑某个用例；表结构让加新 case 只需加一行，不需要复制粘贴断言逻辑。
:::

::: details 第 2 题参考答案

```go
func BenchmarkConcat(b *testing.B) {
	strs := []string{"hello", "world", "go", "test"}
	b.Run("plus", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = strs[0] + strs[1] + strs[2] + strs[3]
		}
	})
	b.Run("sprintf", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = fmt.Sprintf("%s%s%s%s", strs[0], strs[1], strs[2], strs[3])
		}
	})
	b.Run("builder", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var sb strings.Builder
			for _, s := range strs {
				sb.WriteString(s)
			}
			_ = sb.String()
		}
	})
}
```

运行 `go test -bench=BenchmarkConcat -benchmem` 会看到 `builder` 的 `allocs/op` 最低（预分配容量时可以降到 1 次）。

**为什么这样写更好**：用 `b.Run` 分组让三种方法的结果并排对比，`-benchmem` 直接看到内存差异，比猜测或凭感觉选方案可靠。
:::

::: details 第 3 题参考答案

```go
func FetchUserName(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Name, nil
}

func TestFetchUserName(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"name":"Alice"}`)
	}))
	defer ts.Close()

	name, err := FetchUserName(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if name != "Alice" {
		t.Errorf("name = %q, want Alice", name)
	}
}
```

**为什么这样写更好**：`httptest.NewServer` 让测试不依赖外部 API，跑得快、结果可控；测试覆盖了 JSON 解析和错误处理路径。
:::

::: details 第 4 题参考答案

```go
type EmailSender interface {
	Send(to, body string) error
}

func NotifyUser(sender EmailSender, email, message string) error {
	return sender.Send(email, message)
}

type MockEmailSender struct {
	calls []struct{ to, body string }
}

func (m *MockEmailSender) Send(to, body string) error {
	m.calls = append(m.calls, struct{ to, body string }{to, body})
	return nil
}

func TestNotifyUser(t *testing.T) {
	mock := &MockEmailSender
	err := NotifyUser(mock, "user@example.com", "Hello")
	if err != nil {
		t.Fatal(err)
	}
	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.calls))
	}
	if mock.calls[0].to != "user@example.com" || mock.calls[0].body != "Hello" {
		t.Errorf("unexpected call: %+v", mock.calls[0])
	}
}
```

**为什么这样写更好**：接口让业务逻辑和邮件发送解耦，测试时 mock 不会真的发邮件，还能验证调用参数；生产代码注入真实的 SMTP 实现即可。
:::

::: details 第 5 题参考答案

```go
func FuzzSplitJoin(f *testing.F) {
	f.Add("a,b,c", ",")
	f.Add("hello", " ")
	f.Fuzz(func(t *testing.T, s, sep string) {
		parts := strings.Split(s, sep)
		joined := strings.Join(parts, sep)
		if s != joined {
			t.Errorf("Split/Join not reversible: %q -> %q", s, joined)
		}
	})
}
```

**注意**：这个测试会发现反例，比如 `s=""`, `sep="x"`时 `Split` 返回 `[""]`，`Join` 得到 `""`，但原始是 `""`；或 `s="a"`, `sep="a"` 时 `Split` 返回 `["", ""]`，`Join` 得到 `"a"`，但原始是 `"a"`。这说明 `Split`/`Join` 不是所有情况下都互逆。

**为什么这样写更好**：模糊测试自动发现边界 case，人工列举容易漏掉；这个练习也展示了「看似对称的 API 其实有坑」。
:::

::: details 第 6 题参考答案

```go
// 被测函数（来自第 11 章）
func ProcessFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		return errors.New("empty file")
	}
	// 处理 data...
	return nil
}

// 测试
func TestProcessFile(t *testing.T) {
	// 成功路径
	t.Run("valid file", func(t *testing.T) {
		f := createTempFile(t, "content")
		if err := ProcessFile(f); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// 错误路径 1：文件不存在
	t.Run("not found", func(t *testing.T) {
		err := ProcessFile("/nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	// 错误路径 2：空文件
	t.Run("empty file", func(t *testing.T) {
		f := createTempFile(t, "")
		err := ProcessFile(f)
		if err == nil || err.Error() != "empty file" {
			t.Errorf("expected 'empty file' error, got %v", err)
		}
	})
}

func createTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.WriteString(content)
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}
```

运行 `go test -cover` 应该看到接近 100% 覆盖率。

**为什么这样写更好**：覆盖所有 `if err != nil` 分支确保错误处理逻辑被测试到；`t.Cleanup` 自动清理临时文件，避免污染测试环境。
:::

## 19.15 小结

- Go 的测试工具链内置在 `go test` 中，测试文件命名为 `*_test.go`，测试函数以 `Test` 开头。
- 表驱动测试用结构体切片组织用例，配合 `t.Run` 可以单独运行和报告每个子测试。
- `t.Parallel()` 标记子测试并发执行，节省时间；`t.Helper()` 让辅助函数的失败报告调用方行号。
- `t.Cleanup()` 注册清理函数，`t.TempDir()` 自动创建临时目录并在测试结束时删除。
- 基准测试用 `BenchmarkXxx(b *testing.B)` 测量性能，`-benchmem` 查看内存分配，`b.ResetTimer()` 排除准备时间。
- 模糊测试（Go 1.18+）用 `FuzzXxx(f *testing.F)` 自动生成随机输入，发现边界 bug。
- 示例测试用 `ExampleXxx()` 和 `// Output:` 注释验证输出，同时作为文档展示。
- `go test -cover` 统计覆盖率，`-coverprofile` 生成详细报告，`go tool cover -html` 可视化未覆盖的行。
- `httptest` 提供 `NewRecorder` 模拟 `ResponseWriter`、`NewServer` 模拟 HTTP 服务器，方便测试 HTTP handler 和客户端。
- 面向接口编程 + 依赖注入让测试可以用 mock 实现替换真实依赖（数据库、API、邮件等），不需要真实环境。
- `go vet` 检查常见错误，`staticcheck` 和 `golangci-lint` 提供更全面的静态分析，在提交前发现潜在问题。

下一章进入并发编程，讲 goroutine、channel、`select` 和 `sync` 包的基础用法。
