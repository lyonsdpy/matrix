package perm

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// SyncCatalog 将 Codes 列表 UPSERT 同步至 permissions 表。
// 启动时调用一次，幂等可重跑：新增权限码会插入，已有的更新名称/排序，
// 数据库里多余的（被代码删除的）会被清理，保证库与代码一致。
func SyncCatalog(ctx context.Context, db *sqlx.DB) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("perm sync: begin tx: %w", err)
	}
	defer tx.Rollback()

	// 两遍 UPSERT：第一遍只写非自引用列，第二遍补 parent_code
	// 原因：parent_code 自引用，如果父子在同一批且父在后面，单遍会因 FK 失败
	for _, p := range Codes {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO permissions (code, name, module, kind, sort)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (code) DO UPDATE
			   SET name = EXCLUDED.name,
			       module = EXCLUDED.module,
			       kind = EXCLUDED.kind,
			       sort = EXCLUDED.sort`,
			p.Code, p.Name, p.Module, string(p.Kind), p.Sort,
		)
		if err != nil {
			return fmt.Errorf("perm sync: upsert %s: %w", p.Code, err)
		}
	}

	for _, p := range Codes {
		var parent sql.NullString
		if p.Parent != "" {
			parent = sql.NullString{String: p.Parent, Valid: true}
		}
		_, err := tx.ExecContext(ctx,
			`UPDATE permissions SET parent_code = $1 WHERE code = $2`,
			parent, p.Code,
		)
		if err != nil {
			return fmt.Errorf("perm sync: update parent %s: %w", p.Code, err)
		}
	}

	// 清理：代码里已删除但库里还在的权限码
	codes := make([]string, 0, len(Codes))
	for _, p := range Codes {
		codes = append(codes, p.Code)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM permissions WHERE code <> ALL($1)`,
		codes,
	); err != nil {
		return fmt.Errorf("perm sync: prune stale: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("perm sync: commit: %w", err)
	}
	return nil
}
