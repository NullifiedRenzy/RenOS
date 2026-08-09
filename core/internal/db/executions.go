package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Execution struct {
	ID             string     `json:"id"`
	DeviceID       string     `json:"device_id"`
	UserID         string     `json:"user_id"`
	Command        string     `json:"command"`
	Status         string     `json:"status"`
	ExitCode       *int       `json:"exit_code,omitempty"`
	Shell          string     `json:"shell"`
	TimeoutMs      int64      `json:"timeout_ms"`
	MaxOutputBytes int64      `json:"max_output_bytes"`
	RequestedAt    time.Time  `json:"requested_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type ExecutionOutput struct {
	ID          int64     `json:"id"`
	ExecutionID string    `json:"execution_id"`
	Stream      string    `json:"stream"`
	Content     string    `json:"content"`
	Seq         int       `json:"seq"`
	CreatedAt   time.Time `json:"created_at"`
}

type ExecutionRepository struct {
	pool *pgxpool.Pool
}

func NewExecutionRepository(pool *pgxpool.Pool) *ExecutionRepository {
	return &ExecutionRepository{pool: pool}
}

func (r *ExecutionRepository) Create(ctx context.Context, deviceID, userID, command, shell string, timeoutMs, maxOutputBytes int64) (*Execution, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO executions (device_id, user_id, command, shell, timeout_ms, max_output_bytes)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, device_id, user_id, command, status, exit_code, shell, timeout_ms, max_output_bytes,
		           requested_at, started_at, completed_at, created_at, updated_at`,
		deviceID, userID, command, shell, timeoutMs, maxOutputBytes,
	)
	return scanExecution(row)
}

func (r *ExecutionRepository) FindByID(ctx context.Context, id string) (*Execution, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, device_id, user_id, command, status, exit_code, shell, timeout_ms, max_output_bytes,
		        requested_at, started_at, completed_at, created_at, updated_at
		 FROM executions WHERE id = $1`, id,
	)
	return scanExecution(row)
}

func (r *ExecutionRepository) ListByDevice(ctx context.Context, deviceID string, limit, offset int) ([]*Execution, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM executions WHERE device_id = $1`, deviceID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count executions: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, user_id, command, status, exit_code, shell, timeout_ms, max_output_bytes,
		        requested_at, started_at, completed_at, created_at, updated_at
		 FROM executions WHERE device_id = $1
		 ORDER BY requested_at DESC
		 LIMIT $2 OFFSET $3`,
		deviceID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list executions: %w", err)
	}
	defer rows.Close()

	var list []*Execution
	for rows.Next() {
		e, err := scanExecution(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, e)
	}
	return list, total, rows.Err()
}

func (r *ExecutionRepository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE executions SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id,
	)
	return err
}

func (r *ExecutionRepository) MarkCompleted(ctx context.Context, id string, exitCode int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE executions SET status = 'completed', exit_code = $1, completed_at = NOW(), updated_at = NOW() WHERE id = $2`,
		exitCode, id,
	)
	return err
}

func (r *ExecutionRepository) MarkFailed(ctx context.Context, id string, exitCode int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE executions SET status = 'failed', exit_code = $1, completed_at = NOW(), updated_at = NOW() WHERE id = $2`,
		exitCode, id,
	)
	return err
}

func (r *ExecutionRepository) MarkCancelled(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE executions SET status = 'cancelled', completed_at = NOW(), updated_at = NOW() WHERE id = $1 AND status IN ('pending','running')`,
		id,
	)
	return err
}

func (r *ExecutionRepository) AppendOutput(ctx context.Context, executionID, stream, content string, seq int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO execution_output (execution_id, stream, content, seq) VALUES ($1, $2, $3, $4)`,
		executionID, stream, content, seq,
	)
	return err
}

func (r *ExecutionRepository) GetOutput(ctx context.Context, executionID string) ([]*ExecutionOutput, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, execution_id, stream, content, seq, created_at
		 FROM execution_output WHERE execution_id = $1
		 ORDER BY seq ASC`,
		executionID,
	)
	if err != nil {
		return nil, fmt.Errorf("get output: %w", err)
	}
	defer rows.Close()

	var list []*ExecutionOutput
	for rows.Next() {
		o := &ExecutionOutput{}
		err := rows.Scan(&o.ID, &o.ExecutionID, &o.Stream, &o.Content, &o.Seq, &o.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

func (r *ExecutionRepository) HasRunning(ctx context.Context, deviceID string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM executions WHERE device_id = $1 AND status IN ('pending','running')`,
		deviceID,
	).Scan(&count)
	return count > 0, err
}

func scanExecution(row scanner) (*Execution, error) {
	e := &Execution{}
	err := row.Scan(&e.ID, &e.DeviceID, &e.UserID, &e.Command, &e.Status,
		&e.ExitCode, &e.Shell, &e.TimeoutMs, &e.MaxOutputBytes,
		&e.RequestedAt, &e.StartedAt, &e.CompletedAt, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan execution: %w", err)
	}
	return e, nil
}
