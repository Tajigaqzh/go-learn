package go18_serde_config

import (
	"bytes"
	"encoding/csv"
	"encoding/gob"
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Book 是 xml 与 gob 示例共用的结构体。
type Book struct {
	XMLName xml.Name `xml:"book"`
	Title   string   `xml:"title"`
	Author  string   `xml:"author"`
	Price   float64  `xml:"price"`
}

// --- 18.6 encoding/xml、encoding/csv、encoding/gob ---
func demoOtherFormats() {
	book := Book{Title: "Go 语言", Author: "Ada", Price: 42.5}

	xmlData, err := xml.MarshalIndent(book, "", "  ")
	fmt.Printf("xml.MarshalIndent：\n%s\n（err = %v）\n", xmlData, err)
	var fromXML Book
	if err := xml.Unmarshal(xmlData, &fromXML); err != nil {
		fmt.Println("xml.Unmarshal 失败：", err)
	}
	fmt.Printf("xml 反序列化：%+v\n", fromXML)

	var csvBuf bytes.Buffer
	writer := csv.NewWriter(&csvBuf)
	for _, row := range [][]string{{"id", "name"}, {"1", "Ada"}, {"2", "Bob"}} {
		if err := writer.Write(row); err != nil {
			fmt.Println("写 CSV 失败：", err)
		}
	}
	writer.Flush()
	fmt.Printf("csv.Writer 输出：\n%s", csvBuf.String())

	records, err := csv.NewReader(strings.NewReader(csvBuf.String())).ReadAll()
	fmt.Printf("csv.Reader 读回 %d 行：%v，err = %v\n", len(records), records, err)

	broken := csv.NewReader(strings.NewReader("a,b\n1,2,3\n"))
	_, err = broken.ReadAll()
	fmt.Printf("字段数不一致 → err = %v\n", err)

	var gobBuf bytes.Buffer
	if err := gob.NewEncoder(&gobBuf).Encode(book); err != nil {
		fmt.Println("gob 编码失败：", err)
	}
	fmt.Printf("gob 编码后 %d 字节（同一份数据 JSON 是 %d 字节）\n",
		gobBuf.Len(), len(mustJSON(Book{Title: "Go 语言", Author: "Ada", Price: 42.5})))

	var fromGob Book
	if err := gob.NewDecoder(bytes.NewReader(gobBuf.Bytes())).Decode(&fromGob); err != nil {
		fmt.Println("gob 解码失败：", err)
	}
	fmt.Printf("gob 解码：%+v\n", fromGob)
	fmt.Println("选型：对外接口用 JSON，配置文件用 YAML/TOML，Go 服务之间才考虑 gob（它带类型信息，不可跨语言）。")
}

// --- 18.7 YAML / TOML：标准库没有，手写一个简单解析器 ---
func demoSimpleYAML() {
	const src = `# 应用配置
app:
  name: "go-learn"   # 行尾注释
  debug: true
  port: 8080
database:
  host: 127.0.0.1
  max_open: 20
`

	parsed, err := ParseSimpleYAML(src)
	if err != nil {
		fmt.Println("解析失败：", err)
		return
	}
	fmt.Printf("解析结果（key 已排序，方便对照）：\n%s", renderConfig(parsed, ""))

	_, err = ParseSimpleYAML("app:\n  name go-learn\n")
	fmt.Printf("缺少冒号 → err = %v\n", err)

	value, _ := parsed["app"].(map[string]any)
	fmt.Printf("取值：app.name = %v（%T），app.port = %v（%T）\n",
		value["name"], value["name"], value["port"], value["port"])
	fmt.Println("标准库不含 YAML/TOML：生产项目用 gopkg.in/yaml.v3 或 BurntSushi/toml，")
	fmt.Println("这里手写解析器是为了演示「解析配置要处理注释、缩进、类型推断」这几件事。")
}

// ParseSimpleYAML 解析一个极简 YAML 子集：注释、空行、两级缩进、key: value。
//
// 支持标量类型推断（bool / int / float / 带引号的字符串），不支持列表、
// 多行字符串、锚点等真正的 YAML 特性——需要这些能力时请使用成熟的第三方库。
func ParseSimpleYAML(src string) (map[string]any, error) {
	root := map[string]any{}
	stack := []map[string]any{root}
	indents := []int{-1}

	for i, rawLine := range strings.Split(src, "\n") {
		lineNo := i + 1
		line := stripComment(rawLine)
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		key, rawValue, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			return nil, fmt.Errorf("第 %d 行缺少冒号：%q", lineNo, strings.TrimSpace(line))
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("第 %d 行的 key 为空", lineNo)
		}
		rawValue = strings.TrimSpace(rawValue)

		for len(stack) > 1 && indent <= indents[len(indents)-1] {
			stack = stack[:len(stack)-1]
			indents = indents[:len(indents)-1]
		}
		parent := stack[len(stack)-1]

		if rawValue == "" { // key 后面没有值：进入下一层
			child := map[string]any{}
			parent[key] = child
			stack = append(stack, child)
			indents = append(indents, indent)
			continue
		}

		value, err := parseScalar(rawValue)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行 %s: %w", lineNo, key, err)
		}
		parent[key] = value
	}
	return root, nil
}

// stripComment 去掉行尾注释，引号里的 # 不算注释。
func stripComment(line string) string {
	inSingle, inDouble := false, false
	for i, r := range line {
		switch {
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case r == '#' && !inSingle && !inDouble && (i == 0 || line[i-1] == ' '):
			return line[:i]
		}
	}
	return line
}

// parseScalar 把 YAML 标量解析成 bool / int64 / float64 / string。
func parseScalar(raw string) (any, error) {
	switch raw {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null", "~":
		return nil, nil
	}

	if len(raw) >= 2 {
		if (raw[0] == '"' && raw[len(raw)-1] == '"') || (raw[0] == '\'' && raw[len(raw)-1] == '\'') {
			return raw[1 : len(raw)-1], nil
		}
	}
	if value, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return value, nil
	}
	if value, err := strconv.ParseFloat(raw, 64); err == nil {
		return value, nil
	}
	return raw, nil
}

// renderConfig 按 key 排序把配置渲染成多行文本，保证输出可复现。
func renderConfig(values map[string]any, prefix string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, key := range keys {
		switch v := values[key].(type) {
		case map[string]any:
			fmt.Fprintf(&sb, "%s%s:\n", prefix, key)
			sb.WriteString(renderConfig(v, prefix+"  "))
		default:
			fmt.Fprintf(&sb, "%s%s: %v\n", prefix, key, v)
		}
	}
	return sb.String()
}
