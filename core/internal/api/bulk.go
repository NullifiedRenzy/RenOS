package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/policy"
)

type runGroupCommandPayload struct {
	Command string `json:"command"`
}

func runGroupCommandHandler(groups *db.GroupRepository, groupActions *db.GroupActionRepository, jobs *db.JobRepository, audits *db.AuditRepository, enforcer *policy.Enforcer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		groupID := chi.URLParam(r, "id")

		var req runGroupCommandPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.Command == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "command is required")
			return
		}

		deviceIDs, err := groups.ListDeviceIDs(r.Context(), groupID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Group not found")
			return
		}
		if len(deviceIDs) == 0 {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Group has no devices")
			return
		}

		if r := enforcer.Evaluate(r.Context(), &policy.EvalContext{
			ActionType: "command",
			DeviceIDs:  deviceIDs,
			Command:    req.Command,
		}); r.Action == "deny" {
			writeError(w, http.StatusForbidden, "POLICY_DENIED", r.Message)
			return
		}

		ga, err := groupActions.Create(r.Context(), &groupID, "command", "command", req.Command, claims.UserID, len(deviceIDs))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create group action")
			return
		}

		payload := map[string]interface{}{
			"command":    req.Command,
			"device_ids": deviceIDs,
			"group_id":   groupID,
		}
		payloadBytes, _ := json.Marshal(payload)

		job, err := jobs.Create(r.Context(), "command", payloadBytes, len(deviceIDs), claims.UserID, &ga.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create job")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "group.command", "group", groupID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"group_action": ga,
			"job":          job,
		})
	}
}

type multiDeviceCommandPayload struct {
	DeviceIDs []string `json:"device_ids"`
	Command   string   `json:"command"`
}

func multiDeviceCommandHandler(groups *db.GroupRepository, groupActions *db.GroupActionRepository, jobs *db.JobRepository, audits *db.AuditRepository, enforcer *policy.Enforcer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)

		var req multiDeviceCommandPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.Command == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "command is required")
			return
		}
		if len(req.DeviceIDs) == 0 {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "at least one device_id is required")
			return
		}

		if r := enforcer.Evaluate(r.Context(), &policy.EvalContext{
			ActionType: "command",
			DeviceIDs:  req.DeviceIDs,
			Command:    req.Command,
		}); r.Action == "deny" {
			writeError(w, http.StatusForbidden, "POLICY_DENIED", r.Message)
			return
		}

		ga, err := groupActions.Create(r.Context(), nil, "command", "command", req.Command, claims.UserID, len(req.DeviceIDs))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create group action")
			return
		}

		payload := map[string]interface{}{
			"command":    req.Command,
			"device_ids": req.DeviceIDs,
		}
		payloadBytes, _ := json.Marshal(payload)

		job, err := jobs.Create(r.Context(), "command", payloadBytes, len(req.DeviceIDs), claims.UserID, &ga.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create job")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "multi_device.command", "multi_device", "")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"group_action": ga,
			"job":          job,
		})
	}
}

func getGroupActionHandler(groupActions *db.GroupActionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actionID := chi.URLParam(r, "actionId")

		ga, err := groupActions.FindByID(r.Context(), actionID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Group action not found")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"group_action": ga,
		})
	}
}

func listGroupActionsHandler(groupActions *db.GroupActionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID := chi.URLParam(r, "id")

		list, err := groupActions.ListByGroup(r.Context(), groupID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Group not found")
			return
		}
		if list == nil {
			list = []*db.GroupAction{}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"group_actions": list,
		})
	}
}

type groupContainerActionPayload struct {
	ContainerID string `json:"container_id"`
	Action      string `json:"action"`
}

func groupContainerActionHandler(groups *db.GroupRepository, groupActions *db.GroupActionRepository, jobs *db.JobRepository, audits *db.AuditRepository, enforcer *policy.Enforcer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		groupID := chi.URLParam(r, "id")

		var req groupContainerActionPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.ContainerID == "" || req.Action == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "container_id and action required")
			return
		}
		if req.Action != "start" && req.Action != "stop" && req.Action != "restart" && req.Action != "logs" && req.Action != "stats" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "action must be start, stop, restart, logs, or stats")
			return
		}

		deviceIDs, err := groups.ListDeviceIDs(r.Context(), groupID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Group not found")
			return
		}
		if len(deviceIDs) == 0 {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Group has no devices")
			return
		}

		if r := enforcer.Evaluate(r.Context(), &policy.EvalContext{
			ActionType:   "container_action",
			DeviceIDs:    deviceIDs,
			ContainerID:  req.ContainerID,
			Action:       req.Action,
		}); r.Action == "deny" {
			writeError(w, http.StatusForbidden, "POLICY_DENIED", r.Message)
			return
		}

		ga, err := groupActions.Create(r.Context(), &groupID, "container_action", "container", req.ContainerID, claims.UserID, len(deviceIDs))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create group action")
			return
		}

		payload := map[string]interface{}{
			"container_id": req.ContainerID,
			"action":       req.Action,
			"device_ids":   deviceIDs,
			"group_id":     groupID,
		}
		payloadBytes, _ := json.Marshal(payload)

		job, err := jobs.Create(r.Context(), "container_action", payloadBytes, len(deviceIDs), claims.UserID, &ga.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create job")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "group.container_action."+req.Action, "group", groupID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"group_action": ga,
			"job":          job,
		})
	}
}

type groupServiceActionPayload struct {
	UnitName string `json:"unit_name"`
	Action   string `json:"action"`
}

func groupServiceActionHandler(groups *db.GroupRepository, groupActions *db.GroupActionRepository, jobs *db.JobRepository, audits *db.AuditRepository, enforcer *policy.Enforcer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		groupID := chi.URLParam(r, "id")

		var req groupServiceActionPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.UnitName == "" || req.Action == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "unit_name and action required")
			return
		}
		if req.Action != "start" && req.Action != "stop" && req.Action != "restart" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "action must be start, stop, or restart")
			return
		}

		deviceIDs, err := groups.ListDeviceIDs(r.Context(), groupID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Group not found")
			return
		}
		if len(deviceIDs) == 0 {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Group has no devices")
			return
		}

		if r := enforcer.Evaluate(r.Context(), &policy.EvalContext{
			ActionType: "service_action",
			DeviceIDs:  deviceIDs,
			UnitName:   req.UnitName,
			Action:     req.Action,
		}); r.Action == "deny" {
			writeError(w, http.StatusForbidden, "POLICY_DENIED", r.Message)
			return
		}

		ga, err := groupActions.Create(r.Context(), &groupID, "service_action", "service", req.UnitName, claims.UserID, len(deviceIDs))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create group action")
			return
		}

		payload := map[string]interface{}{
			"unit_name":  req.UnitName,
			"action":     req.Action,
			"device_ids": deviceIDs,
			"group_id":   groupID,
		}
		payloadBytes, _ := json.Marshal(payload)

		job, err := jobs.Create(r.Context(), "service_action", payloadBytes, len(deviceIDs), claims.UserID, &ga.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create job")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "group.service_action."+req.Action, "group", groupID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"group_action": ga,
			"job":          job,
		})
	}
}

type groupUpdateActionPayload struct {
	Action string `json:"action"`
}

func groupUpdateActionHandler(groups *db.GroupRepository, groupActions *db.GroupActionRepository, jobs *db.JobRepository, audits *db.AuditRepository, enforcer *policy.Enforcer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		groupID := chi.URLParam(r, "id")

		var req groupUpdateActionPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.Action == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "action required")
			return
		}
		if req.Action != "check" && req.Action != "upgrade" && req.Action != "dry-run" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "action must be check, upgrade, or dry-run")
			return
		}

		deviceIDs, err := groups.ListDeviceIDs(r.Context(), groupID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Group not found")
			return
		}
		if len(deviceIDs) == 0 {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Group has no devices")
			return
		}

		if r := enforcer.Evaluate(r.Context(), &policy.EvalContext{
			ActionType:   "update",
			DeviceIDs:    deviceIDs,
			UpdateAction: req.Action,
		}); r.Action == "deny" {
			writeError(w, http.StatusForbidden, "POLICY_DENIED", r.Message)
			return
		}

		ga, err := groupActions.Create(r.Context(), &groupID, "update", "update", req.Action, claims.UserID, len(deviceIDs))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create group action")
			return
		}

		payload := map[string]interface{}{
			"action":     req.Action,
			"device_ids": deviceIDs,
			"group_id":   groupID,
		}
		payloadBytes, _ := json.Marshal(payload)

		job, err := jobs.Create(r.Context(), "update", payloadBytes, len(deviceIDs), claims.UserID, &ga.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create job")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "group.update."+req.Action, "group", groupID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"group_action": ga,
			"job":          job,
		})
	}
}
