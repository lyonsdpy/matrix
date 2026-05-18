package postgres

import (
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"matrix/api/internal/testutil"
)

const migrationsPath = "file://migrations"

func TestRunMigrations_UpDown(t *testing.T) {
	cfg, err := testutil.LoadConfig()
	if err != nil || cfg == nil || cfg.Postgres.IsEmpty() {
		t.Skip("no postgres config in testdata/config.toml, skipping")
	}
	dsn := cfg.Postgres.DSN()

	// 先确保是干净状态（忽略错误，表可能不存在）
	_ = runMigrationsDown(dsn, migrationsPath)

	t.Run("up creates tables", func(t *testing.T) {
		if err := RunMigrations(dsn, migrationsPath); err != nil {
			t.Fatalf("migrate up: %v", err)
		}

		db, err := sql.Open("pgx", dsn)
		if err != nil {
			t.Fatalf("open db: %v", err)
		}
		defer func() { _ = db.Close() }()

		for _, table := range []string{
			"departments", "employees", "employee_devices",
			"online_sessions", "violation_records",
		} {
			var exists bool
			err := db.QueryRow(
				`SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)`,
				table,
			).Scan(&exists)
			if err != nil {
				t.Errorf("check table %s: %v", table, err)
				continue
			}
			if !exists {
				t.Errorf("table %s not found after migrate up", table)
			}
		}
	})

	t.Run("up is idempotent", func(t *testing.T) {
		// 再次运行 up 应返回 nil（ErrNoChange 被忽略）
		if err := RunMigrations(dsn, migrationsPath); err != nil {
			t.Fatalf("second migrate up: %v", err)
		}
	})

	t.Run("down drops tables", func(t *testing.T) {
		if err := runMigrationsDown(dsn, migrationsPath); err != nil {
			t.Fatalf("migrate down: %v", err)
		}

		db, err := sql.Open("pgx", dsn)
		if err != nil {
			t.Fatalf("open db: %v", err)
		}
		defer func() { _ = db.Close() }()

		for _, table := range []string{
			"departments", "employees", "employee_devices",
			"online_sessions", "violation_records",
		} {
			var exists bool
			err := db.QueryRow(
				`SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)`,
				table,
			).Scan(&exists)
			if err != nil {
				t.Errorf("check table %s: %v", table, err)
				continue
			}
			if exists {
				t.Errorf("table %s still exists after migrate down", table)
			}
		}
	})
}
