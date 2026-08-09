package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type VaultSecret struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Path           string          `json:"path"`
	SecretType     string          `json:"secret_type"`
	EncryptedValue []byte          `json:"-"`
	Version        int             `json:"version"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedBy      *string         `json:"created_by,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type VaultSecretVersion struct {
	ID             string          `json:"id"`
	SecretID       string          `json:"secret_id"`
	Version        int             `json:"version"`
	EncryptedValue []byte          `json:"-"`
	Metadata       json.RawMessage `json:"metadata"`
	ChangedBy      *string         `json:"changed_by,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

type VaultKey struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Algorithm string     `json:"algorithm"`
	PublicKey string     `json:"public_key"`
	RotatedAt time.Time  `json:"rotated_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Enabled   bool       `json:"enabled"`
	CreatedAt time.Time  `json:"created_at"`
}

type VaultPolicy struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Paths        json.RawMessage `json:"paths"`
	Capabilities json.RawMessage `json:"capabilities"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type VaultGrant struct {
	ID        string    `json:"id"`
	PolicyID  string    `json:"policy_id"`
	UserID    *string   `json:"user_id,omitempty"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type VaultAuditEntry struct {
	ID        string          `json:"id"`
	SecretID  *string         `json:"secret_id,omitempty"`
	Action    string          `json:"action"`
	UserID    *string         `json:"user_id,omitempty"`
	IPAddress string          `json:"ip_address"`
	Details   json.RawMessage `json:"details"`
	CreatedAt time.Time       `json:"created_at"`
}

type VaultSecretRepository struct {
	pool *pgxpool.Pool
}

func NewVaultSecretRepository(pool *pgxpool.Pool) *VaultSecretRepository {
	return &VaultSecretRepository{pool: pool}
}

func (r *VaultSecretRepository) Create(ctx context.Context, s *VaultSecret) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO vault_secrets (name, description, path, secret_type, encrypted_value, metadata, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, version, created_at, updated_at`,
		s.Name, s.Description, s.Path, s.SecretType, s.EncryptedValue, s.Metadata, s.CreatedBy,
	)
	return row.Scan(&s.ID, &s.Version, &s.CreatedAt, &s.UpdatedAt)
}

func (r *VaultSecretRepository) FindByID(ctx context.Context, id string) (*VaultSecret, error) {
	s := &VaultSecret{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, path, secret_type, encrypted_value, version, metadata, created_by, created_at, updated_at
		 FROM vault_secrets WHERE id = $1`, id,
	).Scan(&s.ID, &s.Name, &s.Description, &s.Path, &s.SecretType, &s.EncryptedValue, &s.Version, &s.Metadata, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *VaultSecretRepository) ListAll(ctx context.Context) ([]*VaultSecret, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, path, secret_type, version, metadata, created_by, created_at, updated_at
		 FROM vault_secrets ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*VaultSecret
	for rows.Next() {
		s := &VaultSecret{}
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Path, &s.SecretType, &s.Version, &s.Metadata, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *VaultSecretRepository) Update(ctx context.Context, s *VaultSecret) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE vault_secrets SET name=$1, description=$2, secret_type=$3, encrypted_value=$4, version=version+1, metadata=$5, updated_at=NOW()
		 WHERE id=$6`,
		s.Name, s.Description, s.SecretType, s.EncryptedValue, s.Metadata, s.ID,
	)
	return err
}

func (r *VaultSecretRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM vault_secrets WHERE id = $1`, id)
	return err
}

func (r *VaultSecretRepository) Search(ctx context.Context, query string) ([]*VaultSecret, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, path, secret_type, version, metadata, created_by, created_at, updated_at
		 FROM vault_secrets WHERE name ILIKE '%' || $1 || '%' OR path ILIKE '%' || $1 || '%'
		 ORDER BY path`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*VaultSecret
	for rows.Next() {
		s := &VaultSecret{}
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Path, &s.SecretType, &s.Version, &s.Metadata, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

type VaultSecretVersionRepository struct {
	pool *pgxpool.Pool
}

func NewVaultSecretVersionRepository(pool *pgxpool.Pool) *VaultSecretVersionRepository {
	return &VaultSecretVersionRepository{pool: pool}
}

func (r *VaultSecretVersionRepository) Create(ctx context.Context, v *VaultSecretVersion) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO vault_secret_versions (secret_id, version, encrypted_value, metadata, changed_by)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at`,
		v.SecretID, v.Version, v.EncryptedValue, v.Metadata, v.ChangedBy,
	)
	return row.Scan(&v.ID, &v.CreatedAt)
}

func (r *VaultSecretVersionRepository) ListBySecret(ctx context.Context, secretID string) ([]*VaultSecretVersion, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, secret_id, version, metadata, changed_by, created_at
		 FROM vault_secret_versions WHERE secret_id = $1 ORDER BY version DESC`, secretID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*VaultSecretVersion
	for rows.Next() {
		v := &VaultSecretVersion{}
		if err := rows.Scan(&v.ID, &v.SecretID, &v.Version, &v.Metadata, &v.ChangedBy, &v.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

func (r *VaultSecretVersionRepository) FindByVersion(ctx context.Context, secretID string, version int) (*VaultSecretVersion, error) {
	v := &VaultSecretVersion{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, secret_id, version, encrypted_value, metadata, changed_by, created_at
		 FROM vault_secret_versions WHERE secret_id = $1 AND version = $2`, secretID, version,
	).Scan(&v.ID, &v.SecretID, &v.Version, &v.EncryptedValue, &v.Metadata, &v.ChangedBy, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return v, nil
}
