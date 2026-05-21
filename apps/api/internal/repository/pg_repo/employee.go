package pg_repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Employee 飞书通讯录同步过来的员工数据，存储在 PostgreSQL。
// 不是图中的 domain 概念，是外部系统的镜像副本。
type Employee struct {
	ID            string
	ExternalID    string // 飞书 open_id
	Name          string
	Email         string
	Mobile        string
	Status        int
	DepartmentIDs []string
	UpdatedAt     time.Time
}

type EmployeeRepository interface {
	Save(ctx context.Context, emp *Employee) error
	SaveBatch(ctx context.Context, emps []Employee) error
	FindByID(ctx context.Context, id string) (*Employee, error)
	FindByExternalID(ctx context.Context, externalID string) (*Employee, error)
	FindAll(ctx context.Context) ([]Employee, error)
	DeleteByExternalID(ctx context.Context, externalID string) error
}

type employeeRepo struct {
	db *sqlx.DB
}

func NewEmployeeRepository(db *sqlx.DB) EmployeeRepository {
	return &employeeRepo{db: db}
}

type employeeRow struct {
	ID            string         `db:"id"`
	ExternalID    string         `db:"external_id"`
	Name          string         `db:"name"`
	Email         sql.NullString `db:"email"`
	Mobile        sql.NullString `db:"mobile"`
	Status        int            `db:"status"`
	DepartmentIDs string         `db:"department_ids"` // JSONB → JSON string
	UpdatedAt     time.Time      `db:"updated_at"`
}

const employeeSelectCols = `id, external_id, name, email, mobile, status, department_ids, updated_at`

const saveEmployeeSQL = `
INSERT INTO employees (id, external_id, name, email, mobile, status, department_ids, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, NOW())
ON CONFLICT (id) DO UPDATE SET
    external_id    = EXCLUDED.external_id,
    name           = EXCLUDED.name,
    email          = EXCLUDED.email,
    mobile         = EXCLUDED.mobile,
    status         = EXCLUDED.status,
    department_ids = EXCLUDED.department_ids,
    updated_at     = NOW()`

func (r *employeeRepo) Save(ctx context.Context, emp *Employee) error {
	deptJSON, err := json.Marshal(emp.DepartmentIDs)
	if err != nil {
		return fmt.Errorf("employee repo: marshal department_ids: %w", err)
	}
	_, err = r.db.ExecContext(ctx, saveEmployeeSQL,
		emp.ID, emp.ExternalID, emp.Name, emp.Email, emp.Mobile, emp.Status, string(deptJSON))
	if err != nil {
		return fmt.Errorf("employee repo: save: %w", err)
	}
	return nil
}

func (r *employeeRepo) FindByExternalID(ctx context.Context, externalID string) (*Employee, error) {
	var row employeeRow
	err := r.db.GetContext(ctx, &row,
		`SELECT `+employeeSelectCols+` FROM employees WHERE external_id = $1`, externalID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("employee repo: find by external id: %w", err)
	}
	return toEmployee(row)
}

func (r *employeeRepo) FindByID(ctx context.Context, id string) (*Employee, error) {
	var row employeeRow
	err := r.db.GetContext(ctx, &row,
		`SELECT `+employeeSelectCols+` FROM employees WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("employee repo: find by id: %w", err)
	}
	return toEmployee(row)
}

func (r *employeeRepo) FindAll(ctx context.Context) ([]Employee, error) {
	var rows []employeeRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+employeeSelectCols+` FROM employees ORDER BY name`); err != nil {
		return nil, fmt.Errorf("employee repo: find all: %w", err)
	}
	emps := make([]Employee, 0, len(rows))
	for _, row := range rows {
		emp, err := toEmployee(row)
		if err != nil {
			return nil, err
		}
		emps = append(emps, *emp)
	}
	return emps, nil
}

func (r *employeeRepo) SaveBatch(ctx context.Context, emps []Employee) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("employee repo: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	for i := range emps {
		deptJSON, err := json.Marshal(emps[i].DepartmentIDs)
		if err != nil {
			return fmt.Errorf("employee repo: marshal department_ids: %w", err)
		}
		if _, err = tx.ExecContext(ctx, saveEmployeeSQL,
			emps[i].ID, emps[i].ExternalID, emps[i].Name,
			emps[i].Email, emps[i].Mobile, emps[i].Status, string(deptJSON)); err != nil {
			return fmt.Errorf("employee repo: save batch item: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("employee repo: commit: %w", err)
	}
	return nil
}

func (r *employeeRepo) DeleteByExternalID(ctx context.Context, externalID string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM employees WHERE external_id = $1`, externalID); err != nil {
		return fmt.Errorf("employee repo: delete: %w", err)
	}
	return nil
}

func toEmployee(row employeeRow) (*Employee, error) {
	var deptIDs []string
	if err := json.Unmarshal([]byte(row.DepartmentIDs), &deptIDs); err != nil {
		return nil, fmt.Errorf("employee repo: unmarshal department_ids: %w", err)
	}
	if deptIDs == nil {
		deptIDs = []string{}
	}
	return &Employee{
		ID:            row.ID,
		ExternalID:    row.ExternalID,
		Name:          row.Name,
		Email:         row.Email.String,
		Mobile:        row.Mobile.String,
		Status:        row.Status,
		DepartmentIDs: deptIDs,
		UpdatedAt:     row.UpdatedAt,
	}, nil
}
