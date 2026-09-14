# 第 8 章 · 字符串、字节与 Unicode

前面几章我们把数据装进了变量、切片和映射，但还有一类数据几乎每个程序都在处理：文本。Go 里文本用 `string` 表示，看起来是个再简单不过的类型——直到你把中文写进代码，然后发现 `len(s)` 给出的数字和肉眼数出来的字数对不上，`s[0]` 拿到的不是第一个汉字，`s[:1]` 也切不出一个完整的字符。

这些现象来自同一个事实：**Go 的 string 是一串只读的字节，不是一串字符**。文本含义来自编码约定，Go 源码和运行时统一采用 UTF-8。只要能在「字节视角」和「字符视角」之间自由切换，这一类坑基本会自动消失：什么时候该用 `len`，什么时候该用 `utf8.RuneCountInString`，什么时候该把 string 转成 `[]rune` 或 `[]byte`，`strings.Builder` 和 `bytes.Buffer` 该选哪个，答案都由这个事实决定。

本章从字节层开始，讲到 UTF-8 与 rune，再把 `strings`、`strconv`、`unicode/utf8` 三个高频标准库过一遍，最后用真实基准测试比较四种字符串拼接写法。读完之后，你应该能一眼看出「截断中文得到乱码」「`string(65)` 打印出 `A`」这类问题的成因。

本章配套代码在 `internal/chapter/go08_strings/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

---

## 8.1 string 是只读的字节序列

先给结论：`len(s)` 返回的是**字节数**，`s[i]` 返回的是第 `i` 个 **`byte`**（`uint8`），字符串本身**不可修改**。所有字符级的操作，都要先经过一次显式转换。

```go
s := "Go 学习"

fmt.Println(len(s)) // 9：3 个 ASCII 字节 + 2 个汉字 × 3 字节
fmt.Println(s[0])   // 71，'G' 的字节值
fmt.Println(s[2])   // 32，空格的字节值
```

`"Go 学习"` 一共 5 个字符，但占 9 个字节，因为每个汉字在 UTF-8 里要 3 个字节。

### 实测输出

```text
s = "Go 学习"
len(s) = 9（字节数，不是字符数）
s[0] = 71，%c 打印是 G，%q 打印是 'G'
s[0] 的类型是 uint8（byte 是 uint8 的别名）
s 的每个字节：47 6F 20 E5 AD A6 E4 B9 A0
```

把字节逐个打印出来，能看到「学」被拆成了 `E5 AD A6` 三个字节、「习」是 `E4 B9 A0`。这就是后面所有「中文下标坑」的根源。

### 三条性质

| 性质 | 说明 | 后果 |
| --- | --- | --- |
| 不可变 | 一旦创建，内容不能被修改 | `s[0] = 'g'` 无法编译，只能生成新字符串 |
| 可比较 | `==`、`<`、`>` 都合法 | 比较的是字节序列，和 UTF-8 的码点顺序一致 |
| 有零值 | 零值是 `""`，没有 nil 字符串 | 判断「有没有内容」用 `s == ""`，不用 `s == nil` |

### 字符串比较按字节序

```go
fmt.Println(s == "Go 学习") // true
fmt.Println("Z" < "a")      // true，'Z'=0x5A 小于 'a'=0x61
fmt.Println("go" < "学习")  // true，'g'=0x67 小于 0xE5
```

实测输出：

```text
s == "Go 学习" → true（字符串可以直接比较）
"Z" < "a" → true（按字节值比较，'Z'=0x5A < 'a'=0x61）
"go" < "学习" → true（UTF-8 字节序与码点序一致）
```

UTF-8 有个很有用的性质：**编码后的字节序与码点序一致**。所以按字节比较字符串，得到的顺序和按字符比较一致，排序时不用先转成 `[]rune`。

### 边界与坑

- 字面量里的转义序列直接决定字节内容：`"\xe4\xb8\xad"` 和 `"中"` 是同一个字符串，而 `"\xff"` 是长度为 1 的非法 UTF-8 字符串。
- 字符串可以和 `[]byte` 互相转换，但转换是**拷贝**（见 8.9）；`len(s) == 0` 与 `s == ""` 等价，习惯上用后者。

---

## 8.2 UTF-8、byte 与 rune

结论：`byte` 是「一个字节」，`rune` 是「一个字符（码点）」，两者都是整数类型，但语义完全不同。一个 rune 编码成 1 到 4 个 byte。

```go
cn := '中' // rune 字面量，类型是 int32，值是 0x4E2D
fmt.Println(cn) // 20013，打印出来是数字
fmt.Printf("%c %U\n", cn, cn) // 中 U+4E2D
```

注意 `'中'` 用单引号，是 rune 字面量；`"中"` 用双引号，是 string 字面量。前者可以直接参与算术运算，后者不行。

### UTF-8 的编码宽度

| 码点范围 | 编码长度 | 典型字符 |
| --- | --- | --- |
| U+0000 – U+007F | 1 字节 | ASCII：`A`、`0`、空格 |
| U+0080 – U+07FF | 2 字节 | 拉丁扩展、希腊字母 |
| U+0800 – U+FFFF | 3 字节 | 中日韩文字：`中`、`学` |
| U+10000 – U+10FFFF | 4 字节 | Emoji：`🚀` |

### 类型对照

| 类型 | 底层类型 | 含义 | 出现位置 |
| --- | --- | --- | --- |
| `byte` | `uint8` | 一个原始字节（0–255） | `s[i]`、`[]byte` |
| `rune` | `int32` | 一个码点（字符编号） | `range s`、`[]rune` |
| `string` | 只读字节序列 | 编码后的文本 | 字面量、函数参数 |

`byte` 和 `rune` 都是别名：`byte` 是 `uint8` 的别名，`rune` 是 `int32` 的别名，所以类型名可以互换使用。

### 实测输出

```text
'中' 的码点 = U+4E2D（十进制 20013），utf8.RuneLen = 3 字节
'🚀' 的码点 = U+1F680（十进制 128640），utf8.RuneLen = 4 字节
utf8.UTFMax = 4（一个字符最多用这么多字节）
byte 是 uint8（1 字节，取值 0-255），rune 是 int32（4 字节，表示一个码点）
len("中") = 3，utf8.RuneCountInString("中") = 1
UTF-8 按码点大小分档，宽度从 1 到 4 字节不等：
  A 占 1 字节
  中 占 3 字节
  € 占 3 字节
  🚀 占 4 字节
```

### 边界与坑

- `len(s)` 数的是字节，`utf8.RuneCountInString(s)` 数的是字符，写展示逻辑时别用错。
- `string(65)` 得到的是 `"A"`，不是 `"65"`——因为这是「把码点 65 编码成文本」，详见本章报错章节。
- UTF-8 自带同步信息：首字节的高位标记了该字符占几个字节，所以可以边扫边解码，不需要从头开始；这也是 `range` 能按字符遍历的原理。

---

## 8.3 range 遍历字符串按 rune 迭代

结论：`for i, r := range s` 中，`i` 是**字节下标**，`r` 是 **rune**。这是唯一不需要额外分配就能按字符遍历字符串的方式。

```go
s := "Go 学习"
for i, r := range s {
    fmt.Printf("%d → %q\n", i, r)
}
```

输出是 `0 → 'G'`、`1 → 'o'`、`2 → ' '`、`3 → '学'`、`6 → '习'`：下标跳过了 4 和 5，因为「学」占 3 个字节，下一个字符从字节 6 开始。

### 实测输出

```text
range 遍历：下标是字节偏移，值是 rune
  字节偏移  0 → 'G'（U+0047，宽 1 字节）
  字节偏移  1 → 'o'（U+006F，宽 1 字节）
  字节偏移  2 → ' '（U+0020，宽 1 字节）
  字节偏移  3 → '学'（U+5B66，宽 3 字节）
  字节偏移  6 → '习'（U+4E60，宽 3 字节）
range 数出来的字符数 = 5，len 数出来的字节数 = 9
```

### 三种遍历方式怎么选

| 写法 | 迭代单位 | 是否分配 | 适用场景 |
| --- | --- | --- | --- |
| `for i := 0; i < len(s); i++` | `byte` | 否 | 处理二进制、协议解析、逐字节校验 |
| `for i, r := range s` | `rune`（下标是字节偏移） | 否 | 处理文本、统计字符、判断内容 |
| `for _, r := range []rune(s)` | `rune`（下标是字符序号） | 是 | 需要按下标随机访问字符 |

`range` 内部就是「解码一个 rune、前进相应的字节数」。遇到非法字节时，它不会 panic，而是把该字节替换成 `U+FFFD` 并前进 1 字节——这是 Go 对错误文本的容错策略。

### 边界与坑

- 别把 `range` 拿到的 `i` 当成字符序号；要字符序号就得自己计数。
- 需要「第 n 个字符」这种随机访问时，先转成 `[]rune`；反复转换会反复分配。
- 修改字符串内容不可能，`range` 只给值；要改就先转 `[]rune` 或 `[]byte` 再转回来。

---

## 8.4 中文下标切分的坑

结论：`s[a:b]` 切的是**字节**，切点必须落在字符的首字节上，否则得到的是非法 UTF-8。

```go
s := "Go 学习笔记"

fmt.Println(s[3:6]) // "学"，切点正好在字符边界上
fmt.Println(s[4:6]) // 从「学」的中间切开，结果不再是合法文本
```

### 实测输出

```text
s = "Go 学习笔记"，len(s) = 15 字节
s[3:6] = "学"，utf8.ValidString = true
s[4:6] = "\xad\xa6"，字节为 AD A6，utf8.ValidString = false
[]rune(s) 长度 = 7，前 4 个字符 = "Go 学"
TruncateBytes(s, 4) = "Go "（切点回退到字符边界）
TruncateBytes(s, 6) = "Go 学"
TruncateRunes(s, 5) = "Go 学习"（按字符数截断）
ReverseRunes(s) = "记笔习学 oG"
```

注意 `s[4:6]` 只切掉了「学」的后两个字节，于是 `AD A6` 单独拿出来就是一段非法编码。这种字符串不会报错、不会 panic，只是**静默地坏掉**：`%q` 会把它转义成 `\xad\xa6`，直接打印时终端通常显示成乱码方块，写进日志或数据库后很难排查。

### 正确做法一：按字符切

```go
runes := []rune(s)
firstFour := string(runes[:4]) // "Go 学"
```

`[]rune(s)` 会把字节序列完整解码成一串码点，切分就是普通的切片操作。代价是一次分配和一次解码，只适合「先转再切、切完就完」的场景。

### 正确做法二：按字节切但回退到字符边界

后端接口常常按**字节上限**截断（比如数据库字段限制、日志行长限制），这时候不能直接转 `[]rune`，需要把字节切点回退到最近的字符首字节。本章示例包里的 `TruncateBytes` 就是干这个的：

```go
// TruncateBytes 把 s 截断到不超过 maxBytes 个字节，并保证结果仍是合法 UTF-8。
func TruncateBytes(s string, maxBytes int) string {
    if maxBytes <= 0 {
        return ""
    }
    if len(s) <= maxBytes {
        return s
    }
    end := maxBytes
    for end > 0 && !utf8.RuneStart(s[end]) {
        end--
    }
    return s[:end]
}
```

关键点是 `utf8.RuneStart(b)`：它判断某个字节**是否可能是一个字符的首字节**（UTF-8 的后续字节形如 `10xxxxxx`，首字节不是）。从切点往前退最多 3 个字节，一定能落到边界上。

### 正确做法三：按字符数截断

如果产品需求是「最多显示 10 个字」，那就该按 rune 数截断，而不是按字节。示例包里的 `TruncateRunes` 就是这个版本：

```go
// TruncateRunes 把 s 截断到最多 maxRunes 个字符。
func TruncateRunes(s string, maxRunes int) string {
    if maxRunes <= 0 {
        return ""
    }
    count := 0
    for i := range s { // i 是每个字符的起始字节下标
        if count == maxRunes {
            return s[:i]
        }
        count++
    }
    return s
}
```

`for i := range s` 只取下标、不取值，同样按 rune 前进，而且**不产生任何分配**。对于「只截前几个字」这种需求，它比 `[]rune(s)[:maxRunes]` 更省：后者要为整串字符分配一次切片，哪怕只取前 5 个字。

### 边界与坑

- `s[0:1]` 在英文里没问题，在中文里几乎总是错的——这类 bug 在只测试英文时不会暴露。
- 反转字符串同理：按字节反转会彻底破坏编码，必须按 rune 反转（见本章练习）。
- 截断后如果需要拼接省略号，记得先截断再拼，避免把 `…` 也算进字节预算。

---

## 8.5 unicode/utf8 包

结论：`unicode/utf8` 提供「不解码就能检查、计量、定位」字符的工具，处理非法输入时也不会 panic。常用函数如下。

| 函数 | 作用 | 备注 |
| --- | --- | --- |
| `utf8.RuneCountInString(s)` | 数字符（rune）个数 | 遇到非法字节也算 1 个 |
| `utf8.ValidString(s)` | 判断整串是否合法 UTF-8 | 只看编码，不看内容 |
| `utf8.ValidRune(r)` | 判断码点是否合法 | `U+FFFD` 是合法码点 |
| `utf8.RuneLen(r)` | 某个码点编码后占几个字节 | 非法码点返回 -1 |
| `utf8.RuneStart(b)` | 该字节是否是字符首字节 | 用于安全截断 |
| `utf8.DecodeRuneInString(s)` | 解码首个字符，返回 rune 与宽度 | 失败返回 `RuneError`、宽度 1 |
| `utf8.DecodeLastRuneInString(s)` | 解码末尾字符 | 取「最后一个字」时用 |
| `utf8.AppendRune(buf, r)` | 把码点编码后追加到 `[]byte` | 免去 `[]byte(string(r))` |

### 实测输出

```text
s = "中文 abc 🚀"
len(s) = 15 字节，utf8.RuneCountInString(s) = 8 字符
utf8.ValidString(s) = true
截断的字节序列 E4 B8：ValidString = false
range 遍历非法序列，坏字节逐个变成 U+FFFD：[0]=U+FFFD [1]=U+FFFD
U+FFFD 自己编码要 3 字节，但解码失败时只前进 1 字节
utf8.RuneError = U+FFFD，utf8.ValidRune(utf8.RuneError) = true
DecodeRuneInString("学习") = 学（U+5B66），宽度 3
utf8.RuneStart("学" 的首字节 0xE5) = true
utf8.AppendRune(nil, '🚀') = F0 9F 9A 80
```

`string([]byte{0xE4, 0xB8})` 是人为构造的坏数据：它是「中」的前两个字节。`ValidString` 判定为 `false`，`range` 遍历它得到两个 `U+FFFD`，`DecodeRuneInString` 则返回 `U+FFFD` 和宽度 1。

注意最后两行：`U+FFFD` 自己编码要 3 个字节，但**解码失败时只前进 1 字节**，这是为了跳过错位字节、继续扫描后面的内容，而不是让整段文本失效。

### 这两个函数最常用

```go
// 判断输入是否是可展示的文本（比如用户昵称、JSON 字段）
if !utf8.ValidString(input) {
    return errors.New("输入不是合法的 UTF-8")
}

// 展示「长度」时按字符算，而不是按字节算
fmt.Printf("昵称长度：%d\n", utf8.RuneCountInString(nickname))
```

`utf8.ValidString` 的复杂度是 O(n)，但只在真正需要校验的边界上调用一次即可，不要放进循环里对同一字符串反复校验。

### 边界与坑

- `RuneCountInString` 不会因为非法字节而失败，它把每个坏字节都算作 1 个字符；要区分「合法文本」就得先用 `ValidString`。
- `utf8.RuneLen(-1)` 返回 -1，与「合法但宽度为 1」不是一回事，别把返回值当作字符宽度直接信任。
- 长度为 0 的字符串既合法也没有字符，`ValidString("")` 是 `true`。

---

## 8.6 strings 包：文本处理的主力

结论：日常的查找、切分、拼接、大小写转换都用 `strings`，它是纯函数式 API——所有函数都返回新值，不会修改入参（字符串本来也改不了）。

| 函数 | 作用 | 一句话提醒 |
| --- | --- | --- |
| `Contains(s, sub)` | 是否包含子串 | 判存在用它，别用 `Index(...) > 0` |
| `Index(s, sub)` | 子串首次出现的**字节**下标 | 找不到返回 -1 |
| `Count(s, sub)` | 子串出现次数 | `Count(s, "")` 返回字符数 + 1 |
| `HasPrefix` / `HasSuffix` | 前缀 / 后缀判断 | 比手写切片比较更清楚 |
| `Split(s, sep)` / `Fields(s)` | 按分隔符 / 按任意空白切分 | `Fields` 会丢掉空串 |
| `Join(parts, sep)` | 用分隔符拼接切片 | 拼一条字符串的首选 |
| `ReplaceAll(s, old, new)` | 全部替换 | 只换一次用 `Replace(s, old, new, 1)` |
| `TrimSpace` / `TrimPrefix` / `TrimSuffix` | 去掉空白 / 前后缀 | `TrimSpace` 能处理 `\t`、`\n`、`\r` |
| `ToUpper` / `ToLower` / `EqualFold` | 大小写转换与忽略大小写比较 | 比较用户输入用 `EqualFold` |
| `Cut(s, sep)` | 一次拿到分隔符前后两段 | 比 `Split` + 越界检查更省 |
| `CutPrefix` / `CutSuffix` | 去前缀并返回剩余部分 | 返回 `(rest, ok)`，无需再判断 |
| `ToValidUTF8(s, replacement)` | 把非法字节替换成指定文本 | 清洗外部输入时有用 |
| `SplitSeq` / `FieldsSeq` / `Lines` | 迭代器版本（Go 1.24+） | 边切边处理，不额外分配切片 |

### 实测输出

```text
原始："  Go, Go, Go! 学习 Go  "
TrimSpace → "Go, Go, Go! 学习 Go"
Contains(s, "学习") = true
Count(s, "Go") = 4
Index(s, "学习") = 12（返回字节下标，不是字符下标）
HasPrefix(s, "Go") = true，HasSuffix(s, "Go") = true
Index(s, "Rust") = -1（找不到返回 -1，不是 0）
Split(s, ", ") = ["Go" "Go" "Go! 学习 Go"]（3 段）
Join(parts, "|") = "Go|Go|Go! 学习 Go"
Fields(s) = ["Go," "Go," "Go!" "学习" "Go"]（按任意空白切分，并丢掉空串）
ReplaceAll(s, "Go", "Rust") = "Rust, Rust, Rust! 学习 Rust"
ToUpper(s) = "GO, GO, GO! 学习 GO"，ToLower(s) = "go, go, go! 学习 go"
EqualFold("GO", "go") = true（忽略大小写比较）
Cut(s, ",") = "Go" / " Go, Go! 学习 Go"，found = true
CutPrefix(s, "Go") = ", Go, Go! 学习 Go"，ok = true
ToValidUTF8(非法序列, "?") = "?"
Go 1.24 起提供迭代器版本，可以边切边处理，不额外分配切片：
  SplitSeq → "第一行"
  SplitSeq → "第二行"
```

几个值得记住的细节：

- `Count(s, "Go")` 数到 4 次，因为 `"Go!"` 里的 `Go` 也算一次。
- `Index` 返回的是**字节下标**：`"Go, Go, Go! "` 占 12 个字节，所以 `"学习"` 的下标是 12，而不是字符序号 3。
- `Fields` 和 `Split` 的差别在空串上：`Fields("a  b")` 得到 2 段，`Split("a  b", " ")` 得到 3 段。
- `Cut` 只找第一个分隔符，返回「前、后、是否找到」三个值，省掉了 `Split` 之后的长度检查。

### 一个高频 bug：用 Index 的返回值判断存在性

```go
// 错误：目标出现在开头时 Index 返回 0，条件不成立
if strings.Index(s, prefix) > 0 {
    fmt.Println("有前缀")
}

// 正确：判存在用 Contains，或者写 Index(...) >= 0
if strings.Contains(s, prefix) {
    fmt.Println("有前缀")
}
```

前缀判断还有更直接的 `HasPrefix` / `CutPrefix`。

### 边界与坑

- `strings` 的所有下标都是**字节下标**，把它当成字符下标就会错位。
- 大小写转换只在 Unicode 有明确定义时才有意义，对某些语言（如土耳其语）`ToLower` 并不能满足业务预期。
- `Split` 会一次性分配整个切片；只需要遍历时，用 `SplitSeq` 更省内存。

---

## 8.7 strconv：字符串与数字互转

结论：字符串 ↔ 数字的转换走 `strconv`，不要用 `fmt.Sprintf` 做性能敏感路径上的转换，更不要用 `string(n)` 把数字「转成字符串」。

| 函数 | 方向 | 说明 |
| --- | --- | --- |
| `strconv.Atoi` / `Itoa` | string ↔ int | 10 进制快捷方式，Atoi 返回 `(int, error)` |
| `strconv.ParseInt(s, base, bits)` | string → 整数 | 支持 2–36 进制，可限制位宽 |
| `strconv.FormatInt(i, base)` | 整数 → string | 转二进制、十六进制用它 |
| `strconv.ParseFloat` / `FormatFloat` | string ↔ 浮点 | 可指定精度与格式 |
| `strconv.ParseBool` | string → bool | 只接受 `1/0/t/f/true/false/TRUE/FALSE` |
| `strconv.Quote` / `Unquote` | string ↔ 带引号字面量 | 生成可读的日志、调试输出 |

### 实测输出

```text
Atoi("42") = 42，err = <nil>
Atoi("12a") = 0，err = strconv.Atoi: parsing "12a": invalid syntax
Itoa(42) = "42"
ParseInt("ff", 16, 64) = 255，err = <nil>
ParseInt 超出 int64 范围：err = strconv.ParseInt: parsing "99999999999999999999": value out of range
FormatInt(255, 16) = "ff"，FormatInt(8, 2) = "1000"
ParseFloat("3.14159", 64) = 3.14159，err = <nil>
ParseFloat("1e3", 64) = 1000
FormatFloat(3.14159, 'f', 2, 64) = "3.14"
Quote → "他说：\"你好\""
Unquote → "他说：\"你好\""，err = <nil>
QuoteRune('中') = '中'，QuoteRune('\n') = '\n'（不可见字符会转义）
用法约定：Atoi/Itoa 是 10 进制的快捷方式，其他进制和位宽用 ParseXxx/FormatXxx。
```

### 错误也要判断

`Atoi` 失败时返回 0 和一个 `*strconv.NumError`。解析错误分两类：语法错误（`invalid syntax`）和范围错误（`value out of range`）。用 `errors.Is` 可以区分：

```go
n, err := strconv.Atoi(input)
if err != nil {
    if errors.Is(err, strconv.ErrRange) {
        return fmt.Errorf("数值超出范围: %w", err)
    }
    return fmt.Errorf("不是合法的整数: %w", err)
}
```

### 边界与坑

- `string(65)` 得到 `"A"` 而不是 `"65"`；数字转字符串要用 `strconv.Itoa(65)` 或 `fmt.Sprint(65)`。
- `Atoi("007")` 得到 7，前导零会被丢掉；要保留原样就别做转换。
- `ParseFloat` 接受 `NaN`、`Inf`，也接受 `1e3` 这种科学计数法；如果业务不接受，要自己再校验。
- `Unquote(s)` 要求 `s` 是带引号的字面量，直接传 `hello` 会报 `invalid syntax`。

---

## 8.8 四种拼接写法与真实基准

结论：少量、编译期已知的拼接用 `+`；拼接一个切片用 `strings.Join`；循环里增量构建用 `strings.Builder`；需要 `io.Writer` 能力时用 `bytes.Buffer`。**在循环里用 `+=` 是 O(n²)**，数据一多就会变成性能瓶颈。

```go
// 反例：每次都重新分配并拷贝已有内容
out := ""
for _, p := range parts {
    out += p
}

// 正例：一次算清总长度，只分配一次
var sb strings.Builder
sb.Grow(total)
for _, p := range parts {
    sb.WriteString(p)
}
return sb.String()
```

原因很直接：字符串不可变，`out += p` 必须「分配新缓冲区 + 拷贝旧内容 + 拷贝新片段」。循环 n 次就拷贝了大约 n²/2 个字节。

### 四种写法结果一致（实测输出）

```text
拼接 200 个片段，四种写法的结果长度都是 200
  + 运算符      与 Join 一致：true
  strings.Join  与 Builder 一致：true
  Builder       与 Buffer 一致：true
结果相同、代价不同：循环里用 + 每次都要复制已有内容，整体是 O(n²)。
```

### 基准测试结果

```bash
go test -run '^$' -bench . -benchmem ./internal/chapter/go08_strings/
```

在本机（Windows / amd64，Intel i7-14700F，Go 1.27）跑三次的典型结果：

```text
BenchmarkConcatPlus-28       	  183208	      5640 ns/op	   21472 B/op	     199 allocs/op
BenchmarkConcatJoin-28       	 1804053	       680.8 ns/op	     208 B/op	       1 allocs/op
BenchmarkConcatBuilder-28    	 2935405	       411.2 ns/op	     208 B/op	       1 allocs/op
BenchmarkConcatBuffer-28     	 1923085	       621.9 ns/op	     656 B/op	       4 allocs/op
```

同一份输入（200 个片段），四者的差距是：

| 写法 | 耗时 | 分配次数 | 分配字节 | 适用场景 |
| --- | --- | --- | --- | --- |
| `+` 累加 | 约 5.6 µs | 199 | 21 KB | 编译期已知的少量拼接 |
| `strings.Join` | 约 680 ns | 1 | 208 B | 已有切片，一次拼成一条 |
| `strings.Builder` | 约 411 ns | 1 | 208 B | 循环里增量构建 |
| `bytes.Buffer` | 约 622 ns | 4 | 656 B | 需要 `io.Writer` / 读回字节 |

这些数字会随机器、Go 版本和输入规模变化（`-28` 是并行度，等于 GOMAXPROCS），不要在正文里把它当成绝对结论；有争议时，用自己的机器跑一遍基准测试。

### 选择建议

1. 拼接片段数量在编译期就确定、且只有两三段时，直接用 `+` 或字面量，最直观。
2. 已经有一个 `[]string`，用 `strings.Join(parts, sep)`，一行搞定且只分配一次。
3. 循环里「边长边拼」，用 `strings.Builder`；能预估长度就先 `Grow`，省掉中途扩容。
4. 结果要被 `io.Writer` 消费、或者需要按字节读写，用 `bytes.Buffer`（`WriteString` / `WriteByte` 都支持，注意最后的 `String()` 会多拷贝一次）。
5. 只需要写 HTTP 响应、日志行这种一次性输出，`fmt.Fprintf` 直接写到 `ResponseWriter` / `Writer` 更省事。

### 边界与坑

- `strings.Builder` **不能被复制**：复制会导致后续写入丢失或 panic，`go vet` 的 `copylocks` 检查会报 `matches lock value` 之类的告警。作为结构体字段时要传指针。
- `Builder.Reset()` 会丢弃已写内容但保留底层缓冲，适合复用；`Grow` 之后不要假设容量精确。
- `strings.Join` 的第二个参数是分隔符，写 `Join(parts, "")` 表示直接相连，不要漏掉这个参数。

---

## 8.9 []byte 与 string 互转

结论：`[]byte(s)` 和 `string(b)` **都是拷贝**，不是同一个缓冲区换了类型看。转换次数多的路径上，要选一种表示走到底。

```go
s := "hello 世界"
b := []byte(s) // 拷贝：s 的字节被复制进新的切片
b[0] = 'H'     // 只影响 b，s 不受影响
fmt.Println(s, string(b)) // hello 世界 Hello 世界
```

### 实测输出

```text
s = "hello 世界"（12 字节）
[]byte(s) = 68 65 6C 6C 6F 20 E4 B8 96 E7 95 8C
修改 b[0] 之后：b = "Hello 世界"，s = "hello 世界"
s 没变，说明 []byte(s) 是一次深拷贝，两者不是同一块缓冲区。
string(b) 转回来 = "Hello 世界"（同样是一次拷贝）
bytes.Contains(b, []byte("世界")) = true（只读判断可以全程走 []byte）
Builder.Write([]byte) 直接追加字节："Hello 世界hello 世界"
```

### 为什么必须拷贝

字符串是只读的，`[]byte` 是可写的。如果 `[]byte(s)` 零拷贝地共享底层数组，那么改 `b[0]` 就会「修改」一个常量字符串，或者让一个字符串的字面量被意外篡改；反过来 `string(b)` 若共享，之后改 `b` 就会让已经当成不可变值用出去的字符串变内容。Go 选择用拷贝换正确性。

### 什么时候不必转换

| 需求 | 不必转成 string 的写法 |
| --- | --- |
| 判断子串是否存在 | `bytes.Contains(b, []byte(sub))` |
| 比较两段字节是否相等 | `bytes.Equal(a, b)` |
| 把内容写进 `io.Writer` | `w.Write(b)` |
| 拼接字节 | `strings.Builder.Write` 或 `bytes.Buffer.Write` |
| 按行切分字节流 | `bytes.Split` / `bufio.Scanner` |

反过来也一样：只读场景整条链路都用 `string`，别为了「可能要用」提前转成 `[]byte`。

### 边界与坑

- 别把「转换是拷贝」当成可以忽略的成本：大文件读进来反复 `string(b)`，就会反复复制整份数据。第 24 章会讲怎么用基准测试定位这类开销。
- `unsafe.String` / `unsafe.SliceData` 能在特定条件下做零拷贝转换，但要求调用方保证底层数据在字符串存活期间不被修改，属于第 31 章的内容；普通业务代码不要引入。
- `strings.Builder.String()` 是个例外：它内部用 `unsafe.String` 直接复用已写好的缓冲，不拷贝；而 `bytes.Buffer.String()` 的实现是 `string(b.buf[b.off:])`，会拷贝一次。所以「拼完就取结果」的场景，Builder 更划算。

---

## 4 个真实报错怎么读

### 报错 1：不可修改字符串元素

```go
s := "Go 学习"
s[0] = 'g'
```

编译输出：

```text
main.go:7:2: cannot assign to s[0] (neither addressable nor a map index expression)
```

**怎么读**：编译器直说了 `s[0]` 既不可寻址、也不是 map 索引，所以不能作为赋值目标。字符串的字节是只读的，索引表达式不是可赋值的位置（不是「值不可变」这么含糊的原因，而是语言层面就不允许）。

**怎么改**：先转成可写表示，改完再转回来。

```go
b := []byte(s)
b[0] = 'g'
s = string(b) // "go 学习"
```

### 报错 2：int 直接转 string

```go
score := 95
msg := "成绩：" + string(score)
fmt.Println(msg)
```

`go vet` 输出：

```text
main.go:7:23: conversion from int to string yields a string of one rune, not a string of digits
```

而这段代码**能编译通过**，跑起来打印的是：

```text
成绩：_
```

**怎么读**：`string(整数)` 的语义是「把这个码点编码成字符」，95 对应的字符是 `_`。编译器不管这个业务语义，所以不报错，`go vet` 会拦；如果直接 `go run` 跳过 vet，就会得到莫名其妙的输出。

**怎么改**：用 `strconv.Itoa(score)` 或 `fmt.Sprintf("成绩：%d", score)`。

### 报错 3：按字节切片越界

```go
s := "学习" // 6 字节
fmt.Println(s[:7])
```

运行输出：

```text
panic: runtime error: slice bounds out of range [:7] with length 6

goroutine 1 [running]:
main.main()
        .../main.go:7 +0x9
exit status 2
```

**怎么读**：字符串的长度是**字节长度**，6 字节只能切到 6。「想要 7 个字符」和「切 7 个字节」在这里被混为一谈了。

**怎么改**：先确定单位。要 7 个字节就检查 `len(s) >= 7` 再切，并且用 `TruncateBytes` 保证不切坏字符；要 7 个字符就用 `TruncateRunes`。

```go
s := "学习"
fmt.Println(TruncateRunes(s, 7)) // 学习，字符不够就返回原串
fmt.Println(TruncateBytes(s, 4)) // 学，切点回退到字符边界
```

### 报错 4：strconv 的解析错误

```go
n, err := strconv.Atoi("12a")
fmt.Println(n, err)
```

输出：

```text
0 strconv.Atoi: parsing "12a": invalid syntax
```

**怎么读**：错误信息里包含三层信息——函数名（`strconv.Atoi`）、动作（`parsing "12a"`）、原因（`invalid syntax`）。注意失败时 `n` 是 0，不是「解析到一半的结果 12」，所以**必须**先看 `err` 再用 `n`。

**怎么改**：立刻判断错误，并按语法错误 / 范围错误分别处理，写法见 8.7 的 `errors.Is(err, strconv.ErrRange)` 示例。

### 还有一个不报错的坑

本章 8.4 的 `s[4:6]` 既不会编译失败，也不会 panic，它只是产出非法 UTF-8。这类 bug 最难发现，因为英文用例全绿。检查手段是把 `utf8.ValidString` 加在系统边界上（读配置、收请求、读文件之后），而不是散落各处。

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| `len("学习")` 是 6 而不是 2 | `len` 数的是字节 | 要字符数用 `utf8.RuneCountInString` |
| `s[0]` 不是第一个汉字 | 下标是字节，不是字符 | 按字符遍历用 `range`，或转 `[]rune` |
| `s[0:1]` 打印成乱码 | 切点落在字符中间，得到非法 UTF-8 | 用 `TruncateBytes` / `TruncateRunes` 回退到边界 |
| `s[0] = 'g'` 无法编译 | 字符串只读，索引不可赋值 | 转成 `[]byte` 改完再 `string(b)` |
| `string(65)` 打印出 `A` | 这是码点转字符，不是数字转字符串 | 用 `strconv.Itoa(65)` |
| `strings.Index(...) > 0` 判断前缀失败 | 目标在开头时返回 0 | 用 `Contains` / `HasPrefix` / `CutPrefix` |
| 循环里 `+=` 拼字符串很慢 | 每次都要拷贝已有内容，O(n²) | 用 `strings.Builder` + `Grow` |
| 下标在中文串里错位 | `strings` 的下标都是字节下标 | 需要字符序号就先转 `[]rune`，或自己计数 |
| 收到的文本被判为非法 UTF-8 | 上游截断产生坏字节 | 边界处 `utf8.ValidString` 校验，清洗用 `strings.ToValidUTF8` |
| `for i, r := range s` 里 `i` 不连续 | `i` 是字节偏移 | 要字符序号就自己计数，或转 `[]rune` |

---

## 练习

### 第 1 题

不运行代码，先写出 `len("你好，世界")` 和 `utf8.RuneCountInString("你好，世界")` 的值，然后运行验证，并解释两个数字的差是怎么来的。

::: details 第 1 题参考答案

```go
s := "你好，世界"
fmt.Println(len(s))                        // 15
fmt.Println(utf8.RuneCountInString(s))     // 5
```

**为什么**：串里有 5 个字符，每个都是 3 字节的汉字或全角标点（`你`、`好`、`，`、`世`、`界`），所以是 15 字节、5 个字符。两个数字的差来自「字符用几个字节编码」这件事——ASCII 字符只占 1 字节，所以纯英文串里两个数字相等。

**工程含义**：数据库字段长度、接口限长、UI 显示宽度这三个需求单位不同，先确认要按字节还是按字符，再选 `len` 或 `RuneCountInString`。

:::

### 第 2 题

只保留 `s` 的前 `n` 个字符，并且如果发生了截断就在末尾补上 `…`；要求结果始终是合法 UTF-8，且补省略号后不能超过 `n` 个字符。请实现 `Ellipsis(s string, n int) string`，并针对 `n` 等于 0、1、恰好等于字符数、字符数不足这几种情况写测试。

::: details 第 2 题参考答案

```go
// Ellipsis 最多保留 s 的前 n 个字符；发生截断时用省略号占掉最后一个字符位。
func Ellipsis(s string, n int) string {
    if n <= 0 {
        return ""
    }
    if utf8.RuneCountInString(s) <= n {
        return s
    }
    runes := []rune(s)
    return string(runes[:n-1]) + "…"
}
```

测试要点：

```go
cases := []struct {
    in   string
    n    int
    want string
}{
    {"学习笔记", 0, ""},
    {"学习笔记", 1, "…"},          // 只够放省略号
    {"学习笔记", 2, "学…"},
    {"学习笔记", 4, "学习笔记"},      // 不截断，原样返回
    {"学习笔记", 9, "学习笔记"},      // 超出长度也不截断
}
```

**为什么这样写更好**：先判断「需不需要截断」再动手，避免为了短字符串白白分配 `[]rune`；`n-1` 让省略号占用一个字符位，保证结果长度不超过 `n`；分界情况（`n == 0`、`n == 1`）单独处理，避免出现负数切片下标。

**注意**：这里 `…` 是 3 个字节、1 个字符，若下游按字节限长，还得再走一次 `TruncateBytes`。

:::

### 第 3 题

下面的函数想判断 `s` 是否以 `prefix` 开头，但某些输入下会漏判。指出问题并给出两种修复方式。

```go
func startsWith(s, prefix string) bool {
    return strings.Index(s, prefix) > 0
}
```

::: details 第 3 题参考答案

**问题**：`Index` 在「目标出现在开头」时返回 0，而 `> 0` 把 0 判成了假，所以 `startsWith("Go 学习", "Go")` 返回 `false`。用「下标 > 0」当存在性判断是经典的 off-by-one。

```go
// 修复一：语义最清楚
func startsWith(s, prefix string) bool {
    return strings.HasPrefix(s, prefix)
}

// 修复二：坚持用下标判断，就必须带上等号
func startsWith(s, prefix string) bool {
    return strings.Index(s, prefix) >= 0
}
```

**为什么 `HasPrefix` 更好**：它把「判断前缀」这个意图写进了函数名，不依赖调用方记住 `-1` 和 `0` 的区别；同理，判断存在性用 `Contains`，不要用 `Index(...) >= 0`。

:::

### 第 4 题

把下面这段「一边遍历一边拼字符串」的代码改成用 `strings.Builder`，并写出基准测试，用 `go test -bench . -benchmem` 对比改前改后的分配次数。

```go
func render(lines []string) string {
    out := ""
    for _, line := range lines {
        out += line + "\n"
    }
    return out
}
```

::: details 第 4 题参考答案

先把原来的实现改名成 `renderPlus` 用作对照，再写 Builder 版本：

```go
func render(lines []string) string {
    total := 0
    for _, line := range lines {
        total += len(line) + 1 // 每行末尾还有一个换行
    }
    var sb strings.Builder
    sb.Grow(total)
    for _, line := range lines {
        sb.WriteString(line)
        sb.WriteByte('\n')
    }
    return sb.String()
}

func BenchmarkRenderPlus(b *testing.B) {
    lines := make([]string, 200)
    for i := range lines {
        lines[i] = "line"
    }
    b.ReportAllocs()
    for b.Loop() {
        sink = renderPlus(lines)
    }
}
```

预期结果：`+` 版本每次大约 1 次分配 × 行数，`Builder` 版本是 1 次分配（`Grow` 之后不再扩容）。本章示例包的四种拼接写法就是这个对照：`BenchmarkConcatPlus` 是 199 allocs/op，`BenchmarkConcatBuilder` 是 1 allocs/op。

**为什么这样写更好**：`Grow` 把「需要多大」算在前面，既避免多次扩容，也避免最终 `String()` 再复制一次；`WriteByte('\n')` 比 `WriteString("\n")` 少一次类型转换。

**注意**：`sink` 这样的包级变量不是洁癖，而是防止编译器把「结果没人用」的调用优化掉，导致基准数字失真。

:::

### 第 5 题

下面四个字符串哪些相等？逐一验证并解释。

```go
a := "中"
b := "\u4e2d"
c := string([]byte{0xE4, 0xB8, 0xAD})
d := string(rune(20013))
```

::: details 第 5 题参考答案

```go
fmt.Println(a == b, a == c, a == d) // true true true
fmt.Println(len(a), len(c), len(d)) // 3 3 3
```

四者都是同一个字符串：`"中"` 的码点是 U+4E2D（十进制 20013），UTF-8 编码是 `E4 B8 AD` 三个字节。

- `b` 用的是 `\u` 转义，写的是码点；
- `c` 直接写出字节，等价于 `[]rune` 的 UTF-8 编码结果；
- `d` 是 `string(码点)` 的形式。写成 `string(20013)` 结果一样，但 `go vet` 会报 `conversion from untyped int to string yields a string of one rune, not a string of digits`——这也是本章报错 2 的来源。注意它和「把数字 20013 转成字符串 `"20013"`」完全不同，后者要用 `strconv.Itoa`。

**为什么值得记**：`string([]byte{...})` 这种写法常出现在协议解析、测试用例构造非法输入时；而 `string(数字)` 是本章报错 2 的根源。

:::

### 第 6 题

不用 `[]rune`、不用 `utf8.RuneCountInString`，只用一个 `for range` 循环实现 `countRunes(s string) int`；再用基准测试比较它和 `utf8.RuneCountInString` 的分配情况，说明为什么两者都是 0 allocs/op。

::: details 第 6 题参考答案

```go
func countRunes(s string) int {
    n := 0
    for range s {
        n++
    }
    return n
}
```

基准对比：

```bash
# 自己补上 BenchmarkCountRunes 和 BenchmarkRuneCountInString 后运行
go test -bench 'CountRunes' -benchmem .
```

两者都是 0 allocs/op。原因是它们做的事情一样：`range` 内部就等价于反复调用 `utf8.DecodeRuneInString`，只推进字节下标、不做任何分配；`RuneCountInString` 也是同一个循环，只是写在标准库里。

**结论**：真正会分配的是 `[]rune(s)`——它要把整串字符一次性物化成切片。所以「统计字符数」「只取前几个字符」这类需求用 `range` 就够，「需要按下标反复随机访问字符」才值得转 `[]rune`。示例包里的 `TruncateRunes` 用的是同一个技巧。

:::

---

## 小结

- **string 是只读的字节序列，byte 与 rune 是两层视角**：`len` 数字节、`s[i]` 是 `byte`、内容不可修改；`rune` 才是码点，一个字符在 UTF-8 里占 1–4 字节（汉字 3 字节、Emoji 4 字节）。
- **`range` 是唯一免分配的按字符遍历方式**：`i` 是字节偏移、`r` 是 rune；非法字节会被替换成 `U+FFFD` 并前进 1 字节。
- **下标切片切的是字节**：切点必须落在字符首字节，否则得到非法 UTF-8；按字节限长用 `TruncateBytes`（`utf8.RuneStart` 回退），按字符限长用 `TruncateRunes`（`for i := range s`）。
- **`unicode/utf8` 负责检查与计量**：`ValidString` 验合法性、`RuneCountInString` 数字符、`RuneLen` 查宽度、`DecodeRuneInString` 手动解码，都不 panic。
- **`strings` 的下标全是字节下标**：判存在用 `Contains`、判前缀用 `HasPrefix`、切一段用 `Cut`、拼切片用 `Join`；Go 1.24 起还可以用 `SplitSeq` 一类迭代器省内存。
- **数字与字符串互转用 `strconv`**：`Atoi`/`Itoa` 是 10 进制快捷方式，其他进制用 `ParseInt`/`FormatInt`；`string(95)` 得到的是字符 `_`，不是 `"95"`。
- **拼接选型决定性能**：循环里 `+=` 是 O(n²)（本章基准里 199 次分配、约 5.6 µs），`strings.Join` 与 `strings.Builder` 都是 1 次分配（约 680 ns / 411 ns），`bytes.Buffer` 适合需要 `io.Writer` 的场景。
- **`[]byte` 与 `string` 互转是拷贝**：只读判断可以全程走 `bytes`，写缓冲区用 `strings.Builder`；零拷贝方案（`unsafe.String`）属于第 31 章。

处理文本的规则可以浓缩成一句话：**先想清楚这一刻在处理字节还是字符，再选函数**。下一章进入结构体与方法，看看 Go 怎么把数据和操作数据的行为绑在一起，以及值接收者和指针接收者在方法集上的差别。
