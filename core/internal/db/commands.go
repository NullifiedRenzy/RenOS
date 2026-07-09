package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Command struct {
	ID          string     `json:"id"`
	DeviceID    string     `json:"device_id"`
	UserID      string     `json:"user_id"`
	Command     string     `json:"command"`
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

type CommandRepository struct {
	pool *pgxpool.Pool
}

func NewCommandRepository(pool *pgxpool.Pool) *CommandRepository {
	return &CommandRepository{pool: pool}
}

func (r *CommandRepository) Create(ctx context.Context, deviceID, userID, command string) (*Command, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO commands (device_id, user_id, command, status)
		 VALUES ($1, $2, $3, 'pending')
		 RETURNING id, device_id, user_id, command, status, exit_code, output, error,
		           requested_at, started_at, completed_at, created_at, updated_at`,
		deviceID, userID, command,
	)
	return scanCommand(row)
}

func (r *CommandRepository) FindByID(ctx context.Context, id string) (*Command, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, device_id, user_id, command, status, exit_code, output, error,
		        requested_at, started_at, completed_at, created_at, updated_at
		 FROM commands WHERE id = $1`,
		id,
	)
	return scanCommand(row)
}

func (r *CommandRepository) ListByDevice(ctx context.Context, deviceID string) ([]*Command, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, user_id, command, status, exit_code, output, error,
		        requested_at, started_at, completed_at, created_at, updated_at
		 FROM commands WHERE device_id = $1 ORDER BY requested_at DESC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list commands: %w", err)
	}
	defer rows.Close()

	var commands []*Command
	for rows.Next() {
		c, err := scanCommand(rows)
		if err != nil {
			return nil, err
		}
		commands = append(commands, c)
	}
	return commands, rows.Err()
}

func (r *CommandRepository) PendingByDevice(ctx context.Context, deviceID string) ([]*Command, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, user_id, command, status, exit_code, output, error,
		        requested_at, started_at, completed_at, created_at, updated_at
		 FROM commands WHERE device_id = $1 AND status = 'pending'
		 ORDER BY requested_at ASC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("pending commands: %w", err)
	}
	defer rows.Close()

	var commands []*Command
	for rows.Next() {
		c, err := scanCommand(rows)
		if err != nil {
			return nil, err
		}
		commands = append(commands, c)
	}
	return commands, rows.Err()
}

func (r *CommandRepository) UpdateResult(ctx context.Context, id, status string, exitCode int, output, errMsg string) (*Command, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE commands
		 SET status = $1, exit_code = $2, output = $3, error = $4,
		     started_at = COALESCE(started_at, NOW()),
		     completed_at = NOW(), updated_at = NOW()
		 WHERE id = $5
		 RETURNING id, device_id, user_id, command, status, exit_code, output, error,
		           requested_at, started_at, completed_at, created_at, updated_at`,
		status, exitCode, output, errMsg, id,
	)
	return scanCommand(row)
}

func scanCommand(row scanner) (*Command, error) {
	c := &Command{}
	err := row.Scan(&c.ID, &c.DeviceID, &c.UserID, &c.Command, &c.Status,
		&c.ExitCode, &c.Output, &c.Error,
		&c.RequestedAt, &c.StartedAt, &c.CompletedAt, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan command: %w", err)
	}
	return c, nil
}
