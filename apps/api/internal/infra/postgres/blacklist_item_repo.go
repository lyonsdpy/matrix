package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"matrix/api/domain/violation"
)

var _ violation.SoftwareBlacklistRepository = (*blacklistItemRepo)(nil)

type blacklistItemRepo struct {
	db *sqlx.DB
}

// NewBlacklistItemRepository 创建基于 PostgreSQL 的软件黑名单规则持久化实现。
func NewBlacklistItemRepository(db *sqlx.DB) violation.SoftwareBlacklistRepository {
	return &blacklistItemRepo{db: db}
}

type softwareBlacklistRow struct {
	ID   string `db:"id"`
	Name string `db:"name"`
}

// FindAll 获取所有黑名单规则，按创建时间升序。
func (r *blacklistItemRepo) FindAll(ctx context.Context) ([]violation.BlacklistItem, error) {
	var rows []softwareBlacklistRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT id::text, name FROM software_blacklist ORDER BY created_at ASC`); err != nil {
		return nil, fmt.Errorf("blacklist item repo: find all: %w", err)
	}
	items := make([]violation.BlacklistItem, 0, len(rows))
	for _, row := range rows {
		id, err := uuid.Parse(row.ID)
		if err != nil {
			return nil, fmt.Errorf("blacklist item repo: parse id: %w", err)
		}
		items = append(items, violation.BlacklistItem{ID: id, Name: row.Name})
	}
	return items, nil
}

// Save 保存黑名单规则，重复名称忽略（ON CONFLICT DO NOTHING）。
func (r *blacklistItemRepo) Save(ctx context.Context, item *violation.BlacklistItem) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO software_blacklist (id, name) VALUES ($1, $2) ON CONFLICT (name) DO NOTHING`,
		item.ID.String(), item.Name)
	if err != nil {
		return fmt.Errorf("blacklist item repo: save: %w", err)
	}
	return nil
}

// Delete 按 ID 删除黑名单规则。
func (r *blacklistItemRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM software_blacklist WHERE id = $1`, id.String())
	if err != nil {
		return fmt.Errorf("blacklist item repo: delete: %w", err)
	}
	return nil
}
