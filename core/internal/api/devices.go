package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/ws"
)

func listDevicesHandler(devices *db.DeviceRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)

		var list []*db.Device
		var err error
		if claims.Role == "admin" {
			list, err = devices.ListWithLimit(r.Context(), "", parseListLimit(r.URL.Query().Get("limit")), parseOffset(r.URL.Query().Get("offset")))
		} else {
			list, err = devices.ListWithLimit(r.Context(), claims.UserID, parseListLimit(r.URL.Query().Get("limit")), parseOffset(r.URL.Query().Get("offset")))
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list devices")
			return
		}

		slog.Info("devices listed", "user_id", claims.UserID, "role", claims.Role, "count", len(list))
		audits.Create(r.Context(), "user", claims.UserID, "device.list", "device", "")

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"devices": list,
		})
	}
}

func getDeviceHandler(devices *db.DeviceRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		id := chi.URLParam(r, "id")
		device, err := devices.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Device not found")
			return
		}

		// Non-admin users can only see their own devices
		if claims.Role != "admin" && device.UserID != claims.UserID {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Device not found")
			return
		}

		slog.Info("device fetched", "user_id", claims.UserID, "device_id", id)
		audits.Create(r.Context(), "user", claims.UserID, "device.get", "device", id)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"device": device,
		})
	}
}

// M06 uses a temporary device_secret for authentication until mTLS replaces it in M07.
// The secret is obtained during enrollment and verified against a bcrypt hash in device metadata.
func connectHandler(devices *db.DeviceRepository, audits *db.AuditRepository, hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req deviceAuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		device, apiErr := verifyDeviceSecret(devices, req)
		if apiErr != nil {
			writeError(w, apiErr.status, apiErr.code, apiErr.message)
			return
		}

		if err := devices.UpdateStatus(r.Context(), req.DeviceID, "online"); err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update device status")
			return
		}

		audits.Create(r.Context(), "device", req.DeviceID, "device.connected", "device", req.DeviceID)

		hub.Broadcast(ws.Message{
			Type:     "device.online",
			DeviceID: req.DeviceID,
			Data:     map[string]string{"hostname": device.Hostname},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "online",
		})
	}
}
