package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/ws"
)

type syncUpdatePayload struct {
	PackageName      string `json:"package_name"`
	CurrentVersion   string `json:"current_version"`
	AvailableVersion string `json:"available_version"`
	Severity         string `json:"severity"`
	Status           string `json:"status"`
}

func syncUpdatesHandler(updates *db.UpdateStatusRepository, devices *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		var req struct {
			DeviceID     string              `json:"device_id"`
			DeviceSecret string              `json:"device_secret"`
			Updates      []syncUpdatePayload `json:"updates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		if _, apiErr := verifyDeviceSecret(devices, deviceAuthRequest{DeviceID: req.DeviceID, DeviceSecret: req.DeviceSecret}); apiErr != nil {
			writeError(w, apiErr.status, apiErr.code, apiErr.message)
			return
		}

		if req.DeviceID != deviceID {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "device_id mismatch")
			return
		}

		var records []*db.UpdateStatus
		for _, u := range req.Updates {
			records = append(records, &db.UpdateStatus{
				PackageName:      u.PackageName,
				CurrentVersion:   u.CurrentVersion,
				AvailableVersion: u.AvailableVersion,
				Severity:         u.Severity,
				Status:           u.Status,
			})
		}

		if err := updates.ReplaceForDevice(r.Context(), deviceID, records); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync updates")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "synced"})
	}
}

func listUpdatesHandler(updates *db.UpdateStatusRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		list, err := updates.ListByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list updates")
			return
		}

		if list == nil {
			list = []*db.UpdateStatus{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"updates": list,
		})
	}
}

func requestUpdateActionHandler(updates *db.UpdateStatusRepository, devices *db.DeviceRepository, audits *db.AuditRepository, hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		action := chi.URLParam(r, "action")
		if deviceID == "" || action == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id and action required")
			return
		}

		if action != "check" && action != "upgrade" && action != "dry-run" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "action must be check, upgrade, or dry-run")
			return
		}

		if _, err := devices.FindByID(r.Context(), deviceID); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Device not found")
			return
		}

		claims := r.Context().Value(claimsKey).(*auth.Claims)
		audits.Create(r.Context(), "user", claims.UserID, "update_action."+action, "device", deviceID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   "update action queued",
			"device_id": deviceID,
			"action":    action,
		})
	}
}
