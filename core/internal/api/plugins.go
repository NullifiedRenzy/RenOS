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

type syncPluginPayload struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	PluginType  string  `json:"plugin_type"`
	Status      string  `json:"status"`
	LastOutput  string  `json:"last_output"`
	LastExitCode int32  `json:"last_exit_code"`
	LastRunAt   *string `json:"last_run_at"`
}

func syncPluginsHandler(plugins *db.PluginRepository, devices *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		var req struct {
			DeviceID     string              `json:"device_id"`
			DeviceSecret string              `json:"device_secret"`
			Plugins      []syncPluginPayload `json:"plugins"`
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

		var records []*db.Plugin
		for _, p := range req.Plugins {
			records = append(records, &db.Plugin{
				Name:        p.Name,
				Description: p.Description,
				PluginType:  p.PluginType,
				Status:      p.Status,
				LastOutput:  p.LastOutput,
				LastExitCode: p.LastExitCode,
			})
		}

		if err := plugins.ReplaceForDevice(r.Context(), deviceID, records); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync plugins")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "synced"})
	}
}

func listPluginsHandler(plugins *db.PluginRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		list, err := plugins.ListByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list plugins")
			return
		}

		if list == nil {
			list = []*db.Plugin{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"plugins": list,
		})
	}
}

func requestPluginActionHandler(plugins *db.PluginRepository, devices *db.DeviceRepository, audits *db.AuditRepository, hub *ws.Hub, enforcer *policy.Enforcer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		pluginName := chi.URLParam(r, "name")
		action := chi.URLParam(r, "action")
		if deviceID == "" || pluginName == "" || action == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id, plugin name, and action required")
			return
		}

		if action != "run" && action != "enable" && action != "disable" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "action must be run, enable, or disable")
			return
		}

		if _, err := devices.FindByID(r.Context(), deviceID); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Device not found")
			return
		}

		claims := r.Context().Value(claimsKey).(*auth.Claims)

		if r := enforcer.Evaluate(r.Context(), &policy.EvalContext{
			ActionType: "plugin_action",
			DeviceIDs:  []string{deviceID},
		}); r.Action == "deny" {
			writeError(w, http.StatusForbidden, "POLICY_DENIED", r.Message)
			return
		}
		audits.Create(r.Context(), "user", claims.UserID, "plugin_action."+action, "device", deviceID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":     "plugin action queued",
			"device_id":   deviceID,
			"plugin_name": pluginName,
			"action":      action,
		})
	}
}
