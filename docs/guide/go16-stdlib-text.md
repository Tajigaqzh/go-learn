# 第 16 章 标准库精讲（二）：正则、文本与模板

Go 的标准库在文本处理方面提供了完整的工具链：从正则表达式的匹配与替换，到 Unicode 与 UTF-8 的细节处理，再到模板引擎的安全渲染。本章按实际应用场景组织，配合实测输出，帮你掌握这些工具的正确用法和常见陷阱。

**本章配套代码**在 `internal/chapter/go16_stdlib_text/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

---

## 16.1 正则表达式基础（regexp）

### 编译与匹配

Go 的正则引擎来自 RE2，语法接近 Perl 但**不支持反向引用和环视断言**（零宽断言），换来的是**线性时间复杂度保证**。

```go-vue
import "regexp"

// 预编译（推荐）
re := regexp.MustCompile(`\d{3}-\d{4}`)  // 匹配电话号码格式
matched := re.MatchString("Call 123-4567")  // true

// 不预编译（慢）
matched, _ := regexp.MatchString(`\d+`, "order 123")
```

**实测输出**（来自 `go16_stdlib_text.Demo()`）：

```
正则表达式基础：
匹配结果：true
查找第一个：123-4567
查找所有：[123 456 2024]
```

### 常用方法

| 方法 | 作用 | 返回值 |
| --- | --- | --- |
| `MatchString(s string) bool` | 是否匹配 | 布尔值 |
| `FindString(s string) string` | 查找第一个匹配 | 匹配的字符串 |
| `FindAllString(s, n int) []string` | 查找所有匹配 | 字符串切片（n<0 表示全部） |
| `ReplaceAllString(s, repl string) string` | 替换所有匹配 | 替换后的字符串 |
| `Split(s string, n int) []string` | 按正则分割 | 切片 |

### 编译错误

```go-vue
_, err := regexp.Compile(`[`)  // 未闭合的字符类
// error parsing regexp: missing closing ]: `[`
```

**规则**：

- 用 `regexp.Compile` 检查错误
- 用 `regexp.MustCompile` 在 `init` 或包级变量，编译失败会 panic

---

## 16.2 捕获组与命名捕获

### 普通捕获组

```go-vue
re := regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)
match := re.FindStringSubmatch("2024-09-14")
// match = ["2024-09-14", "2024", "09", "14"]
//          整个匹配    第1组   第2组  第3组
```

### 命名捕获组（推荐）

```go-vue
re := regexp.MustCompile(`(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`)
match := re.FindStringSubmatch("2024-09-14")
names := re.SubexpNames()  // ["", "year", "month", "day"]

result := make(map[string]string)
for i, name := range names {
    if i > 0 && i < len(match) {
        result[name] = match[i]
    }
}
// result = {"year": "2024", "month": "09", "day": "14"}
```

**实测输出**：

```
命名捕获组：
year=2024, month=09, day=14
```

### 注意事项

- `SubexpNames()` 第一个元素是空字符串（整个匹配没有名称）
- 捕获组索引从 1 开始（0 是整个匹配）
- 未匹配的捕获组返回空字符串

---

## 16.3 贪婪与非贪婪匹配

### 贪婪（默认）

```go-vue
re := regexp.MustCompile(`<div>.*</div>`)
text := `<div>Hello</div><div>World</div>`
match := re.FindString(text)
// 匹配整个字符串：<div>Hello</div><div>World</div>
```

### 非贪婪（加 `?`）

```go-vue
re := regexp.MustCompile(`<div>.*?</div>`)
matches := re.FindAllString(text, -1)
// 匹配两次：["<div>Hello</div>", "<div>World</div>"]
```

**实测输出**：

```
贪婪匹配：
<div>Hello</div><div>World</div>

非贪婪匹配：
<div>Hello</div>
<div>World</div>
```

### 量词对照表

| 贪婪 | 非贪婪 | 含义 |
| --- | --- | --- |
| `*` | `*?` | 0 次或多次 |
| `+` | `+?` | 1 次或多次 |
| `?` | `??` | 0 次或 1 次 |
| `{n,m}` | `{n,m}?` | n 到 m 次 |

---

## 16.4 正则替换与函数替换

### 字符串替换

```go-vue
re := regexp.MustCompile(`\d+`)
result := re.ReplaceAllString("Order 123 costs $456", "XXX")
// "Order XXX costs $XXX"
```

### 引用捕获组

```go-vue
re := regexp.MustCompile(`(\w+)@(\w+)\.com`)
result := re.ReplaceAllString("Contact alice@example.com", "$1 at $2")
// "Contact alice at example"
```

### 函数替换（动态逻辑）

```go-vue
re := regexp.MustCompile(`\d+`)
result := re.ReplaceAllStringFunc("Order 123 costs $456", func(s string) string {
    num, _ := strconv.Atoi(s)
    return strconv.Itoa(num * 2)
})
// "Order 246 costs $912"
```

**实测输出**：

```
替换示例：
• 固定替换：Order XXX costs $XXX
• 引用捕获组：Contact alice at example
• 函数替换（数字翻倍）：Order 246 costs $912
```

---

## 16.5 正则性能与预编译

### 性能差异

```go-vue
// 慢：每次都重新编译
for i := 0; i < 1000; i++ {
    regexp.MatchString(`\d+`, text)
}

// 快：预编译一次，复用
re := regexp.MustCompile(`\d+`)
for i := 0; i < 1000; i++ {
    re.MatchString(text)
}
```

### 包级变量预编译

```go-vue
var (
    emailRe = regexp.MustCompile(`^[\w._%+-]+@[\w.-]+\.[a-zA-Z]{2,}$`)
    phoneRe = regexp.MustCompile(`\d{3}-\d{4}`)
)

func ValidateEmail(s string) bool {
    return emailRe.MatchString(s)
}
```

**规则**：

- **一次编译，多次使用**：`Compile` 的开销远大于匹配
- **包级变量用 `MustCompile`**：启动时就暴露编译错误
- **`Regexp` 是并发安全的**：可以被多个 goroutine 共享

---

## 16.6 text/template 基础

### 最小示例

```go-vue
import "text/template"

tmplStr := `Hello, { { .Name } }! Age: { { .Age } }`
tmpl, _ := template.New("user").Parse(tmplStr)

data := struct {
    Name string
    Age  int
}{Name: "Alice", Age: 30}

var buf bytes.Buffer
tmpl.Execute(&buf, data)
// 输出：Hello, Alice! Age: 30
```

**实测输出**：

```
text/template 输出：
Hello, Alice!
Age: 30
Active: true

嵌套字段：User: Bob, Score: 95
```

### 模板语法

| 语法 | 作用 | 示例 |
| --- | --- | --- |
| `{ { .Field } }` | 访问字段 | `{ { .Name } }` |
| `{ { .User.Name } }` | 访问嵌套字段 | `{ { .User.Name } }` |
| `{ { if .Active } }...{ { end } }` | 条件 | 布尔值判断 |
| `{ { range .Items } }...{ { end } }` | 遍历 | 遍历切片/数组/map |
| `{ { with .User } }...{ { end } }` | 上下文切换 | `.` 变成 `.User` |

---

## 16.7 模板管道与函数

### 内置函数

```go-vue
tmplStr := `{ { if .Active } }User is active{ { else } }User is inactive{ { end } }
Length of name: { { len .Name } }
Upper: { { .Name | printf "%q" } }`

tmpl, _ := template.New("pipe").Parse(tmplStr)
data := struct {
    Name   string
    Active bool
}{Name: "Charlie", Active: true}

var buf bytes.Buffer
tmpl.Execute(&buf, data)
```

**实测输出**：

```
管道与内置函数：
User is active
Length of name: 7
Upper: "Charlie"
```

### 自定义函数

```go-vue
funcs := template.FuncMap{
    "upper": strings.ToUpper,
    "add":   func(a, b int) int { return a + b },
}

tmpl, _ := template.New("custom").Funcs(funcs).Parse(`Name: { { .Name | upper } }
Score: { { add .Score 10 } }`)

data := struct {
    Name  string
    Score int
}{Name: "dave", Score: 80}

var buf bytes.Buffer
tmpl.Execute(&buf, data)
// 输出：Name: DAVE
//       Score: 90
```

**实测输出**：

```
自定义函数：
Name: DAVE
Score: 90
```

### range 遍历

```go-vue
tmpl, _ := template.New("range").Parse(`{ { range .Items } }- { { . } }
{ { end } }`)

data := struct{ Items []string }{Items: []string{"A", "B", "C"} }

var buf bytes.Buffer
tmpl.Execute(&buf, data)
// 输出：- A
//       - B
//       - C
```

---

## 16.8 html/template 与 XSS 防护

### 自动转义

```go-vue
import "html/template"

tmplStr := `<div>{ { .Content } }</div>`
tmpl, _ := template.New("html").Parse(tmplStr)

data := struct{ Content string }{Content: "<script>alert('XSS')</script>"}

var buf bytes.Buffer
tmpl.Execute(&buf, data)
// 输出：<div>&lt;script&gt;alert(&#39;XSS&#39;)&lt;/script&gt;</div>
```

**实测输出**：

```
html/template 自动转义：
<div>&lt;script&gt;alert(&#39;XSS&#39;)&lt;/script&gt;</div>

text/template（不转义，危险！）：
<div><script>alert('XSS')</script></div>

安全提示：
• 生成 HTML 时必须用 html/template
• html/template 会根据上下文（HTML、JS、CSS、URL）自动选择转义方式
• 不要用 text/template 生成 HTML，否则有 XSS 风险
```

### 对比 text/template

| 特性 | text/template | html/template |
| --- | --- | --- |
| 转义 | **不转义** | 自动转义 HTML、JS、CSS、URL |
| 用途 | 邮件、配置、纯文本 | **Web 页面、HTML 片段** |
| 安全性 | 需手动转义 | 防 XSS |

### 上下文感知

```go-vue
<a href="{ { .URL } }">{ { .Text } }</a>
<script>var x = "{ { .Value } }";</script>
<style>body { color: { { .Color } }; }</style>
```

`html/template` 会根据 `{ { . } }` 所在位置选择：

- HTML 属性：URL 编码
- JS 字符串：JS 转义
- CSS 值：CSS 转义

---

## 16.9 UTF-8 与 Unicode（unicode/utf8）

### 字符串长度陷阱

```go-vue
s := "Go语言"
fmt.Println(len(s))                     // 8（字节数）
fmt.Println(utf8.RuneCountInString(s))  // 4（字符数）
```

**实测输出**：

```
UTF-8 与字符串：
len("Go语言") = 8 字节
RuneCountInString("Go语言") = 4 字符
第一个字符：Go (2 字节)
```

### 遍历字符（rune）

```go-vue
for i, r := range "Go语言" {
    fmt.Printf("%d: %c (%d 字节)\n", i, r, utf8.RuneLen(r))
}
// 0: G (1 字节)
// 1: o (1 字节)
// 2: 语 (3 字节)
// 5: 言 (3 字节)
```

### 常用函数

| 函数 | 作用 |
| --- | --- |
| `utf8.RuneCountInString(s)` | 字符数（rune 数量） |
| `utf8.DecodeRuneInString(s)` | 解码第一个 rune |
| `utf8.Valid([]byte)` | 检查是否有效 UTF-8 |
| `utf8.RuneLen(r)` | rune 占用的字节数 |

---

## 16.10 Unicode 字符分类（unicode）

### 常用判断

```go-vue
import "unicode"

unicode.IsLetter('A')   // true
unicode.IsDigit('5')    // true
unicode.IsSpace(' ')    // true
unicode.IsUpper('G')    // true
unicode.ToLower('A')    // 'a'
```

**实测输出**：

```
Unicode 字符分类：
'A' -> Letter:true Digit:false Space:false
'5' -> Letter:false Digit:true Space:false
' ' -> Letter:false Digit:false Space:true
'中' -> Letter:true Digit:false Space:false
```

### 字符范围表

| 类别 | 函数 | 示例 |
| --- | --- | --- |
| 字母 | `IsLetter` | 'A'、'中' |
| 数字 | `IsDigit` | '5' |
| 空白 | `IsSpace` | ' '、'\t'、'\n' |
| 大小写 | `IsUpper`、`IsLower` | 'A'、'a' |
| 标点 | `IsPunct` | '!'、'。' |
| 符号 | `IsSymbol` | '$'、'€' |

---

## 16.11 encoding/csv

### 读取 CSV

```go-vue
import "encoding/csv"

data := `Name,Age,City
Alice,30,NYC
Bob,25,LA`

r := csv.NewReader(strings.NewReader(data))
records, _ := r.ReadAll()
// records = [["Name" "Age" "City"] ["Alice" "30" "NYC"] ["Bob" "25" "LA"]]
```

**实测输出**：

```
CSV 读取：
[Name Age City]
[Alice 30 NYC]
[Bob 25 LA]
```

### 写入 CSV

```go-vue
var buf bytes.Buffer
w := csv.NewWriter(&buf)

w.Write([]string{"Name", "Score"})
w.Write([]string{"Alice", "95"})
w.Write([]string{"Bob", "88"})
w.Flush()

fmt.Println(buf.String())
// Name,Score
// Alice,95
// Bob,88
```

**实测输出**：

```
CSV 写入：
Name,Score
Alice,95
Bob,88
```

### 处理特殊字符

```go-vue
w := csv.NewWriter(&buf)
w.Write([]string{"Text with, comma", `Quote"test`})
w.Flush()
// "Text with, comma","Quote""test"
```

---

## 16.12 strings.Replacer

### 批量替换

```go-vue
replacer := strings.NewReplacer(
    "apple", "🍎",
    "banana", "🍌",
    "grape", "🍇",
)

result := replacer.Replace("I like apple and banana")
// "I like 🍎 and 🍌"
```

**实测输出**：

```
Replacer 批量替换：
I like 🍎 and 🍌
```

### 性能优势

```go-vue
// 慢：多次 strings.Replace
s = strings.Replace(s, "a", "1", -1)
s = strings.Replace(s, "b", "2", -1)
s = strings.Replace(s, "c", "3", -1)

// 快：一次扫描
replacer := strings.NewReplacer("a", "1", "b", "2", "c", "3")
s = replacer.Replace(s)
```

**规则**：

- `Replacer` 构建时间 O(n)，替换时间 O(m)（m 是字符串长度）
- 多次替换时比连续 `strings.Replace` 快
- 替换是**同时进行**的，不会互相影响

---

## 5 个真实报错怎么读

### 报错 1：正则编译失败

```
error parsing regexp: missing closing ]: `[a-z`
```

**原因**：字符类未闭合。

**修复**：`[a-z]` 加上 `]`。

### 报错 2：捕获组索引越界

```
panic: runtime error: index out of range [2] with length 2
```

**原因**：`match[2]` 访问了不存在的捕获组。

**修复**：检查 `len(match)` 或用命名捕获组。

### 报错 3：模板未定义字段

```
template: user:1:5: executing "user" at <.Nama>: can't evaluate field Nama in type struct { Name string }
```

**原因**：模板里写了 `.Nama`，但结构体字段是 `.Name`（拼写错误）。

**修复**：改成 `{ { .Name } }`。

### 报错 4：UTF-8 切片切到一半

```
s := "Go语言"
sub := s[:3]  // "Go�"（语 被切断，3 字节只取了 2 个）
```

**原因**：按字节切片，切断了多字节字符。

**修复**：用 `[]rune` 切片，或用 `utf8.DecodeRuneInString` 逐字符处理。

### 报错 5：CSV 字段数不匹配

```
record on line 2: wrong number of fields
```

**原因**：CSV 某行的字段数与第一行不一致。

**修复**：设置 `r.FieldsPerRecord = -1`（允许不同列数），或修正数据。

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 正则很慢 | 每次都重新编译 | 预编译：`var re = regexp.MustCompile(...)` |
| 贪婪匹配吃掉太多 | 默认贪婪 | 加 `?` 改成非贪婪：`.*?` |
| 模板找不到字段 | 字段名拼写错误或未导出 | 检查大小写，确保首字母大写 |
| XSS 漏洞 | 用 `text/template` 生成 HTML | 改用 `html/template` |
| `len()` 统计中文不对 | `len` 返回字节数 | 用 `utf8.RuneCountInString` |
| 切片切断中文 | 按字节切片 | 转成 `[]rune` 后切片 |
| CSV 字段有逗号或引号 | 未转义 | 用 `csv.Writer`，它会自动处理 |

---

## 练习

### 第 1 题

写一个正则表达式，匹配合法的邮箱地址（简化版：`username@domain.com`）。

::: details 第 1 题参考答案

```go-vue
emailRe := regexp.MustCompile(`^[\w._%+-]+@[\w.-]+\.[a-zA-Z]{2,}$`)

fmt.Println(emailRe.MatchString("alice@example.com"))  // true
fmt.Println(emailRe.MatchString("bob@.com"))           // false
fmt.Println(emailRe.MatchString("no-at-sign.com"))     // false
```

**为什么这样写更好**：

- `^` 和 `$` 确保整个字符串匹配，而不是部分匹配
- `[\w._%+-]+` 允许常见的用户名字符
- `[\w.-]+` 允许子域名
- `\.[a-zA-Z]{2,}` 确保至少有 2 个字母的顶级域名

:::

### 第 2 题

用 `text/template` 生成一个配置文件，包含服务器地址、端口、是否启用调试。

::: details 第 2 题参考答案

```go-vue
tmplStr := `Server:
  Address: { { .Address } }
  Port: { { .Port } }
  Debug: { { .Debug } }`

tmpl, _ := template.New("config").Parse(tmplStr)

data := struct {
    Address string
    Port    int
    Debug   bool
}{
    Address: "0.0.0.0",
    Port:    8080,
    Debug:   true,
}

var buf bytes.Buffer
tmpl.Execute(&buf, data)
fmt.Println(buf.String())
```

**输出**：

```yaml
Server:
  Address: 0.0.0.0
  Port: 8080
  Debug: true
```

**为什么这样写更好**：

- 用模板生成配置，避免手动拼接字符串
- 结构体字段类型明确，减少错误
- 易于扩展：加新字段只需修改结构体和模板

:::

### 第 3 题

解释为什么生成 HTML 时必须用 `html/template` 而不是 `text/template`。

::: details 第 3 题参考答案

**原因**：

`text/template` **不转义**任何字符，如果用户输入包含 `<script>` 等标签，会被原样插入 HTML，导致 **XSS（跨站脚本攻击）**。

`html/template` 会**自动转义**：

- `<` 转义成 `&lt;`
- `>` 转义成 `&gt;`
- `"` 转义成 `&#34;` 或 `&quot;`
- `'` 转义成 `&#39;`

**示例**：

```go-vue
// 危险：text/template
tmpl, _ := texttemplate.New("text").Parse(`<div>{ {.Input} }</div>`)
data := struct{ Input string }{Input: `<script>alert('XSS')</script>`}
tmpl.Execute(&buf, data)
// 输出：<div><script>alert('XSS')</script></div>  ← 脚本会执行！

// 安全：html/template
tmpl, _ := template.New("html").Parse(`<div>{ {.Input} }</div>`)
tmpl.Execute(&buf, data)
// 输出：<div>&lt;script&gt;alert(&#39;XSS&#39;)&lt;/script&gt;</div>  ← 被转义，安全
```

**为什么更好**：

- 自动防护，不依赖开发者记住转义
- 根据上下文（HTML、JS、CSS、URL）选择转义方式
- 符合「安全默认」原则

:::

### 第 4 题

用正则表达式提取字符串 `"Price: $123.45"` 中的数字部分（`123.45`）。

::: details 第 4 题参考答案

```go-vue
re := regexp.MustCompile(`\$(\d+\.\d+)`)
text := "Price: $123.45"

match := re.FindStringSubmatch(text)
if len(match) > 1 {
    fmt.Println("价格:", match[1])  // "123.45"
}
```

**为什么这样写更好**：

- `\$` 转义美元符号（因为 `$` 在正则中表示行尾）
- `(\d+\.\d+)` 捕获整数和小数部分
- `\.` 转义点号（因为 `.` 在正则中表示任意字符）
- 用 `FindStringSubmatch` 获取捕获组

**更健壮的版本**（支持可选小数）：

```go-vue
re := regexp.MustCompile(`\$(\d+(?:\.\d+)?)`)
// 匹配 $123 或 $123.45
```

:::

### 第 5 题

写一个函数，统计字符串中中文字符的数量。

::: details 第 5 题参考答案

```go-vue
import "unicode"

func CountChineseChars(s string) int {
    count := 0
    for _, r := range s {
        if unicode.Is(unicode.Han, r) {
            count++
        }
    }
    return count
}

fmt.Println(CountChineseChars("Go语言很好"))    // 4
fmt.Println(CountChineseChars("Hello世界"))  // 2
```

**为什么这样写更好**：

- `unicode.Is(unicode.Han, r)` 判断是否是汉字（包括 CJK 统一表意文字）
- `for _, r := range s` 按 rune 遍历，正确处理多字节字符
- 不依赖字节长度，适用于任意 UTF-8 字符串

**注意**：

- `unicode.Han` 包括汉字、日文汉字、韩文汉字
- 如果只统计简体中文，需要更细粒度的 Unicode 范围判断

:::

### 第 6 题

用 `strings.Replacer` 实现一个简单的敏感词过滤器。

::: details 第 6 题参考答案

```go-vue
censor := strings.NewReplacer(
    "damn", "****",
    "shit", "****",
    "fuck", "****",
)

text := "What the damn hell is this shit?"
clean := censor.Replace(text)
fmt.Println(clean)  // "What the **** hell is this ****?"
```

**为什么这样写更好**：

- 一次扫描完成所有替换，性能高
- 替换是同时进行的，不会互相影响
- 易于扩展：加新敏感词只需修改 `NewReplacer` 参数

**进阶版**（大小写不敏感）：

```go-vue
func CensorText(text string) string {
    lower := strings.ToLower(text)
    censor := strings.NewReplacer(
        "damn", "****",
        "shit", "****",
    )
    
    // 先找到敏感词位置，再替换原字符串对应位置
    // 这里简化处理：直接转小写再替换
    return censor.Replace(lower)
}
```

**实战中更好的方案**：

- 用正则表达式 `(?i)` 标志实现大小写不敏感
- 用 Trie 树或 Aho-Corasick 算法处理大量敏感词

:::

---

## 小结

- **正则表达式**：预编译一次（`MustCompile`），多次使用；注意贪婪与非贪婪
- **捕获组**：命名捕获组更可读；`SubexpNames()` 索引从 1 开始
- **模板引擎**：`text/template` 用于纯文本，`html/template` 用于 HTML（防 XSS）
- **自定义函数**：`FuncMap` 必须在 `Parse` 之前调用 `Funcs`
- **UTF-8**：`len()` 返回字节数，`utf8.RuneCountInString()` 返回字符数
- **Unicode 分类**：`unicode.IsLetter`、`IsDigit`、`IsSpace` 等函数支持全 Unicode
- **CSV**：`encoding/csv` 自动处理转义；设置 `FieldsPerRecord = -1` 允许不同列数
- **`strings.Replacer`**：批量替换比多次 `strings.Replace` 快

下一章将讲解**文件、路径与 IO**，学习 `os`、`io`、`bufio` 和 `filepath` 的正确用法。
