package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/db"
)

type volumePayload struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
}

func syncDockerVolumesHandler(volumes *db.DockerVolumeRepository, devices *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		var req struct {
			DeviceID     string         `json:"device_id"`
			DeviceSecret string         `json:"device_secret"`
			Volumes      []volumePayload `json:"volumes"`
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

		var records []*db.DockerVolume
		for _, v := range req.Volumes {
			records = append(records, &db.DockerVolume{
				Name:       v.Name,
				Driver:     v.Driver,
				Mountpoint: v.Mountpoint,
			})
		}

		if err := volumes.ReplaceForDevice(r.Context(), deviceID, records); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync docker volumes")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "synced"})
	}
}

func listDockerVolumesHandler(volumes *db.DockerVolumeRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		list, err := volumes.ListByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list Docker volumes")
			return
		}

		if list == nil {
			list = []*db.DockerVolume{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"volumes": list,
		})
	}
}
