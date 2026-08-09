package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Template struct {
	Name        string
	Description string
	Type        string
}

var templates = []Template{
	{Name: "dashboard-page", Description: "A TypeScript React dashboard page component", Type: "dashboard-page"},
	{Name: "api-route", Description: "A Go HTTP handler with auth middleware", Type: "api-route"},
	{Name: "worker", Description: "A background worker with NATS subscription", Type: "worker"},
	{Name: "schedule", Description: "A cron-based scheduled task", Type: "schedule"},
	{Name: "sidebar-item", Description: "A navigation item for the RenOS hub UI", Type: "sidebar-item"},
	{Name: "webhook", Description: "A webhook receiver with signature verification", Type: "webhook"},
	{Name: "workflow-action", Description: "A custom workflow step action", Type: "workflow-action"},
}

func listTemplates() {
	fmt.Println("Available templates:")
	for _, t := range templates {
		fmt.Printf("  %-20s %s\n", t.Name, t.Description)
	}
}

func findTemplate(name string) *Template {
	for _, t := range templates {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

func GenerateTemplate(tmplType, pluginDir string) error {
	tmpl := findTemplate(tmplType)
	if tmpl == nil {
		return fmt.Errorf("unknown template type %q", tmplType)
	}

	manifest, err := LoadManifest(pluginDir)
	if err != nil {
		return fmt.Errorf("loading plugin manifest: %w", err)
	}

	switch tmplType {
	case "dashboard-page":
		return generateDashboardPage(pluginDir)
	case "api-route":
		return generateAPIRoute(pluginDir, manifest)
	case "worker":
		return generateWorker(pluginDir, manifest)
	case "schedule":
		return generateSchedule(pluginDir, manifest)
	case "sidebar-item":
		return generateSidebarItem(pluginDir, manifest)
	case "webhook":
		return generateWebhook(pluginDir, manifest)
	case "workflow-action":
		return generateWorkflowAction(pluginDir, manifest)
	}
	return nil
}

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func generateDashboardPage(pluginDir string) error {
	dir := filepath.Join(pluginDir, "frontend", "pages")
	if err := ensureDir(dir); err != nil {
		return err
	}

	content := `import { useEffect, useState } from "react";

interface DashboardData {
  title: string;
  description: string;
  stats: { label: string; value: number }[];
}

export default function DashboardPage() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch("/api/plugin/dashboard")
      .then((res) => {
        if (!res.ok) throw new Error("Failed to fetch dashboard data");
        return res.json();
      })
      .then(setData)
      .catch((err) => setError(err.message));
  }, []);

  if (error) {
    return <div className="dashboard-error">Error: {error}</div>;
  }

  if (!data) {
    return <div className="dashboard-loading">Loading...</div>;
  }

  return (
    <div className="dashboard">
      <h1>{data.title}</h1>
      <p>{data.description}</p>
      <div className="dashboard-stats">
        {data.stats.map((stat, i) => (
          <div key={i} className="stat-card">
            <span className="stat-label">{stat.label}</span>
            <span className="stat-value">{stat.value}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
`
	path := filepath.Join(dir, "Dashboard.tsx")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	fmt.Println("  Created frontend/pages/Dashboard.tsx")
	return nil
}

func generateAPIRoute(pluginDir string, manifest *PluginManifest) error {
	dir := filepath.Join(pluginDir, "routes")
	if err := ensureDir(dir); err != nil {
		return err
	}

	content := fmt.Sprintf(`package routes

import (
	"encoding/json"
	"net/http"
	"strings"
)

type contextKey string

const userKey contextKey = "user"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, `+"`"+`{"error":"unauthorized"}`+"`"+`, http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" {
			http.Error(w, `+"`"+`{"error":"invalid token"}`+"`"+`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(
			r.Context(),
		))
	})
}

func ExampleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Hello from %s plugin",
		"status":  "ok",
	})
}
`, manifest.Name)

	path := filepath.Join(dir, "example.go")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	fmt.Println("  Created routes/example.go")

	addRoute := RouteDef{
		Method:  "GET",
		Path:    "/api/plugins/" + manifest.Name + "/example",
		Handler: "routes.ExampleHandler",
	}
	manifest.Routes = append(manifest.Routes, addRoute)
	if err := manifest.Save(pluginDir); err != nil {
		return err
	}
	fmt.Println("  Updated plugin.json with new route")
	return nil
}

func generateWorker(pluginDir string, manifest *PluginManifest) error {
	dir := filepath.Join(pluginDir, "worker")
	if err := ensureDir(dir); err != nil {
		return err
	}

	name := "example"
	content := `package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
)

type TaskPayload struct {
	ID      string ` + "`" + `json:"id"` + "`" + `
	Type    string ` + "`" + `json:"type"` + "`" + `
	Payload json.RawMessage ` + "`" + `json:"payload"` + "`" + `
}

func Start(conn *nats.Conn) error {
	sub, err := conn.Subscribe("plugin.tasks." + ` + fmt.Sprintf("%q", name) + `, func(msg *nats.Msg) {
		var task TaskPayload
		if err := json.Unmarshal(msg.Data, &task); err != nil {
			log.Printf("failed to parse task: %v", err)
			return
		}
		log.Printf("processing task %s of type %s", task.ID, task.Type)
		if err := msg.Respond([]byte(` + "`" + `{"status":"done"}` + "`" + `)); err != nil {
			log.Printf("failed to respond: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("subscribing to tasks: %w", err)
	}
	defer sub.Unsubscribe()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	return nil
}
`
	path := filepath.Join(dir, fmt.Sprintf("%s.go", name))
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	fmt.Printf("  Created worker/%s.go\n", name)

	addWorker := WorkerDef{
		Name:     name,
		Interval: "10s",
	}
	manifest.Workers = append(manifest.Workers, addWorker)
	if err := manifest.Save(pluginDir); err != nil {
		return err
	}
	fmt.Println("  Updated plugin.json with new worker")
	return nil
}

func generateSchedule(pluginDir string, manifest *PluginManifest) error {
	dir := filepath.Join(pluginDir, "schedules")
	if err := ensureDir(dir); err != nil {
		return err
	}

	name := "dailyTask"
	content := fmt.Sprintf(`package schedules

import (
	"log"
	"time"
)

type TaskFunc func() error

var tasks = map[string]TaskFunc{
	%q: DailyTask,
}

func DailyTask() error {
	log.Println("running daily scheduled task at", time.Now().Format(time.RFC3339))
	return nil
}
`, name)

	path := filepath.Join(dir, "tasks.go")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	fmt.Println("  Created schedules/tasks.go")

	addSchedule := ScheduleDef{
		Name: name,
		Cron: "0 0 * * *",
	}
	manifest.Schedules = append(manifest.Schedules, addSchedule)
	if err := manifest.Save(pluginDir); err != nil {
		return err
	}
	fmt.Println("  Updated plugin.json with new schedule")
	return nil
}

func generateSidebarItem(pluginDir string, manifest *PluginManifest) error {
	dir := filepath.Join(pluginDir, "frontend", "components")
	if err := ensureDir(dir); err != nil {
		return err
	}

	name := "example"
	content := `import { NavLink } from "react-router-dom";

interface SidebarItemProps {
  collapsed: boolean;
}

export default function SidebarItem({ collapsed }: SidebarItemProps) {
  return (
    <li className="sidebar-item">
      <NavLink
        to="/plugin/` + manifest.Name + `"
        className={({ isActive }) =>
          ` + "`sidebar-link ${isActive ? 'active' : ''}`" + `
        }
        title="` + name + `"
      >
        <span className="sidebar-icon">
          <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
            <path d="M10 2a8 8 0 100 16 8 8 0 000-16zM9 6a1 1 0 112 0v2a1 1 0 11-2 0V6zm1 8a1.5 1.5 0 100-3 1.5 1.5 0 000 3z" />
          </svg>
        </span>
        {!collapsed && <span className="sidebar-label">` + name + `</span>}
      </NavLink>
    </li>
  );
}
`
	path := filepath.Join(dir, "SidebarItem.tsx")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	fmt.Println("  Created frontend/components/SidebarItem.tsx")

	addSidebar := SidebarItem{
		Name: name,
		Path: "/plugin/" + manifest.Name,
		Icon: "information-circle",
	}
	manifest.Sidebar = append(manifest.Sidebar, addSidebar)
	if err := manifest.Save(pluginDir); err != nil {
		return err
	}
	fmt.Println("  Updated plugin.json with new sidebar item")
	return nil
}

func generateWebhook(pluginDir string, manifest *PluginManifest) error {
	dir := filepath.Join(pluginDir, "webhooks")
	if err := ensureDir(dir); err != nil {
		return err
	}

	name := "example"
	content := `package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

const signatureHeader = "X-RenOS-Signature"

func VerifySignature(payload []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expected))
}

func HandleExampleWebhook(secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		sig := r.Header.Get(signatureHeader)
		if sig == "" || !VerifySignature(body, sig, secret) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		var event struct {
			Type    string          ` + "`" + `json:"type"` + "`" + `
			Payload json.RawMessage ` + "`" + `json:"payload"` + "`" + `
		}
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, "invalid event payload", http.StatusBadRequest)
			return
		}

		log.Printf("received webhook event: %s", event.Type)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "received"})
	}
}
`
	path := filepath.Join(dir, fmt.Sprintf("%s.go", name))
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	fmt.Printf("  Created webhooks/%s.go\n", name)

	addWebhook := WebhookDef{
		Name:   name,
		Events: []string{"example.event"},
	}
	manifest.Webhooks = append(manifest.Webhooks, addWebhook)
	if err := manifest.Save(pluginDir); err != nil {
		return err
	}
	fmt.Println("  Updated plugin.json with new webhook")
	return nil
}

func generateWorkflowAction(pluginDir string, manifest *PluginManifest) error {
	dir := filepath.Join(pluginDir, "actions")
	if err := ensureDir(dir); err != nil {
		return err
	}

	name := "example"
	title := strings.Title(name)
	content := fmt.Sprintf(`package actions

import (
	"encoding/json"
	"errors"
	"fmt"
)

type ActionInput struct {
	Data  map[string]interface{} ` + "`" + `json:"data"` + "`" + `
	Context map[string]interface{} ` + "`" + `json:"context"` + "`" + `
}

type ActionOutput struct {
	Success bool                   ` + "`" + `json:"success"` + "`" + `
	Data    map[string]interface{} ` + "`" + `json:"data"` + "`" + `
	Error   string                 ` + "`" + `json:"error,omitempty"` + "`" + `
}

type Action struct {
	Name        string
	Description string
	Execute     func(input ActionInput) (ActionOutput, error)
}

var %sAction = Action{
	Name:        "%s",
	Description: "Example workflow action for %s plugin",
	Execute: func(input ActionInput) (ActionOutput, error) {
		if input.Data == nil {
			return ActionOutput{}, errors.New("input data is required")
		}
		result := make(map[string]interface{})
		result["processed"] = true
		result["original"] = input.Data
		return ActionOutput{
			Success: true,
			Data:    result,
		}, nil
	},
}

func GetActions() []Action {
	return []Action{%sAction}
}
`, title, name, manifest.Name, title)

	path := filepath.Join(dir, fmt.Sprintf("%s.go", name))
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	fmt.Printf("  Created actions/%s.go\n", name)
	return nil
}

func generateTemplateFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := ensureDir(dir); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	return nil
}
