// Package go16_stdlib_text 演示正则、文本与模板。
package go16_stdlib_text

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"html/template"
	"regexp"
	"strings"
	texttemplate "text/template"
	"unicode"
	"unicode/utf8"
)

// Demo 演示第十六章：标准库精讲（二）：正则、文本与模板。
func Demo() {
	fmt.Println("==========================================")
	fmt.Println("========== go16_stdlib_text: 正则、文本与模板 ==========")
	fmt.Println("==========================================")

	fmt.Println("\n--- 1. regexp 预编译与基本匹配 ---")
	demoRegexpBasics()

	fmt.Println("\n--- 2. Find 系列：查找匹配 ---")
	demoRegexpFind()

	fmt.Println("\n--- 3. Replace 系列：替换匹配 ---")
	demoRegexpReplace()

	fmt.Println("\n--- 4. Split 与分组捕获 ---")
	demoRegexpSplit()

	fmt.Println("\n--- 5. 命名捕获组 ---")
	demoNamedCapture()

	fmt.Println("\n--- 6. 贪婪与非贪婪 ---")
	demoGreedy()

	fmt.Println("\n--- 7. text/template 基础 ---")
	demoTextTemplate()

	fmt.Println("\n--- 8. 模板管道与函数 ---")
	demoTemplatePipeline()

	fmt.Println("\n--- 9. html/template 自动转义 ---")
	demoHTMLTemplate()

	fmt.Println("\n--- 10. unicode 与 utf8 ---")
	demoUnicode()

	fmt.Println("\n--- 11. encoding/csv ---")
	demoCSV()

	fmt.Println("\n--- 12. strings.Replacer 批量替换 ---")
	demoReplacer()

	fmt.Println("\n==========================================")
	fmt.Println("========== 正则、文本与模板演示结束 ==========")
	fmt.Println("==========================================")
}

// demoRegexpBasics 演示正则表达式预编译与基本匹配。
func demoRegexpBasics() {
	// MustCompile 在编译失败时 panic，适合常量模式
	re := regexp.MustCompile(`\d{3}-\d{4}`)

	fmt.Println("匹配电话号尾：")
	fmt.Printf("  \"Call 123-4567\" → %v\n", re.MatchString("Call 123-4567"))
	fmt.Printf("  \"No phone\"     → %v\n", re.MatchString("No phone"))

	// Compile 返回错误，适合动态模式
	pattern := `\d+`
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Printf("编译失败：%v\n", err)
		return
	}
	fmt.Printf("\n动态模式 %q 匹配 \"abc123\" → %v\n", pattern, compiled.MatchString("abc123"))

	// 性能提示：预编译正则并复用，避免每次调用都编译
	fmt.Println("\n性能提示：")
	fmt.Println("• 包级变量存储编译好的正则：var phoneRE = regexp.MustCompile(...)")
	fmt.Println("• regexp.Regexp 是并发安全的，可以多 goroutine 共享")
}

// demoRegexpFind 演示 Find 系列方法。
func demoRegexpFind() {
	re := regexp.MustCompile(`\d+`)
	text := "Order 123 costs $456 in 2024"

	// FindString：返回第一个匹配
	first := re.FindString(text)
	fmt.Printf("第一个数字：%s\n", first)

	// FindAllString：返回所有匹配
	all := re.FindAllString(text, -1) // -1 表示全部
	fmt.Printf("所有数字：%v\n", all)

	// FindAllString 限制数量
	two := re.FindAllString(text, 2)
	fmt.Printf("前两个数字：%v\n", two)

	// FindStringIndex：返回第一个匹配的位置 [start, end)
	idx := re.FindStringIndex(text)
	fmt.Printf("第一个数字的位置：%v → %q\n", idx, text[idx[0]:idx[1]])

	// FindStringSubmatch：捕获分组
	dateRE := regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)
	date := "Today is 2024-09-14"
	matches := dateRE.FindStringSubmatch(date)
	fmt.Printf("\n日期匹配（带分组）：%v\n", matches)
	if len(matches) > 0 {
		fmt.Printf("  完整匹配：%s\n", matches[0])
		fmt.Printf("  年：%s，月：%s，日：%s\n", matches[1], matches[2], matches[3])
	}
}

// demoRegexpReplace 演示替换。
func demoRegexpReplace() {
	re := regexp.MustCompile(`\d+`)
	text := "Price is 100 dollars and 50 cents"

	// ReplaceAllString：替换所有匹配
	replaced := re.ReplaceAllString(text, "XX")
	fmt.Printf("替换所有数字：%s\n", replaced)

	// ReplaceAllStringFunc：用函数决定替换内容
	doubled := re.ReplaceAllStringFunc(text, func(s string) string {
		return "[" + s + "]"
	})
	fmt.Printf("给每个数字加括号：%s\n", doubled)

	// 使用捕获组引用：$1 $2
	phoneRE := regexp.MustCompile(`(\d{3})-(\d{4})`)
	phone := "Call 123-4567 or 890-1234"
	formatted := phoneRE.ReplaceAllString(phone, "($1) $2")
	fmt.Printf("\n格式化电话：%s\n", formatted)
}

// demoRegexpSplit 演示 Split。
func demoRegexpSplit() {
	re := regexp.MustCompile(`\s+`) // 一个或多个空白
	text := "Go   is    awesome"

	parts := re.Split(text, -1)
	fmt.Printf("按空白切分：%v\n", parts)

	// 限制切分次数
	two := re.Split(text, 2)
	fmt.Printf("最多切 2 份：%v\n", two)
}

// demoNamedCapture 演示命名捕获组。
func demoNamedCapture() {
	// ?P<name> 定义命名捕获组
	re := regexp.MustCompile(`(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`)
	text := "Release date: 2024-09-14"

	match := re.FindStringSubmatch(text)
	if match == nil {
		fmt.Println("未匹配")
		return
	}

	// SubexpNames 返回捕获组名称（第一个是空字符串，对应完整匹配）
	names := re.SubexpNames()
	fmt.Println("命名捕获组：")
	for i, name := range names {
		if i == 0 || name == "" {
			continue
		}
		fmt.Printf("  %s = %s\n", name, match[i])
	}

	// 构建 map
	result := make(map[string]string)
	for i, name := range names {
		if i > 0 && i < len(match) {
			result[name] = match[i]
		}
	}
	fmt.Printf("结果 map：%v\n", result)
}

// demoGreedy 演示贪婪与非贪婪。
func demoGreedy() {
	text := `<div>Hello</div><div>World</div>`

	// 贪婪：.*? 匹配尽可能少的字符
	greedy := regexp.MustCompile(`<div>.*</div>`)
	fmt.Printf("贪婪模式：%s\n", greedy.FindString(text))

	// 非贪婪：.*? 匹配尽可能少的字符
	nonGreedy := regexp.MustCompile(`<div>.*?</div>`)
	all := nonGreedy.FindAllString(text, -1)
	fmt.Printf("非贪婪模式：%v\n", all)

	fmt.Println("\n规则：")
	fmt.Println("• * 贪婪：尽可能多")
	fmt.Println("• *? 非贪婪：尽可能少")
	fmt.Println("• + 贪婪，+? 非贪婪")
	fmt.Println("• ? 贪婪（0 或 1），?? 非贪婪")
}

// demoTextTemplate 演示 text/template 基础。
func demoTextTemplate() {
	// 定义模板
	tmplStr := `Hello, {{.Name}}!
Age: {{.Age}}
Active: {{.Active}}`

	tmpl, err := texttemplate.New("user").Parse(tmplStr)
	if err != nil {
		fmt.Printf("解析模板失败：%v\n", err)
		return
	}

	// 执行模板
	data := struct {
		Name   string
		Age    int
		Active bool
	}{
		Name:   "Alice",
		Age:    30,
		Active: true,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		fmt.Printf("执行模板失败：%v\n", err)
		return
	}

	fmt.Println("text/template 输出：")
	fmt.Println(buf.String())

	// 访问嵌套字段
	tmpl2, _ := texttemplate.New("nested").Parse("User: {{.User.Name}}, Score: {{.User.Score}}")
	data2 := struct {
		User struct {
			Name  string
			Score int
		}
	}{
		User: struct {
			Name  string
			Score int
		}{Name: "Bob", Score: 95},
	}
	buf.Reset()
	tmpl2.Execute(&buf, data2)
	fmt.Printf("\n嵌套字段：%s\n", buf.String())
}

// demoTemplatePipeline 演示模板管道与函数。
func demoTemplatePipeline() {
	// 内置函数：and, or, not, len, index, printf 等
	tmplStr := `{{if .Active}}User is active{{else}}User is inactive{{end}}
Length of name: {{len .Name}}
Upper: {{.Name | printf "%q"}}`

	tmpl, _ := texttemplate.New("pipe").Parse(tmplStr)
	data := struct {
		Name   string
		Active bool
	}{Name: "Charlie", Active: true}

	var buf bytes.Buffer
	tmpl.Execute(&buf, data)
	fmt.Println("管道与内置函数：")
	fmt.Println(buf.String())

	// 自定义函数
	funcs := texttemplate.FuncMap{
		"upper": strings.ToUpper,
		"add":   func(a, b int) int { return a + b },
	}
	tmpl2, _ := texttemplate.New("custom").Funcs(funcs).Parse(`Name: {{.Name | upper}}
Score: {{add .Score 10}}`)
	data2 := struct {
		Name  string
		Score int
	}{Name: "dave", Score: 80}

	buf.Reset()
	tmpl2.Execute(&buf, data2)
	fmt.Println("\n自定义函数：")
	fmt.Println(buf.String())

	// range 遍历
	tmpl3, _ := texttemplate.New("range").Parse(`{{range .Items}}- {{.}}
{{end}}`)
	data3 := struct{ Items []string }{Items: []string{"A", "B", "C"}}
	buf.Reset()
	tmpl3.Execute(&buf, data3)
	fmt.Println("\nrange 遍历：")
	fmt.Print(buf.String())
}

// demoHTMLTemplate 演示 html/template 自动转义。
func demoHTMLTemplate() {
	// html/template 会自动转义 HTML 特殊字符
	tmplStr := `<div>{{.Content}}</div>`

	tmpl, _ := template.New("html").Parse(tmplStr)
	data := struct{ Content string }{Content: "<script>alert('XSS')</script>"}

	var buf bytes.Buffer
	tmpl.Execute(&buf, data)
	fmt.Println("html/template 自动转义：")
	fmt.Println(buf.String())

	// 对比 text/template（不转义）
	tmplText, _ := texttemplate.New("text").Parse(tmplStr)
	buf.Reset()
	tmplText.Execute(&buf, data)
	fmt.Println("\ntext/template（不转义，危险！）：")
	fmt.Println(buf.String())

	fmt.Println("\n安全提示：")
	fmt.Println("• 生成 HTML 时必须用 html/template")
	fmt.Println("• html/template 会根据上下文（HTML、JS、CSS、URL）自动选择转义方式")
	fmt.Println("• 不要用 text/template 生成 HTML，否则有 XSS 风险")
}

// demoUnicode 演示 unicode 与 utf8。
func demoUnicode() {
	s := "Go语言"

	// utf8 包：按字节操作
	fmt.Printf("字符串：%q\n", s)
	fmt.Printf("len(s) = %d（字节数）\n", len(s))
	fmt.Printf("utf8.RuneCountInString(s) = %d（字符数）\n", utf8.RuneCountInString(s))

	// 遍历 rune
	fmt.Println("\n遍历 rune：")
	for i, r := range s {
		fmt.Printf("  [%d] %c (U+%04X)\n", i, r, r)
	}

	// unicode 包：判断字符类别
	fmt.Println("\n字符分类：")
	testChars := []rune{'A', '中', '5', ' ', '!'}
	for _, r := range testChars {
		fmt.Printf("  %c: 字母=%v 数字=%v 空格=%v 汉字=%v\n",
			r,
			unicode.IsLetter(r),
			unicode.IsDigit(r),
			unicode.IsSpace(r),
			unicode.Is(unicode.Han, r),
		)
	}

	// utf8.DecodeRuneInString：手动解码第一个字符
	r, size := utf8.DecodeRuneInString(s)
	fmt.Printf("\n第一个字符：%c，占 %d 字节\n", r, size)
}

// demoCSV 演示 encoding/csv。
func demoCSV() {
	// 读取 CSV
	input := `Name,Age,City
Alice,30,NYC
Bob,25,LA
Carol,35,SF`

	r := csv.NewReader(strings.NewReader(input))
	records, err := r.ReadAll()
	if err != nil {
		fmt.Printf("读取 CSV 失败：%v\n", err)
		return
	}

	fmt.Println("读取 CSV：")
	for i, record := range records {
		fmt.Printf("  [%d] %v\n", i, record)
	}

	// 写入 CSV
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write([]string{"Name", "Score"})
	w.Write([]string{"Dave", "95"})
	w.Write([]string{"Eve", "88"})
	w.Flush()

	if err := w.Error(); err != nil {
		fmt.Printf("写入 CSV 失败：%v\n", err)
		return
	}

	fmt.Println("\n写入 CSV：")
	fmt.Println(buf.String())

	// 自定义分隔符
	tsvInput := "Name\tAge\nFrank\t40"
	r2 := csv.NewReader(strings.NewReader(tsvInput))
	r2.Comma = '\t' // 使用 Tab 分隔
	tsvRecords, _ := r2.ReadAll()
	fmt.Println("读取 TSV（Tab 分隔）：")
	for _, rec := range tsvRecords {
		fmt.Printf("  %v\n", rec)
	}
}

// demoReplacer 演示 strings.Replacer 批量替换。
func demoReplacer() {
	// Replacer 一次性执行多个替换
	replacer := strings.NewReplacer(
		"apple", "🍎",
		"banana", "🍌",
		"orange", "🍊",
	)

	text := "I like apple and banana, but not orange"
	result := replacer.Replace(text)
	fmt.Printf("批量替换：%s\n", result)

	// 与多次 strings.Replace 的对比
	text2 := "Go is fast, Go is simple, Go is powerful"
	replacer2 := strings.NewReplacer("Go", "Golang")
	result2 := replacer2.Replace(text2)
	fmt.Printf("\n单一替换（多次出现）：%s\n", result2)

	fmt.Println("\n性能提示：")
	fmt.Println("• Replacer 内部用 Trie 树，一次遍历完成所有替换")
	fmt.Println("• 比多次 strings.Replace 更高效")
	fmt.Println("• Replacer 是并发安全的，可以复用")
}
