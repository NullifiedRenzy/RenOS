package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DockerNetwork struct {
	ID        string    `json:"id"`
	DeviceID  string    `json:"device_id"`
	NetworkID string    `json:"network_id"`
	Name      string    `json:"name"`
	Driver    string    `json:"driver"`
	Scope     string    `json:"scope"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DockerNetworkRepository struct {
	pool *pgxpool.Pool
}

func NewDockerNetworkRepository(pool *pgxpool.Pool) *DockerNetworkRepository {
	return &DockerNetworkRepository{pool: pool}
}

func (r *DockerNetworkRepository) ReplaceForDevice(ctx context.Context, deviceID string, networks []*DockerNetwork) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM docker_networks WHERE device_id = $1`, deviceID); err != nil {
		return fmt.Errorf("delete docker_networks: %w", err)
	}

	for _, net := range networks {
		if _, err := tx.Exec(ctx,
			`INSERT INTO docker_networks (device_id, network_id, name, driver, scope)
			 VALUES ($1, $2, $3, $4, $5)`,
			deviceID, net.NetworkID, net.Name, net.Driver, net.Scope,
		); err != nil {
			return fmt.Errorf("insert docker_network: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *DockerNetworkRepository) ListByDevice(ctx context.Context, deviceID string) ([]*DockerNetwork, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, network_id, name, driver, scope, created_at, updated_at
		 FROM docker_networks WHERE device_id = $1 ORDER BY name ASC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list docker_networks: %w", err)
	}
	defer rows.Close()

	var networks []*DockerNetwork
	for rows.Next() {
		n, err := scanDockerNetwork(rows)
		if err != nil {
			return nil, err
		}
		networks = append(networks, n)
	}
	return networks, rows.Err()
}

func scanDockerNetwork(row scanner) (*DockerNetwork, error) {
	n := &DockerNetwork{}
	err := row.Scan(&n.ID, &n.DeviceID, &n.NetworkID, &n.Name, &n.Driver, &n.Scope,
		&n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan docker_network: %w", err)
	}
	return n, nil
}
