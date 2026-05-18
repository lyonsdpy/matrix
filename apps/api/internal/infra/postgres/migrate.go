package postgres

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // pgx5 驱动
	_ "github.com/golang-migrate/migrate/v4/source/file"     // file source
)

// RunMigrations 执行数据库 schema 迁移（up）。
// migrationsPath 为迁移文件目录，如 "file://migrations"。
// DSN 支持 postgres:// 或 postgresql:// 前缀，内部自动转换为 pgx5://。
func RunMigrations(dsn string, migrationsPath string) error {
	m, err := migrate.New(migrationsPath, toPgx5DSN(dsn))
	if err != nil {
		return fmt.Errorf("migrate: new: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: up: %w", err)
	}
	return nil
}

// runMigrationsDown 回滚所有迁移（仅供测试使用）。
func runMigrationsDown(dsn string, migrationsPath string) error {
	m, err := migrate.New(migrationsPath, toPgx5DSN(dsn))
	if err != nil {
		return fmt.Errorf("migrate: new: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: down: %w", err)
	}
	return nil
}

// toPgx5DSN 将 postgres:// 或 postgresql:// 前缀转换为 pgx5://。
func toPgx5DSN(dsn string) string {
	dsn = strings.Replace(dsn, "postgresql://", "pgx5://", 1)
	dsn = strings.Replace(dsn, "postgres://", "pgx5://", 1)
	return dsn
}
