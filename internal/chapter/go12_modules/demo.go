// Package go12_modules 演示包、模块与依赖管理。
package go12_modules

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Demo 演示第十二章：包、模块与依赖管理。
func Demo() {
	fmt.Println("==========================================")
	fmt.Println("========== go12_modules: 包、模块与依赖管理 ==========")
	fmt.Println("==========================================")

	fmt.Println("\n--- 1. 包的基本概念 ---")
	demoPackageBasics()

	fmt.Println("\n--- 2. 包的导入与可见性 ---")
	demoPackageVisibility()

	fmt.Println("\n--- 3. init 函数与包初始化顺序 ---")
	demoInitOrder()

	fmt.Println("\n--- 4. 内部包（internal）---")
	demoInternalPackage()

	fmt.Println("\n--- 5. 模块（go.mod）基础 ---")
	demoModuleBasics()

	fmt.Println("\n--- 6. 依赖版本管理 ---")
	demoVersionManagement()

	fmt.Println("\n--- 7. go get 与 go mod 命令 ---")
	demoGoModCommands()

	fmt.Println("\n--- 8. 替换与本地开发（replace）---")
	demoReplace()

	fmt.Println("\n--- 9. vendor 目录 ---")
	demoVendor()

	fmt.Println("\n--- 10. 工作区模式（go.work）---")
	demoWorkspace()

	fmt.Println("\n--- 11. GOPROXY 与私有仓库 ---")
	demoProxy()

	fmt.Println("\n--- 12. 常见依赖问题诊断 ---")
	demoDependencyProblems()

	fmt.Println("\n==========================================")
	fmt.Println("========== 包、模块与依赖管理演示结束 ==========")
	fmt.Println("==========================================")
}

// demoPackageBasics 演示包的基本概念。
func demoPackageBasics() {
	// 包名通常是目录名的最后一部分，但不是强制的
	fmt.Println("当前包名：go12_modules")
	fmt.Println("所属模块：go-learn")

	// 一个目录下所有 .go 文件必须属于同一个包（_test.go 除外）
	fmt.Println("\n包的组成规则：")
	fmt.Println("• 同目录下所有 .go 文件（除测试）必须声明相同的 package")
	fmt.Println("• 包名建议与目录名一致（main 包和_test 包除外）")
	fmt.Println("• 一个包可以拆成多个 .go 文件，逻辑上仍是一个包")

	// GOPATH 与 Go Modules
	fmt.Println("\n历史演进：")
	fmt.Println("• Go 1.11 之前：GOPATH 模式，所有代码必须在 $GOPATH/src 下")
	fmt.Println("• Go 1.11-1.12：引入 Go Modules，需手动开启 GO111MODULE=on")
	fmt.Println("• Go 1.13+：默认启用 Go Modules，在有 go.mod 的目录树下自动生效")
	fmt.Println("• Go 1.16+：默认 GO111MODULE=on，即使没有 go.mod 也用模块模式")
}

// demoPackageVisibility 演示包的导入与可见性。
func demoPackageVisibility() {
	// 大写字母开头：导出（public）
	// 小写字母开头：未导出（package-private）
	fmt.Println("可见性规则：")
	fmt.Println("• Demo() 函数大写开头 → 可被其他包导入")
	fmt.Println("• demoPackageBasics() 小写开头 → 只能在 go12_modules 包内调用")
	fmt.Println("• 结构体字段、接口方法同理")

	fmt.Println("\n导入路径：")
	fmt.Println("• 标准库：import \"fmt\"")
	fmt.Println("• 本模块内包：import \"go-learn/internal/chapter/go12_modules\"")
	fmt.Println("• 第三方模块：import \"github.com/gin-gonic/gin\"")

	fmt.Println("\n导入别名：")
	fmt.Println("• import f \"fmt\" → f.Println()")
	fmt.Println("• import . \"fmt\" → Println() 直接可用（不推荐，易混淆）")
	fmt.Println("• import _ \"net/http/pprof\" → 只执行 init，不使用标识符")
}

// demoInitOrder 演示 init 函数与包初始化顺序。
func demoInitOrder() {
	fmt.Println("包初始化顺序（同一个包内）：")
	fmt.Println("1. 导入的包先初始化（深度优先）")
	fmt.Println("2. 包级变量按声明顺序初始化（依赖关系决定实际顺序）")
	fmt.Println("3. 所有 init() 函数按出现顺序执行")
	fmt.Println("4. 一个包可以有多个 init()，甚至一个文件里也可以有多个")

	fmt.Println("\ninit 函数规则：")
	fmt.Println("• 无参数、无返回值")
	fmt.Println("• 不能被显式调用")
	fmt.Println("• 每个包的 init 只会执行一次，即使被多次导入")

	// 演示当前包的某些初始化行为
	fmt.Printf("\n当前 GOOS=%s，GOARCH=%s\n", runtime.GOOS, runtime.GOARCH)
}

// demoInternalPackage 演示内部包。
func demoInternalPackage() {
	fmt.Println("internal 包的可见性规则：")
	fmt.Println("• 路径中包含 internal 的包，只能被其父目录及子目录导入")
	fmt.Println("• 例如 myproject/internal/auth 只能被 myproject 下的包导入")
	fmt.Println("• github.com/other/repo 无法导入 myproject/internal/auth")

	fmt.Println("\n实战用途：")
	fmt.Println("• 隐藏实现细节，避免外部依赖不稳定的内部 API")
	fmt.Println("• 本项目的 internal/chapter/* 就是内部包")

	// 检查当前包路径
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	if strings.Contains(dir, "internal") {
		fmt.Printf("✓ 当前包在 internal 路径下：%s\n", dir)
	}
}

// demoModuleBasics 演示模块基础。
func demoModuleBasics() {
	fmt.Println("go.mod 文件结构：")
	fmt.Println("module go-learn         ← 模块路径（导入前缀）")
	fmt.Println("go 1.21                 ← Go 版本（最低兼容版本）")
	fmt.Println("require (")
	fmt.Println("    github.com/gin-gonic/gin v1.9.1")
	fmt.Println("    golang.org/x/text v0.14.0")
	fmt.Println(")")

	fmt.Println("\n模块路径规范：")
	fmt.Println("• 开源项目：github.com/用户名/仓库名")
	fmt.Println("• 企业内部：公司域名/项目名（如 gitlab.company.com/team/proj）")
	fmt.Println("• 本地实验：任意名称（如 myapp、demo）")

	fmt.Println("\ngo.sum 文件：")
	fmt.Println("• 记录每个依赖的校验和（哈希）")
	fmt.Println("• 保证构建可重现，防止依赖被篡改")
	fmt.Println("• 应该提交到版本控制系统")
}

// demoVersionManagement 演示依赖版本管理。
func demoVersionManagement() {
	fmt.Println("语义化版本（SemVer）：")
	fmt.Println("• v1.2.3 → 主版本.次版本.补丁版本")
	fmt.Println("• v1.2.3 兼容 v1.2.0，不兼容 v2.0.0")
	fmt.Println("• Go Modules 强制 v2+ 必须在模块路径加 /v2 后缀")

	fmt.Println("\n版本选择（MVS - Minimal Version Selection）：")
	fmt.Println("• 选择满足所有约束的最低版本（不是最新版）")
	fmt.Println("• A 依赖 C v1.2，B 依赖 C v1.3 → 最终使用 v1.3")
	fmt.Println("• 确定性构建：相同的 go.mod 总是得到相同的依赖树")

	fmt.Println("\n伪版本（Pseudo-version）：")
	fmt.Println("• v0.0.0-20230101120000-abcdef123456")
	fmt.Println("• 用于引用未打标签的 commit")
	fmt.Println("• 格式：基础版本-时间戳-commit 哈希前缀")

	fmt.Println("\n主版本升级：")
	fmt.Println("• v1 → v2 需要修改 go.mod 的 module 行")
	fmt.Println("• module github.com/user/repo/v2")
	fmt.Println("• 导入时也要加 /v2：import \"github.com/user/repo/v2/pkg\"")
}

// demoGoModCommands 演示 go mod 命令。
func demoGoModCommands() {
	fmt.Println("常用 go mod 命令：")
	fmt.Println("\ngo mod init <模块路径>")
	fmt.Println("  创建 go.mod，初始化新模块")

	fmt.Println("\ngo mod tidy")
	fmt.Println("  • 添加缺失的依赖")
	fmt.Println("  • 移除未使用的依赖")
	fmt.Println("  • 更新 go.sum")
	fmt.Println("  （最常用，每次修改 import 后运行）")

	fmt.Println("\ngo mod download")
	fmt.Println("  下载依赖到本地缓存（$GOPATH/pkg/mod）")

	fmt.Println("\ngo mod verify")
	fmt.Println("  验证依赖的校验和是否匹配 go.sum")

	fmt.Println("\ngo mod graph")
	fmt.Println("  打印依赖关系图")

	fmt.Println("\ngo mod why <包路径>")
	fmt.Println("  解释为什么需要某个依赖")

	fmt.Println("\ngo get 命令：")
	fmt.Println("• go get github.com/gin-gonic/gin@latest  ← 升级到最新版本")
	fmt.Println("• go get github.com/gin-gonic/gin@v1.9.0  ← 指定版本")
	fmt.Println("• go get github.com/gin-gonic/gin@master  ← 使用分支")
	fmt.Println("• go get -u ./...                         ← 升级所有依赖")
}

// demoReplace 演示 replace 指令。
func demoReplace() {
	fmt.Println("replace 指令用途：")
	fmt.Println("\n1. 本地开发（修改依赖源码调试）")
	fmt.Println("   replace github.com/user/lib => ../lib")

	fmt.Println("\n2. fork 替换（修复第三方库 bug）")
	fmt.Println("   replace github.com/old/lib => github.com/myuser/lib v1.2.3")

	fmt.Println("\n3. 使用私有镜像")
	fmt.Println("   replace golang.org/x/text => github.com/golang/text v0.14.0")

	fmt.Println("\n注意事项：")
	fmt.Println("• replace 只在主模块的 go.mod 中生效")
	fmt.Println("• 作为库发布时，replace 会被忽略")
	fmt.Println("• 本地路径必须是相对路径或绝对路径")

	// 检查当前模块的缓存位置
	gopathEnv := os.Getenv("GOPATH")
	if gopathEnv == "" {
		gopathEnv = build.Default.GOPATH
	}
	fmt.Printf("\n当前 GOPATH=%s\n", gopathEnv)
	fmt.Printf("模块缓存位置：%s/pkg/mod\n", gopathEnv)
}

// demoVendor 演示 vendor 目录。
func demoVendor() {
	fmt.Println("vendor 机制：")
	fmt.Println("\ngo mod vendor")
	fmt.Println("  将所有依赖复制到 vendor/ 目录")
	fmt.Println("\ngo build -mod=vendor")
	fmt.Println("  从 vendor/ 构建，不访问网络和模块缓存")

	fmt.Println("\n使用场景：")
	fmt.Println("• CI/CD 环境网络受限")
	fmt.Println("• 离线构建")
	fmt.Println("• 确保依赖不会因上游删除而消失")

	fmt.Println("\n注意事项：")
	fmt.Println("• vendor/ 目录会让仓库体积变大")
	fmt.Println("• 每次 go mod tidy 后需要重新 go mod vendor")
	fmt.Println("• Go 1.14+ 默认忽略 vendor，需显式指定 -mod=vendor")
}

// demoWorkspace 演示工作区模式。
func demoWorkspace() {
	fmt.Println("go.work 文件（Go 1.18+）：")
	fmt.Println("\ngo 1.21")
	fmt.Println("use (")
	fmt.Println("    ./project-a")
	fmt.Println("    ./project-b")
	fmt.Println("    ../lib")
	fmt.Println(")")

	fmt.Println("\n用途：多模块联合开发")
	fmt.Println("• 同时修改多个相关模块，无需 replace")
	fmt.Println("• project-a 导入 lib 时，直接使用本地 ../lib")
	fmt.Println("• go.work 不应提交到版本控制（类似 .gitignore）")

	fmt.Println("\n命令：")
	fmt.Println("• go work init ./project-a ./project-b")
	fmt.Println("• go work use ../lib")
	fmt.Println("• go work sync  （同步 go.work 到各模块的 go.mod）")
}

// demoProxy 演示 GOPROXY 与私有仓库。
func demoProxy() {
	fmt.Println("GOPROXY 环境变量：")
	proxyEnv := os.Getenv("GOPROXY")
	if proxyEnv == "" {
		proxyEnv = "https://proxy.golang.org,direct"
	}
	fmt.Printf("当前 GOPROXY=%s\n", proxyEnv)

	fmt.Println("\n国内常用代理：")
	fmt.Println("• https://goproxy.cn,direct")
	fmt.Println("• https://goproxy.io,direct")
	fmt.Println("• https://mirrors.aliyun.com/goproxy/,direct")

	fmt.Println("\n设置方法：")
	fmt.Println("  go env -w GOPROXY=https://goproxy.cn,direct")

	fmt.Println("\n私有仓库配置（GOPRIVATE）：")
	fmt.Println("  go env -w GOPRIVATE=gitlab.company.com")
	fmt.Println("  • 跳过代理，直接访问")
	fmt.Println("  • 跳过校验和数据库（sum.golang.org）")

	fmt.Println("\nGONOPROXY / GONOSUMDB：")
	fmt.Println("  • 更细粒度控制哪些模块不走代理/校验")
	fmt.Println("  • 支持通配符：*.company.com")
}

// demoDependencyProblems 演示常见依赖问题诊断。
func demoDependencyProblems() {
	fmt.Println("常见问题诊断：")

	fmt.Println("\n1. 依赖下载失败")
	fmt.Println("   → 检查 GOPROXY 设置")
	fmt.Println("   → 检查网络连接和防火墙")
	fmt.Println("   → 私有仓库设置 GOPRIVATE")

	fmt.Println("\n2. 版本冲突")
	fmt.Println("   → go mod graph | grep <模块>  查看依赖链")
	fmt.Println("   → go mod why <模块>          查看为什么需要")
	fmt.Println("   → 手动 go get <模块>@<版本>   指定版本")

	fmt.Println("\n3. 校验和不匹配")
	fmt.Println("   → go clean -modcache  清理缓存")
	fmt.Println("   → 删除 go.sum 重新 go mod tidy")
	fmt.Println("   → 检查是否有人篡改了依赖")

	fmt.Println("\n4. 循环依赖")
	fmt.Println("   → 提取公共接口到独立包")
	fmt.Println("   → 重新设计包边界")

	fmt.Println("\n5. 本地缓存损坏")
	fmt.Println("   → go clean -modcache")
	fmt.Println("   → go mod download")

	fmt.Println("\n6. IDE 找不到依赖")
	fmt.Println("   → go mod download 确保依赖已下载")
	fmt.Println("   → 重启 IDE 的 Go 插件")
	fmt.Println("   → 检查 GOPATH 和 Go 版本配置")
}
