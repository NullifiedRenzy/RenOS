package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/db"
)

type containerMetricsSyncRequest struct {
	DeviceID     string                 `json:"device_id"`
	DeviceSecret string                 `json:"device_secret"`
	Metrics      []*db.ContainerMetric  `json:"metrics"`
}

func syncContainerMetricsHandler(metrics *db.ContainerMetricsRepository, devices *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		var req containerMetricsSyncRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		if _, apiErr := verifyDeviceSecret(devices, deviceAuthRequest{DeviceID: deviceID, DeviceSecret: req.DeviceSecret}); apiErr != nil {
			writeError(w, apiErr.status, apiErr.code, apiErr.message)
			return
		}

		if err := metrics.InsertMetrics(r.Context(), deviceID, req.Metrics); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to insert container metrics")
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"inserted": len(req.Metrics)})
	}
}

func getLatestContainerMetricsHandler(metrics *db.ContainerMetricsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		list, err := metrics.GetLatest(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to get container metrics")
			return
		}
		if list == nil {
			list = []*db.ContainerMetric{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"metrics": list,
		})
	}
}

func getContainerMetricHistoryHandler(metrics *db.ContainerMetricsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		containerID := chi.URLParam(r, "containerId")
		if deviceID == "" || containerID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id and container id required")
			return
		}

		minutes := 30
		if v := r.URL.Query().Get("minutes"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 240 {
				minutes = n
			}
		}

		list, err := metrics.GetHistory(r.Context(), deviceID, containerID, minutes)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to get container metric history")
			return
		}
		if list == nil {
			list = []*db.ContainerMetric{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"metrics": list,
		})
	}
}
