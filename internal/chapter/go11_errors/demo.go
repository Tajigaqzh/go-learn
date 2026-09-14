// Package go11_errors 演示 Go 的错误处理：error 接口、错误包装与错误链、
// errors.Is / As / Join、自定义错误类型、错误与日志的分工，以及 panic/recover 的边界。
//
// 运行方式：
//
//	go run ./cmd/go-learn
//
// 只想验证这一章：
//
//	go test ./internal/chapter/go11_errors/
package go11_errors

import (
	"errors"
	"fmt"
	"strconv"
)

// Demo 运行第 11 章的所有示例。
func Demo() {
	fmt.Println("========== go11_errors: 错误处理 ==========")

	fmt.Println("\n--- 1. error 是一个接口 ---")
	demoErrorInterface()

	fmt.Println("\n--- 2. 创建错误的三种方式 ---")
	demoCreateErrors()

	fmt.Println("\n--- 3. 错误是返回值，必须立即检查 ---")
	demoCheckImmediately()

	fmt.Println("\n--- 4. 包装错误：%w 建立错误链 ---")
	demoWrapping()

	fmt.Println("\n--- 5. errors.Is：沿错误链判断「是不是它」 ---")
	demoErrorsIs()

	fmt.Println("\n--- 6. errors.As：从错误链里取回具体类型 ---")
	demoErrorsAs()

	fmt.Println("\n--- 7. errors.Join：把多个错误合成一个 ---")
	demoErrorsJoin()

	fmt.Println("\n--- 8. 自定义错误类型的设计 ---")
	demoCustomErrorType()

	fmt.Println("\n--- 9. 错误信息与日志的分工 ---")
	demoMessageAndLogging()

	fmt.Println("\n--- 10. defer 与 Close 的错误 ---")
	demoDeferAndClose()

	fmt.Println("\n--- 11. panic 与 recover 的边界 ---")
	demoPanicRecover()

	fmt.Println("\n========== 错误处理演示结束 ==========")
}

// demoErrorInterface 演示 error 的本质：只有一个方法的接口。
func demoErrorInterface() {
	// error 的定义就是 type error interface { Error() string }。
	var empty error
	fmt.Printf("未赋值的 error = %v，err == nil → %t\n", empty, empty == nil)

	builtin := errors.New("something failed")
	fmt.Printf("errors.New 返回的底层类型是 %T\n", builtin)
	fmt.Printf("打印时自动调用 Error()：%%v = %v，%%s = %s\n", builtin, builtin)

	var custom error = &ValidationError{Field: "age", Msg: "out of range"}
	fmt.Printf("自定义类型也能当 error 用：%v（底层类型 %T）\n", custom, custom)
	fmt.Println("所以：任何实现了 Error() string 的类型都是 error，error 只是一个约定。")
}

// demoCreateErrors 对比「包级哨兵 / 带上下文包装 / 就地新建」三种写法。
func demoCreateErrors() {
	sentinel := ErrNotFound                             // 包级哨兵：调用方能用 errors.Is 认出它
	wrapped := fmt.Errorf("load user 42: %w", sentinel) // 包装：补上下文，保留身份
	onTheFly := errors.New("cache miss")                // 就地新建：调用方无法识别

	fmt.Printf("包级哨兵 ErrNotFound        → %v\n", sentinel)
	fmt.Printf("fmt.Errorf + %%w 包装        → %v\n", wrapped)
	fmt.Printf("就地 errors.New             → %v\n", onTheFly)
	fmt.Printf("errors.Is(wrapped, ErrNotFound)   → %t（包装不丢身份）\n", errors.Is(wrapped, ErrNotFound))
	fmt.Printf("errors.Is(onTheFly, ErrNotFound)  → %t（就地新建的错误没有身份）\n", errors.Is(onTheFly, ErrNotFound))
}

// demoCheckImmediately 演示「错误必须在调用点处理或上抛」。
func demoCheckImmediately() {
	if n, err := strconv.Atoi("42"); err == nil {
		fmt.Printf("Atoi(\"42\") = %d，err = %v\n", n, err)
	}

	// 反例：忽略 err 之后继续用返回值，程序不会报错，但结果是错的。
	n, _ := strconv.Atoi("abc")
	fmt.Printf("忽略 err：Atoi(\"abc\") 返回 n = %d，继续算 n*2 = %d（结果没有意义）\n", n, n*2)

	if _, err := FindUser(0); err != nil {
		fmt.Printf("上抛时补上下文：%v\n", err)
	}

	fmt.Println("规则：每个 err 要么在这一层处理掉，要么包装后上抛，不能既不处理也不上抛。")
}

// demoWrapping 对比 %v 与 %w 的差别，以及多层错误链。
func demoWrapping() {
	textOnly := fmt.Errorf("divide 10 by 0: %v", ErrDivideByZero) // %v：只拼文本，断链
	linked := fmt.Errorf("divide 10 by 0: %w", ErrDivideByZero)   // %w：保留错误链

	fmt.Printf("%%v 包装：%v，Unwrap = %v\n", textOnly, errors.Unwrap(textOnly))
	fmt.Printf("%%w 包装：%v，Unwrap = %v\n", linked, errors.Unwrap(linked))
	fmt.Printf("两者文本完全相同：%t\n", textOnly.Error() == linked.Error())
	fmt.Printf("errors.Is(textOnly, ErrDivideByZero) = %t，errors.Is(linked, ErrDivideByZero) = %t\n",
		errors.Is(textOnly, ErrDivideByZero), errors.Is(linked, ErrDivideByZero))

	layer1 := fmt.Errorf("read config: %w", linked)
	layer2 := fmt.Errorf("start server: %w", layer1)
	fmt.Printf("两层包装后的完整信息：%v\n", layer2)
	fmt.Printf("errors.Is 会自己沿链往下找：%t\n", errors.Is(layer2, ErrDivideByZero))
}

// demoErrorsIs 演示哨兵错误的正确判断方式。
func demoErrorsIs() {
	_, err := FindUser(0)
	fmt.Printf("FindUser(0) 的错误：%v\n", err)
	fmt.Printf("err == ErrNotFound             → %t（== 只看最外层）\n", err == ErrNotFound)
	fmt.Printf("errors.Is(err, ErrNotFound)    → %t（沿错误链匹配）\n", errors.Is(err, ErrNotFound))
	fmt.Printf("errors.Is(err, ErrDivideByZero) → %t（无关的哨兵不匹配）\n", errors.Is(err, ErrDivideByZero))
	fmt.Println("判断错误永远不要匹配文案：strings.Contains(err.Error(), \"not found\") 会随措辞变化而失效。")
}

// demoErrorsAs 演示从错误链里取回结构化的错误类型。
func demoErrorsAs() {
	err := fmt.Errorf("validate user: %w", &ValidationError{Field: "age", Msg: "out of range"})

	if _, ok := err.(*ValidationError); !ok {
		fmt.Println("直接类型断言 err.(*ValidationError) 失败：包装后的类型是 *fmt.wrapError")
	}

	var ve *ValidationError
	if errors.As(err, &ve) {
		fmt.Printf("errors.As 取回字段：Field=%q，Msg=%q（完整信息：%v）\n", ve.Field, ve.Msg, ve)
	}
	fmt.Println("errors.As 需要传「指向目标类型的指针」，它会自己沿链查找并赋值。")
}

// demoErrorsJoin 演示把多个错误合并成一个，并遍历其中的子错误。
func demoErrorsJoin() {
	err := ValidateUser("", 200)
	fmt.Println("ValidateUser(\"\", 200) 的错误（多行就是 Join 的默认格式）：")
	fmt.Printf("%v\n", err)

	// Join 返回的错误实现了 Unwrap() []error，可以逐个取出子错误。
	var multi interface{ Unwrap() []error }
	if errors.As(err, &multi) {
		for i, sub := range multi.Unwrap() {
			fmt.Printf("  子错误 %d：%v\n", i+1, sub)
		}
	}

	var ve *ValidationError
	if errors.As(err, &ve) {
		fmt.Printf("errors.As 拿到的是第一个 *ValidationError：Field=%q\n", ve.Field)
	}
	fmt.Printf("errors.Join() = %v，== nil → %t\n", errors.Join(), errors.Join() == nil)
	fmt.Printf("errors.Join(nil, nil) == nil → %t（nil 会被忽略）\n", errors.Join(nil, nil) == nil)
}

// demoCustomErrorType 演示自定义错误类型的三个组成部分。
func demoCustomErrorType() {
	err := queryUser(0)
	fmt.Printf("queryUser(0) = %v\n", err)
	fmt.Printf("底层类型：%T\n", err)
	fmt.Printf("实现了 Unwrap，链能继续往下走：errors.Is(err, ErrNotFound) = %t\n",
		errors.Is(err, ErrNotFound))

	var qe *QueryError
	if errors.As(err, &qe) {
		fmt.Printf("errors.As 取回：Query=%q，底层 Err=%v\n", qe.Query, qe.Err)
	}
	fmt.Println("设计分工：Error() 说明发生了什么，字段给出结构化细节，Unwrap() 保留属于哪一类错误。")
}

// demoMessageAndLogging 演示错误信息的书写规范和「只在最外层记录一次」。
func demoMessageAndLogging() {
	bad := errors.New("Error: Failed to open config file.")
	good := errors.New("open config file: permission denied")
	fmt.Printf("不推荐的写法：%q\n", bad)
	fmt.Printf("推荐的写法：  %q\n", good)

	inner := errors.New("permission denied")
	layer1 := fmt.Errorf("open /etc/app.conf: %w", inner)
	layer2 := fmt.Errorf("load config: %w", layer1)
	layer3 := fmt.Errorf("start server: %w", layer2)
	fmt.Printf("最外层看到的完整信息：%v\n", layer3)
	fmt.Printf("底层哨兵仍然可辨认：errors.Is(layer3, inner) = %t\n", errors.Is(layer3, inner))
	fmt.Println("每层只补一段上下文，日志在最外层记录一次；每层都 log 会让一次故障变成三条日志。")
}

// demoDeferAndClose 演示用命名返回值 + defer 收集 Close/Flush 的错误。
func demoDeferAndClose() {
	fmt.Printf("Save(\"data\", 刷新成功) = %v\n", Save("data", false))
	fmt.Printf("Save(\"\", 刷新成功)     = %v\n", Save("", false))
	fmt.Printf("Save(\"data\", 刷新失败) = %v\n", Save("data", true))
	fmt.Printf("errors.Is(Save(\"data\", 刷新失败), ErrFlushFailed) = %t\n",
		errors.Is(Save("data", true), ErrFlushFailed))
	fmt.Println("要点：只有命名返回值，defer 才能在函数 return 之后再修改最终返回的错误。")
}

// demoPanicRecover 演示 panic 的边界和 recover 的正确用法。
func demoPanicRecover() {
	quotient, err := Divide(10, 2)
	fmt.Printf("Divide(10, 2) → %v，err = %v\n", quotient, err)
	_, err = Divide(10, 0)
	fmt.Printf("Divide(10, 0) → err = %v\n", err)

	fmt.Printf("在非 defer 语句里调用 recover() = %v（拦不到任何 panic）\n", recover())

	result, err := SafeDivide(10, 0)
	fmt.Printf("SafeDivide(10, 0) → result=%d，err=%v\n", result, err)
	result, err = SafeDivide(10, 2)
	fmt.Printf("SafeDivide(10, 2) → result=%d，err=%v\n", result, err)

	fmt.Println("边界：recover 只能拦住同一个 goroutine 里、由 defer 调用的那次；")
	fmt.Println("库代码优先返回 error，只有「调用方用错了 API」这类程序 bug 才值得 panic。")
}
