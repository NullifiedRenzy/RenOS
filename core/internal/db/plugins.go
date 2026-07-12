package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Plugin struct {
	ID          string     `json:"id"`
	DeviceID    string     `json:"device_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	PluginType  string     `json:"plugin_type"`
	Status      string     `json:"status"`
	LastOutput  string     `json:"last_output"`
	LastExitCode int32     `json:"last_exit_code"`
	LastRunAt   *time.Time `json:"last_run_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type PluginRepository struct {
	pool *pgxpool.Pool
}

func NewPluginRepository(pool *pgxpool.Pool) *PluginRepository {
	return &PluginRepository{pool: pool}
}

func (r *PluginRepository) ReplaceForDevice(ctx context.Context, deviceID string, plugins []*Plugin) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM plugins WHERE device_id = $1`, deviceID); err != nil {
		return fmt.Errorf("delete plugins: %w", err)
	}

	for _, p := range plugins {
		if _, err := tx.Exec(ctx,
			`INSERT INTO plugins (device_id, name, description, plugin_type, status, last_output, last_exit_code, last_run_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			deviceID, p.Name, p.Description, p.PluginType, p.Status, p.LastOutput, p.LastExitCode, p.LastRunAt,
		); err != nil {
			return fmt.Errorf("insert plugin: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PluginRepository) ListByDevice(ctx context.Context, deviceID string) ([]*Plugin, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, name, description, plugin_type, status,
		        last_output, last_exit_code, last_run_at, created_at, updated_at
		 FROM plugins WHERE device_id = $1 ORDER BY name ASC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list plugins: %w", err)
	}
	defer rows.Close()

	var plugins []*Plugin
	for rows.Next() {
		p, err := scanPlugin(rows)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, p)
	}
	return plugins, rows.Err()
}

func scanPlugin(row scanner) (*Plugin, error) {
	p := &Plugin{}
	err := row.Scan(&p.ID, &p.DeviceID, &p.Name, &p.Description, &p.PluginType,
		&p.Status, &p.LastOutput, &p.LastExitCode, &p.LastRunAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan plugin: %w", err)
	}
	return p, nil
}
