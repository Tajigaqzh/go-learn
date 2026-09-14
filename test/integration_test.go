// Package test 把编译出来的命令行程序当成黑盒来验证。
//
// 集成测试不关心内部实现，只检查「构建得出来、跑得起来、输出里有该有的标记」。
// 这是第 19 章测试分层会展开的做法，这里先用它守住每章 Demo 的输出契约。
package test

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBinaryPrintsChapterMarkers(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "go-learn")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	build := exec.Command("go", "build", "-o", bin, "./cmd/go-learn")
	// 测试运行时的工作目录是包目录（test/），回到模块根目录再构建
	build.Dir = ".."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("构建失败: %v\n%s", err, out)
	}

	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatalf("运行失败: %v\n%s", err, out)
	}

	stdout := string(out)
	for _, marker := range []string{
		"========== go01_hello: 环境搭建与第一个 Go 程序 ==========",
		"========== 环境搭建与第一个 Go 程序演示结束 ==========",
		"========== go08_strings: 字符串、字节与 Unicode ==========",
		"========== 字符串、字节与 Unicode演示结束 ==========",
		"========== go11_errors: 错误处理 ==========",
		"========== 错误处理演示结束 ==========",
		"========== go14_reflection: 反射 ==========",
		"========== 反射演示结束 ==========",
		"========== go18_serde_config: 序列化与配置 ==========",
		"========== 序列化与配置演示结束 ==========",
		"========== go19_testing: 测试、基准与代码质量 ==========",
		"========== 测试、基准与代码质量演示结束 ==========",
		"========== go24_performance: 性能分析与优化 ==========",
		"========== 性能分析与优化演示结束 ==========",
		"========== go26_http_server: HTTP 服务端与 REST API ==========",
		"========== HTTP 服务端与 REST API 演示结束 ==========",
		"========== go27_http_client: HTTP 客户端与外部服务 ==========",
		"========== HTTP 客户端与外部服务演示结束 ==========",
		"========== go29_cli_logging: 命令行工具、日志与配置 ==========",
		"========== 命令行工具、日志与配置演示结束 ==========",
	} {
		if !strings.Contains(stdout, marker) {
			t.Errorf("输出里找不到章节标记 %q\n完整输出:\n%s", marker, stdout)
		}
	}
}
