package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/db"
)

type servicePayload struct {
	UnitName    string `json:"unit_name"`
	LoadState   string `json:"load_state"`
	ActiveState string `json:"active_state"`
	SubState    string `json:"sub_state"`
	Description string `json:"description"`
	Enabled     string `json:"enabled"`
}

func syncSystemdServicesHandler(services *db.SystemdServiceRepository, devices *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		var req struct {
			DeviceID     string           `json:"device_id"`
			DeviceSecret string           `json:"device_secret"`
			Services     []servicePayload `json:"services"`
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

		var records []*db.SystemdService
		for _, s := range req.Services {
			records = append(records, &db.SystemdService{
				UnitName:    s.UnitName,
				LoadState:   s.LoadState,
				ActiveState: s.ActiveState,
				SubState:    s.SubState,
				Description: s.Description,
				Enabled:     s.Enabled,
			})
		}

		if err := services.ReplaceForDevice(r.Context(), deviceID, records); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync systemd services")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "synced"})
	}
}

func listSystemdServicesHandler(services *db.SystemdServiceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		list, err := services.ListByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list systemd services")
			return
		}

		if list == nil {
			list = []*db.SystemdService{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"services": list,
		})
	}
}
