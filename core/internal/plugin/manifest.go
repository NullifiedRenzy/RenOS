package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Manifest struct {
	Name            string            `toml:"name"`
	DisplayName     string            `toml:"display_name"`
	Version         string            `toml:"version"`
	Author          string            `toml:"author"`
	AuthorURL       string            `toml:"author_url"`
	Homepage        string            `toml:"homepage"`
	License         string            `toml:"license"`
	Description     string            `toml:"description"`
	PluginType      string            `toml:"plugin_type"`
	EntryPoint      string            `toml:"entry_point"`
	MinCoreVersion  string            `toml:"min_core_version"`
	Permissions     []string          `toml:"permissions"`
	Dependencies    []Dependency      `toml:"dependencies"`
	ConfigSchema    map[string]Config `toml:"config_schema"`
	Capabilities    Capabilities      `toml:"capabilities"`
}

type Dependency struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
}

type Config struct {
	Type        string      `toml:"type"`
	Default     interface{} `toml:"default"`
	Description string      `toml:"description"`
	Required    bool        `toml:"required"`
	Options     []string    `toml:"options"`
}

type Capabilities struct {
	Routes   []RouteSpec   `toml:"routes"`
	Pages    []PageSpec    `toml:"pages"`
	Sidebar  []SidebarSpec `toml:"sidebar"`
	Workers  []WorkerSpec  `toml:"workers"`
	Schedules []ScheduleSpec `toml:"schedules"`
}

type RouteSpec struct {
	Method      string `toml:"method"`
	Path        string `toml:"path"`
	Handler     string `toml:"handler"`
	Description string `toml:"description"`
	RequireAuth bool   `toml:"require_auth"`
	MinRole     string `toml:"min_role"`
	RateLimit   int    `toml:"rate_limit"`
}

type PageSpec struct {
	PagePath    string `toml:"page_path"`
	Title       string `toml:"title"`
	Description string `toml:"description"`
	Icon        string `toml:"icon"`
	Component   string `toml:"component"`
	ParentPath  string `toml:"parent_path"`
	SortOrder   int    `toml:"sort_order"`
	RequireAuth bool   `toml:"require_auth"`
	MinRole     string `toml:"min_role"`
}

type SidebarSpec struct {
	Label       string `toml:"label"`
	Icon        string `toml:"icon"`
	Path        string `toml:"path"`
	ParentLabel string `toml:"parent_label"`
	SortOrder   int    `toml:"sort_order"`
	MinRole     string `toml:"min_role"`
}

type WorkerSpec struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
	Handler     string `toml:"handler"`
	Concurrency int    `toml:"concurrency"`
}

type ScheduleSpec struct {
	Name            string `toml:"name"`
	Description     string `toml:"description"`
	Handler         string `toml:"handler"`
	CronExpr        string `toml:"cron_expr"`
	IntervalSeconds int    `toml:"interval_seconds"`
}

func LoadManifest(pluginDir string) (*Manifest, error) {
	manifestPath := filepath.Join(pluginDir, "plugin.toml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin.toml not found in %s", pluginDir)
	}

	var m Manifest
	if _, err := toml.DecodeFile(manifestPath, &m); err != nil {
		return nil, fmt.Errorf("parse plugin.toml: %w", err)
	}

	if err := m.Validate(); err != nil {
		return nil, err
	}

	return &m, nil
}

func (m *Manifest) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("plugin version is required")
	}
	if m.PluginType == "" {
		m.PluginType = "go"
	}
	if m.Capabilities.Workers != nil {
		for _, w := range m.Capabilities.Workers {
			if w.Concurrency <= 0 {
				w.Concurrency = 1
			}
		}
	}
	if m.Capabilities.Routes != nil {
		for i, r := range m.Capabilities.Routes {
			if r.RateLimit <= 0 {
				m.Capabilities.Routes[i].RateLimit = 100
			}
			if r.MinRole == "" {
				m.Capabilities.Routes[i].MinRole = "viewer"
			}
		}
	}
	return nil
}

func (m *Manifest) HasCapability(cap string) bool {
	switch cap {
	case "routes":
		return len(m.Capabilities.Routes) > 0
	case "pages":
		return len(m.Capabilities.Pages) > 0
	case "sidebar":
		return len(m.Capabilities.Sidebar) > 0
	case "workers":
		return len(m.Capabilities.Workers) > 0
	case "schedules":
		return len(m.Capabilities.Schedules) > 0
	}
	return false
}
