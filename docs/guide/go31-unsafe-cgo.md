# 第 31 章 · unsafe、cgo 与代码生成

在前面的章节里，我们学习了 Go 的核心语言特性、标准库和常用第三方生态。Go 以类型安全、内存安全和高层抽象著称，但在某些场景下——调用 C 库、优化性能热点、与底层硬件交互——你需要突破这些安全边界。

本章介绍 Go 的"逃生舱"：`unsafe` 包提供指针运算与内存布局操作，cgo 让 Go 与 C 互操作，`//go:generate` 和 `//go:embed` 简化代码生成与资源嵌入。你会学到：

- `unsafe.Sizeof` / `Offsetof` / `Alignof` 与内存对齐
- `unsafe.Pointer` 与 `uintptr` 的转换规则与陷阱
- `runtime.KeepAlive` 的作用与 GC 可见性
- cgo 类型映射（`C.int`、`*C.char`）与字符串转换
- `C.CString` 配 `C.free` 的内存管理
- `//export` 反向导出 Go 函数给 C 调用
- cgo 的代价、构建约束与 `CGO_ENABLED=0`
- `//go:generate` 与 `//go:embed` 的实用场景

**重要提示**：`unsafe` 与 cgo 破坏了 Go 的类型安全与可移植性。**只在必要时使用**（性能瓶颈、调用 C 库、与硬件交互），并且要清楚规则边界——错误使用会导致段错误、数据损坏或随机 crash。绝大多数 Go 代码不需要碰这些工具。

本章配套代码在 `internal/chapter/go31_unsafe_cgo/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。cgo 示例仅展示接口设计与规则，不实际引入 C 编译器依赖（避免仓库构建复杂化）。

---

## 31.1 unsafe 包的三个函数

`unsafe` 包提供三个函数，用于查询类型的内存布局：

| 函数 | 作用 | 返回值 |
| --- | --- | --- |
| `Sizeof(x)` | 返回类型 `x` 的大小（字节） | `uintptr` |
| `Offsetof(x.f)` | 返回字段 `f` 在结构体 `x` 中的偏移量 | `uintptr` |
| `Alignof(x)` | 返回类型 `x` 的对齐要求 | `uintptr` |

### 示例：查询结构体内存布局

```go
type Example struct {
    a bool   // 1 字节
    b int32  // 4 字节
    c int64  // 8 字节
    d string // 16 字节（指针 + 长度）
}

var e Example
fmt.Printf("Sizeof(Example) = %d\n", unsafe.Sizeof(e))       // 输出：32（有内存对齐）
fmt.Printf("Offsetof(e.a) = %d\n", unsafe.Offsetof(e.a))    // 输出：0
fmt.Printf("Offsetof(e.b) = %d\n", unsafe.Offsetof(e.b))    // 输出：4
fmt.Printf("Offsetof(e.c) = %d\n", unsafe.Offsetof(e.c))    // 输出：8
fmt.Printf("Offsetof(e.d) = %d\n", unsafe.Offsetof(e.d))    // 输出：16
fmt.Printf("Alignof(e.c) = %d\n", unsafe.Alignof(e.c))      // 输出：8（int64 要求 8 字节对齐）
```

**内存布局**：
```
偏移 0: a (bool, 1 字节)
偏移 1-3: 填充（对齐 b）
偏移 4-7: b (int32, 4 字节)
偏移 8-15: c (int64, 8 字节)
偏移 16-31: d (string, 16 字节)
总大小：32 字节
```

### 为什么结构体有内存对齐？

CPU 访问对齐的内存更快（某些架构下未对齐访问会 crash）。Go 自动插入填充字节，使每个字段的起始地址是其对齐要求的倍数。

**优化技巧**：把小字段放一起，大字段放一起，可以减少填充：

```go
// 浪费 7 字节填充
type Bad struct {
    a bool   // 1 + 7 填充
    b int64  // 8
    c bool   // 1 + 7 填充
}

// 只浪费 6 字节填充
type Good struct {
    b int64  // 8
    a bool   // 1
    c bool   // 1 + 6 填充
}
```

---

## 31.2 unsafe.Pointer 与 uintptr 转换规则

### 两种类型的区别

| 类型 | 本质 | GC 追踪 | 算术运算 |
| --- | --- | --- | --- |
| `unsafe.Pointer` | 通用指针类型 | 是（GC 知道它指向的对象） | 否 |
| `uintptr` | 整数类型 | 否（只是一个数字） | 是（可以 +/-） |

### 合法的转换模式

**模式 1：类型转换**
```go
var i int32 = 42
p := unsafe.Pointer(&i)    // *int32 -> unsafe.Pointer
f := (*float32)(p)         // unsafe.Pointer -> *float32
fmt.Println(*f)            // 按 float32 解释 int32 的字节（5.88e-44）
```

**模式 2：临时计算偏移**
```go
type Point struct{ x, y int32 }
p := Point{10, 20}
// 通过偏移访问字段 y
py := (*int32)(unsafe.Pointer(uintptr(unsafe.Pointer(&p)) + unsafe.Offsetof(p.y)))
*py = 30
fmt.Println(p.y)  // 输出：30
```

**关键规则**：`uintptr -> unsafe.Pointer` 的转换**必须在一个表达式里完成**，不能把 `uintptr` 存到变量再转回来：

```go
// 错误：对象可能被 GC 移动，ptr 指向无效内存
ptr := uintptr(unsafe.Pointer(&p))
time.Sleep(1 * time.Millisecond)  // GC 可能触发
py := (*int32)(unsafe.Pointer(ptr + unsafe.Offsetof(p.y)))  // 野指针！

// 正确：一行完成
py := (*int32)(unsafe.Pointer(uintptr(unsafe.Pointer(&p)) + unsafe.Offsetof(p.y)))
```

### 为什么不能保存 uintptr？

`uintptr` 只是一个数字，GC 不认为它是指针。如果你把 `uintptr` 存到变量，GC 可能会：
1. 移动对象（压缩堆）
2. 回收对象（认为无引用）

此时 `uintptr` 指向的内存已失效，再转回 `unsafe.Pointer` 就是野指针。

---

## 31.3 runtime.KeepAlive 的作用

### 问题场景

当你通过 `uintptr` 访问对象内存时，Go 可能会认为原对象已无引用，在 GC 时移动或回收它：

```go
func readField(obj *MyStruct) int {
    ptr := uintptr(unsafe.Pointer(obj))
    // 此时 obj 可能被 GC 回收（因为只有 ptr 引用，而 ptr 是整数）
    fieldPtr := (*int)(unsafe.Pointer(ptr + 8))
    return *fieldPtr  // 野指针！
}
```

### 解决方案：runtime.KeepAlive

在使用 `uintptr` 后立即调用 `runtime.KeepAlive(obj)`，告诉 GC 该对象在此之前必须保持有效：

```go
import "runtime"

func readField(obj *MyStruct) int {
    ptr := uintptr(unsafe.Pointer(obj))
    fieldPtr := (*int)(unsafe.Pointer(ptr + 8))
    value := *fieldPtr
    runtime.KeepAlive(obj)  // 保证 obj 在此之前不被回收
    return value
}
```

**原理**：`KeepAlive` 是一个空函数，但编译器会把它当作"使用了 obj"，阻止 GC 提前回收。

**适用场景**：
- 通过 `uintptr` 访问内存
- 调用 cgo 函数时传递 Go 对象指针
- 手动管理对象生命周期

---

## 31.4 cgo 类型映射

cgo 提供 C 与 Go 类型的互操作。常见类型映射：

| C 类型 | Go 类型 | 说明 |
| --- | --- | --- |
| `char` | `C.char` | 1 字节整数 |
| `int` | `C.int` | 平台相关（通常 4 字节） |
| `long` | `C.long` | 平台相关（32 位 4 字节，64 位 8 字节） |
| `float` | `C.float` | 4 字节浮点 |
| `double` | `C.double` | 8 字节浮点 |
| `void*` | `unsafe.Pointer` | 通用指针 |
| `char*` | `*C.char` | C 字符串（NUL 结尾） |

### 字符串转换

**Go string → C char***：
```go
s := "hello"
cs := C.CString(s)  // 分配 C 内存，复制字符串，追加 '\0'
defer C.free(unsafe.Pointer(cs))  // 必须手动释放
C.puts(cs)  // 调用 C 函数
```

**C char* → Go string**：
```go
cs := C.CString("world")
defer C.free(unsafe.Pointer(cs))
s := C.GoString(cs)  // 复制到 Go string
fmt.Println(s)
```

### 切片传递

Go 切片不能直接传给 C，需要传指针 + 长度：

```go
data := []byte{1, 2, 3, 4}
C.process(&data[0], C.int(len(data)))  // C 函数：void process(char* p, int len)
```

**注意**：切片底层数组可能被 GC 移动，调用 C 函数期间要用 `runtime.KeepAlive`：

```go
C.process(&data[0], C.int(len(data)))
runtime.KeepAlive(data)
```

---

## 31.5 C.CString 与内存管理

### C.CString 的内存模型

1. `C.CString(s)` 在 **C 堆**分配内存（不是 Go 堆）
2. 复制 Go 字符串内容，追加 NUL 终止符 `'\0'`
3. 返回 `*C.char` 指针
4. **Go GC 不管理这块内存**，使用完毕后必须调用 `C.free`

### 示例

```go
// import "C"
s := "hello, world"
cs := C.CString(s)
defer C.free(unsafe.Pointer(cs))  // 确保释放

// 调用 C 函数
C.printf(C.CString("%s\n"), cs)  // 错误：又分配了一个 C 字符串，没释放！

// 正确写法
formatCs := C.CString("%s\n")
defer C.free(unsafe.Pointer(formatCs))
C.printf(formatCs, cs)
```

### 常见错误

| 错误 | 后果 |
| --- | --- |
| 忘记 `C.free` | 内存泄漏（C 堆不断增长） |
| 使用已释放的 `cs` | 野指针（段错误、数据损坏） |
| 把 Go 字符串指针直接传给 C | 不安全（GC 可能移动） |
| 在循环里 `C.CString` 不释放 | 快速耗尽内存 |

---

## 31.6 //export 反向导出

`//export` 用于把 Go 函数导出给 C 调用（动态库、回调）。

### 示例

```go
package main

import "C"

//export Add
func Add(a, b C.int) C.int {
    return a + b
}

func main() {}
```

### 编译成动态库

```bash
go build -buildmode=c-shared -o libadd.so add.go
# 生成 libadd.so（Linux）或 libadd.dll（Windows）和 libadd.h
```

### C 代码调用

```c
#include "libadd.h"
#include <stdio.h>

int main() {
    int result = Add(10, 20);
    printf("10 + 20 = %d\n", result);
    return 0;
}
```

编译：
```bash
gcc -o test test.c -L. -ladd
LD_LIBRARY_PATH=. ./test
```

### 规则与限制

1. **参数和返回值必须是 C 类型**（`C.int`、`*C.char`、`unsafe.Pointer`）
2. **不能导出泛型函数**
3. **panic 不能穿透到 C**：导出函数内必须 `defer recover()`，否则 C 调用时 panic 会 crash
4. **线程安全**：C 可能在任意线程调用，Go 函数要考虑并发安全
5. **生命周期**：动态库加载时调用 `init()`，卸载时没有钩子（资源需手动管理）

### Python 调用示例（ctypes）

```python
from ctypes import cdll, c_int

lib = cdll.LoadLibrary('./libadd.so')
result = lib.Add(c_int(10), c_int(20))
print(f"10 + 20 = {result}")
```

---

## 31.7 cgo 的代价与构建约束

### cgo 的代价

| 代价 | 影响 |
| --- | --- |
| **编译慢** | 需要 C 编译器，交叉编译复杂（需要目标平台的 C 工具链） |
| **调用开销大** | Go ↔ C 切换约 100ns（纯 Go 函数调用 ~1ns） |
| **二进制体积大** | 静态链接 C 库（如 SQLite ~1MB） |
| **丧失纯 Go 优势** | 不能 `CGO_ENABLED=0` 交叉编译，部署需要 C 运行时 |

### 构建约束（build tags）

用 `// +build` 注释控制文件是否参与编译：

**cgo 版本**（`db_cgo.go`）：
```go
// +build cgo

package db

// #cgo LDFLAGS: -lsqlite3
// #include <sqlite3.h>
import "C"

func Open(path string) (*DB, error) {
    // 调用 C.sqlite3_open
}
```

**纯 Go 版本**（`db_pure.go`）：
```go
// +build !cgo

package db

func Open(path string) (*DB, error) {
    // 使用 modernc.org/sqlite（纯 Go 实现）
}
```

编译时选择：
```bash
CGO_ENABLED=1 go build  # 使用 cgo 版本
CGO_ENABLED=0 go build  # 使用纯 Go 版本
```

### 禁用 cgo 的场景

1. **交叉编译**：在 x86 上编译 ARM 二进制
   ```bash
   CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build
   ```
2. **Docker 多阶段构建**：避免 C 运行时依赖
   ```dockerfile
   FROM golang:1.21 AS builder
   ENV CGO_ENABLED=0
   RUN go build -o app

   FROM scratch
   COPY --from=builder /app /app
   CMD ["/app"]
   ```
3. **纯静态二进制**：不依赖 libc
   ```bash
   CGO_ENABLED=0 go build -ldflags '-s -w' -o app
   ```

---

## 31.8 //go:generate 与 //go:embed

### //go:generate 代码生成

在源文件顶部写：
```go
//go:generate stringer -type=Status
```

然后运行：
```bash
go generate ./...
```

自动生成 `status_string.go`。

### 常用生成工具

| 工具 | 作用 | 安装 |
| --- | --- | --- |
| `stringer` | 为枚举生成 `String()` 方法 | `go install golang.org/x/tools/cmd/stringer@latest` |
| `mockgen` | 生成 mock 接口（gomock） | `go install github.com/golang/mock/mockgen@latest` |
| `protoc-gen-go` | 从 `.proto` 生成 Go 代码 | `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest` |
| `wire` | 依赖注入代码生成 | `go install github.com/google/wire/cmd/wire@latest` |

### //go:embed 嵌入静态资源

Go 1.16+ 支持编译时嵌入文件：

```go
import _ "embed"

//go:embed version.txt
var version string  // 嵌入文件内容到字符串

//go:embed templates/*.html
var templates embed.FS  // 嵌入目录到虚拟文件系统
```

**使用示例**：
```go
html, _ := templates.ReadFile("templates/index.html")
fmt.Println(string(html))
```

**优点**：
- 编译时打包到二进制，无需外部文件
- 支持单个文件、目录、glob 匹配

**限制**：
- 只能嵌入同包或子目录文件
- 不能嵌入 `.` 或 `_` 开头的文件（除非显式指定 `all:templates`）
- 不能嵌入符号链接

---

## 真实报错与常见陷阱

### 报错 1：unsafe.Pointer 转换后段错误

```bash
panic: runtime error: invalid memory address or nil pointer dereference
```

**原因**：把 `uintptr` 存到变量，对象被 GC 移动后转回 `unsafe.Pointer`。

**解决**：
```go
// 错误
ptr := uintptr(unsafe.Pointer(&obj))
time.Sleep(1 * time.Millisecond)
p := (*int)(unsafe.Pointer(ptr))  // 野指针

// 正确
p := (*int)(unsafe.Pointer(uintptr(unsafe.Pointer(&obj)) + 8))
runtime.KeepAlive(&obj)
```

### 报错 2：cgo 内存泄漏

```bash
# 内存使用不断增长，OOM
```

**原因**：忘记 `C.free`。

**解决**：
```go
cs := C.CString(s)
defer C.free(unsafe.Pointer(cs))  // 确保释放
```

### 报错 3：cgo 编译失败

```bash
# github.com/mattn/go-sqlite3
cgo: C compiler "gcc" not found: exec: "gcc": executable file not found in $PATH
```

**原因**：cgo 需要 C 编译器。

**解决**：
- Linux: `apt install build-essential`
- macOS: `xcode-select --install`
- Windows: 安装 MinGW 或用纯 Go 库

### 报错 4：交叉编译失败

```bash
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build
# cgo: C compiler "aarch64-linux-gnu-gcc" not found
```

**原因**：交叉编译需要目标平台的 C 工具链。

**解决**：
1. 安装交叉工具链（复杂）
2. 改用 `CGO_ENABLED=0`（放弃 cgo）
3. 在目标平台上编译

### 报错 5：//export 函数 panic

```bash
# C 调用 Go 函数时 crash
```

**原因**：panic 穿透到 C。

**解决**：
```go
//export SafeFunc
func SafeFunc() C.int {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("panic: %v", r)
        }
    }()
    // ...
}
```

### 报错 6：//go:embed 找不到文件

```bash
pattern templates/*.html: no matching files found
```

**原因**：文件不在同包或子目录。

**解决**：
- 把文件移到包目录下
- 或者用 `//go:embed all:templates` 包含隐藏文件

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 段错误（SIGSEGV） | uintptr 转 unsafe.Pointer 时对象被移动 | 一行完成转换 + runtime.KeepAlive |
| 内存泄漏 | C.CString 不释放 | defer C.free(unsafe.Pointer(cs)) |
| cgo 编译慢 | 每次都重新编译 C 代码 | 用缓存（`go env GOCACHE`） |
| 交叉编译失败 | cgo 需要目标平台 C 工具链 | CGO_ENABLED=0 或在目标平台编译 |
| 二进制体积大 | 静态链接 C 库 | 动态链接或改用纯 Go 库 |
| //export 函数 crash | panic 穿透到 C | defer recover() |
| unsafe.Offsetof 返回错误值 | 结构体内存对齐改变了布局 | 用 unsafe.Offsetof 而非硬编码偏移 |
| go generate 不生成代码 | 忘记运行 go generate | 加到 Makefile 或 CI |
| //go:embed 找不到文件 | 文件在父目录或以 . 开头 | 移到包目录或用 all: |

---

## 练习

### 第 1 题

下面的代码有什么问题？如何修复？

```go
func readInt(p *MyStruct) int {
    addr := uintptr(unsafe.Pointer(p))
    time.Sleep(10 * time.Millisecond)
    return *(*int)(unsafe.Pointer(addr + 8))
}
```

::: details 第 1 题参考答案

**问题**：
1. 把 `uintptr` 存到变量 `addr`，GC 可能移动 `p`，导致 `addr` 指向无效内存。
2. 没有 `runtime.KeepAlive`。

**修复**：
```go
func readInt(p *MyStruct) int {
    value := *(*int)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + 8))
    runtime.KeepAlive(p)
    return value
}
```

**说明**：
- `uintptr -> unsafe.Pointer` 转换在一行完成
- `runtime.KeepAlive(p)` 保证 `p` 在此之前不被回收
:::

### 第 2 题

下面的 cgo 代码会内存泄漏吗？为什么？

```go
for i := 0; i < 1000; i++ {
    cs := C.CString("hello")
    C.puts(cs)
}
```

::: details 第 2 题参考答案

**会泄漏**。

**原因**：
- `C.CString` 在 C 堆分配内存，Go GC 不管理。
- 循环 1000 次，分配了 1000 个 C 字符串，都没释放。

**修复**：
```go
for i := 0; i < 1000; i++ {
    cs := C.CString("hello")
    C.puts(cs)
    C.free(unsafe.Pointer(cs))  // 立即释放
}
```

**更好的写法**（避免重复分配）：
```go
cs := C.CString("hello")
defer C.free(unsafe.Pointer(cs))
for i := 0; i < 1000; i++ {
    C.puts(cs)
}
```
:::

### 第 3 题

为什么下面的代码可能 crash？

```go
//export Callback
func Callback() {
    panic("oops")
}
```

::: details 第 3 题参考答案

**原因**：
- `//export` 导出的函数被 C 调用时，panic 会穿透到 C 层。
- C 不懂 Go 的 panic 机制，导致整个进程 crash。

**修复**：
```go
//export Callback
func Callback() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("panic recovered: %v", r)
        }
    }()
    panic("oops")  // 现在不会 crash
}
```

**规则**：所有 `//export` 函数都应该 `defer recover()`。
:::

### 第 4 题

`unsafe.Sizeof(s)` 返回多少字节？

```go
s := "hello, world"
fmt.Println(unsafe.Sizeof(s))
```

::: details 第 4 题参考答案

**答案**：**16 字节**（64 位平台）。

**说明**：
- Go 字符串的内部表示是 `struct { ptr *byte; len int }`
- 指针 8 字节 + 长度 8 字节 = 16 字节
- **注意**：`Sizeof` 返回的是字符串头的大小，不是字符串内容的长度
- 字符串内容长度用 `len(s)`
:::

### 第 5 题

下面的交叉编译命令会成功吗？为什么？

```bash
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build
```

::: details 第 5 题参考答案

**大概率失败**。

**原因**：
- `CGO_ENABLED=1` 表示启用 cgo
- 交叉编译 ARM64 需要 ARM64 的 C 工具链（`aarch64-linux-gnu-gcc`）
- 大多数开发机只有本机架构的 C 编译器

**解决方案**：
1. **禁用 cgo**（推荐）：
   ```bash
   CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build
   ```
2. **安装交叉工具链**（复杂）：
   ```bash
   apt install gcc-aarch64-linux-gnu
   CC=aarch64-linux-gnu-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build
   ```
3. **在目标平台上编译**（最简单）
:::

### 第 6 题

`//go:embed` 能嵌入父目录的文件吗？如何嵌入隐藏文件（`.gitignore`）？

::: details 第 6 题参考答案

**不能嵌入父目录文件**。

**原因**：
- `//go:embed` 只能嵌入同包或子目录文件
- 这是安全限制，避免意外嵌入敏感文件

**嵌入隐藏文件**：
```go
//go:embed all:config
var configFS embed.FS  // 包含 config/.gitignore
```

**说明**：
- 默认不嵌入 `.` 开头的文件
- `all:` 前缀包含隐藏文件
- 也可以显式指定：`//go:embed config/.gitignore`
:::

---

## 小结

本章介绍了 Go 的底层能力，核心要点：

1. **unsafe 包**：`Sizeof` / `Offsetof` / `Alignof` 查询内存布局，`unsafe.Pointer` 与 `uintptr` 转换要遵守规则。
2. **uintptr 陷阱**：`uintptr` 不是指针，GC 不追踪它；转换必须一行完成，使用后调用 `runtime.KeepAlive`。
3. **cgo 类型映射**：`C.int`、`*C.char` 等；字符串用 `C.CString` / `C.GoString`；切片传指针 + 长度。
4. **C.CString 内存管理**：分配在 C 堆，必须 `C.free`，否则泄漏。
5. **//export**：导出 Go 函数给 C 调用，参数和返回值必须是 C 类型，panic 必须 `defer recover()`。
6. **cgo 代价**：编译慢、调用开销大、丧失纯 Go 优势；优先用纯 Go 库。
7. **构建约束**：用 `// +build cgo` / `!cgo` 区分 cgo 和纯 Go 版本。
8. **//go:generate**：自动生成代码（stringer、mockgen、protobuf）。
9. **//go:embed**：编译时嵌入静态资源，无需外部文件。

**使用原则**：**只在必要时使用 unsafe 和 cgo**——性能瓶颈、调用 C 库、与硬件交互。绝大多数 Go 代码不需要碰这些工具。错误使用会导致段错误、数据损坏或随机 crash。

下一章将进入工程实践篇，讲解**项目结构、分层架构与可测试性设计**，把前面学到的技术整合成可维护的 Go 项目。
