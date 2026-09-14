# 第 11 章 · 错误处理

前面几章里，`err != nil` 已经出现过很多次：`strconv.Atoi`、`os.Open`、`json.Unmarshal` 都用「第二个返回值」报告失败。Go 没有异常机制，错误就是普通的**值**——这既是 Go 最容易被吐槽的地方，也是它最实用的设计之一：错误可以被返回、被包装、被比较、被结构化地检查，调用方一眼就能看出哪一行可能失败。

但「返回值」这个设计只有配上明确的规则才成立：什么情况下该返回错误，什么时候该新建哨兵错误，`%v` 和 `%w` 到底差在哪，`errors.Is` 和 `errors.As` 分别解决什么问题，多个校验错误怎么合并，日志该在哪一层打，以及 `panic` / `recover` 到底该不该用。这些规则不统一，就会出现「同一个错误在每层都打印一遍」「用字符串匹配判断错误类型」「panic 穿越到调用方」这些典型的工程问题。

本章按「从错误值到错误链」的顺序展开：先看清 `error` 只是一个接口，再学会创建、包装、判断和合并错误，最后讨论自定义错误类型、日志分工和 panic 的边界。读完你应该能回答：为什么 `err == ErrNotFound` 会失效，为什么 `defer f.Close()` 会吞掉错误，以及什么样的场景才值得 panic。

本章配套代码在 `internal/chapter/go11_errors/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

---

## 11.1 error 是一个接口

结论：`error` 只是一个只有一个方法的接口——`Error() string`。任何实现了它的类型都能当错误用，所以「错误」既可以是一个简单的字符串，也可以是一个带字段的结构体。

```go
type error interface {
    Error() string
}
```

### 实测输出

```text
未赋值的 error = <nil>，err == nil → true
errors.New 返回的底层类型是 *errors.errorString
打印时自动调用 Error()：%v = something failed，%s = something failed
自定义类型也能当 error 用：invalid age: out of range（底层类型 *go11_errors.ValidationError）
```

几个由此推出的结论：

- `error` 的零值是 `nil`，判断「有没有出错」就是 `err != nil`。
- `fmt` 打印错误时自动调用 `Error()`，所以 `%v`、`%s` 输出的是错误信息。
- `errors.New` 返回的是一个**未导出的具体类型**（`*errors.errorString`），这也解释了为什么调用方不能对它做类型断言，只能用 `errors.Is` 比较。
- 自定义错误类型同样可以实现 `error`，并携带结构化字段（11.8 展开）。

### 边界与坑

- 接口值比较是「类型 + 值」的比较：只要具体类型不同，`err1 == err2` 就是 `false`，即使 `Error()` 返回的文本一样。
- 接口里的 `nil` 和接口本身的 `nil` 不是一回事：把一个 `nil` 的具体指针赋给 `error`，接口就不等于 `nil` 了，这是最容易踩的坑之一（见报错 4）。
- 只有需要判断「是不是某一类错误」时才需要类型信息，其余场景请一律用 `errors.Is` / `errors.As`。

---

## 11.2 创建错误的三种方式

结论：需要调用方识别的原因用**包级哨兵错误**；需要补上下文的地方用 `fmt.Errorf` + `%w`；两者都不需要时才直接新建一个错误。

```go
var ErrNotFound = errors.New("resource not found")   // 哨兵：包级变量，调用方可以用 errors.Is 认出来

fmt.Errorf("load user 42: %w", ErrNotFound)          // 包装：补上下文，保留链
errors.New("cache miss")                             // 就地新建：调用方无法识别
```

### 实测输出

```text
包级哨兵 ErrNotFound        → resource not found
fmt.Errorf + %w 包装        → load user 42: resource not found
就地 errors.New             → cache miss
errors.Is(wrapped, ErrNotFound)   → true（包装不丢身份）
errors.Is(onTheFly, ErrNotFound)  → false（就地新建的错误没有身份）
```

三种写法的选择标准很直接：

| 写法 | 什么时候用 | 调用方怎么判断 |
| --- | --- | --- |
| 包级哨兵 `ErrXxx` | 是「可预期的业务结果」，调用方需要分支 | `errors.Is(err, ErrXxx)` |
| `fmt.Errorf` + `%w` | 需要给下层错误补上「哪一步失败了」 | 继续用 `errors.Is` / `errors.As` |
| 就地 `errors.New` | 不需要调用方识别，只要一个说明 | 只能看文本 |

哨兵错误的命名约定是 `ErrXxx`（导出）或 `errXxx`（包内），定义成包级变量，**不要在函数里反复新建**——新建出来的错误没有身份，`errors.Is` 永远匹配不上。

### 边界与坑

- 哨兵错误的信息里不要再重复包名和函数名，`resource not found` 比 `user.ErrNotFound: resource not found` 更清楚。
- 导出的哨兵错误是 API 的一部分，改动文案可能影响下游用户，要当作兼容性来处理。
- `fmt.Errorf` 里用 `%v` 而不是 `%w` 也能输出一样的信息，但会**断链**，这是 11.4 的主题。

---

## 11.3 错误是返回值，必须立即检查

结论：每个 `err` 要么在这一层处理掉，要么包装后上抛；不能既不处理也不上抛。Go 编译器只强制「变量被使用」，不强制「错误被处理」，所以这条规则要靠人守。

```go
n, err := strconv.Atoi("abc") // 错误：忽略了 err
fmt.Println(n * 2)            // n 是 0，结果毫无意义但程序不会报错
```

### 实测输出

```text
Atoi("42") = 42，err = <nil>
忽略 err：Atoi("abc") 返回 n = 0，继续算 n*2 = 0（结果没有意义）
上抛时补上下文：find user 0: resource not found
```

### 几个常见形态

```go
// 1. 立即处理：能自己修复或转换的就地处理
if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
    return createDefault(path)
}

// 2. 立即上抛并补上下文：自己处理不了的交给上层
data, err := os.ReadFile(path)
if err != nil {
    return fmt.Errorf("read %s: %w", path, err)
}

// 3. 用 if err := f(); err != nil 缩短作用域
if err := validate(input); err != nil {
    return err
}
```

### 边界与坑

- `_ = err` 只应该在「确实无所谓」的地方出现，比如 `defer f.Close()` 里关闭只读文件失败；即便这样也建议写明理由。
- 忽略返回值的错误比忽略错误本身更危险：`n, _ := strconv.Atoi(s)` 之后拿 `n` 参与计算，出错时用的是 0。
- 循环里只 `log.Print(err)` 不返回，会让上层以为调用成功——日志不能代替错误传递。

---

## 11.4 包装错误：%w 与错误链

结论：`fmt.Errorf` 里的 `%w` 会在生成新错误的同时**保存**被包装的错误，形成一条错误链；`%v` 只把文本拼进去，链就断了。

```go
textOnly := fmt.Errorf("divide 10 by 0: %v", ErrDivideByZero) // 断链
linked := fmt.Errorf("divide 10 by 0: %w", ErrDivideByZero)   // 保留链
```

### 实测输出

```text
%v 包装：divide 10 by 0: divide by zero，Unwrap = <nil>
%w 包装：divide 10 by 0: divide by zero，Unwrap = divide by zero
两者文本完全相同：true
errors.Is(textOnly, ErrDivideByZero) = false，errors.Is(linked, ErrDivideByZero) = true
两层包装后的完整信息：start server: read config: divide 10 by 0: divide by zero
errors.Is 会自己沿链往下找：true
```

注意第三行：两种写法的**输出文本完全一样**，区别只体现在后续能否被识别。这正是「用日志看不出问题、但代码判断会失效」的典型场景。

### 错误链长什么样

```text
start server: read config: divide 10 by 0: divide by zero
└─ start server（最外层的 %w 包装，补充「在做什么」）
   └─ read config（第二层 %w 包装，继续补上下文）
      └─ divide 10 by 0（最内层 %w 包装）
         └─ divide by zero（哨兵错误，errors.Is 最终命中的就是它）
```

包装时每一层只补一段「我在做什么」的上下文，由内向外读就是一次完整的调用过程。

### 边界与坑

- 一个 `fmt.Errorf` 里可以写多个 `%w`（Go 1.20 起），会得到一个多错误链；但可读性会下降，多个错误建议用 `errors.Join`。
- `errors.Unwrap` 只能往下剥一层，手写多层遍历容易出错，判断类型请直接用 `errors.Is` / `errors.As`。
- 不要包装成一个新哨兵：`fmt.Errorf("not found: %w", ErrInternal)` 会把「真正的原因」藏起来，`errors.Is` 也会答错。

---

## 11.5 errors.Is：沿错误链判断「是不是它」

结论：判断「错误是不是某一类」永远用 `errors.Is`，不要用 `==`，也不要匹配 `Error()` 的文本。

```go
_, err := FindUser(0)
fmt.Println(err == ErrNotFound)          // false：== 只看最外层
fmt.Println(errors.Is(err, ErrNotFound)) // true：Is 会沿链查找
```

### 实测输出

```text
FindUser(0) 的错误：find user 0: resource not found
err == ErrNotFound             → false（== 只看最外层）
errors.Is(err, ErrNotFound)    → true（沿错误链匹配）
errors.Is(err, ErrDivideByZero) → false（无关的哨兵不匹配）
判断错误永远不要匹配文案：strings.Contains(err.Error(), "not found") 会随措辞变化而失效。
```

`errors.Is(err, target)` 的规则是：

1. 如果 `err` 和目标**相等**（`==`），返回 `true`；
2. 否则如果 `err` 实现了 `Is(error) bool`，用它的判断结果；
3. 否则沿 `Unwrap()` 往下走，重复以上过程；
4. 走到链尾仍然不匹配，返回 `false`。

所以 `errors.Is` 既能匹配哨兵错误，也能匹配实现了 `Is` 的自定义错误——这也是标准库 `fs.ErrNotExist`、`os.ErrPermission` 的用法：

```go
if _, err := os.Open(path); errors.Is(err, fs.ErrNotExist) {
    return newDefaultConfig()
}
```

### 边界与坑

- `errors.Is` 的目标必须是**同一个错误值**（哨兵）或实现了 `Is` 的类型；临时 `fmt.Errorf` 出来的错误永远匹配不上。
- `errors.Is(nil, x)` 永远是 `false`，`errors.Is(err, nil)` 等价于 `err == nil`，不要用它做「有没有错」的判断。
- 文案匹配（`strings.Contains(err.Error(), "not found")`）在包装层数、语言、措辞变化后都会失效，只是把 bug 推迟到线上。

---

## 11.6 errors.As：从错误链里取回具体类型

结论：需要错误里的**结构化字段**时用 `errors.As`，它会把链上第一个匹配类型的错误取回来；直接类型断言在包装之后就失效了。

```go
err := fmt.Errorf("validate user: %w", &ValidationError{Field: "age", Msg: "out of range"})

_, ok := err.(*ValidationError) // false：最外层是 *fmt.wrapError

var ve *ValidationError
if errors.As(err, &ve) {        // true：As 会沿链找到它
    fmt.Println(ve.Field, ve.Msg)
}
```

### 实测输出

```text
直接类型断言 err.(*ValidationError) 失败：包装后的类型是 *fmt.wrapError
errors.As 取回字段：Field="age"，Msg="out of range"（完整信息：invalid age: out of range）
errors.As 需要传「指向目标类型的指针」，它会自己沿链查找并赋值。
```

### Is 和 As 怎么选

| 场景 | 用哪个 | 例子 |
| --- | --- | --- |
| 判断「是不是这类失败」 | `errors.Is` | `errors.Is(err, ErrNotFound)` |
| 取出错误里的字段 | `errors.As` | `errors.As(err, &validationErr)` |
| 判断「实现了某个接口」 | `errors.As` | `errors.As(err, &temporaryErr)` |

### 边界与坑

- 第二个参数必须是**指向目标类型的指针**（`&ve`，`ve` 的类型是 `*ValidationError`），少写 `&` 会直接 panic（见报错 3）。
- `errors.As` 只取链上第一个匹配项；`errors.Join` 出来的多错误需要遍历 `Unwrap() []error` 才能看全（11.7）。
- 目标类型用指针接收者实现 `Error()` 时，`As` 的目标也要是**指向该类型的指针**，这两层指针很容易写错。

---

## 11.7 errors.Join：把多个错误合成一个

结论：一次操作有多个独立失败原因时（表单校验、批量任务），用 `errors.Join` 合并，它保留全部子错误，并且实现 `Unwrap() []error`。

```go
err := errors.Join(
    &ValidationError{Field: "name", Msg: "must not be empty"},
    &ValidationError{Field: "age", Msg: "out of range"},
)
```

### 实测输出

```text
ValidateUser("", 200) 的错误（多行就是 Join 的默认格式）：
invalid name: must not be empty
invalid age: out of range
  子错误 1：invalid name: must not be empty
  子错误 2：invalid age: out of range
errors.As 拿到的是第一个 *ValidationError：Field="name"
errors.Join() = <nil>，== nil → true
errors.Join(nil, nil) == nil → true（nil 会被忽略）
```

### 三个使用要点

1. **输出是多行文本**：`Join` 用换行符拼接子错误，写日志时要注意多行日志的排版；返回给前端前通常要转成结构化列表。
2. **`Is` / `As` 会依次检查所有子错误**：所以 `errors.Is(joined, ErrNotFound)` 只要任一子错误匹配就为真。
3. **`Join()` 没有参数时返回 `nil`**：可以放心地把收集到的错误切片直接丢进去，不用先判断长度。

```go
func ValidateUser(name string, age int) error {
    var errs []error
    if name == "" {
        errs = append(errs, &ValidationError{Field: "name", Msg: "must not be empty"})
    }
    if err := ValidateAge(age); err != nil {
        errs = append(errs, err)
    }
    return errors.Join(errs...) // 全部通过时 errs 为空，返回 nil
}
```

### 边界与坑

- 只有需要「一次报出全部问题」时才值得合并；单点失败继续用「包装上抛」的链式结构更清晰。
- `Join` 出的错误没有自己的类型，`errors.As(err, &multi)` 时目标要写成 `interface{ Unwrap() []error }`。
- 想给每个子错误补上下文，要在 `Join` 之前逐个 `fmt.Errorf(..., %w)` 包装好。

---

## 11.8 自定义错误类型的设计

结论：当错误需要携带**结构化信息**（字段名、重试建议、HTTP 状态码等），就用结构体实现 `error`；同时实现 `Unwrap` 把链接起来。

```go
// QueryError 给底层错误补上「哪次查询失败」，并保留错误链。
type QueryError struct {
    Query string
    Err   error
}

func (e *QueryError) Error() string { return fmt.Sprintf("query %q failed: %v", e.Query, e.Err) }
func (e *QueryError) Unwrap() error { return e.Err }
```

### 实测输出

```text
queryUser(0) = query "select id=0 from users" failed: resource not found
底层类型：*go11_errors.QueryError
实现了 Unwrap，链能继续往下走：errors.Is(err, ErrNotFound) = true
errors.As 取回：Query="select id=0 from users"，底层 Err=resource not found
```

### 三个组成部分各管什么

| 成员 | 职责 | 注意 |
| --- | --- | --- |
| `Error() string` | 描述「发生了什么」，给人看 | 只拼自己这一段信息，不要重复底层错误的文案 |
| 结构体字段 | 给程序看的结构化信息 | 命名要能自解释，如 `Field`、`Retryable` |
| `Unwrap() error` | 保留「属于哪一类错误」 | 有包装就必须实现，否则 `errors.Is` / `As` 会断在这里 |

### 值接收者还是指针接收者

```go
func (e ValidationError) Error() string  { ... } // 值接收者：error 里存值
func (e *QueryError) Error() string      { ... } // 指针接收者：error 里存指针
```

两种都能用，但要**一致**：`Error()` 用指针接收者时，只有 `*QueryError` 实现了 `error`，`QueryError{}` 这个值不能当错误用；`errors.As` 的目标也要相应地写成 `var qe *QueryError`。

### 边界与坑

- `Error()` 里不要做有副作用或可能 panic 的事情（发请求、改状态、解引用可能为 nil 的字段）。
- 让错误类型实现 `Is` 或 `As` 可以自定义匹配规则，但除非确有需要，不要覆盖默认行为。
- 字段多了，`Error()` 的输出仍应保持一行、可直接放进日志。

---

## 11.9 错误信息与日志的分工

结论：**错误信息负责说清「哪一步失败、为什么」，日志负责在最外层记录一次**；两者都写、层层都写，就会得到一堆重复日志。

### 书写规范对照

```text
不推荐的写法："Error: Failed to open config file."
推荐的写法：  "open config file: permission denied"
```

不推荐的那句有三个问题：`Error:` 前缀是多余信息（日志系统已经有级别）、首字母大写和句尾句号不符合 Go 惯例（错误常被拼接进更大的句子）、`Failed to` 没有说出真正的原因。推荐的写法由「做了什么」加「为什么失败」组成，由内向外读就是调用链。

### 每层只补一段上下文

```go
inner := errors.New("permission denied")
layer1 := fmt.Errorf("open /etc/app.conf: %w", inner) // 哪一步
layer2 := fmt.Errorf("load config: %w", layer1)       // 哪一步
layer3 := fmt.Errorf("start server: %w", layer2)      // 哪一步
```

### 实测输出

```text
最外层看到的完整信息：start server: load config: open /etc/app.conf: permission denied
底层哨兵仍然可辨认：errors.Is(layer3, inner) = true
```

也就是说，**中间层只包装、不打印**，把日志留给最外层（`main`、HTTP 中间件、任务调度器）记录一次即可。

### 判断错误用 Is/As，不用字符串

```go
// 反模式：文案一改就失效，包装层数一变也可能失效
if strings.Contains(err.Error(), "not found") { ... }

// 正确：判断「身份」
if errors.Is(err, ErrNotFound) { ... }
```

### 边界与坑

- 中间层如果既 `return err` 又 `log.Println(err)`，同一次故障会在日志里出现 N 次，排查时反而更乱。
- 需要记录上下文（请求 ID、用户 ID）时用结构化日志的字段，不要拼进错误信息里。
- 返回给外部的错误信息要**脱敏**：数据库连接串、文件绝对路径、内部主机名不要原样透出，在边界层转换成对外文案。

---

## 11.10 defer 与 Close 的错误

结论：`Close` / `Flush` 也会失败，用**命名返回值 + defer** 才能把它们的错误并进最终返回值；否则错误会被静默丢掉。

```go
func Save(payload string, failFlush bool) (err error) {
    defer func() {
        if flushErr := flush(); flushErr != nil && err == nil {
            err = fmt.Errorf("save %q: %w", payload, flushErr)
        }
    }()
    ...
}
```

### 实测输出

```text
Save("data", 刷新成功) = <nil>
Save("", 刷新成功)     = empty payload
Save("data", 刷新失败) = save "data": flush failed
errors.Is(Save("data", 刷新失败), ErrFlushFailed) = true
```

### 三个关键点

1. **必须用命名返回值**：只有命名返回值才能被 `defer` 里的函数修改；`func Save(...) error` 这种写法里，`defer` 改的是局部副本。
2. **`err == nil` 时才覆盖**：已经有真正的失败原因时，关闭错误只是次生问题，不应该掩盖它。
3. **用 `%w` 保留链**：这样上层仍然能用 `errors.Is` 判断是「业务失败」还是「关闭失败」。

### 读文件时的等价写法

```go
f, err := os.Open(path)
if err != nil {
    return fmt.Errorf("open %s: %w", path, err)
}
defer func() {
    if cerr := f.Close(); cerr != nil && err == nil {
        err = fmt.Errorf("close %s: %w", path, cerr)
    }
}()
```

### 边界与坑

- 只读场景里 `defer f.Close()` 忽略错误通常可以接受，但要在注释里写明理由。
- `defer` 里调用 `recover()` 是唯一能拦住 panic 的方式；`defer` 里 panic 则会替换掉原来的 panic。
- 多个资源要关闭时，按打开顺序**倒序**关闭（`defer` 天然就是后进先出）。

---

## 11.11 panic 与 recover 的边界

结论：**可预期的失败返回 error，不可预期的程序 bug 才 panic**；`recover` 只应该在「必须让程序活下来」的边界上使用。

### 该返回 error 的情形

```go
func Divide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, fmt.Errorf("divide %g by %g: %w", a, b, ErrDivideByZero)
    }
    return a / b, nil
}
```

### 实测输出

```text
Divide(10, 2) → 5，err = <nil>
Divide(10, 0) → err = divide 10 by 0: divide by zero
在非 defer 语句里调用 recover() = <nil>（拦不到任何 panic）
SafeDivide(10, 0) → result=0，err=safe divide 10/0: runtime error: integer divide by zero
SafeDivide(10, 2) → result=5，err=<nil>
```

注意第三行：`recover()` 只有在 `defer` 调用的函数里执行才有效，直接写在普通语句里永远是 `nil`——这是 `recover` 最常见的失效原因。

### 把 panic 转成错误

```go
func SafeDivide(a, b int) (result int, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("safe divide %d/%d: %v", a, b, r)
        }
    }()
    return a / b, nil // b == 0 时触发 runtime panic，被上面的 recover 接住
}
```

`recover` 适合放在这类边界上：

- HTTP 中间件、RPC 服务的入口：一个请求 panic 不应该让整个进程退出；
- 插件 / 脚本 / 表达式求值：执行的是外部传入的代码；
- 第三方库已知会 panic 的调用点（能用 error 版本就优先用 error 版本）。

### 边界与坑

- `recover` 只能拦住**同一个 goroutine** 里的 panic；子 goroutine 里 panic 仍然会让整个进程崩溃，要单独在自己的 goroutine 里 defer。
- `recover` 如果用来掩盖自己代码的 bug（空指针、越界），会把「配置错误」变成「静默返回 0」，比崩溃更难排查。
- 库代码不要 panic 给调用方：把「调用方用错 API」以外的场景都改成返回 error。
- `panic` 传值可以是任意类型，`recover` 拿到的是 `any`，恢复后要自己判断它是不是 error。

---

## 4 个真实报错怎么读

### 报错 1：错误变量没被使用

```go
value, err := doWork()
fmt.Println("value =", value) // 拿到了 err 却没用
```

编译输出：

```text
main.go:9:9: declared and not used: err
```

**怎么读**：Go 要求局部变量「声明了就必须被读取」，所以「把错误接进变量却从不使用」会被直接拦下。但它只检查「变量有没有被读」，不检查「错误有没有被处理」：

```go
doWork()                      // 两个返回值直接丢弃，编译通过（实测 exit=0）
n, err := strconv.Atoi(input) // err 被读过，编译也通过
if err != nil {
    log.Println(err)          // 只记日志，然后继续用 n 这个脏数据
}
fmt.Println(n)
```

所以 `go vet`、`errcheck`、`staticcheck` 这类工具仍然有必要：编译器只能保证你「看见」了错误，不能保证你「处理」了错误。

**怎么改**：立即判断并上抛，或者显式写 `_ = err` 并在注释里说明为什么可以忽略。

### 报错 2：%w 的参数不是 error

```go
err := fmt.Errorf("load user 42: %w", "not found") // 第二个参数是 string
fmt.Println(err)
```

`go build` 能通过，`go vet` 输出：

```text
main.go:8:35: fmt.Errorf format %w has arg "not found" of wrong type string
```

**怎么读**：`%w` 的语义是「把这个**错误**记进错误链」，所以参数必须是 `error`；给字符串只能用 `%s` / `%v`（但那样就不成链了）。这类格式化动词错误编译期检查不到，是 `go vet` 的 `printf` 检查在兜底——这也是为什么 `go test` 默认会跑一部分 vet 检查。

**怎么改**：要么把字符串换成错误值，要么把动词换成 `%s` 并接受「不再成链」。

```go
err := fmt.Errorf("load user 42: %w", ErrNotFound) // 带上哨兵，保留链
```

### 报错 3：errors.As 的目标不是指针

```go
var target *ValidationError
matched := errors.As(err, target) // 少写了 &target
```

运行输出：

```text
panic: errors: target must be a non-nil pointer

goroutine 1 [running]:
errors.As(...)
        .../src/errors/wrap.go:112 +0x1ff
main.main()
        .../main.go:19 +0x53
exit status 2
```

**怎么读**：`errors.As` 需要把匹配到的错误**写回**你传进去的变量，所以第二个参数必须是「指向目标类型的指针」。这里传进去的是一个 nil 的 `*ValidationError`，`errors.As` 无法往它里面写值，于是直接 panic。

**怎么改**：传 `&target`（`target` 的类型就是 `*ValidationError`，所以取地址正好得到 `**ValidationError`）：

```go
var target *ValidationError
if errors.As(err, &target) {
    fmt.Println(target.Field)
}
```

### 报错 4：接口里的 typed nil

```go
type MyError struct{ Code int }

func (e *MyError) Error() string { return fmt.Sprintf("code %d", e.Code) }

func main() {
    var e *MyError
    var err error = e // 接口非空，但里面装的是一个 nil 指针
    fmt.Println("err == nil →", err == nil)
    fmt.Println("err.Error() →", err.Error())
}
```

运行输出：

```text
err == nil → false
panic: runtime error: invalid memory address or nil pointer dereference
[signal 0xc0000005 code=0x0 addr=0x0 pc=0x...]
goroutine 1 [running]:
main.(*MyError).Error(...)
        .../main.go:12
main.main()
        .../main.go:19 +0x6d
exit status 2
```

**怎么读**：`err == nil` 是 `false`，说明上层以为「有错误」，于是去调用 `err.Error()`；但接口里装的具体值是 nil 指针，方法体里访问 `e.Code` 就炸了。这不是「错误值忘了初始化」，而是**接口的 nil 判定规则**：接口只有在「类型和值都是 nil」时才等于 `nil`。

**怎么改**：函数返回错误时不要返回「具体类型为 nil 的指针」，直接返回字面量 `nil`；或者让方法对 nil 接收者做防御：

```go
func (e *MyError) Error() string {
    if e == nil {
        return "unknown error"
    }
    return fmt.Sprintf("code %d", e.Code)
}

func find() error {
    var e *MyError // 别这样返回
    if e == nil {
        return nil // 直接返回无类型的 nil，接口才是 nil
    }
    return e
}
```

### 还有一个不报错的坑

```go
if err == ErrNotFound { // 包装之后的错误永远匹配不上
    return createDefault()
}
```

这段代码既不会编译失败，也不会 panic，只是**分支永远不成立**：`FindUser` 返回的是 `fmt.Errorf("find user 0: %w", ErrNotFound)`，最外层的具体类型是 `*fmt.wrapError`，与哨兵不是同一个值。改成 `errors.Is(err, ErrNotFound)` 即可（实测见 11.5）。

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| `err == ErrNotFound` 匹配不上 | 包装后最外层是另一个值 | 用 `errors.Is(err, ErrNotFound)` |
| 类型断言 `err.(*MyError)` 失败 | 错误被 `%w` 包装过 | 用 `errors.As(err, &target)` |
| `errors.As` panic | 第二个参数不是非 nil 指针 | 传 `&target` |
| `err != nil` 但输出/调用时 panic | 接口里装的是一个 nil 指针（typed nil） | 直接 `return nil`，或让方法防御 nil 接收者 |
| `errors.Is` 对 `fmt.Errorf("...: %v", err)` 失效 | `%v` 断链 | 需要成链的地方一律用 `%w` |
| 中间层日志刷屏 | 每层都 `log` 又都 `return` | 中间层只包装，最外层记录一次 |
| `defer f.Close()` 丢了错误 | 返回值未被 defer 修改 | 命名返回值 + defer 里判断 `err == nil` |
| `recover()` 拿不到 panic | 不在 `defer` 调用的函数里 | 只在 `defer func(){ recover() }()` 中调用 |
| 子 goroutine panic 导致进程退出 | `recover` 不跨 goroutine | 每个 goroutine 自己 defer + recover |
| 判断错误时匹配了文案 | `strings.Contains(err.Error(), ...)` | 用 `errors.Is` / `errors.As` 判断身份 |
| 校验只报第一个错误 | 提前 `return` | 收集后用 `errors.Join` 一次返回 |
| 错误信息重复且冗长 | 每层都重复底层文案 | 每层只补「我在做什么」，用 `%w` 拼链 |

---

## 练习

### 第 1 题

下面的函数想「找不到用户就创建默认用户」，但分支永远进不去。指出原因，并给出修改后的完整函数。

```go
func LoadUser(id int) (string, error) {
    name, err := FindUser(id)
    if err == ErrNotFound {
        return "default", nil
    }
    return name, err
}
```

::: details 第 1 题参考答案

**原因**：`FindUser` 返回的是 `fmt.Errorf("find user %d: %w", id, ErrNotFound)`，最外层的具体类型是 `*fmt.wrapError`，而 `ErrNotFound` 是 `*errors.errorString`，两者不是同一个值，`==` 恒为 `false`。程序不报错，只是行为错了。

```go
func LoadUser(id int) (string, error) {
    name, err := FindUser(id)
    if errors.Is(err, ErrNotFound) {
        return "default", nil
    }
    if err != nil {
        return "", fmt.Errorf("load user %d: %w", id, err)
    }
    return name, nil
}
```

**为什么这样写更好**：`errors.Is` 会沿 `Unwrap` 链查找，无论包装多少层都能命中哨兵；顺带补上了「其它错误要包装上抛」这条分支，调用方拿到的错误里既有上下文又能用 `errors.Is` 继续判断。

:::

### 第 2 题

实现 `Retryable` 错误类型：它能包住任意底层错误，并让 `errors.Is(err, &RetryableError{})` 判断为真（表示「这个错误值得重试」）。写出类型定义和一段验证代码。

::: details 第 2 题参考答案

```go
// RetryableError 表示调用方可以重试的失败。
type RetryableError struct {
    Err error
}

func (e *RetryableError) Error() string { return "retryable: " + e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

// Is 让 errors.Is(err, &RetryableError{}) 能识别出「这一类」错误。
func (e *RetryableError) Is(target error) bool {
    _, ok := target.(*RetryableError)
    return ok
}
```

验证：

```go
err := fmt.Errorf("call api: %w", &RetryableError{Err: io.ErrUnexpectedEOF})
fmt.Println(errors.Is(err, &RetryableError{})) // true
fmt.Println(errors.Is(err, io.ErrUnexpectedEOF)) // true，底层错误也没丢
```

**为什么这样写更好**：`Is` 方法把「哪些错误算这一类」的判断封装在类型自己身上，调用方只写 `errors.Is`，不需要了解内部结构；同时 `Unwrap` 让底层原因依然可查，两个判断互不干扰。

**注意**：`Is` 的目标是一个**空实例**（`&RetryableError{}`），所以 `Is` 里不要读字段。

:::

### 第 3 题

下面这段代码在每一层都打印日志，请改写成「中间层只包装、最外层记录一次」，并说明改动后日志和错误信息分别变成什么样。

```go
func loadConfig(path string) error {
    data, err := readFile(path)
    if err != nil {
        log.Printf("loadConfig 失败: %v", err)
        return err
    }
    _ = data
    return nil
}
```

::: details 第 3 题参考答案

```go
func loadConfig(path string) error {
    data, err := readFile(path)
    if err != nil {
        return fmt.Errorf("load config %s: %w", path, err)
    }
    _ = data
    return nil
}

func main() {
    if err := loadConfig("app.conf"); err != nil {
        log.Printf("start server: %v", err) // 只在这里记录一次
        os.Exit(1)
    }
}
```

**改动后的效果**：

- 日志只有一条，例如 `start server: load config app.conf: permission denied`，从外到内就是完整调用路径；
- 错误信息由每层各补一段，不再重复 (`loadConfig 失败` 这种「只有函数名没有原因」的描述被删掉了)；
- 上层仍然可以用 `errors.Is(err, fs.ErrPermission)` 判断具体原因。

**为什么这样写更好**：日志的价值在于「一次故障一条可检索的记录」，重复打印只会让排查时更难定位；错误链已经携带了上下文，日志只需要在最外层把它输出一次。

:::

### 第 4 题

写一个 `ValidateAll`，一次收集姓名、年龄、邮箱三个字段的校验错误并返回；要求没有错误时返回 `nil`，有多个错误时调用方能区分出具体字段。给出实现、测试和一次实测输出。

::: details 第 4 题参考答案

```go
func ValidateAll(name string, age int, email string) error {
    var errs []error
    if name == "" {
        errs = append(errs, &ValidationError{Field: "name", Msg: "must not be empty"})
    }
    if err := ValidateAge(age); err != nil {
        errs = append(errs, err)
    }
    if !strings.Contains(email, "@") {
        errs = append(errs, &ValidationError{Field: "email", Msg: "must contain @"})
    }
    return errors.Join(errs...)
}
```

测试要点：

```go
// 1. 全部合法时返回 nil
if err := ValidateAll("Alice", 30, "a@b.com"); err != nil {
    t.Fatalf("期望 nil，得到 %v", err)
}

// 2. 三个字段都错时能取回三个子错误
err := ValidateAll("", 200, "nope")
var multi interface{ Unwrap() []error }
if !errors.As(err, &multi) || len(multi.Unwrap()) != 3 {
    t.Fatalf("期望 3 个子错误，得到 %v", err)
}

// 3. 顺序与添加顺序一致，字段可单独取出
var ve *ValidationError
if !errors.As(err, &ve) || ve.Field != "name" {
    t.Fatalf("errors.As 应该拿到第一个错误，得到 %v", ve)
}
```

`ValidateAll("", 200, "nope")` 的输出是三行：

```text
invalid name: must not be empty
invalid age: out of range
invalid email: must contain @
```

**为什么这样写更好**：一次性返回全部问题，用户不必「改一个报一个」；`errors.Join` 保留了每个子错误的身份，调用方既能把它们转成结构化列表返回给前端，也能用 `errors.As` 单独取出某个字段。

:::

### 第 5 题

给 `os.ReadFile` 写一个包装函数，要求在文件不存在时返回一个「可以创建默认值」的错误：调用方能同时知道「是文件不存在」和「是哪个文件」。写出实现和判断代码。

::: details 第 5 题参考答案

```go
// ErrConfigMissing 表示配置文件缺失，调用方可以据此创建默认配置。
var ErrConfigMissing = errors.New("config missing")

func LoadConfig(path string) ([]byte, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        if errors.Is(err, fs.ErrNotExist) {
            // 换成业务哨兵，同时保留原始错误
            return nil, fmt.Errorf("%w: %s", ErrConfigMissing, path)
        }
        return nil, fmt.Errorf("read config %s: %w", path, err)
    }
    return data, nil
}
```

调用方：

```go
data, err := LoadConfig("app.conf")
switch {
case errors.Is(err, ErrConfigMissing):
    data = defaultConfig() // 文件不存在：用默认值
case err != nil:
    return fmt.Errorf("load config: %w", err)
}
```

**为什么这样写更好**：`fs.ErrNotExist` 是标准库的底层原因，业务层更关心「配置缺失」，所以在这一层换成业务哨兵；`fmt.Errorf("%w: %s", ...)` 只保留业务身份的链（如果把 `path` 也包进去，上层可能误判成普通 IO 错误）。调用方用 `switch` + `errors.Is` 表达分支，比字符串匹配可靠得多。

:::

### 第 6 题

说明下面两个函数的差别，并判断各自适合什么场景：`ParseInt`（返回 error）和 `MustParseInt`（失败时 panic）。再解释为什么 `recover` 不该用来「兜住」业务错误。

```go
func MustParseInt(s string) int {
    n, err := strconv.Atoi(s)
    if err != nil {
        panic(fmt.Sprintf("MustParseInt(%q): %v", s, err))
    }
    return n
}
```

::: details 第 6 题参考答案

**差别**：`ParseInt` 把「输入可能非法」当成正常情况，交给调用方处理；`MustParseInt` 断言「这个输入一定合法」，一旦不成立就 panic，属于**程序员错误**的信号。

**适用场景**：

- `MustParseInt` 只适合输入在编译期或启动期就确定、失败即说明代码写错了的地方，比如包级变量的初始化、测试用例、`regexp.MustCompile` 这类常量模式；
- 处理用户输入、配置文件、网络数据时必须用返回 error 的版本。

**为什么 `recover` 不该兜业务错误**：

1. panic 抛出的是「我假设这里不会失败」，用它表达业务失败会让代码的失败路径不可见；
2. `recover` 只能拦同一个 goroutine 的 panic，跨 goroutine 依然会崩溃，兜底并不完整；
3. 一旦用 `recover` 掩盖了空指针、越界这类 bug，程序会带病继续运行，问题被推迟到更难排查的地方。

**结论**：`panic` 留给「程序写错了」，`error` 留给「世界不符合预期」；`recover` 只放在服务入口、插件调用这类必须活下来的边界上。

:::

---

## 小结

- **`error` 是只有一个方法的接口**：任何实现了 `Error() string` 的类型都是错误，`nil` 表示没有错误；接口里的 typed nil 不等于 `nil`。
- **创建错误有三种方式**：包级哨兵 `ErrXxx`（可被 `errors.Is` 识别）、`fmt.Errorf` + `%w`（补上下文并保留链）、就地 `errors.New`（只当文案用）。
- **错误必须立即处理或上抛**：`_ = err` 要有理由，忽略 `strconv.Atoi` 的错误再使用返回值，会拿到 0 这样「看起来合法」的脏数据。
- **`%w` 才是链，`%v` 只是文本**：两者输出可能完全一样，但 `%v` 会让 `errors.Is` / `errors.As` 全部失效。
- **判断身份用 `errors.Is`，取结构化信息用 `errors.As`**：`==` 只看最外层，直接类型断言在包装后必然失败，匹配文案则会随措辞变化而失效。
- **多个失败原因用 `errors.Join`**：它保留全部子错误、实现 `Unwrap() []error`，`Join()` 无参数时返回 `nil`。
- **自定义错误类型三件套**：`Error()` 说明发生了什么、字段提供结构化信息、`Unwrap()` 保留错误链；接收者类型要和 `errors.As` 的目标保持一致。
- **错误与日志分工明确**：中间层只包装、最外层记录一次；错误信息小写、无句号、说清「做了什么 + 为什么失败」。
- **`panic` 用于程序 bug，`recover` 只放在边界**：`defer` + 命名返回值既能收集 `Close` 的错误，也是唯一能拦住 panic 的位置。

下一章进入包、模块与依赖管理，看看这些错误类型和哨兵如何在**包之间**共享：哪些该导出、`internal` 包怎么用，以及 `go.mod`、语义化版本和 `replace` 在协作中的作用。
