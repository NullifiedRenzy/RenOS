package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/policy"
	"github.com/renos-project/renos/core/internal/ws"
)

type requestCommandPayload struct {
	Command string `json:"command"`
}

func requestCommandHandler(commands *db.CommandRepository, devices *db.DeviceRepository, audits *db.AuditRepository, hub *ws.Hub, enforcer *policy.Enforcer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		if _, err := devices.FindByID(r.Context(), deviceID); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Device not found")
			return
		}

		var req requestCommandPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.Command == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "command is required")
			return
		}

		claims := r.Context().Value(claimsKey).(*auth.Claims)

		if r := enforcer.Evaluate(r.Context(), &policy.EvalContext{
			ActionType: "command",
			DeviceIDs:  []string{deviceID},
			Command:    req.Command,
		}); r.Action == "deny" {
			writeError(w, http.StatusForbidden, "POLICY_DENIED", r.Message)
			return
		}

		cmd, err := commands.Create(r.Context(), deviceID, claims.UserID, req.Command)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create command")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "command.requested", "device", deviceID)
		hub.Broadcast(ws.Message{
			Type:     "command.requested",
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"command_id": cmd.ID,
				"command":    cmd.Command,
				"status":     cmd.Status,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"command": cmd,
		})
	}
}

func listCommandsHandler(commands *db.CommandRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		list, err := commands.ListByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list commands")
			return
		}

		if list == nil {
			list = []*db.Command{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"commands": list,
		})
	}
}

func getCommandHandler(commands *db.CommandRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		cmdID := chi.URLParam(r, "cmdId")

		cmd, err := commands.FindByID(r.Context(), cmdID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Command not found")
			return
		}
		if cmd.DeviceID != deviceID {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Command not found for this device")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"command": cmd,
		})
	}
}

func pendingCommandsHandler(commands *db.CommandRepository, devices *db.DeviceRepository) http.HandlerFunc {
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

		list, err := commands.PendingByDevice(r.Context(), req.DeviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch pending commands")
			return
		}

		if list == nil {
			list = []*db.Command{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"commands": list,
		})
	}
}

type submitResultPayload struct {
	DeviceID     string `json:"device_id"`
	DeviceSecret string `json:"device_secret"`
	ExitCode     int    `json:"exit_code"`
	Output       string `json:"output"`
	Error        string `json:"error"`
}

func submitResultHandler(commands *db.CommandRepository, devices *db.DeviceRepository, groupActions *db.GroupActionRepository, audits *db.AuditRepository, hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cmdID := chi.URLParam(r, "cmdId")
		if cmdID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "command id required")
			return
		}

		var req submitResultPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		device, apiErr := verifyDeviceSecret(devices, deviceAuthRequest{DeviceID: req.DeviceID, DeviceSecret: req.DeviceSecret})
		if apiErr != nil {
			writeError(w, apiErr.status, apiErr.code, apiErr.message)
			return
		}

		cmd, err := commands.FindByID(r.Context(), cmdID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Command not found")
			return
		}
		if cmd.DeviceID != device.ID {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Command does not belong to this device")
			return
		}
		if cmd.Status != "pending" && cmd.Status != "running" {
			writeError(w, http.StatusConflict, "CONFLICT", "Command is not in a pending or running state")
			return
		}

		status := "completed"
		if req.ExitCode != 0 {
			status = "failed"
		}

		updated, err := commands.UpdateResult(r.Context(), cmdID, status, req.ExitCode, req.Output, req.Error)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update command result")
			return
		}

		if ga, err := groupActions.FindByCommandID(r.Context(), cmdID); err == nil && ga != nil {
			failedDelta := 0
			completedDelta := 0
			if status == "completed" {
				completedDelta = 1
			} else {
				failedDelta = 1
			}
			groupActions.UpdateResult(r.Context(), ga.ID, completedDelta, failedDelta)
		}

		audits.Create(r.Context(), "device", device.ID, "command.completed", "command", cmdID)
		hub.Broadcast(ws.Message{
			Type:     "command.completed",
			DeviceID: device.ID,
			Data: map[string]interface{}{
				"command_id": cmdID,
				"command":    updated.Command,
				"status":     updated.Status,
				"exit_code":  updated.ExitCode,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"command": updated,
		})
	}
}
