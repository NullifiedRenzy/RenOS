package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/db"
)

type composeStackSyncRequest struct {
	DeviceID    string             `json:"device_id"`
	DeviceSecret string            `json:"device_secret"`
	Stacks      []*db.ComposeStack `json:"stacks"`
}

func syncComposeStacksHandler(stacks *db.ComposeStackRepository, devices *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		var req composeStackSyncRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		if _, apiErr := verifyDeviceSecret(devices, deviceAuthRequest{DeviceID: deviceID, DeviceSecret: req.DeviceSecret}); apiErr != nil {
			writeError(w, apiErr.status, apiErr.code, apiErr.message)
			return
		}

		if err := stacks.ReplaceForDevice(r.Context(), deviceID, req.Stacks); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync compose stacks")
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"synced": len(req.Stacks)})
	}
}

func listComposeStacksHandler(stacks *db.ComposeStackRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		list, err := stacks.ListByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list compose stacks")
			return
		}
		if list == nil {
			list = []*db.ComposeStack{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"compose_stacks": list,
		})
	}
}
