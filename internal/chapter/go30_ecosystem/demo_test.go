package go30_ecosystem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDemoPrintsMarkers 是 smoke 测试：本章正文是选型说明（无独立可验逻辑），
// 但 Demo 的起止标记是集成测试和文档依赖的契约，这里在章节包内守住它，防止
// 未来改动 Demo 时漏改标记而没被发现。
func TestDemoPrintsMarkers(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	path := f.Name()

	old := os.Stdout
	os.Stdout = f
	Demo()
	os.Stdout = old
	_ = f.Close()

	out, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("读取输出失败: %v", err)
	}
	text := string(out)
	if !strings.Contains(text, "========== go30_ecosystem: 常用第三方生态与选型 ==========") ||
		!strings.Contains(text, "========== 常用第三方生态与选型演示结束 ==========") {
		t.Fatalf("Demo 输出缺少起止标记")
	}
}
