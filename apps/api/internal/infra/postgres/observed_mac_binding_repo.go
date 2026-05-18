package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"matrix/api/domain/identity"
)

var _ identity.ObservedMACBindingRepository = (*observedMACBindingRepo)(nil)

type observedMACBindingRepo struct {
	db *sqlx.DB
}

// NewObservedMACBindingRepository 创建基于 PostgreSQL 的实测 MAC 绑定记录持久化实现。
func NewObservedMACBindingRepository(db *sqlx.DB) identity.ObservedMACBindingRepository {
	return &observedMACBindingRepo{db: db}
}

type observedMACBindingRow struct {
	ID         string    `db:"id"`
	Office     string    `db:"office"`
	IP         string    `db:"ip"`
	MAC        string    `db:"mac"`
	LastSeenAt time.Time `db:"last_seen_at"`
}

const observedMACSelectCols = `id::text, office, ip, mac, last_seen_at`

// Upsert 插入或更新 MAC 绑定记录（按 office/ip/mac 唯一冲突时更新 last_seen_at）。
func (r *observedMACBindingRepo) Upsert(ctx context.Context, binding *identity.ObservedMACBinding) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO observed_mac_bindings (id, office, ip, mac, last_seen_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (office, ip, mac) DO UPDATE SET last_seen_at = now()`,
		binding.ID.String(), binding.Office, binding.IP, binding.MAC, binding.LastSeenAt)
	if err != nil {
		return fmt.Errorf("observed mac binding repo: upsert: %w", err)
	}
	return nil
}

// FindByIP 按 IP 地址查询 MAC 绑定记录。
func (r *observedMACBindingRepo) FindByIP(ctx context.Context, ip string) ([]identity.ObservedMACBinding, error) {
	var rows []observedMACBindingRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+observedMACSelectCols+` FROM observed_mac_bindings WHERE ip = $1 ORDER BY last_seen_at DESC`,
		ip); err != nil {
		return nil, fmt.Errorf("observed mac binding repo: find by ip: %w", err)
	}
	return scanObservedMACRows(rows)
}

// FindByMAC 按 MAC 地址查询 MAC 绑定记录。
func (r *observedMACBindingRepo) FindByMAC(ctx context.Context, mac string) ([]identity.ObservedMACBinding, error) {
	var rows []observedMACBindingRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+observedMACSelectCols+` FROM observed_mac_bindings WHERE mac = $1 ORDER BY last_seen_at DESC`,
		mac); err != nil {
		return nil, fmt.Errorf("observed mac binding repo: find by mac: %w", err)
	}
	return scanObservedMACRows(rows)
}

// FindAll 获取所有实测 MAC 绑定记录，按最后观测时间降序。
func (r *observedMACBindingRepo) FindAll(ctx context.Context) ([]identity.ObservedMACBinding, error) {
	var rows []observedMACBindingRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+observedMACSelectCols+` FROM observed_mac_bindings ORDER BY last_seen_at DESC`); err != nil {
		return nil, fmt.Errorf("observed mac binding repo: find all: %w", err)
	}
	return scanObservedMACRows(rows)
}

func scanObservedMACRows(rows []observedMACBindingRow) ([]identity.ObservedMACBinding, error) {
	bindings := make([]identity.ObservedMACBinding, 0, len(rows))
	for _, row := range rows {
		id, err := uuid.Parse(row.ID)
		if err != nil {
			return nil, fmt.Errorf("observed mac binding repo: parse id: %w", err)
		}
		bindings = append(bindings, identity.ObservedMACBinding{
			ID:         id,
			Office:     row.Office,
			IP:         row.IP,
			MAC:        row.MAC,
			LastSeenAt: row.LastSeenAt,
		})
	}
	return bindings, nil
}
