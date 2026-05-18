package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"matrix/api/domain/identity"
)

type sessionRepo struct {
	db *sqlx.DB
}

// NewOnlineSessionRepository 创建基于 PostgreSQL 的在线会话持久化实现。
func NewOnlineSessionRepository(db *sqlx.DB) identity.OnlineSessionRepository {
	return &sessionRepo{db: db}
}

// sessionRow 映射 online_sessions 表的非自增字段（排除 id 和 fetched_at）。
type sessionRow struct {
	EmployeeID            sql.NullString `db:"employee_id"`
	EmployeeName          string         `db:"employee_name"`
	DeviceID              sql.NullString `db:"device_id"`
	Office                string         `db:"office"`
	IP                    string         `db:"ip"`
	MAC                   string         `db:"mac"`
	LoginTime             string         `db:"login_time"`
	OnlineTime            string         `db:"online_time"`
	Platform              string         `db:"platform"`
	System                string         `db:"system"`
	Device                string         `db:"device"`
	ComplianceCheckResult string         `db:"compliance_check_result"`
}

const sessionSelectCols = `employee_id, employee_name, device_id, office, ip, mac, login_time, online_time, platform, system, device, compliance_check_result`

const insertSessionSQL = `
INSERT INTO online_sessions (employee_id, employee_name, device_id, office, ip, mac, login_time, online_time, platform, system, device, compliance_check_result)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

// ReplaceByOffice 事务内先删除指定 office 的所有会话，再批量插入新会话。
func (r *sessionRepo) ReplaceByOffice(ctx context.Context, office string, sessions []identity.OnlineSession) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("session repo: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err = tx.ExecContext(ctx, `DELETE FROM online_sessions WHERE office = $1`, office); err != nil {
		return fmt.Errorf("session repo: delete by office: %w", err)
	}

	for i := range sessions {
		s := &sessions[i]
		if _, err = tx.ExecContext(ctx, insertSessionSQL,
			nullStr(s.EmployeeID), s.EmployeeName, nullStr(s.DeviceID),
			s.Office, s.IP, s.MAC,
			s.LoginTime, s.OnlineTime, s.Platform,
			s.System, s.Device, s.ComplianceCheckResult,
		); err != nil {
			return fmt.Errorf("session repo: insert session: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("session repo: commit: %w", err)
	}
	return nil
}

func (r *sessionRepo) FindByOffice(ctx context.Context, office string) ([]identity.OnlineSession, error) {
	var rows []sessionRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+sessionSelectCols+` FROM online_sessions WHERE office = $1 ORDER BY employee_name`,
		office); err != nil {
		return nil, fmt.Errorf("session repo: find by office: %w", err)
	}
	return toSessions(rows), nil
}

func (r *sessionRepo) FindByEmployeeID(ctx context.Context, employeeID string) ([]identity.OnlineSession, error) {
	var rows []sessionRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+sessionSelectCols+` FROM online_sessions WHERE employee_id = $1 ORDER BY office`,
		employeeID); err != nil {
		return nil, fmt.Errorf("session repo: find by employee id: %w", err)
	}
	return toSessions(rows), nil
}

func (r *sessionRepo) FindAll(ctx context.Context) ([]identity.OnlineSession, error) {
	var rows []sessionRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+sessionSelectCols+` FROM online_sessions ORDER BY office, employee_name`); err != nil {
		return nil, fmt.Errorf("session repo: find all: %w", err)
	}
	return toSessions(rows), nil
}

func toSessions(rows []sessionRow) []identity.OnlineSession {
	sessions := make([]identity.OnlineSession, 0, len(rows))
	for _, row := range rows {
		sessions = append(sessions, identity.OnlineSession{
			EmployeeID:            row.EmployeeID.String,
			EmployeeName:          row.EmployeeName,
			DeviceID:              row.DeviceID.String,
			Office:                row.Office,
			IP:                    row.IP,
			MAC:                   row.MAC,
			LoginTime:             row.LoginTime,
			OnlineTime:            row.OnlineTime,
			Platform:              row.Platform,
			System:                row.System,
			Device:                row.Device,
			ComplianceCheckResult: row.ComplianceCheckResult,
		})
	}
	return sessions
}
