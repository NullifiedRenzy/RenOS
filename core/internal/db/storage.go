package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StorageDevice struct {
	ID             string    `json:"id"`
	DeviceID       string    `json:"device_id"`
	Name           string    `json:"name"`
	Size           int64     `json:"size"`
	DeviceType     string    `json:"device_type"`
	Model          string    `json:"model"`
	MountPoint     string    `json:"mount_point"`
	Filesystem     string    `json:"filesystem"`
	UUID           string    `json:"uuid"`
	IsRemovable    bool      `json:"is_removable"`
	IsReadonly     bool      `json:"is_readonly"`
	UsedBytes      int64     `json:"used_bytes"`
	AvailableBytes int64     `json:"available_bytes"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type StorageDeviceRepository struct {
	pool *pgxpool.Pool
}

func NewStorageDeviceRepository(pool *pgxpool.Pool) *StorageDeviceRepository {
	return &StorageDeviceRepository{pool: pool}
}

func (r *StorageDeviceRepository) ReplaceForDevice(ctx context.Context, deviceID string, devices []*StorageDevice) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM storage_devices WHERE device_id = $1`, deviceID); err != nil {
		return fmt.Errorf("delete storage_devices: %w", err)
	}

	for _, d := range devices {
		if _, err := tx.Exec(ctx,
			`INSERT INTO storage_devices (device_id, name, size, device_type, model, mount_point, filesystem, uuid, is_removable, is_readonly, used_bytes, available_bytes)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			deviceID, d.Name, d.Size, d.DeviceType, d.Model, d.MountPoint, d.Filesystem,
			d.UUID, d.IsRemovable, d.IsReadonly, d.UsedBytes, d.AvailableBytes,
		); err != nil {
			return fmt.Errorf("insert storage_device: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *StorageDeviceRepository) ListByDevice(ctx context.Context, deviceID string) ([]*StorageDevice, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, name, size, device_type, model, mount_point, filesystem, uuid,
		        is_removable, is_readonly, used_bytes, available_bytes, created_at, updated_at
		 FROM storage_devices WHERE device_id = $1 ORDER BY name ASC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list storage_devices: %w", err)
	}
	defer rows.Close()

	var list []*StorageDevice
	for rows.Next() {
		d, err := scanStorageDevice(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func scanStorageDevice(row scanner) (*StorageDevice, error) {
	d := &StorageDevice{}
	err := row.Scan(&d.ID, &d.DeviceID, &d.Name, &d.Size, &d.DeviceType, &d.Model,
		&d.MountPoint, &d.Filesystem, &d.UUID,
		&d.IsRemovable, &d.IsReadonly, &d.UsedBytes, &d.AvailableBytes,
		&d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan storage_device: %w", err)
	}
	return d, nil
}
