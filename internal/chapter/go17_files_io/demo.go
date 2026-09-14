// Package go17_files_io 演示 Go 标准库中的文件、路径与 IO 操作。
//
// 涵盖主题：
//   - io.Reader / io.Writer 接口与组合
//   - os.ReadFile / WriteFile / OpenFile
//   - bufio.Scanner 与行长限制
//   - io.Copy / TeeReader / MultiReader
//   - io/fs 抽象与 os.DirFS
//   - filepath 跨平台路径处理
//   - filepath.WalkDir 遍历目录
//   - 临时文件、权限与原子写
//   - //go:embed 嵌入静态资源
//
// 运行方式：
//
//	go run ./cmd/go-learn
//
// 只想验证这一章：
//
//	go test ./internal/chapter/go17_files_io/
package go17_files_io

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed embed_demo.txt
var embedFS embed.FS

// Demo 是第 17 章的入口函数，按小节顺序演示文件、路径与 IO 的核心能力。
func Demo() {
	fmt.Println("========================================")
	fmt.Printf("========== go%02d_%s ==========\n", Chapter, "files_io")
	fmt.Println("========================================")

	section1_ReaderWriter()
	section2_OSFileRW()
	section3_Scanner()
	section4_CopyAndTee()
	section5_IOFS()
	section6_Filepath()
	section7_WalkDir()
	section8_TempAndAtomicWrite()
	section9_Embed()

	fmt.Println("========================================")
	fmt.Println("========== 文件、路径与 IO 演示结束 ==========")
	fmt.Println()
}

// --- 17.1 io.Reader / io.Writer 接口与组合 ---
func section1_ReaderWriter() {
	fmt.Println("\n--- 17.1 io.Reader / io.Writer 接口与组合 ---")

	// strings.NewReader 实现了 io.Reader
	r := strings.NewReader("Hello, io.Reader!")
	data, err := io.ReadAll(r)
	if err != nil {
		fmt.Printf("读取失败: %v\n", err)
		return
	}
	fmt.Printf("从 strings.NewReader 读取: %q\n", string(data))

	// bytes.Buffer 同时实现了 io.Reader 和 io.Writer
	var buf bytes.Buffer
	buf.WriteString("Hello, ")
	buf.WriteString("io.Writer!")
	fmt.Printf("bytes.Buffer 内容: %q\n", buf.String())

	// io.MultiWriter：同时写入多个目的地
	var buf1, buf2 bytes.Buffer
	mw := io.MultiWriter(&buf1, &buf2)
	fmt.Fprint(mw, "同时写入两个 buffer")
	fmt.Printf("buf1: %q, buf2: %q\n", buf1.String(), buf2.String())

	// 自定义 Reader：计数读取器
	counter := &countReader{Reader: strings.NewReader("abcdef")}
	io.ReadAll(counter)
	fmt.Printf("自定义 Reader 共读取 %d 字节\n", counter.n)
}

// countReader 是一个包装 Reader，统计读取的字节数。
type countReader struct {
	io.Reader
	n int
}

func (c *countReader) Read(p []byte) (int, error) {
	n, err := c.Reader.Read(p)
	c.n += n
	return n, err
}

// --- 17.2 os 文件读写：ReadFile、WriteFile 与 OpenFile ---
func section2_OSFileRW() {
	fmt.Println("\n--- 17.2 os 文件读写：ReadFile、WriteFile 与 OpenFile ---")

	// 创建临时目录，演示结束后清理
	tmpDir, err := os.MkdirTemp("", "go17_demo_*")
	if err != nil {
		fmt.Printf("创建临时目录失败: %v\n", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	// os.WriteFile：一次性写入（Go 1.16+）
	path := filepath.Join(tmpDir, "hello.txt")
	content := []byte("Hello, os.WriteFile!\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		fmt.Printf("写入失败: %v\n", err)
		return
	}
	fmt.Printf("已写入文件: %s\n", path)

	// os.ReadFile：一次性读取整个文件（Go 1.16+）
	readBack, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("读取失败: %v\n", err)
		return
	}
	fmt.Printf("os.ReadFile 读取结果: %q\n", string(readBack))

	// os.OpenFile：更精细的控制（追加、创建、权限等）
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Printf("OpenFile 失败: %v\n", err)
		return
	}
	// defer Close 的陷阱：如果 Close 返回错误，defer 里很难处理
	// 对于只读操作通常直接 defer；对于写操作，应在最后显式 Close 并检查错误
	defer f.Close()

	if _, err := f.WriteString("追加一行\n"); err != nil {
		fmt.Printf("追加失败: %v\n", err)
		return
	}
	// 显式关闭并检查错误（写操作尤其重要）
	if err := f.Close(); err != nil {
		fmt.Printf("Close 失败: %v\n", err)
	}

	readBack2, _ := os.ReadFile(path)
	fmt.Printf("追加后内容:\n%s", string(readBack2))
}

// --- 17.3 bufio.Scanner 与行长限制 ---
func section3_Scanner() {
	fmt.Println("\n--- 17.3 bufio.Scanner 与行长限制 ---")

	input := "第一行\n第二行\n第三行"
	scanner := bufio.NewScanner(strings.NewReader(input))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		fmt.Printf("  第%d行: %q\n", lineNum, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Scanner 错误: %v\n", err)
	}

	// Scanner 的默认 Split 函数是 ScanLines，单行长不能超过 64K
	// 超过会返回 bufio.ErrTooLong
	fmt.Println("Scanner 默认最大行长度: 64KB（bufio.MaxScanTokenSize）")
	fmt.Println("超长行应改用 bufio.Reader.ReadLine 或自定义 Split 函数")
}

// --- 17.4 io.Copy、TeeReader 与 MultiReader ---
func section4_CopyAndTee() {
	fmt.Println("\n--- 17.4 io.Copy、TeeReader 与 MultiReader ---")

	// io.Copy：从 Reader 复制到 Writer
	src := strings.NewReader("复制这段文字到 buffer")
	var dst bytes.Buffer
	n, err := io.Copy(&dst, src)
	if err != nil {
		fmt.Printf("Copy 失败: %v\n", err)
		return
	}
	fmt.Printf("io.Copy 复制了 %d 字节: %q\n", n, dst.String())

	// io.TeeReader：读取的同时写入另一个 Writer
	var teeBuf bytes.Buffer
	teeR := io.TeeReader(strings.NewReader("TeeReader 演示"), &teeBuf)
	data, _ := io.ReadAll(teeR)
	fmt.Printf("TeeReader 读取: %q, 同时写入: %q\n", string(data), teeBuf.String())

	// io.MultiReader：把多个 Reader 串联成一个
	mr := io.MultiReader(
		strings.NewReader("A"),
		strings.NewReader("B"),
		strings.NewReader("C"),
	)
	result, _ := io.ReadAll(mr)
	fmt.Printf("MultiReader(A,B,C) = %q\n", string(result))
}

// --- 17.5 io/fs 抽象与 os.DirFS ---
func section5_IOFS() {
	fmt.Println("\n--- 17.5 io/fs 抽象与 os.DirFS ---")

	// os.DirFS 把目录包装成 fs.FS 接口
	root := "."
	fsys := os.DirFS(root)

	// 用 fs.ReadFile 读取（不依赖具体操作系统）
	data, err := fs.ReadFile(fsys, "go.mod")
	if err != nil {
		fmt.Printf("fs.ReadFile 失败: %v\n", err)
		return
	}
	fmt.Printf("fs.ReadFile 读取 go.mod 前 40 字节: %q...\n", string(data[:min(len(data), 40)]))

	// fs.ReadDir 读取目录
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		fmt.Printf("fs.ReadDir 失败: %v\n", err)
		return
	}
	fmt.Printf("当前目录包含 %d 个条目:\n", len(entries))
	for i, e := range entries {
		if i >= 5 {
			fmt.Println("  ...")
			break
		}
		fmt.Printf("  %s (isDir=%t)\n", e.Name(), e.IsDir())
	}

	fmt.Println("io/fs 抽象的好处：测试时可以用 os.DirFS 或 testing/fstest.MapFS 替换真实文件系统")
}

// --- 17.6 filepath 跨平台路径处理 ---
func section6_Filepath() {
	fmt.Println("\n--- 17.6 filepath 跨平台路径处理 ---")

	// filepath.Join：自动使用正确的路径分隔符
	p := filepath.Join("a", "b", "c.txt")
	fmt.Printf("filepath.Join: %q\n", p)

	// Base / Dir / Ext
	path := filepath.Join("home", "user", "doc.txt")
	fmt.Printf("路径: %q\n", path)
	fmt.Printf("  Base=%q, Dir=%q, Ext=%q\n", filepath.Base(path), filepath.Dir(path), filepath.Ext(path))

	// IsAbs
	fmt.Printf("  IsAbs(%q)=%t, IsAbs(%q)=%t\n", path, filepath.IsAbs(path), "/absolute/path", filepath.IsAbs("/absolute/path"))

	// Clean：规范化路径
	messy := "a/../b/./c//d"
	fmt.Printf("filepath.Clean(%q) = %q\n", messy, filepath.Clean(messy))

	// ToSlash / FromSlash（与 URL 或跨平台传输交互时使用）
	fmt.Printf("filepath.ToSlash(%q) = %q\n", p, filepath.ToSlash(p))
}

// --- 17.7 filepath.WalkDir 遍历目录 ---
func section7_WalkDir() {
	fmt.Println("\n--- 17.7 filepath.WalkDir 遍历目录 ---")

	// WalkDir 比旧版 Walk 更高效，因为它不会对每个文件调用 os.Lstat
	count := 0
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// 遇到权限错误等不要直接返回，可以记录并跳过
			fmt.Printf("  访问 %s 出错: %v\n", path, err)
			return nil // skip and continue
		}
		if d.IsDir() && path != "." {
			// 不递归进入子目录，保持输出简洁
			return fs.SkipDir
		}
		count++
		if count <= 5 {
			fmt.Printf("  %s (size=%d)\n", path, fileSize(d))
		}
		return nil
	})
	if err != nil {
		fmt.Printf("WalkDir 失败: %v\n", err)
	}
	fmt.Printf("共遍历到 %d 个文件（当前目录，不递归）\n", count)
}

// fileSize 返回 DirEntry 的文件大小，出错时返回 -1。
func fileSize(d fs.DirEntry) int64 {
	info, err := d.Info()
	if err != nil {
		return -1
	}
	return info.Size()
}

// --- 17.8 临时文件、权限与原子写 ---
func section8_TempAndAtomicWrite() {
	fmt.Println("\n--- 17.8 临时文件、权限与原子写 ---")

	// os.MkdirTemp 创建临时目录
	tmpDir, err := os.MkdirTemp("", "go17_atomic_*")
	if err != nil {
		fmt.Printf("创建临时目录失败: %v\n", err)
		return
	}
	defer os.RemoveAll(tmpDir)
	fmt.Printf("临时目录: %s\n", tmpDir)

	// os.CreateTemp 创建临时文件
	tmpFile, err := os.CreateTemp(tmpDir, "tmp_*.txt")
	if err != nil {
		fmt.Printf("创建临时文件失败: %v\n", err)
		return
	}
	fmt.Printf("临时文件: %s\n", tmpFile.Name())
	tmpFile.Close()

	// 原子写：先写到临时文件，再重命名覆盖目标文件
	target := filepath.Join(tmpDir, "final.txt")
	if err := atomicWriteFile(target, []byte("原子写入的内容\n")); err != nil {
		fmt.Printf("原子写失败: %v\n", err)
		return
	}
	data, _ := os.ReadFile(target)
	fmt.Printf("原子写结果: %q\n", string(data))

	// 文件权限
	info, _ := os.Stat(target)
	fmt.Printf("文件权限: %o\n", info.Mode().Perm())
}

// atomicWriteFile 将 data 原子地写入 path。
// 实现方式：写到同目录的临时文件，然后调用 os.Rename 覆盖。
func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".atomic_*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()

	// 写完必须关闭，否则 Windows 上无法 Rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp: %w", err)
	}

	// Rename 在 POSIX 上是原子的；Windows 上如果目标已存在也能覆盖
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// --- 17.9 //go:embed 嵌入静态资源 ---
func section9_Embed() {
	fmt.Println("\n--- 17.9 //go:embed 嵌入静态资源 ---")

	// 通过 embed.FS 读取嵌入的文件
	data, err := embedFS.ReadFile("embed_demo.txt")
	if err != nil {
		fmt.Printf("读取嵌入文件失败: %v\n", err)
		return
	}
	fmt.Printf("嵌入文件内容:\n%s\n", string(data))

	// 也可以直接嵌入为 string 或 []byte
	// //go:embed embed_demo.txt
	// var embedString string

	fmt.Println("//go:embed 规则：")
	fmt.Println("  - 指令必须紧跟在包含它的变量声明之前")
	fmt.Println("  - 只能用于包级变量，不能用于局部变量")
	fmt.Println("  - 路径不支持绝对路径和 .. 上级目录")
	fmt.Println("  - 支持通配符，如 //go:embed *.txt")
}
