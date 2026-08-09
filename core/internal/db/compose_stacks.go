package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ComposeStack struct {
	ID          string    `json:"id"`
	DeviceID    string    `json:"device_id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	ConfigFiles string    `json:"config_files"`
	Services    string    `json:"services"`
	WorkingDir  string    `json:"working_dir"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ComposeStackRepository struct {
	pool *pgxpool.Pool
}

func NewComposeStackRepository(pool *pgxpool.Pool) *ComposeStackRepository {
	return &ComposeStackRepository{pool: pool}
}

func (r *ComposeStackRepository) ReplaceForDevice(ctx context.Context, deviceID string, stacks []*ComposeStack) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM compose_stacks WHERE device_id = $1`, deviceID); err != nil {
		return fmt.Errorf("delete old compose stacks: %w", err)
	}

	for _, s := range stacks {
		s.DeviceID = deviceID
		if _, err := tx.Exec(ctx,
			`INSERT INTO compose_stacks (device_id, name, status, config_files, services, working_dir)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (device_id, name) DO UPDATE SET
			   status = EXCLUDED.status, config_files = EXCLUDED.config_files,
			   services = EXCLUDED.services, working_dir = EXCLUDED.working_dir, updated_at = NOW()`,
			s.DeviceID, s.Name, s.Status, s.ConfigFiles, s.Services, s.WorkingDir,
		); err != nil {
			return fmt.Errorf("insert compose stack: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *ComposeStackRepository) ListByDevice(ctx context.Context, deviceID string) ([]*ComposeStack, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, name, status, config_files, services, working_dir, created_at, updated_at
		 FROM compose_stacks WHERE device_id = $1 ORDER BY name`, deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list compose stacks: %w", err)
	}
	defer rows.Close()

	var list []*ComposeStack
	for rows.Next() {
		s := &ComposeStack{}
		if err := rows.Scan(&s.ID, &s.DeviceID, &s.Name, &s.Status, &s.ConfigFiles, &s.Services, &s.WorkingDir, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan compose stack: %w", err)
		}
		list = append(list, s)
	}
	return list, rows.Err()
}
