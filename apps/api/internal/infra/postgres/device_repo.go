package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"matrix/api/domain/identity"
)

type deviceRepo struct {
	db *sqlx.DB
}

// NewDeviceRepository 创建基于 PostgreSQL 的员工设备持久化实现。
func NewDeviceRepository(db *sqlx.DB) identity.DeviceRepository {
	return &deviceRepo{db: db}
}

type deviceRow struct {
	ID             string         `db:"id"`
	EmployeeID     sql.NullString `db:"employee_id"`
	ExternalID     string         `db:"external_id"`
	DeviceName     string         `db:"device_name"`
	Platform       string         `db:"platform"`
	SerialNumber   string         `db:"serial_number"`
	OSVersion      string         `db:"os_version"`
	Status         string         `db:"status"`
	TrustLevel     string         `db:"trust_level"`
	LastOnlineTime sql.NullInt64  `db:"last_online_time"`
	UpdatedAt      time.Time      `db:"updated_at"`
}

const deviceSelectCols = `id, employee_id, external_id, device_name, platform, serial_number, os_version, status, trust_level, last_online_time, updated_at`

const saveDeviceSQL = `
INSERT INTO employee_devices (id, employee_id, external_id, device_name, platform, serial_number, os_version, status, trust_level, last_online_time, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
ON CONFLICT (id) DO UPDATE SET
    employee_id      = EXCLUDED.employee_id,
    external_id      = EXCLUDED.external_id,
    device_name      = EXCLUDED.device_name,
    platform         = EXCLUDED.platform,
    serial_number    = EXCLUDED.serial_number,
    os_version       = EXCLUDED.os_version,
    status           = EXCLUDED.status,
    trust_level      = EXCLUDED.trust_level,
    last_online_time = EXCLUDED.last_online_time,
    updated_at       = NOW()`

func (r *deviceRepo) Save(ctx context.Context, dev *identity.EmployeeDevice) error {
	var lastOnline any
	if dev.LastOnlineTime != 0 {
		lastOnline = dev.LastOnlineTime
	}
	_, err := r.db.ExecContext(ctx, saveDeviceSQL,
		dev.ID, nullStr(dev.EmployeeID), dev.ExternalID,
		dev.DeviceName, dev.Platform, dev.SerialNumber,
		dev.OSVersion, dev.Status, dev.TrustLevel, lastOnline)
	if err != nil {
		return fmt.Errorf("device repo: save: %w", err)
	}
	return nil
}

func (r *deviceRepo) FindByExternalID(ctx context.Context, externalID string) (*identity.EmployeeDevice, error) {
	var row deviceRow
	err := r.db.GetContext(ctx, &row,
		`SELECT `+deviceSelectCols+` FROM employee_devices WHERE external_id = $1`, externalID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("device repo: find by external id: %w", err)
	}
	return toDevice(row), nil
}

func (r *deviceRepo) FindByEmployeeID(ctx context.Context, employeeID string) ([]identity.EmployeeDevice, error) {
	var rows []deviceRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+deviceSelectCols+` FROM employee_devices WHERE employee_id = $1`, employeeID); err != nil {
		return nil, fmt.Errorf("device repo: find by employee id: %w", err)
	}
	devs := make([]identity.EmployeeDevice, 0, len(rows))
	for _, row := range rows {
		devs = append(devs, *toDevice(row))
	}
	return devs, nil
}

func (r *deviceRepo) SaveBatch(ctx context.Context, devs []identity.EmployeeDevice) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("device repo: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	for i := range devs {
		var lastOnline any
		if devs[i].LastOnlineTime != 0 {
			lastOnline = devs[i].LastOnlineTime
		}
		if _, err = tx.ExecContext(ctx, saveDeviceSQL,
			devs[i].ID, nullStr(devs[i].EmployeeID), devs[i].ExternalID,
			devs[i].DeviceName, devs[i].Platform, devs[i].SerialNumber,
			devs[i].OSVersion, devs[i].Status, devs[i].TrustLevel, lastOnline); err != nil {
			return fmt.Errorf("device repo: save batch item: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("device repo: commit: %w", err)
	}
	return nil
}

func (r *deviceRepo) DeleteByExternalID(ctx context.Context, externalID string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM employee_devices WHERE external_id = $1`, externalID); err != nil {
		return fmt.Errorf("device repo: delete: %w", err)
	}
	return nil
}

func toDevice(row deviceRow) *identity.EmployeeDevice {
	return &identity.EmployeeDevice{
		ID:             row.ID,
		EmployeeID:     row.EmployeeID.String,
		ExternalID:     row.ExternalID,
		DeviceName:     row.DeviceName,
		Platform:       row.Platform,
		SerialNumber:   row.SerialNumber,
		OSVersion:      row.OSVersion,
		Status:         row.Status,
		TrustLevel:     row.TrustLevel,
		LastOnlineTime: row.LastOnlineTime.Int64,
		UpdatedAt:      row.UpdatedAt,
	}
}
