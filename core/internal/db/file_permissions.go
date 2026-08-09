package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FilePermission struct {
	ID           string          `json:"id"`
	DeviceID     string          `json:"device_id"`
	UserID       string          `json:"user_id"`
	CanBrowse    bool            `json:"can_browse"`
	CanModify    bool            `json:"can_modify"`
	CanUpload    bool            `json:"can_upload"`
	CanDownload  bool            `json:"can_download"`
	CanDelete    bool            `json:"can_delete"`
	AllowedRoots json.RawMessage `json:"allowed_roots"`
	CreatedAt    interface{}     `json:"created_at"`
	UpdatedAt    interface{}     `json:"updated_at"`
}

type FilePermissionRepository struct {
	pool *pgxpool.Pool
}

func NewFilePermissionRepository(pool *pgxpool.Pool) *FilePermissionRepository {
	return &FilePermissionRepository{pool: pool}
}

func (r *FilePermissionRepository) GetForUser(ctx context.Context, deviceID, userID string) (*FilePermission, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, device_id, user_id, can_browse, can_modify, can_upload, can_download, can_delete, allowed_roots, created_at, updated_at
		 FROM file_permissions WHERE device_id = $1 AND user_id = $2`,
		deviceID, userID,
	)
	return scanFilePermission(row)
}

func (r *FilePermissionRepository) Upsert(ctx context.Context, deviceID, userID string, canBrowse, canModify, canUpload, canDownload, canDelete bool, allowedRoots json.RawMessage) (*FilePermission, error) {
	if allowedRoots == nil {
		allowedRoots = json.RawMessage(`["/"]`)
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO file_permissions (device_id, user_id, can_browse, can_modify, can_upload, can_download, can_delete, allowed_roots)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (device_id, user_id) DO UPDATE SET
		   can_browse = $3, can_modify = $4, can_upload = $5, can_download = $6, can_delete = $7, allowed_roots = $8, updated_at = NOW()
		 RETURNING id, device_id, user_id, can_browse, can_modify, can_upload, can_download, can_delete, allowed_roots, created_at, updated_at`,
		deviceID, userID, canBrowse, canModify, canUpload, canDownload, canDelete, allowedRoots,
	)
	return scanFilePermission(row)
}

func (r *FilePermissionRepository) ListByDevice(ctx context.Context, deviceID string) ([]*FilePermission, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, user_id, can_browse, can_modify, can_upload, can_download, can_delete, allowed_roots, created_at, updated_at
		 FROM file_permissions WHERE device_id = $1 ORDER BY created_at ASC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list file permissions: %w", err)
	}
	defer rows.Close()

	var perms []*FilePermission
	for rows.Next() {
		p, err := scanFilePermission(rows)
		if err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *FilePermissionRepository) Delete(ctx context.Context, deviceID, userID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM file_permissions WHERE device_id = $1 AND user_id = $2`,
		deviceID, userID,
	)
	return err
}

func scanFilePermission(row scanner) (*FilePermission, error) {
	p := &FilePermission{}
	err := row.Scan(&p.ID, &p.DeviceID, &p.UserID, &p.CanBrowse, &p.CanModify, &p.CanUpload, &p.CanDownload, &p.CanDelete, &p.AllowedRoots, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan file permission: %w", err)
	}
	return p, nil
}
