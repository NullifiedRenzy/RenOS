package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Job struct {
	ID                   string          `json:"id"`
	Type                 string          `json:"type"`
	Payload              json.RawMessage `json:"payload"`
	Status               string          `json:"status"`
	TotalDevices         int             `json:"total_devices"`
	CompletedDevices     int             `json:"completed_devices"`
	FailedDevices        int             `json:"failed_devices"`
	Retries              int             `json:"retries"`
	MaxRetries           int             `json:"max_retries"`
	CreatedBy            string          `json:"created_by"`
	GroupActionID        *string         `json:"group_action_id,omitempty"`
	MaxJobDurationMs     int64           `json:"max_job_duration_ms"`
	MaxDeviceExecutionMs int64           `json:"max_device_execution_ms"`
	StopOnFailure        bool            `json:"stop_on_failure"`
	BatchSize            int             `json:"batch_size"`
	BatchDelayMs         int             `json:"batch_delay_ms"`
	StoppedReason        *string         `json:"stopped_reason,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	StartedAt            *time.Time      `json:"started_at,omitempty"`
	CompletedAt          *time.Time      `json:"completed_at,omitempty"`
	CancelledAt          *time.Time      `json:"cancelled_at,omitempty"`
}

type JobRepository struct {
	pool *pgxpool.Pool
}

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{pool: pool}
}

func (r *JobRepository) Create(ctx context.Context, jobType string, payload json.RawMessage, totalDevices int, createdBy string, groupActionID *string) (*Job, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO jobs (type, payload, total_devices, created_by, group_action_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, type, payload, status, total_devices, completed_devices, failed_devices,
		           retries, max_retries, created_by, group_action_id,
		           max_job_duration_ms, max_device_execution_ms, stop_on_failure,
		           batch_size, batch_delay_ms, stopped_reason,
		           created_at, started_at, completed_at, cancelled_at`,
		jobType, payload, totalDevices, createdBy, groupActionID,
	)
	return scanJob(row)
}

func (r *JobRepository) FindByID(ctx context.Context, id string) (*Job, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, type, payload, status, total_devices, completed_devices, failed_devices,
		        retries, max_retries, created_by, group_action_id,
		        max_job_duration_ms, max_device_execution_ms, stop_on_failure,
		        batch_size, batch_delay_ms, stopped_reason,
		        created_at, started_at, completed_at, cancelled_at
		 FROM jobs WHERE id = $1`,
		id,
	)
	return scanJob(row)
}

func (r *JobRepository) FindAll(ctx context.Context, limit, offset int) ([]*Job, error) {
	var lim *int
	if limit > 0 {
		lim = &limit
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, type, payload, status, total_devices, completed_devices, failed_devices,
		        retries, max_retries, created_by, group_action_id,
		        max_job_duration_ms, max_device_execution_ms, stop_on_failure,
		        batch_size, batch_delay_ms, stopped_reason,
		        created_at, started_at, completed_at, cancelled_at
		 FROM jobs ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		lim, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	var list []*Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		list = append(list, j)
	}
	return list, rows.Err()
}

func (r *JobRepository) Claim(ctx context.Context) (*Job, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE jobs SET status = 'running', started_at = NOW()
		 WHERE id = (
		     SELECT id FROM jobs WHERE status = 'pending'
		     ORDER BY created_at ASC LIMIT 1
		     FOR UPDATE SKIP LOCKED
		 )
		 RETURNING id, type, payload, status, total_devices, completed_devices, failed_devices,
		           retries, max_retries, created_by, group_action_id,
		           max_job_duration_ms, max_device_execution_ms, stop_on_failure,
		           batch_size, batch_delay_ms, stopped_reason,
		           created_at, started_at, completed_at, cancelled_at`,
	)
	return scanJob(row)
}

func (r *JobRepository) UpdateProgress(ctx context.Context, id string, completedInc, failedInc int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs
		 SET completed_devices = completed_devices + $2,
		     failed_devices = failed_devices + $3,
		     status = CASE
		         WHEN completed_devices + $2 + failed_devices + $3 >= total_devices THEN 'completed'
		         ELSE 'running'
		     END,
		     completed_at = CASE
		         WHEN completed_devices + $2 + failed_devices + $3 >= total_devices THEN NOW()
		         ELSE NULL
		     END
		 WHERE id = $1`,
		id, completedInc, failedInc,
	)
	return err
}

func (r *JobRepository) MarkFailed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs SET status = 'failed', completed_at = NOW() WHERE id = $1`,
		id,
	)
	return err
}

func (r *JobRepository) Cancel(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs SET status = 'cancelled', cancelled_at = NOW(), completed_at = NOW()
		 WHERE id = $1 AND status IN ('pending', 'running')`,
		id,
	)
	return err
}

func (r *JobRepository) MarkStopped(ctx context.Context, id, reason string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs SET status = 'stopped', completed_at = NOW(), stopped_reason = $2
		 WHERE id = $1 AND status = 'running'`, id, reason)
	return err
}

func (r *JobRepository) IncrementRetries(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs SET retries = retries + 1 WHERE id = $1`,
		id,
	)
	return err
}

func (r *JobRepository) ResetStatus(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs
		 SET status = 'running',
		     completed_devices = 0,
		     failed_devices = 0,
		     started_at = NOW(),
		     completed_at = NULL
		 WHERE id = $1 AND status = 'failed'`,
		id,
	)
	return err
}

func (r *JobRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	query := `SELECT COUNT(*) FROM jobs`
	if status != "" {
		query += ` WHERE status = $1`
		var count int64
		err := r.pool.QueryRow(ctx, query, status).Scan(&count)
		return count, err
	}
	var count int64
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	return count, err
}

type StatusCounts struct {
	Total    int64
	ByStatus map[string]int64
	AvgMs    float64
}

func (r *JobRepository) CountByStatuses(ctx context.Context) (*StatusCounts, error) {
	rows, err := r.pool.Query(ctx, `SELECT status, COUNT(*) FROM jobs GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &StatusCounts{ByStatus: make(map[string]int64)}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		result.ByStatus[status] = count
		result.Total += count
	}

	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at)) * 1000), 0)
		 FROM jobs WHERE completed_at IS NOT NULL AND started_at IS NOT NULL`,
	).Scan(&result.AvgMs)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func scanJob(s scanner) (*Job, error) {
	var j Job
	var payload []byte
	err := s.Scan(&j.ID, &j.Type, &payload, &j.Status,
		&j.TotalDevices, &j.CompletedDevices, &j.FailedDevices,
		&j.Retries, &j.MaxRetries, &j.CreatedBy, &j.GroupActionID,
		&j.MaxJobDurationMs, &j.MaxDeviceExecutionMs, &j.StopOnFailure,
		&j.BatchSize, &j.BatchDelayMs, &j.StoppedReason,
		&j.CreatedAt, &j.StartedAt, &j.CompletedAt, &j.CancelledAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan job: %w", err)
	}
	if payload != nil {
		j.Payload = json.RawMessage(payload)
	}
	return &j, nil
}
