package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/ws"
)

type createGroupPayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func createGroupHandler(groups *db.GroupRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)

		var req createGroupPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required")
			return
		}

		group, err := groups.Create(r.Context(), req.Name, req.Description, claims.UserID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create group")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "group.created", "group", group.ID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"group": group,
		})
	}
}

func listGroupsHandler(groups *db.GroupRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)

		list, err := groups.FindAll(r.Context(), parseListLimit(r.URL.Query().Get("limit")), parseOffset(r.URL.Query().Get("offset")))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list groups")
			return
		}
		if list == nil {
			list = []*db.Group{}
		}

		audits.Create(r.Context(), "user", claims.UserID, "group.list", "group", "")

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"groups": list,
		})
	}
}

func getGroupHandler(groups *db.GroupRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		id := chi.URLParam(r, "id")

		group, err := groups.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Group not found")
			return
		}

		deviceIDs, err := groups.ListDeviceIDs(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list group devices")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "group.get", "group", id)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"group":      group,
			"device_ids": deviceIDs,
		})
	}
}

type updateGroupPayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func updateGroupHandler(groups *db.GroupRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		id := chi.URLParam(r, "id")

		var req updateGroupPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required")
			return
		}

		group, err := groups.Update(r.Context(), id, req.Name, req.Description)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Group not found")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "group.updated", "group", id)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"group": group,
		})
	}
}

func deleteGroupHandler(groups *db.GroupRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		id := chi.URLParam(r, "id")

		if err := groups.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Group not found")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "group.deleted", "group", id)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	}
}

type addDevicePayload struct {
	DeviceID string `json:"device_id"`
}

func addDeviceToGroupHandler(groups *db.GroupRepository, devices *db.DeviceRepository, audits *db.AuditRepository, hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		id := chi.URLParam(r, "id")

		var req addDevicePayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.DeviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device_id is required")
			return
		}

		if _, err := devices.FindByID(r.Context(), req.DeviceID); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Device not found")
			return
		}

		if err := groups.AddDevice(r.Context(), id, req.DeviceID); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to add device to group")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "group.device_added", "group", id)
		hub.Broadcast(ws.Message{
			Type:     "group.device_added",
			DeviceID: req.DeviceID,
			Data: map[string]interface{}{
				"group_id":  id,
				"device_id": req.DeviceID,
			},
		})

		writeJSON(w, http.StatusOK, map[string]string{"status": "added"})
	}
}

func removeDeviceFromGroupHandler(groups *db.GroupRepository, audits *db.AuditRepository, hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		id := chi.URLParam(r, "id")
		deviceID := chi.URLParam(r, "deviceId")

		if err := groups.RemoveDevice(r.Context(), id, deviceID); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Group or device not found")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "group.device_removed", "group", id)
		hub.Broadcast(ws.Message{
			Type:     "group.device_removed",
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"group_id":  id,
				"device_id": deviceID,
			},
		})

		writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
	}
}
