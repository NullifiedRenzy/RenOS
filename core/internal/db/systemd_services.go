package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SystemdService struct {
	ID          string    `json:"id"`
	DeviceID    string    `json:"device_id"`
	UnitName    string    `json:"unit_name"`
	LoadState   string    `json:"load_state"`
	ActiveState string    `json:"active_state"`
	SubState    string    `json:"sub_state"`
	Description string    `json:"description"`
	Enabled     string    `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SystemdServiceRepository struct {
	pool *pgxpool.Pool
}

func NewSystemdServiceRepository(pool *pgxpool.Pool) *SystemdServiceRepository {
	return &SystemdServiceRepository{pool: pool}
}

func (r *SystemdServiceRepository) ReplaceForDevice(ctx context.Context, deviceID string, services []*SystemdService) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM systemd_services WHERE device_id = $1`, deviceID); err != nil {
		return fmt.Errorf("delete systemd_services: %w", err)
	}

	for _, svc := range services {
		if _, err := tx.Exec(ctx,
			`INSERT INTO systemd_services (device_id, unit_name, load_state, active_state, sub_state, description, enabled)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			deviceID, svc.UnitName, svc.LoadState, svc.ActiveState, svc.SubState, svc.Description, svc.Enabled,
		); err != nil {
			return fmt.Errorf("insert systemd_service: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *SystemdServiceRepository) ListByDevice(ctx context.Context, deviceID string) ([]*SystemdService, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, unit_name, load_state, active_state, sub_state, description, enabled, created_at, updated_at
		 FROM systemd_services WHERE device_id = $1 ORDER BY unit_name ASC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list systemd_services: %w", err)
	}
	defer rows.Close()

	var services []*SystemdService
	for rows.Next() {
		s, err := scanSystemdService(rows)
		if err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, rows.Err()
}

func scanSystemdService(row scanner) (*SystemdService, error) {
	s := &SystemdService{}
	err := row.Scan(&s.ID, &s.DeviceID, &s.UnitName, &s.LoadState, &s.ActiveState,
		&s.SubState, &s.Description, &s.Enabled, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan systemd_service: %w", err)
	}
	return s, nil
}
