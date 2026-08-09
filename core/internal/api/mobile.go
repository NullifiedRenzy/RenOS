package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/renos-project/renos/core/internal/db"
)

func registerPushHandler(pushRepos *db.PushRegistrationRepository) http.HandlerFunc {
	type request struct {
		Platform    string `json:"platform"`
		Token       string `json:"token"`
		DeviceName  string `json:"device_name"`
		DeviceModel string `json:"device_model"`
		OSVersion   string `json:"os_version"`
		AppVersion  string `json:"app_version"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid body")
			return
		}
		if req.Platform == "" || req.Token == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "platform and token required")
			return
		}

		reg := &db.PushRegistration{
			UserID:      userID,
			Platform:    req.Platform,
			Token:       req.Token,
			DeviceName:  req.DeviceName,
			DeviceModel: req.DeviceModel,
			OSVersion:   req.OSVersion,
			AppVersion:  req.AppVersion,
			Enabled:     true,
		}
		if err := pushRepos.Register(r.Context(), reg); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"status": "registered"})
	}
}

func listPushRegistrationsHandler(pushRepos *db.PushRegistrationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		list, err := pushRepos.ListByUser(r.Context(), userID)
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", err.Error()); return }
		if list == nil { list = []*db.PushRegistration{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func deletePushRegistrationHandler(pushRepos *db.PushRegistrationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" { writeError(w, http.StatusBadRequest, "invalid_request", "id required"); return }
		if err := pushRepos.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func mobileDashboardHandler(devices *db.DeviceRepository, alerts *db.AlertInstanceRepository, nodes *db.ClusterNodeRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		_ = userID

		deviceList, _ := devices.FindAll(r.Context())
		onlineCount := 0
		for _, d := range deviceList {
			if d.Status == "online" { onlineCount++ }
		}

		summary := map[string]interface{}{
			"total_devices":  len(deviceList),
			"online_devices": onlineCount,
			"offline_devices": len(deviceList) - onlineCount,
			"updated_at":     time.Now().UTC(),
		}
		writeJSON(w, http.StatusOK, summary)
	}
}

func mobileDeviceListHandler(devices *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceList, err := devices.FindAll(r.Context())
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", err.Error()); return }
		if deviceList == nil { deviceList = []*db.Device{} }
		writeJSON(w, http.StatusOK, deviceList)
	}
}

func mobileAlertsSummaryHandler(alerts *db.AlertInstanceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		active, err := alerts.ListActive(r.Context())
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", err.Error()); return }
		if active == nil { active = []*db.AlertInstance{} }
		writeJSON(w, http.StatusOK, map[string]interface{}{"active_alerts": active, "count": len(active)})
	}
}

func mobileOfflineSyncHandler(offlineSync *db.OfflineSyncRepository) http.HandlerFunc {
	type enqueueRequest struct {
		DeviceID string          `json:"device_id"`
		Action   string          `json:"action"`
		Payload  json.RawMessage `json:"payload"`
		Priority int             `json:"priority"`
	}
	type completeRequest struct {
		ItemID string `json:"item_id"`
		Error  string `json:"error"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		switch r.Method {
		case "GET":
			deviceID := r.URL.Query().Get("device_id")
			if deviceID == "" { writeError(w, http.StatusBadRequest, "invalid_request", "device_id required"); return }
			items, err := offlineSync.ListPending(r.Context(), deviceID, 50)
			if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", err.Error()); return }
			if items == nil { items = []*db.OfflineSyncItem{} }
			writeJSON(w, http.StatusOK, items)
		case "POST":
			var req enqueueRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid_request", "invalid body")
				return
			}
			item := &db.OfflineSyncItem{
				UserID:   userID,
				DeviceID: req.DeviceID,
				Action:   req.Action,
				Payload:  req.Payload,
				Priority: req.Priority,
				Status:   "pending",
				MaxRetries: 3,
			}
			if item.Payload == nil { item.Payload = []byte(`{}`) }
			if err := offlineSync.Enqueue(r.Context(), item); err != nil {
				writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, item)
		case "PUT":
			var req completeRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.Error != "" {
				offlineSync.Fail(r.Context(), req.ItemID, req.Error)
			} else {
				offlineSync.Complete(r.Context(), req.ItemID)
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
		}
	}
}

func listDeviceApprovalsHandler(approvals *db.DeviceApprovalRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		status := r.URL.Query().Get("status")
		var list []*db.DeviceApproval
		var err error
		if status == "pending" {
			list, err = approvals.ListPending(r.Context())
		} else {
			list, err = approvals.ListByUser(r.Context(), userID)
		}
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", err.Error()); return }
		if list == nil { list = []*db.DeviceApproval{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func requestDeviceApprovalHandler(approvals *db.DeviceApprovalRepository) http.HandlerFunc {
	type request struct {
		DeviceID    string `json:"device_id"`
		DeviceName  string `json:"device_name"`
		DeviceModel string `json:"device_model"`
		IPAddress   string `json:"ip_address"`
		Fingerprint string `json:"fingerprint"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid body")
			return
		}
		d := &db.DeviceApproval{
			UserID:      userID,
			DeviceID:    req.DeviceID,
			DeviceName:  req.DeviceName,
			DeviceModel: req.DeviceModel,
			IPAddress:   req.IPAddress,
			Fingerprint: req.Fingerprint,
			Status:      "pending",
		}
		if err := approvals.Request(r.Context(), d); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, d)
	}
}

func approveDeviceHandler(approvals *db.DeviceApprovalRepository) http.HandlerFunc {
	type request struct {
		Reason string `json:"reason"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		approvalID := r.URL.Query().Get("id")
		userID := getUserID(r)
		if approvalID == "" { writeError(w, http.StatusBadRequest, "invalid_request", "id required"); return }
		if err := approvals.Approve(r.Context(), approvalID, userID); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
	}
}

func rejectDeviceHandler(approvals *db.DeviceApprovalRepository) http.HandlerFunc {
	type request struct {
		Reason string `json:"reason"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		approvalID := r.URL.Query().Get("id")
		var req request
		json.NewDecoder(r.Body).Decode(&req)
		if approvalID == "" { writeError(w, http.StatusBadRequest, "invalid_request", "id required"); return }
		if err := approvals.Reject(r.Context(), approvalID, req.Reason); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
	}
}

func mobileSessionsHandler(sessions *db.MobileSessionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		list, err := sessions.ListByUser(r.Context(), userID)
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", err.Error()); return }
		if list == nil { list = []*db.MobileSession{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func deleteMobileSessionHandler(sessions *db.MobileSessionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" { writeError(w, http.StatusBadRequest, "invalid_request", "id required"); return }
		if err := sessions.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func mobileSummaryHandler(summaries *db.MobileSummaryRepository, devices *db.DeviceRepository, alerts *db.AlertInstanceRepository, nodes *db.ClusterNodeRepository, clusterMetrics *db.ClusterMetricsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		summaryType := r.URL.Query().Get("type")
		if summaryType == "" { summaryType = "overview" }

		var data map[string]interface{}
		switch summaryType {
		case "overview":
			deviceList, _ := devices.FindAll(r.Context())
			online := 0
			for _, d := range deviceList { if d.Status == "online" { online++ } }
			data = map[string]interface{}{
				"total_devices":  len(deviceList),
				"online_devices": online,
			}
		case "devices":
			deviceList, _ := devices.FindAll(r.Context())
			data = map[string]interface{}{"devices": deviceList}
		case "alerts":
			active, _ := alerts.ListActive(r.Context())
			data = map[string]interface{}{"active_alerts": len(active)}
		case "cluster":
			clusterNodes, _ := nodes.ListAll(r.Context())
			leader, _ := nodes.FindLeader(r.Context())
			leaderID := ""
			if leader != nil { leaderID = leader.NodeID }
			metrics, _ := clusterMetrics.LatestByNode(r.Context())
			data = map[string]interface{}{"nodes": clusterNodes, "leader_id": leaderID, "metrics": metrics}
		default:
			data = map[string]interface{}{}
		}

		jsonData, _ := json.Marshal(data)
		s := &db.MobileSummary{UserID: userID, SummaryType: summaryType, Data: jsonData, Version: 1}
		summaries.Upsert(r.Context(), s)
		writeJSON(w, http.StatusOK, data)
	}
}
