package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/renos-project/renos/core/internal/db"
)

const (
	inAppDefaultLimit = 50
	inAppMaxLimit     = 200
)

// listInAppNotificationsHandler returns the current user's notifications with
// optional filters: ?unread=true&type=workflow&search=deploy&limit=30&offset=0
func listInAppNotificationsHandler(repo *db.InAppNotificationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		f := db.InAppNotificationFilter{
			UnreadOnly: r.URL.Query().Get("unread") == "true",
			Type:       r.URL.Query().Get("type"),
			Search:     r.URL.Query().Get("search"),
			Offset:     0,
		}
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 {
				f.Limit = min(n, inAppMaxLimit)
			}
		}
		if o := r.URL.Query().Get("offset"); o != "" {
			if n, err := strconv.Atoi(o); err == nil && n > 0 {
				f.Offset = min(n, 10000)
			}
		}
		if f.Limit <= 0 {
			f.Limit = inAppDefaultLimit
		}
		list, err := repo.ListForUser(r.Context(), userID, f)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list notifications: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.InAppNotification{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

// unreadCountHandler returns { count: N } for the current user.
func unreadCountHandler(repo *db.InAppNotificationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		count, err := repo.UnreadCount(r.Context(), getUserID(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to count notifications: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]int{"count": count})
	}
}

// markNotificationReadHandler marks a single notification read.
func markNotificationReadHandler(repo *db.InAppNotificationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := repo.MarkRead(r.Context(), getUserID(r), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to mark notification read: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"read": true})
	}
}

// markAllNotificationsReadHandler marks every unread notification read.
func markAllNotificationsReadHandler(repo *db.InAppNotificationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := repo.MarkAllRead(r.Context(), getUserID(r)); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to mark notifications read: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"read": true})
	}
}
