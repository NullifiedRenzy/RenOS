package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
)

type upsertFilePermissionPayload struct {
	UserID       string          `json:"user_id"`
	CanBrowse    bool            `json:"can_browse"`
	CanModify    bool            `json:"can_modify"`
	CanUpload    bool            `json:"can_upload"`
	CanDownload  bool            `json:"can_download"`
	CanDelete    bool            `json:"can_delete"`
	AllowedRoots json.RawMessage `json:"allowed_roots"`
}

func upsertFilePermissionHandler(perms *db.FilePermissionRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		claims := r.Context().Value(claimsKey).(*auth.Claims)

		var req upsertFilePermissionPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.UserID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "user_id is required")
			return
		}

		perm, err := perms.Upsert(r.Context(), deviceID, req.UserID, req.CanBrowse, req.CanModify, req.CanUpload, req.CanDownload, req.CanDelete, req.AllowedRoots)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update file permissions")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "file_permission.update", "device", deviceID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"file_permission": perm,
		})
	}
}

func getFilePermissionForUserHandler(perms *db.FilePermissionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		userID := chi.URLParam(r, "userId")
		if deviceID == "" || userID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id and user id required")
			return
		}

		perm, err := perms.GetForUser(r.Context(), deviceID, userID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "File permission not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"file_permission": perm,
		})
	}
}

func listFilePermissionsHandler(perms *db.FilePermissionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		list, err := perms.ListByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list file permissions")
			return
		}
		if list == nil {
			list = []*db.FilePermission{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"file_permissions": list,
		})
	}
}

func deleteFilePermissionHandler(perms *db.FilePermissionRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		userID := chi.URLParam(r, "userId")
		if deviceID == "" || userID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id and user id required")
			return
		}

		claims := r.Context().Value(claimsKey).(*auth.Claims)

		if err := perms.Delete(r.Context(), deviceID, userID); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete file permission")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "file_permission.delete", "device", deviceID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "deleted"})
	}
}
