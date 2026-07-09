package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DockerVolume struct {
	ID         string    `json:"id"`
	DeviceID   string    `json:"device_id"`
	Name       string    `json:"name"`
	Driver     string    `json:"driver"`
	Mountpoint string    `json:"mountpoint"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type DockerVolumeRepository struct {
	pool *pgxpool.Pool
}

func NewDockerVolumeRepository(pool *pgxpool.Pool) *DockerVolumeRepository {
	return &DockerVolumeRepository{pool: pool}
}

func (r *DockerVolumeRepository) ReplaceForDevice(ctx context.Context, deviceID string, volumes []*DockerVolume) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM docker_volumes WHERE device_id = $1`, deviceID); err != nil {
		return fmt.Errorf("delete docker_volumes: %w", err)
	}

	for _, vol := range volumes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO docker_volumes (device_id, name, driver, mountpoint)
			 VALUES ($1, $2, $3, $4)`,
			deviceID, vol.Name, vol.Driver, vol.Mountpoint,
		); err != nil {
			return fmt.Errorf("insert docker_volume: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *DockerVolumeRepository) ListByDevice(ctx context.Context, deviceID string) ([]*DockerVolume, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, name, driver, mountpoint, created_at, updated_at
		 FROM docker_volumes WHERE device_id = $1 ORDER BY name ASC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list docker_volumes: %w", err)
	}
	defer rows.Close()

	var volumes []*DockerVolume
	for rows.Next() {
		v, err := scanDockerVolume(rows)
		if err != nil {
			return nil, err
		}
		volumes = append(volumes, v)
	}
	return volumes, rows.Err()
}

func scanDockerVolume(row scanner) (*DockerVolume, error) {
	v := &DockerVolume{}
	err := row.Scan(&v.ID, &v.DeviceID, &v.Name, &v.Driver, &v.Mountpoint, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan docker_volume: %w", err)
	}
	return v, nil
}
