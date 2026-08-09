package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
)

type listUserItem struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	CreatedAt   string `json:"created_at"`
}

func listUsersHandler(users *db.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userList, err := users.ListAll(r.Context(), parseListLimit(r.URL.Query().Get("limit")), parseOffset(r.URL.Query().Get("offset")))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list users")
			return
		}

		items := make([]listUserItem, 0, len(userList))
		for _, u := range userList {
			items = append(items, listUserItem{
				ID:          u.ID,
				Email:       u.Email,
				DisplayName: u.DisplayName,
				Role:        u.Role,
				CreatedAt:   u.CreatedAt.Format("2006-01-02T15:04:05Z"),
			})
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"users": items,
		})
	}
}

type updateRolePayload struct {
	Role string `json:"role"`
}

func updateUserRoleHandler(users *db.UserRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		targetUserID := chi.URLParam(r, "id")

		// Only admin can change roles
		if !auth.RoleIs(claims.Role, "admin") {
			audits.Create(r.Context(), "system", "rbac", "permission.denied", "user", claims.UserID)
			writePermissionDenied(w)
			return
		}

		// Cannot change your own role (prevent privilege escalation through self-demotion)
		if claims.UserID == targetUserID {
			writeError(w, http.StatusForbidden, "CANNOT_SELF_ASSIGN", "Cannot change your own role")
			return
		}

		var req updateRolePayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		if !db.IsValidRole(req.Role) {
			writeError(w, http.StatusBadRequest, "INVALID_ROLE", "Role must be one of: admin, operator, viewer")
			return
		}

		// Prevent assigning a role higher than the caller's own role
		if !db.CanAssignRole(claims.Role, req.Role) {
			audits.Create(r.Context(), "system", "rbac", "permission.denied", "user", claims.UserID)
			writePermissionDenied(w)
			return
		}

		prevRole, err := users.FindByID(r.Context(), targetUserID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}

		if err := users.UpdateRole(r.Context(), targetUserID, req.Role); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update role")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "role.assigned", "user", targetUserID)
		audits.Create(r.Context(), "user", claims.UserID, "role.removed", "user", targetUserID)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "updated",
			"user": map[string]string{
				"id":         targetUserID,
				"role":       req.Role,
				"previous":   prevRole.Role,
			},
		})
	}
}
