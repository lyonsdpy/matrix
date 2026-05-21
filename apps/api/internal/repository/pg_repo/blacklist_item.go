package pg_repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// BlacklistItem 软件黑名单规则，存储在 PostgreSQL。
type BlacklistItem struct {
	ID   uuid.UUID
	Name string
}

type SoftwareBlacklistRepository interface {
	FindAll(ctx context.Context) ([]BlacklistItem, error)
	Save(ctx context.Context, item *BlacklistItem) error
	Delete(ctx context.Context, id uuid.UUID) error
}

var _ SoftwareBlacklistRepository = (*blacklistItemRepo)(nil)

type blacklistItemRepo struct {
	db *sqlx.DB
}

func NewBlacklistItemRepository(db *sqlx.DB) SoftwareBlacklistRepository {
	return &blacklistItemRepo{db: db}
}

type softwareBlacklistRow struct {
	ID   string `db:"id"`
	Name string `db:"name"`
}

func (r *blacklistItemRepo) FindAll(ctx context.Context) ([]BlacklistItem, error) {
	var rows []softwareBlacklistRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT id::text, name FROM software_blacklist ORDER BY created_at ASC`); err != nil {
		return nil, fmt.Errorf("blacklist item repo: find all: %w", err)
	}
	items := make([]BlacklistItem, 0, len(rows))
	for _, row := range rows {
		id, err := uuid.Parse(row.ID)
		if err != nil {
			return nil, fmt.Errorf("blacklist item repo: parse id: %w", err)
		}
		items = append(items, BlacklistItem{ID: id, Name: row.Name})
	}
	return items, nil
}

func (r *blacklistItemRepo) Save(ctx context.Context, item *BlacklistItem) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO software_blacklist (id, name) VALUES ($1, $2) ON CONFLICT (name) DO NOTHING`,
		item.ID.String(), item.Name)
	if err != nil {
		return fmt.Errorf("blacklist item repo: save: %w", err)
	}
	return nil
}

func (r *blacklistItemRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM software_blacklist WHERE id = $1`, id.String())
	if err != nil {
		return fmt.Errorf("blacklist item repo: delete: %w", err)
	}
	return nil
}
