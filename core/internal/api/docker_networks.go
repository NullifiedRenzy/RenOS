package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/db"
)

type networkPayload struct {
	NetworkID string `json:"network_id"`
	Name      string `json:"name"`
	Driver    string `json:"driver"`
	Scope     string `json:"scope"`
}

func syncDockerNetworksHandler(networks *db.DockerNetworkRepository, devices *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		var req struct {
			DeviceID     string           `json:"device_id"`
			DeviceSecret string           `json:"device_secret"`
			Networks     []networkPayload `json:"networks"`
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

		var records []*db.DockerNetwork
		for _, n := range req.Networks {
			records = append(records, &db.DockerNetwork{
				NetworkID: n.NetworkID,
				Name:      n.Name,
				Driver:    n.Driver,
				Scope:     n.Scope,
			})
		}

		if err := networks.ReplaceForDevice(r.Context(), deviceID, records); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync docker networks")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "synced"})
	}
}

func listDockerNetworksHandler(networks *db.DockerNetworkRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		list, err := networks.ListByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list Docker networks")
			return
		}

		if list == nil {
			list = []*db.DockerNetwork{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"networks": list,
		})
	}
}
