package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/ws"
)

func listDevicesHandler(devices *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := devices.FindAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list devices")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"devices": list,
		})
	}
}

func getDeviceHandler(devices *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		device, err := devices.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Device not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"device": device,
		})
	}
}

// M06 uses a temporary device_secret for authentication until mTLS replaces it in M07.
// The secret is obtained during enrollment and verified against a bcrypt hash in device metadata.
type connectRequest struct {
	DeviceID     string `json:"device_id"`
	DeviceSecret string `json:"device_secret"`
}

func connectHandler(devices *db.DeviceRepository, audits *db.AuditRepository, hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req connectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		if req.DeviceID == "" || req.DeviceSecret == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device_id and device_secret required")
			return
		}

		device, err := devices.FindByID(r.Context(), req.DeviceID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Device not found")
			return
		}

		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(device.Metadata), &metadata); err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to parse device metadata")
			return
		}

		hash, ok := metadata["device_secret_hash"].(string)
		if !ok || hash == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Device not enrolled")
			return
		}

		if !auth.VerifyPassword(hash, req.DeviceSecret) {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid device secret")
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
