package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/db"
)

type syncNetworkInterfacePayload struct {
	Name             string   `json:"name"`
	MacAddress       string   `json:"mac_address"`
	IPv4             []string `json:"ipv4"`
	IPv6             []string `json:"ipv6"`
	Status           string   `json:"status"`
	Speed            int64    `json:"speed"`
	InterfaceType    string   `json:"interface_type"`
	IsDefaultGateway bool     `json:"is_default_gateway"`
	DNSServers       []string `json:"dns_servers"`
	MTU              int32    `json:"mtu"`
}

func syncNetworkInterfacesHandler(interfaces *db.NetworkInterfaceRepository, devices *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		var req struct {
			DeviceID     string                         `json:"device_id"`
			DeviceSecret string                         `json:"device_secret"`
			Interfaces   []syncNetworkInterfacePayload   `json:"interfaces"`
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

		var records []*db.NetworkInterface
		for _, iface := range req.Interfaces {
			records = append(records, &db.NetworkInterface{
				Name:             iface.Name,
				MacAddress:       iface.MacAddress,
				IPv4:             iface.IPv4,
				IPv6:             iface.IPv6,
				Status:           iface.Status,
				Speed:            iface.Speed,
				InterfaceType:    iface.InterfaceType,
				IsDefaultGateway: iface.IsDefaultGateway,
				DNSServers:       iface.DNSServers,
				MTU:              iface.MTU,
			})
		}

		if err := interfaces.ReplaceForDevice(r.Context(), deviceID, records); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to sync network interfaces")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "synced"})
	}
}

func listNetworkInterfacesHandler(interfaces *db.NetworkInterfaceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		list, err := interfaces.ListByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list network interfaces")
			return
		}

		if list == nil {
			list = []*db.NetworkInterface{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"interfaces": list,
		})
	}
}
