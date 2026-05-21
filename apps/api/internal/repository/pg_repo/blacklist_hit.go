package pg_repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// BlacklistHit 黑名单命中记录，存储在 PostgreSQL。
type BlacklistHit struct {
	ID              uuid.UUID
	EmployeeID      string
	EmployeeName    string
	DeviceID        string
	DeviceName      string
	SoftwareName    string
	SoftwareVersion string
	DetectedAt      time.Time
}

type BlacklistHitRepository interface {
	SaveBatch(ctx context.Context, hits []BlacklistHit) error
	FindByEmployeeID(ctx context.Context, employeeID string) ([]BlacklistHit, error)
	FindAll(ctx context.Context) ([]BlacklistHit, error)
}

var _ BlacklistHitRepository = (*blacklistHitRepo)(nil)

type blacklistHitRepo struct {
	db *sqlx.DB
}

func NewBlacklistHitRepository(db *sqlx.DB) BlacklistHitRepository {
	return &blacklistHitRepo{db: db}
}

type blacklistHitRow struct {
	ID              string    `db:"id"`
	EmployeeID      string    `db:"employee_id"`
	EmployeeName    string    `db:"employee_name"`
	DeviceID        string    `db:"device_id"`
	DeviceName      string    `db:"device_name"`
	SoftwareName    string    `db:"software_name"`
	SoftwareVersion string    `db:"software_version"`
	DetectedAt      time.Time `db:"detected_at"`
}

const blacklistHitSelectCols = `id::text, employee_id, employee_name, device_id, device_name, software_name, software_version, detected_at`

const saveBlacklistHitSQL = `
INSERT INTO blacklist_hits (id, employee_id, employee_name, device_id, device_name, software_name, software_version, detected_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO NOTHING`

func (r *blacklistHitRepo) SaveBatch(ctx context.Context, hits []BlacklistHit) error {
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

func (r *blacklistHitRepo) FindByEmployeeID(ctx context.Context, employeeID string) ([]BlacklistHit, error) {
	var rows []blacklistHitRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+blacklistHitSelectCols+` FROM blacklist_hits WHERE employee_id = $1 ORDER BY detected_at DESC`,
		employeeID); err != nil {
		return nil, fmt.Errorf("blacklist hit repo: find by employee id: %w", err)
	}
	return scanBlacklistHitRows(rows)
}

func (r *blacklistHitRepo) FindAll(ctx context.Context) ([]BlacklistHit, error) {
	var rows []blacklistHitRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+blacklistHitSelectCols+` FROM blacklist_hits ORDER BY detected_at DESC`); err != nil {
		return nil, fmt.Errorf("blacklist hit repo: find all: %w", err)
	}
	return scanBlacklistHitRows(rows)
}

func scanBlacklistHitRows(rows []blacklistHitRow) ([]BlacklistHit, error) {
	hits := make([]BlacklistHit, 0, len(rows))
	for _, row := range rows {
		id, err := uuid.Parse(row.ID)
		if err != nil {
			return nil, fmt.Errorf("blacklist hit repo: parse id: %w", err)
		}
		hits = append(hits, BlacklistHit{
			ID:              id,
			EmployeeID:      row.EmployeeID,
			EmployeeName:    row.EmployeeName,
			DeviceID:        row.DeviceID,
			DeviceName:      row.DeviceName,
			SoftwareName:    row.SoftwareName,
			SoftwareVersion: row.SoftwareVersion,
			DetectedAt:      row.DetectedAt,
		})
	}
	return hits, nil
}
