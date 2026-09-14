# 第 14 章 · 反射

前面几章里，类型都是编译期就写死的：`Person`、`[]string`、`string`——编译器知道每个变量是什么类型，因此能做类型检查、能内联、能把字段访问编译成一条指令。但在某些场景下，**类型本身是运行时才知道的**：把任意结构体序列化成 JSON、根据配置文件里的字段名给结构体赋值、按名字调用插件暴露的方法。这些需求用静态类型都表达不出来，只能用反射。

反射的代价很直接：它在运行时解析类型信息，绕过了编译器的检查，`reflect.Value.Set` 写错一个字段名不会编译失败，而是直接 panic。所以这一章除了讲「怎么用」，更重要的是讲清三条定律背后的约束——什么时候能改值、什么时候必须传指针、什么时候 `Kind` 会和 `Type` 不一致，以及 `errors` 那章强调过的原则在反射里同样成立：**边界处校验，热路径别用**。

本章按「三定律 → 取值与改值 → 标签与动态调用 → 性能与取舍」的顺序展开，最后手写一个简化版 `json.Marshal`，对照标准库看看 `encoding/json` 在背后究竟做了什么。读完你应该能回答：为什么 `ValueOf(x).SetInt()` 会 panic，为什么 `json.Unmarshal` 必须传指针，以及什么时候该用泛型或接口代替反射。

本章配套代码在 `internal/chapter/go14_reflection/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

---

## 14.1 反射三定律与 TypeOf/ValueOf

结论：`reflect.TypeOf` 拿到**类型**，`reflect.ValueOf` 拿到**值**，两者都是从接口值里「拆」出来的。反射三定律是：

1. 反射可以把接口值拆成 `Type` 和 `Value`；
2. 从 `Value` 可以用 `Interface()` 还原成接口值；
3. 要修改值，`Value` 必须是**可寻址**的（通常来自指针的 `Elem()`）。

```go
var value any = 42
fmt.Println(reflect.TypeOf(value))  // int
fmt.Println(reflect.ValueOf(value)) // 42
fmt.Println(reflect.ValueOf(value).Interface()) // 42（any）
```

### 实测输出

```text
TypeOf(42) = int（Kind=int，Name=int）
ValueOf(42) = 42（Kind=int，Int()=42）
val.Interface() = 42（int）
TypeOf(nil) = <nil>
ValueOf(nil).IsValid() = false，Kind = invalid
```

注意最后两行：`TypeOf(nil)` 返回的是 `nil`（一个「没有类型」的 Type），`ValueOf(nil)` 返回**无效值**，`IsValid()` 是 `false`、`Kind()` 是 `invalid`。所以在处理 `any` 参数时，第一件事永远是判空：

```go
func inspect(v any) {
    rv := reflect.ValueOf(v)
    if !rv.IsValid() {
        return // 拿到的是 nil
    }
    // ...
}
```

### 边界与坑

- `TypeOf` 和 `ValueOf` 接收的都是 `any`，传进去的值会被**装箱**成接口；这也是反射比直接访问慢的原因之一。
- `Value.Interface()` 在 `Value` 无效时会 panic（`reflect: call of reflect.Value.Interface on zero Value`），所以先 `IsValid()`。
- 传 `nil` 接口和传「装了 nil 的具体指针」是两回事（第 11 章的 typed nil 陷阱），对反射同样成立。

---

## 14.2 Kind 与 Type 不是一回事

结论：`Type` 是「具体类型」（`Celsius`、`[]int`、`*Person`），`Kind` 是「底层种类」（`float64`、`slice`、`ptr`）。分支处理一律看 `Kind`，因为 `Kind` 的数量有限（26 种），`Type` 是无穷的。

```go
type Celsius float64

ct := reflect.TypeOf(Celsius(36.6))
fmt.Println(ct.String()) // go14_reflection.Celsius
fmt.Println(ct.Name())   // Celsius（匿名类型是空字符串）
fmt.Println(ct.Kind())   // float64
```

### 实测输出

```text
Celsius: Type=go14_reflection.Celsius，Name="Celsius"，Kind=float64
PkgPath="go-learn/internal/chapter/go14_reflection"（命名类型才有包路径）
```

同一个 `Kind` 可以对应很多 `Type`：

```text
  Kind=int      Type=int
  Kind=string   Type=string
  Kind=float64  Type=float64
  Kind=bool     Type=bool
  Kind=uint8    Type=uint8
  Kind=int32    Type=int32
  Kind=slice    Type=[]int
  Kind=array    Type=[2]int
  Kind=map      Type=map[string]int
  Kind=struct   Type=go14_reflection.Person
  Kind=ptr      Type=*go14_reflection.Person
  Kind=chan     Type=chan int
  Kind=func     Type=func()
```

几个容易记混的点：`byte` 的 Kind 是 `uint8`，`rune` 的 Kind 是 `int32`（第 2 章讲过它们是别名）；`Type.Name()` 对匿名类型（如 `[]int`、`map[string]int`）返回空字符串；`PkgPath()` 只有命名类型才有值。

### 边界与坑

- 判断「是不是某个具体类型」用 `Type` 比较（`t == reflect.TypeOf(User{})`），判断「能做什么操作」用 `Kind`。
- `reflect.Type` 是可比较的接口值，可以直接当 map 的 key 做类型缓存——这是标准库和很多框架的常规优化手段。
- 指针的 `Kind()` 是 `ptr`，不要以为取 `Elem()` 后的 Kind 会自动「继承」过来。

---

## 14.3 从 Value 取值：按 Kind 分支

结论：`Value` 提供了一组 `Int()`、`String()`、`Bool()`、`Float()` 之类的取值方法，**必须按 `Kind` 调用对应的方法**；用错了多数不会报错，而是给出意外的结果。

```go
number := reflect.ValueOf(42)
fmt.Println(number.String()) // "<int Value>"，不是 "42"
fmt.Println(number.Int())    // 42
fmt.Println(number.Interface()) // 42（any）
```

### 实测输出

```text
字符串 Value.String() = "hello"
整数   Value.Int()    = 42
浮点   Value.Float()  = 3.5
布尔   Value.Bool()   = true
整数调 String() = "<int Value>"（不是 "42"）
想拿原值就用 Interface()：42（int）
```

`Value.String()` 是个特例：它被设计成「无论如何都能返回字符串」，对非字符串类型返回的是 `<类型 Value>` 这样的描述，而不是 panic。所以它常被用来调试，**不能**用来取值。

正确的取值套路是先看 `Kind`：

```go
switch v.Kind() {
case reflect.String:
    s := v.String()
case reflect.Int, reflect.Int64:
    n := v.Int()
case reflect.Bool:
    b := v.Bool()
default:
    // 其它类型走 Interface() 或返回错误
}
```

### 边界与坑

- `Int()`、`Bool()`、`Float()` 用错 Kind 会直接 panic（例如对 `string` 调 `Int()`），所以顺序永远是「先 Kind，后取值」。
- `Interface()` 是最通用的取值方式，但它要把值装箱成 `any`，比按 Kind 走专用方法慢不少（本章基准里约 33 ns/op，而按下标取字段是 1.5 ns/op）；热路径上优先用专用方法。
- 对指针调用 `Interface()` 拿到的是指针本身，`IsNil()` 可以判断它是否为空指针。

---

## 14.4 可寻址性与 Set：改值必须通过指针

结论：`reflect.ValueOf(x)` 拿到的是 `x` 的**副本**，不可寻址、不可修改；想改调用方的变量，必须传指针，再用 `Elem()` 取到可寻址的值。这就是第三定律。

```go
n := 10
reflect.ValueOf(n).CanSet()            // false
reflect.ValueOf(&n).Elem().CanSet()    // true
reflect.ValueOf(&n).Elem().SetInt(20)  // n 变成 20
```

### 实测输出

```text
ValueOf(n).CanSet() = false（只是副本，改不了）
ValueOf(&n).Elem()：CanAddr() = true，CanSet() = true
SetInt(20) 之后 n = 20
结构体字段：CanSet() = true，原值 = "Alice"
SetString 之后 p.Name = "Bob"
未导出字段 "note"：CanSet() = false（反射也不能绕过可见性）
```

### 三个层次的「能不能改」

| 检查 | 含义 | 什么时候成立 |
| --- | --- | --- |
| `IsValid()` | 是不是一个有效的 `Value` | 不是 `ValueOf(nil)` |
| `CanAddr()` | 能不能取地址 | 来自指针 / 切片元素 / 可寻址的结构体字段 |
| `CanSet()` | 能不能被赋值 | `CanAddr()` 为真，且字段是导出的 |

### 结构体字段的两种情况

```go
p := &Person{Name: "Alice"}
reflect.ValueOf(p).Elem().FieldByName("Name").CanSet() // true，已导出
reflect.ValueOf(p).Elem().FieldByName("note").CanSet() // false，未导出
```

未导出字段通过反射**可以读**（`Field(i)` 拿得到，值也能打印），但**不能写**——这是语言层面的可见性约束在反射里的延续，不是缺陷。想改就只能通过包内的方法或导出的字段。

### 边界与坑

- 直接对不可寻址的 `Value` 调 `Set*` 会 panic：`reflect: reflect.Value.SetInt using unaddressable value`（见报错 4）。
- 传指针时别忘了 `Elem()`：`ValueOf(&n)` 的 Kind 是 `ptr`，直接对它调 `SetInt` 一样会 panic，必须先 `Elem()` 拿到指针指向的那个值。
- 工程代码里不要直接 `Set*`，先检查 `CanSet()` 再动手，把「用法错误」变成返回 error（本章的 `SetInt64Field` 就是这么写的）。

---

## 14.5 结构体字段与标签解析

结论：结构体反射的入口是 `Type.NumField()` / `Field(i)`，字段的元信息（名字、类型、标签、是否导出）全在 `reflect.StructField` 里；标签就是一段普通字符串，**取值时要自己解析**。

```go
type User struct {
    ID       int    `json:"id"`
    Name     string `json:"name"`
    Password string `json:"-"`
    Nickname string `json:"nickname,omitempty"`
    internal string
}
```

### 实测输出

```text
User 有 5 个字段：
  ID        int     json="id"                   有标签=true 导出=true
  Name      string  json="name"                 有标签=true 导出=true
  Password  string  json="-"                    有标签=true 导出=true
  Nickname  string  json="nickname,omitempty"   有标签=true 导出=true
  internal  string  json=""                     有标签=false 导出=false
```

### 标签的取值与解析

```go
field, _ := t.FieldByName("Nickname")
raw := field.Tag.Get("json")          // "nickname,omitempty"，没有该标签时返回 ""
raw, ok := field.Tag.Lookup("json")   // 想区分「没有标签」和「标签为空」时用 Lookup
```

标准库的做法也很朴素：按逗号切开，第一段是名字，其余是选项。本章的 `ParseTag` 就是同一个思路：

```text
  Name      → 输出名 "name"       选项 []
  Nickname  → 输出名 "nickname"   选项 [omitempty]
  Password  → 输出名 "-"          选项 []
```

注意 `json:"-"` 和 `omitempty` 是两种完全不同的东西：前者表示「这个字段不参与序列化」，后者表示「零值时不输出」。`Password` 的输出名就是字面量 `-`，由使用方（这里是 `encoding/json`）解释成「跳过」。

### 用标签推导输出字段名

```go
// JSONFieldNames 返回按 `json` 标签序列化时会用到的字段名。
func JSONFieldNames(v any) ([]string, error) {
    t, err := structType(v) // 前置检查：nil、指针解引用、非结构体报错
    if err != nil {
        return nil, err
    }
    names := make([]string, 0, t.NumField())
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        if !field.IsExported() { // 未导出字段不参与序列化
            continue
        }
        name, _ := ParseTag(field.Tag.Get("json")) // 标签就是普通字符串
        switch name {
        case "-": // json:"-" 表示跳过
            continue
        case "": // 没有标签时退回字段名
            name = field.Name
        }
        names = append(names, name)
    }
    return names, nil
}
```

### 边界与坑

- `Tag.Get` 对不存在的 key 返回空字符串，和「标签值就是空」无法区分；需要区分时用 `Tag.Lookup`。
- 标签写错格式（少个引号）不会编译失败，只是被解析器忽略，字段名会退回 Go 字段名——`go vet` 能查到这类问题，见报错 3。
- 只有导出字段才会被 `encoding/json` 序列化，给未导出字段写 `json` 标签是无意义的（`go vet` 也会提醒）。

---

## 14.6 动态调用方法

结论：`Value.MethodByName(name)` 拿到方法，`Call(args)` 执行，返回值是 `[]reflect.Value`。参数个数、类型只在运行时检查，所以调用前必须自己校验，否则 `Call` 会直接 panic。

```go
m := reflect.ValueOf(recv).MethodByName("Add")
if !m.IsValid() { /* 方法不存在 */ }
out := m.Call([]reflect.Value{reflect.ValueOf(2), reflect.ValueOf(3)})
fmt.Println(out[0].Interface()) // 5
```

### 实测输出

```text
CallByName(Add, 2, 3)      = [5]，err = <nil>
CallByName(Divide, 10, 0)  = [0 divide by zero]，err = <nil>
CallByName(Missing)        → err = CallByName: *go14_reflection.Calculator 上没有方法 "Missing"
CallByName(Add, 1)         → err = CallByName: Add 需要 2 个参数，收到 1 个
CallByName(Add, 1, "x")      → err = CallByName: Add 第 2 个参数需要 int，收到 string
```

注意第二行：`Divide` 返回 `(float64, error)`，反射拿到的 `[]reflect.Value` 里第二个元素就是那个 error——**反射调用不会把方法返回的错误「展开」**，调用方要自己判断它是不是 `error` 并向上返回。

### 方法集与接收者

`MethodByName` 能看到的方法取决于**方法集**（第 9 章讲过）：

| 接收者 | 通过值调用 | 通过指针调用 |
| --- | --- | --- |
| 值接收者方法 | ✅ | ✅ |
| 指针接收者方法 | ❌（不在方法集里） | ✅ |

所以动态调用时统一传指针（`&Calculator{}`），否则指针接收者的方法会「找不到」。

### 边界与坑

- 参数类型必须**可赋值**给方法形参：`int` 传不进去 `int64` 参数，反射不会替你隐式转换。
- 变参方法在反射眼里就是「最后一个参数是切片」：`NumIn()` 把变参算作一个切片参数，`Call` 也要求传这个切片；想让反射替你展开，得用 `CallSlice`。
- 返回值里的 `error` 需要显式检查；把 `[]reflect.Value` 直接 `Interface()` 成 `any` 会丢掉类型信息。

---

## 14.7 用反射创建值

结论：反射不仅能读，还能**造**——`reflect.New` 造指针、`MakeSlice` / `MakeMap` 造容器、`Zero` 造零值。它们是「运行时才知道类型」的工厂。

```go
ptr := reflect.New(reflect.TypeOf(Person{})) // 等价于 &Person{}
ptr.Elem().FieldByName("Name").SetString("Carol")
fmt.Printf("%+v\n", ptr.Interface())         // &{Name:Carol Age:0}
```

### 实测输出

```text
reflect.New(go14_reflection.Person) → Kind=ptr，Elem 的类型 go14_reflection.Person
设置字段后：&{Name:Carol Age:0 note:}
MakeSlice + Append = [7]，len=1，cap=3
MakeMap + SetMapIndex = map[k:1]
reflect.Zero(int) = 0，IsZero=true
```

### 对应关系

| 反射写法 | 等价代码 | 说明 |
| --- | --- | --- |
| `reflect.New(t)` | `&T{}` | 返回 `*T` 的 Value（Kind 是 `ptr`） |
| `reflect.Zero(t)` | `var v T` | 该类型的零值，可读不可寻址 |
| `reflect.MakeSlice(t, len, cap)` | `make([]T, len, cap)` | 只能用于 slice |
| `reflect.MakeMap(t)` | `make(map[K]V)` | 只能用于 map |
| `reflect.MakeChan(t, size)` | `make(chan T, size)` | 只能用于 channel |

`Kind` 决定了哪些工厂函数可用：对 `int` 调 `MakeSlice` 会 panic（`reflect.MakeSlice of non-slice type`），所以调用前先判断 Kind。

### 工程上的典型用法

反序列化框架就是靠这套 API 工作的：拿到目标类型 → `New` 一个指针 → 递归填充字段 → 返回。本章 14.10 的 `UnmarshalInto` 就是这个思路的最小版本。

### 边界与坑

- `reflect.New` 返回的指针指向**零值**，所有字段都是零值，不要以为它会调用构造函数。
- `MakeSlice` 的参数是「长度」和「容量」，写反了不会报错，只是行为和你预期不同。
- 反射创建的对象同样受可见性约束：未导出字段依旧 `CanSet() == false`。

---

## 14.8 reflect.DeepEqual 的边界

结论：`reflect.DeepEqual` 是「递归比较」，能比较切片、map、指针指向的内容，但它**不等于** `==`，语义上有几个反直觉之处。

```go
reflect.DeepEqual([]int{1, 2, 3}, []int{1, 2, 3}) // true，切片不能用 ==
reflect.DeepEqual([]int(nil), []int{})            // false，nil 切片 ≠ 空切片
reflect.DeepEqual(math.NaN(), math.NaN())         // false，NaN 永远不等于自己
```

### 实测输出

```text
切片不能用 ==，DeepEqual([]int{1,2,3}, 同内容) = true
nil 切片 vs 空切片：len 都是 0/0，但 DeepEqual = false
同内容 map：DeepEqual = true
NaN：== 比较 false，DeepEqual = false（DeepEqual 不是「数学相等」）
不同指针、相同内容：DeepEqual = true
含函数字段的同值结构体：DeepEqual = false（函数只有都为 nil 才相等）
```

### 三条使用建议

1. **同类型之间比较**：`DeepEqual(1, int64(1))` 是 `false`，类型不同直接不相等，它不会做数值转换。
2. **别拿它比较含指针的复杂结构**：`DeepEqual` 会顺着指针往下递归，遇到环（自引用结构）会一直走下去，最终栈溢出。
3. **优先用具体方法**：切片用 `slices.Equal`，map 用 `maps.Equal`（都要求元素可比较），既快又不会踩到上面的坑；`DeepEqual` 更多是测试代码里的兜底手段。

### 边界与坑

- `nil` map 和空 map：`DeepEqual(map[string]int(nil), map[string]int{})` 同样是 `false`。
- 结构体里含 `sync.Mutex` 之类的内部状态字段时，比较结果没有业务意义。
- 性能上 `DeepEqual` 全程走反射，比手写比较慢得多，不要放在热路径上。

---

## 14.9 反射的性能代价

结论：反射不是「稍微慢一点」，而是**按名字查找**这类操作会慢一到两个数量级。热路径上要么不用反射，要么把反射结果缓存起来。

```bash
go test -run '^$' -bench=. -benchmem ./internal/chapter/go14_reflection/
```

在本机（Windows / amd64，Intel i7-14700F，Go 1.27）跑两次的典型结果：

```text
BenchmarkDirectField-28            	1000000000	         0.3875 ns/op	       0 B/op	       0 allocs/op
BenchmarkReflectFieldByIndex-28    	767115301	         1.544 ns/op	       0 B/op	       0 allocs/op
BenchmarkReflectFieldByName-28     	37484302	        32.27 ns/op	       0 B/op	       0 allocs/op
BenchmarkReflectInterface-28       	36290614	        33.21 ns/op	       0 B/op	       0 allocs/op
BenchmarkJSONMarshal-28            	 7146676	       168.0 ns/op	      96 B/op	       3 allocs/op
BenchmarkSimpleJSON-28             	 3088284	       410.6 ns/op	     176 B/op	      12 allocs/op
```

### 怎么读这组数字

| 写法 | 耗时 | 相对直接访问 | 原因 |
| --- | --- | --- | --- |
| 直接读字段 | 约 0.39 ns | 1× | 编译器生成一条加载指令 |
| 反射按下标读字段 | 约 1.5 ns | 4× | 有边界检查和标志位判断，但没有分配 |
| 反射按名字读字段 | 约 32 ns | 83× | 要按名字查字段表，每次调用都查 |
| 反射取值转接口 | 约 33 ns | 85× | 装箱成 `any` + 运行时方法调用 |
| `encoding/json` 序列化 | 约 168 ns | — | 反射开销被类型缓存摊薄 |
| 手写反射序列化 | 约 411 ns | — | 每次重新查字段名、重复分配 |

（`-28` 是并行度，等于 GOMAXPROCS；具体数字随机器和 Go 版本浮动，有争议时跑一遍自己的基准。）

### 三条优化思路

1. **类型信息缓存一次，反复使用**：`FieldByName` 的结果（字段下标）在同一个类型上是固定的，缓存 `reflect.Type` → 字段下标即可把 32 ns 降到接近 1.5 ns。
2. **能用接口或泛型就不用反射**：编译期能确定的事情，没必要搬到运行时。
3. **批量处理时把反射限制在入口**：比如反序列化只在请求进入时做一次，后续业务逻辑用普通结构体。

---

## 14.10 encoding/json 内部在做什么

结论：`encoding/json` 就是「反射 + 标签 + 缓存」的工业级实现。本章写了一个 100 多行的 `SimpleJSON`，输出与标准库完全一致，但性能和健壮性差得远——这个差距正好说明了工程化的价值。

### 手写实现的核心：按 Kind 递归

```go
func writeJSON(sb *strings.Builder, v reflect.Value) error {
    if !v.IsValid() { sb.WriteString("null"); return nil }
    switch v.Kind() {
    case reflect.Pointer, reflect.Interface:
        if v.IsNil() { sb.WriteString("null"); return nil }
        return writeJSON(sb, v.Elem())
    case reflect.Struct:
        return writeJSONStruct(sb, v)   // 遍历字段 + 读 json 标签
    case reflect.Map:
        return writeJSONMap(sb, v)      // key 排序后逐个输出
    case reflect.Slice, reflect.Array:
        return writeJSONSlice(sb, v)
    case reflect.String:
        quoted, err := json.Marshal(v.String()) // 转义交给标准库
        ...
    }
}
```

`writeJSONStruct` 里做的判断，和 `encoding/json` 的规则一一对应：跳过未导出字段、标签 `-` 跳过、`omitempty` 时跳过零值、其余按标签名或字段名输出。

### 实测输出

```text
encoding/json = {"id":1,"name":"Ada"}
手写反射实现  = {"id":1,"name":"Ada"}
两者一致：true，err = <nil>
Balance=30 时：encoding/json = {"id":2,"name":"Bob","balance":30}，手写实现 = {"id":2,"name":"Bob","balance":30}，一致：true
```

第一个 `Account` 的 `Balance` 是 0，带 `omitempty` 被跳过；未导出的 `secret` 字段被忽略——两条规则手写实现都复现了。

### 为什么标准库更快

同样的输入，`encoding/json` 约 168 ns/3 allocs，手写版约 411 ns/12 allocs。差距来自三处：

1. **字段信息缓存**：标准库把「类型 → 字段列表 + 编码器」缓存起来，同一个类型只解析一次，之后直接按下标访问；手写版每次都在 `FieldByName` 和字符串拼接上重新查。
2. **专门的编码器**：为 `int`、`string` 等类型生成 encode 函数，避免通用分支和多余的接口装箱。
3. **缓冲区复用**：通过 `Encoder` 复用内部缓冲，而不是每次都新建 `strings.Builder`。

### 反序列化为什么必须传指针

```text
Unmarshal 到指针：{ID:3 Name:Cy Balance:7 secret:}，err = <nil>
Unmarshal 传值 → err = json: Unmarshal(non-pointer go14_reflection.Account)
手写 UnmarshalInto 到指针：{Name:Eve Age:25 note:}
手写 UnmarshalInto 传值 → err = UnmarshalInto: 目标必须是指针，收到 struct
```

标准库和手写实现的报错信息不同，但原因完全一样：反序列化要往调用方的变量里**写值**，而反射只有拿到指针才能得到可寻址的 `Value`（第三定律）。顺带注意手写版把未导出的 `note` 跳过了——`CanSet()` 为假就写不进去。

### 边界与坑

- `encoding/json` 只处理导出字段，给未导出字段打 `json` 标签是无效操作（`go vet` 会提示）。
- 反射版实现天然不支持循环引用，遇到自引用结构会无限递归；标准库编码时会在递归到一定深度后开始检测指针复用，直接返回 `json: unsupported value: encountered a cycle via *main.node`，而不是爆栈。
- 想给类型自定义序列化行为，实现 `json.Marshaler` / `json.Unmarshaler` 即可，不需要自己写反射——接口优先于反射。

---

## 14.11 什么时候不该用反射

结论：反射适合「类型在运行时才知道、且只做一次」的场景；其余情况下，接口、泛型、代码生成几乎总是更好的选择。

### 优先级的顺序

| 需求 | 首选方案 | 理由 |
| --- | --- | --- |
| 一组类型共享行为 | 接口 | 编译期检查，零运行时开销 |
| 逻辑相同、类型不同 | 泛型 | 保留类型安全，实例化后仍是静态代码 |
| 固定结构的数据转换 | 手写代码或 `go generate` | 完全可控，性能最好 |
| 运行期才拿到类型 | 反射 | 唯一可行的办法，但要限制在边界 |

### 本章代码里的正面例子

`SetInt64Field` 把「可能 panic 的反射操作」包装成了返回 error 的函数，边界清晰：

```text
用反射设置年龄：{Name:Dora Age:42 note:}
  传值（不可寻址） → SetInt64Field: 需要非 nil 的结构体指针，收到 go14_reflection.Person/struct
  未导出字段 → SetInt64Field: 字段 "note" 不可设置（未导出或值不可寻址）
  不存在的字段 → SetInt64Field: go14_reflection.Person 没有字段 "Email"
  类型不匹配 → SetInt64Field: 字段 "Name" 是 string，不是整数
```

每一次失败都给出了「为什么不行」，而不是让调用方去猜 panic 栈。

### 类型缓存：反射场景的标配

```go
cache := map[reflect.Type][]string{} // reflect.Type 可比较，能直接当 key
```

实测输出：

```text
reflect.Type 可比较，能当 map key 做类型缓存：[Name Age note]
```

注意缓存里连未导出字段名都能拿到——反射能「看见」它们，只是不能改。

### 边界与坑

- 反射代码要把「类型判断」和「取值」分开写，先把 `Kind`、`CanSet`、`IsNil` 都检查完再动手。
- 反射是 `internal` 实现细节的好朋友，但不要把它暴露成公开 API 的语义（调用方看不到编译期约束）。
- 写框架时优先看标准库怎么做的：`encoding/json`、`text/template`、`database/sql` 的反射用法都是现成的范例。

---

## 5 个真实报错怎么读

### 报错 1：Call 的参数类型不对

```go
value := reflect.ValueOf(42)
values := value.Call(1)
```

编译输出：

```text
main.go:11:23: cannot use 1 (untyped int constant) as []reflect.Value value in argument to value.Call
```

**怎么读**：`Call` 的签名是 `Call(in []Value) []Value`，参数必须是一个 `[]reflect.Value` 切片，而不是「若干个值」。这个错误好在编译期就暴露了——只要参数类型写错，编译器直接拦住。

**怎么改**：把参数装进切片，或者用 `CallByName` 这类封装：

```go
values := value.Call([]reflect.Value{reflect.ValueOf(2), reflect.ValueOf(3)})
```

### 报错 2：结构体标签少写了引号

```go
type Config struct {
    Name string `json:name` // 少了一对引号
}
```

`go vet` 输出：

```text
main.go:9:2: struct field tag `json:name` not compatible with reflect.StructTag.Get: bad syntax for struct tag value
```

而这段代码**能编译、能运行**，`json.Marshal(Config{Name: "app"})` 的输出是：

```text
{"Name":"app"}
```

**怎么读**：标签的语法要求是 `key:"value"`，缺了引号时 `reflect.StructTag.Get("json")` 解析失败、返回空字符串，`encoding/json` 于是退回使用 Go 字段名 `Name`。**这不会报错，只会让契约悄悄失效**——所以结构性标签一定要让 `go vet` 过一遍。

**怎么改**：写全引号 `json:"name"`，并在 CI 里保留 `go vet ./...`。顺带一提，`go vet` 还会提醒「未导出字段带着 json 标签」这类无效标签。

### 报错 3：Unmarshal 传了值而不是指针

```go
var c Config
err := json.Unmarshal([]byte(`{"name":"app"}`), c)
```

`go vet` 输出：

```text
main.go:14:23: call of Unmarshal passes non-pointer as second argument
```

运行时输出：

```text
{} json: Unmarshal(non-pointer main.Config)
```

**怎么读**：反序列化要往调用方的变量里写值，而反射只有在**可寻址**的 `Value` 上才能 `Set`——传值进来的是一个副本，`CanSet()` 为假。标准库选择返回错误而不是 panic，所以看到的是 `non-pointer` 而不是崩溃。

**怎么改**：传 `&c`。自己写反射式的反序列化时，也应该像 `json.Unmarshal` 一样先检查 `Kind() == reflect.Pointer`。

### 报错 4：对不可寻址的值调用 Set

```go
n := 10
value := reflect.ValueOf(n)
value.SetInt(20)
```

运行输出：

```text
panic: reflect: reflect.Value.SetInt using unaddressable value
goroutine 1 [running]:
...
main.main()
        .../main.go:11 +0x45
exit status 2
```

**怎么读**：`reflect.ValueOf(n)` 拿到的是 `n` 的副本，标志位里没有「可寻址」，所以 `SetInt` 立刻 panic。这是第三定律最直白的一次提醒：**反射改值必须通过指针**。

**怎么改**：`reflect.ValueOf(&n).Elem().SetInt(20)`；不确定时先判断 `CanSet()`，把它变成返回 error 的分支（本章的 `SetInt64Field`）。

### 报错 5：对非结构体取字段

```go
value := reflect.ValueOf(42)
fmt.Println(value.FieldByName("Name"))
```

运行输出：

```text
panic: reflect: call of reflect.Value.FieldByName on int Value
goroutine 1 [running]:
...
main.main()
        .../main.go:10 +0x4b
exit status 2
```

**怎么读**：错误信息把「你做了什么 + 拿到的是什么」都写清楚了：对 `int` 的 Value 调 `FieldByName`。反射的 panic 信息普遍是这个格式（`reflect: call of reflect.Value.Xxx on Yyy Value`），读几次就能快速定位。

**怎么改**：先判断 Kind 再取字段：

```go
if v.Kind() == reflect.Struct {
    field := v.FieldByName("Name")
    ...
}
```

### 还有一个不报错的坑

```go
number := reflect.ValueOf(42)
fmt.Println(number.String()) // 打印 "<int Value>"，不是 "42"
```

`Value.String()` 对任何类型都返回字符串，非字符串类型拿到的是类型描述。它既不报错也不 panic，但结果不是你想要的——类似地，`Kind` 判断写错、标签写错、`Interface()` 装在错误的类型上，都会以「结果不对」的形式出现，而不是以错误的形式出现。反射代码的测试因此格外重要。

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| `Set*` panic：unaddressable | 值来自 `ValueOf(x)` 的副本 | 传指针再 `Elem()`，或先判断 `CanSet()` |
| `FieldByName` panic | 目标不是结构体（Kind 不是 `struct`） | 先判 `Kind()`，再取字段 |
| `Interface()` panic | 拿到了无效 Value（`ValueOf(nil)`） | 先 `IsValid()` |
| 反射写不进未导出字段 | `CanSet()` 为假 | 通过包内方法修改，或改成导出字段 |
| `Kind` 和 `Type` 搞混 | 命名类型的 Kind 是底层类型 | 分支用 `Kind`，类型判定用 `Type` |
| 标签不起作用 | 标签语法写错（少了引号） | 写全 `key:"value"`，用 `go vet` 兜底 |
| `Unmarshal` 报 non-pointer | 目标不是指针，无法寻址 | 传 `&v` |
| 动态调用 panic | 参数个数/类型与方法签名不匹配 | 调用前用 `Method.Type()` 校验 |
| 指针接收者方法「找不到」 | 值的方法集不含指针接收者方法 | 反射时统一传指针 |
| 反射代码慢一个数量级 | 每次都按名字查字段 | 缓存 `reflect.Type` → 字段下标 |
| `DeepEqual` 结果反直觉 | nil 切片 ≠ 空切片、NaN ≠ NaN | 切片用 `slices.Equal`，map 用 `maps.Equal` |

---

## 练习

### 第 1 题

解释下面这段代码为什么 panic，并写出两种修复方式（一种改调用方，一种改函数签名）。

```go
func bump(v any) {
    reflect.ValueOf(v).SetInt(reflect.ValueOf(v).Int() + 1)
}

func main() {
    n := 10
    bump(n)
}
```

::: details 第 1 题参考答案

**为什么 panic**：`reflect.ValueOf(n)` 拿到的是 `n` 的**副本**，它不可寻址（`CanSet() == false`），对副本调用 `SetInt` 会 panic：`reflect: reflect.Value.SetInt using unaddressable value`。即使把 `n` 换成指针传进去，`ValueOf(&n)` 的 Kind 是 `ptr`，直接 `SetInt` 依然不对，得先 `Elem()`。

**修复一：改调用方（传指针，函数内部 `Elem()`）**

```go
func bump(v any) {
    rv := reflect.ValueOf(v)
    if rv.Kind() != reflect.Pointer || rv.IsNil() {
        return
    }
    elem := rv.Elem()
    if elem.Kind() != reflect.Int || !elem.CanSet() {
        return
    }
    elem.SetInt(elem.Int() + 1)
}

bump(&n) // n 变成 11
```

**修复二：改签名，让类型在编译期就确定**

```go
func bump(n *int) {
    if n == nil {
        return
    }
    *n++
}
```

**为什么这样写更好**：能被静态类型表达的需求，就不该交给反射——第二种写法编译期就能发现「传错类型」，也不会有任何运行时开销。反射版本里那一串 `Kind`/`CanSet` 检查，正是「把编译期检查搬回运行时」的代价。

:::

### 第 2 题

把 14.5 的 `JSONFieldNames` 改造成 `JSONFieldTypes(v any) (map[string]string, error)`：返回「输出字段名 → 字段类型的字符串」（例如 `{"id": "int", "name": "string"}`），跳过未导出字段和 `json:"-"`，指针入参也要支持。

::: details 第 2 题参考答案

和 14.5 的 `JSONFieldNames` 只差「收集方式」和「返回类型」两处，函数签名、指针解引用与字段筛选完全复用：

```go
func JSONFieldTypes(v any) (map[string]string, error) {
    t, err := structType(v) // 与 JSONFieldNames 共用的前置检查：nil / 指针 / 非结构体
    if err != nil {
        return nil, err
    }
    out := make(map[string]string, t.NumField())
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        if !field.IsExported() {
            continue
        }
        name, _ := ParseTag(field.Tag.Get("json"))
        switch name {
        case "-":
            continue
        case "":
            name = field.Name
        }
        out[name] = field.Type.String() // 唯一的新东西：把类型名记下来
    }
    return out, nil
}
```

**注意**：把 14.5 里重复的前置检查抽成 `structType(v)` 这样的私有辅助函数，是这类「反射工具函数」的标准做法——前置检查只写一遍，各取所需的部分才分开。

验证：

```go
type Account struct {
    ID       int    `json:"id"`
    Name     string `json:"name"`
    Password string `json:"-"`
    Balance  int    `json:"balance,omitempty"`
    note     string
}
got, _ := JSONFieldTypes(Account{})
fmt.Println(got)
// map[balance:int id:int name:string]
```

**为什么这样写更好**：把「字段筛选规则」集中在一处，`-`、空标签、未导出三种情况都用同一套判断处理；返回 map 而不是切片，调用方查找顺手，代价是丢掉了字段顺序（需要顺序时应返回有序切片，或者按 `NumField()` 顺序遍历）。

:::

### 第 3 题

写一个 `ZeroFields(v any) ([]string, error)`，用反射列出结构体里所有**零值**的字段名（含未导出字段），并说明为什么未导出的字段也能出现在结果里。

::: details 第 3 题参考答案

```go
func ZeroFields(v any) ([]string, error) {
    rv := reflect.ValueOf(v)
    for rv.Kind() == reflect.Pointer {
        if rv.IsNil() {
            return nil, fmt.Errorf("ZeroFields: 收到 nil 指针")
        }
        rv = rv.Elem()
    }
    if rv.Kind() != reflect.Struct {
        return nil, fmt.Errorf("ZeroFields: 需要结构体，收到 %s", rv.Kind())
    }

    t := rv.Type()
    var zero []string
    for i := 0; i < t.NumField(); i++ {
        if rv.Field(i).IsZero() {
            zero = append(zero, t.Field(i).Name)
        }
    }
    return zero, nil
}
```

验证：

```go
fmt.Println(ZeroFields(Person{}))          // [Name Age note]
fmt.Println(ZeroFields(&Person{Name: "A"})) // [Age note]
```

**为什么未导出字段也能读**：反射的可见性限制只作用在**写**上——`CanSet()` 需要字段导出，而 `Field(i)` 读取和 `IsZero()` 判断不需要。原因是「读」不会破坏封装的不变量，而「写」会。所以框架可以读取未导出字段做调试输出，但不能替你修改它们。

:::

### 第 4 题

给 `Calculator` 增加一个 `Mul(a, b int) int` 方法，用 `CallByName` 调用它；再写一段代码，把返回的 `[]any` 结果安全地转成 `int`（类型不符时返回错误而不是 panic）。

::: details 第 4 题参考答案

```go
// Calculator.Mul 返回两个整数的乘积。
func (c *Calculator) Mul(a, b int) int {
    return a * b
}
```

调用与转换：

```go
// CallInt 调用只返回一个 int 的方法。
func CallInt(recv any, method string, args ...any) (int, error) {
    results, err := CallByName(recv, method, args...)
    if err != nil {
        return 0, err
    }
    if len(results) != 1 {
        return 0, fmt.Errorf("%s 应该返回 1 个值，实际 %d 个", method, len(results))
    }
    v, ok := results[0].(int)
    if !ok {
        return 0, fmt.Errorf("%s 的返回值是 %T，不是 int", method, results[0])
    }
    return v, nil
}

n, err := CallInt(&Calculator{}, "Mul", 6, 7)
fmt.Println(n, err) // 42 <nil>

// Divide 返回两个值，先被「返回值个数」这道防线拦下
n, err = CallInt(&Calculator{}, "Divide", 6.0, 2.0)
fmt.Println(n, err) // 0 Divide 应该返回 1 个值，实际 2 个

// 个数对了但类型不对时，才会走到类型断言这道防线
type ratio struct{}
func (ratio) Value() float64 { return 1.5 }

n, err = CallInt(ratio{}, "Value")
fmt.Println(n, err) // 0 Value 的返回值是 float64，不是 int
```

**为什么这样写更好**：反射出来的 `any` 必须经过**带 ok 的类型断言**才能安全使用；直接 `results[0].(int)` 在类型不符时会 panic，把「运行时才知道的类型」又变回了一次崩溃风险。用 `CallInt` 收口之后，上层代码恢复成普通的 `(int, error)` 接口。

:::

### 第 5 题

14.9 里「按名字查字段」比「按下标读字段」慢约 20 倍。请给 `FieldByName` 加一层类型缓存（`reflect.Type` → 字段名到下标），再用基准测试说明缓存后接近按下标访问的水平。

::: details 第 5 题参考答案

```go
// fieldIndexCache 缓存「类型 + 字段名 → 下标」，避免每次都查字段表。
var fieldIndexCache sync.Map // map[reflect.Type]map[string]int

func fieldIndex(t reflect.Type, name string) (int, bool) {
    if cached, ok := fieldIndexCache.Load(t); ok {
        idx, found := cached.(map[string]int)[name]
        return idx, found
    }
    indexes := make(map[string]int, t.NumField())
    for i := 0; i < t.NumField(); i++ {
        indexes[t.Field(i).Name] = i
    }
    actual, _ := fieldIndexCache.LoadOrStore(t, indexes)
    idx, found := actual.(map[string]int)[name]
    return idx, found
}

func ageViaCache(v reflect.Value) (int, bool) {
    idx, ok := fieldIndex(v.Type(), "Age")
    if !ok {
        return 0, false
    }
    return int(v.Field(idx).Int()), true
}
```

再仿照 `BenchmarkReflectFieldByName` 写一项 `BenchmarkReflectFieldByNameCached`（循环里改用 `ageViaCache`），预期它会接近 `BenchmarkReflectFieldByIndex`（1.5 ns 量级），远快于按名字直查的 32 ns 量级，代价是首次访问要为每个类型建一次索引表。

**为什么这样写更好**：字段表在同一个类型上是**不变的**，把它缓存起来就是把「每次调用都解析」变成「每个类型解析一次」，这也是 `encoding/json` 快过裸反射实现的主要原因。注意 `sync.Map` 适合「读多写少、key 集合相对稳定」的场景；如果类型很少，用 `sync.RWMutex` + 普通 map 也完全够用。

:::

### 第 6 题

下面四个需求，分别该用接口、泛型还是反射？说明理由。

1. 一批图形（圆、矩形）都要能算面积；
2. 给任意结构体生成日志字段（键值对）；
3. 求任意可比较类型切片的最大值；
4. 把结构体按 `json` 标签序列化成 JSON。

::: details 第 6 题参考答案

| 需求 | 首选 | 理由 |
| --- | --- | --- |
| 1. 图形算面积 | **接口** | 行为固定（`Area() float64`）、实现类型有限，编译期可检查，零运行时开销 |
| 2. 任意结构体生成日志字段 | **反射**（+ 类型缓存） | 类型在运行时才知道，且结构体是外部传入的；缓存字段信息抵消大部分开销 |
| 3. 求可比较切片最大值 | **泛型** | `func Max[T cmp.Ordered](s []T) T` 保留类型安全，实例化后是静态代码，没有反射开销 |
| 4. 按标签序列化 JSON | **反射**（标准库已实现） | 规则由标签在运行时决定；直接用 `encoding/json`，不要自己写 |

**判断顺序**：先问「行为是不是固定的」→ 接口；再问「逻辑相同、类型不同吗」→ 泛型；再问「类型是不是运行时才知道」→ 反射（并加缓存）；最后问「能不能在构建期生成代码」→ 代码生成。

**为什么把反射排在最后**：反射把编译期检查推迟到运行时，错误会以 panic 或「结果不对」的形式出现；接口和泛型都能让编译器帮你挡住大部分错误。只有当类型信息真的只在运行时存在时，反射才是唯一可行的选择。

:::

---

## 小结

- **反射三定律**：接口值可拆成 `Type` + `Value`；`Value.Interface()` 可还原接口值；要修改值，`Value` 必须可寻址（来自指针的 `Elem()`）。
- **`Kind` 与 `Type` 分工明确**：分支处理看 `Kind`（有限、通用），类型判定看 `Type`；`Type` 可比较，可当 map key 做类型缓存。
- **取值要先看 Kind**：`Value.String()` 对非字符串返回 `<int Value>` 这类描述，用错取值方法会 panic；`Interface()` 通用但更慢。
- **改值必须走指针**：`CanAddr` / `CanSet` 是两道闸门，未导出字段永远不可写；工程代码应先检查再动手，把 panic 变成返回 error。
- **标签是运行时字符串**：`Tag.Get` 与 `Tag.Lookup` 语义不同，语法写错会被静默忽略（`go vet` 能查到）。
- **动态调用要自己兜错**：`Call` 只接受 `[]reflect.Value`，参数个数/类型不匹配会 panic，方法返回的 error 也不会自动展开。
- **`DeepEqual` ≠ `==`**：nil 切片和空切片不相等、NaN 不等于自己、含函数字段的结构体不相等；能用 `slices.Equal` / `maps.Equal` 就别用它。
- **性能代价明确**：按名字查字段比直接访问慢约 80 倍，`encoding/json` 靠类型缓存把开销摊薄；热路径上缓存类型信息，或者干脆用接口、泛型、代码生成。
- **`encoding/json` 就是反射的工业级范例**：字段缓存 + 专门的编码器 + 缓冲区复用，外加对循环引用的检测；它要求传指针反序列化，正是第三定律的直接后果。

下一章进入标准库精讲，从 `time`、`math` 与排序开始，看看时间格式化、单调时钟、`math/rand/v2` 和 `slices.SortFunc` 这些日常工具该怎么用对。
