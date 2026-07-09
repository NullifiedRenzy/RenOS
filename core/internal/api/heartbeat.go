package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/ws"
)

type heartbeatRequest struct {
	DeviceID     string  `json:"device_id"`
	DeviceSecret string  `json:"device_secret"`
	Hostname     string  `json:"hostname"`
	UptimeSeconds float64 `json:"uptime_seconds"`
	CPUPercent   float64 `json:"cpu_percent"`
	MemoryTotalGB float64 `json:"memory_total_gb"`
	MemoryUsedGB float64 `json:"memory_used_gb"`
	DiskTotalGB  float64 `json:"disk_total_gb"`
	DiskUsedGB   float64 `json:"disk_used_gb"`
	LocalIP      string  `json:"local_ip,omitempty"`
}

func heartbeatHandler(devices *db.DeviceRepository, audits *db.AuditRepository, hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req heartbeatRequest
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

		wasOffline := device.Status != "online"

		healthJSON := fmt.Sprintf(`{"last_health":{"hostname":"%s","uptime_seconds":%.0f,"cpu_percent":%.1f,"memory_total_gb":%.1f,"memory_used_gb":%.1f,"disk_total_gb":%.1f,"disk_used_gb":%.1f,"local_ip":"%s","reported_at":"%s"}}`,
			req.Hostname, req.UptimeSeconds, req.CPUPercent,
			req.MemoryTotalGB, req.MemoryUsedGB, req.DiskTotalGB, req.DiskUsedGB,
			req.LocalIP, time.Now().UTC().Format(time.RFC3339))

		if err := devices.UpdateStatusAndMetadata(r.Context(), req.DeviceID, "online", healthJSON); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update device")
			return
		}

		hub.Broadcast(ws.Message{
			Type:     "device.health",
			DeviceID: req.DeviceID,
			Data: map[string]interface{}{
				"cpu_percent":     req.CPUPercent,
				"memory_used_gb":  req.MemoryUsedGB,
				"memory_total_gb": req.MemoryTotalGB,
				"disk_used_gb":    req.DiskUsedGB,
				"disk_total_gb":   req.DiskTotalGB,
				"hostname":        req.Hostname,
			},
		})

		// Audit only on offline→online transition, not every heartbeat
		if wasOffline {
			audits.Create(r.Context(), "device", req.DeviceID, "device.online", "device", req.DeviceID)
			hub.Broadcast(ws.Message{
				Type:     "device.online",
				DeviceID: req.DeviceID,
				Data:     map[string]string{"hostname": req.Hostname},
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}
