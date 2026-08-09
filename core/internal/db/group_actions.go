package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GroupAction struct {
	ID               string          `json:"id"`
	GroupID          *string         `json:"group_id,omitempty"`
	ActionType       string          `json:"action_type"`
	TargetType       string          `json:"target_type"`
	TargetValue      string          `json:"target_value"`
	Status           string          `json:"status"`
	RequestedBy      string          `json:"requested_by"`
	TotalDevices     int             `json:"total_devices"`
	CompletedDevices int             `json:"completed_devices"`
	FailedDevices    int             `json:"failed_devices"`
	ResultSummary    json.RawMessage `json:"result_summary"`
	CreatedAt        time.Time       `json:"created_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
}

type GroupActionRepository struct {
	pool *pgxpool.Pool
}

func NewGroupActionRepository(pool *pgxpool.Pool) *GroupActionRepository {
	return &GroupActionRepository{pool: pool}
}

func (r *GroupActionRepository) Create(ctx context.Context, groupID *string, actionType, targetType, targetValue, requestedBy string, totalDevices int) (*GroupAction, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO group_actions (group_id, action_type, target_type, target_value, requested_by, total_devices)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, group_id, action_type, target_type, target_value, status,
		           requested_by, total_devices, completed_devices, failed_devices,
		           result_summary, created_at, completed_at`,
		groupID, actionType, targetType, targetValue, requestedBy, totalDevices,
	)
	return scanGroupAction(row)
}

func (r *GroupActionRepository) FindByID(ctx context.Context, id string) (*GroupAction, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, group_id, action_type, target_type, target_value, status,
		        requested_by, total_devices, completed_devices, failed_devices,
		        result_summary, created_at, completed_at
		 FROM group_actions WHERE id = $1`,
		id,
	)
	return scanGroupAction(row)
}

func (r *GroupActionRepository) ListByGroup(ctx context.Context, groupID string) ([]*GroupAction, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, group_id, action_type, target_type, target_value, status,
		        requested_by, total_devices, completed_devices, failed_devices,
		        result_summary, created_at, completed_at
		 FROM group_actions
		 WHERE group_id = $1
		 ORDER BY created_at DESC`,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("list group actions: %w", err)
	}
	defer rows.Close()

	var list []*GroupAction
	for rows.Next() {
		a, err := scanGroupAction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan group action: %w", err)
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (r *GroupActionRepository) UpdateResult(ctx context.Context, id string, completedDelta, failedDelta int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE group_actions
		 SET completed_devices = completed_devices + $2,
		     failed_devices = failed_devices + $3,
		     status = CASE WHEN completed_devices + $2 + failed_devices + $3 >= total_devices THEN 'completed' ELSE 'running' END,
		     completed_at = CASE WHEN completed_devices + $2 + failed_devices + $3 >= total_devices THEN NOW() ELSE NULL END
		 WHERE id = $1`,
		id, completedDelta, failedDelta,
	)
	return err
}

func (r *GroupActionRepository) LinkCommand(ctx context.Context, groupActionID, commandID, deviceID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO group_action_commands (group_action_id, command_id, device_id)
		 VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		groupActionID, commandID, deviceID,
	)
	return err
}

func (r *GroupActionRepository) FindByCommandID(ctx context.Context, commandID string) (*GroupAction, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT ga.id, ga.group_id, ga.action_type, ga.target_type, ga.target_value, ga.status,
		        ga.requested_by, ga.total_devices, ga.completed_devices, ga.failed_devices,
		        ga.result_summary, ga.created_at, ga.completed_at
		 FROM group_actions ga
		 JOIN group_action_commands gac ON gac.group_action_id = ga.id
		 WHERE gac.command_id = $1`,
		commandID,
	)
	return scanGroupAction(row)
}

func scanGroupAction(s scanner) (*GroupAction, error) {
	var a GroupAction
	var summary []byte
	err := s.Scan(&a.ID, &a.GroupID, &a.ActionType, &a.TargetType, &a.TargetValue,
		&a.Status, &a.RequestedBy, &a.TotalDevices, &a.CompletedDevices, &a.FailedDevices,
		&summary, &a.CreatedAt, &a.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan group action: %w", err)
	}
	if summary != nil {
		a.ResultSummary = json.RawMessage(summary)
	}
	return &a, nil
}
