package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/db"
)

type syncStorageDevicePayload struct {
	Name           string `json:"name"`
	Size           int64  `json:"size"`
	DeviceType     string `json:"device_type"`
	Model          string `json:"model"`
	MountPoint     string `json:"mount_point"`
	Filesystem     string `json:"filesystem"`
	UUID           string `json:"uuid"`
	IsRemovable    bool   `json:"is_removable"`
	IsReadonly     bool   `json:"is_readonly"`
	UsedBytes      int64  `json:"used_bytes"`
	AvailableBytes int64  `json:"available_bytes"`
}

func syncStorageDevicesHandler(devices *db.StorageDeviceRepository, deviceRepo *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		var req struct {
			DeviceID     string                     `json:"device_id"`
			DeviceSecret string                     `json:"device_secret"`
			Devices      []syncStorageDevicePayload  `json:"devices"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		if _, apiErr := verifyDeviceSecret(deviceRepo, deviceAuthRequest{DeviceID: req.DeviceID, DeviceSecret: req.DeviceSecret}); apiErr != nil {
			writeError(w, apiErr.status, apiErr.code, apiErr.message)
			return
		}

		if req.DeviceID != deviceID {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "device_id mismatch")
			return
		}

		var records []*db.StorageDevice
		for _, d := range req.Devices {
			records = append(records, &db.StorageDevice{
				Name:           d.Name,
				Size:           d.Size,
				DeviceType:     d.DeviceType,
				Model:          d.Model,
				MountPoint:     d.MountPoint,
				Filesystem:     d.Filesystem,
				UUID:           d.UUID,
				IsRemovable:    d.IsRemovable,
				IsReadonly:     d.IsReadonly,
				UsedBytes:      d.UsedBytes,
				AvailableBytes: d.AvailableBytes,
			})
		}

		if err := devices.ReplaceForDevice(r.Context(), deviceID, records); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync storage devices")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "synced"})
	}
}

func listStorageDevicesHandler(devices *db.StorageDeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		list, err := devices.ListByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list storage devices")
			return
		}

		if list == nil {
			list = []*db.StorageDevice{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"devices": list,
		})
	}
}
