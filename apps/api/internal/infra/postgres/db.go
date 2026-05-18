// Package postgres 提供 pgx stdlib 驱动的 *sql.DB 初始化封装。
package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // 注册 "pgx" 驱动名
)

// ErrEmptyDSN 当 DSN 为空时返回。
var ErrEmptyDSN = errors.New("postgres: DSN must not be empty")

// Open 使用 pgx stdlib 驱动返回 *sql.DB。
// dsn 格式："postgres://user:pass@host:5432/dbname?sslmode=disable"
// 连接参数：MaxOpenConns=25, MaxIdleConns=5, ConnMaxLifetime=5min。
// 调用 db.Ping() 验证连通性，失败则关闭并返回错误。
func Open(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, ErrEmptyDSN
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: open: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return db, nil
}
