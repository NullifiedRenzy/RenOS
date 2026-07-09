package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Device struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	Hostname        string    `json:"hostname"`
	OS              string    `json:"os"`
	Arch            string    `json:"arch"`
	AgentVersion    string    `json:"agent_version"`
	CertFingerprint string    `json:"cert_fingerprint"`
	EnrolledAt      *time.Time `json:"enrolled_at,omitempty"`
	LastSeen        *time.Time `json:"last_seen,omitempty"`
	Status          string    `json:"status"`
	Labels          string    `json:"labels"`
	Capabilities    string    `json:"capabilities"`
	Metadata        string    `json:"metadata"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type DeviceRepository struct {
	pool *pgxpool.Pool
}

func NewDeviceRepository(pool *pgxpool.Pool) *DeviceRepository {
	return &DeviceRepository{pool: pool}
}

func (r *DeviceRepository) Create(ctx context.Context, userID, hostname, os, arch, agentVersion, certFingerprint string) (*Device, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO devices (user_id, hostname, os, arch, agent_version, cert_fingerprint, status, enrolled_at)
		 VALUES ($1, $2, $3, $4, $5, $6, 'offline', NOW())
		 RETURNING id, user_id, hostname, os, arch, agent_version, cert_fingerprint,
		           enrolled_at, last_seen, status, labels, capabilities, metadata, created_at, updated_at`,
		userID, hostname, os, arch, agentVersion, certFingerprint,
	)

	return scanDevice(row)
}

func (r *DeviceRepository) FindByID(ctx context.Context, id string) (*Device, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, hostname, os, arch, agent_version, cert_fingerprint,
		        enrolled_at, last_seen, status, labels, capabilities, metadata, created_at, updated_at
		 FROM devices WHERE id = $1`,
		id,
	)
	return scanDevice(row)
}

func (r *DeviceRepository) FindByCertFingerprint(ctx context.Context, fingerprint string) (*Device, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, hostname, os, arch, agent_version, cert_fingerprint,
		        enrolled_at, last_seen, status, labels, capabilities, metadata, created_at, updated_at
		 FROM devices WHERE cert_fingerprint = $1`,
		fingerprint,
	)
	return scanDevice(row)
}

func (r *DeviceRepository) FindAll(ctx context.Context) ([]*Device, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, hostname, os, arch, agent_version, cert_fingerprint,
		        enrolled_at, last_seen, status, labels, capabilities, metadata, created_at, updated_at
		 FROM devices ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	var devices []*Device
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func (r *DeviceRepository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE devices SET status = $1, last_seen = NOW() WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update device status: %w", err)
	}
	return nil
}

func (r *DeviceRepository) UpdateMetadata(ctx context.Context, id, metadataJSON string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE devices SET metadata = metadata || $1::jsonb WHERE id = $2`,
		metadataJSON, id,
	)
	if err != nil {
		return fmt.Errorf("update device metadata: %w", err)
	}
	return nil
}

func (r *DeviceRepository) UpdateLastSeen(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE devices SET last_seen = NOW() WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("update device last_seen: %w", err)
	}
	return nil
}

func (r *DeviceRepository) UpdateStatusAndMetadata(ctx context.Context, id, status, metadataJSON string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE devices SET status = $1, metadata = metadata || $2::jsonb, last_seen = NOW() WHERE id = $3`,
		status, metadataJSON, id,
	)
	if err != nil {
		return fmt.Errorf("update device status and metadata: %w", err)
	}
	return nil
}

func (r *DeviceRepository) MarkDevicesOffline(ctx context.Context, threshold string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`UPDATE devices SET status = 'offline' WHERE status = 'online' AND last_seen < NOW() - $1::interval RETURNING id`,
		threshold,
	)
	if err != nil {
		return nil, fmt.Errorf("mark devices offline: %w", err)
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

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanDevice(row scanner) (*Device, error) {
	d := &Device{}
	err := row.Scan(&d.ID, &d.UserID, &d.Hostname, &d.OS, &d.Arch, &d.AgentVersion,
		&d.CertFingerprint, &d.EnrolledAt, &d.LastSeen, &d.Status, &d.Labels,
		&d.Capabilities, &d.Metadata, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan device: %w", err)
	}
	return d, nil
}
