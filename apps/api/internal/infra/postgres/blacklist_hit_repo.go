package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"matrix/api/domain/violation"
)

var _ violation.BlacklistHitRepository = (*blacklistHitRepo)(nil)

type blacklistHitRepo struct {
	db *sqlx.DB
}

// NewBlacklistHitRepository 创建基于 PostgreSQL 的黑名单命中记录持久化实现。
func NewBlacklistHitRepository(db *sqlx.DB) violation.BlacklistHitRepository {
	return &blacklistHitRepo{db: db}
}

type blacklistHitRow struct {
	ID              string `db:"id"`
	EmployeeID      string `db:"employee_id"`
	EmployeeName    string `db:"employee_name"`
	DeviceID        string `db:"device_id"`
	DeviceName      string `db:"device_name"`
	SoftwareName    string `db:"software_name"`
	SoftwareVersion string `db:"software_version"`
	DetectedAt      string `db:"detected_at"`
}

const blacklistHitSelectCols = `id::text, employee_id, employee_name, device_id, device_name, software_name, software_version, detected_at::text`

const saveBlacklistHitSQL = `
INSERT INTO blacklist_hits (id, employee_id, employee_name, device_id, device_name, software_name, software_version, detected_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO NOTHING`

// SaveBatch 批量保存命中记录，使用事务。
func (r *blacklistHitRepo) SaveBatch(ctx context.Context, hits []violation.BlacklistHit) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("blacklist hit repo: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	for i := range hits {
		h := &hits[i]
		if _, err = tx.ExecContext(ctx, saveBlacklistHitSQL,
			h.ID.String(), h.EmployeeID, h.EmployeeName,
			h.DeviceID, h.DeviceName, h.SoftwareName, h.SoftwareVersion, h.DetectedAt); err != nil {
			return fmt.Errorf("blacklist hit repo: save batch item: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("blacklist hit repo: commit: %w", err)
	}
	return nil
}

// FindByEmployeeID 按员工 ID 查询命中记录，按检测时间降序。
func (r *blacklistHitRepo) FindByEmployeeID(ctx context.Context, employeeID string) ([]violation.BlacklistHit, error) {
	var rows []blacklistHitRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+blacklistHitSelectCols+` FROM blacklist_hits WHERE employee_id = $1 ORDER BY detected_at DESC`,
		employeeID); err != nil {
		return nil, fmt.Errorf("blacklist hit repo: find by employee id: %w", err)
	}
	return scanBlacklistHitRows(rows)
}

// FindAll 获取所有命中记录，按检测时间降序。
func (r *blacklistHitRepo) FindAll(ctx context.Context) ([]violation.BlacklistHit, error) {
	var rows []blacklistHitRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+blacklistHitSelectCols+` FROM blacklist_hits ORDER BY detected_at DESC`); err != nil {
		return nil, fmt.Errorf("blacklist hit repo: find all: %w", err)
	}
	return scanBlacklistHitRows(rows)
}

func scanBlacklistHitRows(rows []blacklistHitRow) ([]violation.BlacklistHit, error) {
	hits := make([]violation.BlacklistHit, 0, len(rows))
	for _, row := range rows {
		id, err := uuid.Parse(row.ID)
		if err != nil {
			return nil, fmt.Errorf("blacklist hit repo: parse id: %w", err)
		}
		hits = append(hits, violation.BlacklistHit{
			ID:              id,
			EmployeeID:      row.EmployeeID,
			EmployeeName:    row.EmployeeName,
			DeviceID:        row.DeviceID,
			DeviceName:      row.DeviceName,
			SoftwareName:    row.SoftwareName,
			SoftwareVersion: row.SoftwareVersion,
		})
	}
	return hits, nil
}
