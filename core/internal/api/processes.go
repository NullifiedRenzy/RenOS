package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/ws"
)

type processSnapshotMessage struct {
	DeviceID  string             `json:"device_id"`
	Processes []*db.ProcessSnapshot `json:"processes"`
}

type processActionMessage struct {
	ActionID string `json:"action_id"`
	DeviceID string `json:"device_id"`
	PID      int    `json:"pid"`
	Action   string `json:"action"`
}

type processActionResultMessage struct {
	ActionID string `json:"action_id"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
	Error    string `json:"error"`
}

// HandleProcessSnapshot receives a full process list from the agent and persists it
func HandleProcessSnapshot(processes *db.ProcessRepository, hub *ws.Hub, history *db.ProcessRepository) func(*nats.Msg) {
	return func(msg *nats.Msg) {
		var snapshotMsg processSnapshotMessage
		if err := json.Unmarshal(msg.Data, &snapshotMsg); err != nil {
			return
		}

		ctx := context.Background()

		// Upsert all process snapshots
		if err := processes.UpsertSnapshot(ctx, snapshotMsg.DeviceID, snapshotMsg.Processes); err != nil {
			return
		}

		// Compute aggregates and record history
		totalCPU, totalMemBytes, totalMemUsage, count, err := processes.GetAggregates(ctx, snapshotMsg.DeviceID)
		if err == nil && count > 0 {
			_ = history.RecordHistory(ctx, snapshotMsg.DeviceID, totalCPU, totalMemBytes, totalMemUsage, count)
		}

		// Broadcast to WebSocket clients
		hub.Broadcast(ws.Message{
			Type:     "process.snapshot",
			DeviceID: snapshotMsg.DeviceID,
			Data: map[string]interface{}{
				"device_id":    snapshotMsg.DeviceID,
				"process_count": len(snapshotMsg.Processes),
			},
		})
	}
}

// HandleProcessActionResult receives a completed process action result from the agent
func HandleProcessActionResult(processes *db.ProcessRepository, audits *db.AuditRepository, hub *ws.Hub) func(*nats.Msg) {
	return func(msg *nats.Msg) {
		var resultMsg processActionResultMessage
		if err := json.Unmarshal(msg.Data, &resultMsg); err != nil {
			return
		}

		ctx := context.Background()

		action, err := processes.FindActionByID(ctx, resultMsg.ActionID)
		if err != nil {
			return
		}

		status := "completed"
		if resultMsg.ExitCode != 0 {
			status = "failed"
		}

		if err := processes.UpdateActionStatus(ctx, resultMsg.ActionID, status, resultMsg.ExitCode, resultMsg.Output, resultMsg.Error); err != nil {
			return
		}

		audits.Create(ctx, "device", action.DeviceID, "process_action."+action.ActionType, "process", strconv.Itoa(action.PID))
		hub.Broadcast(ws.Message{
			Type:     "process.action.completed",
			DeviceID: action.DeviceID,
			Data: map[string]interface{}{
				"action_id":  resultMsg.ActionID,
				"pid":        action.PID,
				"action":     action.ActionType,
				"status":     status,
				"exit_code":  resultMsg.ExitCode,
			},
		})
	}
}

// listProcessesHandler returns current process snapshots for a device
func listProcessesHandler(processes *db.ProcessRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		list, err := processes.ListByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list processes")
			return
		}
		if list == nil {
			list = []*db.ProcessSnapshot{}
		}

		totalCPU, totalMemBytes, totalMemUsage, count, _ := processes.GetAggregates(r.Context(), deviceID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"processes":       list,
			"total_cpu":       totalCPU,
			"total_memory_bytes": totalMemBytes,
			"total_memory_usage": totalMemUsage,
			"process_count":   count,
		})
	}
}

// searchProcessesHandler searches processes by name/executable/command
func searchProcessesHandler(processes *db.ProcessRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		query := r.URL.Query().Get("q")
		if deviceID == "" || query == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id and query required")
			return
		}

		list, err := processes.SearchByName(r.Context(), deviceID, query)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to search processes")
			return
		}
		if list == nil {
			list = []*db.ProcessSnapshot{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"processes": list,
		})
	}
}

// getProcessHistoryHandler returns CPU/memory history for graphs
func getProcessHistoryHandler(processes *db.ProcessRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		minutes := 60
		if v := r.URL.Query().Get("minutes"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 240 {
				minutes = n
			}
		}

		list, err := processes.GetHistory(r.Context(), deviceID, minutes)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to get process history")
			return
		}
		if list == nil {
			list = []*db.ProcessHistoryEntry{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"history": list,
		})
	}
}

// getProcessAggregatesHandler returns current totals for a device
func getProcessAggregatesHandler(processes *db.ProcessRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		totalCPU, totalMemBytes, totalMemUsage, count, err := processes.GetAggregates(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to get aggregates")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_cpu":          totalCPU,
			"total_memory_bytes": totalMemBytes,
			"total_memory_usage": totalMemUsage,
			"process_count":      count,
		})
	}
}

// createProcessActionHandler sends a process action (kill/terminate/suspend/resume/restart) to the agent
func createProcessActionHandler(
	processes *db.ProcessRepository,
	devices *db.DeviceRepository,
	audits *db.AuditRepository,
	nc *nats.Conn,
	hub *ws.Hub,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		device, err := devices.FindByID(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Device not found")
			return
		}
		if device.Status != "online" {
			writeError(w, http.StatusConflict, "DEVICE_OFFLINE", "Device is not online")
			return
		}

		claims := r.Context().Value(claimsKey).(*auth.Claims)

		var req struct {
			PID        int    `json:"pid"`
			ActionType string `json:"action_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.PID <= 0 {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "pid must be positive")
			return
		}
		validActions := map[string]bool{"kill": true, "terminate": true, "suspend": true, "resume": true, "restart": true}
		if !validActions[req.ActionType] {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "action_type must be kill, terminate, suspend, resume, or restart")
			return
		}

		action, err := processes.CreateAction(r.Context(), deviceID, claims.UserID, req.PID, req.ActionType)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create process action")
			return
		}

		// Publish action to NATS for the agent
		natsSubject := "process.action." + deviceID
		msg := processActionMessage{
			ActionID: action.ID,
			DeviceID: deviceID,
			PID:      req.PID,
			Action:   req.ActionType,
		}
		if err := nc.Publish(natsSubject, mustMarshal(msg)); err != nil {
			processes.UpdateActionStatus(r.Context(), action.ID, "failed", 1, "", "failed to dispatch to agent")
			writeError(w, http.StatusServiceUnavailable, "NATS_ERROR", "Failed to dispatch action to agent")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "process_action."+req.ActionType, "device", deviceID)
		hub.Broadcast(ws.Message{
			Type:     "process.action.requested",
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"action_id": action.ID,
				"pid":       req.PID,
				"action":    req.ActionType,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"process_action": action,
		})
	}
}

// listProcessActionsHandler returns action history for a device
func listProcessActionsHandler(processes *db.ProcessRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		limit := 50
		offset := 0
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		if v := r.URL.Query().Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				offset = n
			}
		}

		list, err := processes.ListActionsByDevice(r.Context(), deviceID, limit, offset)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list process actions")
			return
		}
		if list == nil {
			list = []*db.ProcessAction{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"process_actions": list,
		})
	}
}
