package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"matrix/api/domain/identity"
)

var _ identity.UserBindItemRepository = (*userBindItemRepo)(nil)

type userBindItemRepo struct {
	db *sqlx.DB
}

// NewUserBindItemRepository 创建基于 PostgreSQL 的用户绑定条目持久化实现。
func NewUserBindItemRepository(db *sqlx.DB) identity.UserBindItemRepository {
	return &userBindItemRepo{db: db}
}

type userBindItemRow struct {
	ID        string    `db:"id"`
	ACGDevice string    `db:"acg_device"`
	UserPath  string    `db:"user_path"`
	BindType  string    `db:"bind_type"`
	Address   string    `db:"address"`
	IsExclude bool      `db:"is_exclude"`
	UpdatedAt time.Time `db:"updated_at"`
}

const userBindItemSelectCols = `id::text, acg_device, user_path, bind_type, address, is_exclude, updated_at`

const insertUserBindItemSQL = `
INSERT INTO user_bind_items (id, acg_device, user_path, bind_type, address, is_exclude, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, now())`

// ReplaceByDevice 原子替换指定 ACG 设备的用户绑定条目（全量覆盖）。
func (r *userBindItemRepo) ReplaceByDevice(ctx context.Context, acgDevice string, items []identity.UserBindItem) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("user bind item repo: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err = tx.ExecContext(ctx,
		`DELETE FROM user_bind_items WHERE acg_device = $1`, acgDevice); err != nil {
		return fmt.Errorf("user bind item repo: delete: %w", err)
	}

	for i := range items {
		item := &items[i]
		if _, err = tx.ExecContext(ctx, insertUserBindItemSQL,
			item.ID.String(), acgDevice, item.UserPath, item.BindType, item.Address, item.IsExclude); err != nil {
			return fmt.Errorf("user bind item repo: insert item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("user bind item repo: commit: %w", err)
	}
	return nil
}

// FindByUserPath 按用户路径查询绑定条目。
func (r *userBindItemRepo) FindByUserPath(ctx context.Context, userPath string) ([]identity.UserBindItem, error) {
	var rows []userBindItemRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+userBindItemSelectCols+` FROM user_bind_items WHERE user_path = $1`,
		userPath); err != nil {
		return nil, fmt.Errorf("user bind item repo: find by user path: %w", err)
	}
	return scanUserBindItemRows(rows)
}

// FindAll 获取所有用户绑定条目。
func (r *userBindItemRepo) FindAll(ctx context.Context) ([]identity.UserBindItem, error) {
	var rows []userBindItemRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+userBindItemSelectCols+` FROM user_bind_items`); err != nil {
		return nil, fmt.Errorf("user bind item repo: find all: %w", err)
	}
	return scanUserBindItemRows(rows)
}

func scanUserBindItemRows(rows []userBindItemRow) ([]identity.UserBindItem, error) {
	items := make([]identity.UserBindItem, 0, len(rows))
	for _, row := range rows {
		id, err := uuid.Parse(row.ID)
		if err != nil {
			return nil, fmt.Errorf("user bind item repo: parse id: %w", err)
		}
		items = append(items, identity.UserBindItem{
			ID:        id,
			ACGDevice: row.ACGDevice,
			UserPath:  row.UserPath,
			BindType:  row.BindType,
			Address:   row.Address,
			IsExclude: row.IsExclude,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return items, nil
}
