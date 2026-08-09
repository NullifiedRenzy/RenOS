package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/plugin"
)

// --- Admin Installed Plugin Endpoints ---

func listInstalledPluginsHandler(installedRepo *db.InstalledPluginRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := installedRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list installed plugins")
			return
		}
		if list == nil {
			list = []*db.InstalledPlugin{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"plugins": list,
		})
	}
}

func getInstalledPluginHandler(installedRepo *db.InstalledPluginRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "plugin name required")
			return
		}
		p, err := installedRepo.FindByName(r.Context(), name)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Plugin not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"plugin": p,
		})
	}
}

func enableInstalledPluginHandler(loader *plugin.Loader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "plugin name required")
			return
		}
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		if claims.Role != "admin" {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "admin role required")
			return
		}
		if err := loader.EnablePlugin(r.Context(), name); err != nil {
			writeError(w, http.StatusInternalServerError, "PLUGIN_ERROR", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "plugin enabled",
			"name":    name,
		})
	}
}

func disableInstalledPluginHandler(loader *plugin.Loader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "plugin name required")
			return
		}
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		if claims.Role != "admin" {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "admin role required")
			return
		}
		if err := loader.DisablePlugin(r.Context(), name); err != nil {
			writeError(w, http.StatusInternalServerError, "PLUGIN_ERROR", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "plugin disabled",
			"name":    name,
		})
	}
}

func removeInstalledPluginHandler(loader *plugin.Loader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "plugin name required")
			return
		}
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		if claims.Role != "admin" {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "admin role required")
			return
		}
		if err := loader.RemovePlugin(r.Context(), name); err != nil {
			writeError(w, http.StatusInternalServerError, "PLUGIN_ERROR", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "plugin removed",
			"name":    name,
		})
	}
}

func updateInstalledPluginConfigHandler(installedRepo *db.InstalledPluginRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "plugin name required")
			return
		}
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		if claims.Role != "admin" {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "admin role required")
			return
		}
		var req struct {
			Config json.RawMessage `json:"config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		p, err := installedRepo.FindByName(r.Context(), name)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Plugin not found")
			return
		}
		if err := installedRepo.UpdateConfig(r.Context(), p.ID, req.Config); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update config")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "config updated",
			"name":    name,
		})
	}
}

func listInstalledPluginRoutesHandler(installedRepo *db.InstalledPluginRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "plugin name required")
			return
		}
		p, err := installedRepo.FindByName(r.Context(), name)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Plugin not found")
			return
		}
		routes, err := installedRepo.ListRoutes(r.Context(), p.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list routes")
			return
		}
		if routes == nil {
			routes = []*db.PluginRoute{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"routes": routes,
		})
	}
}

func listInstalledPluginPagesHandler(installedRepo *db.InstalledPluginRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "plugin name required")
			return
		}
		p, err := installedRepo.FindByName(r.Context(), name)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Plugin not found")
			return
		}
		pages, err := installedRepo.ListPages(r.Context(), p.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list pages")
			return
		}
		if pages == nil {
			pages = []*db.PluginPage{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"pages": pages,
		})
	}
}

func listInstalledPluginSidebarHandler(installedRepo *db.InstalledPluginRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := installedRepo.ListSidebar(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list sidebar entries")
			return
		}
		if entries == nil {
			entries = []*db.PluginSidebarEntry{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"sidebar": entries,
		})
	}
}

func listInstalledPluginWorkersHandler(installedRepo *db.InstalledPluginRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "plugin name required")
			return
		}
		p, err := installedRepo.FindByName(r.Context(), name)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Plugin not found")
			return
		}
		workers, err := installedRepo.ListWorkers(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list workers")
			return
		}
		var result []*db.PluginWorker
		for _, w := range workers {
			if w.PluginID == p.ID {
				result = append(result, w)
			}
		}
		if result == nil {
			result = []*db.PluginWorker{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"workers": result,
		})
	}
}

func listInstalledPluginSchedulesHandler(installedRepo *db.InstalledPluginRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "plugin name required")
			return
		}
		p, err := installedRepo.FindByName(r.Context(), name)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Plugin not found")
			return
		}
		schedules, err := installedRepo.ListSchedules(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list schedules")
			return
		}
		var result []*db.PluginSchedule
		for _, s := range schedules {
			if s.PluginID == p.ID {
				result = append(result, s)
			}
		}
		if result == nil {
			result = []*db.PluginSchedule{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"schedules": result,
		})
	}
}

func listAllEnabledPluginRoutesHandler(installedRepo *db.InstalledPluginRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routes, err := installedRepo.ListAllRoutes(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list routes")
			return
		}
		if routes == nil {
			routes = []*db.PluginRoute{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"routes": routes,
		})
	}
}

func listAllEnabledPluginPagesHandler(installedRepo *db.InstalledPluginRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pages, err := installedRepo.ListAllPages(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list pages")
			return
		}
		if pages == nil {
			pages = []*db.PluginPage{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"pages": pages,
		})
	}
}

// DynamicPluginRoute mounts an enabled plugin's API routes into the router
func DynamicPluginRoute(installedRepo *db.InstalledPluginRepository, pluginLoader *plugin.Loader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/v1/plugins/"), "/", 2)
		if len(parts) < 2 {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid plugin route")
			return
		}
		pluginName := parts[0]
		_ = parts[1]

		p, err := installedRepo.FindByName(r.Context(), pluginName)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Plugin not found")
			return
		}
		if p.Status != "enabled" {
			writeError(w, http.StatusForbidden, "PLUGIN_DISABLED", "Plugin is not enabled")
			return
		}

		loaded, ok := pluginLoader.GetPlugin(pluginName)
		if !ok {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Plugin not loaded")
			return
		}

		_ = loaded

		writeJSON(w, http.StatusNotImplemented, map[string]interface{}{
			"error":   "plugin handler not implemented",
			"plugin":  pluginName,
			"message": "Plugin routes require a Go/WASM handler to be registered",
		})
	}
}
