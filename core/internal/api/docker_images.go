package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/db"
)

type imagePayload struct {
	ImageID    string `json:"image_id"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Size       string `json:"size"`
	Created    string `json:"created"`
}

func syncDockerImagesHandler(images *db.DockerImageRepository, devices *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		var req struct {
			DeviceID     string         `json:"device_id"`
			DeviceSecret string         `json:"device_secret"`
			Images       []imagePayload `json:"images"`
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

		var records []*db.DockerImage
		for _, img := range req.Images {
			records = append(records, &db.DockerImage{
				ImageID:    img.ImageID,
				Repository: img.Repository,
				Tag:        img.Tag,
				Size:       img.Size,
				Created:    img.Created,
			})
		}

		if err := images.ReplaceForDevice(r.Context(), deviceID, records); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync docker images")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "synced"})
	}
}

func listDockerImagesHandler(images *db.DockerImageRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		list, err := images.ListByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list Docker images")
			return
		}

		if list == nil {
			list = []*db.DockerImage{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"images": list,
		})
	}
}
