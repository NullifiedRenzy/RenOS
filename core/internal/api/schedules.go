package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorhill/cronexpr"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/scheduler"
)

const maxScheduleNameLen = 255

type createSchedulePayload struct {
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload"`
	TargetType     string          `json:"target_type"`
	TargetIDs      []string        `json:"target_ids"`
	CronExpression string          `json:"cron_expression"`
}

func createScheduleHandler(scheds *db.ScheduleRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)

		var req createSchedulePayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		if req.Name == "" || len(req.Name) > maxScheduleNameLen {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required and must be under 256 characters")
			return
		}
		if req.Type == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "type is required")
			return
		}
		validTypes := map[string]bool{"command": true, "container_action": true, "service_action": true, "update": true}
		if !validTypes[req.Type] {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "type must be command, container_action, service_action, or update")
			return
		}
		if req.TargetType == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "target_type is required")
			return
		}
		validTargets := map[string]bool{"device": true, "group": true, "multi": true}
		if !validTargets[req.TargetType] {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "target_type must be device, group, or multi")
			return
		}
		if len(req.TargetIDs) == 0 {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "at least one target_id is required")
			return
		}
		if req.CronExpression == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "cron_expression is required")
			return
		}
		if err := scheduler.ValidateCronExpression(req.CronExpression); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_CRON", "Invalid cron expression: "+err.Error())
			return
		}

		targetIDsBytes, _ := json.Marshal(req.TargetIDs)
		now := time.Now().UTC()
		nextRun := cronexpr.MustParse(req.CronExpression).Next(now)

		if req.Payload == nil {
			req.Payload = json.RawMessage("{}")
		}

		s, err := scheds.Create(r.Context(), req.Name, req.Type, req.Payload, req.TargetType, targetIDsBytes, req.CronExpression, nextRun, claims.UserID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create schedule")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "schedule.created", "schedule", s.ID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"schedule": s,
		})
	}
}

func listSchedulesHandler(scheds *db.ScheduleRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := scheds.FindAll(r.Context(), parseListLimit(r.URL.Query().Get("limit")), parseOffset(r.URL.Query().Get("offset")))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list schedules")
			return
		}
		if list == nil {
			list = []*db.Schedule{}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"schedules": list,
		})
	}
}

func getScheduleHandler(scheds *db.ScheduleRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		s, err := scheds.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Schedule not found")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"schedule": s,
		})
	}
}

type updateSchedulePayload struct {
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload"`
	TargetType     string          `json:"target_type"`
	TargetIDs      []string        `json:"target_ids"`
	CronExpression string          `json:"cron_expression"`
}

func updateScheduleHandler(scheds *db.ScheduleRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		id := chi.URLParam(r, "id")

		var req updateSchedulePayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		if req.Name == "" || len(req.Name) > maxScheduleNameLen {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required and must be under 256 characters")
			return
		}
		validTypes := map[string]bool{"command": true, "container_action": true, "service_action": true, "update": true}
		if !validTypes[req.Type] {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "type must be command, container_action, service_action, or update")
			return
		}
		if req.CronExpression != "" {
			if err := scheduler.ValidateCronExpression(req.CronExpression); err != nil {
				writeError(w, http.StatusBadRequest, "INVALID_CRON", "Invalid cron expression: "+err.Error())
				return
			}
		}

		targetIDsBytes, _ := json.Marshal(req.TargetIDs)
		now := time.Now().UTC()
		nextRun := cronexpr.MustParse(req.CronExpression).Next(now)

		if req.Payload == nil {
			req.Payload = json.RawMessage("{}")
		}

		s, err := scheds.Update(r.Context(), id, req.Name, req.Type, req.Payload, req.TargetType, targetIDsBytes, req.CronExpression, nextRun)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Schedule not found")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "schedule.updated", "schedule", id)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"schedule": s,
		})
	}
}

func deleteScheduleHandler(scheds *db.ScheduleRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		id := chi.URLParam(r, "id")

		if err := scheds.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Schedule not found")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "schedule.deleted", "schedule", id)

		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func enableScheduleHandler(scheds *db.ScheduleRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		id := chi.URLParam(r, "id")

		if err := scheds.SetEnabled(r.Context(), id, true); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Schedule not found")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "schedule.enabled", "schedule", id)

		writeJSON(w, http.StatusOK, map[string]string{"status": "enabled"})
	}
}

func disableScheduleHandler(scheds *db.ScheduleRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		id := chi.URLParam(r, "id")

		if err := scheds.SetEnabled(r.Context(), id, false); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Schedule not found")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "schedule.disabled", "schedule", id)

		writeJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
	}
}

func listScheduleRunsHandler(runs *db.ScheduleRunRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		list, err := runs.ListBySchedule(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Schedule not found")
			return
		}
		if list == nil {
			list = []*db.ScheduleRun{}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"runs": list,
		})
	}
}
