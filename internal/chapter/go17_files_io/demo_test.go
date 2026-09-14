package go17_files_io

import (
	"bufio"
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCountReader 验证自定义 Reader 的计数功能。
func TestCountReader(t *testing.T) {
	s := "hello"
	cr := &countReader{Reader: strings.NewReader(s)}
	data, err := io.ReadAll(cr)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(data) != s {
		t.Errorf("data = %q, want %q", string(data), s)
	}
	if cr.n != len(s) {
		t.Errorf("count = %d, want %d", cr.n, len(s))
	}
}

// TestReadFileWriteFile 验证 os.WriteFile 与 os.ReadFile。
func TestReadFileWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello, world\n")

	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	readBack, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !bytes.Equal(readBack, content) {
		t.Errorf("ReadFile = %q, want %q", readBack, content)
	}
}

// TestScannerLines 验证 bufio.Scanner 逐行读取。
func TestScannerLines(t *testing.T) {
	input := "a\nb\nc"
	scanner := strings.NewReader(input)
	// 简单测试：Scanner 能读完 3 行
	buf := bufio.NewScanner(scanner)
	lines := 0
	for buf.Scan() {
		lines++
	}
	if err := buf.Err(); err != nil {
		t.Fatalf("Scanner error: %v", err)
	}
	if lines != 3 {
		t.Errorf("lines = %d, want 3", lines)
	}
}

// TestTeeReader 验证 TeeReader 同时读取和写入。
func TestTeeReader(t *testing.T) {
	var buf bytes.Buffer
	tr := io.TeeReader(strings.NewReader("abc"), &buf)
	data, err := io.ReadAll(tr)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(data) != "abc" {
		t.Errorf("read = %q, want abc", string(data))
	}
	if buf.String() != "abc" {
		t.Errorf("tee buf = %q, want abc", buf.String())
	}
}

// TestMultiReader 验证 MultiReader 串联多个 Reader。
func TestMultiReader(t *testing.T) {
	mr := io.MultiReader(strings.NewReader("a"), strings.NewReader("b"))
	data, err := io.ReadAll(mr)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(data) != "ab" {
		t.Errorf("data = %q, want ab", string(data))
	}
}

// TestDirFS 验证 os.DirFS 能读取当前目录的文件。
func TestDirFS(t *testing.T) {
	fsys := os.DirFS(".")
	data, err := fs.ReadFile(fsys, "version.go")
	if err != nil {
		t.Fatalf("fs.ReadFile failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("version.go should not be empty")
	}
}

// TestFilepathJoin 验证 filepath.Join 使用正确分隔符。
func TestFilepathJoin(t *testing.T) {
	p := filepath.Join("a", "b", "c.txt")
	if !strings.Contains(p, "c.txt") {
		t.Errorf("Join result = %q, should contain c.txt", p)
	}
}

// TestFilepathBaseDirExt 验证 filepath 的解析函数。
func TestFilepathBaseDirExt(t *testing.T) {
	p := filepath.Join("home", "user", "doc.txt")
	if filepath.Base(p) != "doc.txt" {
		t.Errorf("Base = %q, want doc.txt", filepath.Base(p))
	}
	if filepath.Ext(p) != ".txt" {
		t.Errorf("Ext = %q, want .txt", filepath.Ext(p))
	}
}

// TestAtomicWriteFile 验证原子写入。
func TestAtomicWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "atomic.txt")
	content := []byte("atomic content")

	if err := atomicWriteFile(target, content); err != nil {
		t.Fatalf("atomicWriteFile failed: %v", err)
	}

	readBack, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !bytes.Equal(readBack, content) {
		t.Errorf("content = %q, want %q", readBack, content)
	}
}

// TestEmbedFS 验证嵌入文件可读。
func TestEmbedFS(t *testing.T) {
	data, err := embedFS.ReadFile("embed_demo.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(data), "go:embed") {
		t.Errorf("embed content = %q, should contain 'go:embed'", string(data))
	}
}

// TestChapterMetadata 保证章节编号和标题跟文档里的章节名一致。
func TestChapterMetadata(t *testing.T) {
	if Chapter != 17 {
		t.Errorf("Chapter = %d, want 17", Chapter)
	}
	if ChapterTitle != "文件、路径与 IO" {
		t.Errorf("ChapterTitle = %q, want 文件、路径与 IO", ChapterTitle)
	}
}
