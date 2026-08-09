package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UpdateHistoryEntry struct {
	ID           string          `json:"id"`
	VersionFrom  string          `json:"version_from"`
	VersionTo    string          `json:"version_to"`
	Channel      string          `json:"channel"`
	Status       string          `json:"status"`
	Components   json.RawMessage `json:"components"`
	BackupID     *string         `json:"backup_id,omitempty"`
	ErrorMessage string          `json:"error_message"`
	StartedBy    *string         `json:"started_by,omitempty"`
	StartedAt    time.Time       `json:"started_at"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

type UpdateBackup struct {
	ID               string     `json:"id"`
	Version          string     `json:"version"`
	BackupType       string     `json:"backup_type"`
	BackupPath       string     `json:"backup_path"`
	DatabaseDumpPath string     `json:"database_dump_path"`
	ConfigBackupPath string     `json:"config_backup_path"`
	SizeBytes        int64      `json:"size_bytes"`
	Checksum         string     `json:"checksum"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
}

type PluginCompatEntry struct {
	ID              string    `json:"id"`
	PluginName      string    `json:"plugin_name"`
	MinRenOSVersion string    `json:"min_renos_version"`
	MaxRenOSVersion string    `json:"max_renos_version"`
	Tested          bool      `json:"tested"`
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
}

type UpdateHistoryRepository struct{ pool *pgxpool.Pool }

func NewUpdateHistoryRepository(pool *pgxpool.Pool) *UpdateHistoryRepository {
	return &UpdateHistoryRepository{pool: pool}
}

func (r *UpdateHistoryRepository) Create(ctx context.Context, e *UpdateHistoryEntry) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO update_history (version_from, version_to, channel, status, components, started_by)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, started_at, created_at`,
		e.VersionFrom, e.VersionTo, e.Channel, e.Status, e.Components, e.StartedBy,
	)
	return row.Scan(&e.ID, &e.StartedAt, &e.CreatedAt)
}

func (r *UpdateHistoryRepository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx, `UPDATE update_history SET status=$1 WHERE id=$2`, status, id)
	return err
}

func (r *UpdateHistoryRepository) Complete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE update_history SET status='completed', completed_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *UpdateHistoryRepository) Fail(ctx context.Context, id, errMsg string) error {
	_, err := r.pool.Exec(ctx, `UPDATE update_history SET status='failed', error_message=$1 WHERE id=$2`, errMsg, id)
	return err
}

func (r *UpdateHistoryRepository) ListAll(ctx context.Context, limit int) ([]*UpdateHistoryEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, version_from, version_to, channel, status, components, backup_id, error_message, started_by, started_at, completed_at, created_at
		 FROM update_history ORDER BY started_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUpdateHistory(rows)
}

func scanUpdateHistory(rows pgx.Rows) ([]*UpdateHistoryEntry, error) {
	var list []*UpdateHistoryEntry
	for rows.Next() {
		e := &UpdateHistoryEntry{}
		if err := rows.Scan(&e.ID, &e.VersionFrom, &e.VersionTo, &e.Channel, &e.Status, &e.Components, &e.BackupID, &e.ErrorMessage, &e.StartedBy, &e.StartedAt, &e.CompletedAt, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

type UpdateBackupRepository struct{ pool *pgxpool.Pool }

func NewUpdateBackupRepository(pool *pgxpool.Pool) *UpdateBackupRepository {
	return &UpdateBackupRepository{pool: pool}
}

func (r *UpdateBackupRepository) Create(ctx context.Context, b *UpdateBackup) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO update_backups (version, backup_type, backup_path, database_dump_path, config_backup_path, size_bytes, checksum, status, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, created_at`,
		b.Version, b.BackupType, b.BackupPath, b.DatabaseDumpPath, b.ConfigBackupPath, b.SizeBytes, b.Checksum, b.Status, b.ExpiresAt,
	)
	return row.Scan(&b.ID, &b.CreatedAt)
}

func (r *UpdateBackupRepository) ListAll(ctx context.Context) ([]*UpdateBackup, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, version, backup_type, backup_path, database_dump_path, config_backup_path, size_bytes, checksum, status, created_at, expires_at
		 FROM update_backups ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*UpdateBackup
	for rows.Next() {
		b := &UpdateBackup{}
		if err := rows.Scan(&b.ID, &b.Version, &b.BackupType, &b.BackupPath, &b.DatabaseDumpPath, &b.ConfigBackupPath, &b.SizeBytes, &b.Checksum, &b.Status, &b.CreatedAt, &b.ExpiresAt); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

func (r *UpdateBackupRepository) FindByID(ctx context.Context, id string) (*UpdateBackup, error) {
	b := &UpdateBackup{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, version, backup_type, backup_path, database_dump_path, config_backup_path, size_bytes, checksum, status, created_at, expires_at
		 FROM update_backups WHERE id=$1`, id,
	).Scan(&b.ID, &b.Version, &b.BackupType, &b.BackupPath, &b.DatabaseDumpPath, &b.ConfigBackupPath, &b.SizeBytes, &b.Checksum, &b.Status, &b.CreatedAt, &b.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *UpdateBackupRepository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx, `UPDATE update_backups SET status=$1 WHERE id=$2`, status, id)
	return err
}
