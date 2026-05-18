package postgres

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// RunMigrationsEmbedded 使用嵌入的迁移文件执行 schema 迁移。
// 调用方无需关心文件路径，二进制在任意工作目录均可正常运行。
func RunMigrationsEmbedded(dsn string) error {
	d, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("migrate: iofs: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", d, toPgx5DSN(dsn))
	if err != nil {
		return fmt.Errorf("migrate: new: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: up: %w", err)
	}
	return nil
}
