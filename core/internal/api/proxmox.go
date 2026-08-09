package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/ws"
)

type proxmoxSyncRequest struct {
	DeviceID     string               `json:"device_id"`
	DeviceSecret string               `json:"device_secret"`
	Nodes        []*db.ProxmoxNode    `json:"nodes"`
	VMs          []*db.ProxmoxVM      `json:"vms"`
	LXCs         []*db.ProxmoxLXC     `json:"lxcs"`
	Storage      []*db.ProxmoxStorage `json:"storage"`
	Networks     []*db.ProxmoxNetwork `json:"networks"`
	Snapshots    []*db.ProxmoxSnapshot `json:"snapshots"`
	Tasks        []*db.ProxmoxTask    `json:"tasks"`
}

func syncProxmoxHandler(proxmox *db.ProxmoxRepository, devices *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		var req proxmoxSyncRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		if _, apiErr := verifyDeviceSecret(devices, deviceAuthRequest{DeviceID: deviceID, DeviceSecret: req.DeviceSecret}); apiErr != nil {
			writeError(w, apiErr.status, apiErr.code, apiErr.message)
			return
		}

		// Find the connection for this device
		conns, err := proxmox.ListConnectionsByDevice(r.Context(), deviceID)
		if err != nil || len(conns) == 0 {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "No proxmox connection found for device")
			return
		}
		conn := conns[0]

		if err := proxmox.ReplaceNodes(r.Context(), conn.ID, req.Nodes); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync proxmox nodes")
			return
		}
		if err := proxmox.ReplaceVMs(r.Context(), conn.ID, req.VMs); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync proxmox vms")
			return
		}
		if err := proxmox.ReplaceLXCs(r.Context(), conn.ID, req.LXCs); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync proxmox lxcs")
			return
		}
		if err := proxmox.ReplaceStorage(r.Context(), conn.ID, req.Storage); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync proxmox storage")
			return
		}
		if err := proxmox.ReplaceNetworks(r.Context(), conn.ID, req.Networks); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync proxmox networks")
			return
		}
		if err := proxmox.ReplaceSnapshots(r.Context(), conn.ID, req.Snapshots); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync proxmox snapshots")
			return
		}
		if err := proxmox.ReplaceTasks(r.Context(), conn.ID, req.Tasks); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync proxmox tasks")
			return
		}

		proxmox.UpdateConnectionStatus(r.Context(), conn.ID, "connected", "")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"synced_nodes":     len(req.Nodes),
			"synced_vms":       len(req.VMs),
			"synced_lxcs":      len(req.LXCs),
			"synced_storage":   len(req.Storage),
			"synced_networks":  len(req.Networks),
			"synced_snapshots": len(req.Snapshots),
			"synced_tasks":     len(req.Tasks),
		})
	}
}

// --- Connection management ---

func createProxmoxConnectionHandler(proxmox *db.ProxmoxRepository, devices *db.DeviceRepository, audits *db.AuditRepository) http.HandlerFunc {
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

		claims := r.Context().Value(claimsKey).(*auth.Claims)

		var req struct {
			Name      string `json:"name"`
			Host      string `json:"host"`
			Port      int    `json:"port"`
			AuthType  string `json:"auth_type"`
			TokenID   string `json:"token_id"`
			TokenSecret string `json:"token_secret"`
			Username  string `json:"username"`
			Password  string `json:"password"`
			VerifySSL bool   `json:"verify_ssl"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		if req.Host == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "host is required")
			return
		}
		if req.Port == 0 {
			req.Port = 8006
		}
		if req.AuthType == "" {
			req.AuthType = "token"
		}

		conn, err := proxmox.CreateConnection(r.Context(), deviceID, req.Name, req.Host, req.Port, req.AuthType, req.TokenID, req.TokenSecret, req.Username, req.Password, req.VerifySSL)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create proxmox connection")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "proxmox_connection.create", "device", deviceID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"connection": conn,
		})
	}
}

func listProxmoxConnectionsHandler(proxmox *db.ProxmoxRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		list, err := proxmox.ListConnectionsByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list proxmox connections")
			return
		}
		if list == nil {
			list = []*db.ProxmoxConnection{}
		}

		// Strip credentials from response
		for _, c := range list {
			c.TokenID = ""
			c.TokenSecret = ""
			c.Password = ""
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"connections": list,
		})
	}
}

func deleteProxmoxConnectionHandler(proxmox *db.ProxmoxRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		connID := chi.URLParam(r, "connectionId")
		if connID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "connection id required")
			return
		}

		claims := r.Context().Value(claimsKey).(*auth.Claims)

		if err := proxmox.DeleteConnection(r.Context(), connID); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to delete proxmox connection")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "proxmox_connection.delete", "proxmox_connection", connID)

		w.WriteHeader(http.StatusNoContent)
	}
}

// --- User-facing resource listing ---

func listProxmoxNodesHandler(proxmox *db.ProxmoxRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		connID := r.URL.Query().Get("connection_id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		conns, err := proxmox.ListConnectionsByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list connections")
			return
		}

		var allNodes []*db.ProxmoxNode
		for _, c := range conns {
			if connID != "" && c.ID != connID {
				continue
			}
			nodes, err := proxmox.ListNodes(r.Context(), c.ID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list nodes")
				return
			}
			allNodes = append(allNodes, nodes...)
		}
		if allNodes == nil {
			allNodes = []*db.ProxmoxNode{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"nodes": allNodes,
		})
	}
}

func listProxmoxVMsHandler(proxmox *db.ProxmoxRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		connID := r.URL.Query().Get("connection_id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		conns, err := proxmox.ListConnectionsByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list connections")
			return
		}

		var allVMs []*db.ProxmoxVM
		for _, c := range conns {
			if connID != "" && c.ID != connID {
				continue
			}
			vms, err := proxmox.ListVMs(r.Context(), c.ID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list vms")
				return
			}
			allVMs = append(allVMs, vms...)
		}
		if allVMs == nil {
			allVMs = []*db.ProxmoxVM{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"vms": allVMs,
		})
	}
}

func listProxmoxLXCsHandler(proxmox *db.ProxmoxRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		connID := r.URL.Query().Get("connection_id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		conns, err := proxmox.ListConnectionsByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list connections")
			return
		}

		var allLXCs []*db.ProxmoxLXC
		for _, c := range conns {
			if connID != "" && c.ID != connID {
				continue
			}
			lxcs, err := proxmox.ListLXCs(r.Context(), c.ID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list lxcs")
				return
			}
			allLXCs = append(allLXCs, lxcs...)
		}
		if allLXCs == nil {
			allLXCs = []*db.ProxmoxLXC{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"lxcs": allLXCs,
		})
	}
}

func listProxmoxStorageHandler(proxmox *db.ProxmoxRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		connID := r.URL.Query().Get("connection_id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		conns, err := proxmox.ListConnectionsByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list connections")
			return
		}

		var allStorage []*db.ProxmoxStorage
		for _, c := range conns {
			if connID != "" && c.ID != connID {
				continue
			}
			items, err := proxmox.ListStorage(r.Context(), c.ID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list storage")
				return
			}
			allStorage = append(allStorage, items...)
		}
		if allStorage == nil {
			allStorage = []*db.ProxmoxStorage{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"storage": allStorage,
		})
	}
}

func listProxmoxNetworksHandler(proxmox *db.ProxmoxRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		connID := r.URL.Query().Get("connection_id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		conns, err := proxmox.ListConnectionsByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list connections")
			return
		}

		var allNets []*db.ProxmoxNetwork
		for _, c := range conns {
			if connID != "" && c.ID != connID {
				continue
			}
			nets, err := proxmox.ListNetworks(r.Context(), c.ID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list networks")
				return
			}
			allNets = append(allNets, nets...)
		}
		if allNets == nil {
			allNets = []*db.ProxmoxNetwork{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"networks": allNets,
		})
	}
}

func listProxmoxSnapshotsHandler(proxmox *db.ProxmoxRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		connID := r.URL.Query().Get("connection_id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		conns, err := proxmox.ListConnectionsByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list connections")
			return
		}

		var allSnaps []*db.ProxmoxSnapshot
		for _, c := range conns {
			if connID != "" && c.ID != connID {
				continue
			}
			snaps, err := proxmox.ListSnapshots(r.Context(), c.ID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list snapshots")
				return
			}
			allSnaps = append(allSnaps, snaps...)
		}
		if allSnaps == nil {
			allSnaps = []*db.ProxmoxSnapshot{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"snapshots": allSnaps,
		})
	}
}

func listProxmoxTasksHandler(proxmox *db.ProxmoxRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		connID := r.URL.Query().Get("connection_id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		limit := 50
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}

		conns, err := proxmox.ListConnectionsByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list connections")
			return
		}

		var allTasks []*db.ProxmoxTask
		for _, c := range conns {
			if connID != "" && c.ID != connID {
				continue
			}
			tasks, err := proxmox.ListTasks(r.Context(), c.ID, limit)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list tasks")
				return
			}
			allTasks = append(allTasks, tasks...)
		}
		if allTasks == nil {
			allTasks = []*db.ProxmoxTask{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tasks": allTasks,
		})
	}
}

// --- Power actions via NATS ---

func proxmoxPowerActionHandler(
	proxmox *db.ProxmoxRepository,
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
			NodeName string `json:"node_name"`
			VMID     int    `json:"vm_id"`
			Action   string `json:"action"`
			Type     string `json:"type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		validActions := map[string]bool{"start": true, "stop": true, "reboot": true, "shutdown": true}
		if !validActions[req.Action] {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "action must be start, stop, reboot, or shutdown")
			return
		}
		validTypes := map[string]bool{"qemu": true, "lxc": true}
		if !validTypes[req.Type] {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "type must be qemu or lxc")
			return
		}
		if req.NodeName == "" || req.VMID <= 0 {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "node_name and vm_id are required")
			return
		}

		// Publish power action to agent via NATS
		subject := "proxmox.power." + deviceID
		msg := map[string]interface{}{
			"action":    req.Action,
			"type":      req.Type,
			"node_name": req.NodeName,
			"vm_id":     req.VMID,
		}
		if err := nc.Publish(subject, mustMarshal(msg)); err != nil {
			writeError(w, http.StatusServiceUnavailable, "NATS_ERROR", "Failed to dispatch action to agent")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "proxmox.power."+req.Action, "device", deviceID)
		hub.Broadcast(ws.Message{
			Type:     "proxmox.power.requested",
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"action":    req.Action,
				"type":      req.Type,
				"node_name": req.NodeName,
				"vm_id":     req.VMID,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "dispatched",
		})
	}
}
