# 第 28 章 · 数据库编程

上一章我们学会了用 `pprof` 和 `trace` 分析性能瓶颈。现实项目中，数据库往往是性能的关键节点：不正确的连接池配置会导致连接耗尽，忘记关闭 `rows` 会泄漏连接，事务未提交会锁死表。本章聚焦 `database/sql` 标准接口与 SQLite 实战，讲透连接池参数、CRUD 操作、预处理语句、NULL 处理、事务管理、context 超时和错误处理，帮你写出健壮、高效的数据库代码。

本章配套代码在 `internal/chapter/go28_database/`，执行 `go run ./cmd/go-learn` 可以看到全部输出。

---

## 28.1 database/sql 与驱动注册

Go 的 `database/sql` 包提供了统一的数据库访问接口，具体实现由第三方驱动提供。常见驱动：

- **PostgreSQL**：`github.com/lib/pq`（纯 Go）、`github.com/jackc/pgx`（高性能）
- **MySQL**：`github.com/go-sql-driver/mysql`
- **SQLite**：`github.com/mattn/go-sqlite3`（cgo）、`modernc.org/sqlite`（纯 Go）
- **SQL Server**：`github.com/denisenkom/go-mssqldb`

驱动通过 `init` 函数注册到 `sql` 包，使用时需要匿名导入：

```go
import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3" // 驱动注册
)

db, err := sql.Open("sqlite3", ":memory:") // 第一个参数是驱动名
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

**关键要点**：

1. `sql.Open` **只初始化连接池**，不立即建立连接。
2. `db.Ping()` 才真正建立连接并验证可用性。
3. `*sql.DB` 是连接池，不是单个连接，可以并发使用，无需自己加锁。
4. `db.Close()` 关闭连接池，通常在 `main` 结束或应用关闭时调用。

**实测输出**：

```
数据库驱动：sqlite3（内存模式）
sql.Open 返回的 *sql.DB 是连接池，不是单个连接
db.Ping() 成功，连接池可用
```

---

## 28.2 连接池参数与 Ping

`*sql.DB` 是连接池，行为由四个参数控制：

```go
db.SetMaxOpenConns(10)                  // 最大打开连接数（默认无限）
db.SetMaxIdleConns(5)                   // 最大空闲连接数（默认 2）
db.SetConnMaxLifetime(30 * time.Minute) // 连接最大生存时间
db.SetConnMaxIdleTime(5 * time.Minute)  // 连接最大空闲时间
```

**参数说明**：

- **MaxOpenConns**：限制同时打开的连接总数。默认无限，生产环境应设置（通常与数据库服务器的 `max_connections` 对齐）。
- **MaxIdleConns**：空闲连接池大小。过小会频繁建连接，过大会浪费资源。默认 2，通常设为 `MaxOpenConns` 的一半。
- **ConnMaxLifetime**：连接的最大生存时间，超过后会关闭并重建。用于防止数据库服务器主动关闭长连接。
- **ConnMaxIdleTime**：连接的最大空闲时间，超过后会关闭。Go 1.15 新增。

**连接池状态**：

```go
stats := db.Stats()
fmt.Printf("OpenConnections=%d InUse=%d Idle=%d WaitCount=%d\n",
    stats.OpenConnections, stats.InUse, stats.Idle, stats.WaitCount)
```

- `OpenConnections`：当前打开的连接数
- `InUse`：正在使用的连接数
- `Idle`：空闲连接数
- `WaitCount`：因连接池满而等待的次数（非零说明连接不够用）

**实测输出**：

```
连接池配置：MaxOpen=10 MaxIdle=5
当前统计：OpenConnections=0 InUse=0 Idle=0
```

---

## 28.3 Query / QueryRow / Exec

`database/sql` 提供三个核心方法：

### 28.3.1 Exec：执行不返回行的语句

用于 `CREATE`、`INSERT`、`UPDATE`、`DELETE`：

```go
result, err := db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", "Alice", "alice@example.com")
if err != nil {
    log.Fatal(err)
}

lastID, _ := result.LastInsertId()   // 最后插入的自增 ID（SQLite 支持，PostgreSQL 不支持）
affected, _ := result.RowsAffected() // 受影响的行数
fmt.Printf("LastInsertId=%d RowsAffected=%d\n", lastID, affected)
```

**注意**：

- `LastInsertId()` 不是所有数据库都支持（PostgreSQL 需要用 `RETURNING id`）。
- `RowsAffected()` 对 `SELECT` 无效。

### 28.3.2 QueryRow：查询单行

用于期望返回一行（或零行）的查询：

```go
var name, email string
err := db.QueryRow("SELECT name, email FROM users WHERE id = ?", 1).Scan(&name, &email)
if err == sql.ErrNoRows {
    fmt.Println("未找到记录")
} else if err != nil {
    log.Fatal(err)
}
fmt.Printf("name=%s email=%s\n", name, email)
```

**关键点**：

- `QueryRow` 找不到记录时，`Scan` 返回 `sql.ErrNoRows`（不是 `QueryRow` 本身返回）。
- 不需要手动 `Close`，`QueryRow` 内部会自动关闭。

### 28.3.3 Query：查询多行

用于返回多行的查询：

```go
rows, err := db.Query("SELECT id, name, email FROM users ORDER BY id")
if err != nil {
    log.Fatal(err)
}
defer rows.Close() // 必须关闭

for rows.Next() {
    var id int
    var name, email string
    if err := rows.Scan(&id, &name, &email); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("id=%d name=%s email=%s\n", id, name, email)
}

if err := rows.Err(); err != nil { // 检查遍历过程中的错误
    log.Fatal(err)
}
```

**必须记住**：

1. `rows.Close()` **必须调用**，否则连接不会归还到连接池，最终耗尽连接。
2. `rows.Err()` 检查遍历过程中的错误（`Next()` 返回 `false` 可能是遍历完成，也可能是出错）。
3. `Scan` 的参数顺序必须与 `SELECT` 列顺序一致。

**实测输出**：

```
INSERT 成功：LastInsertId=1 RowsAffected=1
QueryRow 查询 id=1：name=Alice email=alice@example.com
Query 查询所有用户：
  id=1 name=Alice email=alice@example.com
  id=2 name=Bob email=bob@example.com
  id=3 name=Charlie email=charlie@example.com
```

---

## 28.4 预处理语句与 SQL 注入

### 28.4.1 SQL 注入陷阱

**错误示范**（拼接 SQL）：

```go
unsafeInput := "Laptop' OR '1'='1"
unsafeQuery := fmt.Sprintf("SELECT * FROM products WHERE name = '%s'", unsafeInput)
// 实际执行：SELECT * FROM products WHERE name = 'Laptop' OR '1'='1'
// → 绕过条件，返回所有记录
```

**正确做法**（占位符）：

```go
rows, err := db.Query("SELECT name, price FROM products WHERE name = ?", userInput)
```

`database/sql` 会自动转义参数，防止注入。**占位符语法因驱动而异**：

- PostgreSQL：`$1`、`$2`
- MySQL / SQLite：`?`
- SQL Server：`@p1`、`@p2`

### 28.4.2 预处理语句

多次执行同一 SQL 时，可以显式预处理：

```go
stmt, err := db.Prepare("SELECT name, price FROM products WHERE price > ?")
if err != nil {
    log.Fatal(err)
}
defer stmt.Close()

rows, _ := stmt.Query(50.0)
defer rows.Close()
// 第二次执行
rows2, _ := stmt.Query(100.0)
defer rows2.Close()
```

**何时用 `Prepare`**：

- 同一 SQL 执行多次（性能提升）。
- 循环插入大量数据。

**何时不需要**：

- `db.Query` / `Exec` 内部已经做了预处理，单次执行无需显式 `Prepare`。

**实测输出**：

```
不安全的 SQL（拼接）：SELECT name, price FROM products WHERE name = 'Laptop' OR '1'='1'
  → 直接拼接用户输入会导致 SQL 注入漏洞
安全的 SQL（占位符）：
  name=Laptop price=999.99
预处理语句查询 price > 50：
  name=Laptop price=999.99
```

---

## 28.5 NULL 处理与 sql.NullXxx

数据库的 `NULL` 无法直接 `Scan` 到 Go 的基本类型：

```go
var bio string
err := db.QueryRow("SELECT bio FROM authors WHERE name = ?", "Bob").Scan(&bio)
// 如果 bio 是 NULL，Scan 返回错误：sql: Scan error on column index 0, name "bio": converting NULL to string is unsupported
```

**解决方案**：使用 `sql.NullXxx` 类型：

```go
var bio sql.NullString
err := db.QueryRow("SELECT bio FROM authors WHERE name = ?", "Bob").Scan(&bio)
if err != nil {
    log.Fatal(err)
}

if bio.Valid {
    fmt.Printf("bio=%s\n", bio.String)
} else {
    fmt.Println("bio=NULL")
}
```

**常用 NULL 类型**：

- `sql.NullString`
- `sql.NullInt64` / `NullInt32`
- `sql.NullFloat64`
- `sql.NullBool`
- `sql.NullTime`（Go 1.13+）

**实测输出**：

```
Scan NULL 到 string 失败（预期）：sql: Scan error on column index 1, name "bio": converting NULL to string is unsupported
name=Bob bio=NULL
sql 包提供的 NULL 类型：NullString NullInt64 NullFloat64 NullBool NullTime NullInt32 NullInt16 NullByte
```

---

## 28.6 事务与回滚

事务保证多个操作的原子性（要么全成功，要么全失败）。

### 28.6.1 基本用法

```go
tx, err := db.Begin()
if err != nil {
    log.Fatal(err)
}

_, err = tx.Exec("UPDATE accounts SET balance = balance - ? WHERE name = ?", 30.0, "Alice")
if err != nil {
    tx.Rollback() // 回滚
    log.Fatal(err)
}

_, err = tx.Exec("UPDATE accounts SET balance = balance + ? WHERE name = ?", 30.0, "Bob")
if err != nil {
    tx.Rollback()
    log.Fatal(err)
}

if err := tx.Commit(); err != nil { // 提交
    log.Fatal(err)
}
```

**关键点**：

1. `Begin()` 开启事务，返回 `*sql.Tx`。
2. 在 `tx` 上调用 `Exec` / `Query`（不是 `db`）。
3. 出错时调用 `Rollback()`，成功时调用 `Commit()`。
4. `Commit` / `Rollback` 之后，`tx` 不可再用。

### 28.6.2 defer Rollback 惯用法

```go
tx, err := db.Begin()
if err != nil {
    return err
}
defer tx.Rollback() // Commit 后 Rollback 会返回 sql.ErrTxDone，可以忽略

// ... 业务逻辑 ...

return tx.Commit()
```

**实测输出**：

```
事务提交成功（Alice -30, Bob +30）
转账后余额：
  Alice: 70.00
  Bob: 80.00
演示回滚：UPDATE 后 Rollback，余额未改变
```

---

## 28.7 context 超时与取消

`database/sql` 的所有方法都有 `Context` 版本：

- `db.QueryContext(ctx, ...)`
- `db.ExecContext(ctx, ...)`
- `db.QueryRowContext(ctx, ...)`
- `db.BeginTx(ctx, opts)`

**超时示例**：

```go
ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()

rows, err := db.QueryContext(ctx, "SELECT * FROM logs")
if err != nil {
    fmt.Println(err) // context deadline exceeded
}
```

**事务 context**：

```go
ctx := context.Background()
tx, err := db.BeginTx(ctx, nil)
// ctx 取消时，事务会自动回滚
```

**实测输出**：

```
  id=1 message=log1
  id=2 message=log2
超时示例：context deadline exceeded
```

---

## 28.8 sql.ErrNoRows 与错误处理

`QueryRow` 找不到记录时，`Scan` 返回 `sql.ErrNoRows`：

```go
var value string
err := db.QueryRow("SELECT value FROM settings WHERE key = ?", "lang").Scan(&value)
if err != nil {
    if errors.Is(err, sql.ErrNoRows) {
        fmt.Println("key 不存在")
    } else {
        log.Fatal(err)
    }
}
```

**重要区分**：

- `sql.ErrNoRows`：**不是错误**，是业务层的「找不到」，通常返回 404 或默认值。
- 其他错误：数据库连接失败、SQL 语法错误、权限不足等，应该记日志或返回 500。

**仓储层惯用法**：

```go
func GetValue(db *sql.DB, key string) (string, error) {
    var val string
    err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
    if errors.Is(err, sql.ErrNoRows) {
        return "", fmt.Errorf("key %q 不存在", key)
    }
    return val, err
}
```

**实测输出**：

```
查询存在的 key: theme=dark
查询不存在的 key: 返回 sql.ErrNoRows（不是数据库错误，是业务层的「找不到」）
getValue("theme"): dark (err=<nil>)
getValue("lang"): (err=key "lang" 不存在)
```

---

## 6 个真实报错怎么读

### 报错 1：忘记关闭 rows

```
panic: runtime error: invalid memory address or nil pointer dereference
```

**原因**：`rows.Close()` 未调用，连接未归还，连接池耗尽后新请求拿不到连接。

**修复**：在 `Query` 后立即 `defer rows.Close()`。

### 报错 2：Scan 参数数量不匹配

```
sql: expected 3 destination arguments in Scan, not 2
```

**原因**：`SELECT` 返回 3 列，`Scan` 只提供 2 个变量。

**修复**：确保 `Scan` 参数与列数一致，或用 `sql.RawBytes` 跳过不需要的列。

### 报错 3：在关闭的 tx 上操作

```
sql: transaction has already been committed or rolled back
```

**原因**：`Commit` 后又调用了 `tx.Exec`。

**修复**：事务结束后不再使用 `tx`。

### 报错 4：NULL 转基本类型

```
sql: Scan error: converting NULL to string is unsupported
```

**原因**：数据库列为 `NULL`，尝试 `Scan` 到 `string`。

**修复**：使用 `sql.NullString`。

### 报错 5：连接池耗尽

```
pq: sorry, too many clients already
```

**原因**：`MaxOpenConns` 超过数据库服务器的 `max_connections`。

**修复**：调小 `MaxOpenConns` 或增加数据库的连接限制。

### 报错 6：SQL 语法错误

```
near "SELET": syntax error
```

**原因**：SQL 拼写错误（`SELET` 应为 `SELECT`）。

**修复**：检查 SQL 语句，用 IDE 语法高亮或数据库客户端验证。

---

## 常见坑速查

| 现象 | 原因 | 解法 |
| --- | --- | --- |
| 连接池耗尽 | `rows.Close()` 未调用 | 在 `Query` 后立即 `defer rows.Close()` |
| `Scan` 失败 | 列数与参数不匹配 | 确保 `Scan` 参数与 `SELECT` 列一一对应 |
| NULL 转换错误 | NULL 无法转基本类型 | 使用 `sql.NullXxx` |
| 事务未生效 | 在 `db` 而非 `tx` 上执行 | 事务内操作必须用 `tx.Exec` / `tx.Query` |
| `LastInsertId` 为 0 | 数据库不支持 | PostgreSQL 用 `RETURNING id`，MySQL/SQLite 支持 |
| context 取消但查询继续 | 未传 `Context` 版本方法 | 用 `QueryContext` 而非 `Query` |
| SQL 注入 | 拼接用户输入 | 始终用占位符（`?` 或 `$1`） |
| 预处理语句泄漏 | `stmt.Close()` 未调用 | `defer stmt.Close()` |
| 查询慢 | 未建索引或全表扫描 | 用 `EXPLAIN` 分析查询计划 |

---

## 练习

### 第 1 题

写一个函数 `CountUsers(db *sql.DB) (int, error)`，返回 `users` 表的行数。

::: details 第 1 题参考答案

```go
func CountUsers(db *sql.DB) (int, error) {
    var count int
    err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
    return count, err
}
```

**为什么这样写**：`COUNT` 返回单行单列，用 `QueryRow` + `Scan` 最简洁。

:::

### 第 2 题

写一个函数 `GetUserByID(db *sql.DB, id int) (*User, error)`，找不到时返回自定义错误，而不是 `sql.ErrNoRows`。

::: details 第 2 题参考答案

```go
type User struct {
    ID    int
    Name  string
    Email string
}

func GetUserByID(db *sql.DB, id int) (*User, error) {
    var u User
    err := db.QueryRow("SELECT id, name, email FROM users WHERE id = ?", id).Scan(&u.ID, &u.Name, &u.Email)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, fmt.Errorf("用户 ID=%d 不存在", id)
    }
    if err != nil {
        return nil, err
    }
    return &u, nil
}
```

**为什么这样写**：业务层不应暴露 `sql.ErrNoRows`，转成自定义错误更清晰。

:::

### 第 3 题

写一个事务函数 `TransferMoney(db *sql.DB, fromID, toID int, amount float64) error`，从账户 A 转账给账户 B，余额不足时返回错误。

::: details 第 3 题参考答案

```go
func TransferMoney(db *sql.DB, fromID, toID int, amount float64) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // 检查余额
    var balance float64
    err = tx.QueryRow("SELECT balance FROM accounts WHERE id = ?", fromID).Scan(&balance)
    if err != nil {
        return fmt.Errorf("查询余额失败: %w", err)
    }
    if balance < amount {
        return fmt.Errorf("余额不足：当前 %.2f，需要 %.2f", balance, amount)
    }

    // 扣款
    _, err = tx.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ?", amount, fromID)
    if err != nil {
        return fmt.Errorf("扣款失败: %w", err)
    }

    // 入账
    _, err = tx.Exec("UPDATE accounts SET balance = balance + ? WHERE id = ?", amount, toID)
    if err != nil {
        return fmt.Errorf("入账失败: %w", err)
    }

    return tx.Commit()
}
```

**为什么这样写**：

1. `defer tx.Rollback()` 确保出错时回滚。
2. 先查余额，不足则提前返回（避免无效的事务）。
3. 用 `fmt.Errorf` 包装错误，保留上下文。

:::

### 第 4 题

`db.Query` 返回的 `rows` 如果不调用 `rows.Close()`，会发生什么？写代码验证。

::: details 第 4 题参考答案

```go
func LeakConnections(db *sql.DB) {
    for i := 0; i < 20; i++ {
        rows, err := db.Query("SELECT * FROM users")
        if err != nil {
            log.Fatal(err)
        }
        // 故意不关闭 rows
        fmt.Printf("查询 %d 次\n", i+1)
    }

    stats := db.Stats()
    fmt.Printf("OpenConnections=%d InUse=%d Idle=%d\n",
        stats.OpenConnections, stats.InUse, stats.Idle)
}
```

**现象**：`InUse` 持续增长，最终达到 `MaxOpenConns`，后续查询阻塞或超时。

**为什么会泄漏**：`rows` 持有连接，不关闭则连接不归还到连接池。

:::

### 第 5 题

写一个函数 `BatchInsert(db *sql.DB, names []string) error`，批量插入用户（用预处理语句优化）。

::: details 第 5 题参考答案

```go
func BatchInsert(db *sql.DB, names []string) error {
    stmt, err := db.Prepare("INSERT INTO users (name) VALUES (?)")
    if err != nil {
        return err
    }
    defer stmt.Close()

    for _, name := range names {
        if _, err := stmt.Exec(name); err != nil {
            return fmt.Errorf("插入 %s 失败: %w", name, err)
        }
    }

    return nil
}
```

**为什么这样写**：预处理语句只编译一次 SQL，循环执行参数，比每次 `db.Exec` 更高效。

**进一步优化**：用事务包裹，批量提交：

```go
tx, _ := db.Begin()
defer tx.Rollback()
stmt, _ := tx.Prepare("INSERT INTO users (name) VALUES (?)")
for _, name := range names {
    stmt.Exec(name)
}
return tx.Commit()
```

:::

### 第 6 题

`db.QueryRow` 和 `db.Query` 返回空结果时的错误有什么不同？

::: details 第 6 题参考答案

**`QueryRow`**：

```go
err := db.QueryRow("SELECT * FROM users WHERE id = 999").Scan(&user)
// err == sql.ErrNoRows
```

**`Query`**：

```go
rows, err := db.Query("SELECT * FROM users WHERE id = 999")
// err == nil（查询成功，只是没有行）
defer rows.Close()
for rows.Next() {
    // 不会进入循环
}
// rows.Err() == nil
```

**区别**：

- `QueryRow` 期望有一行，找不到返回 `sql.ErrNoRows`。
- `Query` 允许零行，不报错，通过 `rows.Next()` 判断是否有数据。

:::

---

## 小结

本章讲解了 Go 数据库编程的核心技能：

1. `database/sql` 是标准接口，驱动通过匿名导入注册。
2. `*sql.DB` 是连接池，需要配置 `MaxOpenConns` / `MaxIdleConns` / `ConnMaxLifetime`。
3. `Exec` 用于写操作，`QueryRow` 查单行，`Query` 查多行，**`rows.Close()` 必须调用**。
4. 用占位符（`?` 或 `$1`）防止 SQL 注入，不要拼接用户输入。
5. NULL 值必须用 `sql.NullXxx` 类型处理。
6. 事务用 `Begin` / `Commit` / `Rollback`，出错时立即回滚。
7. 用 `Context` 版本方法控制超时（`QueryContext` / `ExecContext`）。
8. `sql.ErrNoRows` 不是错误，是业务层的「找不到」，应转成自定义错误。

下一章我们将学习命令行工具、日志与配置，构建生产级的 CLI 应用。
