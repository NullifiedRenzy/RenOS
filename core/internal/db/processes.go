package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProcessSnapshot struct {
	ID          string    `json:"id"`
	DeviceID    string    `json:"device_id"`
	PID         int       `json:"pid"`
	Name        string    `json:"name"`
	Executable  string    `json:"executable"`
	CommandLine string    `json:"command_line"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryBytes int64     `json:"memory_bytes"`
	MemoryUsage float64   `json:"memory_usage"`
	Threads     int       `json:"threads"`
	OpenHandles int       `json:"open_handles"`
	ParentPID   int       `json:"parent_pid"`
	Owner       string    `json:"owner"`
	Status      string    `json:"status"`
	StartTime   int64     `json:"start_time"`
	CollectedAt time.Time `json:"collected_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type ProcessAction struct {
	ID          string     `json:"id"`
	DeviceID    string     `json:"device_id"`
	UserID      string     `json:"user_id"`
	PID         int        `json:"pid"`
	ActionType  string     `json:"action_type"`
	Status      string     `json:"status"`
	ExitCode    *int       `json:"exit_code,omitempty"`
	Output      string     `json:"output"`
	Error       string     `json:"error"`
	RequestedAt time.Time  `json:"requested_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type ProcessHistoryEntry struct {
	ID               int64     `json:"id"`
	DeviceID         string    `json:"device_id"`
	TotalCPU         float64   `json:"total_cpu"`
	TotalMemoryBytes int64     `json:"total_memory_bytes"`
	TotalMemoryUsage float64   `json:"total_memory_usage"`
	ProcessCount     int       `json:"process_count"`
	RecordedAt       time.Time `json:"recorded_at"`
}

type ProcessRepository struct {
	pool *pgxpool.Pool
}

func NewProcessRepository(pool *pgxpool.Pool) *ProcessRepository {
	return &ProcessRepository{pool: pool}
}

// UpsertSnapshot replaces all process data for a device (full refresh each cycle)
func (r *ProcessRepository) UpsertSnapshot(ctx context.Context, deviceID string, processes []*ProcessSnapshot) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete old snapshots for this device
	if _, err := tx.Exec(ctx, `DELETE FROM process_snapshots WHERE device_id = $1`, deviceID); err != nil {
		return fmt.Errorf("delete old snapshots: %w", err)
	}

	// Insert new snapshots in batches
	batch := &pgx.Batch{}
	for _, p := range processes {
		batch.Queue(
			`INSERT INTO process_snapshots (device_id, pid, name, executable, command_line, cpu_usage, memory_bytes, memory_usage, threads, open_handles, parent_pid, owner, status, start_time)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
			deviceID, p.PID, p.Name, p.Executable, p.CommandLine, p.CPUUsage, p.MemoryBytes, p.MemoryUsage,
			p.Threads, p.OpenHandles, p.ParentPID, p.Owner, p.Status, p.StartTime,
		)
	}

	br := tx.SendBatch(ctx, batch)
	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return fmt.Errorf("insert snapshot %d: %w", i, err)
		}
	}
	br.Close()

	return tx.Commit(ctx)
}

// ListByDevice returns current process snapshots for a device
func (r *ProcessRepository) ListByDevice(ctx context.Context, deviceID string) ([]*ProcessSnapshot, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, pid, name, executable, command_line, cpu_usage, memory_bytes, memory_usage,
		        threads, open_handles, parent_pid, owner, status, start_time, collected_at, created_at
		 FROM process_snapshots WHERE device_id = $1
		 ORDER BY cpu_usage DESC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list process snapshots: %w", err)
	}
	defer rows.Close()

	var list []*ProcessSnapshot
	for rows.Next() {
		p, err := scanProcessSnapshot(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// SearchByName returns processes matching a name query
func (r *ProcessRepository) SearchByName(ctx context.Context, deviceID, query string) ([]*ProcessSnapshot, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, pid, name, executable, command_line, cpu_usage, memory_bytes, memory_usage,
		        threads, open_handles, parent_pid, owner, status, start_time, collected_at, created_at
		 FROM process_snapshots WHERE device_id = $1 AND (name ILIKE '%' || $2 || '%' OR executable ILIKE '%' || $2 || '%' OR command_line ILIKE '%' || $2 || '%')
		 ORDER BY cpu_usage DESC`,
		deviceID, query,
	)
	if err != nil {
		return nil, fmt.Errorf("search processes: %w", err)
	}
	defer rows.Close()

	var list []*ProcessSnapshot
	for rows.Next() {
		p, err := scanProcessSnapshot(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// GetAggregates returns total CPU, memory, and process count for a device
func (r *ProcessRepository) GetAggregates(ctx context.Context, deviceID string) (float64, int64, float64, int, error) {
	var totalCPU float64
	var totalMemBytes int64
	var totalMemUsage float64
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(cpu_usage), 0), COALESCE(SUM(memory_bytes), 0), COALESCE(SUM(memory_usage), 0), COUNT(*)
		 FROM process_snapshots WHERE device_id = $1`,
		deviceID,
	).Scan(&totalCPU, &totalMemBytes, &totalMemUsage, &count)
	return totalCPU, totalMemBytes, totalMemUsage, count, err
}

// CreateAction records a process action request
func (r *ProcessRepository) CreateAction(ctx context.Context, deviceID, userID string, pid int, actionType string) (*ProcessAction, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO process_actions (device_id, user_id, pid, action_type, status)
		 VALUES ($1, $2, $3, $4, 'pending')
		 RETURNING id, device_id, user_id, pid, action_type, status, exit_code, output, error,
		           requested_at, started_at, completed_at, created_at, updated_at`,
		deviceID, userID, pid, actionType,
	)
	return scanProcessAction(row)
}

// FindActionByID returns a process action by ID
func (r *ProcessRepository) FindActionByID(ctx context.Context, id string) (*ProcessAction, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, device_id, user_id, pid, action_type, status, exit_code, output, error,
		        requested_at, started_at, completed_at, created_at, updated_at
		 FROM process_actions WHERE id = $1`, id,
	)
	return scanProcessAction(row)
}

// ListActionsByDevice returns action history for a device
func (r *ProcessRepository) ListActionsByDevice(ctx context.Context, deviceID string, limit, offset int) ([]*ProcessAction, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, user_id, pid, action_type, status, exit_code, output, error,
		        requested_at, started_at, completed_at, created_at, updated_at
		 FROM process_actions WHERE device_id = $1
		 ORDER BY requested_at DESC LIMIT $2 OFFSET $3`,
		deviceID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list process actions: %w", err)
	}
	defer rows.Close()

	var list []*ProcessAction
	for rows.Next() {
		a := &ProcessAction{}
		if err := rows.Scan(&a.ID, &a.DeviceID, &a.UserID, &a.PID, &a.ActionType, &a.Status,
			&a.ExitCode, &a.Output, &a.Error, &a.RequestedAt, &a.StartedAt, &a.CompletedAt,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan process action: %w", err)
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// UpdateActionStatus updates the status of a process action
func (r *ProcessRepository) UpdateActionStatus(ctx context.Context, id, status string, exitCode int, output, errMsg string) error {
	completedClause := ""
	if status == "completed" || status == "failed" {
		completedClause = ", completed_at = NOW()"
	}
	_, err := r.pool.Exec(ctx,
		fmt.Sprintf(`UPDATE process_actions SET status = $1, exit_code = $2, output = $3, error = $4, updated_at = NOW()%s WHERE id = $5`, completedClause),
		status, exitCode, output, errMsg, id,
	)
	return err
}

// RecordHistory records a point in the process history time series
func (r *ProcessRepository) RecordHistory(ctx context.Context, deviceID string, totalCPU float64, totalMemBytes int64, totalMemUsage float64, processCount int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO process_history (device_id, total_cpu, total_memory_bytes, total_memory_usage, process_count)
		 VALUES ($1, $2, $3, $4, $5)`,
		deviceID, totalCPU, totalMemBytes, totalMemUsage, processCount,
	)
	return err
}

// GetHistory returns process history for graphs (last N minutes)
func (r *ProcessRepository) GetHistory(ctx context.Context, deviceID string, minutes int) ([]*ProcessHistoryEntry, error) {
	if minutes <= 0 {
		minutes = 60
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, total_cpu, total_memory_bytes, total_memory_usage, process_count, recorded_at
		 FROM process_history WHERE device_id = $1 AND recorded_at > NOW() - ($2 || ' minutes')::interval
		 ORDER BY recorded_at ASC`,
		deviceID, fmt.Sprintf("%d", minutes),
	)
	if err != nil {
		return nil, fmt.Errorf("get process history: %w", err)
	}
	defer rows.Close()

	var list []*ProcessHistoryEntry
	for rows.Next() {
		h := &ProcessHistoryEntry{}
		if err := rows.Scan(&h.ID, &h.DeviceID, &h.TotalCPU, &h.TotalMemoryBytes, &h.TotalMemoryUsage, &h.ProcessCount, &h.RecordedAt); err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, rows.Err()
}

func scanProcessSnapshot(row scanner) (*ProcessSnapshot, error) {
	p := &ProcessSnapshot{}
	err := row.Scan(&p.ID, &p.DeviceID, &p.PID, &p.Name, &p.Executable, &p.CommandLine,
		&p.CPUUsage, &p.MemoryBytes, &p.MemoryUsage, &p.Threads, &p.OpenHandles,
		&p.ParentPID, &p.Owner, &p.Status, &p.StartTime, &p.CollectedAt, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan process snapshot: %w", err)
	}
	return p, nil
}

func scanProcessAction(row scanner) (*ProcessAction, error) {
	a := &ProcessAction{}
	err := row.Scan(&a.ID, &a.DeviceID, &a.UserID, &a.PID, &a.ActionType, &a.Status,
		&a.ExitCode, &a.Output, &a.Error, &a.RequestedAt, &a.StartedAt, &a.CompletedAt,
		&a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan process action: %w", err)
	}
	return a, nil
}
