// Package go14_reflection 演示 Go 的反射：reflect.TypeOf / ValueOf、Kind 与 Type、
// 可寻址性与 Set、结构体标签解析、动态调用、reflect.DeepEqual、反射的性能代价，
// 以及 encoding/json 这类库内部大致是怎么用反射的。
//
// 运行方式：
//
//	go run ./cmd/go-learn
//
// 只想验证这一章：
//
//	go test ./internal/chapter/go14_reflection/
package go14_reflection

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// Person 是本章演示结构体反射用的示例类型。
type Person struct {
	Name string
	Age  int
	note string // 未导出字段：反射能看见，但不能设置，也不会被序列化
}

// Demo 运行第 14 章的所有示例。
func Demo() {
	fmt.Println("========== go14_reflection: 反射 ==========")

	fmt.Println("\n--- 1. 反射三定律与 TypeOf/ValueOf ---")
	demoThreeLaws()

	fmt.Println("\n--- 2. Kind 与 Type 不是一回事 ---")
	demoKindAndType()

	fmt.Println("\n--- 3. 从 Value 取出具体的值 ---")
	demoValueExtract()

	fmt.Println("\n--- 4. 可寻址性与 Set ---")
	demoAddressability()

	fmt.Println("\n--- 5. 结构体字段与标签解析 ---")
	demoStructTags()

	fmt.Println("\n--- 6. 动态调用方法 ---")
	demoDynamicCall()

	fmt.Println("\n--- 7. 用反射创建值 ---")
	demoCreateValue()

	fmt.Println("\n--- 8. reflect.DeepEqual 的边界 ---")
	demoDeepEqual()

	fmt.Println("\n--- 9. 反射的性能代价 ---")
	demoPerformance()

	fmt.Println("\n--- 10. encoding/json 内部在做什么 ---")
	demoJSONByReflection()

	fmt.Println("\n--- 11. 什么时候不该用反射 ---")
	demoWhenNotToUse()

	fmt.Println("\n========== 反射演示结束 ==========")
}

// demoThreeLaws 演示反射三定律的前两条。
func demoThreeLaws() {
	var value any = 42

	typ := reflect.TypeOf(value)
	val := reflect.ValueOf(value)
	fmt.Printf("TypeOf(42) = %v（Kind=%v，Name=%v）\n", typ, typ.Kind(), typ.Name())
	fmt.Printf("ValueOf(42) = %v（Kind=%v，Int()=%d）\n", val, val.Kind(), val.Int())

	// 第二定律：从 Value 可以还原成接口值。
	back := val.Interface()
	fmt.Printf("val.Interface() = %v（%T）\n", back, back)

	// 接口本身为 nil 时：TypeOf 返回 nil，ValueOf 返回无效值（不是零值）。
	var nilValue any
	fmt.Printf("TypeOf(nil) = %v\n", reflect.TypeOf(nilValue))
	invalid := reflect.ValueOf(nilValue)
	fmt.Printf("ValueOf(nil).IsValid() = %t，Kind = %v\n", invalid.IsValid(), invalid.Kind())
	fmt.Println("第一定律：接口值可以拆成 Type + Value；第三定律：想改值，Value 必须可寻址（见 14.4）。")
}

// demoKindAndType 对比 Kind（底层种类）与 Type（具体类型）。
func demoKindAndType() {
	type Celsius float64 // 命名类型：Kind 是 float64，Type 是 Celsius

	c := Celsius(36.6)
	ct := reflect.TypeOf(c)
	fmt.Printf("Celsius: Type=%v，Name=%q，Kind=%v\n", ct, ct.Name(), ct.Kind())
	fmt.Printf("PkgPath=%q（命名类型才有包路径）\n", ct.PkgPath())

	values := []any{
		42, "text", 3.5, true, byte(1), int32(7), []int{1}, [2]int{},
		map[string]int{}, Person{}, &Person{}, make(chan int), func() {},
	}
	fmt.Println("同一个 Kind 可以对应很多不同的 Type：")
	for _, item := range values {
		t := reflect.TypeOf(item)
		fmt.Printf("  Kind=%-8v Type=%v\n", t.Kind(), t)
	}
}

// demoValueExtract 演示按 Kind 分支取值。
func demoValueExtract() {
	fmt.Printf("字符串 Value.String() = %q\n", reflect.ValueOf("hello").String())
	fmt.Printf("整数   Value.Int()    = %d\n", reflect.ValueOf(42).Int())
	fmt.Printf("浮点   Value.Float()  = %v\n", reflect.ValueOf(3.5).Float())
	fmt.Printf("布尔   Value.Bool()   = %t\n", reflect.ValueOf(true).Bool())

	// Value.String() 对非字符串类型不会报错，而是返回类型描述，很容易误判。
	number := reflect.ValueOf(42)
	fmt.Printf("整数调 String() = %q（不是 \"42\"）\n", number.String())
	fmt.Printf("想拿原值就用 Interface()：%v（%T）\n", number.Interface(), number.Interface())
	fmt.Println("所以取值前先看 Kind，再调用对应的 Xxx() 方法。")
}

// demoAddressability 演示「反射改值必须通过指针」。
func demoAddressability() {
	n := 10
	direct := reflect.ValueOf(n)
	fmt.Printf("ValueOf(n).CanSet() = %t（只是副本，改不了）\n", direct.CanSet())

	viaPointer := reflect.ValueOf(&n).Elem()
	fmt.Printf("ValueOf(&n).Elem()：CanAddr() = %t，CanSet() = %t\n",
		viaPointer.CanAddr(), viaPointer.CanSet())
	viaPointer.SetInt(20)
	fmt.Printf("SetInt(20) 之后 n = %d\n", n)

	p := &Person{Name: "Alice"}
	name := reflect.ValueOf(p).Elem().FieldByName("Name")
	fmt.Printf("结构体字段：CanSet() = %t，原值 = %q\n", name.CanSet(), name.String())
	name.SetString("Bob")
	fmt.Printf("SetString 之后 p.Name = %q\n", p.Name)

	unexported := reflect.ValueOf(p).Elem().FieldByName("note")
	fmt.Printf("未导出字段 %q：CanSet() = %t（反射也不能绕过可见性）\n", "note", unexported.CanSet())
}

// demoStructTags 演示字段遍历与结构体标签解析。
func demoStructTags() {
	type User struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Password string `json:"-"`
		Nickname string `json:"nickname,omitempty"`
		internal string
	}

	t := reflect.TypeOf(User{})
	fmt.Printf("User 有 %d 个字段：\n", t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag, ok := field.Tag.Lookup("json")
		fmt.Printf("  %-9s %-7v json=%-22q 有标签=%t 导出=%t\n",
			field.Name, field.Type, tag, ok, field.IsExported())
	}

	fmt.Println("标签就是一段字符串，取值时要自己解析：")
	for _, name := range []string{"Name", "Nickname", "Password"} {
		field, ok := t.FieldByName(name)
		if !ok {
			continue
		}
		fieldName, options := ParseTag(field.Tag.Get("json"))
		fmt.Printf("  %-9s → 输出名 %-12q 选项 %v\n", name, fieldName, options)
	}
	fmt.Println("注意 json:\"-\" 表示跳过，omitempty 是选项，二者语义完全不同。")
}

// demoDynamicCall 演示用反射按名字调用方法。
func demoDynamicCall() {
	calc := &Calculator{}

	results, err := CallByName(calc, "Add", 2, 3)
	fmt.Printf("CallByName(Add, 2, 3)      = %v，err = %v\n", results, err)

	results, err = CallByName(calc, "Divide", 10.0, 0.0)
	fmt.Printf("CallByName(Divide, 10, 0)  = %v，err = %v\n", results, err)

	_, err = CallByName(calc, "Missing")
	fmt.Printf("CallByName(Missing)        → err = %v\n", err)

	_, err = CallByName(calc, "Add", 1)
	fmt.Printf("CallByName(Add, 1)         → err = %v\n", err)

	_, err = CallByName(calc, "Add", 1, "x")
	fmt.Printf("CallByName(Add, 1, \"x\")      → err = %v\n", err)
	fmt.Println("动态调用的参数个数与类型只在运行时才能检查，所以调用前要自己兜住错误。")
}

// ParseTag 解析形如 "name,omitempty" 的结构体标签，返回字段名与选项列表。
//
// 标签语法就是逗号分隔的字符串，标准库的做法也是这么朴素。
func ParseTag(tag string) (name string, options []string) {
	if tag == "" {
		return "", nil
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	for _, opt := range parts[1:] {
		if opt != "" {
			options = append(options, opt)
		}
	}
	return name, options
}

// Describe 用反射描述任意值的类型与底层种类，例如 "Celsius/float64"。
func Describe(v any) string {
	t := reflect.TypeOf(v)
	if t == nil {
		return "<nil>/invalid"
	}
	return fmt.Sprintf("%s/%s", t, t.Kind())
}

// JSONFieldNames 返回结构体按 `json` 标签序列化时会用到的字段名。
//
// 未导出字段和标签为 "-" 的字段会被跳过，与 encoding/json 的规则一致。
func JSONFieldNames(v any) ([]string, error) {
	t := reflect.TypeOf(v)
	if t == nil {
		return nil, fmt.Errorf("JSONFieldNames: 收到 nil")
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("JSONFieldNames: 需要结构体，收到 %s", t.Kind())
	}

	names := make([]string, 0, t.NumField())
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
		names = append(names, name)
	}
	return names, nil
}

// SetInt64Field 用反射把结构体某个数字字段设置为 value。
//
// 必须传入结构体指针，否则返回错误而不是 panic——这正是「第三定律」的工程写法：
// 先检查 CanSet，再动手。
func SetInt64Field(target any, fieldName string, value int64) error {
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("SetInt64Field: 需要非 nil 的结构体指针，收到 %s", Describe(target))
	}
	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("SetInt64Field: 需要指向结构体的指针，收到 %s", Describe(target))
	}

	field := elem.FieldByName(fieldName)
	if !field.IsValid() {
		return fmt.Errorf("SetInt64Field: %s 没有字段 %q", elem.Type(), fieldName)
	}
	if !field.CanSet() {
		return fmt.Errorf("SetInt64Field: 字段 %q 不可设置（未导出或值不可寻址）", fieldName)
	}

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.SetInt(value)
	default:
		return fmt.Errorf("SetInt64Field: 字段 %q 是 %s，不是整数", fieldName, field.Kind())
	}
	return nil
}

// UnmarshalInto 演示「反序列化目标必须是指针」这条规则：
// 传值直接返回错误，而不是像 json.Unmarshal 那样报 non-pointer。
func UnmarshalInto(target any, data map[string]any) error {
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Pointer {
		return fmt.Errorf("UnmarshalInto: 目标必须是指针，收到 %s", rv.Kind())
	}
	elem := rv.Elem()
	for key, raw := range data {
		field := elem.FieldByName(key)
		if !field.IsValid() || !field.CanSet() {
			continue
		}
		src := reflect.ValueOf(raw)
		if !src.IsValid() || !src.Type().AssignableTo(field.Type()) {
			continue
		}
		field.Set(src)
	}
	return nil
}

// jsonSanityCheck 在演示里对比标准库与手写实现的输出，保证两者一致。
func jsonSanityCheck(v any) (string, string, bool, error) {
	std, err := json.Marshal(v)
	if err != nil {
		return "", "", false, err
	}
	mine, err := SimpleJSON(v)
	if err != nil {
		return string(std), "", false, err
	}
	return string(std), mine, string(std) == mine, nil
}
