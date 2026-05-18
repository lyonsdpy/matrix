package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"matrix/api/domain/violation"
)

type violationRepo struct {
	db *sqlx.DB
}

// NewViolationRepository 创建基于 PostgreSQL 的违规记录持久化实现。
func NewViolationRepository(db *sqlx.DB) violation.ViolationRepository {
	return &violationRepo{db: db}
}

type violationRow struct {
	ID           string         `db:"id"`
	EmployeeID   sql.NullString `db:"employee_id"`
	EmployeeName string         `db:"employee_name"`
	Office       string         `db:"office"`
	IP           string         `db:"ip"`
	MAC          string         `db:"mac"`
	Status       string         `db:"status"`
	DetectedAt   sql.NullTime   `db:"detected_at"`
}

const violationSelectCols = `id, employee_id, employee_name, office, ip, mac, status, detected_at`

const saveViolationSQL = `
INSERT INTO violation_records (id, employee_id, employee_name, office, ip, mac, status, detected_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE SET
    employee_id   = EXCLUDED.employee_id,
    employee_name = EXCLUDED.employee_name,
    office        = EXCLUDED.office,
    ip            = EXCLUDED.ip,
    mac           = EXCLUDED.mac,
    status        = EXCLUDED.status,
    detected_at   = EXCLUDED.detected_at`

func (r *violationRepo) Save(ctx context.Context, rec *violation.ViolationRecord) error {
	_, err := r.db.ExecContext(ctx, saveViolationSQL,
		rec.ID, nullStr(rec.EmployeeID), rec.EmployeeName,
		rec.Office, rec.IP, rec.MAC, string(rec.Status), rec.DetectedAt)
	if err != nil {
		return fmt.Errorf("violation repo: save: %w", err)
	}
	return nil
}

func (r *violationRepo) SaveBatch(ctx context.Context, records []violation.ViolationRecord) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("violation repo: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	for i := range records {
		rec := &records[i]
		if _, err = tx.ExecContext(ctx, saveViolationSQL,
			rec.ID, nullStr(rec.EmployeeID), rec.EmployeeName,
			rec.Office, rec.IP, rec.MAC, string(rec.Status), rec.DetectedAt); err != nil {
			return fmt.Errorf("violation repo: save batch item: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("violation repo: commit: %w", err)
	}
	return nil
}

func (r *violationRepo) FindByOffice(ctx context.Context, office string) ([]violation.ViolationRecord, error) {
	var rows []violationRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+violationSelectCols+` FROM violation_records WHERE office = $1 ORDER BY detected_at DESC`,
		office); err != nil {
		return nil, fmt.Errorf("violation repo: find by office: %w", err)
	}
	return scanViolationRows(rows), nil
}

func (r *violationRepo) FindAll(ctx context.Context) ([]violation.ViolationRecord, error) {
	var rows []violationRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+violationSelectCols+` FROM violation_records ORDER BY detected_at DESC`); err != nil {
		return nil, fmt.Errorf("violation repo: find all: %w", err)
	}
	return scanViolationRows(rows), nil
}

func scanViolationRows(rows []violationRow) []violation.ViolationRecord {
	records := make([]violation.ViolationRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, violation.ViolationRecord{
			ID:           row.ID,
			EmployeeID:   row.EmployeeID.String,
			EmployeeName: row.EmployeeName,
			Office:       row.Office,
			IP:           row.IP,
			MAC:          row.MAC,
			Status:       violation.ComplianceStatus(row.Status),
			DetectedAt:   row.DetectedAt.Time,
		})
	}
	return records
}
