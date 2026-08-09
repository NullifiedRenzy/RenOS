package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type VaultKeyRepository struct {
	pool *pgxpool.Pool
}

func NewVaultKeyRepository(pool *pgxpool.Pool) *VaultKeyRepository {
	return &VaultKeyRepository{pool: pool}
}

func (r *VaultKeyRepository) Create(ctx context.Context, k *VaultKey) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO vault_keys (name, algorithm, public_key, rotated_at, expires_at, enabled)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at`,
		k.Name, k.Algorithm, k.PublicKey, k.RotatedAt, k.ExpiresAt, k.Enabled,
	)
	return row.Scan(&k.ID, &k.CreatedAt)
}

func (r *VaultKeyRepository) FindByID(ctx context.Context, id string) (*VaultKey, error) {
	k := &VaultKey{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, algorithm, public_key, rotated_at, expires_at, enabled, created_at
		 FROM vault_keys WHERE id = $1`, id,
	).Scan(&k.ID, &k.Name, &k.Algorithm, &k.PublicKey, &k.RotatedAt, &k.ExpiresAt, &k.Enabled, &k.CreatedAt)
	if err != nil {
		return nil, err
	}
	return k, nil
}

func (r *VaultKeyRepository) ListAll(ctx context.Context) ([]*VaultKey, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, algorithm, public_key, rotated_at, expires_at, enabled, created_at
		 FROM vault_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*VaultKey
	for rows.Next() {
		k := &VaultKey{}
		if err := rows.Scan(&k.ID, &k.Name, &k.Algorithm, &k.PublicKey, &k.RotatedAt, &k.ExpiresAt, &k.Enabled, &k.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, k)
	}
	return list, rows.Err()
}

func (r *VaultKeyRepository) Enable(ctx context.Context, id string, enabled bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE vault_keys SET enabled = $1 WHERE id = $2`, enabled, id)
	return err
}

type VaultPolicyRepository struct {
	pool *pgxpool.Pool
}

func NewVaultPolicyRepository(pool *pgxpool.Pool) *VaultPolicyRepository {
	return &VaultPolicyRepository{pool: pool}
}

func (r *VaultPolicyRepository) Create(ctx context.Context, p *VaultPolicy) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO vault_policies (name, description, paths, capabilities)
		 VALUES ($1,$2,$3,$4) RETURNING id, created_at, updated_at`,
		p.Name, p.Description, p.Paths, p.Capabilities,
	)
	return row.Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *VaultPolicyRepository) ListAll(ctx context.Context) ([]*VaultPolicy, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, paths, capabilities, created_at, updated_at
		 FROM vault_policies ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*VaultPolicy
	for rows.Next() {
		p := &VaultPolicy{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Paths, &p.Capabilities, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *VaultPolicyRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM vault_policies WHERE id = $1`, id)
	return err
}

type VaultGrantRepository struct {
	pool *pgxpool.Pool
}

func NewVaultGrantRepository(pool *pgxpool.Pool) *VaultGrantRepository {
	return &VaultGrantRepository{pool: pool}
}

func (r *VaultGrantRepository) Create(ctx context.Context, g *VaultGrant) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO vault_grants (policy_id, user_id, role)
		 VALUES ($1,$2,$3) RETURNING id, created_at`,
		g.PolicyID, g.UserID, g.Role,
	)
	return row.Scan(&g.ID, &g.CreatedAt)
}

func (r *VaultGrantRepository) ListByUser(ctx context.Context, userID string) ([]*VaultGrant, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, policy_id, user_id, role, created_at
		 FROM vault_grants WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*VaultGrant
	for rows.Next() {
		g := &VaultGrant{}
		if err := rows.Scan(&g.ID, &g.PolicyID, &g.UserID, &g.Role, &g.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	return list, rows.Err()
}

func (r *VaultGrantRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM vault_grants WHERE id = $1`, id)
	return err
}

type VaultAuditRepository struct {
	pool *pgxpool.Pool
}

func NewVaultAuditRepository(pool *pgxpool.Pool) *VaultAuditRepository {
	return &VaultAuditRepository{pool: pool}
}

func (r *VaultAuditRepository) Log(ctx context.Context, entry *VaultAuditEntry) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO vault_audit_log (secret_id, action, user_id, ip_address, details)
		 VALUES ($1,$2,$3,$4,$5)`,
		entry.SecretID, entry.Action, entry.UserID, entry.IPAddress, entry.Details,
	)
	return err
}

func (r *VaultAuditRepository) ListRecent(ctx context.Context, limit int) ([]*VaultAuditEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, secret_id, action, user_id, ip_address, details, created_at
		 FROM vault_audit_log ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*VaultAuditEntry
	for rows.Next() {
		e := &VaultAuditEntry{}
		if err := rows.Scan(&e.ID, &e.SecretID, &e.Action, &e.UserID, &e.IPAddress, &e.Details, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

func (r *VaultAuditRepository) ListBySecret(ctx context.Context, secretID string, limit int) ([]*VaultAuditEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, secret_id, action, user_id, ip_address, details, created_at
		 FROM vault_audit_log WHERE secret_id = $1 ORDER BY created_at DESC LIMIT $2`, secretID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*VaultAuditEntry
	for rows.Next() {
		e := &VaultAuditEntry{}
		if err := rows.Scan(&e.ID, &e.SecretID, &e.Action, &e.UserID, &e.IPAddress, &e.Details, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}
