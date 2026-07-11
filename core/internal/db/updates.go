package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UpdateStatus struct {
	ID               string    `json:"id"`
	DeviceID         string    `json:"device_id"`
	PackageName      string    `json:"package_name"`
	CurrentVersion   string    `json:"current_version"`
	AvailableVersion string    `json:"available_version"`
	Severity         string    `json:"severity"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type UpdateStatusRepository struct {
	pool *pgxpool.Pool
}

func NewUpdateStatusRepository(pool *pgxpool.Pool) *UpdateStatusRepository {
	return &UpdateStatusRepository{pool: pool}
}

func (r *UpdateStatusRepository) ReplaceForDevice(ctx context.Context, deviceID string, updates []*UpdateStatus) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM update_status WHERE device_id = $1`, deviceID); err != nil {
		return fmt.Errorf("delete update_status: %w", err)
	}

	for _, u := range updates {
		if _, err := tx.Exec(ctx,
			`INSERT INTO update_status (device_id, package_name, current_version, available_version, severity, status)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			deviceID, u.PackageName, u.CurrentVersion, u.AvailableVersion, u.Severity, u.Status,
		); err != nil {
			return fmt.Errorf("insert update_status: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *UpdateStatusRepository) ListByDevice(ctx context.Context, deviceID string) ([]*UpdateStatus, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, package_name, current_version, available_version,
		        severity, status, created_at, updated_at
		 FROM update_status WHERE device_id = $1 ORDER BY package_name ASC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list update_status: %w", err)
	}
	defer rows.Close()

	var updates []*UpdateStatus
	for rows.Next() {
		u, err := scanUpdateStatus(rows)
		if err != nil {
			return nil, err
		}
		updates = append(updates, u)
	}
	return updates, rows.Err()
}

func scanUpdateStatus(row scanner) (*UpdateStatus, error) {
	u := &UpdateStatus{}
	err := row.Scan(&u.ID, &u.DeviceID, &u.PackageName, &u.CurrentVersion,
		&u.AvailableVersion, &u.Severity, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan update_status: %w", err)
	}
	return u, nil
}
