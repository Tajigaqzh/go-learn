# 第 30 章 · 常用第三方生态与选型

在前面的章节里，我们系统学习了 Go 的语言特性、标准库和工程实践。实际项目中，你还会面临一个重要问题：**什么时候该用第三方库，用哪一个，怎么避免过度依赖**？

Go 的标准库覆盖面很广（HTTP、JSON、数据库、测试），许多场景下标准库已经够用。但在 Web 框架、ORM、日志、配置管理等领域，社区积累了大量优秀的第三方库。它们提供更高层的抽象、更友好的 API，能显著提升开发效率——但也带来依赖管理、版本兼容、框架锁定等代价。

本章不会引入真实的第三方库（避免仓库依赖膨胀），而是用对比表格和代码示例，帮你建立**选型思维**：理解每类库的适用场景、设计哲学、常见陷阱，以及如何通过接口抽象降低耦合。你会学到：

- Web 框架（gin / echo / fiber / 标准库）的对比与选型
- ORM 与 SQL 构建器（GORM / ent / sqlc / sqlx）的取舍
- 依赖注入（wire / fx / 手动注入）的适用场景
- 参数校验、测试库、日志库的常见方案
- gRPC、WebSocket、消息队列的概览
- **选型原则**与**避免过度依赖**的实践建议

本章配套代码在 `internal/chapter/go30_ecosystem/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

---

## 30.1 Web 框架选型

Go 的 `net/http` 标准库设计优秀，性能出色，很多团队直接用标准库构建生产服务。但标准库的路由功能有限（Go 1.22 前只支持固定路径，Go 1.22 加入了方法匹配和 `{id}` 通配符），中间件需要自己实现，没有参数绑定和校验。社区的 Web 框架填补了这些空白。

### 常见 Web 框架对比

| 框架 | 优点 | 缺点 | 适用场景 |
| --- | --- | --- | --- |
| **标准库 net/http** | 零依赖、接口稳定、性能优秀 | 路由功能有限、中间件需自己实现 | 小型 API、微服务、学习基础 |
| **gin** | 性能高、API 友好、中间件丰富、社区大 | 错误处理偏魔法、context 绑定到框架 | REST API、中小型 Web 服务 |
| **echo** | 性能好、中间件设计清晰、HTTP/2 支持 | 社区比 gin 小 | REST API、需要 HTTP/2 的场景 |
| **fiber** | 性能极高（基于 fasthttp）、Express 风格 API | 不兼容标准库 net/http、生态隔离 | 追求极致性能、不需要标准库兼容 |

### gin 示例（伪代码）

```go
// gin 的典型用法（不实际引入）
router := gin.Default()
router.GET("/users/:id", func(c *gin.Context) {
    id := c.Param("id")
    c.JSON(200, gin.H{"id": id, "name": "Alice"})
})
router.Run(":8080")
```

**优点**：路由、参数绑定、JSON 响应一气呵成。  
**缺点**：业务逻辑依赖 `gin.Context`，换框架时需要大量改动。

### 标准库示例（实际代码见第 26 章）

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    json.NewEncoder(w).Encode(map[string]string{"id": id, "name": "Alice"})
})
http.ListenAndServe(":8080", mux)
```

**优点**：零依赖，Go 1.22+ 的路由增强已足够大部分场景。  
**缺点**：中间件、错误处理需要自己封装。

### 选型建议

1. **新项目优先标准库**：Go 1.22+ 的路由增强已覆盖 80% 场景，性能、稳定性无可挑剔。
2. **需要快速开发选 gin**：生态最成熟，中间件、插件丰富，团队上手快。
3. **追求极致性能且能接受非标准库选 fiber**：性能比 gin 高 30%–50%，但与标准库生态隔离。
4. **避免框架锁定**：业务逻辑用接口抽象，HTTP 层只做适配（见 30.8）。

---

## 30.2 ORM 与 SQL 构建器

数据库访问是后端开发的核心。Go 的 `database/sql` 标准库轻量高效，但手写 SQL、扫描结果的模板代码多。社区提供了从"薄包装"到"全功能 ORM"的多种方案。

### 常见数据库库对比

| 库 | 优点 | 缺点 | 适用场景 |
| --- | --- | --- | --- |
| **database/sql** | 稳定、轻量、无抽象损耗 | 手写 SQL、无类型安全、重复代码多 | 简单查询、对性能敏感 |
| **sqlx** | 标准库扩展、StructScan、Named 查询 | 仍需手写 SQL、无 migration | 想少写模板代码但不想用 ORM |
| **sqlc** | SQL → Go 代码生成、类型安全、性能无损 | 需要学 sqlc 语法、不支持动态查询 | SQL 优先、类型安全、不想学 ORM DSL |
| **GORM** | 功能全、自动 migration、preload / association | 性能开销、魔法多、复杂查询难写 | 快速原型、CRUD 为主 |
| **ent** | 类型安全、图遍历、schema as code、代码生成 | 学习曲线陡、生成代码量大 | 复杂关系模型、强类型需求 |

### sqlx 示例（伪代码）

```go
// sqlx 的 StructScan 示例
var users []User
err := db.Select(&users, "SELECT * FROM users WHERE age > ?", 18)
// 免去手写 Scan(&user.ID, &user.Name, ...)
```

### GORM 示例（伪代码）

```go
// GORM 的链式查询
db.Where("age > ?", 18).Order("created_at DESC").Limit(10).Find(&users)
// 自动生成 SQL、无需手写
```

**优点**：API 友好，快速开发。  
**缺点**：复杂查询（JOIN、子查询）难以表达，生成的 SQL 不直观。

### sqlc 示例（伪代码）

```sql
-- queries.sql
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;
```

```go
// sqlc 生成的代码
user, err := queries.GetUser(ctx, 42)
// 类型安全、无运行时反射、性能等同手写 SQL
```

### 选型建议

1. **简单项目用 database/sql + sqlx**：依赖少，性能好，SQL 技能可迁移。
2. **SQL 优先且要类型安全选 sqlc**：写 SQL、生成 Go 代码，两全其美。
3. **快速开发且 CRUD 为主选 GORM**：自动 migration、关联加载省时间。
4. **复杂关系模型选 ent**：图遍历、类型安全的代价是学习曲线。

**避坑建议**：不要把 ORM 当成"不学 SQL 的捷径"——复杂查询时你仍需理解 SQL，否则生成的查询可能低效。

---

## 30.3 依赖注入

依赖注入（Dependency Injection, DI）解决"如何组装对象"的问题。Go 没有反射容器的传统，社区分为三派：手动注入、编译期代码生成（wire）、运行时反射（fx）。

### 常见依赖注入方案

| 方案 | 优点 | 缺点 | 适用场景 |
| --- | --- | --- | --- |
| **手动注入** | 显式、IDE 友好、零依赖 | 代码量大、层级深时繁琐 | 小项目、依赖关系简单 |
| **wire** | 编译期代码生成、类型安全、无运行时反射 | 需要学 wire 语法、生成代码需提交 | 中大型项目、依赖关系复杂 |
| **fx** | 运行时注入、生命周期管理、热替换 | 运行时反射、错误信息不直观 | 长期运行服务、需要生命周期管理 |

### 手动注入示例

```go
func main() {
    config := LoadConfig()
    db := NewDB(config.DSN)
    repo := NewUserRepo(db)
    service := NewUserService(repo)
    handler := NewUserHandler(service)
    // ...
}
```

**优点**：一目了然，IDE 跳转无障碍。  
**缺点**：10 层依赖时写起来痛苦。

### wire 示例（伪代码）

```go
// wire.go
//go:build wireinject
func InitializeApp() (*App, error) {
    wire.Build(
        ProvideDB,
        ProvideRepo,
        ProvideService,
        ProvideHandler,
        NewApp,
    )
    return nil, nil
}
// wire 生成 wire_gen.go，自动按依赖顺序调用构造函数
```

**优点**：编译期生成，无运行时开销，错误提前发现。  
**缺点**：需要学 `wire.Build`、`wire.Bind` 语法。

### 选型建议

1. **小项目手动注入即可**：5–10 个依赖时手写比学 wire 快。
2. **依赖层级深、构造繁琐选 wire**：编译期保证类型安全。
3. **需要生命周期管理（启动/关闭顺序）选 fx**：适合微服务框架。

---

## 30.4 参数校验

HTTP 请求、配置文件的参数校验是常见需求。Go 社区的主流方案是标签驱动的 `validator`。

### 常见校验方案

| 方案 | 优点 | 缺点 | 适用场景 |
| --- | --- | --- | --- |
| **手动校验** | 显式、灵活、无依赖 | 重复代码、错误信息不统一 | 校验规则简单 |
| **validator** | 标签驱动、规则丰富、错误信息可定制 | 标签嵌套复杂时可读性差 | 规则复杂、需要统一错误格式 |

### 手动校验示例

```go
if req.Email == "" {
    return errors.New("email 不能为空")
}
if len(req.Password) < 8 {
    return errors.New("密码至少 8 位")
}
```

**优点**：逻辑清晰。  
**缺点**：10 个字段时代码臃肿。

### validator 示例（伪代码）

```go
type User struct {
    Email    string `validate:"required,email"`
    Password string `validate:"required,min=8"`
    Age      int    `validate:"gte=0,lte=120"`
}

validate := validator.New()
err := validate.Struct(user)
// 自动校验所有字段，返回结构化错误
```

**优点**：声明式，规则集中。  
**缺点**：复杂规则（跨字段依赖）时标签难读。

### 选型建议

1. **校验规则简单用手动校验**：3–5 个字段时手写更快。
2. **规则复杂、需要统一错误格式选 validator**：省去重复代码。

---

## 30.5 测试库

Go 的 `testing` 标准库功能完整，但断言不友好（需要手写 `if err != nil { t.Fatal() }`）。社区的 `testify` 和 `gomock` 填补了断言和 mock 的空白。

### 常见测试库

| 库 | 优点 | 缺点 | 适用场景 |
| --- | --- | --- | --- |
| **testing** | 零依赖、稳定 | 断言不友好 | 简单测试 |
| **testify/assert** | 断言友好、suite 支持 | 引入依赖 | 需要友好断言 |
| **gomock** | 官方 mock 工具、接口 mock 代码生成 | 需要生成代码、接口优先设计 | 单元测试、需要 mock 外部依赖 |

### testify 示例（伪代码）

```go
import "github.com/stretchr/testify/assert"

func TestAdd(t *testing.T) {
    result := Add(1, 2)
    assert.Equal(t, 3, result)
    assert.NoError(t, err)
}
```

**优点**：比 `if result != 3 { t.Errorf() }` 更简洁。

### gomock 示例（伪代码）

```go
// 生成 mock
//go:generate mockgen -source=user_repo.go -destination=mock_user_repo.go

func TestService(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    mockRepo := NewMockUserRepo(ctrl)
    mockRepo.EXPECT().GetUser(1).Return(&User{Name: "Alice"}, nil)
    // 测试 service 逻辑
}
```

**优点**：隔离外部依赖（数据库、HTTP）。

### 选型建议

1. **简单测试用标准库**：表驱动测试已够用。
2. **需要友好断言选 testify/assert**：提升可读性。
3. **需要 mock 接口选 gomock**：单元测试的标配。

---

## 30.6 日志、配置与 CLI 库

日志、配置、CLI 是工程化的基础设施。Go 1.21 引入了 `log/slog` 结构化日志，大幅缩小了第三方日志库的必要性。

### 日志库对比

| 库 | 优点 | 缺点 | 适用场景 |
| --- | --- | --- | --- |
| **log/slog** | 结构化日志、零依赖、性能好 | Go 1.21+ | 新项目优先 |
| **zap** | 性能极高、字段类型化 | API 偏底层 | 对性能极度敏感 |
| **logrus** | API 友好、Hook 丰富 | 性能较低、维护不活跃 | 老项目迁移 |

### slog 示例

```go
slog.Info("user login", "user_id", 42, "ip", "127.0.0.1")
// 输出：time=2024-01-01T10:00:00 level=INFO msg="user login" user_id=42 ip=127.0.0.1
```

### 配置库对比

| 库 | 优点 | 缺点 | 适用场景 |
| --- | --- | --- | --- |
| **flag** | 简单、零依赖 | 只支持命令行参数 | 简单配置 |
| **viper** | 支持多格式（JSON/YAML/TOML）、环境变量、热重载 | 依赖重 | 多环境配置 |

### CLI 库对比

| 库 | 优点 | 缺点 | 适用场景 |
| --- | --- | --- | --- |
| **flag** | 简单、零依赖 | 不支持子命令 | 单命令工具 |
| **cobra** | 子命令支持、自动生成帮助 | 引入依赖 | kubectl / docker 风格 CLI |

### 选型建议

- **日志**：Go 1.21+ 用 slog，否则用 zap。
- **配置**：简单用 flag，多格式用 viper。
- **CLI**：无子命令用 flag，有子命令用 cobra。

---

## 30.7 gRPC、WebSocket 与消息队列概览

微服务、实时通信、异步解耦场景下，你会接触 gRPC、WebSocket、消息队列。

### gRPC

- **库**：`google.golang.org/grpc`
- **用途**：微服务间通信、强类型 RPC
- **优点**：Protobuf 序列化高效、双向流、负载均衡
- **缺点**：浏览器支持有限（需 grpc-web）

### WebSocket

- **库**：`github.com/gorilla/websocket`、`nhooyr.io/websocket`
- **用途**：实时推送、聊天、游戏
- **优点**：双向通信、浏览器原生支持
- **缺点**：需处理重连、心跳

### 消息队列

| 队列 | 优点 | 适用场景 |
| --- | --- | --- |
| **NATS** | 轻量、高性能、云原生 | 微服务解耦、事件驱动 |
| **RabbitMQ** | 功能丰富、持久化、死信队列 | 传统企业、复杂路由 |
| **Kafka** | 高吞吐、日志存储、事件溯源 | 大数据、日志收集 |

---

## 30.8 选型原则与避免过度依赖

引入第三方库是权衡：收益（开发效率）vs 成本（依赖管理、框架锁定、安全漏洞）。

### 选型原则

1. **优先标准库**：零依赖、长期稳定、Go 团队持续优化。
2. **评估维护活跃度**：最近提交、issue 响应、发布频率。
3. **看社区规模**：GitHub stars、下载量、文档质量、生产案例。
4. **检查 Go 版本要求**：避免引入过新或过旧的库。
5. **避免深度绑定**：接口抽象、依赖倒置（见下文）。

### 避免过度依赖

1. **不要为了「少写一行代码」引入库**  
   示例：字符串处理用标准库 `strings`，不需要 lodash 风格库。

2. **警惕「框架锁定」**  
   错误：
   ```go
   func GetUser(c *gin.Context) {
       // 业务逻辑依赖 gin.Context
   }
   ```
   正确：
   ```go
   type UserService interface {
       GetUser(ctx context.Context, id int) (*User, error)
   }
   // HTTP 层只做适配
   func ginGetUser(svc UserService) gin.HandlerFunc {
       return func(c *gin.Context) {
           user, err := svc.GetUser(c.Request.Context(), ...)
           // 返回 JSON
       }
   }
   ```

3. **大型依赖评估「值不值」**  
   示例：只用 viper 读 YAML，可以换成 `gopkg.in/yaml.v3`（体积小 10 倍）。

4. **定期清理无用依赖**  
   命令：`go mod tidy`

### 依赖倒置实战

**问题**：业务逻辑依赖 gin，换框架时需要改 100 个文件。

**解决**：业务逻辑定义接口，HTTP 层实现适配器。

```go
// domain/user.go（业务层，不依赖任何框架）
type UserService struct {
    repo UserRepo
}
func (s *UserService) GetUser(ctx context.Context, id int) (*User, error) {
    return s.repo.FindByID(ctx, id)
}

// api/gin_adapter.go（适配层）
func NewGinHandler(svc *domain.UserService) *gin.Engine {
    r := gin.Default()
    r.GET("/users/:id", func(c *gin.Context) {
        id, _ := strconv.Atoi(c.Param("id"))
        user, err := svc.GetUser(c.Request.Context(), id)
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, user)
    })
    return r
}
```

**收益**：换成 echo / fiber / 标准库时，只需重写 `api/` 层，`domain/` 层不动。

---

## 真实报错与常见陷阱

### 报错 1：引入库后 `go mod tidy` 失败

```bash
go: finding module for package github.com/xxx/yyy
go: example.com/myapp imports
    github.com/xxx/yyy: cannot find module providing package github.com/xxx/yyy
```

**原因**：库名拼错、版本不存在、`GOPROXY` 访问失败。

**解决**：
1. 检查库名（go.dev 搜索）
2. 设置国内代理：`go env -w GOPROXY=https://goproxy.cn,direct`
3. 指定版本：`go get github.com/xxx/yyy@v1.2.3`

### 报错 2：框架升级后 API 不兼容

```bash
# 升级 gin v1.8 → v1.9
./handler.go:42: c.Writer.WriteString undefined (type gin.ResponseWriter has no field or method WriteString)
```

**原因**：大版本升级可能破坏 API。

**解决**：
1. 查看 CHANGELOG / 迁移指南
2. 锁定版本：`go.mod` 中 `require github.com/gin-gonic/gin v1.8.2`
3. 用接口抽象隔离框架（依赖倒置）

### 报错 3：ORM 生成低效 SQL

```bash
# GORM 查询
db.Where("age > ?", 18).Find(&users)
// 生成 SQL：SELECT * FROM users WHERE age > 18
// 扫描所有字段，包含 200KB 的 profile_json
```

**原因**：ORM 默认 `SELECT *`，不支持部分字段。

**解决**：显式指定字段 `db.Select("id, name").Where(...).Find(&users)`，或改用 SQL。

### 报错 4：依赖注入循环依赖

```bash
# wire 报错
wire: cycle detected:
    *UserService -> *AuthService -> *UserService
```

**原因**：A 依赖 B，B 依赖 A。

**解决**：引入接口打破循环：
```go
type UserGetter interface {
    GetUser(id int) (*User, error)
}
type AuthService struct {
    userGetter UserGetter // 依赖接口，不依赖 *UserService
}
```

### 报错 5：viper 配置热重载导致并发读写 panic

```bash
fatal error: concurrent map read and map write
```

**原因**：viper 的 `WatchConfig` 回调中修改配置，业务代码同时读取。

**解决**：加锁或用 `viper.AllSettings()` 快照。

### 报错 6：第三方库依赖过旧的 Go 版本

```bash
go: github.com/xxx/yyy@v1.0.0 requires go >= 1.18
```

**原因**：项目用 Go 1.17，库要求 1.18。

**解决**：
1. 升级 Go 版本
2. 降级库版本：`go get github.com/xxx/yyy@v0.9.0`
3. 换其他库

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 引入库后编译变慢 | cgo 依赖、大量代码生成 | 检查 `go.mod`，避免 cgo 库（如 go-sqlite3） |
| 框架升级后 panic | API 不兼容 | 查看 CHANGELOG，锁定版本 |
| ORM 查询慢 | `SELECT *`、N+1 查询 | 显式指定字段、用 `Preload` |
| 依赖冲突 | 同一库多个版本 | `go mod why`、`replace` 指令 |
| 测试失败"找不到 mock 文件" | 忘记 `go generate` | 运行 `go generate ./...` |
| viper 读不到环境变量 | 未调用 `AutomaticEnv()` | 先 `viper.AutomaticEnv()`，再读配置 |
| gin 的 `c.Query` 返回空 | 参数名大小写错误 | 检查 URL 参数名（区分大小写） |
| wire 生成代码 git 冲突 | 多人同时修改 `wire.go` | 先 `go generate`，再提交 |
| 第三方库 panic 无法 recover | 库内部 panic | 在调用前 `defer recover()`，或换库 |

---

## 练习

### 第 1 题

你在写一个 REST API，需要路由、JSON 绑定、中间件。你会选 gin 还是标准库？说明理由。

::: details 第 1 题参考答案

**答案**：取决于项目规模和团队经验。

**选标准库**（推荐）：
- 理由：Go 1.22 的路由增强（`GET /users/{id}`）已覆盖 80% 场景，零依赖，长期稳定。
- 代价：中间件、错误处理需自己封装（10–20 行），但可复用。

**选 gin**：
- 理由：团队熟悉 gin、需要快速开发、已有 gin 中间件生态。
- 代价：引入依赖、框架锁定（业务逻辑依赖 `gin.Context`）。

**建议**：新项目优先标准库，用接口抽象隔离框架（30.8 节的依赖倒置）。
:::

### 第 2 题

你的项目需要查询用户表、订单表（多表 JOIN），你会选 GORM 还是 database/sql？说明理由。

::: details 第 2 题参考答案

**答案**：优先 database/sql + sqlx。

**理由**：
- 多表 JOIN 用 GORM 难写（链式 API 表达复杂查询不直观），生成的 SQL 可能低效。
- database/sql 手写 SQL，性能无损，调试方便（打印 SQL 即可）。
- sqlx 的 `StructScan` 减少模板代码：`db.Select(&orders, "SELECT ... FROM orders JOIN users ...")`。

**GORM 的适用场景**：简单 CRUD、需要自动 migration、关联加载（Preload）。

**更优选择**：sqlc（写 SQL，生成类型安全的 Go 代码）。
:::

### 第 3 题

下面的代码有什么问题？如何改进？

```go
func GetUser(c *gin.Context) {
    id, _ := strconv.Atoi(c.Param("id"))
    db := c.MustGet("db").(*sql.DB)
    var user User
    db.QueryRow("SELECT * FROM users WHERE id = ?", id).Scan(&user.ID, &user.Name)
    c.JSON(200, user)
}
```

::: details 第 3 题参考答案

**问题**：
1. **框架锁定**：业务逻辑依赖 `gin.Context`，换框架时需大量改动。
2. **全局状态**：通过 `c.MustGet("db")` 传递 `db`，测试时难以 mock。
3. **错误处理缺失**：`QueryRow` 可能失败，`Scan` 可能失败，都未检查。

**改进**：
```go
// domain/user.go（业务层）
type UserRepo interface {
    GetUser(ctx context.Context, id int) (*User, error)
}
type UserService struct {
    repo UserRepo
}
func (s *UserService) GetUser(ctx context.Context, id int) (*User, error) {
    return s.repo.GetUser(ctx, id)
}

// api/handler.go（适配层）
func NewHandler(svc *UserService) gin.HandlerFunc {
    return func(c *gin.Context) {
        id, _ := strconv.Atoi(c.Param("id"))
        user, err := svc.GetUser(c.Request.Context(), id)
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, user)
    }
}
```

**收益**：业务逻辑可测试、可复用、与框架解耦。
:::

### 第 4 题

你的项目用 viper 读 YAML 配置，但 `go.mod` 显示 viper 引入了 30 个间接依赖。你会怎么办?

::: details 第 4 题参考答案

**答案**：评估是否真的需要 viper。

**如果只用到读 YAML**：
- 换成 `gopkg.in/yaml.v3`（零间接依赖，体积小 10 倍）。
- 示例：
  ```go
  var config Config
  data, _ := os.ReadFile("config.yaml")
  yaml.Unmarshal(data, &config)
  ```

**如果需要环境变量覆盖、热重载**：
- 保留 viper，但锁定版本避免意外升级。

**原则**：不要为了"省一行代码"引入大型依赖。
:::

### 第 5 题

你写了一个库，需要打日志。你会用 slog、zap 还是 logrus？

::: details 第 5 题参考答案

**答案**：**不要在库里硬编码日志库**。

**原因**：库的使用者可能用其他日志库，你强制引入 zap 会导致依赖冲突。

**正确做法**：
1. **定义日志接口**：
   ```go
   type Logger interface {
       Info(msg string, keysAndValues ...any)
       Error(msg string, err error)
   }
   ```
2. **库接受 Logger 参数**：
   ```go
   func NewClient(logger Logger) *Client { ... }
   ```
3. **使用者传入适配器**：
   ```go
   client := NewClient(&SlogAdapter{slog.Default()})
   ```

**标准库示例**：`database/sql` 不打日志，只返回 `error`，由调用方决定是否记录。
:::

### 第 6 题

你的团队在 gin 和 echo 之间犹豫，你会怎么选？给出决策树。

::: details 第 6 题参考答案

**决策树**：

```
1. 是否需要 Web 框架？
   → 否：用标准库（零依赖，Go 1.22 路由增强够用）
   → 是：继续

2. 是否需要 HTTP/2 服务端推送？
   → 是：echo（原生支持）
   → 否：继续

3. 团队是否熟悉 gin？
   → 是：gin（生态最成熟）
   → 否：继续

4. 是否追求极致性能？
   → 是：fiber（但不兼容标准库）
   → 否：gin（社区大、中间件多）
```

**建议**：新项目优先标准库，除非有明确的框架需求（如 gin 的中间件生态）。
:::

---

## 小结

本章介绍了 Go 常用第三方生态的选型思路，核心要点：

1. **优先标准库**：Go 的标准库覆盖面广、性能好、长期稳定，许多场景下不需要第三方库。
2. **Web 框架**：新项目优先标准库（Go 1.22+），需要快速开发选 gin。
3. **ORM**：简单查询用 database/sql，SQL 优先选 sqlc，快速原型选 GORM。
4. **依赖注入**：小项目手动注入，复杂项目选 wire。
5. **日志**：Go 1.21+ 用 slog，否则用 zap；库代码不要硬编码日志库。
6. **选型原则**：评估维护活跃度、社区规模、Go 版本兼容、依赖成本。
7. **避免过度依赖**：不要为了"少写一行代码"引入库，警惕框架锁定，用接口抽象隔离框架。
8. **依赖倒置**：业务逻辑定义接口，框架层实现适配器，降低耦合。

下一章将进入工程实践篇，讲解**项目结构、分层架构与可测试性设计**，把前面学到的技术整合成可维护的 Go 项目。
