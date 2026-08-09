package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SystemLog struct {
	ID        string          `json:"id"`
	Level     string          `json:"level"`
	Source    string          `json:"source"`
	Message   string          `json:"message"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
}

type LogFilter struct {
	Level    string
	Source   string
	Since    *time.Time
	Until    *time.Time
	Offset   int
	Limit    int
}

type SystemLogRepository struct {
	pool *pgxpool.Pool
}

func NewSystemLogRepository(pool *pgxpool.Pool) *SystemLogRepository {
	return &SystemLogRepository{pool: pool}
}

func (r *SystemLogRepository) Create(ctx context.Context, level, source, message string, metadata json.RawMessage) (*SystemLog, error) {
	if metadata == nil {
		metadata = json.RawMessage("{}")
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO system_logs (level, source, message, metadata)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, level, source, message, metadata, created_at`,
		level, source, message, metadata,
	)
	return scanLog(row)
}

func (r *SystemLogRepository) FindAll(ctx context.Context, filter *LogFilter) ([]*SystemLog, int, error) {
	if filter == nil {
		filter = &LogFilter{Limit: 50}
	}
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}

	where := ""
	args := []interface{}{}
	argIdx := 1

	if filter.Level != "" {
		where += fmt.Sprintf(" AND level = $%d", argIdx)
		args = append(args, filter.Level)
		argIdx++
	}
	if filter.Source != "" {
		where += fmt.Sprintf(" AND source = $%d", argIdx)
		args = append(args, filter.Source)
		argIdx++
	}
	if filter.Since != nil {
		where += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *filter.Since)
		argIdx++
	}
	if filter.Until != nil {
		where += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *filter.Until)
		argIdx++
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM system_logs WHERE 1=1` + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count logs: %w", err)
	}

	query := `SELECT id, level, source, message, metadata, created_at FROM system_logs WHERE 1=1` + where +
		` ORDER BY created_at DESC LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list logs: %w", err)
	}
	defer rows.Close()

	var list []*SystemLog
	for rows.Next() {
		l, err := scanLog(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan log: %w", err)
		}
		list = append(list, l)
	}
	return list, total, rows.Err()
}

func scanLog(s scanner) (*SystemLog, error) {
	var l SystemLog
	var meta []byte
	err := s.Scan(&l.ID, &l.Level, &l.Source, &l.Message, &meta, &l.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan log: %w", err)
	}
	if meta != nil {
		l.Metadata = json.RawMessage(meta)
	}
	return &l, nil
}
