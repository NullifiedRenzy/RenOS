package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/ws"
)

func requestContainerActionHandler(actions *db.ContainerActionRepository, devices *db.DeviceRepository, audits *db.AuditRepository, hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		containerID := chi.URLParam(r, "containerId")
		action := chi.URLParam(r, "action")
		if deviceID == "" || containerID == "" || action == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id, container id, and action required")
			return
		}

		if action != "start" && action != "stop" && action != "restart" && action != "logs" && action != "stats" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "action must be start, stop, restart, logs, or stats")
			return
		}

		if _, err := devices.FindByID(r.Context(), deviceID); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Device not found")
			return
		}

		claims := r.Context().Value(claimsKey).(*auth.Claims)

		act, err := actions.Create(r.Context(), deviceID, claims.UserID, containerID, action)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create container action")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "container_action."+action, "device", deviceID)
		hub.Broadcast(ws.Message{
			Type:     "container_action.requested",
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"action_id":     act.ID,
				"container_id":  act.ContainerID,
				"action":        act.Action,
				"status":        act.Status,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"container_action": act,
		})
	}
}

func pendingContainerActionsHandler(actions *db.ContainerActionRepository, devices *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req deviceAuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		if _, apiErr := verifyDeviceSecret(devices, req); apiErr != nil {
			writeError(w, apiErr.status, apiErr.code, apiErr.message)
			return
		}

		list, err := actions.PendingByDevice(r.Context(), req.DeviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch pending container actions")
			return
		}

		if list == nil {
			list = []*db.ContainerAction{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"container_actions": list,
		})
	}
}

type submitContainerActionResultPayload struct {
	DeviceID     string `json:"device_id"`
	DeviceSecret string `json:"device_secret"`
	ExitCode     int    `json:"exit_code"`
	Output       string `json:"output"`
	Error        string `json:"error"`
}

func submitContainerActionResultHandler(actions *db.ContainerActionRepository, devices *db.DeviceRepository, audits *db.AuditRepository, hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actionID := chi.URLParam(r, "actionId")
		if actionID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "action id required")
			return
		}

		var req submitContainerActionResultPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		device, apiErr := verifyDeviceSecret(devices, deviceAuthRequest{DeviceID: req.DeviceID, DeviceSecret: req.DeviceSecret})
		if apiErr != nil {
			writeError(w, apiErr.status, apiErr.code, apiErr.message)
			return
		}

		act, err := actions.FindByID(r.Context(), actionID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Container action not found")
			return
		}
		if act.DeviceID != device.ID {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Action does not belong to this device")
			return
		}
		if act.Status != "pending" && act.Status != "running" {
			writeError(w, http.StatusConflict, "CONFLICT", "Action is not in a pending or running state")
			return
		}

		status := "completed"
		if req.ExitCode != 0 {
			status = "failed"
		}

		updated, err := actions.UpdateResult(r.Context(), actionID, status, req.ExitCode, req.Output, req.Error)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update container action result")
			return
		}

		audits.Create(r.Context(), "device", device.ID, "container_action.completed", "container_action", actionID)
		hub.Broadcast(ws.Message{
			Type:     "container_action.completed",
			DeviceID: device.ID,
			Data: map[string]interface{}{
				"action_id":    actionID,
				"container_id": updated.ContainerID,
				"action":       updated.Action,
				"status":       updated.Status,
				"exit_code":    updated.ExitCode,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"container_action": updated,
		})
	}
}
