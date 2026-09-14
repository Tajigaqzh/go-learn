# 第 17 章 · 文件、路径与 IO

上一章我们梳理了 `time`、`math`、`sort` 等标准库；从本章开始，连续几章都会聚焦「写真实程序时天天打交道」的标准库能力。IO 与文件操作是服务端、CLI 工具、数据处理程序的基石：配置文件怎么读、日志怎么写、上传的文件存哪里、怎么安全地覆盖旧文件——这些问题的答案都在 `io`、`os`、`path/filepath`、`bufio`、`embed` 等包里。

本章配套代码在 `internal/chapter/go17_files_io/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

## 17.1 io.Reader / io.Writer 接口与组合

Go 的 IO 设计哲学是**小接口、大组合**。几乎所有输入输出都围绕两个核心接口展开：

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}
```

只要实现了 `Read` 或 `Write`，就能接入标准库的庞大生态：`io.Copy`、`bufio.Scanner`、`http.Request.Body`、`json.Decoder`……它们要求的不是具体类型，而是行为。

### 把字符串当作 Reader

```go
r := strings.NewReader("Hello, io.Reader!")
data, err := io.ReadAll(r)
```

`strings.NewReader` 返回一个 `*strings.Reader`，它实现了 `io.Reader`、`io.ReaderAt`、`io.Seeker`、`io.WriterTo` 等接口，可以当作文件一样读取。

### bytes.Buffer 兼顾读和写

```go
var buf bytes.Buffer
buf.WriteString("Hello, ")
buf.WriteString("io.Writer!")
fmt.Println(buf.String())
```

`bytes.Buffer` 同时实现了 `io.Reader` 和 `io.Writer`，是单元测试里模拟输入输出的常客。

### io.MultiWriter：一份数据，多个目的地

```go
var buf1, buf2 bytes.Buffer
mw := io.MultiWriter(&buf1, &buf2)
fmt.Fprint(mw, "同时写入两个 buffer")
```

`io.MultiWriter` 把多个 `Writer` 包装成一个：每次写入会同步写给所有底层 Writer。这在「既要写日志文件，又要输出到控制台」的场景里非常有用。

### 自定义 Reader

```go
type countReader struct {
    io.Reader
    n int
}

func (c *countReader) Read(p []byte) (int, error) {
    n, err := c.Reader.Read(p)
    c.n += n
    return n, err
}
```

通过**嵌入接口**并覆盖特定方法，可以在不破坏原有行为的前提下增强功能。这是 Go 里装饰器模式的惯用写法。

**实测输出（ Windows 环境，路径分隔符为 `\`）：**

```text
从 strings.NewReader 读取: "Hello, io.Reader!"
bytes.Buffer 内容: "Hello, io.Writer!"
buf1: "同时写入两个 buffer", buf2: "同时写入两个 buffer"
自定义 Reader 共读取 6 字节
```

## 17.2 os 文件读写：ReadFile、WriteFile 与 OpenFile

Go 1.16 引入了 `os.ReadFile` 和 `os.WriteFile`，让「一次性读完整文件」和「一次性写完整文件」不再需要手写 `Open` + `ReadAll` + `Close`。

### 一次性读写

```go
// 写
err := os.WriteFile("hello.txt", []byte("Hello\n"), 0o644)

// 读
data, err := os.ReadFile("hello.txt")
```

权限 `0o644` 表示：所有者读写，组和其他人只读。Go 支持 `0o` 前缀表示八进制（Go 1.13+）。

### 追加与精细化控制：OpenFile

```go
f, err := os.OpenFile("log.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
if err != nil { ... }
defer f.Close()

f.WriteString("新日志行\n")
```

常用标志位：

| 标志 | 含义 |
| --- | --- |
| `os.O_RDONLY` | 只读 |
| `os.O_WRONLY` | 只写 |
| `os.O_RDWR` | 读写 |
| `os.O_APPEND` | 追加 |
| `os.O_CREATE` | 不存在则创建 |
| `os.O_TRUNC` | 存在则截断 |
| `os.O_EXCL` | 与 CREATE 配合，文件已存在时返回错误 |

### defer Close 的陷阱

`defer f.Close()` 很方便，但有一个隐蔽问题：**`Close` 本身可能返回错误**。对于只读文件，直接 `defer` 通常无妨；但对于写操作（尤其是带缓存的写入），`Close` 可能把缓冲区里剩余的数据刷盘，此时磁盘满或网络文件系统断开都会报错。如果只用 `defer`，这个错误会被静默丢弃。

**推荐做法（写场景）：**

```go
func writeImportant(path string, data []byte) error {
    f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
    if err != nil {
        return err
    }

    if _, err := f.Write(data); err != nil {
        f.Close() // 注意：这里不能 defer，因为下面还要返回 Close 的错误
        return err
    }

    return f.Close() // 显式关闭，并把错误返回给调用方
}
```

**实测输出：**

```text
已写入文件: C:\Users\...\Temp\go17_demo_...\hello.txt
os.ReadFile 读取结果: "Hello, os.WriteFile!\n"
追加后内容:
Hello, os.WriteFile!
追加一行
```

## 17.3 bufio.Scanner 与行长限制

逐行读取大文件时，`bufio.Scanner` 是最顺手的选择：

```go
scanner := bufio.NewScanner(file)
for scanner.Scan() {
    line := scanner.Text()
    // 处理每一行
}
if err := scanner.Err(); err != nil {
    // 处理错误
}
```

### 64KB 行长限制

`Scanner` 的默认 `Split` 函数是 `ScanLines`，它把**单行长度限制在 64KB**（`bufio.MaxScanTokenSize`）。如果一行超过这个长度，`scanner.Err()` 会返回 `bufio.ErrTooLong`。

**解决方案：**

1. 用 `bufio.Reader.ReadLine`（可以处理任意长度，但需要手动拼接）
2. 用 `bufio.Reader.ReadString('\n')`
3. 自定义 `SplitFunc` 并调用 `scanner.Buffer` 扩大 token 容量

```go
scanner := bufio.NewScanner(file)
// 把容量扩展到 1MB，最大 token 也是 1MB
const maxCapacity = 1024 * 1024
buf := make([]byte, maxCapacity)
scanner.Buffer(buf, maxCapacity)
```

**实测输出：**

```text
  第1行: "第一行"
  第2行: "第二行"
  第3行: "第三行"
Scanner 默认最大行长度: 64KB（bufio.MaxScanTokenSize）
超长行应改用 bufio.Reader.ReadLine 或自定义 Split 函数
```

## 17.4 io.Copy、TeeReader 与 MultiReader

### io.Copy：流式复制

```go
n, err := io.Copy(dst, src)
```

`io.Copy` 会自动处理缓冲，适合大文件复制，不会因为文件太大而耗尽内存。返回的 `n` 是复制的字节数。

### io.TeeReader：边读边写

```go
var log bytes.Buffer
tr := io.TeeReader(src, &log)
data, _ := io.ReadAll(tr) // 读取 src 的同时，内容也会写入 log
```

常用于调试或审计：你需要消费 `Reader` 的数据，但同时想保留一份副本。

### io.MultiReader：串联多个 Reader

```go
mr := io.MultiReader(
    strings.NewReader("A"),
    strings.NewReader("B"),
    strings.NewReader("C"),
)
```

`MultiReader` 按顺序把多个 `Reader` 拼接成一个。 HTTP 请求体由 header 和 body 拼接时，底层就可能用到类似的思路。

**实测输出：**

```text
io.Copy 复制了 28 字节: "复制这段文字到 buffer"
TeeReader 读取: "TeeReader 演示", 同时写入: "TeeReader 演示"
MultiReader(A,B,C) = "ABC"
```

## 17.5 io/fs 抽象与 os.DirFS

Go 1.16 引入的 `io/fs` 包定义了一套**文件系统抽象**：

```go
type FS interface {
    Open(name string) (File, error)
}
```

只要实现 `Open`，就可以被 `fs.ReadFile`、`fs.ReadDir`、`fs.WalkDir` 等工具函数操作。这意味着测试时可以用内存中的 `testing/fstest.MapFS` 替代真实磁盘，而业务代码完全无感知。

### os.DirFS：把目录变成 fs.FS

```go
fsys := os.DirFS(".")
data, err := fs.ReadFile(fsys, "go.mod")
entries, err := fs.ReadDir(fsys, ".")
```

`os.DirFS` 返回的 `fs.FS` 把所有路径都当作相对路径处理，**不支持绝对路径和以 `..` 开头的路径**。这是为了与 `embed.FS` 等行为保持一致。

**实测输出：**

```text
fs.ReadFile 读取 go.mod 前 40 字节: "module go-learn\n\ngo 1.26\n"...
当前目录包含 19 个条目:
  .claude (isDir=true)
  .git (isDir=true)
  ...
io/fs 抽象的好处：测试时可以用 os.DirFS 或 testing/fstest.MapFS 替换真实文件系统
```

## 17.6 filepath 跨平台路径处理

`path` 包操作**斜杠分隔**的路径（URL、zip 内部路径），而 `path/filepath` 操作**与操作系统一致**的本地文件路径。写 CLI 或处理文件系统时，几乎总是用 `filepath`。

### 核心函数

```go
p := filepath.Join("a", "b", "c.txt")   // 自动使用正确分隔符
base := filepath.Base(p)                 // "c.txt"
dir := filepath.Dir(p)                   // "a\b"
ext := filepath.Ext(p)                   // ".txt"
abs := filepath.IsAbs(p)                 // false
clean := filepath.Clean("a/../b/./c//d") // "b\c\d"
slash := filepath.ToSlash(p)             // "a/b/c.txt"
```

### 平台差异

- **Windows**：分隔符是 `\`，绝对路径需要盘符（如 `C:\foo`）
- **Unix**：分隔符是 `/`，以 `/` 开头即为绝对路径

因此 `filepath.IsAbs("/absolute/path")` 在 Windows 上返回 `false`，在 Linux/macOS 上返回 `true`。跨平台代码不应该硬编码 `/` 或 `\`，始终使用 `filepath.Join`。

**实测输出（Windows）：**

```text
filepath.Join: "a\\b\\c.txt"
路径: "home\\user\\doc.txt"
  Base="doc.txt", Dir="home\\user", Ext=".txt"
  IsAbs("home\\user\\doc.txt")=false, IsAbs("/absolute/path")=false
filepath.Clean("a/../b/./c//d") = "b\\c\\d"
filepath.ToSlash("a\\b\\c.txt") = "a/b/c.txt"
```

## 17.7 filepath.WalkDir 遍历目录

遍历目录树的标准做法是 `filepath.WalkDir`（Go 1.16+）：

```go
err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
    if err != nil {
        // 权限不足等问题：可以记录日志并继续遍历
        return nil
    }
    if d.IsDir() && path != "." {
        return fs.SkipDir // 不进入子目录
    }
    fmt.Println(path, d.Name())
    return nil
})
```

### WalkDir vs Walk

- `WalkDir` 传递的是 `fs.DirEntry`，避免了对每个文件调用 `os.Lstat`，性能更好
- `Walk` 传递的是 `os.FileInfo`，需要额外的系统调用
- 新项目应直接使用 `WalkDir`

回调函数签名的 `error` 参数很重要：如果某个子目录无权访问，`WalkDir` 不会直接失败退出，而是把错误传进回调，由你决定是 `return err`（终止遍历）还是 `return nil`（跳过并继续）。

**实测输出：**

```text
  . (size=4096)
  .gitignore (size=171)
  AGENTS.md (size=9406)
  CLAUDE.md (size=9406)
  README.md (size=6776)
共遍历到 9 个文件（当前目录，不递归）
```

## 17.8 临时文件、权限与原子写

### 临时文件与目录

```go
tmpDir, err := os.MkdirTemp("", "prefix_*")    // 创建临时目录
tmpFile, err := os.CreateTemp(tmpDir, "*.txt") // 在指定目录创建临时文件
```

第二个参数中的 `*` 会被替换为随机字符串，避免冲突。临时文件在程序退出前应主动清理（`defer os.RemoveAll(tmpDir)`）。

### 文件权限

创建文件时给出的权限位（如 `0o644`）会受 `umask` 影响：最终权限是 `指定权限 & ^umask`。在 Windows 上，权限模型的语义与 Unix 不同，`0o666` 是常见的可写文件默认值。

### 原子写文件

直接覆盖目标文件有一个风险：如果程序在写入过程中崩溃，目标文件可能处于**半写状态**（损坏）。原子写的做法是：

1. 把数据写到**同目录的临时文件**
2. 调用 `os.Rename` 把临时文件**重命名**为目标文件

```go
func atomicWriteFile(path string, data []byte) error {
    dir := filepath.Dir(path)
    tmp, err := os.CreateTemp(dir, ".atomic_*.tmp")
    if err != nil {
        return fmt.Errorf("create temp: %w", err)
    }
    tmpPath := tmp.Name()

    if _, err := tmp.Write(data); err != nil {
        tmp.Close()
        os.Remove(tmpPath)
        return fmt.Errorf("write temp: %w", err)
    }
    if err := tmp.Close(); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("close temp: %w", err)
    }

    if err := os.Rename(tmpPath, path); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("rename: %w", err)
    }
    return nil
}
```

**关键细节：**

- 临时文件必须和目标文件**在同一个文件系统**（同一个挂载点），否则 `Rename` 不是原子的
- **Windows 上必须先 `Close` 再 `Rename`**，因为打开的文件不能被重命名覆盖
- 每次出错都要删除临时文件，避免磁盘堆积

**实测输出：**

```text
临时目录: C:\Users\...\Temp\go17_atomic_...
临时文件: C:\Users\...\Temp\go17_atomic_...\tmp_....txt
原子写结果: "原子写入的内容\n"
文件权限: 666
```

## 17.9 //go:embed 嵌入静态资源

Go 1.16 支持在编译时把静态文件嵌入二进制，通过 `embed` 包实现：

```go
import "embed"

//go:embed embed_demo.txt
var content embed.FS
```

### 使用规则

- `//go:embed` 指令必须紧跟在**包级变量**声明之前，不能用于局部变量
- 路径是相对于当前 **Go 源文件** 的目录，不支持绝对路径和 `..`
- 支持通配符，如 `//go:embed *.txt`、`*/*.html`
- 变量类型可以是 `embed.FS`（文件系统）、`string` 或 `[]byte`

```go
//go:embed hello.txt
var hello string

//go:embed logo.png
var logo []byte
```

### 为什么用 embed.FS 而不是 string

`embed.FS` 实现了 `fs.FS` 接口，可以用 `fs.ReadFile`、`fs.ReadDir` 统一操作。如果嵌入的是多个文件或需要目录遍历，优先用 `embed.FS`；如果只有单个文本文件，用 `string` 更直接。

**实测输出：**

```text
嵌入文件内容:
Hello from go:embed!
这是第 17 章嵌入文件的演示内容。

//go:embed 规则：
  - 指令必须紧跟在包含它的变量声明之前
  - 只能用于包级变量，不能用于局部变量
  - 路径不支持绝对路径和 .. 上级目录
  - 支持通配符，如 //go:embed *.txt
```

## 3 个真实报错怎么读

### 1. open xxx: The system cannot find the file specified.

```text
fs.ReadFile 失败: open version.go: The system cannot find the file specified.
```

**原因：** 工作目录和源文件目录不一致。`go run ./cmd/go-learn` 时，工作目录是项目根目录，而 `version.go` 在 `internal/chapter/go17_files_io/` 下。  
**修复：** 使用相对工作目录的正确路径，或用 `os.DirFS("internal/chapter/go17_files_io")`。

### 2. bufio.Scanner: token too long

```go
scanner := bufio.NewScanner(file)
for scanner.Scan() { ... }
// scanner.Err() == bufio.ErrTooLong
```

**原因：** 单行超过 64KB。  
**修复：** 调用 `scanner.Buffer(make([]byte, 1024*1024), 1024*1024)` 扩容，或改用 `bufio.Reader.ReadLine`。

### 3. rename xxx yyy: The process cannot access the file because it is being used by another process.

**原因：** Windows 上尝试 `os.Rename` 覆盖一个**仍然打开**的文件。  
**修复：** 确保目标文件和所有临时文件都已 `Close()` 再执行 `Rename`。

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| `defer f.Close()` 没报错但文件内容丢失 | `Close` 把缓存刷盘时才暴露写入错误，defer 丢弃了返回值 | 写场景显式 `Close()` 并检查错误 |
| `Scanner` 读到一半报错 | 单行超过 64KB | 扩容 buffer 或改用 `bufio.Reader` |
| `filepath.Join` 在 Windows 产出 `\\` | 传入的参数本身以 `\` 结尾 | 用 `filepath.Clean` 规范化 |
| `os.Rename` 不是原子的 | 临时文件和目标文件不在同一文件系统 | 把临时文件创建到目标文件同目录 |
| Windows 上原子写失败 | 文件未关闭就 Rename | 先 `Close()` 再 `Rename` |
| `embed` 找不到文件 | `//go:embed` 路径是相对源文件目录，且不支持 `..` | 检查路径，确保文件在包目录内 |
| `WalkDir` 遇到权限错误就中断 | 回调里没有处理 `err` 参数 | 在回调里判断 `err != nil` 时返回 `nil` 继续遍历 |

## 练习与参考答案

### 第 1 题
用 `io.MultiWriter` 实现一个函数，把数据同时写入文件和 `bytes.Buffer`，并返回 buffer 的内容。

::: details 第 1 题
```go
func writeToBoth(path string, data []byte) (string, error) {
    f, err := os.Create(path)
    if err != nil {
        return "", err
    }
    defer f.Close()

    var buf bytes.Buffer
    mw := io.MultiWriter(f, &buf)
    if _, err := mw.Write(data); err != nil {
        return "", err
    }
    return buf.String(), f.Close()
}
```

**为什么这样写：** `MultiWriter` 让我们只写一次数据就同时落盘和留内存副本；`f.Close()` 显式调用并返回错误，不依赖 defer。
:::

### 第 2 题
读取一个文本文件，过滤掉空行和 `#` 开头的注释行，把剩余内容写入新文件。要求使用原子写，避免输出文件损坏。

::: details 第 2 题
```go
func filterFile(src, dst string) error {
    data, err := os.ReadFile(src)
    if err != nil {
        return err
    }

    var out bytes.Buffer
    scanner := bufio.NewScanner(bytes.NewReader(data))
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        out.WriteString(line)
        out.WriteByte('\n')
    }

    return atomicWriteFile(dst, out.Bytes()) // 复用本章的 atomicWriteFile
}
```

**为什么这样写：** 先用 `bytes.Buffer` 在内存里组装完整输出，再一次性原子写入，避免目标文件处于中间状态。
:::

### 第 3 题
用 `filepath.WalkDir` 统计某个目录下所有 `.go` 文件的总字节数，遇到子目录继续递归。

::: details 第 3 题
```go
func totalGoBytes(dir string) (int64, error) {
    var total int64
    err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return nil // 跳过无权限的文件，继续遍历
        }
        if d.IsDir() {
            return nil
        }
        if filepath.Ext(path) != ".go" {
            return nil
        }
        info, err := d.Info()
        if err != nil {
            return nil
        }
        total += info.Size()
        return nil
    })
    return total, err
}
```

**为什么这样写：** `WalkDir` 本身已经递归了，回调里只需判断是否为 `.go` 文件并累加大小；出错时返回 `nil` 保证遍历不中断。
:::

### 第 4 题
用 `//go:embed` 嵌入一个 JSON 配置文件，在 `init` 中解析并打印其中的一个字段。

::: details 第 4 题
假设包目录下有 `config.json`：

```json
{"name": "demo", "port": 8080}
```

代码：

```go
//go:embed config.json
var configFS embed.FS

var AppConfig struct {
    Name string `json:"name"`
    Port int    `json:"port"`
}

func init() {
    data, err := configFS.ReadFile("config.json")
    if err != nil {
        panic(err)
    }
    if err := json.Unmarshal(data, &AppConfig); err != nil {
        panic(err)
    }
}
```

**为什么这样写：** 配置随二进制一起发布，避免运行时找不到配置文件；`init` 保证程序启动前就完成解析。
:::

### 第 5 题
解释为什么 `filepath.Join("a", "../b")` 在 Windows 上输出 `a\..\b`，而 `filepath.Clean` 之后才是 `b`。

::: details 第 5 题
`filepath.Join` 的核心职责是「用正确分隔符拼接」，它**不会**解析 `..` 和 `.`。 `filepath.Clean` 的职责才是「规范化路径」：去掉冗余的 `.`、`..` 和多余的分隔符。这是两个函数职责分离的设计，需要组合使用：

```go
p := filepath.Clean(filepath.Join("a", "../b"))
```
:::

### 第 6 题
写一段代码，用 `io.TeeReader` 计算 MD5 哈希的同时把内容拷贝到文件。

::: details 第 6 题
```go
func copyWithHash(dstPath string, src io.Reader) (string, error) {
    f, err := os.Create(dstPath)
    if err != nil {
        return "", err
    }
    defer f.Close()

    h := md5.New()
    tr := io.TeeReader(src, h)
    if _, err := io.Copy(f, tr); err != nil {
        return "", err
    }
    return fmt.Sprintf("%x", h.Sum(nil)), f.Close()
}
```

**为什么这样写：** `TeeReader` 让 `io.Copy` 读到的每一份数据都同时流向 `md5.Writer`（它实现了 `io.Writer`），这样只读一次源数据就同时完成了拷贝和哈希计算。
:::

## 小结

- `io.Reader` 和 `io.Writer` 是 Go IO 的基石，通过组合（`MultiWriter`、`TeeReader`、`MultiReader`）可以搭出复杂的数据流而无需中间缓存。
- `os.ReadFile` / `WriteFile` 适合一次性读写；`os.OpenFile` 提供追加、创建、截断等精细控制。
- 写文件时 `defer f.Close()` 会静默丢弃 `Close` 的错误，重要数据应显式关闭并检查返回值。
- `bufio.Scanner` 默认单行限制 64KB，超长行需要扩容 buffer 或改用 `bufio.Reader`。
- `io/fs` 抽象让业务代码与真实文件系统解耦，测试时用 `MapFS` 替换 `os.DirFS` 非常方便。
- `path/filepath` 处理本地路径，始终用它替代手写的 `/` 或 `\`，保证跨平台。
- `filepath.WalkDir` 比旧版 `Walk` 更高效，回调里的 `err` 参数让你决定是跳过还是终止。
- 原子写的标准模式是「同目录临时文件 + Rename」，Windows 上尤其要注意先 Close 再 Rename。
- `//go:embed` 把静态资源编译进二进制，路径相对于源文件目录，不支持 `..`。

下一章我们将进入「序列化与配置」，看看 `encoding/json`、`xml`、`csv` 以及如何处理配置分层与优先级。
