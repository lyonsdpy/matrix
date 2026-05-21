package pg_repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ObservedMACBinding 实测 MAC-IP 绑定关系，存储在 PostgreSQL。
type ObservedMACBinding struct {
	ID         uuid.UUID
	Office     string
	IP         string
	MAC        string
	LastSeenAt time.Time
}

type ObservedMACBindingRepository interface {
	Upsert(ctx context.Context, binding *ObservedMACBinding) error
	FindByIP(ctx context.Context, ip string) ([]ObservedMACBinding, error)
	FindByMAC(ctx context.Context, mac string) ([]ObservedMACBinding, error)
	FindAll(ctx context.Context) ([]ObservedMACBinding, error)
}

var _ ObservedMACBindingRepository = (*observedMACBindingRepo)(nil)

type observedMACBindingRepo struct {
	db *sqlx.DB
}

func NewObservedMACBindingRepository(db *sqlx.DB) ObservedMACBindingRepository {
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

func (r *observedMACBindingRepo) Upsert(ctx context.Context, binding *ObservedMACBinding) error {
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

func (r *observedMACBindingRepo) FindByIP(ctx context.Context, ip string) ([]ObservedMACBinding, error) {
	var rows []observedMACBindingRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+observedMACSelectCols+` FROM observed_mac_bindings WHERE ip = $1 ORDER BY last_seen_at DESC`,
		ip); err != nil {
		return nil, fmt.Errorf("observed mac binding repo: find by ip: %w", err)
	}
	return scanObservedMACRows(rows)
}

func (r *observedMACBindingRepo) FindByMAC(ctx context.Context, mac string) ([]ObservedMACBinding, error) {
	var rows []observedMACBindingRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+observedMACSelectCols+` FROM observed_mac_bindings WHERE mac = $1 ORDER BY last_seen_at DESC`,
		mac); err != nil {
		return nil, fmt.Errorf("observed mac binding repo: find by mac: %w", err)
	}
	return scanObservedMACRows(rows)
}

func (r *observedMACBindingRepo) FindAll(ctx context.Context) ([]ObservedMACBinding, error) {
	var rows []observedMACBindingRow
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT `+observedMACSelectCols+` FROM observed_mac_bindings ORDER BY last_seen_at DESC`); err != nil {
		return nil, fmt.Errorf("observed mac binding repo: find all: %w", err)
	}
	return scanObservedMACRows(rows)
}

func scanObservedMACRows(rows []observedMACBindingRow) ([]ObservedMACBinding, error) {
	bindings := make([]ObservedMACBinding, 0, len(rows))
	for _, row := range rows {
		id, err := uuid.Parse(row.ID)
		if err != nil {
			return nil, fmt.Errorf("observed mac binding repo: parse id: %w", err)
		}
		bindings = append(bindings, ObservedMACBinding{
			ID:         id,
			Office:     row.Office,
			IP:         row.IP,
			MAC:        row.MAC,
			LastSeenAt: row.LastSeenAt,
		})
	}
	return bindings, nil
}
