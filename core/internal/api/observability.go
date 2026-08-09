package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/renos-project/renos/core/internal/db"
)

type MetricsResponse struct {
	TotalJobs        int64   `json:"total_jobs"`
	RunningJobs      int64   `json:"running_jobs"`
	FailedJobs       int64   `json:"failed_jobs"`
	CompletedJobs    int64   `json:"completed_jobs"`
	StoppedJobs      int64   `json:"stopped_jobs"`
	AvgJobDurationMs float64 `json:"avg_job_duration_ms"`
	PolicyBlocks     int64   `json:"policy_blocks"`
	ScheduleExecs    int64   `json:"schedule_executions"`
}

func metricsHandler(jobs *db.JobRepository, policies *db.PolicyRepository, schedules *db.ScheduleRunRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		stats, err := jobs.CountByStatuses(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to count jobs")
			return
		}

		policyBlocks, err := policies.CountPolicyBlocks(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to count policy blocks")
			return
		}

		scheduleExecs, err := schedules.Count(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to count schedule runs")
			return
		}

		writeJSON(w, http.StatusOK, MetricsResponse{
			TotalJobs:        stats.Total,
			RunningJobs:      stats.ByStatus["running"],
			FailedJobs:       stats.ByStatus["failed"],
			CompletedJobs:    stats.ByStatus["completed"],
			StoppedJobs:      stats.ByStatus["stopped"],
			AvgJobDurationMs: stats.AvgMs,
			PolicyBlocks:     policyBlocks,
			ScheduleExecs:    scheduleExecs,
		})
	}
}

type JobStatsResponse struct {
	TotalJobs        int64   `json:"total_jobs"`
	RunningJobs      int64   `json:"running_jobs"`
	FailedJobs       int64   `json:"failed_jobs"`
	CompletedJobs    int64   `json:"completed_jobs"`
	StoppedJobs      int64   `json:"stopped_jobs"`
	CancelledJobs    int64   `json:"cancelled_jobs"`
	AvgJobDurationMs float64 `json:"avg_job_duration_ms"`
}

func jobStatsHandler(jobs *db.JobRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		stats, err := jobs.CountByStatuses(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to count jobs")
			return
		}

		writeJSON(w, http.StatusOK, JobStatsResponse{
			TotalJobs:        stats.Total,
			RunningJobs:      stats.ByStatus["running"],
			FailedJobs:       stats.ByStatus["failed"],
			CompletedJobs:    stats.ByStatus["completed"],
			StoppedJobs:      stats.ByStatus["stopped"],
			CancelledJobs:    stats.ByStatus["cancelled"],
			AvgJobDurationMs: stats.AvgMs,
		})
	}
}

func listLogsHandler(logs *db.SystemLogRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := &db.LogFilter{}

		if lvl := r.URL.Query().Get("level"); lvl != "" {
			filter.Level = lvl
		}
		if src := r.URL.Query().Get("source"); src != "" {
			filter.Source = src
		}
		if since := r.URL.Query().Get("since"); since != "" {
			t, err := time.Parse(time.RFC3339, since)
			if err == nil {
				filter.Since = &t
			}
		}
		if until := r.URL.Query().Get("until"); until != "" {
			t, err := time.Parse(time.RFC3339, until)
			if err == nil {
				filter.Until = &t
			}
		}
		if offset := r.URL.Query().Get("offset"); offset != "" {
			if n, err := strconv.Atoi(offset); err == nil && n >= 0 {
				filter.Offset = n
			}
		}
		if limit := r.URL.Query().Get("limit"); limit != "" {
			if n, err := strconv.Atoi(limit); err == nil && n > 0 {
				filter.Limit = n
			}
		}

		list, total, err := logs.FindAll(r.Context(), filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list logs")
			return
		}
		if list == nil {
			list = []*db.SystemLog{}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"logs":  list,
			"total": total,
			"limit": filter.Limit,
			"offset": filter.Offset,
		})
	}
}

func createLogHandler(logs *db.SystemLogRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Level    string          `json:"level"`
			Source   string          `json:"source"`
			Message  string          `json:"message"`
			Metadata json.RawMessage `json:"metadata,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.Level == "" || req.Source == "" || req.Message == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "level, source, and message are required")
			return
		}

		log, err := logs.Create(r.Context(), req.Level, req.Source, req.Message, req.Metadata)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Failed to create log: %v", err))
			return
		}

		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"log": log,
		})
	}
}
