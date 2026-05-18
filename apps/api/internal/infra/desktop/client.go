// Package desktop 提供亚信桌面管理（AIS Desktop）数据库客户端和 DeviceFetcher 实现。
// 使用 sqlx + go-sql-driver/mysql 连接桌面管理 MySQL 只读数据库。
package desktop

import (
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	// 注册 MySQL 驱动（副作用导入）。
	_ "github.com/go-sql-driver/mysql"
)

// ErrMissingDSN 表示 DSN 为空，无法建立数据库连接。
var ErrMissingDSN = errors.New("desktop: DSN is required")

// NewDB 创建并验证与桌面管理 MySQL 数据库的只读连接。
// dsn 格式示例："user:pass@tcp(host:port)/ASM?charset=utf8mb4&parseTime=true"
// 连接池参数：MaxOpenConns=5、MaxIdleConns=2、ConnMaxLifetime=30min。
func NewDB(dsn string) (*sqlx.DB, error) {
	if dsn == "" {
		return nil, ErrMissingDSN
	}

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}
