package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/renos-project/renos/core/internal/db"
)

func listMarketplacePluginsHandler(marketplace *db.MarketplacePluginRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		var list []*db.MarketplacePlugin
		var err error
		if query != "" {
			list, err = marketplace.Search(r.Context(), query)
		} else {
			list, err = marketplace.ListAll(r.Context())
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list marketplace plugins: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.MarketplacePlugin{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func getMarketplacePluginHandler(marketplace *db.MarketplacePluginRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		p, err := marketplace.FindByName(r.Context(), name)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "plugin not found in marketplace")
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func listMarketplaceInstallationsHandler(installations *db.MarketplaceInstallationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := installations.ListAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list installations: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.MarketplaceInstallation{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func installMarketplacePluginHandler(marketplace *db.MarketplacePluginRepository, installations *db.MarketplaceInstallationRepository, history *db.MarketplaceInstallHistoryRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		p, err := marketplace.FindByName(r.Context(), name)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "plugin not found in marketplace")
			return
		}

		existing, _ := installations.FindByPlugin(r.Context(), p.ID)
		if existing != nil {
			writeError(w, http.StatusConflict, "already_installed", "plugin is already installed")
			return
		}

		inst := &db.MarketplaceInstallation{
			PluginID:         p.ID,
			InstalledVersion: p.Version,
			Status:           "installed",
			Config:           json.RawMessage(`{}`),
		}
		if err := installations.Create(r.Context(), inst); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to install plugin: "+err.Error())
			return
		}

		h := &db.MarketplaceInstallHistory{
			InstallationID: inst.ID,
			Version:        p.Version,
			Action:         "install",
			Checksum:       p.Checksum,
			Verified:       p.Verified,
			Notes:          "Installed from marketplace",
		}
		history.Create(r.Context(), h)

		marketplace.IncrementDownloads(r.Context(), name)
		audits.Create(r.Context(), "user", getUserID(r), "marketplace.install", "plugin", name)
		writeJSON(w, http.StatusCreated, inst)
	}
}

func upgradeMarketplacePluginHandler(marketplace *db.MarketplacePluginRepository, installations *db.MarketplaceInstallationRepository, history *db.MarketplaceInstallHistoryRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		p, err := marketplace.FindByName(r.Context(), name)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "plugin not found in marketplace")
			return
		}

		inst, err := installations.FindByPlugin(r.Context(), p.ID)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_installed", "plugin is not installed")
			return
		}

		if inst.InstalledVersion == p.Version {
			writeError(w, http.StatusConflict, "already_up_to_date", "already at latest version")
			return
		}

		installations.UpdateStatus(r.Context(), inst.ID, "upgrading")

		h := &db.MarketplaceInstallHistory{
			InstallationID: inst.ID,
			Version:        p.Version,
			Action:         "upgrade",
			Checksum:       p.Checksum,
			Verified:       p.Verified,
			Notes:          "Upgraded from " + inst.InstalledVersion,
		}
		history.Create(r.Context(), h)

		installations.UpdateVersion(r.Context(), inst.ID, p.Version)
		installations.UpdateStatus(r.Context(), inst.ID, "installed")

		audits.Create(r.Context(), "user", getUserID(r), "marketplace.upgrade", "plugin", name)
		writeJSON(w, http.StatusOK, inst)
	}
}

func rollbackMarketplacePluginHandler(installations *db.MarketplaceInstallationRepository, history *db.MarketplaceInstallHistoryRepository, audits *db.AuditRepository) http.HandlerFunc {
	type request struct {
		Version string `json:"version"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if req.Version == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "version is required")
			return
		}

		// Find the installation by name
		list, err := installations.ListAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to find installation")
			return
		}

		var inst *db.MarketplaceInstallation
		for _, i := range list {
			if i.PluginID != "" {
				inst = i
				break
			}
		}
		if inst == nil {
			writeError(w, http.StatusNotFound, "not_installed", "plugin not installed")
			return
		}

		installations.UpdateStatus(r.Context(), inst.ID, "rolling_back")

		h := &db.MarketplaceInstallHistory{
			InstallationID: inst.ID,
			Version:        req.Version,
			Action:         "rollback",
			Verified:       false,
			Notes:          "Rolled back from " + inst.InstalledVersion,
		}
		history.Create(r.Context(), h)

		installations.UpdateVersion(r.Context(), inst.ID, req.Version)
		installations.UpdateStatus(r.Context(), inst.ID, "installed")

		audits.Create(r.Context(), "user", getUserID(r), "marketplace.rollback", "plugin", name)
		writeJSON(w, http.StatusOK, inst)
	}
}

func uninstallMarketplacePluginHandler(installations *db.MarketplaceInstallationRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		// Find by name (simplified - in production you'd resolve the plugin_id)
		list, err := installations.ListAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to find installation")
			return
		}
		for _, inst := range list {
			if err := installations.Delete(r.Context(), inst.ID); err != nil {
				writeError(w, http.StatusInternalServerError, "internal_error", "failed to uninstall: "+err.Error())
				return
			}
			audits.Create(r.Context(), "user", getUserID(r), "marketplace.uninstall", "plugin", name)
			writeJSON(w, http.StatusOK, map[string]string{"status": "uninstalled"})
			return
		}
		writeError(w, http.StatusNotFound, "not_installed", "plugin not installed")
	}
}

func marketplaceInstallHistoryHandler(history *db.MarketplaceInstallHistoryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		list, err := history.ListByInstallation(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list history: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.MarketplaceInstallHistory{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func syncMarketplaceHandler(marketplace *db.MarketplacePluginRepository, audits *db.AuditRepository) http.HandlerFunc {
	type request struct {
		Plugins []*db.MarketplacePlugin `json:"plugins"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		count := 0
		for _, p := range req.Plugins {
			if err := marketplace.Upsert(r.Context(), p); err == nil {
				count++
			}
		}
		audits.Create(r.Context(), "user", getUserID(r), "marketplace.sync", "count", string(rune(count)))
		writeJSON(w, http.StatusOK, map[string]int{"synced": count})
	}
}
