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

var validFileActionTypes = map[string]bool{
	"list_directory": true, "read_file": true, "search_files": true,
	"create_directory": true, "delete_path": true, "rename_path": true,
	"move_path": true, "copy_path": true, "set_permissions": true,
	"upload_file": true, "download_file": true, "preview_file": true,
}

type requestFileActionPayload struct {
	Path       string          `json:"path"`
	TargetPath string          `json:"target_path,omitempty"`
	Query      string          `json:"query,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

func requestFileActionHandler(actions *db.FileActionRepository, devices *db.DeviceRepository, audits *db.AuditRepository, hub *ws.Hub, enforcer *policy.Enforcer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		actionType := chi.URLParam(r, "actionType")
		if deviceID == "" || actionType == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id and action type required")
			return
		}
		if !validFileActionTypes[actionType] {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "unsupported action type")
			return
		}

		var req requestFileActionPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.Path == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "path is required")
			return
		}

		if _, err := devices.FindByID(r.Context(), deviceID); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Device not found")
			return
		}

		claims := r.Context().Value(claimsKey).(*auth.Claims)

		if r := enforcer.Evaluate(r.Context(), &policy.EvalContext{
			ActionType: "file_action",
			DeviceIDs:  []string{deviceID},
		}); r.Action == "deny" {
			writeError(w, http.StatusForbidden, "POLICY_DENIED", r.Message)
			return
		}

		var act *db.FileAction
		var err error
		if req.TargetPath != "" || req.Metadata != nil {
			act, err = actions.CreateWithTarget(r.Context(), deviceID, claims.UserID, actionType, req.Path, req.TargetPath, req.Query, req.Metadata)
		} else {
			act, err = actions.Create(r.Context(), deviceID, claims.UserID, actionType, req.Path, req.Query)
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create file action")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "file_action."+actionType, "device", deviceID)
		hub.Broadcast(ws.Message{
			Type:     "file_action.requested",
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"action_id":   act.ID,
				"action_type": act.ActionType,
				"path":        act.Path,
				"target_path": act.TargetPath,
				"status":      act.Status,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"file_action": act,
		})
	}
}

func getFileActionResultHandler(actions *db.FileActionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actionID := chi.URLParam(r, "actionId")
		if actionID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "action id required")
			return
		}

		act, err := actions.FindByID(r.Context(), actionID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "File action not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"file_action": act,
		})
	}
}

func listFileActionsHandler(actions *db.FileActionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		list, err := actions.ListByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list file actions")
			return
		}
		if list == nil {
			list = []*db.FileAction{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"file_actions": list,
		})
	}
}

func pendingFileActionsHandler(actions *db.FileActionRepository, devices *db.DeviceRepository) http.HandlerFunc {
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
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch pending file actions")
			return
		}

		if list == nil {
			list = []*db.FileAction{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"file_actions": list,
		})
	}
}

type submitFileActionResultPayload struct {
	DeviceID     string          `json:"device_id"`
	DeviceSecret string          `json:"device_secret"`
	ExitCode     int             `json:"exit_code"`
	Output       string          `json:"output"`
	Error        string          `json:"error"`
	ResultJSON   json.RawMessage `json:"result_json"`
}

func submitFileActionResultHandler(actions *db.FileActionRepository, devices *db.DeviceRepository, audits *db.AuditRepository, hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actionID := chi.URLParam(r, "actionId")
		if actionID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "action id required")
			return
		}

		var req submitFileActionResultPayload
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
			writeError(w, http.StatusNotFound, "NOT_FOUND", "File action not found")
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

		updated, err := actions.UpdateResult(r.Context(), actionID, status, req.ExitCode, req.Output, req.Error, req.ResultJSON)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update file action result")
			return
		}

		audits.Create(r.Context(), "device", device.ID, "file_action.completed", "file_action", actionID)
		hub.Broadcast(ws.Message{
			Type:     "file_action.completed",
			DeviceID: device.ID,
			Data: map[string]interface{}{
				"action_id":   actionID,
				"action_type": updated.ActionType,
				"path":        updated.Path,
				"status":      updated.Status,
				"exit_code":   updated.ExitCode,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"file_action": updated,
		})
	}
}
