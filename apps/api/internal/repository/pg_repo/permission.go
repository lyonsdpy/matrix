package pg_repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Permission 权限码记录（与 pkg/perm.Permission 同义，但带 DB 视角字段）
type Permission struct {
	Code       string
	Name       string
	Module     string
	Kind       string // "page" | "action"
	ParentCode string // 空字符串表示顶级
	Sort       int
}

type PermissionRepository interface {
	// ListAll 列出所有权限码，按 sort 升序；UI 树形展示用
	ListAll(ctx context.Context) ([]Permission, error)
}

type permissionRepo struct {
	db *sqlx.DB
}

func NewPermissionRepository(db *sqlx.DB) PermissionRepository {
	return &permissionRepo{db: db}
}

type permissionRow struct {
	Code       string         `db:"code"`
	Name       string         `db:"name"`
	Module     string         `db:"module"`
	Kind       string         `db:"kind"`
	ParentCode sql.NullString `db:"parent_code"`
	Sort       int            `db:"sort"`
}

func (r *permissionRepo) ListAll(ctx context.Context) ([]Permission, error) {
	var rows []permissionRow
	err := r.db.SelectContext(ctx, &rows,
		`SELECT code, name, module, kind, parent_code, sort FROM permissions ORDER BY sort, code`,
	)
	if err != nil {
		return nil, fmt.Errorf("permission repo: list all: %w", err)
	}
	out := make([]Permission, len(rows))
	for i, row := range rows {
		out[i] = Permission{
			Code:       row.Code,
			Name:       row.Name,
			Module:     row.Module,
			Kind:       row.Kind,
			ParentCode: nsToString(row.ParentCode),
			Sort:       row.Sort,
		}
	}
	return out, nil
}

// nsToString 将 sql.NullString 转为 string，NULL 视为空字符串
// （与 department.go 中 nullStr 方向相反：那个把 string→NULL）
func nsToString(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}
