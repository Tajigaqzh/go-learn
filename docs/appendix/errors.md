# 附录 B：常见编译错误与运行时报错

错误信息先看文件和行号，再看最靠前的原因。下面的输出来自 Go 工具链常见行为；地址、临时目录、goroutine 编号和 map 顺序可能随运行变化。

## 编译错误

| 报错 | 通常原因 | 修复方向 |
| --- | --- | --- |
| `undefined: name` | 标识符拼写错误、作用域外使用或缺少导入 | 检查声明位置和包名 |
| `declared and not used: x` | 局部变量声明后未使用 | 删除、使用，或仅在明确需要时使用 `_` |
| `imported and not used: "fmt"` | 导入包没有引用 | 删除导入或补充实际调用 |
| `cannot use x (variable of type int) as int64 value` | Go 不做隐式数值转换 | 显式 `int64(x)`，先确认溢出边界 |
| `multiple-value f() in single-value context` | 多返回值直接放入只接受一个值的位置 | 接收多个返回值或只取一个 |
| `assignment to entry in nil map` | 对 nil map 写入 | 先 `make(map[K]V)` |
| `invalid operation: s == t (slice can only be compared to nil)` | 切片不能互相比较 | 使用 `slices.Equal` 或手写比较 |
| `cannot assign to struct field ... in map` | map 索引得到的是不可寻址副本 | 取出修改后再写回，或 map 存指针 |

### 例：未使用变量

```text
./main.go:8:2: declared and not used: count
```

先定位 `main.go:8`，再判断是遗漏业务逻辑还是应删除变量。不要为了“让编译通过”把所有变量改成 `_`，那会掩盖真正的逻辑遗漏。

### 例：`go vet` 的 Printf 检查

```text
./main.go:9:2: fmt.Printf format %d has arg name of wrong type string
```

这不是运行时 panic，而是 `go vet` 根据格式动词和参数静态推断出的错误。把 `%d` 改为 `%s`，或把参数改成数字类型；不要用 `%v` 盲目掩盖格式契约。

## 运行时 panic

| 报错 | 触发场景 | 排查方式 |
| --- | --- | --- |
| `panic: runtime error: index out of range` | 切片或数组下标越界 | 检查 len，区分字节索引和 rune 索引 |
| `panic: assignment to entry in nil map` | nil map 写入 | 初始化 map，并测试零值路径 |
| `panic: close of closed channel` | 同一 channel 被重复关闭 | 由发送方统一关闭，必要时用 `sync.Once` |
| `fatal error: concurrent map writes` | 并发读写普通 map | 用 mutex、`sync.Map` 或 channel 串行化 |
| `panic: send on closed channel` | 关闭后仍发送 | 重新设计所有权和关闭顺序 |
| `panic: runtime error: invalid memory address or nil pointer dereference` | 解引用 nil 指针或 nil 接口内部指针 | 在边界处校验 nil，区分 typed nil 接口 |

panic 栈中的第一段通常是最值得先看的业务调用；`runtime` 内部帧用于解释机制。生产服务应在合适边界 `recover`，把 panic 转为错误响应，同时保留栈和请求 ID；库函数不要随意 recover 后静默返回。

## 并发诊断

使用：

```bash
go test -race ./...
```

典型报告会包含：

```text
WARNING: DATA RACE
Read at 0x... by goroutine 8:
  example.com/app.TestRace()
Previous write at 0x... by goroutine 7:
  example.com/app.TestRace()
```

它说明至少有一对并发访问没有建立同步关系，不一定说明某一行“必然错误”。沿两条堆栈向上找共享变量的所有权，再用 mutex、atomic 或 channel 建立 happens-before。不要通过增加 `time.Sleep` 让报告暂时消失。

## `go test` 与测试失败

```text
--- FAIL: TestParse (0.00s)
    parse_test.go:27: 期望 "ok"，实际 "OK"
FAIL
```

先看测试名和行号，再确认失败的是输入、期望还是实现。并行测试中不要依赖 map 遍历顺序、全局可变状态或墙上时钟；使用表驱动用例、固定时钟和 `t.TempDir`。

## 错误包装的读法

```text
读取配置: 解析端口 "70000": strconv.Atoi: parsing "70000": invalid syntax
```

从外到内读上下文：哪个操作失败、哪个字段失败、底层具体原因是什么。使用 `%w` 包装，调用方才能用 `errors.Is` / `errors.As` 判断类型；仅用 `%v` 会丢失错误链。

## 诊断顺序

1. 复制完整命令、Go 版本、操作系统和环境变量摘要。
2. 记录第一个报错，不要只截取最后一行。
3. 缩小到最小可运行示例，再验证假设。
4. 用 `go doc`、源码和测试确认 API 契约。
5. 修复后重跑原命令，并补一个能复现问题的测试。
