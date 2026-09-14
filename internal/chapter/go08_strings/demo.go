// Package go08_strings 演示 Go 的字符串、字节与 Unicode。
//
// 涵盖主题：
//   - string 是只读的字节序列，len 数的是字节而不是字符
//   - UTF-8 编码、byte 与 rune 的区别
//   - range 遍历字符串按 rune 迭代，下标却是字节偏移
//   - 用下标切分中文字符串的坑，以及按字符边界安全截断
//   - unicode/utf8 包的常用函数
//
// 运行方式：
//
//	go run ./cmd/go-learn
//
// 只想验证这一章：
//
//	go test ./internal/chapter/go08_strings/
package go08_strings

import (
	"fmt"
	"unicode/utf8"
)

// Demo 是第 8 章的入口函数，按小节顺序演示字符串、字节与 Unicode。
func Demo() {
	fmt.Println("========================================")
	fmt.Printf("========== go%02d_%s: %s ==========\n", Chapter, "strings", ChapterTitle)
	fmt.Println("========================================")

	section1_StringIsBytes()
	section2_UTF8AndRune()
	section3_RangeOverString()
	section4_SliceByBytesPitfall()
	section5_UTF8Package()
	section6_StringsFunctions()
	section7_StrconvConversions()
	section8_ConcatStrategies()
	section9_ByteSliceConversion()

	fmt.Println("========================================")
	fmt.Printf("========== %s演示结束 ==========\n", ChapterTitle)
	fmt.Println()
}

// --- 8.1 string 是只读的字节序列 ---
func section1_StringIsBytes() {
	fmt.Println("\n--- 8.1 string 是只读的字节序列 ---")

	s := "Go 学习"
	fmt.Printf("s = %q\n", s)
	fmt.Printf("len(s) = %d（字节数，不是字符数）\n", len(s))
	fmt.Printf("s[0] = %d，%%c 打印是 %c，%%q 打印是 %q\n", s[0], s[0], s[0])
	fmt.Printf("s[0] 的类型是 %T（byte 是 uint8 的别名）\n", s[0])

	fmt.Print("s 的每个字节：")
	for i := 0; i < len(s); i++ {
		fmt.Printf("%02X ", s[i])
	}
	fmt.Println()

	// 字符串不可变：下面这行无法编译，要改内容必须先转成 []byte。
	// s[0] = 'g' // 编译错误：cannot assign to s[0]

	fmt.Printf("s == \"Go 学习\" → %t（字符串可以直接比较）\n", s == "Go 学习")
	fmt.Printf("\"Z\" < \"a\" → %t（按字节值比较，'Z'=0x5A < 'a'=0x61）\n", "Z" < "a")
	fmt.Printf("\"go\" < \"学习\" → %t（UTF-8 字节序与码点序一致）\n", "go" < "学习")

	fmt.Println("结论：string 只承诺「一串字节」，文本含义要靠编码约定来解读。")
}

// --- 8.2 UTF-8 与 rune ---
func section2_UTF8AndRune() {
	fmt.Println("\n--- 8.2 UTF-8 与 rune ---")

	cn := '中'     // rune 字面量，类型是 int32
	rocket := '🚀' // 4 字节的 emoji

	fmt.Printf("'中' 的码点 = %U（十进制 %d），utf8.RuneLen = %d 字节\n", cn, cn, utf8.RuneLen(cn))
	fmt.Printf("'🚀' 的码点 = %U（十进制 %d），utf8.RuneLen = %d 字节\n", rocket, rocket, utf8.RuneLen(rocket))
	fmt.Printf("utf8.UTFMax = %d（一个字符最多用这么多字节）\n", utf8.UTFMax)
	fmt.Printf("byte 是 %T（1 字节，取值 0-255），rune 是 %T（4 字节，表示一个码点）\n",
		byte(0), rune(0))
	fmt.Printf("len(\"中\") = %d，utf8.RuneCountInString(\"中\") = %d\n",
		len("中"), utf8.RuneCountInString("中"))

	fmt.Println("UTF-8 按码点大小分档，宽度从 1 到 4 字节不等：")
	for _, r := range []rune{'A', '中', '€', '🚀'} {
		fmt.Printf("  %c 占 %d 字节\n", r, utf8.RuneLen(r))
	}

	fmt.Println("UTF-8 自带同步信息：首字节的高位标记了这个字符的字节数，所以能边扫边解码。")
}

// --- 8.3 range 遍历字符串按 rune 迭代 ---
func section3_RangeOverString() {
	fmt.Println("\n--- 8.3 range 遍历字符串按 rune 迭代 ---")

	s := "Go 学习"
	fmt.Printf("s = %q，len(s) = %d 字节\n", s, len(s))
	fmt.Println("range 遍历：下标是字节偏移，值是 rune")
	for i, r := range s {
		fmt.Printf("  字节偏移 %2d → %q（%U，宽 %d 字节）\n", i, r, r, utf8.RuneLen(r))
	}

	fmt.Println("按字节遍历：拿到的是 byte，中文会被拆成断片")
	for i := 0; i < len(s); i++ {
		fmt.Printf("  s[%d] = 0x%02X\n", i, s[i])
	}

	runeCount := 0
	for range s {
		runeCount++
	}
	fmt.Printf("range 数出来的字符数 = %d，len 数出来的字节数 = %d\n", runeCount, len(s))
}

// --- 8.4 中文下标切片的坑 ---
func section4_SliceByBytesPitfall() {
	fmt.Println("\n--- 8.4 中文下标切片的坑 ---")

	s := "Go 学习笔记"
	fmt.Printf("s = %q，len(s) = %d 字节\n", s, len(s))

	good := s[3:6]   // 「学」正好占 3 个字节
	broken := s[4:6] // 从「学」的中间切开
	fmt.Printf("s[3:6] = %q，utf8.ValidString = %t\n", good, utf8.ValidString(good))
	fmt.Printf("s[4:6] = %q，字节为 % X，utf8.ValidString = %t\n",
		broken, []byte(broken), utf8.ValidString(broken))
	fmt.Println("被切掉后半截的字符不再合法，打印时会转义成 \\x.. ，终端里通常显示为乱码方块。")

	runes := []rune(s)
	fmt.Printf("[]rune(s) 长度 = %d，前 4 个字符 = %q\n", len(runes), string(runes[:4]))

	fmt.Printf("TruncateBytes(s, 4) = %q（切点回退到字符边界）\n", TruncateBytes(s, 4))
	fmt.Printf("TruncateBytes(s, 6) = %q\n", TruncateBytes(s, 6))
	fmt.Printf("TruncateRunes(s, 5) = %q（按字符数截断）\n", TruncateRunes(s, 5))
	fmt.Printf("ReverseRunes(s) = %q\n", ReverseRunes(s))

	fmt.Println("按字节切分的规则：切点必须落在字符的首字节上，否则得到的是无效 UTF-8。")
}

// --- 8.5 unicode/utf8 包 ---
func section5_UTF8Package() {
	fmt.Println("\n--- 8.5 unicode/utf8 常用函数 ---")

	s := "中文 abc 🚀"
	fmt.Printf("s = %q\n", s)
	fmt.Printf("len(s) = %d 字节，utf8.RuneCountInString(s) = %d 字符\n",
		len(s), utf8.RuneCountInString(s))
	fmt.Printf("utf8.ValidString(s) = %t\n", utf8.ValidString(s))

	// 「中」的 UTF-8 是 E4 B8 AD，这里只保留前两个字节，人为制造一段非法序列。
	truncated := string([]byte{0xE4, 0xB8})
	fmt.Printf("截断的字节序列 % X：ValidString = %t\n", []byte(truncated), utf8.ValidString(truncated))

	fmt.Print("range 遍历非法序列，坏字节逐个变成 U+FFFD：")
	for i, r := range truncated {
		fmt.Printf("[%d]=%U ", i, r)
	}
	fmt.Println()
	fmt.Printf("U+FFFD 自己编码要 %d 字节，但解码失败时只前进 1 字节\n", utf8.RuneLen(utf8.RuneError))
	fmt.Printf("utf8.RuneError = %U，utf8.ValidRune(utf8.RuneError) = %t\n",
		utf8.RuneError, utf8.ValidRune(utf8.RuneError))

	r, size := utf8.DecodeRuneInString("学习")
	fmt.Printf("DecodeRuneInString(\"学习\") = %c（%U），宽度 %d\n", r, r, size)
	r, size = utf8.DecodeRuneInString(truncated)
	fmt.Printf("解码非法序列 = %c（%U），宽度 %d\n", r, r, size)

	word := "学"
	fmt.Printf("utf8.RuneStart(%q 的首字节 0x%02X) = %t\n",
		word, word[0], utf8.RuneStart(word[0]))

	buf := make([]byte, 0, utf8.UTFMax)
	buf = utf8.AppendRune(buf, '🚀')
	fmt.Printf("utf8.AppendRune(nil, '🚀') = % X\n", buf)
}

// TruncateBytes 把 s 截断到不超过 maxBytes 个字节，并保证结果仍是合法 UTF-8。
//
// 直接从中间切断多字节字符会得到无效字节序列，所以这里用 utf8.RuneStart
// 从切点向前回退，直到落在某个字符的首字节上。
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

// ReverseRunes 按字符（rune）反转字符串，而不是按字节反转。
//
// 按字节反转会把多字节字符的字节顺序颠倒，得到一段无效 UTF-8。
func ReverseRunes(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// TruncateRunes 把 s 截断到最多 maxRunes 个字符。
//
// for i := range s 只取每个字符的起始字节下标，因此整个过程不产生分配，
// 比起 []rune(s)[:maxRunes] 更省内存。
func TruncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	count := 0
	for i := range s {
		if count == maxRunes {
			return s[:i]
		}
		count++
	}
	return s
}
