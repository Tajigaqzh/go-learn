// Package go28_database 演示 database/sql 与数据库编程。
//
// 本章使用 SQLite 作为演示数据库（github.com/mattn/go-sqlite3），
// 涵盖连接池、CRUD、预处理语句、NULL 处理、事务、context 超时等主题。
package go28_database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite 驱动（纯 Go 实现可用 modernc.org/sqlite）
)

// Demo 是第 28 章的统一入口。
func Demo() {
	fmt.Println("\n========== go28_database: 数据库编程 ==========")

	// 28.1 database/sql 与驱动注册
	demo28_1()

	// 28.2 连接池参数与 Ping
	demo28_2()

	// 28.3 Query / QueryRow / Exec
	demo28_3()

	// 28.4 预处理语句与 SQL 注入
	demo28_4()

	// 28.5 NULL 处理与 sql.NullXxx
	demo28_5()

	// 28.6 事务与回滚
	demo28_6()

	// 28.7 context 超时与取消
	demo28_7()

	// 28.8 sql.ErrNoRows 与错误处理
	demo28_8()

	fmt.Println("\n========== 数据库编程演示结束 ==========")
}

// demo28_1 演示驱动注册与 sql.Open。
func demo28_1() {
	fmt.Println("\n--- 28.1 database/sql 与驱动注册 ---")

	// sql.Open 只是初始化连接池，不立即建立连接
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("sql.Open 失败: %v", err)
	}
	defer db.Close()

	// 真正建立连接需要 Ping
	if err := db.Ping(); err != nil {
		log.Fatalf("db.Ping 失败: %v", err)
	}

	fmt.Println("数据库驱动：sqlite3（内存模式）")
	fmt.Println("sql.Open 返回的 *sql.DB 是连接池，不是单个连接")
	fmt.Println("db.Ping() 成功，连接池可用")
}

// demo28_2 演示连接池参数配置。
func demo28_2() {
	fmt.Println("\n--- 28.2 连接池参数与 Ping ---")

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("sql.Open 失败: %v", err)
	}
	defer db.Close()

	// 配置连接池
	db.SetMaxOpenConns(10)                  // 最大打开连接数（默认无限）
	db.SetMaxIdleConns(5)                   // 最大空闲连接数（默认 2）
	db.SetConnMaxLifetime(30 * time.Minute) // 连接最大生存时间
	db.SetConnMaxIdleTime(5 * time.Minute)  // 连接最大空闲时间

	stats := db.Stats()
	fmt.Printf("连接池配置：MaxOpen=%d MaxIdle=%d\n", 10, 5)
	fmt.Printf("当前统计：OpenConnections=%d InUse=%d Idle=%d\n",
		stats.OpenConnections, stats.InUse, stats.Idle)
}

// demo28_3 演示 Query / QueryRow / Exec 的基本用法。
func demo28_3() {
	fmt.Println("\n--- 28.3 Query / QueryRow / Exec ---")

	db := mustOpenDB()
	defer db.Close()

	// Exec：执行不返回行的语句（CREATE / INSERT / UPDATE / DELETE）
	_, err := db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatalf("CREATE TABLE 失败: %v", err)
	}

	// 插入数据
	result, err := db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", "Alice", "alice@example.com")
	if err != nil {
		log.Fatalf("INSERT 失败: %v", err)
	}
	lastID, _ := result.LastInsertId()
	affected, _ := result.RowsAffected()
	fmt.Printf("INSERT 成功：LastInsertId=%d RowsAffected=%d\n", lastID, affected)

	db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", "Bob", "bob@example.com")
	db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", "Charlie", "charlie@example.com")

	// QueryRow：查询单行（找不到时 Scan 返回 sql.ErrNoRows）
	var name, email string
	err = db.QueryRow("SELECT name, email FROM users WHERE id = ?", 1).Scan(&name, &email)
	if err != nil {
		log.Fatalf("QueryRow 失败: %v", err)
	}
	fmt.Printf("QueryRow 查询 id=1：name=%s email=%s\n", name, email)

	// Query：查询多行
	rows, err := db.Query("SELECT id, name, email FROM users ORDER BY id")
	if err != nil {
		log.Fatalf("Query 失败: %v", err)
	}
	defer rows.Close() // 必须关闭 rows 释放连接

	fmt.Println("Query 查询所有用户：")
	for rows.Next() {
		var id int
		var name, email string
		if err := rows.Scan(&id, &name, &email); err != nil {
			log.Fatalf("Scan 失败: %v", err)
		}
		fmt.Printf("  id=%d name=%s email=%s\n", id, name, email)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("rows.Err: %v", err)
	}
}

// demo28_4 演示预处理语句与 SQL 注入防护。
func demo28_4() {
	fmt.Println("\n--- 28.4 预处理语句与 SQL 注入 ---")

	db := mustOpenDB()
	defer db.Close()

	db.Exec(`CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT, price REAL)`)
	db.Exec("INSERT INTO products (name, price) VALUES (?, ?), (?, ?)", "Laptop", 999.99, "Mouse", 19.99)

	// 错误示范：拼接 SQL（存在注入风险）
	unsafeInput := "Laptop' OR '1'='1"
	unsafeQuery := fmt.Sprintf("SELECT name, price FROM products WHERE name = '%s'", unsafeInput)
	fmt.Printf("不安全的 SQL（拼接）：%s\n", unsafeQuery)
	fmt.Println("  → 直接拼接用户输入会导致 SQL 注入漏洞")

	// 正确做法：使用占位符（database/sql 会自动转义）
	rows, err := db.Query("SELECT name, price FROM products WHERE name = ?", "Laptop")
	if err != nil {
		log.Fatalf("Query 失败: %v", err)
	}
	defer rows.Close()

	fmt.Println("安全的 SQL（占位符）：")
	for rows.Next() {
		var name string
		var price float64
		rows.Scan(&name, &price)
		fmt.Printf("  name=%s price=%.2f\n", name, price)
	}

	// Prepare：显式预处理语句（多次执行同一语句时更高效）
	stmt, err := db.Prepare("SELECT name, price FROM products WHERE price > ?")
	if err != nil {
		log.Fatalf("Prepare 失败: %v", err)
	}
	defer stmt.Close()

	rows2, _ := stmt.Query(50.0)
	defer rows2.Close()
	fmt.Println("预处理语句查询 price > 50：")
	for rows2.Next() {
		var name string
		var price float64
		rows2.Scan(&name, &price)
		fmt.Printf("  name=%s price=%.2f\n", name, price)
	}
}

// demo28_5 演示 NULL 值处理。
func demo28_5() {
	fmt.Println("\n--- 28.5 NULL 处理与 sql.NullXxx ---")

	db := mustOpenDB()
	defer db.Close()

	db.Exec(`CREATE TABLE authors (id INTEGER PRIMARY KEY, name TEXT, bio TEXT)`)
	db.Exec("INSERT INTO authors (name, bio) VALUES (?, ?)", "Alice", "Writer")
	db.Exec("INSERT INTO authors (name, bio) VALUES (?, NULL)", "Bob") // bio 为 NULL

	// 直接 Scan 到 string 会失败（NULL 无法转成 string）
	var name string
	var bio string
	err := db.QueryRow("SELECT name, bio FROM authors WHERE name = ?", "Bob").Scan(&name, &bio)
	if err != nil {
		fmt.Printf("Scan NULL 到 string 失败（预期）：%v\n", err)
	}

	// 使用 sql.NullString 处理 NULL
	var name2 string
	var bio2 sql.NullString
	err = db.QueryRow("SELECT name, bio FROM authors WHERE name = ?", "Bob").Scan(&name2, &bio2)
	if err != nil {
		log.Fatalf("Scan 失败: %v", err)
	}

	if bio2.Valid {
		fmt.Printf("name=%s bio=%s\n", name2, bio2.String)
	} else {
		fmt.Printf("name=%s bio=NULL\n", name2)
	}

	// 其他 NullXxx 类型
	fmt.Println("sql 包提供的 NULL 类型：NullString NullInt64 NullFloat64 NullBool NullTime NullInt32 NullInt16 NullByte")
}

// demo28_6 演示事务与回滚。
func demo28_6() {
	fmt.Println("\n--- 28.6 事务与回滚 ---")

	db := mustOpenDB()
	defer db.Close()

	db.Exec(`CREATE TABLE accounts (id INTEGER PRIMARY KEY, name TEXT, balance REAL)`)
	db.Exec("INSERT INTO accounts (name, balance) VALUES (?, ?), (?, ?)", "Alice", 100.0, "Bob", 50.0)

	// 转账示例：从 Alice 转 30 给 Bob
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Begin 失败: %v", err)
	}

	// 扣除 Alice 的余额
	_, err = tx.Exec("UPDATE accounts SET balance = balance - ? WHERE name = ?", 30.0, "Alice")
	if err != nil {
		tx.Rollback()
		log.Fatalf("UPDATE Alice 失败: %v", err)
	}

	// 增加 Bob 的余额
	_, err = tx.Exec("UPDATE accounts SET balance = balance + ? WHERE name = ?", 30.0, "Bob")
	if err != nil {
		tx.Rollback()
		log.Fatalf("UPDATE Bob 失败: %v", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		log.Fatalf("Commit 失败: %v", err)
	}

	fmt.Println("事务提交成功（Alice -30, Bob +30）")

	// 查询最终余额
	rows, _ := db.Query("SELECT name, balance FROM accounts ORDER BY name")
	defer rows.Close()
	fmt.Println("转账后余额：")
	for rows.Next() {
		var name string
		var balance float64
		rows.Scan(&name, &balance)
		fmt.Printf("  %s: %.2f\n", name, balance)
	}

	// 演示回滚
	tx2, _ := db.Begin()
	tx2.Exec("UPDATE accounts SET balance = 9999 WHERE name = ?", "Alice")
	tx2.Rollback() // 回滚，Alice 余额不变
	fmt.Println("演示回滚：UPDATE 后 Rollback，余额未改变")
}

// demo28_7 演示 context 超时与取消。
func demo28_7() {
	fmt.Println("\n--- 28.7 context 超时与取消 ---")

	db := mustOpenDB()
	defer db.Close()

	db.Exec(`CREATE TABLE logs (id INTEGER PRIMARY KEY, message TEXT)`)
	db.Exec("INSERT INTO logs (message) VALUES (?), (?)", "log1", "log2")

	// 设置 100ms 超时
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// QueryContext：带 context 的查询
	rows, err := db.QueryContext(ctx, "SELECT id, message FROM logs")
	if err != nil {
		fmt.Printf("QueryContext 失败: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var msg string
		rows.Scan(&id, &msg)
		fmt.Printf("  id=%d message=%s\n", id, msg)
	}

	// 演示超时（模拟慢查询）
	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel2()

	time.Sleep(5 * time.Millisecond) // 模拟延迟
	err = db.QueryRowContext(ctx2, "SELECT COUNT(*) FROM logs").Scan(new(int))
	if err != nil {
		fmt.Printf("超时示例：%v\n", err)
	}
}

// demo28_8 演示 sql.ErrNoRows 与错误处理。
func demo28_8() {
	fmt.Println("\n--- 28.8 sql.ErrNoRows 与错误处理 ---")

	db := mustOpenDB()
	defer db.Close()

	db.Exec(`CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT)`)
	db.Exec("INSERT INTO settings (key, value) VALUES (?, ?)", "theme", "dark")

	// 查询存在的记录
	var value string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", "theme").Scan(&value)
	if err != nil {
		log.Fatalf("QueryRow 失败: %v", err)
	}
	fmt.Printf("查询存在的 key: theme=%s\n", value)

	// 查询不存在的记录
	err = db.QueryRow("SELECT value FROM settings WHERE key = ?", "lang").Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Println("查询不存在的 key: 返回 sql.ErrNoRows（不是数据库错误，是业务层的「找不到」）")
		} else {
			log.Fatalf("QueryRow 失败: %v", err)
		}
	}

	// 统一的查询函数
	getValue := func(db *sql.DB, key string) (string, error) {
		var val string
		err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("key %q 不存在", key)
		}
		return val, err
	}

	v, err := getValue(db, "theme")
	fmt.Printf("getValue(\"theme\"): %s (err=%v)\n", v, err)

	_, err = getValue(db, "lang")
	fmt.Printf("getValue(\"lang\"): (err=%v)\n", err)
}

// mustOpenDB 创建内存 SQLite 数据库（辅助函数）。
func mustOpenDB() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("sql.Open 失败: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("db.Ping 失败: %v", err)
	}
	return db
}
