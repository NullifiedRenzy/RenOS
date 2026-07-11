package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ServiceAction struct {
	ID          string     `json:"id"`
	DeviceID    string     `json:"device_id"`
	UserID      string     `json:"user_id"`
	UnitName    string     `json:"unit_name"`
	Action      string     `json:"action"`
	Status      string     `json:"status"`
	ExitCode    *int       `json:"exit_code"`
	Output      string     `json:"output"`
	Error       string     `json:"error"`
	RequestedAt time.Time  `json:"requested_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type ServiceActionRepository struct {
	pool *pgxpool.Pool
}

func NewServiceActionRepository(pool *pgxpool.Pool) *ServiceActionRepository {
	return &ServiceActionRepository{pool: pool}
}

func (r *ServiceActionRepository) Create(ctx context.Context, deviceID, userID, unitName, action string) (*ServiceAction, error) {
	act := &ServiceAction{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO service_actions (device_id, user_id, unit_name, action)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, device_id, user_id, unit_name, action, status, exit_code, output, error, requested_at, created_at, updated_at`,
		deviceID, userID, unitName, action,
	).Scan(&act.ID, &act.DeviceID, &act.UserID, &act.UnitName, &act.Action,
		&act.Status, &act.ExitCode, &act.Output, &act.Error,
		&act.RequestedAt, &act.CreatedAt, &act.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create service_action: %w", err)
	}
	return act, nil
}

func (r *ServiceActionRepository) PendingByDevice(ctx context.Context, deviceID string) ([]*ServiceAction, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, user_id, unit_name, action, status, exit_code, output, error, requested_at, created_at, updated_at
		 FROM service_actions
		 WHERE device_id = $1 AND status = 'pending'
		 ORDER BY requested_at ASC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list pending service_actions: %w", err)
	}
	defer rows.Close()

	var actions []*ServiceAction
	for rows.Next() {
		a, err := scanServiceAction(rows)
		if err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

func (r *ServiceActionRepository) FindByID(ctx context.Context, id string) (*ServiceAction, error) {
	a := &ServiceAction{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, device_id, user_id, unit_name, action, status, exit_code, output, error, requested_at, created_at, updated_at
		 FROM service_actions WHERE id = $1`,
		id,
	).Scan(&a.ID, &a.DeviceID, &a.UserID, &a.UnitName, &a.Action,
		&a.Status, &a.ExitCode, &a.Output, &a.Error,
		&a.RequestedAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("find service_action: %w", err)
	}
	return a, nil
}

func (r *ServiceActionRepository) UpdateResult(ctx context.Context, id, status string, exitCode int, output, error string) (*ServiceAction, error) {
	a := &ServiceAction{}
	err := r.pool.QueryRow(ctx,
		`UPDATE service_actions
		 SET status = $2, exit_code = $3, output = $4, error = $5
		 WHERE id = $1
		 RETURNING id, device_id, user_id, unit_name, action, status, exit_code, output, error, requested_at, created_at, updated_at`,
		id, status, exitCode, output, error,
	).Scan(&a.ID, &a.DeviceID, &a.UserID, &a.UnitName, &a.Action,
		&a.Status, &a.ExitCode, &a.Output, &a.Error,
		&a.RequestedAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update service_action: %w", err)
	}
	return a, nil
}

func scanServiceAction(row scanner) (*ServiceAction, error) {
	a := &ServiceAction{}
	err := row.Scan(&a.ID, &a.DeviceID, &a.UserID, &a.UnitName, &a.Action,
		&a.Status, &a.ExitCode, &a.Output, &a.Error,
		&a.RequestedAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan service_action: %w", err)
	}
	return a, nil
}
