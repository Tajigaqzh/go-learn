# 第 18 章 · 序列化与配置

上一章处理的是「字节怎么进出程序」，这一章要回答两个更贴近业务的问题：内存里的结构体怎么变成可以传输、可以落盘的格式；配置又怎么从默认值、配置文件和部署环境汇聚成一份最终结果。

Go 的答案是「标准库优先」：`encoding/json` 负责最常用的 JSON，`encoding/xml`、`encoding/csv`、`encoding/gob` 各管一类格式，而 YAML、TOML 这类人类友好的配置格式标准库**没有**，需要第三方库。把结构体和 JSON 之间的映射规则吃透（标签、`omitempty`、指针与嵌套、数字精度），能避免一大批「字段莫名消失」「大整数莫名变值」的问题；把配置的分层规则写清楚（默认值 → 文件 → 环境变量），能让同一份二进制在本地和线上都跑得对。

本章按「JSON 细节 → 其它格式 → 配置分层」的顺序展开，最后落到敏感信息的脱敏和格式选型。读完你应该能回答：`omitempty` 到底把什么当作「空」，为什么雪花 ID 用 `any` 接会变值，以及为什么日志里不该出现密码。

本章配套代码在 `internal/chapter/go18_serde_config/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

---

## 18.1 json.Marshal / Unmarshal 与标签规则

结论：字段与 JSON key 的映射由 `json` 标签决定，四条规则覆盖绝大多数场景——`omitempty` 省略零值、`-` 永远跳过、`,string` 把数字写成字符串、未导出字段一律不参与。

```go
type User struct {
    ID       int    `json:"id"`
    Name     string `json:"name"`
    Email    string `json:"email,omitempty"` // 零值时整个字段不输出
    Password string `json:"-"`               // 永远不参与序列化
    Age      int    `json:"age,string"`      // 以 JSON 字符串形式编码数字
    internal string                          // 未导出字段：不参与
}
```

### 实测输出

```text
Marshal   = {"id":1,"name":"Ada","age":"36"}，err = <nil>
Unmarshal = {ID:1 Name:Ada Email: Password: Age:36 internal:}
Password 没有被序列化：true
age 收到非数字字符串：err = json: cannot unmarshal number abc into Go struct field User.age of type int
```

注意 `age` 在 JSON 里是**字符串** `"36"`，这是 `,string` 标签的效果；反序列化时如果拿到 `"abc"`，`Unmarshal` 会返回错误而不是 panic——错误信息里有字段名 `User.age`，定位很快。

### 三个实践要点

1. **不要给未导出字段打 `json` 标签**：它既不生效，`go vet` 还会提示「struct field has json tag but is not exported」。
2. **`omitempty` 不是「有默认值就不输出」**，它对 `0`、`false`、空串、nil 指针、空切片都成立，写 API 时要想清楚「0 和缺失」是不是一回事。
3. **接口字段用 `any` 接会得到 `float64`**，大整数会丢精度（见 18.3）。

### 边界与坑

- 反序列化目标必须是**指针**，传值会返回 `json: Unmarshal(non-pointer ...)`。
- JSON 里多余字段默认会被忽略；想严格校验用 `Decoder.DisallowUnknownFields`（见 18.4）。
- 标签写错（比如少一对引号）不会报错，只会静默失效——`go vet` 能查出这类问题。

---

## 18.2 嵌套、指针与 omitempty 的真实语义

结论：`omitempty` 把「零值」当空，而指针为 `nil`、切片长度为 0 都算空；**没有** `omitempty` 时，nil 指针和 nil 切片会输出 `null`，而不是被省略。

```go
type Account struct {
    Name    string   `json:"name"`
    Address *Address `json:"address,omitempty"`
    Tags    []string `json:"tags,omitempty"`
    Empty   []string `json:"empty"`            // 没有 omitempty
    Score   *int     `json:"score,omitempty"`  // 指向 0 的指针不算空
}
```

### 实测输出

```text
全零值          → {"name":"Ada","empty":null}（err = <nil>）
带地址和空切片      → {"name":"Ada","address":{"city":"上海"},"empty":[]}（err = <nil>）
指针指向零值       → {"name":"Ada","empty":null,"score":0}（err = <nil>）
嵌套 nil 指针 → {"code":200,"data":null}
```

三行输出分别印证了三件事：nil 切片在没有 `omitempty` 时是 `null`，空切片在有 `omitempty` 时被省略（`Tags` 消失了）、但换成非 nil 的空切片又会被序列化成 `[]`；指针指向 0 时**不算空**，字段照常输出。

### 「区分零值和缺失」的标准做法

```go
type Patch struct {
    Name  *string `json:"name"`  // nil = 不改；指向 "" = 改成空串
    Age   *int    `json:"age"`
}
```

需要表达「没传这个字段」时用指针；需要表达「传了空值」时用指针指向零值。这也是 PATCH 接口的常见约定。

### 边界与坑

- 结构体类型的零值**永远不会**被 `omitempty` 省略（Go 对结构体没有「空」的定义），要省略就用指针。
- `omitempty` 对 `time.Time` 也无效：它的零值不是「空」，会被序列化成 `"0001-01-01T00:00:00Z"`，需要指针或自定义 `MarshalJSON`。
- `map` 为 nil 时同样输出 `null`，空 map 输出 `{}`。

---

## 18.3 数字精度：json.Number 与 UseNumber

结论：用 `any`（或 `map[string]any`）接 JSON 数字，会统一变成 `float64`，超过 2^53 的整数**安静地丢精度**；要精确处理就解到具体类型，或者用 `Decoder.UseNumber()`。

```go
const payload = `{"id": 9007199254740993}` // 2^53 + 1
```

### 实测输出

```text
原始 JSON：{"id": 9007199254740993}
解到 any：9.007199254740992e+15（float64），转回 int64 = 9007199254740992（精度已经丢了）
UseNumber：9007199254740993（json.Number），Int64() = 9007199254740993，err = <nil>
直接解到 int64 字段：9007199254740993（精确）
```

三条路线的差别很清楚：

| 接法 | 结果 | 适用场景 |
| --- | --- | --- |
| `map[string]any` / `any` | `float64`，超 2^53 丢精度 | 只做展示、数值范围小的场景 |
| 具体字段类型（`int64` / `string`） | 精确 | 业务结构体，首选 |
| `Decoder.UseNumber()` | `json.Number`（原始文本） | 结构不固定、需要延迟解析 |

```go
decoder := json.NewDecoder(r)
decoder.UseNumber()
var body map[string]any
_ = decoder.Decode(&body)
id, err := body["id"].(json.Number).Int64() // 需要时再转，不会先丢精度
```

### 边界与坑

- 浮点数本身就可能有精度问题（`0.1 + 0.2`），金额建议用最小单位整数（分）或 `string`。
- `json.Number` 只是「原始文本」的包装，`Int64()` / `Float64()` 解析失败会返回错误，别忽略。
- `json.Marshal` 序列化 `float64` 时用的是最短可还原表示，反过来不一定能还原成同一个浮点值——跨语言传输时尤其要注意。

---

## 18.4 Decoder：流式解析与严格模式

结论：`json.Unmarshal` 适合一次性拿到完整字节的场景；数据来自文件、网络流、或者「一行一个 JSON」时用 `json.Decoder`，它可以连续解析多个值，还能开严格模式。

```go
decoder := json.NewDecoder(strings.NewReader(ndjson))
for {
    var event Event
    err := decoder.Decode(&event)
    if errors.Is(err, io.EOF) {
        break
    }
    // 处理 event
}
```

### 实测输出

```text
读到事件 1：first
读到事件 2：second
读到事件 3：third
DisallowUnknownFields → err = json: unknown field "unknown"
```

### 什么时候用哪个

| 场景 | 用法 |
| --- | --- |
| HTTP 请求体、小文件 | `json.Unmarshal(io.ReadAll(r))` 或 `Decoder` 均可 |
| 大文件、日志流、NDJSON | `json.Decoder` + 循环 `Decode`，内存占用恒定 |
| 配置文件要严格校验 | `Decoder.DisallowUnknownFields()`，拼写错误立刻报错 |
| 一次请求里有多个 JSON 值 | 连续 `Decode`（`Decoder` 自己维护读取位置） |

### 边界与坑

- 循环里遇到 `io.EOF` 是**正常结束**，要用 `errors.Is(err, io.EOF)` 判断，不能当成错误上报。
- `DisallowUnknownFields` 一旦开启，多一个字段就整条失败——适合配置文件，不适合「客户端版本比服务端新」的接口。
- `Decoder` 会缓冲，`Decode` 之后如果需要继续读原始流（比如后面还有别的协议数据），要用 `decoder.Buffered()` 拿回剩余内容。

---

## 18.5 自定义 MarshalJSON / UnmarshalJSON

结论：想让字段以自定义形式出现在 JSON 里（枚举、时间格式、脱敏字段），就实现 `json.Marshaler` 和 `json.Unmarshaler`。**`MarshalJSON` 用值接收者，`UnmarshalJSON` 用指针接收者**。

```go
type Status int

// MarshalJSON 用值接收者：值和指针都走这里。
func (s Status) MarshalJSON() ([]byte, error) {
    name, ok := statusNames[s]
    if !ok {
        return nil, fmt.Errorf("unknown status %d", int(s))
    }
    return json.Marshal(name)
}

// UnmarshalJSON 用指针接收者：要写回调用方的值。
func (s *Status) UnmarshalJSON(data []byte) error { /* 解析字符串 → 枚举 */ }
```

### 实测输出

```text
枚举序列化 → {"name":"backup","status":"paused"}，err = <nil>
枚举反序列化 → Status=paused（1）
未知枚举值 → err = unknown status "unknown"
指针接收者的 BadStatus（值字段）→ {"status":1}
指针接收者的 BadStatus（取地址后）→ {"status":"bad"}
```

后两行是本章最值得记住的坑：把 `MarshalJSON` 定义在**指针接收者**上时，结构体里的**值字段**不会被特殊处理，输出的是原始数字；只有把外层结构体取地址（或者字段本身是指针）才会调用自定义方法。同一份数据因为「值 vs 指针」产生了两种完全不同的 JSON，这在线上极难排查。

### 自定义之后记得校验

自定义反序列化意味着**你**负责校验：未知枚举值应该返回错误（示例里的 `unknown status "unknown"`），而不是默默赋一个零值。同理，自定义时间格式时要在 `UnmarshalJSON` 里校验格式，让错误在这里暴露。

### 边界与坑

- `MarshalJSON` 里**不要**再调用 `json.Marshal(self)`，会无限递归直到栈溢出（见报错 2）；要输出别名就用 `type alias Status` 转换后序列化。
- `MarshalJSON` 返回的字节必须是合法 JSON，否则外层 `json.Marshal` 会返回 `json: error calling MarshalJSON for type ...`。
- 实现了 `MarshalJSON` 的类型，`omitempty` 依然按「零值」判断，不会按自定义输出判断。

---

## 18.6 encoding/xml、encoding/csv、encoding/gob

结论：标准库自带多个格式，选择依据是「和谁交换数据」：XML 对老系统、CSV 对表格、gob 只在 Go 进程之间用。

```go
type Book struct {
    XMLName xml.Name `xml:"book"`
    Title   string   `xml:"title"`
    Author  string   `xml:"author"`
    Price   float64  `xml:"price"`
}
```

### 实测输出

```text
xml.MarshalIndent：
<book>
  <title>Go 语言</title>
  <author>Ada</author>
  <price>42.5</price>
</book>
csv.Reader 读回 3 行：[[id name] [1 Ada] [2 Bob]]，err = <nil>
字段数不一致 → err = record on line 2: wrong number of fields
gob 编码后 128 字节（同一份数据 JSON 是 83 字节）
```

注意最后一行的反直觉结论：**gob 不一定比 JSON 小**。gob 会写入类型信息，短数据下开销反而更大；它的优势是编解码快、能忠实保留 Go 的类型（含未导出字段的零值语义），代价是**只能 Go ↔ Go**，且跨版本要小心类型变更。

### 各自的适用面

| 编码 | 适用场景 | 注意 |
| --- | --- | --- |
| `encoding/json` | 对外接口、日志、配置 | 数字精度、`omitempty` 语义 |
| `encoding/xml` | 对接老系统、SOAP | 标签语义和 JSON 不同，`attr` 表示属性 |
| `encoding/csv` | 表格导入导出 | 字段数不一致会报错，注意引号与换行转义 |
| `encoding/gob` | Go 服务之间传对象、缓存 | 不跨语言，类型定义变更要兼容 |

### 边界与坑

- CSV 的 `Reader` 默认要求每行字段数一致，`FieldsPerRecord = -1` 可以关掉这个校验；`Writer.Flush()` 之后必须检查 `Writer.Error()`。
- XML 的 `XMLName` 字段必须是导出字段，且要放在结构体里才会生效。
- gob 需要 `gob.Register` 注册接口的具体类型，否则接口字段编码会失败（`gob: type not registered`）。

---

## 18.7 YAML / TOML：标准库没有，手写一个简单解析器

结论：**标准库不含 YAML 和 TOML**。生产项目用 `gopkg.in/yaml.v3` 或 `github.com/BurntSushi/toml`；本章不引第三方依赖，改成手写一个「YAML 子集」解析器，把「解析配置要处理什么」摊开看。

```yaml
# 应用配置
app:
  name: "go-learn"   # 行尾注释
  debug: true
  port: 8080
```

### 实测输出

```text
解析结果（key 已排序，方便对照）：
app:
  debug: true
  name: go-learn
  port: 8080
database:
  host: 127.0.0.1
  max_open: 20
缺少冒号 → err = 第 2 行缺少冒号："name go-learn"
取值：app.name = go-learn（string），app.port = 8080（int64）
```

### 解析器要处理的四件事

1. **注释与空行**：`#` 之后的内容要丢掉，但引号里的 `#` 不算注释；
2. **缩进层级**：用「当前缩进 vs 栈顶缩进」决定是进入下一层还是回到上层；
3. **类型推断**：`true` / `8080` / `0.5` / `"带引号的字符串"` 分别对应 bool、int64、float64、string；
4. **错误定位**：解析失败要给出**行号**（`第 2 行缺少冒号`），否则配置文件一长就没法排查。

### 真正的 YAML 库还需要什么

列表、多行字符串（`|` / `>`）、锚点与别名、类型标签、流式风格、跨文档分隔符（`---`）……这些都不该自己实现。手写解析器的价值只在于理解「解析配置的边界条件有哪些」。

### 边界与坑

- 引号里的 `#` 不能当注释；`password: "a#b"` 里的 `#` 属于值。
- YAML 的 `no` / `off` 在 1.1 语义下是布尔值，`"yes"` 要不要加引号取决于库和版本——**密码、版本号这类值一律加引号**。
- TOML 的语义比 YAML 更明确（没有隐式类型转换），配置格式的选型里 TOML 通常更安全。

---

## 18.8 环境变量与默认值

结论：环境变量一律用 `os.LookupEnv` 读取，它同时告诉你「值是什么」和「有没有设置」；`os.Getenv` 分不清「空值」和「未设置」。

```go
port, ok := os.LookupEnv("APP_PORT")
if !ok {
    port = "8080" // 只有「没设置」时才用默认值
}
value, err := ParseInt64("APP_PORT", port)
```

### 实测输出

```text
LookupEnv(APP_PORT) = "8080"，ok = true
LookupEnv(APP_MISSING) → ok = false
Getenv(APP_MISSING) = ""（只调 Getenv 分不清「空值」和「没设置」）
解析成整数：8080，err = <nil>
解析非法值 → err = 解析 APP_PORT="abc": strconv.ParseInt: parsing "abc": invalid syntax
```

### 解析辅助函数

```go
// ParseInt64 解析十进制整数字符串，失败时返回带上下文的错误。
func ParseInt64(field, text string) (int64, error) {
    value, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
    if err != nil {
        return 0, fmt.Errorf("解析 %s=%q: %w", field, text, err)
    }
    return value, nil
}
```

错误信息里带上**变量名和原值**，比 `strconv.ParseInt: parsing "abc": invalid syntax` 有用得多——线上排查时你会庆幸自己写了 `APP_PORT="abc"`。

### 边界与坑

- 解析失败要直接报错返回，不要 `v, _ := strconv.Atoi(...)` 用零值继续跑。
- 测试里改环境变量请用 `t.Setenv`（第 19 章），它会在用例结束后自动还原；直接 `os.Setenv` 会污染同进程的其它测试。
- 敏感值（密码、Token）只从环境变量或密钥服务读取，配置文件里不要留明文。

---

## 18.9 配置分层与优先级

结论：配置按「默认值 → 配置文件 → 环境变量」三层合并，后面的层只覆盖自己提供过的字段。这样本地开发只写默认值，线上用环境变量覆盖部署相关的项。

```go
// LoadConfig 按「默认值 → 配置文件 → 环境变量」的顺序合并配置。
func LoadConfig(defaults Config, fileJSON string, lookup EnvLookup) (Config, error) {
    cfg := defaults
    if strings.TrimSpace(fileJSON) != "" {
        if err := json.Unmarshal([]byte(fileJSON), &cfg); err != nil {
            return Config{}, fmt.Errorf("解析配置文件: %w", err)
        }
    }
    if lookup == nil {
        return cfg, nil
    }
    if value, ok := lookup(EnvPrefix + "PORT"); ok {
        port, err := ParseInt64("APP_PORT", value)
        if err != nil {
            return Config{}, err
        }
        cfg.Port = int(port)
    }
    // 其余字段同理……
    return cfg, nil
}
```

### 实测输出

```text
① 默认值        Config{Host:"localhost" Port:8080 LogLevel:"info" Password:(未设置)}
② 叠加配置文件  Config{Host:"0.0.0.0" Port:9000 LogLevel:"debug" Password:f**********}，err = <nil>
③ 叠加环境变量  Config{Host:"0.0.0.0" Port:7000 LogLevel:"debug" Password:e*********}，err = <nil>
文件只写一个字段，其余保留默认值：Config{Host:"localhost" Port:9000 LogLevel:"info" Password:(未设置)}，err = <nil>
配置文件类型写错 → err = 解析配置文件: json: cannot unmarshal string into Go struct field Config.port of type int
环境变量类型写错 → err = 解析 APP_PORT="不是数字": strconv.ParseInt: parsing "不是数字": invalid syntax
```

三层的效果都在输出里：文件覆盖了 Host/Port/LogLevel，环境变量又把 Port 从 9000 改回 7000；配置文件里没写的 `Host` 在第二行仍是 `0.0.0.0`（被文件覆盖），而只写 `port` 的那次 `Host` 保持默认 `localhost`。

### 为什么不用「全部从文件读」或「全部从环境变量读」

- 只读文件：容器化部署时改一个开关就要重新挂载配置或重启；
- 只读环境变量：项目一多、层级一深，环境变量会变成几十个难以维护的名字；
- 分层合并：结构化的部分留给配置文件，部署相关的部分交给环境变量，边界清晰。

### 优先级里最容易错的两种情况

1. **把「没设置」当成「空值」**：`os.Getenv` 返回空串时无法区分，于是默认值被空串覆盖——必须用 `LookupEnv`。
2. **每层各自解析类型**：文件用 `json.Unmarshal`、环境变量手动 `Atoi`，两套校验逻辑容易不一致；让两层最终都写回同一个结构体字段，错误信息也保持一致。

### 边界与坑

- 合并前先校验，合并后再校验一次「业务规则」（比如端口范围、日志级别枚举），两层都要检查。
- 环境变量名统一加前缀（`APP_`），避免和其他服务的变量撞名。
- 配置结构体一旦导出给下游，字段改名等同于破坏兼容性；对外用独立的 DTO。

---

## 18.10 敏感信息脱敏与格式选型

结论：让配置类型的 `String()`（或 `MarshalJSON`）默认脱敏，日志只接收脱敏后的副本；否则一次 `log.Printf("%+v", cfg)` 就会把密码写进日志系统。

```go
// String 让 Config 在打印时自动脱敏，避免密码进日志。
func (c Config) String() string {
    return fmt.Sprintf("Config{Host:%q Port:%d LogLevel:%q Password:%s}",
        c.Host, c.Port, c.LogLevel, MaskSecret(c.Password))
}

// MaskSecret 保留首字符，其余用 * 代替；空串返回「(未设置)」。
func MaskSecret(secret string) string {
    if secret == "" {
        return "(未设置)"
    }
    runes := []rune(secret)
    if len(runes) == 1 {
        return "*"
    }
    return string(runes[0]) + strings.Repeat("*", len(runes)-1)
}
```

### 实测输出

```text
打印配置（自动走 String()）：Config{Host:"0.0.0.0" Port:8080 LogLevel:"info" Password:s***********}
Redacted() 之后序列化：{"host":"0.0.0.0","port":8080,"log_level":"info","password":"s***********"}，err = <nil>
直接序列化 Config：{"host":"0.0.0.0","port":8080,"log_level":"info","password":"s3cr3t-value"}，err = <nil>
```

第三行是关键：`Config` 只实现了 `String()`，`encoding/json` **不会**调用它，所以直接 `json.Marshal(cfg)` 照样把密码原样写出去。要么给日志用专门的 `Redacted()` 结构体，要么实现 `MarshalJSON` 做脱敏——但后者会让「真的需要原值」的场景变得别扭，所以通常推荐前者。

### 格式选型一并说清

```text
  对外 HTTP 接口     encoding/json（标准、可读、跨语言）
  本地配置文件       YAML / TOML（人类友好，用第三方库）+ 环境变量覆盖
  Go 进程之间        gob（带类型信息，启动有注册开销，不能跨语言）
  跨语言高性能       protobuf（本章范围外）
```

选型的判断顺序是：**谁要读这份数据** → 对方用什么格式 → 再考虑体积和性能。绝大多数接口用 JSON 就够了，提前上 protobuf 往往得不偿失。

### 边界与坑

- 脱敏要覆盖所有出口：日志、监控标签、错误信息、调试接口，漏一个就等于没做。
- `String()` 用值接收者，这样指针和值打印都能脱敏。
- 排查现场需要「确实拿到了密码」的证据时，用长度或哈希（`sha256` 前几位）代替原值。

---

## 5 个真实报错怎么读

### 报错 1：字段类型对不上

```go
var u User
err := json.Unmarshal([]byte(`{"age":"abc"}`), &u)
```

输出：

```text
json: cannot unmarshal number abc into Go struct field User.age of type int
```

**怎么读**：错误信息给了四样东西——动作（`cannot unmarshal number`）、值（`abc`）、出错字段（`User.age`）、目标类型（`int`）。注意它说的是 `number abc`，因为 `Age` 带了 `,string` 标签，解析器按「字符串形式的数字」去读，读到 `abc` 就失败了。

**怎么改**：要么让输入符合约定，要么把字段类型放宽成 `string` 或 `json.Number` 再手工校验。

### 报错 2：MarshalJSON 里递归调用自己

```go
func (l Loop) MarshalJSON() ([]byte, error) {
    return json.Marshal(l) // 又调回自己
}
```

运行输出：

```text
runtime: goroutine stack exceeds 1000000000-byte limit
runtime: sp=0x... stack=[0x..., 0x...]
fatal error: stack overflow
runtime stack:
runtime.throw(...)
```

**怎么读**：`json.Marshal(l)` 会再次调用 `Loop.MarshalJSON`，无限递归直到栈耗尽。这类错误不会给出「哪里递归了」，只能靠自己排查 `MarshalJSON` 的实现。

**怎么改**：用类型别名跳出方法集：

```go
func (l Loop) MarshalJSON() ([]byte, error) {
    type plain Loop            // 别名没有 MarshalJSON 方法
    return json.Marshal(plain(l))
}
```

### 报错 3：MarshalJSON 返回的不是合法 JSON

```go
func (b Bad) MarshalJSON() ([]byte, error) {
    return []byte("{oops"), nil
}
```

输出：

```text
json: error calling MarshalJSON for type *main.Bad: invalid character 'o' looking for beginning of object key string
```

**怎么读**：错误分两段——外层告诉你「是自定义方法出的问题」（`error calling MarshalJSON for type *main.Bad`），内层是真正的原因（`invalid character 'o' ...`）。自己拼 JSON 字符串最容易出这类问题，所以自定义方法里应该**用 `json.Marshal` 组装**，而不是手写字符串。

### 报错 4：CSV 字段数不一致

```go
records, err := csv.NewReader(strings.NewReader("a,b\n1,2,3\n")).ReadAll()
```

输出：

```text
record on line 2: wrong number of fields
```

**怎么读**：错误给了行号（第 2 行）和原因（字段数不对）。默认情况下 `csv.Reader` 以第一行的字段数为准，多一列少一列都会失败。

**怎么改**：源头修数据；确实允许变长时设 `reader.FieldsPerRecord = -1`，但这意味着所有列数校验都要自己做。

### 报错 5：未知 JSON 字段（严格模式）

```go
decoder := json.NewDecoder(strings.NewReader(`{"id":1,"unknown":true}`))
decoder.DisallowUnknownFields()
err := decoder.Decode(&event)
```

输出：

```text
json: unknown field "unknown"
```

**怎么读**：这是**主动开启**的严格校验，不是 bug——配置里多写了拼错的 key（比如 `prot` 写成 `port`）时，它能第一时间拦住。默认模式下多余字段会被安静忽略，拼错的字段就用默认值跑到底。

**怎么改**：配置文件解析开启它；对外接口不要开，否则客户端多传字段就会 400。

### 还有一个不报错的坑：omitempty 对结构体无效

```go
type Outer struct {
    In   Inner   `json:"in,omitempty"`   // 结构体：永远输出
    Ptr  *Inner  `json:"ptr,omitempty"`  // 指针：nil 时省略
    List []Inner `json:"list,omitempty"` // 切片：空时省略
}

data, _ := json.Marshal(Outer{})
fmt.Println(string(data))
```

输出：

```text
{"in":{"a":0}}
```

**怎么读**：`omitempty` 只会省略「零值」的**基础类型、指针、切片、map、字符串**，结构体没有「空」的定义，所以 `In` 照常输出；而 `Ptr` 和 `List` 都被省略了。如果接口约定里「没有这个对象」就不能出现在 JSON 里，字段必须用指针。

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 字段莫名消失 | `omitempty` 把 0 / false / "" / nil 当空 | 需要区分「零值」和「缺失」时改用指针 |
| 空切片变成 `null` | nil 切片没有 `omitempty` | 加 `omitempty`，或初始化成空切片 |
| 结构体字段没被省略 | `omitempty` 对结构体无效 | 字段改成指针 |
| 大整数 ID 变值 | 用 `any` 接数字 → `float64` | 解到 `int64` 字段，或 `UseNumber` |
| 反序列化报 non-pointer | 传了值，反射无法写回 | 传 `&v` |
| 拼错的配置 key 被忽略 | 默认忽略未知字段 | `Decoder.DisallowUnknownFields()` |
| 自定义 MarshalJSON 失效 | 方法定义在指针接收者上 | 改成值接收者（或给外层的值取地址） |
| 栈溢出 | `MarshalJSON` 内部又 `json.Marshal(self)` | 用 `type alias T` 转换后再序列化 |
| CSV 解析报 wrong number of fields | 各行列数不一致 | 修数据，或设 `FieldsPerRecord = -1` |
| 日志里出现密码 | 直接打印/序列化配置结构体 | `String()` 脱敏 + 只传 `Redacted()` 给日志 |
| 测试之间环境变量串味 | 用了 `os.Setenv` | 测试里用 `t.Setenv` |

---

## 练习

### 第 1 题

给下面这个 PATCH 请求体设计字段类型，要求：`name` 不传表示「不改」，传 `""` 表示「改成空串」；`age` 同理（0 是合法年龄）。

```go
type PatchUser struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}
```

::: details 第 1 题参考答案

```go
type PatchUser struct {
    Name *string `json:"name"` // nil = 不改；指向 "" = 改成空串
    Age  *int    `json:"age"`  // nil = 不改；指向 0 = 改成 0
}
```

判断方式：

```go
if patch.Name != nil {
    user.Name = *patch.Name
}
```

**为什么这样写**：`omitempty` 和零值都无法区分「没传」和「传了零值」，而指针可以——`nil` 表示「字段不存在」，指向零值表示「显式传了零值」。代价是取值前要判空，所以只在与「部分更新」相关的地方使用，普通接口仍然用值类型更省心。

:::

### 第 2 题

给 `User` 增加一个 `CreatedAt time.Time` 字段，要求序列化成 Unix 秒（形如 `"created_at": 1735689600`），反序列化时也能接收同一个格式。写出实现并说明如何避免 18.5 里的递归陷阱。

::: details 第 2 题参考答案

```go
// UnixTime 以 Unix 秒的形式参与 JSON 编解码。
type UnixTime time.Time

// MarshalJSON 用值接收者，避免「值字段不生效」的坑。
func (t UnixTime) MarshalJSON() ([]byte, error) {
    return json.Marshal(time.Time(t).Unix())
}

// UnmarshalJSON 用指针接收者，因为要写回调用方。
func (t *UnixTime) UnmarshalJSON(data []byte) error {
    var seconds int64
    if err := json.Unmarshal(data, &seconds); err != nil {
        return fmt.Errorf("created_at 需要 Unix 秒: %w", err)
    }
    *t = UnixTime(time.Unix(seconds, 0).UTC())
    return nil
}

type User struct {
    Name      string   `json:"name"`
    CreatedAt UnixTime `json:"created_at"`
}
```

验证：

```go
u := User{Name: "Ada", CreatedAt: UnixTime(time.Unix(1735689600, 0).UTC())}
data, _ := json.Marshal(u)
fmt.Println(string(data)) // {"name":"Ada","created_at":1735689600}
```

**关于递归**：`UnixTime` 的方法体里调用的是 `time.Time(t).Unix()`（转换 + 普通方法），**没有**对 `UnixTime` 自己再调 `json.Marshal`，所以不存在递归。如果确实需要「按默认规则序列化自己」，用 `type plain UnixTime` 转换一次再序列化即可。

**为什么不用 `string` 时间格式**：Unix 秒跨语言、跨时区都无歧义，省掉了时区和格式协商；需要可读性时再用 RFC3339（`time.Time` 的默认格式）。

:::

### 第 3 题

下面这段代码把订单 ID 从 `9007199254740993` 变成了 `9007199254740992`。解释原因，并给出三种修复方式。

```go
var body map[string]any
json.Unmarshal(data, &body)
id := body["id"].(float64)
```

::: details 第 3 题参考答案

**原因**：解到 `any` 时，`encoding/json` 把 JSON 数字统一放进 `float64`。`float64` 只有 53 位有效位，`9007199254740993`（2^53 + 1）无法精确表示，只能存成 `9007199254740992`，于是精度丢失。

**修复一：解到具体类型**

```go
var order struct {
    ID int64 `json:"id"`
}
json.Unmarshal(data, &order)
```

**修复二：使用 `Decoder.UseNumber()`**

```go
decoder := json.NewDecoder(bytes.NewReader(data))
decoder.UseNumber()
var body map[string]any
decoder.Decode(&body)
id, err := body["id"].(json.Number).Int64()
```

**修复三：从源头就用字符串传输**

```json
{"id": "9007199254740993"}
```

很多开放平台直接把大整数 ID 定义成字符串，就是为了避免各语言 JSON 实现的精度差异。

**为什么推荐第一种**：结构体字段有明确类型，全程没有装箱和浮点转换，性能最好、语义最清楚；只有「结构不固定」时才用 `UseNumber`。

:::

### 第 4 题

写一个 `CountNDJSON(r io.Reader) (int, error)`：统计一行一个 JSON 的输入里有多少条记录；遇到非法行时返回错误，并带上行号。

::: details 第 4 题参考答案

```go
func CountNDJSON(r io.Reader) (int, error) {
    decoder := json.NewDecoder(r)
    count := 0
    for {
        var raw json.RawMessage
        err := decoder.Decode(&raw)
        if errors.Is(err, io.EOF) {
            return count, nil
        }
        if err != nil {
            return count, fmt.Errorf("第 %d 条记录解析失败: %w", count+1, err)
        }
        count++
    }
}
```

要点：

- 用 `json.RawMessage` 承接，只校验「是不是合法 JSON」，不关心具体字段；
- `io.EOF` 表示正常结束，必须用 `errors.Is` 判断；
- 错误信息里的 `count+1` 就是出错的行号（NDJSON 一行一条记录）。

测试：

```go
n, err := CountNDJSON(strings.NewReader("{\"a\":1}\n{\"b\":2}\n"))
// n = 2, err = nil

n, err = CountNDJSON(strings.NewReader("{\"a\":1}\n{oops}\n"))
// n = 1, err = 第 2 条记录解析失败: invalid character 'o' looking for beginning of object key string
```

**为什么用 `RawMessage`**：它的单位是「一个完整的 JSON 值」，既能校验合法性又不做多余解析；如果直接解到 `map[string]any`，既慢又丢精度。

:::

### 第 5 题

给 18.9 的 `LoadConfig` 增加第四层「命令行参数」（优先级最高）。要求实现可测试：不能在函数里直接读 `os.Args`。

::: details 第 5 题参考答案

把「命令行参数查询」也抽象成函数类型，保持依赖注入：

```go
// FlagLookup 抽象命令行参数查询：生产传解析结果，测试传假实现。
type FlagLookup func(name string) (string, bool)

func LoadConfigWithFlags(defaults Config, fileJSON string, env EnvLookup, flags FlagLookup) (Config, error) {
    cfg, err := LoadConfig(defaults, fileJSON, env) // 复用前三层
    if err != nil || flags == nil {
        return cfg, err
    }
    if value, ok := flags("port"); ok {
        port, err := ParseInt64("--port", value)
        if err != nil {
            return Config{}, err
        }
        cfg.Port = int(port)
    }
    // 其余字段同理……
    return cfg, nil
}
```

生产环境里用 `flag` 包解析一次，再把结果包成 `FlagLookup`：

```go
port := flag.Int("port", 0, "监听端口")
flag.Parse()
flags := func(name string) (string, bool) {
    if name == "port" && *port != 0 {
        return strconv.Itoa(*port), true
    }
    return "", false
}
```

测试里直接传假实现，完全不碰 `os.Args`：

```go
cfg, err := LoadConfigWithFlags(defaults, `{"port":9000}`, env, func(name string) (string, bool) {
    return "6000", name == "port"
})
// cfg.Port == 6000，命令行覆盖了文件和环境变量
```

**为什么这样拆分**：把「数据从哪来」抽象成函数，测试就能在不改全局状态的前提下覆盖任意优先级组合；直接读 `os.Args` 会让测试只能通过改进程参数来构造场景。

:::

### 第 6 题

有人建议「给 `Config` 实现 `MarshalJSON` 做脱敏，这样所有输出都安全」。请说明这样做的问题，并给出更合适的方案。

::: details 第 6 题参考答案

**问题**：

1. `MarshalJSON` 会**全局生效**：日志安全了，但需要把配置原样写入加密存储、或传给需要真实密码的下游组件时，序列化结果也是脱敏的，等于数据丢失；
2. 脱敏规则写死在类型上，不同环境（本地调试 vs 生产审计）无法差异化；
3. 隐藏了信息：调用方看到 `"password":"s***********"` 时无法判断「本来就没设置」还是「被脱敏了」。

**更合适的方案**：显式提供一个脱敏视图，让「要打日志」这件事在调用点可见。

```go
// Redacted 返回可以安全写进日志的副本。
func (c Config) Redacted() LogView {
    return LogView{
        Host:     c.Host,
        Port:     c.Port,
        LogLevel: c.LogLevel,
        Password: MaskSecret(c.Password),
    }
}

log.Printf("启动配置: %+v", cfg.Redacted())
```

配合 `String()` 兜底：即使有人直接打印 `Config`，看到的也是脱敏结果（示例里的 `Password:s***********`）。

**为什么这样更好**：把「脱敏」变成一个需要显式调用的动作，审计时一眼就能看出哪些出口带了敏感数据；同时原始值仍然可用，不会被全局规则剥夺。真正的防线是「敏感字段永不进日志」的评审约定，而不是指望类型自己记住。

:::

---

## 小结

- **json 标签四条规则**：`omitempty` 省略零值、`-` 永不输出、`,string` 数字转字符串、未导出字段一律跳过；标签写错是静默失效，用 `go vet` 兜底。
- **`omitempty` 的「空」有明确清单**：0、false、空串、nil 指针、空切片/空 map；**结构体不算空**，要省略就用指针。
- **区分「零值」和「缺失」用指针**：PATCH 类接口的字段定义成 `*T`，`nil` 表示不改，指向零值表示显式修改。
- **大整数必须小心**：用 `any` 接 JSON 数字会变成 `float64`（超 2^53 丢精度）；解到 `int64` 字段或开 `UseNumber`，跨语言接口直接把 ID 定义成字符串。
- **流式数据用 `Decoder`**：它能连续解析多个 JSON 值、支持 `DisallowUnknownFields` 严格校验；`io.EOF` 是正常结束信号，用 `errors.Is` 判定。
- **自定义编解码的接收者约定**：`MarshalJSON` 用值接收者（否则值字段不生效），`UnmarshalJSON` 用指针接收者；方法里别再 `json.Marshal` 自己，否则栈溢出。
- **格式选型看对方是谁**：对外 JSON、配置 YAML/TOML、Go 之间 gob、跨语言高性能 protobuf；gob 不一定更小，但一定不跨语言。
- **配置分层固定顺序**：默认值 → 配置文件 → 环境变量（→ 命令行参数），每层只覆盖自己提供的字段；环境变量用 `LookupEnv` 区分「未设置」和「空值」。
- **敏感信息默认脱敏**：`String()` 兜底，日志只接收 `Redacted()`；直接 `json.Marshal(cfg)` 不会走 `String()`，这是最容易漏掉的出口。

下一章进入测试、基准与代码质量：把这一章里那些「只能靠运行时才发现」的问题（精度、标签、脱敏），用表驱动测试、模糊测试和 lint 规则钉死在 CI 里。
