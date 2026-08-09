package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
)

func listJobsHandler(jobs *db.JobRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)

		list, err := jobs.FindAll(r.Context(), parseListLimit(r.URL.Query().Get("limit")), parseOffset(r.URL.Query().Get("offset")))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list jobs")
			return
		}
		if list == nil {
			list = []*db.Job{}
		}

		audits.Create(r.Context(), "user", claims.UserID, "job.list", "job", "")

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"jobs": list,
		})
	}
}

func getJobHandler(jobs *db.JobRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		job, err := jobs.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Job not found")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"job": job,
		})
	}
}

func cancelJobHandler(jobs *db.JobRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		id := chi.URLParam(r, "id")

		if err := jobs.Cancel(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to cancel job")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "job.cancelled", "job", id)

		writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
	}
}

func retryJobHandler(jobs *db.JobRepository, groupActions *db.GroupActionRepository, groups *db.GroupRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		id := chi.URLParam(r, "id")

		job, err := jobs.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Job not found")
			return
		}
		if job.Status != "failed" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Only failed jobs can be retried")
			return
		}

		if job.Retries >= job.MaxRetries {
			// Reset retries to give it another full cycle
			job.Retries = 0
		}

		if err := jobs.ResetStatus(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to reset job")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "job.retried", "job", id)

		writeJSON(w, http.StatusOK, map[string]string{"status": "retried"})
	}
}
