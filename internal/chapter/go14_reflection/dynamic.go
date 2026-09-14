package go14_reflection

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Calculator 是动态调用示例的目标类型。
type Calculator struct {
	Last float64
}

// Add 返回两个整数之和。
func (c *Calculator) Add(a, b int) int {
	return a + b
}

// Divide 返回两个浮点数之商，除数为 0 时返回错误。
func (c *Calculator) Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("divide by zero")
	}
	return a / b, nil
}

// Store 记录上一次的计算结果，演示「方法带指针接收者」时的动态调用。
func (c *Calculator) Store(v float64) {
	c.Last = v
}

// CallByName 按名字动态调用 recv 上的方法，并把返回值转成 []any。
//
// 动态调用的参数个数、类型都只在运行时才知道，所以这里先做校验再 Call：
// reflect.Value.Call 遇到签名不匹配会直接 panic，不适合暴露给业务代码。
func CallByName(recv any, method string, args ...any) ([]any, error) {
	rv := reflect.ValueOf(recv)
	if !rv.IsValid() {
		return nil, fmt.Errorf("CallByName: 收到 nil")
	}

	m := rv.MethodByName(method)
	if !m.IsValid() {
		return nil, fmt.Errorf("CallByName: %s 上没有方法 %q", rv.Type(), method)
	}

	mt := m.Type()
	if mt.NumIn() != len(args) {
		return nil, fmt.Errorf("CallByName: %s 需要 %d 个参数，收到 %d 个", method, mt.NumIn(), len(args))
	}

	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
		got := in[i].Type()
		if !got.AssignableTo(mt.In(i)) {
			return nil, fmt.Errorf("CallByName: %s 第 %d 个参数需要 %s，收到 %s",
				method, i+1, mt.In(i), got)
		}
	}

	out := m.Call(in)
	values := make([]any, len(out))
	for i, v := range out {
		values[i] = v.Interface()
	}
	return values, nil
}

// --- 7. 用反射创建值 ---
func demoCreateValue() {
	personType := reflect.TypeOf(Person{})
	ptr := reflect.New(personType) // 等价于 &Person{}
	fmt.Printf("reflect.New(%v) → Kind=%v，Elem 的类型 %v\n",
		personType, ptr.Kind(), ptr.Elem().Type())
	ptr.Elem().FieldByName("Name").SetString("Carol")
	fmt.Printf("设置字段后：%+v\n", ptr.Interface())

	slice := reflect.MakeSlice(reflect.TypeOf([]int{}), 0, 3)
	slice = reflect.Append(slice, reflect.ValueOf(7))
	fmt.Printf("MakeSlice + Append = %v，len=%d，cap=%d\n",
		slice.Interface(), slice.Len(), slice.Cap())

	m := reflect.MakeMap(reflect.TypeOf(map[string]int{}))
	m.SetMapIndex(reflect.ValueOf("k"), reflect.ValueOf(1))
	fmt.Printf("MakeMap + SetMapIndex = %v\n", m.Interface())

	zero := reflect.Zero(reflect.TypeOf(0))
	fmt.Printf("reflect.Zero(int) = %v，IsZero=%t\n", zero, zero.IsZero())
	fmt.Println("反射能凭空造值，但每一处都在运行时决定类型，代价是编译期检查消失。")
}

// --- 8. reflect.DeepEqual 的边界 ---
func demoDeepEqual() {
	a := []int{1, 2, 3}
	b := []int{1, 2, 3}
	fmt.Printf("切片不能用 ==，DeepEqual([]int{1,2,3}, 同内容) = %t\n", reflect.DeepEqual(a, b))

	var nilSlice []int
	emptySlice := []int{}
	fmt.Printf("nil 切片 vs 空切片：len 都是 %d/%d，但 DeepEqual = %t\n",
		len(nilSlice), len(emptySlice), reflect.DeepEqual(nilSlice, emptySlice))

	m1 := map[string]int{"a": 1}
	m2 := map[string]int{"a": 1}
	fmt.Printf("同内容 map：DeepEqual = %t\n", reflect.DeepEqual(m1, m2))

	nan := math.NaN()
	fmt.Printf("NaN：== 比较 %t，DeepEqual = %t（DeepEqual 不是「数学相等」）\n",
		nan == nan, reflect.DeepEqual(nan, nan))

	p1 := &Person{Name: "Alice"}
	p2 := &Person{Name: "Alice"}
	fmt.Printf("不同指针、相同内容：DeepEqual = %t\n", reflect.DeepEqual(p1, p2))

	type withFunc struct{ F func() }
	f := func() {}
	fmt.Printf("含函数字段的同值结构体：DeepEqual = %t（函数只有都为 nil 才相等）\n",
		reflect.DeepEqual(withFunc{f}, withFunc{f}))
}

// --- 9. 反射的性能代价 ---
func demoPerformance() {
	fmt.Println("反射的每一步都要在运行时解析类型信息，比直接访问慢，也更容易分配内存。")
	fmt.Println("本章基准对比：")
	fmt.Println("  直接读字段          BenchmarkDirectField")
	fmt.Println("  反射按下标读字段    BenchmarkReflectFieldByIndex")
	fmt.Println("  反射按名字查字段    BenchmarkReflectFieldByName")
	fmt.Println("  反射取值转接口      BenchmarkReflectInterface")
	fmt.Println("  encoding/json 序列化 BenchmarkJSONMarshal")
	fmt.Println("  手写反射序列化      BenchmarkSimpleJSON")
	fmt.Println("运行：go test -bench=. -benchmem ./internal/chapter/go14_reflection/")
	fmt.Println("结论：热路径上「先反射拿到结果，再缓存起来」，不要每次调用都重新解析。")
}

// --- 10. encoding/json 内部在做什么 ---
func demoJSONByReflection() {
	type Account struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Balance int    `json:"balance,omitempty"`
		secret  string
	}

	first := Account{ID: 1, Name: "Ada", secret: "不该被序列化"}
	std, mine, same, err := jsonSanityCheck(first)
	fmt.Printf("encoding/json = %s\n", std)
	fmt.Printf("手写反射实现  = %s\n", mine)
	fmt.Printf("两者一致：%t，err = %v\n", same, err)

	second := Account{ID: 2, Name: "Bob", Balance: 30}
	std, mine, same, _ = jsonSanityCheck(second)
	fmt.Printf("Balance=30 时：encoding/json = %s，手写实现 = %s，一致：%t\n", std, mine, same)

	var decoded Account
	raw := []byte(`{"id":3,"name":"Cy","balance":7}`)
	err = json.Unmarshal(raw, &decoded)
	fmt.Printf("Unmarshal 到指针：%+v，err = %v\n", decoded, err)

	// 这里故意传值。中转成 any 是为了绕过 go vet 的提前拦截，
	// 否则 go vet 会直接报 "call of Unmarshal passes non-pointer as second argument"。
	var notAPointer any = decoded
	err = json.Unmarshal(raw, notAPointer)
	fmt.Printf("Unmarshal 传值 → err = %v\n", err)

	// 手写一个同样要求指针的版本：反射要改调用方的变量，就只能通过指针。
	target := Person{}
	_ = UnmarshalInto(&target, map[string]any{"Name": "Eve", "Age": 25, "note": "跳过"})
	fmt.Printf("手写 UnmarshalInto 到指针：%+v\n", target)
	err = UnmarshalInto(target, map[string]any{"Name": "Eve"})
	fmt.Printf("手写 UnmarshalInto 传值 → err = %v\n", err)
	fmt.Println("这就是「第三定律」的由来：反射想改调用方的变量，就必须拿到指针。")
}

// --- 11. 什么时候不该用反射 ---
func demoWhenNotToUse() {
	names, err := JSONFieldNames(Person{})
	fmt.Printf("按标签解析 Person 的输出字段：%v，err = %v\n", names, err)
	_, err = JSONFieldNames(42)
	fmt.Printf("对非结构体调用 → err = %v\n", err)

	person := &Person{Name: "Dora", Age: 1}
	_ = SetInt64Field(person, "Age", 42)
	fmt.Printf("用反射设置年龄：%+v\n", *person)
	for _, bad := range []struct {
		desc   string
		target any
		field  string
	}{
		{"传值（不可寻址）", Person{}, "Age"},
		{"未导出字段", &Person{}, "note"},
		{"不存在的字段", &Person{}, "Email"},
		{"类型不匹配", &Person{}, "Name"},
	} {
		err := SetInt64Field(bad.target, bad.field, 7)
		fmt.Printf("  %s → %v\n", bad.desc, err)
	}

	cache := map[reflect.Type][]string{}
	t := reflect.TypeOf(Person{})
	if _, ok := cache[t]; !ok {
		fields := make([]string, 0, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			fields = append(fields, t.Field(i).Name)
		}
		cache[t] = fields
	}
	fmt.Printf("reflect.Type 可比较，能当 map key 做类型缓存：%v\n", cache[t])
	fmt.Println("替代方案：编译期能确定的用接口或泛型；只有「运行时才知道类型」时才用反射。")
}

// SimpleJSON 用反射把值序列化成 JSON，用来演示 encoding/json 内部大致做了什么。
//
// 支持结构体、map、切片、数组、指针和基础类型；结构体字段优先取 `json` 标签，
// 标签为 "-" 时跳过，带 omitempty 时零值字段不输出。map 的 key 会排序，
// 保证同一份数据每次输出都一样（标准库也这么做）。
func SimpleJSON(v any) (string, error) {
	var sb strings.Builder
	if err := writeJSON(&sb, reflect.ValueOf(v)); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// writeJSON 递归写出 JSON 片段，是「按 Kind 分支」的典型写法。
func writeJSON(sb *strings.Builder, v reflect.Value) error {
	if !v.IsValid() {
		sb.WriteString("null")
		return nil
	}

	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			sb.WriteString("null")
			return nil
		}
		return writeJSON(sb, v.Elem())
	case reflect.Struct:
		return writeJSONStruct(sb, v)
	case reflect.Map:
		return writeJSONMap(sb, v)
	case reflect.Slice, reflect.Array:
		return writeJSONSlice(sb, v)
	case reflect.String:
		quoted, err := json.Marshal(v.String()) // 复用标准库做转义，避免手写遗漏
		if err != nil {
			return err
		}
		sb.Write(quoted)
		return nil
	case reflect.Bool:
		sb.WriteString(strconv.FormatBool(v.Bool()))
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		sb.WriteString(strconv.FormatInt(v.Int(), 10))
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		sb.WriteString(strconv.FormatUint(v.Uint(), 10))
		return nil
	case reflect.Float32, reflect.Float64:
		sb.WriteString(strconv.FormatFloat(v.Float(), 'g', -1, 64))
		return nil
	default:
		return fmt.Errorf("SimpleJSON: 还不支持 %s", v.Type())
	}
}

// writeJSONStruct 按字段顺序输出对象，跳过未导出字段和标签为 "-" 的字段。
func writeJSONStruct(sb *strings.Builder, v reflect.Value) error {
	t := v.Type()
	sb.WriteByte('{')
	written := 0
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		name, omitEmpty := jsonTag(field)
		if name == "-" {
			continue
		}
		fv := v.Field(i)
		if omitEmpty && fv.IsZero() {
			continue
		}
		if written > 0 {
			sb.WriteByte(',')
		}
		written++
		key, err := json.Marshal(name)
		if err != nil {
			return err
		}
		sb.Write(key)
		sb.WriteByte(':')
		if err := writeJSON(sb, fv); err != nil {
			return err
		}
	}
	sb.WriteByte('}')
	return nil
}

// writeJSONMap 输出对象，key 统一按字符串排序。
func writeJSONMap(sb *strings.Builder, v reflect.Value) error {
	if v.IsNil() {
		sb.WriteString("null")
		return nil
	}
	keys := v.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
	})

	sb.WriteByte('{')
	for i, key := range keys {
		if i > 0 {
			sb.WriteByte(',')
		}
		encoded, err := json.Marshal(fmt.Sprint(key.Interface()))
		if err != nil {
			return err
		}
		sb.Write(encoded)
		sb.WriteByte(':')
		if err := writeJSON(sb, v.MapIndex(key)); err != nil {
			return err
		}
	}
	sb.WriteByte('}')
	return nil
}

// writeJSONSlice 输出数组；nil 切片与标准库一致，序列化成 null。
func writeJSONSlice(sb *strings.Builder, v reflect.Value) error {
	if v.Kind() == reflect.Slice && v.IsNil() {
		sb.WriteString("null")
		return nil
	}
	sb.WriteByte('[')
	for i := 0; i < v.Len(); i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		if err := writeJSON(sb, v.Index(i)); err != nil {
			return err
		}
	}
	sb.WriteByte(']')
	return nil
}

// jsonTag 解析字段的 `json` 标签，返回输出名与是否带 omitempty。
func jsonTag(field reflect.StructField) (name string, omitEmpty bool) {
	tag, ok := field.Tag.Lookup("json")
	if !ok {
		return field.Name, false
	}
	name, options := ParseTag(tag)
	if name == "" {
		name = field.Name
	}
	for _, opt := range options {
		if opt == "omitempty" {
			omitEmpty = true
		}
	}
	return name, omitEmpty
}
