package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Schedule struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload"`
	TargetType     string          `json:"target_type"`
	TargetIDs      json.RawMessage `json:"target_ids"`
	CronExpression string          `json:"cron_expression"`
	Enabled        bool            `json:"enabled"`
	LastRunAt      *time.Time      `json:"last_run_at,omitempty"`
	NextRunAt      *time.Time      `json:"next_run_at,omitempty"`
	CreatedBy      string          `json:"created_by"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type ScheduleRun struct {
	ID          string     `json:"id"`
	ScheduleID  string     `json:"schedule_id"`
	JobID       *string    `json:"job_id,omitempty"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type ScheduleRepository struct {
	pool *pgxpool.Pool
}

func NewScheduleRepository(pool *pgxpool.Pool) *ScheduleRepository {
	return &ScheduleRepository{pool: pool}
}

func (r *ScheduleRepository) Create(ctx context.Context, name, schedType string, payload json.RawMessage, targetType string, targetIDs json.RawMessage, cronExpression string, nextRunAt time.Time, createdBy string) (*Schedule, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO schedules (name, type, payload, target_type, target_ids, cron_expression, next_run_at, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, name, type, payload, target_type, target_ids, cron_expression,
		           enabled, last_run_at, next_run_at, created_by, created_at, updated_at`,
		name, schedType, payload, targetType, targetIDs, cronExpression, nextRunAt, createdBy,
	)
	return scanSchedule(row)
}

func (r *ScheduleRepository) FindByID(ctx context.Context, id string) (*Schedule, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, type, payload, target_type, target_ids, cron_expression,
		        enabled, last_run_at, next_run_at, created_by, created_at, updated_at
		 FROM schedules WHERE id = $1`,
		id,
	)
	return scanSchedule(row)
}

func (r *ScheduleRepository) FindAll(ctx context.Context, limit, offset int) ([]*Schedule, error) {
	var lim *int
	if limit > 0 {
		lim = &limit
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, type, payload, target_type, target_ids, cron_expression,
		        enabled, last_run_at, next_run_at, created_by, created_at, updated_at
		 FROM schedules ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		lim, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list schedules: %w", err)
	}
	defer rows.Close()

	var list []*Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *ScheduleRepository) Update(ctx context.Context, id, name, schedType string, payload json.RawMessage, targetType string, targetIDs json.RawMessage, cronExpression string, nextRunAt time.Time) (*Schedule, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE schedules
		 SET name = $2, type = $3, payload = $4, target_type = $5, target_ids = $6,
		     cron_expression = $7, next_run_at = $8, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, name, type, payload, target_type, target_ids, cron_expression,
		           enabled, last_run_at, next_run_at, created_by, created_at, updated_at`,
		id, name, schedType, payload, targetType, targetIDs, cronExpression, nextRunAt,
	)
	return scanSchedule(row)
}

func (r *ScheduleRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM schedules WHERE id = $1`, id)
	return err
}

func (r *ScheduleRepository) SetEnabled(ctx context.Context, id string, enabled bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE schedules SET enabled = $2, updated_at = NOW() WHERE id = $1`,
		id, enabled,
	)
	return err
}

func (r *ScheduleRepository) ListEnabledDue(ctx context.Context, now time.Time) ([]*Schedule, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, type, payload, target_type, target_ids, cron_expression,
		        enabled, last_run_at, next_run_at, created_by, created_at, updated_at
		 FROM schedules
		 WHERE enabled = true AND next_run_at <= $1
		 ORDER BY next_run_at ASC
		 LIMIT 50`,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("list due schedules: %w", err)
	}
	defer rows.Close()

	var list []*Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, fmt.Errorf("scan due schedule: %w", err)
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *ScheduleRepository) UpdateRunTimes(ctx context.Context, id string, lastRunAt, nextRunAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE schedules
		 SET last_run_at = $2, next_run_at = $3, updated_at = NOW()
		 WHERE id = $1`,
		id, lastRunAt, nextRunAt,
	)
	return err
}

// --- ScheduleRunRepository ---

type ScheduleRunRepository struct {
	pool *pgxpool.Pool
}

func NewScheduleRunRepository(pool *pgxpool.Pool) *ScheduleRunRepository {
	return &ScheduleRunRepository{pool: pool}
}

func (r *ScheduleRunRepository) Create(ctx context.Context, scheduleID, jobID, status string) (*ScheduleRun, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO schedule_runs (schedule_id, job_id, status)
		 VALUES ($1, $2, $3)
		 RETURNING id, schedule_id, job_id, status, started_at, completed_at`,
		scheduleID, jobID, status,
	)
	return scanScheduleRun(row)
}

func (r *ScheduleRunRepository) ListBySchedule(ctx context.Context, scheduleID string) ([]*ScheduleRun, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, schedule_id, job_id, status, started_at, completed_at
		 FROM schedule_runs WHERE schedule_id = $1
		 ORDER BY started_at DESC`,
		scheduleID,
	)
	if err != nil {
		return nil, fmt.Errorf("list schedule runs: %w", err)
	}
	defer rows.Close()

	var list []*ScheduleRun
	for rows.Next() {
		sr, err := scanScheduleRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan schedule run: %w", err)
		}
		list = append(list, sr)
	}
	return list, rows.Err()
}

func (r *ScheduleRunRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM schedule_runs`).Scan(&count)
	return count, err
}

func (r *ScheduleRunRepository) UpdateStatus(ctx context.Context, id, status string, completedAt *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE schedule_runs SET status = $2, completed_at = $3 WHERE id = $1`,
		id, status, completedAt,
	)
	return err
}

// --- scanners ---

func scanSchedule(s scanner) (*Schedule, error) {
	var sched Schedule
	var payload, targetIDs []byte
	err := s.Scan(&sched.ID, &sched.Name, &sched.Type, &payload, &sched.TargetType,
		&targetIDs, &sched.CronExpression, &sched.Enabled,
		&sched.LastRunAt, &sched.NextRunAt, &sched.CreatedBy,
		&sched.CreatedAt, &sched.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan schedule: %w", err)
	}
	if payload != nil {
		sched.Payload = json.RawMessage(payload)
	}
	if targetIDs != nil {
		sched.TargetIDs = json.RawMessage(targetIDs)
	}
	return &sched, nil
}

func scanScheduleRun(s scanner) (*ScheduleRun, error) {
	var sr ScheduleRun
	err := s.Scan(&sr.ID, &sr.ScheduleID, &sr.JobID, &sr.Status, &sr.StartedAt, &sr.CompletedAt)
	if err != nil {
		return nil, fmt.Errorf("scan schedule run: %w", err)
	}
	return &sr, nil
}
