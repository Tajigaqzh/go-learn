// Command go-learn 依次运行各章的演示代码。
//
// 用法：
//
//	go run ./cmd/go-learn
//
// 每写完一章，就在下面的 import 区和 main 里按章节顺序登记该章的 Demo 函数。
// 章节包放在 internal/chapter/ 下，命名形如 goNN_主题。
package main

import (
	"go-learn/internal/chapter/go01_hello"
	"go-learn/internal/chapter/go02_variables"
	"go-learn/internal/chapter/go03_operators_fmt"
	"go-learn/internal/chapter/go04_control_flow"
	"go-learn/internal/chapter/go05_functions"
	"go-learn/internal/chapter/go06_pointers"
	"go-learn/internal/chapter/go07_slices_maps"
	"go-learn/internal/chapter/go08_strings"
	"go-learn/internal/chapter/go09_structs_methods"
	"go-learn/internal/chapter/go10_interfaces"
	"go-learn/internal/chapter/go11_errors"
	"go-learn/internal/chapter/go12_modules"
	"go-learn/internal/chapter/go13_generics"
	"go-learn/internal/chapter/go14_reflection"
	"go-learn/internal/chapter/go15_stdlib_time_sort"
	"go-learn/internal/chapter/go16_stdlib_text"
	"go-learn/internal/chapter/go17_files_io"
	"go-learn/internal/chapter/go18_serde_config"
	"go-learn/internal/chapter/go19_testing"
	"go-learn/internal/chapter/go20_concurrency"
	"go-learn/internal/chapter/go21_concurrency_patterns"
	"go-learn/internal/chapter/go22_context"
	"go-learn/internal/chapter/go23_runtime"
	"go-learn/internal/chapter/go24_performance"
	"go-learn/internal/chapter/go25_net"
	"go-learn/internal/chapter/go26_http_server"
	"go-learn/internal/chapter/go27_http_client"
	"go-learn/internal/chapter/go28_database"
	"go-learn/internal/chapter/go29_cli_logging"
	"go-learn/internal/chapter/go30_ecosystem"
	"go-learn/internal/chapter/go31_unsafe_cgo"
	"go-learn/internal/chapter/go32_engineering"
	"go-learn/internal/chapter/go33_app"
)

func main() {
	// 章节注册区：新增章节时在这里按顺序追加 goNN_主题.Demo()。
	go01_hello.Demo()
	go02_variables.Demo()
	go03_operators_fmt.Demo()
	go04_control_flow.Demo()
	go05_functions.Demo()
	go06_pointers.Demo()
	go07_slices_maps.Demo()
	go08_strings.Demo()
	go09_structs_methods.Demo()
	go10_interfaces.Demo()
	go11_errors.Demo()
	go12_modules.Demo()
	go13_generics.Demo()
	go14_reflection.Demo()
	go15_stdlib_time_sort.Demo()
	go16_stdlib_text.Demo()
	go17_files_io.Demo()
	go18_serde_config.Demo()
	go19_testing.Demo()
	go20_concurrency.Demo()
	go21_concurrency_patterns.Demo()
	go22_context.Demo()
	go23_runtime.Demo()
	go24_performance.Demo()
	go25_net.Demo()
	go26_http_server.Demo()
	go27_http_client.Demo()
	go28_database.Demo()
	go29_cli_logging.Demo()
	go30_ecosystem.Demo()
	go31_unsafe_cgo.Demo()
	go32_engineering.Demo()
	go33_app.Demo()
}
