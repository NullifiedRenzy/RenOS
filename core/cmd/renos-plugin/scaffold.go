package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ScaffoldPlugin(name string) error {
	pluginDir := name
	if _, err := os.Stat(pluginDir); err == nil {
		return fmt.Errorf("directory %q already exists", pluginDir)
	}
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("creating plugin directory: %w", err)
	}

	manifest := DefaultManifest(name)

	if err := scaffoldManifest(pluginDir, &manifest); err != nil {
		return err
	}
	if err := scaffoldMainGo(pluginDir, &manifest); err != nil {
		return err
	}
	if err := scaffoldRoutes(pluginDir, &manifest); err != nil {
		return err
	}
	if err := scaffoldFrontend(pluginDir); err != nil {
		return err
	}
	if err := scaffoldWorkerDir(pluginDir); err != nil {
		return err
	}
	if err := scaffoldSchedulesDir(pluginDir); err != nil {
		return err
	}
	if err := scaffoldReadme(pluginDir, &manifest); err != nil {
		return err
	}
	if err := scaffoldMakefile(pluginDir, &manifest); err != nil {
		return err
	}
	if err := scaffoldGoMod(pluginDir, &manifest); err != nil {
		return err
	}

	fmt.Printf("\nPlugin %q scaffold created at %s/\n", name, pluginDir)
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", pluginDir)
	fmt.Println("  go mod tidy")
	fmt.Println("  go build -o plugin .")
	fmt.Println("  renos-plugin validate")
	fmt.Println("  renos-plugin build")
	return nil
}

func scaffoldManifest(dir string, m *PluginManifest) error {
	return m.Save(dir)
}

func scaffoldMainGo(dir string, m *PluginManifest) error {
	content := fmt.Sprintf(`package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting %s plugin v%s", %q, %q)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/plugin/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":    %q,
			"version": %q,
			"status":  "running",
		})
	})

	mux.HandleFunc("/api/plugin/dashboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"title":       %q,
			"description": %q,
			"stats": []map[string]interface{}{
				{"label": "Requests", "value": 0},
				{"label": "Active", "value": 1},
				{"label": "Uptime", "value": "0s"},
			},
		})
	})

	port := os.Getenv("PLUGIN_PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		log.Println("Plugin server listening on :" + port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %%v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("Shutting down plugin server...")
}
`, m.Name, m.Version, m.Name, m.Version, m.Name, m.Version, m.Name, m.Version)

	path := filepath.Join(dir, "main.go")
	return os.WriteFile(path, []byte(content), 0644)
}

func scaffoldRoutes(dir string, m *PluginManifest) error {
	routeDir := filepath.Join(dir, "routes")
	if err := os.MkdirAll(routeDir, 0755); err != nil {
		return err
	}

	content := `package routes

import (
	"encoding/json"
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/plugin/` + m.Name + `/hello", HelloHandler)
}

func HelloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Hello from ` + m.Name + ` plugin",
	})
}
`
	path := filepath.Join(routeDir, "routes.go")
	return os.WriteFile(path, []byte(content), 0644)
}

func scaffoldFrontend(dir string) error {
	frontendDir := filepath.Join(dir, "frontend")
	if err := os.MkdirAll(frontendDir, 0755); err != nil {
		return err
	}

	pagesDir := filepath.Join(frontendDir, "pages")
	if err := os.MkdirAll(pagesDir, 0755); err != nil {
		return err
	}

	componentsDir := filepath.Join(frontendDir, "components")
	if err := os.MkdirAll(componentsDir, 0755); err != nil {
		return err
	}

	pageContent := `import { useEffect, useState } from "react";

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

  if (error) return <div className="p-4 text-red-500">Error: {error}</div>;
  if (!data) return <div className="p-4 text-gray-400">Loading...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">{data.title}</h1>
      <p className="text-gray-600 mb-6">{data.description}</p>
      <div className="grid grid-cols-3 gap-4">
        {data.stats.map((stat, i) => (
          <div key={i} className="rounded-lg border bg-card p-4 shadow-sm">
            <div className="text-sm text-muted-foreground">{stat.label}</div>
            <div className="text-3xl font-bold">{stat.value}</div>
          </div>
        ))}
      </div>
    </div>
  );
}
`
	if err := os.WriteFile(filepath.Join(pagesDir, "Dashboard.tsx"), []byte(pageContent), 0644); err != nil {
		return err
	}

	componentContent := `export default function PluginLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="plugin-container">
      <aside className="plugin-sidebar">
        <nav>
          <ul>
            <li>
              <a href="/plugin/` + filepath.Base(dir) + `">Dashboard</a>
            </li>
          </ul>
        </nav>
      </aside>
      <main className="plugin-content">{children}</main>
    </div>
  );
}
`
	if err := os.WriteFile(filepath.Join(componentsDir, "Layout.tsx"), []byte(componentContent), 0644); err != nil {
		return err
	}

	return nil
}

func scaffoldWorkerDir(dir string) error {
	workerDir := filepath.Join(dir, "worker")
	if err := os.MkdirAll(workerDir, 0755); err != nil {
		return err
	}
	content := `package worker

import (
	"log"
	"time"
)

type Worker struct {
	Name     string
	Interval time.Duration
	Run      func() error
}

func StartWorkers(workers []Worker, stop <-chan struct{}) {
	for _, w := range workers {
		go func(w Worker) {
			ticker := time.NewTicker(w.Interval)
			defer ticker.Stop()
			log.Printf("worker %q started (interval: %s)", w.Name, w.Interval)
			for {
				select {
				case <-ticker.C:
					if err := w.Run(); err != nil {
						log.Printf("worker %q error: %v", w.Name, err)
					}
				case <-stop:
					log.Printf("worker %q stopped", w.Name)
					return
				}
			}
		}(w)
	}
}
`
	return os.WriteFile(filepath.Join(workerDir, "worker.go"), []byte(content), 0644)
}

func scaffoldSchedulesDir(dir string) error {
	schedDir := filepath.Join(dir, "schedules")
	if err := os.MkdirAll(schedDir, 0755); err != nil {
		return err
	}
	content := `package schedules

import (
	"log"
	"time"
)

type ScheduledTask struct {
	Name string
	Cron string
	Run  func() error
}

func parseCron(cron string) (time.Duration, error) {
	return time.Hour, nil
}

func StartScheduler(tasks []ScheduledTask, stop <-chan struct{}) {
	for _, t := range tasks {
		go func(t ScheduledTask) {
			interval, err := parseCron(t.Cron)
			if err != nil {
				log.Printf("task %q: invalid cron %q: %v", t.Name, t.Cron, err)
				return
			}
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			log.Printf("scheduler %q started (cron: %s)", t.Name, t.Cron)
			for {
				select {
				case <-ticker.C:
					if err := t.Run(); err != nil {
						log.Printf("task %q error: %v", t.Name, err)
					}
				case <-stop:
					log.Printf("scheduler %q stopped", t.Name)
					return
				}
			}
		}(t)
	}
}
`
	return os.WriteFile(filepath.Join(schedDir, "scheduler.go"), []byte(content), 0644)
}

func scaffoldReadme(dir string, m *PluginManifest) error {
	content := fmt.Sprintf("# %s Plugin\n\n%s\n\n## Installation\n\n```bash\nrenos-plugin install %s\n```\n\n## Development\n\n### Prerequisites\n\n- Go 1.21+\n- Node.js 18+ (for frontend development)\n- RenOS Core running\n\n### Build\n\n```bash\nmake build\n```\n\n### Test\n\n```bash\nmake test\n```\n\n### Package\n\n```bash\nmake package\n```\n\n## Configuration\n\n| Variable       | Default  | Description       |\n|----------------|----------|-------------------|\n| PLUGIN_PORT    | 8080     | HTTP server port  |\n\n## Permissions\n\n%s\n\n## License\n\n%s\n", m.Name, m.Description, m.Name, formatPermissions(m.Permissions), m.License)

	path := filepath.Join(dir, "README.md")
	return os.WriteFile(path, []byte(content), 0644)
}

func formatPermissions(perms []string) string {
	if len(perms) == 0 {
		return "None required"
	}
	return "- " + strings.Join(perms, "\n- ")
}

func scaffoldMakefile(dir string, m *PluginManifest) error {
	content := fmt.Sprintf(`.PHONY: build test package clean

NAME = %s
BUILD_DIR = dist

build:
	@echo "Building $(NAME)..."
	go build -o $(BUILD_DIR)/$(NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(NAME)"

test:
	@echo "Running tests..."
	go test ./... -v
	@echo "Tests complete."

package: build
	@echo "Packaging $(NAME)..."
	mkdir -p $(BUILD_DIR)
	cp plugin.json $(BUILD_DIR)/
	cp -r frontend $(BUILD_DIR)/ 2>/dev/null || true
	cp -r worker $(BUILD_DIR)/ 2>/dev/null || true
	cp -r schedules $(BUILD_DIR)/ 2>/dev/null || true
	cd $(BUILD_DIR) && zip -r ../$(NAME).rpk ./*
	@echo "Package created: $(NAME).rpk"

clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f $(NAME).rpk
	@echo "Clean complete."
`, m.Name)

	path := filepath.Join(dir, "Makefile")
	return os.WriteFile(path, []byte(content), 0644)
}

func scaffoldGoMod(dir string, m *PluginManifest) error {
	content := fmt.Sprintf(`module %s

go 1.25.0
`, m.Name)

	path := filepath.Join(dir, "go.mod")
	return os.WriteFile(path, []byte(content), 0644)
}
