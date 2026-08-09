package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Container struct {
	ID          string    `json:"id"`
	DeviceID    string    `json:"device_id"`
	ContainerID string    `json:"container_id"`
	Name        string    `json:"name"`
	Image       string    `json:"image"`
	Status      string    `json:"status"`
	State       string    `json:"state"`
	Ports       string    `json:"ports"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ContainerRepository struct {
	pool *pgxpool.Pool
}

func NewContainerRepository(pool *pgxpool.Pool) *ContainerRepository {
	return &ContainerRepository{pool: pool}
}

func (r *ContainerRepository) ReplaceForDevice(ctx context.Context, deviceID string, containers []*Container) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM containers WHERE device_id = $1`, deviceID); err != nil {
		return fmt.Errorf("delete containers: %w", err)
	}

	batch := &pgx.Batch{}
	for _, c := range containers {
		batch.Queue(
			`INSERT INTO containers (device_id, container_id, name, image, status, state, ports)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			deviceID, c.ContainerID, c.Name, c.Image, c.Status, c.State, c.Ports,
		)
	}
	br := tx.SendBatch(ctx, batch)
	if err := br.Close(); err != nil {
		return fmt.Errorf("insert containers: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *ContainerRepository) ListByDevice(ctx context.Context, deviceID string) ([]*Container, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, container_id, name, image, status, state, ports, created_at, updated_at
		 FROM containers WHERE device_id = $1 ORDER BY name ASC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	defer rows.Close()

	var containers []*Container
	for rows.Next() {
		c, err := scanContainer(rows)
		if err != nil {
			return nil, err
		}
		containers = append(containers, c)
	}
	return containers, rows.Err()
}

func scanContainer(row scanner) (*Container, error) {
	c := &Container{}
	err := row.Scan(&c.ID, &c.DeviceID, &c.ContainerID, &c.Name, &c.Image,
		&c.Status, &c.State, &c.Ports, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan container: %w", err)
	}
	return c, nil
}
