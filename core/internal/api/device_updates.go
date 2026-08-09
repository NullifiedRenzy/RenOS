package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/ws"
	"github.com/renos-project/renos/core/internal/policy"
)

func syncUpdatesHandler(updates *db.UpdateStatusRepository, devices *db.DeviceRepository) http.HandlerFunc {
	type request struct {
		Updates []struct {
			UpdateType  string `json:"update_type"`
			PackageName string `json:"package_name"`
			CurrentVer  string `json:"current_version"`
			AvailableVer string `json:"available_version"`
		} `json:"updates"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		for _, u := range req.Updates {
			update := &db.DeviceUpdate{
				DeviceID:     deviceID,
				UpdateType:   u.UpdateType,
				PackageName:  u.PackageName,
				CurrentVer:   u.CurrentVer,
				AvailableVer: u.AvailableVer,
				Status:       "available",
			}
			updates.Upsert(r.Context(), update)
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "synced"})
	}
}

func listUpdatesHandler(updates *db.UpdateStatusRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		list, err := updates.ListByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list updates")
			return
		}
		if list == nil { list = []*db.DeviceUpdate{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func requestUpdateActionHandler(updates *db.UpdateStatusRepository, devices *db.DeviceRepository, audits *db.AuditRepository, hub *ws.Hub, enforcer *policy.Enforcer) http.HandlerFunc {
	type request struct {
		PackageName string `json:"package_name"`
		Action      string `json:"action"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		action := chi.URLParam(r, "action")
		var req request
		json.NewDecoder(r.Body).Decode(&req)

		uid := getUserID(r)
		audits.Create(r.Context(), "user", uid, "device.update."+action, "device", deviceID)

		hub.Broadcast(ws.Message{
			Type:     "device.update." + action,
			DeviceID: deviceID,
		})

		writeJSON(w, http.StatusAccepted, map[string]string{"status": "requested"})
	}
}
