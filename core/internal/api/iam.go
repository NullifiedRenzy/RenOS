package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renos-project/renos/core/internal/db"
)

// --- Identity Providers ---

func listIdentityProvidersHandler(repos *db.IdentityProviderRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := repos.ListAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list identity providers")
			return
		}
		if list == nil { list = []*db.IdentityProvider{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func createIdentityProviderHandler(repos *db.IdentityProviderRepository, audits *db.IAMAuditRepository) http.HandlerFunc {
	type request struct {
		Name               string          `json:"name"`
		ProviderType       string          `json:"provider_type"`
		ClientID           string          `json:"client_id"`
		ClientSecret       string          `json:"client_secret"`
		IssuerURL          string          `json:"issuer_url"`
		AuthURL            string          `json:"auth_url"`
		TokenURL           string          `json:"token_url"`
		UserinfoURL        string          `json:"userinfo_url"`
		JwksURL            string          `json:"jwks_url"`
		Scopes             json.RawMessage `json:"scopes"`
		AttributeMapping   json.RawMessage `json:"attribute_mapping"`
		GroupMapping       json.RawMessage `json:"group_mapping"`
		BaseDN             string          `json:"base_dn"`
		BindDN             string          `json:"bind_dn"`
		BindPassword       string          `json:"bind_password"`
		UserSearchFilter   string          `json:"user_search_filter"`
		GroupSearchFilter  string          `json:"group_search_filter"`
		Enabled            *bool           `json:"enabled"`
		DefaultRole        string          `json:"default_role"`
		AutoProvision      *bool           `json:"auto_provision"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if req.Name == "" || req.ProviderType == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "name and provider_type are required")
			return
		}
		enabled := true
		if req.Enabled != nil { enabled = *req.Enabled }
		autoProv := true
		if req.AutoProvision != nil { autoProv = *req.AutoProvision }
		if req.Scopes == nil { req.Scopes = json.RawMessage(`["openid","profile","email"]`) }
		if req.AttributeMapping == nil { req.AttributeMapping = json.RawMessage(`{}`) }
		if req.GroupMapping == nil { req.GroupMapping = json.RawMessage(`{}`) }
		if req.DefaultRole == "" { req.DefaultRole = "viewer" }

		p := &db.IdentityProvider{
			Name: req.Name, ProviderType: req.ProviderType, ClientID: req.ClientID, ClientSecret: req.ClientSecret,
			IssuerURL: req.IssuerURL, AuthURL: req.AuthURL, TokenURL: req.TokenURL, UserinfoURL: req.UserinfoURL, JwksURL: req.JwksURL,
			Scopes: req.Scopes, AttributeMapping: req.AttributeMapping, GroupMapping: req.GroupMapping,
			BaseDN: req.BaseDN, BindDN: req.BindDN, BindPassword: req.BindPassword,
			UserSearchFilter: req.UserSearchFilter, GroupSearchFilter: req.GroupSearchFilter,
			Enabled: enabled, DefaultRole: req.DefaultRole, AutoProvision: autoProv,
		}
		if err := repos.Create(r.Context(), p); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create identity provider: "+err.Error())
			return
		}
		iamAudit(r.Context(), audits, getUserID(r), "identity_provider.create", "identity_provider", p.Name, nil, r)
		writeJSON(w, http.StatusCreated, p)
	}
}

func deleteIdentityProviderHandler(repos *db.IdentityProviderRepository, audits *db.IAMAuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := repos.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete: "+err.Error())
			return
		}
		iamAudit(r.Context(), audits, getUserID(r), "identity_provider.delete", "identity_provider", id, nil, r)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

// --- OAuth2 Apps ---

func listOAuth2AppsHandler(repos *db.OAuth2AppRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := repos.ListAll(r.Context())
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", "failed to list OAuth2 apps"); return }
		if list == nil { list = []*db.OAuth2App{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func createOAuth2AppHandler(repos *db.OAuth2AppRepository, audits *db.IAMAuditRepository) http.HandlerFunc {
	type request struct {
		Name           string          `json:"name"`
		RedirectURIs   json.RawMessage `json:"redirect_uris"`
		GrantTypes     json.RawMessage `json:"grant_types"`
		Scopes         json.RawMessage `json:"scopes"`
		Description    string          `json:"description"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if req.Name == "" { writeError(w, http.StatusBadRequest, "invalid_request", "name is required"); return }

		clientID := generateTokenPrefix()
		secret := generateTokenPrefix() + generateTokenPrefix()
		hash := sha256.Sum256([]byte(secret))
		secretHash := hex.EncodeToString(hash[:])

		if req.RedirectURIs == nil { req.RedirectURIs = json.RawMessage(`[]`) }
		if req.GrantTypes == nil { req.GrantTypes = json.RawMessage(`["authorization_code"]`) }
		if req.Scopes == nil { req.Scopes = json.RawMessage(`["read"]`) }

		uid := getUserID(r)
		app := &db.OAuth2App{
			Name: req.Name, ClientID: clientID, ClientSecretHash: secretHash,
			RedirectURIs: req.RedirectURIs, GrantTypes: req.GrantTypes, Scopes: req.Scopes,
			Description: req.Description, Enabled: true, CreatedBy: &uid,
		}
		if err := repos.Create(r.Context(), app); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create OAuth2 app: "+err.Error())
			return
		}
		iamAudit(r.Context(), audits, uid, "oauth2_app.create", "oauth2_app", app.ClientID, nil, r)
		writeJSON(w, http.StatusCreated, map[string]interface{}{"app": app, "client_secret": secret})
	}
}

func deleteOAuth2AppHandler(repos *db.OAuth2AppRepository, audits *db.IAMAuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := repos.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete: "+err.Error())
			return
		}
		iamAudit(r.Context(), audits, getUserID(r), "oauth2_app.delete", "oauth2_app", id, nil, r)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

// --- API Tokens ---

func listAPITokensHandler(repos *db.APITokenRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid := getUserID(r)
		list, err := repos.ListByUser(r.Context(), uid)
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", "failed to list tokens"); return }
		if list == nil { list = []*db.APIToken{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func createAPITokenHandler(repos *db.APITokenRepository, audits *db.IAMAuditRepository) http.HandlerFunc {
	type request struct {
		Name    string          `json:"name"`
		Scopes  json.RawMessage `json:"scopes"`
		Expires *time.Time      `json:"expires_at"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if req.Name == "" { writeError(w, http.StatusBadRequest, "invalid_request", "name is required"); return }
		if req.Scopes == nil { req.Scopes = json.RawMessage(`["read"]`) }

		rawToken := generateTokenPrefix() + generateTokenPrefix() + generateTokenPrefix()
		hash := sha256.Sum256([]byte(rawToken))
		tokenHash := hex.EncodeToString(hash[:])
		prefix := rawToken[:8]

		uid := getUserID(r)
		t := &db.APIToken{
			UserID: uid, Name: req.Name, TokenHash: tokenHash, TokenPrefix: prefix,
			Scopes: req.Scopes, ExpiresAt: req.Expires, Enabled: true,
		}
		if err := repos.Create(r.Context(), t); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create token: "+err.Error())
			return
		}
		iamAudit(r.Context(), audits, uid, "api_token.create", "api_token", t.ID, nil, r)
		writeJSON(w, http.StatusCreated, map[string]interface{}{"token": t, "raw_token": rawToken})
	}
}

func deleteAPITokenHandler(repos *db.APITokenRepository, audits *db.IAMAuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := repos.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete: "+err.Error())
			return
		}
		iamAudit(r.Context(), audits, getUserID(r), "api_token.delete", "api_token", id, nil, r)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

// --- RBAC Roles ---

func listIAMRolesHandler(repos *db.IAMRoleRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := repos.ListAll(r.Context())
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", "failed to list roles"); return }
		if list == nil { list = []*db.IAMRole{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func createIAMRoleHandler(repos *db.IAMRoleRepository, audits *db.IAMAuditRepository) http.HandlerFunc {
	type request struct {
		Name        string          `json:"name"`
		DisplayName string          `json:"display_name"`
		Description string          `json:"description"`
		Permissions json.RawMessage `json:"permissions"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if req.Name == "" { writeError(w, http.StatusBadRequest, "invalid_request", "name is required"); return }
		if req.Permissions == nil { req.Permissions = json.RawMessage(`[]`) }

		role := &db.IAMRole{Name: req.Name, DisplayName: req.DisplayName, Description: req.Description, Permissions: req.Permissions}
		if err := repos.Create(r.Context(), role); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create role: "+err.Error())
			return
		}
		iamAudit(r.Context(), audits, getUserID(r), "role.create", "role", role.Name, nil, r)
		writeJSON(w, http.StatusCreated, role)
	}
}

func deleteIAMRoleHandler(repos *db.IAMRoleRepository, audits *db.IAMAuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := repos.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete: "+err.Error())
			return
		}
		iamAudit(r.Context(), audits, getUserID(r), "role.delete", "role", id, nil, r)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func listIAMRoleAssignmentsHandler(repos *db.IAMRoleAssignmentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := repos.ListAll(r.Context())
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", "failed to list assignments"); return }
		if list == nil { list = []*db.IAMRoleAssignment{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func createIAMRoleAssignmentHandler(repos *db.IAMRoleAssignmentRepository, audits *db.IAMAuditRepository) http.HandlerFunc {
	type request struct {
		UserID       string  `json:"user_id"`
		RoleID       string  `json:"role_id"`
		ResourceType string  `json:"resource_type"`
		ResourceID   string  `json:"resource_id"`
		ExpiresAt    *time.Time `json:"expires_at"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if req.UserID == "" || req.RoleID == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "user_id and role_id are required")
			return
		}
		if req.ResourceType == "" { req.ResourceType = "*" }
		if req.ResourceID == "" { req.ResourceID = "*" }

		uid := getUserID(r)
		a := &db.IAMRoleAssignment{UserID: req.UserID, RoleID: req.RoleID, ResourceType: req.ResourceType, ResourceID: req.ResourceID, GrantedBy: &uid, ExpiresAt: req.ExpiresAt}
		if err := repos.Create(r.Context(), a); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to assign role: "+err.Error())
			return
		}
		iamAudit(r.Context(), audits, uid, "role_assignment.create", "role_assignment", a.ID, nil, r)
		writeJSON(w, http.StatusCreated, a)
	}
}

func deleteIAMRoleAssignmentHandler(repos *db.IAMRoleAssignmentRepository, audits *db.IAMAuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := repos.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete: "+err.Error())
			return
		}
		iamAudit(r.Context(), audits, getUserID(r), "role_assignment.delete", "role_assignment", id, nil, r)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

// --- Approval Workflows ---

func listApprovalWorkflowsHandler(repos *db.ApprovalWorkflowRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := repos.ListAll(r.Context())
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", "failed to list approval workflows"); return }
		if list == nil { list = []*db.ApprovalWorkflow{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func createApprovalWorkflowHandler(repos *db.ApprovalWorkflowRepository, audits *db.IAMAuditRepository) http.HandlerFunc {
	type request struct {
		Name              string          `json:"name"`
		Description       string          `json:"description"`
		ActionPattern     string          `json:"action_pattern"`
		RequiredApprovals int             `json:"required_approvals"`
		ApproverRoles     json.RawMessage `json:"approver_roles"`
		ApproverUsers     json.RawMessage `json:"approver_users"`
		TimeoutSeconds    int             `json:"timeout_seconds"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if req.Name == "" { writeError(w, http.StatusBadRequest, "invalid_request", "name is required"); return }
		if req.RequiredApprovals <= 0 { req.RequiredApprovals = 1 }
		if req.TimeoutSeconds <= 0 { req.TimeoutSeconds = 86400 }
		if req.ActionPattern == "" { req.ActionPattern = "*" }
		if req.ApproverRoles == nil { req.ApproverRoles = json.RawMessage(`["admin"]`) }
		if req.ApproverUsers == nil { req.ApproverUsers = json.RawMessage(`[]`) }

		wf := &db.ApprovalWorkflow{
			Name: req.Name, Description: req.Description, ActionPattern: req.ActionPattern,
			RequiredApprovals: req.RequiredApprovals, ApproverRoles: req.ApproverRoles, ApproverUsers: req.ApproverUsers,
			TimeoutSeconds: req.TimeoutSeconds, Enabled: true,
		}
		if err := repos.Create(r.Context(), wf); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create workflow: "+err.Error())
			return
		}
		iamAudit(r.Context(), audits, getUserID(r), "approval_workflow.create", "approval_workflow", wf.Name, nil, r)
		writeJSON(w, http.StatusCreated, wf)
	}
}

func deleteApprovalWorkflowHandler(repos *db.ApprovalWorkflowRepository, audits *db.IAMAuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := repos.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete: "+err.Error())
			return
		}
		iamAudit(r.Context(), audits, getUserID(r), "approval_workflow.delete", "approval_workflow", id, nil, r)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func listApprovalRequestsHandler(repos *db.ApprovalRequestRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")
		list, err := repos.ListAll(r.Context(), status)
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", "failed to list approval requests"); return }
		if list == nil { list = []*db.ApprovalRequest{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func approveRequestHandler(reqRepos *db.ApprovalRequestRepository, decisionRepos *db.ApprovalDecisionRepository, audits *db.IAMAuditRepository) http.HandlerFunc {
	type request struct {
		Comment string `json:"comment"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var req request
		json.NewDecoder(r.Body).Decode(&req)

		approvalReq, err := reqRepos.FindByID(r.Context(), id)
		if err != nil { writeError(w, http.StatusNotFound, "not_found", "approval request not found"); return }
		if approvalReq.Status != "pending" { writeError(w, http.StatusConflict, "not_pending", "request is not pending"); return }

		uid := getUserID(r)
		d := &db.ApprovalDecision{RequestID: id, ApproverID: uid, Decision: "approved", Comment: req.Comment}
		decisionRepos.Create(r.Context(), d)

		reqRepos.IncrementApproval(r.Context(), id)

		approvalReq.ApprovalCount++
		if approvalReq.ApprovalCount >= approvalReq.RequiredCount {
			reqRepos.UpdateStatus(r.Context(), id, "approved")
		}

		iamAudit(r.Context(), audits, uid, "approval.approve", "approval_request", id, nil, r)
		writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
	}
}

func rejectRequestHandler(reqRepos *db.ApprovalRequestRepository, decisionRepos *db.ApprovalDecisionRepository, audits *db.IAMAuditRepository) http.HandlerFunc {
	type request struct {
		Comment string `json:"comment"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var req request
		json.NewDecoder(r.Body).Decode(&req)

		approvalReq, err := reqRepos.FindByID(r.Context(), id)
		if err != nil { writeError(w, http.StatusNotFound, "not_found", "approval request not found"); return }
		if approvalReq.Status != "pending" { writeError(w, http.StatusConflict, "not_pending", "request is not pending"); return }

		uid := getUserID(r)
		d := &db.ApprovalDecision{RequestID: id, ApproverID: uid, Decision: "rejected", Comment: req.Comment}
		decisionRepos.Create(r.Context(), d)

		reqRepos.UpdateStatus(r.Context(), id, "rejected")

		iamAudit(r.Context(), audits, uid, "approval.reject", "approval_request", id, nil, r)
		writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
	}
}

// --- IAM Audit ---

func iamAuditLogHandler(repos *db.IAMAuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 50
		offset := 0
		list, err := repos.List(r.Context(), limit, offset)
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", "failed to list IAM audit log"); return }
		if list == nil { list = []*db.IAMAuditEntry{} }
		writeJSON(w, http.StatusOK, list)
	}
}

// --- Helpers ---

func iamAudit(ctx context.Context, repos *db.IAMAuditRepository, actorID, action, resourceType, resourceID string, details json.RawMessage, r *http.Request) {
	ip := r.RemoteAddr
	entry := &db.IAMAuditEntry{ActorID: actorID, Action: action, ResourceType: resourceType, ResourceID: resourceID, Details: details, IPAddress: ip}
	repos.Create(ctx, entry)
}

func generateTokenPrefix() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		for i := range b {
			b[i] = "abcdefghijklmnopqrstuvwxyz0123456789"[time.Now().UnixNano()%36]
			time.Sleep(time.Nanosecond)
		}
	}
	return hex.EncodeToString(b)
}
