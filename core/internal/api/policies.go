package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/policy"
)

type createPolicyPayload struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        string          `json:"type"`
	Conditions  json.RawMessage `json:"conditions"`
	Action      string          `json:"action"`
	Priority    int             `json:"priority"`
	Enabled     bool            `json:"enabled"`
}

func createPolicyHandler(policies *db.PolicyRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)

		var req createPolicyPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required")
			return
		}
		validTypes := map[string]bool{"command": true, "container_action": true, "service_action": true, "update": true, "schedule": true, "job": true, "*": true}
		if !validTypes[req.Type] {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "type must be command, container_action, service_action, update, schedule, job, or *")
			return
		}
		validActions := map[string]bool{"allow": true, "deny": true, "warn": true}
		if !validActions[req.Action] {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "action must be allow, deny, or warn")
			return
		}
		if err := policy.ValidatePolicyConditions(req.Conditions); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_CONDITIONS", err.Error())
			return
		}
		if req.Conditions == nil {
			req.Conditions = json.RawMessage("{}")
		}

		p, err := policies.Create(r.Context(), req.Name, req.Description, req.Type, req.Conditions, req.Action, req.Priority, req.Enabled, &claims.UserID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create policy")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "policy.created", "policy", p.ID)

		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"policy": p,
		})
	}
}

func listPoliciesHandler(policies *db.PolicyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := policies.FindAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list policies")
			return
		}
		if list == nil {
			list = []*db.Policy{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"policies": list,
		})
	}
}

func getPolicyHandler(policies *db.PolicyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		p, err := policies.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Policy not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"policy": p,
		})
	}
}

type updatePolicyPayload struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        string          `json:"type"`
	Conditions  json.RawMessage `json:"conditions"`
	Action      string          `json:"action"`
	Priority    int             `json:"priority"`
	Enabled     bool            `json:"enabled"`
}

func updatePolicyHandler(policies *db.PolicyRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		id := chi.URLParam(r, "id")

		var req updatePolicyPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required")
			return
		}
		validTypes := map[string]bool{"command": true, "container_action": true, "service_action": true, "update": true, "schedule": true, "job": true, "*": true}
		if !validTypes[req.Type] {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "type must be command, container_action, service_action, update, schedule, job, or *")
			return
		}
		validActions := map[string]bool{"allow": true, "deny": true, "warn": true}
		if !validActions[req.Action] {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "action must be allow, deny, or warn")
			return
		}
		if err := policy.ValidatePolicyConditions(req.Conditions); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_CONDITIONS", err.Error())
			return
		}
		if req.Conditions == nil {
			req.Conditions = json.RawMessage("{}")
		}

		p, err := policies.Update(r.Context(), id, req.Name, req.Description, req.Type, req.Conditions, req.Action, req.Priority, req.Enabled)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Policy not found")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "policy.updated", "policy", id)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"policy": p,
		})
	}
}

func deletePolicyHandler(policies *db.PolicyRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		id := chi.URLParam(r, "id")

		if err := policies.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Policy not found")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "policy.deleted", "policy", id)

		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func enablePolicyHandler(policies *db.PolicyRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		id := chi.URLParam(r, "id")

		if err := policies.SetEnabled(r.Context(), id, true); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Policy not found")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "policy.enabled", "policy", id)

		writeJSON(w, http.StatusOK, map[string]string{"status": "enabled"})
	}
}

func disablePolicyHandler(policies *db.PolicyRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		id := chi.URLParam(r, "id")

		if err := policies.SetEnabled(r.Context(), id, false); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Policy not found")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "policy.disabled", "policy", id)

		writeJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
	}
}
