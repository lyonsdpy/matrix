package pg_repo

import (
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"matrix/api/internal/infra/postgres"
	"matrix/api/internal/testutil"
)

// testDSN 返回测试用 PostgreSQL DSN，未配置时 skip。
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

	// 使用嵌入的迁移文件，无需依赖文件路径
	if err := postgres.RunMigrationsEmbedded(dsn); err != nil {
		t.Fatalf("setup: run migrations: %v", err)
	}

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("setup: open db: %v", err)
	}
	db := sqlx.NewDb(sqlDB, "pgx")

	truncateAll(t, db)
	t.Cleanup(func() {
		truncateAll(t, db)
		_ = db.Close()
	})

	return db
}

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
		if _, err := db.Exec(`TRUNCATE ` + tbl + ` RESTART IDENTITY CASCADE`); err != nil { //nolint:gosec
			t.Logf("truncateAll: %s: %v", tbl, err)
		}
	}
}
