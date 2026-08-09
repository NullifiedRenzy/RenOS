package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/ws"
)

func getUserID(r *http.Request) string {
	if claims, ok := r.Context().Value(claimsKey).(*auth.Claims); ok && claims != nil {
		return claims.UserID
	}
	return ""
}

// --- Tags ---

func listDeviceTagsHandler(tags *db.DeviceTagRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "deviceId")
		deviceTags, err := tags.ListByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list tags: "+err.Error())
			return
		}
		if deviceTags == nil {
			deviceTags = []*db.DeviceTag{}
		}
		writeJSON(w, http.StatusOK, deviceTags)
	}
}

func setDeviceTagHandler(tags *db.DeviceTagRepository, audits *db.AuditRepository) http.HandlerFunc {
	type request struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "deviceId")
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if req.Key == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "tag key is required")
			return
		}
		if err := tags.Set(r.Context(), deviceID, req.Key, req.Value); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to set tag: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "device.tag.set", "device", deviceID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func removeDeviceTagHandler(tags *db.DeviceTagRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "deviceId")
		key := chi.URLParam(r, "tagKey")
		if err := tags.Remove(r.Context(), deviceID, key); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to remove tag: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "device.tag.remove", "device", deviceID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func listTagKeysHandler(tags *db.DeviceTagRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keys, err := tags.ListAllKeys(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list tag keys: "+err.Error())
			return
		}
		if keys == nil {
			keys = []string{}
		}
		writeJSON(w, http.StatusOK, keys)
	}
}

func findDevicesByTagHandler(tags *db.DeviceTagRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		value := r.URL.Query().Get("value")
		if key == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "key parameter is required")
			return
		}
		deviceIDs, err := tags.FindByTag(r.Context(), key, value)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to find devices: "+err.Error())
			return
		}
		if deviceIDs == nil {
			deviceIDs = []string{}
		}
		writeJSON(w, http.StatusOK, deviceIDs)
	}
}

// --- Inventory ---

func getDeviceInventoryHandler(inventory *db.DeviceInventoryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "deviceId")
		inv, err := inventory.FindByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "inventory not found")
			return
		}
		writeJSON(w, http.StatusOK, inv)
	}
}

func upsertDeviceInventoryHandler(inventory *db.DeviceInventoryRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "deviceId")
		var inv db.DeviceInventory
		if err := json.NewDecoder(r.Body).Decode(&inv); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		inv.DeviceID = deviceID
		now := time.Now()
		inv.LastCollectedAt = &now
		if err := inventory.Upsert(r.Context(), &inv); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to upsert inventory: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "device.inventory.update", "device", deviceID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func listDeviceInventoryHandler(inventory *db.DeviceInventoryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := inventory.ListAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list inventory: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.DeviceInventory{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

// --- Compliance ---

func listComplianceChecksHandler(compliance *db.ComplianceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks, err := compliance.ListChecks(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list checks: "+err.Error())
			return
		}
		if checks == nil {
			checks = []*db.ComplianceCheck{}
		}
		writeJSON(w, http.StatusOK, checks)
	}
}

func createComplianceCheckHandler(compliance *db.ComplianceRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var check db.ComplianceCheck
		if err := json.NewDecoder(r.Body).Decode(&check); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if check.Name == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "name is required")
			return
		}
		if check.CheckType == "" {
			check.CheckType = "manual"
		}
		if check.Severity == "" {
			check.Severity = "warning"
		}
		check.Enabled = true
		if err := compliance.CreateCheck(r.Context(), &check); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create check: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "compliance.check.create", "check", check.ID)
		writeJSON(w, http.StatusCreated, check)
	}
}

func listComplianceResultsHandler(compliance *db.ComplianceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := r.URL.Query().Get("device_id")
		var results []*db.ComplianceResult
		var err error
		if deviceID != "" {
			results, err = compliance.ListResultsByDevice(r.Context(), deviceID)
		} else {
			results, err = compliance.ListResults(r.Context())
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list results: "+err.Error())
			return
		}
		if results == nil {
			results = []*db.ComplianceResult{}
		}
		writeJSON(w, http.StatusOK, results)
	}
}

func upsertComplianceResultHandler(compliance *db.ComplianceRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var res db.ComplianceResult
		if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if res.CheckID == "" || res.DeviceID == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "check_id and device_id are required")
			return
		}
		now := time.Now()
		res.LastCheckedAt = &now
		if err := compliance.UpsertResult(r.Context(), &res); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to upsert result: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "compliance.result.update", "check", res.CheckID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func complianceSummaryHandler(compliance *db.ComplianceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summary, err := compliance.GetSummary(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to get summary: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, summary)
	}
}

// --- Bulk Operations ---

func createBulkOperationHandler(bulkOps *db.BulkOperationRepository, audits *db.AuditRepository, hub *ws.Hub) http.HandlerFunc {
	type request struct {
		Name       string          `json:"name"`
		Action     string          `json:"action"`
		TargetType string          `json:"target_type"`
		TargetIDs  json.RawMessage `json:"target_ids"`
		Config     json.RawMessage `json:"config"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if req.Action == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "action is required")
			return
		}
		if req.TargetType == "" {
			req.TargetType = "devices"
		}

		var ids []string
		if err := json.Unmarshal(req.TargetIDs, &ids); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid target_ids")
			return
		}

		op := &db.BulkOperation{
			Name:       req.Name,
			Action:     req.Action,
			TargetType: req.TargetType,
			TargetIDs:  req.TargetIDs,
			Config:     req.Config,
			TotalCount: len(ids),
		}
		uid := getUserID(r)
		op.CreatedBy = &uid

		if err := bulkOps.Create(r.Context(), op); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create bulk operation: "+err.Error())
			return
		}

		audits.Create(r.Context(), "user", uid, "bulk.create", "operation", op.ID)

		go executeBulkOperation(r.Context(), op, ids, bulkOps, hub)

		writeJSON(w, http.StatusCreated, op)
	}
}

func executeBulkOperation(ctx context.Context, op *db.BulkOperation, deviceIDs []string, bulkOps *db.BulkOperationRepository, hub *ws.Hub) {
	bulkOps.UpdateStatus(ctx, op.ID, "running")

	for _, deviceID := range deviceIDs {
		msg := ws.Message{
			Type:     "bulk.action",
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"bulk_id": op.ID,
				"action":  op.Action,
				"config":  op.Config,
			},
		}
		hub.Broadcast(msg)
		bulkOps.IncrementProgress(ctx, op.ID, true, false)
	}

	bulkOps.UpdateStatus(ctx, op.ID, "completed")
	hub.Broadcast(ws.Message{
		Type: "bulk.complete",
		Data: map[string]interface{}{"bulk_id": op.ID},
	})
}

func listBulkOperationsHandler(bulkOps *db.BulkOperationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ops, err := bulkOps.ListAll(r.Context(), 50)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list operations: "+err.Error())
			return
		}
		if ops == nil {
			ops = []*db.BulkOperation{}
		}
		writeJSON(w, http.StatusOK, ops)
	}
}

func getBulkOperationHandler(bulkOps *db.BulkOperationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		op, err := bulkOps.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "operation not found")
			return
		}
		writeJSON(w, http.StatusOK, op)
	}
}

func cancelBulkOperationHandler(bulkOps *db.BulkOperationRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := bulkOps.UpdateStatus(r.Context(), id, "cancelled"); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to cancel operation: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "bulk.cancel", "operation", id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
	}
}

// --- Fleet Dashboard ---

func fleetDashboardHandler(devices *db.DeviceRepository, inventory *db.DeviceInventoryRepository, compliance *db.ComplianceRepository, groups *db.GroupRepository, tags *db.DeviceTagRepository) http.HandlerFunc {
	type dashboard struct {
		TotalDevices      int64          `json:"total_devices"`
		OnlineDevices     int64          `json:"online_devices"`
		OfflineDevices    int64          `json:"offline_devices"`
		TotalGroups       int64          `json:"total_groups"`
		TotalTags         int64          `json:"total_tags"`
		CompliancePass    int64          `json:"compliance_pass"`
		ComplianceFail    int64          `json:"compliance_fail"`
		CompliancePending int64          `json:"compliance_pending"`
		InventoryCount    int64          `json:"inventory_count"`
		OSDistribution    map[string]int64 `json:"os_distribution"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		dc, err := devices.CountByStatus(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list devices")
			return
		}

		groupCount, _ := groups.Count(ctx)
		tagCount, _ := tags.CountDistinctKeys(ctx)
		invCount, _ := inventory.Count(ctx)

		summary, _ := compliance.GetSummary(ctx)

		dash := dashboard{
			OSDistribution:    dc.OSDist,
			TotalDevices:      dc.Total,
			OnlineDevices:     dc.Online,
			OfflineDevices:    dc.Total - dc.Online,
			TotalGroups:       groupCount,
			TotalTags:         tagCount,
			InventoryCount:    invCount,
			CompliancePass:    int64(summary["pass"]),
			ComplianceFail:    int64(summary["fail"]),
			CompliancePending: int64(summary["pending"]),
		}

		writeJSON(w, http.StatusOK, dash)
	}
}
