package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Group struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeviceCount int       `json:"device_count,omitempty"`
}

type GroupRepository struct {
	pool *pgxpool.Pool
}

func NewGroupRepository(pool *pgxpool.Pool) *GroupRepository {
	return &GroupRepository{pool: pool}
}

func (r *GroupRepository) Create(ctx context.Context, name, description, createdBy string) (*Group, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO groups (name, description, created_by)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, description, created_by, created_at, updated_at`,
		name, description, createdBy,
	)
	return scanGroup(row)
}

func (r *GroupRepository) FindByID(ctx context.Context, id string) (*Group, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT g.id, g.name, g.description, g.created_by, g.created_at, g.updated_at,
		        (SELECT COUNT(*) FROM device_groups dg WHERE dg.group_id = g.id) AS device_count
		 FROM groups g WHERE g.id = $1`,
		id,
	)
	return scanGroup(row)
}

func (r *GroupRepository) FindAll(ctx context.Context, limit, offset int) ([]*Group, error) {
	var lim *int
	if limit > 0 {
		lim = &limit
	}
	rows, err := r.pool.Query(ctx,
		`SELECT g.id, g.name, g.description, g.created_by, g.created_at, g.updated_at,
		        (SELECT COUNT(*) FROM device_groups dg WHERE dg.group_id = g.id) AS device_count
		 FROM groups g ORDER BY g.name LIMIT $1 OFFSET $2`,
		lim, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()

	var list []*Group
	for rows.Next() {
		g, err := scanGroup(rows)
		if err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		list = append(list, g)
	}
	return list, rows.Err()
}

func (r *GroupRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM groups`).Scan(&count)
	return count, err
}

func (r *GroupRepository) Update(ctx context.Context, id, name, description string) (*Group, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE groups SET name = $2, description = $3, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, name, description, created_by, created_at, updated_at`,
		id, name, description,
	)
	return scanGroup(row)
}

func (r *GroupRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM groups WHERE id = $1`, id)
	return err
}

func (r *GroupRepository) AddDevice(ctx context.Context, groupID, deviceID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO device_groups (group_id, device_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		groupID, deviceID,
	)
	return err
}

func (r *GroupRepository) RemoveDevice(ctx context.Context, groupID, deviceID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM device_groups WHERE group_id = $1 AND device_id = $2`,
		groupID, deviceID,
	)
	return err
}

func (r *GroupRepository) ListDeviceIDs(ctx context.Context, groupID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT device_id FROM device_groups WHERE group_id = $1 ORDER BY device_id`,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("list device ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan device id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *GroupRepository) ListDeviceIDsByGroupIDs(ctx context.Context, groupIDs []string) ([]string, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT device_id FROM device_groups WHERE group_id = ANY($1) ORDER BY device_id`,
		groupIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("list device ids by groups: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan device id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func scanGroup(s scanner) (*Group, error) {
	var g Group
	err := s.Scan(&g.ID, &g.Name, &g.Description, &g.CreatedBy, &g.CreatedAt, &g.UpdatedAt, &g.DeviceCount)
	if err != nil {
		return nil, fmt.Errorf("scan group: %w", err)
	}
	return &g, nil
}
