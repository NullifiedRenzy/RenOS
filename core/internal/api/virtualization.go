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

type syncVMPayload struct {
	DomainName        string `json:"domain_name"`
	UUID              string `json:"uuid"`
	State             string `json:"state"`
	VCPUMax           int32  `json:"vcpu_max"`
	VCPUCurrent       int32  `json:"vcpu_current"`
	MemoryMaxBytes    int64  `json:"memory_max_bytes"`
	MemoryCurrentBytes int64 `json:"memory_current_bytes"`
	OSType            string `json:"os_type"`
	Autostart         bool   `json:"autostart"`
	Persistent        bool   `json:"persistent"`
}

func syncVirtualMachinesHandler(vms *db.VirtualMachineRepository, devices *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		var req struct {
			DeviceID     string          `json:"device_id"`
			DeviceSecret string          `json:"device_secret"`
			VMs          []syncVMPayload  `json:"vms"`
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

		var records []*db.VirtualMachine
		for _, vm := range req.VMs {
			records = append(records, &db.VirtualMachine{
				DomainName:        vm.DomainName,
				UUID:              vm.UUID,
				State:             vm.State,
				VCPUMax:           vm.VCPUMax,
				VCPUCurrent:       vm.VCPUCurrent,
				MemoryMaxBytes:    vm.MemoryMaxBytes,
				MemoryCurrentBytes: vm.MemoryCurrentBytes,
				OSType:            vm.OSType,
				Autostart:         vm.Autostart,
				Persistent:        vm.Persistent,
			})
		}

		if err := vms.ReplaceForDevice(r.Context(), deviceID, records); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync virtual machines")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "synced"})
	}
}

func listVirtualMachinesHandler(vms *db.VirtualMachineRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		list, err := vms.ListByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list virtual machines")
			return
		}

		if list == nil {
			list = []*db.VirtualMachine{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"vms": list,
		})
	}
}

func requestVMActionHandler(vms *db.VirtualMachineRepository, devices *db.DeviceRepository, audits *db.AuditRepository, hub *ws.Hub, enforcer *policy.Enforcer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		domainName := chi.URLParam(r, "domainName")
		action := chi.URLParam(r, "action")
		if deviceID == "" || domainName == "" || action == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id, domain name, and action required")
			return
		}

		if action != "start" && action != "shutdown" && action != "destroy" && action != "reset" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "action must be start, shutdown, destroy, or reset")
			return
		}

		if _, err := devices.FindByID(r.Context(), deviceID); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Device not found")
			return
		}

		claims := r.Context().Value(claimsKey).(*auth.Claims)

		if r := enforcer.Evaluate(r.Context(), &policy.EvalContext{
			ActionType: "vm_action",
			DeviceIDs:  []string{deviceID},
		}); r.Action == "deny" {
			writeError(w, http.StatusForbidden, "POLICY_DENIED", r.Message)
			return
		}
		audits.Create(r.Context(), "user", claims.UserID, "vm_action."+action, "device", deviceID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":     "vm action queued",
			"device_id":   deviceID,
			"domain_name": domainName,
			"action":      action,
		})
	}
}
