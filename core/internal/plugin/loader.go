package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/renos-project/renos/core/internal/db"
)

type Loader struct {
	pluginDir   string
	repo        *db.InstalledPluginRepository
	mu          sync.RWMutex
	plugins     map[string]*LoadedPlugin
	stopChans   map[string]chan struct{}
}

type LoadedPlugin struct {
	Manifest   *Manifest
	PluginDir  string
	PluginDB   *db.InstalledPlugin
}

func NewLoader(pluginDir string, repo *db.InstalledPluginRepository) *Loader {
	return &Loader{
		pluginDir: pluginDir,
		repo:      repo,
		plugins:   make(map[string]*LoadedPlugin),
		stopChans: make(map[string]chan struct{}),
	}
}

func (l *Loader) Start(ctx context.Context) error {
	log.Printf("[plugin-loader] scanning plugin directory: %s", l.pluginDir)
	if err := os.MkdirAll(l.pluginDir, 0755); err != nil {
		return fmt.Errorf("create plugin dir: %w", err)
	}

	entries, err := os.ReadDir(l.pluginDir)
	if err != nil {
		return fmt.Errorf("read plugin dir: %w", err)
	}

	loaded := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pluginDir := filepath.Join(l.pluginDir, entry.Name())
		if err := l.loadPlugin(ctx, pluginDir); err != nil {
			log.Printf("[plugin-loader] skip %s: %v", entry.Name(), err)
			continue
		}
		loaded++
	}

	log.Printf("[plugin-loader] loaded %d plugins", loaded)

	go l.watchLoop(ctx)
	return nil
}

func (l *Loader) loadPlugin(ctx context.Context, pluginDir string) error {
	manifest, err := LoadManifest(pluginDir)
	if err != nil {
		return err
	}

	dbPlugin, err := l.repo.FindByName(ctx, manifest.Name)
	if err != nil {
		dbPlugin = &db.InstalledPlugin{
			ID:           uuid.New().String(),
			Name:         manifest.Name,
			DisplayName:  manifest.DisplayName,
			Version:      manifest.Version,
			Author:       manifest.Author,
			Description:  manifest.Description,
			PluginType:   manifest.PluginType,
			Status:       "installed",
			EntryPoint:   manifest.EntryPoint,
		}
		permissionsJSON, _ := json.Marshal(manifest.Permissions)
		dbPlugin.Permissions = permissionsJSON
		configSchemaJSON, _ := json.Marshal(manifest.ConfigSchema)
		dbPlugin.ConfigSchema = configSchemaJSON
		depsJSON, _ := json.Marshal(manifest.Dependencies)
		dbPlugin.Dependencies = depsJSON
		configJSON, _ := json.Marshal(map[string]interface{}{})
		dbPlugin.Config = configJSON

		hash, err := computeDirHash(pluginDir)
		if err == nil {
			dbPlugin.BlobHash = hash
		}
		dbPlugin.BlobPath = pluginDir

		if err := l.repo.Create(ctx, dbPlugin); err != nil {
			return fmt.Errorf("create plugin record: %w", err)
		}
	} else {
		if dbPlugin.Status == "uninstalled" {
			return fmt.Errorf("plugin %s is uninstalled", manifest.Name)
		}
		hash, err := computeDirHash(pluginDir)
		if err == nil && hash != dbPlugin.BlobHash {
			dbPlugin.Version = manifest.Version
			dbPlugin.BlobHash = hash
			dbPlugin.UpdatedAt = time.Now()
		}
	}

	loaded := &LoadedPlugin{
		Manifest: manifest,
		PluginDir: pluginDir,
		PluginDB: dbPlugin,
	}

	l.mu.Lock()
	l.plugins[manifest.Name] = loaded
	l.mu.Unlock()

	if err := l.syncCapabilities(ctx, loaded); err != nil {
		log.Printf("[plugin-loader] sync capabilities for %s: %v", manifest.Name, err)
	}

	if dbPlugin.Status == "enabled" {
		if err := l.StartWorkers(ctx, loaded); err != nil {
			log.Printf("[plugin-loader] start workers for %s: %v", manifest.Name, err)
		}
	}

	return nil
}

func (l *Loader) syncCapabilities(ctx context.Context, p *LoadedPlugin) error {
	m := p.Manifest
	pid := p.PluginDB.ID

	routes := make([]*db.PluginRoute, 0, len(m.Capabilities.Routes))
	for _, r := range m.Capabilities.Routes {
		routes = append(routes, &db.PluginRoute{
			Method:      r.Method,
			Path:        "/api/v1/plugins/" + m.Name + r.Path,
			HandlerName: r.Handler,
			Description: r.Description,
			RequireAuth: r.RequireAuth,
			MinRole:     r.MinRole,
			RateLimit:   r.RateLimit,
		})
	}
	if err := l.repo.ReplaceRoutes(ctx, pid, routes); err != nil {
		return fmt.Errorf("replace routes: %w", err)
	}

	pages := make([]*db.PluginPage, 0, len(m.Capabilities.Pages))
	for _, pg := range m.Capabilities.Pages {
		pages = append(pages, &db.PluginPage{
			PagePath:      "/plugins/" + m.Name + pg.PagePath,
			Title:         pg.Title,
			Description:   pg.Description,
			Icon:          pg.Icon,
			ComponentName: pg.Component,
			ParentPath:    pg.ParentPath,
			SortOrder:     pg.SortOrder,
			RequireAuth:   pg.RequireAuth,
			MinRole:       pg.MinRole,
		})
	}
	if err := l.repo.ReplacePages(ctx, pid, pages); err != nil {
		return fmt.Errorf("replace pages: %w", err)
	}

	sidebar := make([]*db.PluginSidebarEntry, 0, len(m.Capabilities.Sidebar))
	for _, s := range m.Capabilities.Sidebar {
		sidebar = append(sidebar, &db.PluginSidebarEntry{
			Label:       s.Label,
			Icon:        s.Icon,
			Path:        "/plugins/" + m.Name + s.Path,
			ParentLabel: s.ParentLabel,
			SortOrder:   s.SortOrder,
			MinRole:     s.MinRole,
		})
	}
	if err := l.repo.ReplaceSidebar(ctx, pid, sidebar); err != nil {
		return fmt.Errorf("replace sidebar: %w", err)
	}

	workers := make([]*db.PluginWorker, 0, len(m.Capabilities.Workers))
	for _, w := range m.Capabilities.Workers {
		workers = append(workers, &db.PluginWorker{
			WorkerName:   w.Name,
			Description:  w.Description,
			HandlerName:  w.Handler,
			Concurrency:  w.Concurrency,
			Status:       "stopped",
		})
	}
	if err := l.repo.ReplaceWorkers(ctx, pid, workers); err != nil {
		return fmt.Errorf("replace workers: %w", err)
	}

	schedules := make([]*db.PluginSchedule, 0, len(m.Capabilities.Schedules))
	for _, s := range m.Capabilities.Schedules {
		schedules = append(schedules, &db.PluginSchedule{
			ScheduleName:    s.Name,
			Description:     s.Description,
			HandlerName:     s.Handler,
			CronExpr:        s.CronExpr,
			IntervalSeconds: s.IntervalSeconds,
			Enabled:         true,
		})
	}
	if err := l.repo.ReplaceSchedules(ctx, pid, schedules); err != nil {
		return fmt.Errorf("replace schedules: %w", err)
	}

	return nil
}

func (l *Loader) EnablePlugin(ctx context.Context, name string) error {
	p, err := l.repo.FindByName(ctx, name)
	if err != nil {
		return fmt.Errorf("plugin not found: %w", err)
	}
	if err := l.repo.UpdateStatus(ctx, p.ID, "enabled"); err != nil {
		return err
	}
	p.Status = "enabled"
	l.mu.Lock()
	if loaded, ok := l.plugins[name]; ok {
		loaded.PluginDB = p
	}
	l.mu.Unlock()

	log.Printf("[plugin-loader] enabled plugin %s", name)
	return nil
}

func (l *Loader) DisablePlugin(ctx context.Context, name string) error {
	p, err := l.repo.FindByName(ctx, name)
	if err != nil {
		return fmt.Errorf("plugin not found: %w", err)
	}
	l.StopWorkers(name)
	if err := l.repo.UpdateStatus(ctx, p.ID, "disabled"); err != nil {
		return err
	}
	p.Status = "disabled"
	l.mu.Lock()
	if loaded, ok := l.plugins[name]; ok {
		loaded.PluginDB = p
	}
	l.mu.Unlock()

	log.Printf("[plugin-loader] disabled plugin %s", name)
	return nil
}

func (l *Loader) RemovePlugin(ctx context.Context, name string) error {
	p, err := l.repo.FindByName(ctx, name)
	if err != nil {
		return fmt.Errorf("plugin not found: %w", err)
	}
	l.StopWorkers(name)
	if err := l.repo.Delete(ctx, p.ID); err != nil {
		return err
	}

	l.mu.Lock()
	delete(l.plugins, name)
	l.mu.Unlock()

	log.Printf("[plugin-loader] removed plugin %s", name)
	return nil
}

func (l *Loader) GetPlugin(name string) (*LoadedPlugin, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	p, ok := l.plugins[name]
	return p, ok
}

func (l *Loader) ListPlugins() []*LoadedPlugin {
	l.mu.RLock()
	defer l.mu.RUnlock()
	list := make([]*LoadedPlugin, 0, len(l.plugins))
	for _, p := range l.plugins {
		list = append(list, p)
	}
	return list
}

func (l *Loader) StartWorkers(ctx context.Context, p *LoadedPlugin) error {
	workers, err := l.repo.ListWorkers(ctx)
	if err != nil {
		return err
	}
	for _, w := range workers {
		if w.PluginID != p.PluginDB.ID {
			continue
		}
		if w.Status == "running" {
			continue
		}
		stopChan := make(chan struct{})
		l.mu.Lock()
		l.stopChans[p.Manifest.Name+":"+w.WorkerName] = stopChan
		l.mu.Unlock()

		go l.runWorker(ctx, p, w, stopChan)
	}
	return nil
}

func (l *Loader) StopWorkers(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for key, ch := range l.stopChans {
		if strings.HasPrefix(key, name+":") {
			close(ch)
			delete(l.stopChans, key)
		}
	}
}

func (l *Loader) runWorker(ctx context.Context, p *LoadedPlugin, w *db.PluginWorker, stop chan struct{}) {
	log.Printf("[plugin-loader] starting worker %s/%s", p.Manifest.Name, w.WorkerName)
	_ = l.repo.UpdateWorkerStatus(ctx, w.ID, "running", "")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			log.Printf("[plugin-loader] stopping worker %s/%s", p.Manifest.Name, w.WorkerName)
			_ = l.repo.UpdateWorkerStatus(ctx, w.ID, "stopped", "")
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Worker tick — future: invoke actual Go handler
		}
	}
}

func (l *Loader) watchLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			entries, err := os.ReadDir(l.pluginDir)
			if err != nil {
				continue
			}
			existing := make(map[string]bool)
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				pluginDir := filepath.Join(l.pluginDir, entry.Name())
				m, err := LoadManifest(pluginDir)
				if err != nil {
					continue
				}
				existing[m.Name] = true
				l.mu.RLock()
				_, loaded := l.plugins[m.Name]
				l.mu.RUnlock()
				if !loaded {
					if err := l.loadPlugin(ctx, pluginDir); err != nil {
						log.Printf("[plugin-loader] auto-load %s: %v", entry.Name(), err)
					} else {
						log.Printf("[plugin-loader] auto-loaded plugin %s", m.Name)
					}
				}
			}
		}
	}
}

func computeDirHash(dir string) (string, error) {
	h := sha256.New()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		h.Write([]byte(path))
		h.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
