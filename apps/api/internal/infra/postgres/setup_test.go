package postgres

import (
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"matrix/api/internal/testutil"
)

// testDSN 返回测试用 PostgreSQL DSN，配置不存在时 skip。
func testDSN(t *testing.T) string {
	t.Helper()
	cfg, err := testutil.LoadConfig()
	if err != nil || cfg == nil || cfg.Postgres.IsEmpty() {
		t.Skip("no postgres config in testdata/config.toml, skipping")
	}
	return cfg.Postgres.DSN()
}

// setupTestDB 建立测试数据库连接，执行 up 迁移，测试结束后 truncate 清理。
func setupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := testDSN(t)

	if err := RunMigrations(dsn, migrationsPath); err != nil {
		t.Fatalf("setup: run migrations: %v", err)
	}

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("setup: open db: %v", err)
	}
	db := sqlx.NewDb(sqlDB, "pgx")

	// 测试开始前先清理，防止上次测试运行中断留下的脏数据影响结果。
	truncateAll(t, db)

	t.Cleanup(func() {
		truncateAll(t, db)
		_ = db.Close()
	})

	return db
}

// truncateAll 清空所有业务表并重置自增序列，用于测试前后的数据隔离。
// 逐表执行以容忍部分表不存在（迁移版本不同导致），确保已有表被正确清理。
func truncateAll(t *testing.T, db *sqlx.DB) {
	t.Helper()
	tables := []string{
		"bot_interactions",
		"blacklist_hits",
		"blacklist_items",
		"acg_whitelist",
		"user_bind_items",
		"observed_mac_bindings",
		"violation_records",
		"online_sessions",
		"employee_devices",
		"employees",
		"departments",
	}
	for _, tbl := range tables {
		if _, err := db.Exec(`TRUNCATE ` + tbl + ` RESTART IDENTITY CASCADE`); err != nil { //nolint:gosec // table names are hard-coded constants
			t.Logf("truncateAll: %s: %v", tbl, err)
		}
	}
}
