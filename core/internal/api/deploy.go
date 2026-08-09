package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/renos-project/renos/core/internal/deploy"
)

var (
	activeInstaller *deploy.Installer
	installerMu     sync.Mutex
)

func preflightCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		report, err := deploy.RunPreflightChecks(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "preflight_error", "failed to run preflight checks: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, report)
	}
}

func startInstallationHandler() http.HandlerFunc {
	type request struct {
		InstallDir    string `json:"install_dir"`
		AdminEmail    string `json:"admin_email"`
		AdminPassword string `json:"admin_password"`
		ExternalURL   string `json:"external_url"`
		HTTPPort      int    `json:"http_port"`
		LogLevel      string `json:"log_level"`
		Hostname      string `json:"hostname"`
		NonInteractive bool  `json:"non_interactive"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		installerMu.Lock()
		if activeInstaller != nil && activeInstaller.Status.Phase != "complete" && activeInstaller.Status.Phase != "failed" {
			installerMu.Unlock()
			writeError(w, http.StatusConflict, "already_installing", "an installation is already in progress")
			return
		}
		installerMu.Unlock()

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}

		opts := deploy.DefaultConfigOptions()
		if req.InstallDir != "" { opts.InstallDir = req.InstallDir }
		if req.AdminEmail != "" { opts.AdminEmail = req.AdminEmail }
		if req.AdminPassword != "" { opts.AdminPassword = req.AdminPassword }
		if req.ExternalURL != "" { opts.ExternalURL = req.ExternalURL }
		if req.HTTPPort != 0 { opts.HTTPPort = req.HTTPPort }
		if req.LogLevel != "" { opts.LogLevel = req.LogLevel }
		if req.Hostname != "" { opts.Hostname = req.Hostname }

		installer := deploy.NewInstaller(opts, req.NonInteractive)

		installerMu.Lock()
		activeInstaller = installer
		installerMu.Unlock()

		go installer.Run(context.Background())

		writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
	}
}

func installationStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		installerMu.Lock()
		inst := activeInstaller
		installerMu.Unlock()

		if inst == nil {
			writeJSON(w, http.StatusOK, deploy.InstallationStatus{Phase: "idle", Message: "no installation in progress"})
			return
		}
		writeJSON(w, http.StatusOK, inst.Status)
	}
}

func installationStreamHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		installerMu.Lock()
		inst := activeInstaller
		installerMu.Unlock()

		if inst == nil {
			writeError(w, http.StatusNotFound, "not_found", "no active installation")
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeError(w, http.StatusInternalServerError, "unsupported", "streaming not supported")
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		for {
			select {
			case <-r.Context().Done():
				return
			case status, ok := <-inst.Result:
				if !ok { return }
				data, _ := json.Marshal(status)
				w.Write([]byte("data: " + string(data) + "\n\n"))
				flusher.Flush()
				if status.Phase == "complete" || status.Phase == "failed" {
					return
				}
		case <-time.After(30 * time.Second):
			w.Write([]byte("data: {\"keepalive\":true}\n\n"))
			flusher.Flush()
			}
		}
	}
}

func rollbackHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type request struct {
			InstallDir string `json:"install_dir"`
		}
		var req request
		json.NewDecoder(r.Body).Decode(&req)
		installDir := req.InstallDir
		if installDir == "" { installDir = "/opt/renos" }

		deploy.Rollback(installDir)
		writeJSON(w, http.StatusOK, map[string]string{"status": "rolled_back"})
	}
}

func systemInfoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		info := map[string]interface{}{
			"os":          detectOS(),
			"arch":        detectArch(),
			"hostname":    mustHostname(),
			"cpu_cores":   detectCPUCores(),
			"docker":      detectDocker(),
			"docker_compose": detectDockerCompose(),
		}
		writeJSON(w, http.StatusOK, info)
	}
}

func detectOS() string {
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
			}
		}
	}
	return "unknown"
}

func detectArch() string { return runtime.GOARCH }

func mustHostname() string {
	h, _ := os.Hostname()
	return h
}

func detectCPUCores() int { return runtime.NumCPU() }

func detectDocker() string {
	out, err := exec.Command("docker", "--version").CombinedOutput()
	if err != nil { return "not installed" }
	return strings.TrimSpace(string(out))
}

func detectDockerCompose() string {
	out, err := exec.Command("docker", "compose", "version").CombinedOutput()
	if err != nil {
		out, err = exec.Command("docker-compose", "--version").CombinedOutput()
		if err != nil { return "not installed" }
	}
	return strings.TrimSpace(string(out))
}
