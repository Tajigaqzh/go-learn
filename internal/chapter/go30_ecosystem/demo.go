// Package go30_ecosystem 演示 Go 常用第三方生态与选型原则。
//
// 本章不引入真实的第三方库（避免依赖膨胀），而是用代码示例说明各类库的使用场景、
// 接口设计、选型标准和避免过度依赖的原则。
package go30_ecosystem

import (
	"fmt"
)

// Demo 是第 30 章的统一入口。
func Demo() {
	fmt.Println("\n========== go30_ecosystem: 常用第三方生态与选型 ==========")

	// 30.1 Web 框架选型
	demo30_1()

	// 30.2 ORM 与 SQL 构建器
	demo30_2()

	// 30.3 依赖注入
	demo30_3()

	// 30.4 参数校验
	demo30_4()

	// 30.5 测试库
	demo30_5()

	// 30.6 日志、配置与 CLI 库
	demo30_6()

	// 30.7 gRPC、WebSocket 与消息队列概览
	demo30_7()

	// 30.8 选型原则与避免过度依赖
	demo30_8()

	fmt.Println("\n========== 常用第三方生态与选型演示结束 ==========")
}

// demo30_1 演示 Web 框架选型（gin / echo / fiber / 标准库）。
func demo30_1() {
	fmt.Println("\n--- 30.1 Web 框架选型 ---")

	fmt.Println("常见 Web 框架对比：")
	fmt.Println()
	fmt.Println("1. 标准库 net/http")
	fmt.Println("   优点：零依赖、接口稳定、性能优秀")
	fmt.Println("   缺点：路由功能有限、中间件需自己实现")
	fmt.Println("   适合：小型 API、微服务、学习基础")
	fmt.Println()
	fmt.Println("2. gin (github.com/gin-gonic/gin)")
	fmt.Println("   优点：性能高、API 友好、中间件丰富、社区大")
	fmt.Println("   缺点：错误处理偏魔法、context 绑定到框架")
	fmt.Println("   适合：REST API、中小型 Web 服务")
	fmt.Println()
	fmt.Println("3. echo (github.com/labstack/echo)")
	fmt.Println("   优点：性能好、中间件设计清晰、HTTP/2 支持")
	fmt.Println("   缺点：社区比 gin 小")
	fmt.Println("   适合：REST API、需要 HTTP/2 的场景")
	fmt.Println()
	fmt.Println("4. fiber (github.com/gofiber/fiber)")
	fmt.Println("   优点：性能极高（基于 fasthttp）、Express 风格 API")
	fmt.Println("   缺点：不兼容标准库 net/http、生态隔离")
	fmt.Println("   适合：追求极致性能、不需要标准库兼容")
	fmt.Println()
	fmt.Println("选型建议：")
	fmt.Println("  - 新项目优先标准库（足够大部分场景）")
	fmt.Println("  - 需要快速开发选 gin（生态最成熟）")
	fmt.Println("  - 追求性能且能接受非标准库选 fiber")
}

// demo30_2 演示 ORM 与 SQL 构建器选型。
func demo30_2() {
	fmt.Println("\n--- 30.2 ORM 与 SQL 构建器 ---")

	fmt.Println("常见数据库库对比：")
	fmt.Println()
	fmt.Println("1. database/sql (标准库)")
	fmt.Println("   优点：稳定、轻量、无抽象损耗")
	fmt.Println("   缺点：手写 SQL、无类型安全、重复代码多")
	fmt.Println("   适合：简单查询、对性能敏感的场景")
	fmt.Println()
	fmt.Println("2. sqlx (github.com/jmoiron/sqlx)")
	fmt.Println("   优点：标准库扩展、StructScan、Named 查询")
	fmt.Println("   缺点：仍需手写 SQL、无 migration")
	fmt.Println("   适合：想少写模板代码但不想用 ORM")
	fmt.Println()
	fmt.Println("3. sqlc (github.com/sqlc-dev/sqlc)")
	fmt.Println("   优点：SQL → Go 代码生成、类型安全、性能无损")
	fmt.Println("   缺点：需要学 sqlc 语法、不支持动态查询")
	fmt.Println("   适合：SQL 优先、类型安全、不想学 ORM DSL")
	fmt.Println()
	fmt.Println("4. GORM (gorm.io/gorm)")
	fmt.Println("   优点：功能全、自动 migration、preload / association")
	fmt.Println("   缺点：性能开销、魔法多、复杂查询难写")
	fmt.Println("   适合：快速原型、CRUD 为主、团队熟悉 ORM")
	fmt.Println()
	fmt.Println("5. ent (entgo.io)")
	fmt.Println("   优点：类型安全、图遍历、schema as code、代码生成")
	fmt.Println("   缺点：学习曲线陡、生成代码量大")
	fmt.Println("   适合：复杂关系模型、强类型需求")
	fmt.Println()
	fmt.Println("选型建议：")
	fmt.Println("  - 简单项目用 database/sql + sqlx")
	fmt.Println("  - SQL 优先且要类型安全选 sqlc")
	fmt.Println("  - 快速开发且 CRUD 为主选 GORM")
	fmt.Println("  - 复杂关系模型选 ent")
}

// demo30_3 演示依赖注入（wire / fx / 手动注入）。
func demo30_3() {
	fmt.Println("\n--- 30.3 依赖注入 ---")

	fmt.Println("常见依赖注入方案：")
	fmt.Println()
	fmt.Println("1. 手动注入（构造函数）")
	fmt.Println("   优点：显式、IDE 友好、零依赖")
	fmt.Println("   缺点：代码量大、层级深时繁琐")
	fmt.Println("   示例：")
	fmt.Println("     db := NewDB(config)")
	fmt.Println("     repo := NewUserRepo(db)")
	fmt.Println("     service := NewUserService(repo)")
	fmt.Println()
	fmt.Println("2. wire (github.com/google/wire)")
	fmt.Println("   优点：编译期代码生成、类型安全、无运行时反射")
	fmt.Println("   缺点：需要学 wire 语法、生成代码需提交")
	fmt.Println("   适合：中大型项目、依赖关系复杂")
	fmt.Println()
	fmt.Println("3. fx (go.uber.org/fx)")
	fmt.Println("   优点：运行时注入、生命周期管理、热替换")
	fmt.Println("   缺点：运行时反射、错误信息不直观")
	fmt.Println("   适合：长期运行服务、需要生命周期管理")
	fmt.Println()
	fmt.Println("选型建议：")
	fmt.Println("  - 小项目手动注入即可")
	fmt.Println("  - 依赖层级深、构造繁琐选 wire")
	fmt.Println("  - 需要生命周期管理（启动/关闭顺序）选 fx")
}

// demo30_4 演示参数校验（validator / 手动校验）。
func demo30_4() {
	fmt.Println("\n--- 30.4 参数校验 ---")

	fmt.Println("常见校验方案：")
	fmt.Println()
	fmt.Println("1. 手动校验")
	fmt.Println("   优点：显式、灵活、无依赖")
	fmt.Println("   缺点：重复代码、错误信息不统一")
	fmt.Println("   示例：")
	fmt.Println("     if req.Email == \"\" { return errors.New(\"email 不能为空\") }")
	fmt.Println("     if len(req.Password) < 8 { return errors.New(\"密码至少 8 位\") }")
	fmt.Println()
	fmt.Println("2. validator (github.com/go-playground/validator)")
	fmt.Println("   优点：标签驱动、规则丰富、错误信息可定制")
	fmt.Println("   缺点：标签嵌套复杂时可读性差")
	fmt.Println("   示例：")
	fmt.Println("     type User struct {")
	fmt.Println("       Email    string `validate:\"required,email\"`")
	fmt.Println("       Password string `validate:\"required,min=8\"`")
	fmt.Println("     }")
	fmt.Println()
	fmt.Println("选型建议：")
	fmt.Println("  - 校验规则简单用手动校验")
	fmt.Println("  - 规则复杂、需要统一错误格式选 validator")
}

// demo30_5 演示测试库（testify / gomock）。
func demo30_5() {
	fmt.Println("\n--- 30.5 测试库 ---")

	fmt.Println("常见测试库：")
	fmt.Println()
	fmt.Println("1. testing (标准库)")
	fmt.Println("   优点：零依赖、稳定")
	fmt.Println("   缺点：断言不友好（需要手写 if err != nil）")
	fmt.Println()
	fmt.Println("2. testify (github.com/stretchr/testify)")
	fmt.Println("   优点：assert / require 断言友好、suite 支持")
	fmt.Println("   缺点：引入依赖")
	fmt.Println("   示例：")
	fmt.Println("     assert.NoError(t, err)")
	fmt.Println("     assert.Equal(t, expected, actual)")
	fmt.Println()
	fmt.Println("3. gomock (github.com/golang/mock)")
	fmt.Println("   优点：官方 mock 工具、接口 mock 代码生成")
	fmt.Println("   缺点：需要生成代码、接口优先设计")
	fmt.Println("   适合：单元测试、需要 mock 外部依赖")
	fmt.Println()
	fmt.Println("选型建议：")
	fmt.Println("  - 简单测试用标准库")
	fmt.Println("  - 需要友好断言选 testify/assert")
	fmt.Println("  - 需要 mock 接口选 gomock")
}

// demo30_6 演示日志、配置与 CLI 库。
func demo30_6() {
	fmt.Println("\n--- 30.6 日志、配置与 CLI 库 ---")

	fmt.Println("常见库：")
	fmt.Println()
	fmt.Println("日志库：")
	fmt.Println("  1. log/slog (标准库，Go 1.21+)")
	fmt.Println("     优点：结构化日志、零依赖、性能好")
	fmt.Println("     推荐：新项目优先 slog")
	fmt.Println("  2. zap (go.uber.org/zap)")
	fmt.Println("     优点：性能极高、字段类型化")
	fmt.Println("     缺点：API 偏底层")
	fmt.Println("  3. logrus (github.com/sirupsen/logrus)")
	fmt.Println("     优点：API 友好、Hook 丰富")
	fmt.Println("     缺点：性能较低、维护不活跃")
	fmt.Println()
	fmt.Println("配置库：")
	fmt.Println("  1. flag (标准库)")
	fmt.Println("     优点：简单、零依赖")
	fmt.Println("  2. viper (github.com/spf13/viper)")
	fmt.Println("     优点：支持多格式（JSON/YAML/TOML）、环境变量、热重载")
	fmt.Println("     缺点：依赖重")
	fmt.Println()
	fmt.Println("CLI 库：")
	fmt.Println("  1. flag (标准库)")
	fmt.Println("     优点：简单、零依赖")
	fmt.Println("     缺点：不支持子命令")
	fmt.Println("  2. cobra (github.com/spf13/cobra)")
	fmt.Println("     优点：子命令支持、自动生成帮助")
	fmt.Println("     适合：kubectl / docker 风格 CLI")
	fmt.Println()
	fmt.Println("选型建议：")
	fmt.Println("  - 日志：Go 1.21+ 用 slog，否则用 zap")
	fmt.Println("  - 配置：简单用 flag，多格式用 viper")
	fmt.Println("  - CLI：无子命令用 flag，有子命令用 cobra")
}

// demo30_7 演示 gRPC、WebSocket 与消息队列概览。
func demo30_7() {
	fmt.Println("\n--- 30.7 gRPC、WebSocket 与消息队列概览 ---")

	fmt.Println("gRPC：")
	fmt.Println("  库：google.golang.org/grpc")
	fmt.Println("  用途：微服务间通信、强类型 RPC")
	fmt.Println("  优点：Protobuf 序列化高效、双向流、负载均衡")
	fmt.Println("  缺点：浏览器支持有限（需 grpc-web）")
	fmt.Println()
	fmt.Println("WebSocket：")
	fmt.Println("  库：github.com/gorilla/websocket、nhooyr.io/websocket")
	fmt.Println("  用途：实时推送、聊天、游戏")
	fmt.Println("  优点：双向通信、浏览器原生支持")
	fmt.Println("  缺点：需处理重连、心跳")
	fmt.Println()
	fmt.Println("消息队列：")
	fmt.Println("  1. NATS (github.com/nats-io/nats.go)")
	fmt.Println("     优点：轻量、高性能、云原生")
	fmt.Println("     适合：微服务解耦、事件驱动")
	fmt.Println("  2. RabbitMQ (github.com/rabbitmq/amqp091-go)")
	fmt.Println("     优点：功能丰富、持久化、死信队列")
	fmt.Println("     适合：传统企业、复杂路由")
	fmt.Println("  3. Kafka (github.com/segmentio/kafka-go)")
	fmt.Println("     优点：高吞吐、日志存储、事件溯源")
	fmt.Println("     适合：大数据、日志收集")
}

// demo30_8 演示选型原则与避免过度依赖。
func demo30_8() {
	fmt.Println("\n--- 30.8 选型原则与避免过度依赖 ---")

	fmt.Println("选型原则：")
	fmt.Println("  1. 优先标准库（零依赖、长期稳定）")
	fmt.Println("  2. 评估维护活跃度（最近提交、issue 响应）")
	fmt.Println("  3. 看社区规模（GitHub stars、下载量、文档质量）")
	fmt.Println("  4. 检查 Go 版本要求（避免引入过新或过旧的库）")
	fmt.Println("  5. 避免深度绑定（接口抽象、依赖倒置）")
	fmt.Println()
	fmt.Println("避免过度依赖：")
	fmt.Println("  1. 不要为了「少写一行代码」引入库")
	fmt.Println("     示例：字符串处理用标准库 strings，不需要 lodash 风格库")
	fmt.Println("  2. 警惕「框架锁定」")
	fmt.Println("     示例：业务逻辑不要依赖 gin.Context，用接口抽象")
	fmt.Println("  3. 大型依赖评估「值不值」")
	fmt.Println("     示例：只用 viper 读 YAML，可以换成 gopkg.in/yaml.v3")
	fmt.Println("  4. 定期清理无用依赖")
	fmt.Println("     命令：go mod tidy")
	fmt.Println()
	fmt.Println("依赖倒置示例（避免框架锁定）：")
	fmt.Println("  // 错误：业务逻辑依赖 gin")
	fmt.Println("  func GetUser(c *gin.Context) { ... }")
	fmt.Println()
	fmt.Println("  // 正确：业务逻辑与框架解耦")
	fmt.Println("  type UserService interface {")
	fmt.Println("    GetUser(ctx context.Context, id int) (*User, error)")
	fmt.Println("  }")
	fmt.Println("  // HTTP 层只做适配")
	fmt.Println("  func ginGetUser(svc UserService) gin.HandlerFunc {")
	fmt.Println("    return func(c *gin.Context) {")
	fmt.Println("      user, err := svc.GetUser(c.Request.Context(), ...)")
	fmt.Println("      // 返回 JSON")
	fmt.Println("    }")
	fmt.Println("  }")
}
