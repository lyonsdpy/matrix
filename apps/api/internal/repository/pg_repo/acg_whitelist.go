package pg_repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jmoiron/sqlx"

	"matrix/api/internal/infra/aisacg"
)

var _ aisacg.WhitelistRepository = (*acgWhitelistRepo)(nil)

type acgWhitelistRepo struct {
	db *sqlx.DB
}

// NewACGWhitelistRepository 创建基于 PostgreSQL 的 ACG 全局白名单持久化实现。
func NewACGWhitelistRepository(db *sqlx.DB) aisacg.WhitelistRepository {
	return &acgWhitelistRepo{db: db}
}

type acgWhitelistRow struct {
	Enable bool   `db:"enable"`
	Name   string `db:"name"`
	Desc   string `db:"desc"`
	Addrs  string `db:"addrs"`
}

// ReplaceByDevice 原子替换指定 ACG 设备的白名单条目（全量覆盖）。
func (r *acgWhitelistRepo) ReplaceByDevice(ctx context.Context, acgDevice string, entries []aisacg.WhitelistEntry) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("acg whitelist repo: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err = tx.ExecContext(ctx,
		`DELETE FROM acg_whitelist_entries WHERE acg_device = $1`, acgDevice); err != nil {
		return fmt.Errorf("acg whitelist repo: delete: %w", err)
	}

	for i := range entries {
		e := &entries[i]
		addrs := pgtype.FlatArray[string](e.Addrs)
		if _, err = tx.ExecContext(ctx,
			`INSERT INTO acg_whitelist_entries (id, acg_device, enable, name, "desc", addrs, updated_at)
			 VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, now())`,
			acgDevice, e.Enable, e.Name, e.Desc, addrs); err != nil {
			return fmt.Errorf("acg whitelist repo: insert entry: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("acg whitelist repo: commit: %w", err)
	}
	return nil
}

// FindAll 获取所有 ACG 设备的白名单条目。
func (r *acgWhitelistRepo) FindAll(ctx context.Context) ([]aisacg.WhitelistEntry, error) {
	var rows []acgWhitelistRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT enable, name, "desc", array_to_json(addrs)::text as addrs FROM acg_whitelist_entries`); err != nil {
		return nil, fmt.Errorf("acg whitelist repo: find all: %w", err)
	}
	entries := make([]aisacg.WhitelistEntry, 0, len(rows))
	for _, row := range rows {
		var addrs []string
		if row.Addrs != "" && row.Addrs != "null" {
			if err := json.Unmarshal([]byte(row.Addrs), &addrs); err != nil {
				return nil, fmt.Errorf("acg whitelist repo: parse addrs: %w", err)
			}
		}
		entries = append(entries, aisacg.WhitelistEntry{
			Enable: row.Enable,
			Name:   row.Name,
			Desc:   row.Desc,
			Addrs:  addrs,
		})
	}
	return entries, nil
}
