package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DeviceUpdate struct {
	ID          string    `json:"id"`
	DeviceID    string    `json:"device_id"`
	UpdateType  string    `json:"update_type"`
	PackageName string    `json:"package_name"`
	CurrentVer  string    `json:"current_version"`
	AvailableVer string   `json:"available_version"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UpdateStatusRepository struct {
	pool *pgxpool.Pool
}

func NewUpdateStatusRepository(pool *pgxpool.Pool) *UpdateStatusRepository {
	return &UpdateStatusRepository{pool: pool}
}

func (r *UpdateStatusRepository) Upsert(ctx context.Context, u *DeviceUpdate) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO device_updates (device_id, update_type, package_name, current_version, available_version, status)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (device_id, package_name) DO UPDATE SET current_version=$4, available_version=$5, status=$6, updated_at=NOW()`,
		u.DeviceID, u.UpdateType, u.PackageName, u.CurrentVer, u.AvailableVer, u.Status,
	)
	return err
}

func (r *UpdateStatusRepository) ListByDevice(ctx context.Context, deviceID string) ([]*DeviceUpdate, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, update_type, package_name, current_version, available_version, status, created_at, updated_at
		 FROM device_updates WHERE device_id=$1 ORDER BY package_name`, deviceID)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []*DeviceUpdate
	for rows.Next() {
		u := &DeviceUpdate{}
		if err := rows.Scan(&u.ID, &u.DeviceID, &u.UpdateType, &u.PackageName, &u.CurrentVer, &u.AvailableVer, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, u)
	}
	return list, rows.Err()
}
