package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type PluginManifest struct {
	SchemaVersion  int           `json:"schemaVersion"`
	Name           string        `json:"name"`
	Version        string        `json:"version"`
	Description    string        `json:"description"`
	Author         string        `json:"author"`
	License        string        `json:"license"`
	Homepage       string        `json:"homepage,omitempty"`
	Repository     string        `json:"repository,omitempty"`
	MinCoreVersion string        `json:"minCoreVersion"`
	Permissions    []string      `json:"permissions,omitempty"`
	Routes         []RouteDef    `json:"routes,omitempty"`
	Workers        []WorkerDef   `json:"workers,omitempty"`
	Schedules      []ScheduleDef `json:"schedules,omitempty"`
	Webhooks       []WebhookDef  `json:"webhooks,omitempty"`
	Sidebar        []SidebarItem `json:"sidebar,omitempty"`
	Pages          []PageDef     `json:"pages,omitempty"`
	EntryPoint     string        `json:"entryPoint"`
}

type RouteDef struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Handler string `json:"handler"`
}

type WorkerDef struct {
	Name     string `json:"name"`
	Interval string `json:"interval"`
}

type ScheduleDef struct {
	Name string `json:"name"`
	Cron string `json:"cron"`
}

type WebhookDef struct {
	Name   string   `json:"name"`
	Events []string `json:"events"`
}

type SidebarItem struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Icon string `json:"icon"`
}

type PageDef struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Component string `json:"component"`
}

func DefaultManifest(name string) PluginManifest {
	return PluginManifest{
		SchemaVersion:  1,
		Name:           name,
		Version:        "0.1.0",
		Description:    fmt.Sprintf("%s plugin for RenOS", name),
		Author:         "Your Name <email@example.com>",
		License:        "MIT",
		MinCoreVersion: "0.1.0",
		Permissions:    []string{},
		EntryPoint:     "main.go",
	}
}

func LoadManifest(dir string) (*PluginManifest, error) {
	path := filepath.Join(dir, "plugin.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading plugin.json: %w", err)
	}
	var m PluginManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing plugin.json: %w", err)
	}
	return &m, nil
}

func (m *PluginManifest) Save(dir string) error {
	path := filepath.Join(dir, "plugin.json")
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing plugin.json: %w", err)
	}
	return nil
}

func (m *PluginManifest) Validate() []string {
	var errs []string
	if m.SchemaVersion != 1 {
		errs = append(errs, "schemaVersion must be 1")
	}
	if m.Name == "" {
		errs = append(errs, "name is required")
	}
	if !regexp.MustCompile(`^[a-z][a-z0-9_-]*$`).MatchString(m.Name) {
		errs = append(errs, "name must start with a letter and contain only lowercase letters, digits, hyphens, underscores")
	}
	if m.Version == "" {
		errs = append(errs, "version is required")
	}
	if !regexp.MustCompile(`^\d+\.\d+\.\d+`).MatchString(m.Version) {
		errs = append(errs, "version must be semver (e.g. 1.0.0)")
	}
	if m.Author == "" {
		errs = append(errs, "author is required")
	}
	if m.MinCoreVersion == "" {
		errs = append(errs, "minCoreVersion is required")
	}
	if m.EntryPoint == "" {
		errs = append(errs, "entryPoint is required")
	}
	for _, perm := range m.Permissions {
		if strings.TrimSpace(perm) == "" {
			errs = append(errs, "permissions list contains empty values")
		}
	}
	for _, r := range m.Routes {
		if r.Method == "" || r.Path == "" || r.Handler == "" {
			errs = append(errs, "route missing required fields (method, path, handler)")
		}
	}
	for _, w := range m.Workers {
		if w.Name == "" || w.Interval == "" {
			errs = append(errs, "worker missing name or interval")
		}
	}
	for _, s := range m.Schedules {
		if s.Name == "" || s.Cron == "" {
			errs = append(errs, "schedule missing name or cron")
		}
	}
	for _, w := range m.Webhooks {
		if w.Name == "" || len(w.Events) == 0 {
			errs = append(errs, "webhook missing name or events")
		}
	}
	for _, s := range m.Sidebar {
		if s.Name == "" || s.Path == "" || s.Icon == "" {
			errs = append(errs, "sidebar item missing name, path, or icon")
		}
	}
	for _, p := range m.Pages {
		if p.Name == "" || p.Path == "" || p.Component == "" {
			errs = append(errs, "page missing name, path, or component")
		}
	}
	return errs
}
