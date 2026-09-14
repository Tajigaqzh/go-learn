package go08_strings

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// --- 8.6 strings 包常用函数 ---
func section6_StringsFunctions() {
	fmt.Println("\n--- 8.6 strings 包常用函数 ---")

	raw := "  Go, Go, Go! 学习 Go  "
	s := strings.TrimSpace(raw)
	fmt.Printf("原始：%q\n", raw)
	fmt.Printf("TrimSpace → %q\n", s)
	fmt.Printf("Contains(s, \"学习\") = %t\n", strings.Contains(s, "学习"))
	fmt.Printf("Count(s, \"Go\") = %d\n", strings.Count(s, "Go"))
	fmt.Printf("Index(s, \"学习\") = %d（返回字节下标，不是字符下标）\n", strings.Index(s, "学习"))
	fmt.Printf("HasPrefix(s, \"Go\") = %t，HasSuffix(s, \"Go\") = %t\n",
		strings.HasPrefix(s, "Go"), strings.HasSuffix(s, "Go"))
	fmt.Printf("Index(s, \"Rust\") = %d（找不到返回 -1，不是 0）\n", strings.Index(s, "Rust"))

	parts := strings.Split(s, ", ")
	fmt.Printf("Split(s, \", \") = %q（%d 段）\n", parts, len(parts))
	fmt.Printf("Join(parts, \"|\") = %q\n", strings.Join(parts, "|"))
	fmt.Printf("Fields(s) = %q（按任意空白切分，并丢掉空串）\n", strings.Fields(s))
	fmt.Printf("ReplaceAll(s, \"Go\", \"Rust\") = %q\n", strings.ReplaceAll(s, "Go", "Rust"))
	fmt.Printf("ToUpper(s) = %q，ToLower(s) = %q\n", strings.ToUpper(s), strings.ToLower(s))
	fmt.Printf("EqualFold(\"GO\", \"go\") = %t（忽略大小写比较）\n", strings.EqualFold("GO", "go"))

	before, after, found := strings.Cut(s, ",")
	fmt.Printf("Cut(s, \",\") = %q / %q，found = %t\n", before, after, found)
	rest, ok := strings.CutPrefix(s, "Go")
	fmt.Printf("CutPrefix(s, \"Go\") = %q，ok = %t\n", rest, ok)

	invalid := string([]byte{0xE4, 0xB8})
	fmt.Printf("ToValidUTF8(非法序列, \"?\") = %q\n", strings.ToValidUTF8(invalid, "?"))

	fmt.Println("Go 1.24 起提供迭代器版本，可以边切边处理，不额外分配切片：")
	for line := range strings.SplitSeq("第一行\n第二行", "\n") {
		fmt.Printf("  SplitSeq → %q\n", line)
	}
}

// --- 8.7 strconv：字符串与数字互转 ---
func section7_StrconvConversions() {
	fmt.Println("\n--- 8.7 strconv：字符串与数字互转 ---")

	n, err := strconv.Atoi("42")
	fmt.Printf("Atoi(\"42\") = %d，err = %v\n", n, err)
	n, err = strconv.Atoi("12a")
	fmt.Printf("Atoi(\"12a\") = %d，err = %v\n", n, err)
	fmt.Printf("Itoa(42) = %q\n", strconv.Itoa(42))

	hex, err := strconv.ParseInt("ff", 16, 64)
	fmt.Printf("ParseInt(\"ff\", 16, 64) = %d，err = %v\n", hex, err)
	_, err = strconv.ParseInt("99999999999999999999", 10, 64)
	fmt.Printf("ParseInt 超出 int64 范围：err = %v\n", err)
	fmt.Printf("FormatInt(255, 16) = %q，FormatInt(8, 2) = %q\n",
		strconv.FormatInt(255, 16), strconv.FormatInt(8, 2))

	f, err := strconv.ParseFloat("3.14159", 64)
	fmt.Printf("ParseFloat(\"3.14159\", 64) = %v，err = %v\n", f, err)
	fmt.Printf("ParseFloat(\"1e3\", 64) = %v\n", mustFloat("1e3"))
	fmt.Printf("FormatFloat(3.14159, 'f', 2, 64) = %q\n",
		strconv.FormatFloat(3.14159, 'f', 2, 64))

	quoted := strconv.Quote("他说：\"你好\"")
	fmt.Printf("Quote → %s\n", quoted)
	unquoted, err := strconv.Unquote(quoted)
	fmt.Printf("Unquote → %q，err = %v\n", unquoted, err)
	fmt.Printf("QuoteRune('中') = %s，QuoteRune('\\n') = %s（不可见字符会转义）\n",
		strconv.QuoteRune('中'), strconv.QuoteRune('\n'))

	fmt.Println("用法约定：Atoi/Itoa 是 10 进制的快捷方式，其他进制和位宽用 ParseXxx/FormatXxx。")
}

// --- 8.8 拼接策略对比 ---
func section8_ConcatStrategies() {
	fmt.Println("\n--- 8.8 拼接策略对比 ---")

	parts := make([]string, 0, 200)
	for i := 0; i < 200; i++ {
		parts = append(parts, strconv.Itoa(i%10))
	}

	plus := concatWithPlus(parts)
	join := strings.Join(parts, "")
	builder := concatWithBuilder(parts)
	buffer := concatWithBuffer(parts)

	fmt.Printf("拼接 %d 个片段，四种写法的结果长度都是 %d\n", len(parts), len(plus))
	fmt.Printf("  + 运算符      与 Join 一致：%t\n", plus == join)
	fmt.Printf("  strings.Join  与 Builder 一致：%t\n", join == builder)
	fmt.Printf("  Builder       与 Buffer 一致：%t\n", builder == buffer)
	fmt.Println("结果相同、代价不同：循环里用 + 每次都要复制已有内容，整体是 O(n²)。")
	fmt.Println("耗时对比见基准测试：go test -bench=. -benchmem ./internal/chapter/go08_strings/")
}

// --- 8.9 []byte 与 string 互转 ---
func section9_ByteSliceConversion() {
	fmt.Println("\n--- 8.9 []byte 与 string 互转 ---")

	s := "hello 世界"
	b := []byte(s)
	fmt.Printf("s = %q（%d 字节）\n", s, len(s))
	fmt.Printf("[]byte(s) = % X\n", b)

	b[0] = 'H'
	fmt.Printf("修改 b[0] 之后：b = %q，s = %q\n", string(b), s)
	fmt.Println("s 没变，说明 []byte(s) 是一次深拷贝，两者不是同一块缓冲区。")

	fmt.Printf("string(b) 转回来 = %q（同样是一次拷贝）\n", string(b))
	fmt.Printf("bytes.Contains(b, []byte(\"世界\")) = %t（只读判断可以全程走 []byte）\n",
		bytes.Contains(b, []byte("世界")))

	var sb strings.Builder
	sb.Write(b)
	sb.WriteString(s)
	fmt.Printf("Builder.Write([]byte) 直接追加字节：%q\n", sb.String())

	fmt.Println("工程建议：高频转换的路径上，选一种表示走到底；零拷贝转换（unsafe.String）留给第 31 章。")
}

// concatWithPlus 用 + 在循环里累加，是 O(n²) 的典型反例，仅用于对比。
func concatWithPlus(parts []string) string {
	out := ""
	for _, p := range parts {
		out += p
	}
	return out
}

// concatWithBuilder 用 strings.Builder 拼接，先算好总长度再 Grow，避免中途扩容。
func concatWithBuilder(parts []string) string {
	total := 0
	for _, p := range parts {
		total += len(p)
	}
	var sb strings.Builder
	sb.Grow(total)
	for _, p := range parts {
		sb.WriteString(p)
	}
	return sb.String()
}

// concatWithBuffer 用 bytes.Buffer 拼接，Buffer 兼顾字节缓冲和 io.Writer 能力。
func concatWithBuffer(parts []string) string {
	var buf bytes.Buffer
	for _, p := range parts {
		buf.WriteString(p)
	}
	return buf.String()
}

// mustFloat 解析一个约定合法的浮点字面量，解析失败时返回 0 而不是 panic。
func mustFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}
