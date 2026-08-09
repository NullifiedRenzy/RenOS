package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/renos-project/renos/core/internal/db"
)

func listNotificationProvidersHandler(providers *db.NotificationProviderRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := providers.ListAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list providers: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.NotificationProvider{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func createNotificationProviderHandler(providers *db.NotificationProviderRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p db.NotificationProvider
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if p.Name == "" || p.ProviderType == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "name and provider_type are required")
			return
		}
		if p.Config == nil {
			p.Config = json.RawMessage(`{}`)
		}
		p.Enabled = true
		if err := providers.Create(r.Context(), &p); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create provider: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "notification.provider.create", "provider", p.ID)
		writeJSON(w, http.StatusCreated, p)
	}
}

func toggleNotificationProviderHandler(providers *db.NotificationProviderRepository, audits *db.AuditRepository) http.HandlerFunc {
	type request struct {
		Enabled bool `json:"enabled"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if err := providers.ToggleEnabled(r.Context(), id, req.Enabled); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to toggle provider: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "notification.provider.toggle", "provider", id)
		writeJSON(w, http.StatusOK, map[string]bool{"enabled": req.Enabled})
	}
}

func deleteNotificationProviderHandler(providers *db.NotificationProviderRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := providers.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete provider: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "notification.provider.delete", "provider", id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func listNotificationTemplatesHandler(templates *db.NotificationTemplateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := templates.ListAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list templates: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.NotificationTemplate{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func createNotificationTemplateHandler(templates *db.NotificationTemplateRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var t db.NotificationTemplate
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if t.Name == "" || t.Body == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "name and body are required")
			return
		}
		if t.TemplateType == "" {
			t.TemplateType = "message"
		}
		if t.Variables == nil {
			t.Variables = json.RawMessage(`[]`)
		}
		if err := templates.Create(r.Context(), &t); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create template: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "notification.template.create", "template", t.ID)
		writeJSON(w, http.StatusCreated, t)
	}
}

func deleteNotificationTemplateHandler(templates *db.NotificationTemplateRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := templates.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete template: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "notification.template.delete", "template", id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func listNotificationRulesHandler(rules *db.NotificationRuleRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := rules.ListAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list rules: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.NotificationRule{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func createNotificationRuleHandler(rules *db.NotificationRuleRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var nr db.NotificationRule
		if err := json.NewDecoder(r.Body).Decode(&nr); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if nr.Name == "" || nr.EventType == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "name and event_type are required")
			return
		}
		if nr.Conditions == nil {
			nr.Conditions = json.RawMessage(`{}`)
		}
		if nr.ProviderIDs == nil {
			nr.ProviderIDs = json.RawMessage(`[]`)
		}
		if nr.CooldownSeconds == 0 {
			nr.CooldownSeconds = 300
		}
		nr.Enabled = true
		if err := rules.Create(r.Context(), &nr); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create rule: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "notification.rule.create", "rule", nr.ID)
		writeJSON(w, http.StatusCreated, nr)
	}
}

func deleteNotificationRuleHandler(rules *db.NotificationRuleRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := rules.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete rule: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "notification.rule.delete", "rule", id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func listNotificationHistoryHandler(history *db.NotificationHistoryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := history.ListRecent(r.Context(), 100)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list history: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.NotificationHistoryEntry{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func notificationStatsHandler(history *db.NotificationHistoryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := history.GetStats(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to get stats: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, stats)
	}
}
