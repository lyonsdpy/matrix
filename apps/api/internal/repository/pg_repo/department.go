package pg_repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Department 飞书组织架构同步过来的部门数据，存储在 PostgreSQL。
type Department struct {
	ID          string
	ExternalID  string // 飞书 department_id
	Name        string
	ParentID    string
	LeaderID    string
	MemberCount int
	UpdatedAt   time.Time
}

type DepartmentRepository interface {
	Save(ctx context.Context, dept *Department) error
	SaveBatch(ctx context.Context, depts []Department) error
	FindByExternalID(ctx context.Context, externalID string) (*Department, error)
	FindAll(ctx context.Context) ([]Department, error)
	DeleteByExternalID(ctx context.Context, externalID string) error
}

type departmentRepo struct {
	db *sqlx.DB
}

func NewDepartmentRepository(db *sqlx.DB) DepartmentRepository {
	return &departmentRepo{db: db}
}

type departmentRow struct {
	ID          string         `db:"id"`
	ExternalID  string         `db:"external_id"`
	Name        string         `db:"name"`
	ParentID    sql.NullString `db:"parent_id"`
	LeaderID    sql.NullString `db:"leader_id"`
	MemberCount int            `db:"member_count"`
	UpdatedAt   time.Time      `db:"updated_at"`
}

const deptSelectCols = `id, external_id, name, parent_id, leader_id, member_count, updated_at`

const saveDeptSQL = `
INSERT INTO departments (id, external_id, name, parent_id, leader_id, member_count, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (id) DO UPDATE SET
    external_id  = EXCLUDED.external_id,
    name         = EXCLUDED.name,
    parent_id    = EXCLUDED.parent_id,
    leader_id    = EXCLUDED.leader_id,
    member_count = EXCLUDED.member_count,
    updated_at   = NOW()`

func (r *departmentRepo) Save(ctx context.Context, dept *Department) error {
	_, err := r.db.ExecContext(ctx, saveDeptSQL,
		dept.ID, dept.ExternalID, dept.Name,
		nullStr(dept.ParentID), nullStr(dept.LeaderID), dept.MemberCount)
	if err != nil {
		return fmt.Errorf("department repo: save: %w", err)
	}
	return nil
}

func (r *departmentRepo) FindByExternalID(ctx context.Context, externalID string) (*Department, error) {
	var row departmentRow
	err := r.db.GetContext(ctx, &row,
		`SELECT `+deptSelectCols+` FROM departments WHERE external_id = $1`, externalID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("department repo: find by external id: %w", err)
	}
	return toDepartment(row), nil
}

func (r *departmentRepo) FindAll(ctx context.Context) ([]Department, error) {
	var rows []departmentRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+deptSelectCols+` FROM departments ORDER BY name`); err != nil {
		return nil, fmt.Errorf("department repo: find all: %w", err)
	}
	depts := make([]Department, 0, len(rows))
	for _, row := range rows {
		depts = append(depts, *toDepartment(row))
	}
	return depts, nil
}

func (r *departmentRepo) SaveBatch(ctx context.Context, depts []Department) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("department repo: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	for i := range depts {
		if _, err = tx.ExecContext(ctx, saveDeptSQL,
			depts[i].ID, depts[i].ExternalID, depts[i].Name,
			nullStr(depts[i].ParentID), nullStr(depts[i].LeaderID), depts[i].MemberCount); err != nil {
			return fmt.Errorf("department repo: save batch item: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("department repo: commit: %w", err)
	}
	return nil
}

func (r *departmentRepo) DeleteByExternalID(ctx context.Context, externalID string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM departments WHERE external_id = $1`, externalID); err != nil {
		return fmt.Errorf("department repo: delete: %w", err)
	}
	return nil
}

func toDepartment(row departmentRow) *Department {
	return &Department{
		ID:          row.ID,
		ExternalID:  row.ExternalID,
		Name:        row.Name,
		ParentID:    row.ParentID.String,
		LeaderID:    row.LeaderID.String,
		MemberCount: row.MemberCount,
		UpdatedAt:   row.UpdatedAt,
	}
}

// nullStr 将空字符串转为 SQL NULL，非空原样返回。
func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
