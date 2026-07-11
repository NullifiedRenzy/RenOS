package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FileAction struct {
	ID          string          `json:"id"`
	DeviceID    string          `json:"device_id"`
	UserID      string          `json:"user_id"`
	ActionType  string          `json:"action_type"`
	Path        string          `json:"path"`
	Query       string          `json:"query"`
	Status      string          `json:"status"`
	ExitCode    *int            `json:"exit_code,omitempty"`
	Output      *string         `json:"output,omitempty"`
	Error       *string         `json:"error,omitempty"`
	ResultJSON  json.RawMessage `json:"result_json,omitempty"`
	RequestedAt time.Time       `json:"requested_at"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type FileActionRepository struct {
	pool *pgxpool.Pool
}

func NewFileActionRepository(pool *pgxpool.Pool) *FileActionRepository {
	return &FileActionRepository{pool: pool}
}

func (r *FileActionRepository) Create(ctx context.Context, deviceID, userID, actionType, path, query string) (*FileAction, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO file_actions (device_id, user_id, action_type, path, query, status)
		 VALUES ($1, $2, $3, $4, $5, 'pending')
		 RETURNING id, device_id, user_id, action_type, path, query, status,
		           exit_code, output, error, result_json,
		           requested_at, started_at, completed_at, created_at, updated_at`,
		deviceID, userID, actionType, path, query,
	)
	return scanFileAction(row)
}

func (r *FileActionRepository) FindByID(ctx context.Context, id string) (*FileAction, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, device_id, user_id, action_type, path, query, status,
		        exit_code, output, error, result_json,
		        requested_at, started_at, completed_at, created_at, updated_at
		 FROM file_actions WHERE id = $1`,
		id,
	)
	return scanFileAction(row)
}

func (r *FileActionRepository) PendingByDevice(ctx context.Context, deviceID string) ([]*FileAction, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, user_id, action_type, path, query, status,
		        exit_code, output, error, result_json,
		        requested_at, started_at, completed_at, created_at, updated_at
		 FROM file_actions WHERE device_id = $1 AND status = 'pending'
		 ORDER BY requested_at ASC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("pending file actions: %w", err)
	}
	defer rows.Close()

	var actions []*FileAction
	for rows.Next() {
		a, err := scanFileAction(rows)
		if err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

func (r *FileActionRepository) ListByDevice(ctx context.Context, deviceID string) ([]*FileAction, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, user_id, action_type, path, query, status,
		        exit_code, output, error, result_json,
		        requested_at, started_at, completed_at, created_at, updated_at
		 FROM file_actions WHERE device_id = $1 ORDER BY requested_at DESC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list file actions: %w", err)
	}
	defer rows.Close()

	var actions []*FileAction
	for rows.Next() {
		a, err := scanFileAction(rows)
		if err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

func (r *FileActionRepository) UpdateResult(ctx context.Context, id, status string, exitCode int, output, errMsg string, resultJSON json.RawMessage) (*FileAction, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE file_actions
		 SET status = $1, exit_code = $2, output = $3, error = $4, result_json = $5,
		     started_at = COALESCE(started_at, NOW()),
		     completed_at = NOW(), updated_at = NOW()
		 WHERE id = $6
		 RETURNING id, device_id, user_id, action_type, path, query, status,
		           exit_code, output, error, result_json,
		           requested_at, started_at, completed_at, created_at, updated_at`,
		status, exitCode, output, errMsg, resultJSON, id,
	)
	return scanFileAction(row)
}

func scanFileAction(row scanner) (*FileAction, error) {
	a := &FileAction{}
	err := row.Scan(&a.ID, &a.DeviceID, &a.UserID, &a.ActionType, &a.Path, &a.Query, &a.Status,
		&a.ExitCode, &a.Output, &a.Error, &a.ResultJSON,
		&a.RequestedAt, &a.StartedAt, &a.CompletedAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan file action: %w", err)
	}
	return a, nil
}
