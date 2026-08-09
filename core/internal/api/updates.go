package api

import (
	"encoding/json"
	"net/http"

	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/notify"
	"github.com/renos-project/renos/core/internal/update"
)

func getVersionInfoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		planner := update.NewUpdatePlanner(nil, nil)
		writeJSON(w, http.StatusOK, planner.GetVersionInfo())
	}
}

func checkForUpdatesHandler(history *db.UpdateHistoryRepository, notifier *notify.Notifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		planner := update.NewUpdatePlanner(history, nil)
		release, err := planner.CheckForUpdates(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "check_failed", "failed to check for updates: "+err.Error())
			return
		}
		if release == nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{"update_available": false, "current_version": update.CurrentVersion})
			return
		}
		if notifier != nil {
			notifier.Notify(r.Context(), "", "system", "Update available",
				"RenOS "+release.Version+" is available (channel: "+string(release.Channel)+").", "update", release.Version,
				map[string]any{"version": release.Version, "channel": string(release.Channel)})
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"update_available": true, "release": release})
	}
}

func planUpdateHandler(history *db.UpdateHistoryRepository) http.HandlerFunc {
	type request struct {
		TargetVersion string `json:"target_version"`
		Channel       string `json:"channel"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		planner := update.NewUpdatePlanner(history, nil)
		if req.Channel != "" {
			planner.SetChannel(update.Channel(req.Channel))
		}

		release, err := planner.CheckForUpdates(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "check_failed", err.Error())
			return
		}
		if release == nil {
			writeError(w, http.StatusNotFound, "no_update", "no update available for the specified channel")
			return
		}

		plan := planner.PlanUpdate(release)
		writeJSON(w, http.StatusOK, plan)
	}
}

func startUpdateHandler(history *db.UpdateHistoryRepository, backups *db.UpdateBackupRepository, audits *db.AuditRepository) http.HandlerFunc {
	type request struct {
		TargetVersion string `json:"target_version"`
		Channel       string `json:"channel"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}

		planner := update.NewUpdatePlanner(history, backups)
		if req.Channel != "" {
			planner.SetChannel(update.Channel(req.Channel))
		}

		release, err := planner.CheckForUpdates(r.Context())
		if err != nil || release == nil {
			writeError(w, http.StatusNotFound, "no_update", "no update available")
			return
		}

		uid := getUserID(r)
		entry := &db.UpdateHistoryEntry{
			VersionFrom: update.CurrentVersion,
			VersionTo:   release.Version,
			Channel:     string(release.Channel),
			Status:      "pending",
			StartedBy:   &uid,
		}
		if err := history.Create(r.Context(), entry); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create update entry")
			return
		}

		plan := planner.PlanUpdate(release)
		go planner.ApplyUpdate(r.Context(), plan, entry)

		audits.Create(r.Context(), "user", uid, "update.start", "update", entry.ID)
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "started", "update_id": entry.ID})
	}
}

func rollbackUpdateHandler(history *db.UpdateHistoryRepository, backups *db.UpdateBackupRepository, audits *db.AuditRepository) http.HandlerFunc {
	type request struct {
		BackupID string `json:"backup_id"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		json.NewDecoder(r.Body).Decode(&req)
		if req.BackupID == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "backup_id is required")
			return
		}

		planner := update.NewUpdatePlanner(history, backups)
		if err := planner.Rollback(r.Context(), req.BackupID); err != nil {
			writeError(w, http.StatusInternalServerError, "rollback_failed", err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "update.rollback", "backup", req.BackupID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "rolled_back"})
	}
}

func listUpdateHistoryHandler(history *db.UpdateHistoryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := history.ListAll(r.Context(), 50)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list update history")
			return
		}
		if list == nil { list = []*db.UpdateHistoryEntry{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func listUpdateBackupsHandler(backups *db.UpdateBackupRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := backups.ListAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list backups")
			return
		}
		if list == nil { list = []*db.UpdateBackup{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func setUpdateChannelHandler() http.HandlerFunc {
	type request struct {
		Channel string `json:"channel"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid body")
			return
		}
		ch := update.Channel(req.Channel)
		if ch != update.ChannelStable && ch != update.ChannelBeta && ch != update.ChannelNightly {
			writeError(w, http.StatusBadRequest, "invalid_channel", "must be stable, beta, or nightly")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"channel": string(ch)})
	}
}
