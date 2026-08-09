package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IdentityProvider struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	ProviderType      string          `json:"provider_type"`
	ClientID          string          `json:"client_id"`
	ClientSecret      string          `json:"client_secret"`
	IssuerURL         string          `json:"issuer_url"`
	AuthURL           string          `json:"auth_url"`
	TokenURL          string          `json:"token_url"`
	UserinfoURL       string          `json:"userinfo_url"`
	JwksURL           string          `json:"jwks_url"`
	Scopes            json.RawMessage `json:"scopes"`
	AttributeMapping  json.RawMessage `json:"attribute_mapping"`
	GroupMapping      json.RawMessage `json:"group_mapping"`
	BaseDN            string          `json:"base_dn"`
	BindDN            string          `json:"bind_dn"`
	BindPassword      string          `json:"bind_password"`
	UserSearchFilter  string          `json:"user_search_filter"`
	GroupSearchFilter string          `json:"group_search_filter"`
	Enabled           bool            `json:"enabled"`
	DefaultRole       string          `json:"default_role"`
	AutoProvision     bool            `json:"auto_provision"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type OAuth2App struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	ClientID         string          `json:"client_id"`
	ClientSecretHash string          `json:"client_secret_hash"`
	RedirectURIs     json.RawMessage `json:"redirect_uris"`
	GrantTypes       json.RawMessage `json:"grant_types"`
	Scopes           json.RawMessage `json:"scopes"`
	Description      string          `json:"description"`
	Enabled          bool            `json:"enabled"`
	CreatedBy        *string         `json:"created_by,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type OAuth2AuthCode struct {
	ID                  string          `json:"id"`
	Code                string          `json:"code"`
	ClientID            string          `json:"client_id"`
	UserID              string          `json:"user_id"`
	RedirectURI         string          `json:"redirect_uri"`
	Scopes              json.RawMessage `json:"scopes"`
	CodeChallenge       string          `json:"code_challenge"`
	CodeChallengeMethod string          `json:"code_challenge_method"`
	ExpiresAt           time.Time       `json:"expires_at"`
	Used                bool            `json:"used"`
	CreatedAt           time.Time       `json:"created_at"`
}

type APIToken struct {
	ID          string          `json:"id"`
	UserID      string          `json:"user_id"`
	Name        string          `json:"name"`
	TokenHash   string          `json:"token_hash"`
	TokenPrefix string          `json:"token_prefix"`
	Scopes      json.RawMessage `json:"scopes"`
	ExpiresAt   *time.Time      `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time      `json:"last_used_at,omitempty"`
	Enabled     bool            `json:"enabled"`
	CreatedAt   time.Time       `json:"created_at"`
}

type IAMRole struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	DisplayName string          `json:"display_name"`
	Description string          `json:"description"`
	Permissions json.RawMessage `json:"permissions"`
	IsSystem    bool            `json:"is_system"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type IAMRoleAssignment struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	RoleID       string     `json:"role_id"`
	ResourceType string     `json:"resource_type"`
	ResourceID   string     `json:"resource_id"`
	GrantedBy    *string    `json:"granted_by,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type IAMUserGroup struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type IAMUserGroupMember struct {
	ID        string    `json:"id"`
	GroupID   string    `json:"group_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type ApprovalWorkflow struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Description       string          `json:"description"`
	ActionPattern     string          `json:"action_pattern"`
	RequiredApprovals int             `json:"required_approvals"`
	ApproverRoles     json.RawMessage `json:"approver_roles"`
	ApproverUsers     json.RawMessage `json:"approver_users"`
	TimeoutSeconds    int             `json:"timeout_seconds"`
	Enabled           bool            `json:"enabled"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type ApprovalRequest struct {
	ID            string          `json:"id"`
	WorkflowID    string          `json:"workflow_id"`
	ActionType    string          `json:"action_type"`
	ActionData    json.RawMessage `json:"action_data"`
	RequestedBy   string          `json:"requested_by"`
	Status        string          `json:"status"`
	ApprovalCount int             `json:"approval_count"`
	RequiredCount int             `json:"required_count"`
	ExpiresAt     time.Time       `json:"expires_at"`
	DecidedAt     *time.Time      `json:"decided_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type ApprovalDecision struct {
	ID         string    `json:"id"`
	RequestID  string    `json:"request_id"`
	ApproverID string    `json:"approver_id"`
	Decision   string    `json:"decision"`
	Comment    string    `json:"comment"`
	DecidedAt  time.Time `json:"decided_at"`
}

type IAMAuditEntry struct {
	ID           string          `json:"id"`
	ActorID      string          `json:"actor_id"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	Details      json.RawMessage `json:"details"`
	IPAddress    string          `json:"ip_address"`
	CreatedAt    time.Time       `json:"created_at"`
}

// --- Repositories ---

type IdentityProviderRepository struct{ pool *pgxpool.Pool }

func NewIdentityProviderRepository(pool *pgxpool.Pool) *IdentityProviderRepository {
	return &IdentityProviderRepository{pool: pool}
}

func (r *IdentityProviderRepository) Create(ctx context.Context, p *IdentityProvider) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO identity_providers (name, provider_type, client_id, client_secret, issuer_url, auth_url, token_url, userinfo_url, jwks_url, scopes, attribute_mapping, group_mapping, base_dn, bind_dn, bind_password, user_search_filter, group_search_filter, enabled, default_role, auto_provision)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20) RETURNING id, created_at, updated_at`,
		p.Name, p.ProviderType, p.ClientID, p.ClientSecret, p.IssuerURL, p.AuthURL, p.TokenURL, p.UserinfoURL, p.JwksURL, p.Scopes, p.AttributeMapping, p.GroupMapping, p.BaseDN, p.BindDN, p.BindPassword, p.UserSearchFilter, p.GroupSearchFilter, p.Enabled, p.DefaultRole, p.AutoProvision,
	)
	return row.Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *IdentityProviderRepository) ListAll(ctx context.Context) ([]*IdentityProvider, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, provider_type, client_id, client_secret, issuer_url, auth_url, token_url, userinfo_url, jwks_url, scopes, attribute_mapping, group_mapping, base_dn, bind_dn, bind_password, user_search_filter, group_search_filter, enabled, default_role, auto_provision, created_at, updated_at FROM identity_providers ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*IdentityProvider
	for rows.Next() {
		p := &IdentityProvider{}
		if err := rows.Scan(&p.ID, &p.Name, &p.ProviderType, &p.ClientID, &p.ClientSecret, &p.IssuerURL, &p.AuthURL, &p.TokenURL, &p.UserinfoURL, &p.JwksURL, &p.Scopes, &p.AttributeMapping, &p.GroupMapping, &p.BaseDN, &p.BindDN, &p.BindPassword, &p.UserSearchFilter, &p.GroupSearchFilter, &p.Enabled, &p.DefaultRole, &p.AutoProvision, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *IdentityProviderRepository) FindByName(ctx context.Context, name string) (*IdentityProvider, error) {
	p := &IdentityProvider{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, provider_type, client_id, client_secret, issuer_url, auth_url, token_url, userinfo_url, jwks_url, scopes, attribute_mapping, group_mapping, base_dn, bind_dn, bind_password, user_search_filter, group_search_filter, enabled, default_role, auto_provision, created_at, updated_at FROM identity_providers WHERE name=$1`, name,
	).Scan(&p.ID, &p.Name, &p.ProviderType, &p.ClientID, &p.ClientSecret, &p.IssuerURL, &p.AuthURL, &p.TokenURL, &p.UserinfoURL, &p.JwksURL, &p.Scopes, &p.AttributeMapping, &p.GroupMapping, &p.BaseDN, &p.BindDN, &p.BindPassword, &p.UserSearchFilter, &p.GroupSearchFilter, &p.Enabled, &p.DefaultRole, &p.AutoProvision, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *IdentityProviderRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM identity_providers WHERE id=$1`, id)
	return err
}

type OAuth2AppRepository struct{ pool *pgxpool.Pool }

func NewOAuth2AppRepository(pool *pgxpool.Pool) *OAuth2AppRepository {
	return &OAuth2AppRepository{pool: pool}
}

func (r *OAuth2AppRepository) Create(ctx context.Context, a *OAuth2App) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO oauth2_apps (name, client_id, client_secret_hash, redirect_uris, grant_types, scopes, description, enabled, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, created_at, updated_at`,
		a.Name, a.ClientID, a.ClientSecretHash, a.RedirectURIs, a.GrantTypes, a.Scopes, a.Description, a.Enabled, a.CreatedBy,
	)
	return row.Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *OAuth2AppRepository) ListAll(ctx context.Context) ([]*OAuth2App, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, client_id, client_secret_hash, redirect_uris, grant_types, scopes, description, enabled, created_by, created_at, updated_at FROM oauth2_apps ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*OAuth2App
	for rows.Next() {
		a := &OAuth2App{}
		if err := rows.Scan(&a.ID, &a.Name, &a.ClientID, &a.ClientSecretHash, &a.RedirectURIs, &a.GrantTypes, &a.Scopes, &a.Description, &a.Enabled, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (r *OAuth2AppRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM oauth2_apps WHERE id=$1`, id)
	return err
}

type APITokenRepository struct{ pool *pgxpool.Pool }

func NewAPITokenRepository(pool *pgxpool.Pool) *APITokenRepository {
	return &APITokenRepository{pool: pool}
}

func (r *APITokenRepository) Create(ctx context.Context, t *APIToken) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO api_tokens (user_id, name, token_hash, token_prefix, scopes, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at`,
		t.UserID, t.Name, t.TokenHash, t.TokenPrefix, t.Scopes, t.ExpiresAt,
	)
	return row.Scan(&t.ID, &t.CreatedAt)
}

func (r *APITokenRepository) ListByUser(ctx context.Context, userID string) ([]*APIToken, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, user_id, name, token_hash, token_prefix, scopes, expires_at, last_used_at, enabled, created_at FROM api_tokens WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*APIToken
	for rows.Next() {
		t := &APIToken{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.TokenPrefix, &t.Scopes, &t.ExpiresAt, &t.LastUsedAt, &t.Enabled, &t.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (r *APITokenRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM api_tokens WHERE id=$1`, id)
	return err
}

type IAMRoleRepository struct{ pool *pgxpool.Pool }

func NewIAMRoleRepository(pool *pgxpool.Pool) *IAMRoleRepository {
	return &IAMRoleRepository{pool: pool}
}

func (r *IAMRoleRepository) Create(ctx context.Context, role *IAMRole) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO iam_roles (name, display_name, description, permissions, is_system)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at, updated_at`,
		role.Name, role.DisplayName, role.Description, role.Permissions, role.IsSystem,
	)
	return row.Scan(&role.ID, &role.CreatedAt, &role.UpdatedAt)
}

func (r *IAMRoleRepository) ListAll(ctx context.Context) ([]*IAMRole, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, display_name, description, permissions, is_system, created_at, updated_at FROM iam_roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*IAMRole
	for rows.Next() {
		role := &IAMRole{}
		if err := rows.Scan(&role.ID, &role.Name, &role.DisplayName, &role.Description, &role.Permissions, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, role)
	}
	return list, rows.Err()
}

func (r *IAMRoleRepository) FindByName(ctx context.Context, name string) (*IAMRole, error) {
	role := &IAMRole{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, display_name, description, permissions, is_system, created_at, updated_at FROM iam_roles WHERE name=$1`, name,
	).Scan(&role.ID, &role.Name, &role.DisplayName, &role.Description, &role.Permissions, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func (r *IAMRoleRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM iam_roles WHERE id=$1 AND is_system=false`, id)
	return err
}

type IAMRoleAssignmentRepository struct{ pool *pgxpool.Pool }

func NewIAMRoleAssignmentRepository(pool *pgxpool.Pool) *IAMRoleAssignmentRepository {
	return &IAMRoleAssignmentRepository{pool: pool}
}

func (r *IAMRoleAssignmentRepository) Create(ctx context.Context, a *IAMRoleAssignment) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO iam_role_assignments (user_id, role_id, resource_type, resource_id, granted_by, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at`,
		a.UserID, a.RoleID, a.ResourceType, a.ResourceID, a.GrantedBy, a.ExpiresAt,
	)
	return row.Scan(&a.ID, &a.CreatedAt)
}

func (r *IAMRoleAssignmentRepository) ListByUser(ctx context.Context, userID string) ([]*IAMRoleAssignment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, role_id, resource_type, resource_id, granted_by, expires_at, created_at
		 FROM iam_role_assignments WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*IAMRoleAssignment
	for rows.Next() {
		a := &IAMRoleAssignment{}
		if err := rows.Scan(&a.ID, &a.UserID, &a.RoleID, &a.ResourceType, &a.ResourceID, &a.GrantedBy, &a.ExpiresAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (r *IAMRoleAssignmentRepository) ListAll(ctx context.Context) ([]*IAMRoleAssignment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, role_id, resource_type, resource_id, granted_by, expires_at, created_at
		 FROM iam_role_assignments ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*IAMRoleAssignment
	for rows.Next() {
		a := &IAMRoleAssignment{}
		if err := rows.Scan(&a.ID, &a.UserID, &a.RoleID, &a.ResourceType, &a.ResourceID, &a.GrantedBy, &a.ExpiresAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (r *IAMRoleAssignmentRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM iam_role_assignments WHERE id=$1`, id)
	return err
}

type ApprovalWorkflowRepository struct{ pool *pgxpool.Pool }

func NewApprovalWorkflowRepository(pool *pgxpool.Pool) *ApprovalWorkflowRepository {
	return &ApprovalWorkflowRepository{pool: pool}
}

func (r *ApprovalWorkflowRepository) Create(ctx context.Context, w *ApprovalWorkflow) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO approval_workflows (name, description, action_pattern, required_approvals, approver_roles, approver_users, timeout_seconds, enabled)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, created_at, updated_at`,
		w.Name, w.Description, w.ActionPattern, w.RequiredApprovals, w.ApproverRoles, w.ApproverUsers, w.TimeoutSeconds, w.Enabled,
	)
	return row.Scan(&w.ID, &w.CreatedAt, &w.UpdatedAt)
}

func (r *ApprovalWorkflowRepository) ListAll(ctx context.Context) ([]*ApprovalWorkflow, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, description, action_pattern, required_approvals, approver_roles, approver_users, timeout_seconds, enabled, created_at, updated_at FROM approval_workflows ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*ApprovalWorkflow
	for rows.Next() {
		w := &ApprovalWorkflow{}
		if err := rows.Scan(&w.ID, &w.Name, &w.Description, &w.ActionPattern, &w.RequiredApprovals, &w.ApproverRoles, &w.ApproverUsers, &w.TimeoutSeconds, &w.Enabled, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

func (r *ApprovalWorkflowRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM approval_workflows WHERE id=$1`, id)
	return err
}

type ApprovalRequestRepository struct{ pool *pgxpool.Pool }

func NewApprovalRequestRepository(pool *pgxpool.Pool) *ApprovalRequestRepository {
	return &ApprovalRequestRepository{pool: pool}
}

func (r *ApprovalRequestRepository) Create(ctx context.Context, req *ApprovalRequest) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO approval_requests (workflow_id, action_type, action_data, requested_by, required_count, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, status, approval_count, created_at, updated_at`,
		req.WorkflowID, req.ActionType, req.ActionData, req.RequestedBy, req.RequiredCount, req.ExpiresAt,
	)
	return row.Scan(&req.ID, &req.Status, &req.ApprovalCount, &req.CreatedAt, &req.UpdatedAt)
}

func (r *ApprovalRequestRepository) ListPending(ctx context.Context) ([]*ApprovalRequest, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, workflow_id, action_type, action_data, requested_by, status, approval_count, required_count, expires_at, decided_at, created_at, updated_at FROM approval_requests WHERE status='pending' ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*ApprovalRequest
	for rows.Next() {
		req := &ApprovalRequest{}
		if err := rows.Scan(&req.ID, &req.WorkflowID, &req.ActionType, &req.ActionData, &req.RequestedBy, &req.Status, &req.ApprovalCount, &req.RequiredCount, &req.ExpiresAt, &req.DecidedAt, &req.CreatedAt, &req.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, req)
	}
	return list, rows.Err()
}

func (r *ApprovalRequestRepository) ListAll(ctx context.Context, status string) ([]*ApprovalRequest, error) {
	var rows pgx.Rows
	var err error
	if status != "" {
		rows, err = r.pool.Query(ctx, `SELECT id, workflow_id, action_type, action_data, requested_by, status, approval_count, required_count, expires_at, decided_at, created_at, updated_at FROM approval_requests WHERE status=$1 ORDER BY created_at DESC`, status)
	} else {
		rows, err = r.pool.Query(ctx, `SELECT id, workflow_id, action_type, action_data, requested_by, status, approval_count, required_count, expires_at, decided_at, created_at, updated_at FROM approval_requests ORDER BY created_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*ApprovalRequest
	for rows.Next() {
		req := &ApprovalRequest{}
		if err := rows.Scan(&req.ID, &req.WorkflowID, &req.ActionType, &req.ActionData, &req.RequestedBy, &req.Status, &req.ApprovalCount, &req.RequiredCount, &req.ExpiresAt, &req.DecidedAt, &req.CreatedAt, &req.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, req)
	}
	return list, rows.Err()
}

func (r *ApprovalRequestRepository) FindByID(ctx context.Context, id string) (*ApprovalRequest, error) {
	req := &ApprovalRequest{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, workflow_id, action_type, action_data, requested_by, status, approval_count, required_count, expires_at, decided_at, created_at, updated_at FROM approval_requests WHERE id=$1`, id,
	).Scan(&req.ID, &req.WorkflowID, &req.ActionType, &req.ActionData, &req.RequestedBy, &req.Status, &req.ApprovalCount, &req.RequiredCount, &req.ExpiresAt, &req.DecidedAt, &req.CreatedAt, &req.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (r *ApprovalRequestRepository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx, `UPDATE approval_requests SET status=$1, updated_at=NOW(), decided_at=NOW() WHERE id=$2`, status, id)
	return err
}

func (r *ApprovalRequestRepository) IncrementApproval(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE approval_requests SET approval_count = approval_count + 1, updated_at=NOW() WHERE id=$1`, id)
	return err
}

type ApprovalDecisionRepository struct{ pool *pgxpool.Pool }

func NewApprovalDecisionRepository(pool *pgxpool.Pool) *ApprovalDecisionRepository {
	return &ApprovalDecisionRepository{pool: pool}
}

func (r *ApprovalDecisionRepository) Create(ctx context.Context, d *ApprovalDecision) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO approval_decisions (request_id, approver_id, decision, comment)
		 VALUES ($1,$2,$3,$4) RETURNING id, decided_at`,
		d.RequestID, d.ApproverID, d.Decision, d.Comment,
	)
	return row.Scan(&d.ID, &d.DecidedAt)
}

type IAMAuditRepository struct{ pool *pgxpool.Pool }

func NewIAMAuditRepository(pool *pgxpool.Pool) *IAMAuditRepository {
	return &IAMAuditRepository{pool: pool}
}

func (r *IAMAuditRepository) Create(ctx context.Context, e *IAMAuditEntry) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO iam_audit_log (actor_id, action, resource_type, resource_id, details, ip_address)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at`,
		e.ActorID, e.Action, e.ResourceType, e.ResourceID, e.Details, e.IPAddress,
	)
	return row.Scan(&e.ID, &e.CreatedAt)
}

func (r *IAMAuditRepository) List(ctx context.Context, limit, offset int) ([]*IAMAuditEntry, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, actor_id, action, resource_type, resource_id, details, ip_address, created_at FROM iam_audit_log ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*IAMAuditEntry
	for rows.Next() {
		e := &IAMAuditEntry{}
		if err := rows.Scan(&e.ID, &e.ActorID, &e.Action, &e.ResourceType, &e.ResourceID, &e.Details, &e.IPAddress, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}
