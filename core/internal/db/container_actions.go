package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ContainerAction struct {
	ID          string     `json:"id"`
	DeviceID    string     `json:"device_id"`
	UserID      string     `json:"user_id"`
	ContainerID string     `json:"container_id"`
	Action      string     `json:"action"`
	Status      string     `json:"status"`
	ExitCode    *int       `json:"exit_code,omitempty"`
	Output      *string    `json:"output,omitempty"`
	Error       *string    `json:"error,omitempty"`
	RequestedAt time.Time  `json:"requested_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type ContainerActionRepository struct {
	pool *pgxpool.Pool
}

func NewContainerActionRepository(pool *pgxpool.Pool) *ContainerActionRepository {
	return &ContainerActionRepository{pool: pool}
}

func (r *ContainerActionRepository) Create(ctx context.Context, deviceID, userID, containerID, action string) (*ContainerAction, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO container_actions (device_id, user_id, container_id, action, status)
		 VALUES ($1, $2, $3, $4, 'pending')
		 RETURNING id, device_id, user_id, container_id, action, status, exit_code, output, error,
		           requested_at, started_at, completed_at, created_at, updated_at`,
		deviceID, userID, containerID, action,
	)
	return scanContainerAction(row)
}

func (r *ContainerActionRepository) FindByID(ctx context.Context, id string) (*ContainerAction, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, device_id, user_id, container_id, action, status, exit_code, output, error,
		        requested_at, started_at, completed_at, created_at, updated_at
		 FROM container_actions WHERE id = $1`,
		id,
	)
	return scanContainerAction(row)
}

func (r *ContainerActionRepository) PendingByDevice(ctx context.Context, deviceID string) ([]*ContainerAction, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, user_id, container_id, action, status, exit_code, output, error,
		        requested_at, started_at, completed_at, created_at, updated_at
		 FROM container_actions WHERE device_id = $1 AND status = 'pending'
		 ORDER BY requested_at ASC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("pending container actions: %w", err)
	}
	defer rows.Close()

	var actions []*ContainerAction
	for rows.Next() {
		a, err := scanContainerAction(rows)
		if err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

func (r *ContainerActionRepository) ListByDevice(ctx context.Context, deviceID string) ([]*ContainerAction, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, user_id, container_id, action, status, exit_code, output, error,
		        requested_at, started_at, completed_at, created_at, updated_at
		 FROM container_actions WHERE device_id = $1 ORDER BY requested_at DESC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list container actions: %w", err)
	}
	defer rows.Close()

	var actions []*ContainerAction
	for rows.Next() {
		a, err := scanContainerAction(rows)
		if err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

func (r *ContainerActionRepository) UpdateResult(ctx context.Context, id, status string, exitCode int, output, errMsg string) (*ContainerAction, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE container_actions
		 SET status = $1, exit_code = $2, output = $3, error = $4,
		     started_at = COALESCE(started_at, NOW()),
		     completed_at = NOW(), updated_at = NOW()
		 WHERE id = $5
		 RETURNING id, device_id, user_id, container_id, action, status, exit_code, output, error,
		           requested_at, started_at, completed_at, created_at, updated_at`,
		status, exitCode, output, errMsg, id,
	)
	return scanContainerAction(row)
}

func scanContainerAction(row scanner) (*ContainerAction, error) {
	a := &ContainerAction{}
	err := row.Scan(&a.ID, &a.DeviceID, &a.UserID, &a.ContainerID, &a.Action, &a.Status,
		&a.ExitCode, &a.Output, &a.Error,
		&a.RequestedAt, &a.StartedAt, &a.CompletedAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan container action: %w", err)
	}
	return a, nil
}
