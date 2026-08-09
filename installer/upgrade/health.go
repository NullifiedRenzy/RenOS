package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type HealthStatus struct {
	Service   string `json:"service"`
	Healthy   bool   `json:"healthy"`
	Message   string `json:"message,omitempty"`
	Component string `json:"component"`
}

type HealthReport struct {
	Statuses []HealthStatus `json:"statuses"`
	AllOK    bool           `json:"all_ok"`
	Errors   []string       `json:"errors,omitempty"`
}

var healthEndpoints = map[string]string{
	"core":  "http://localhost:8000/api/v1/healthz",
	"hub":   "http://localhost:8100/api/v1/healthz",
	"agent": "http://localhost:8200/api/v1/healthz",
}

var serviceNames = map[string]string{
	"core":  "renos-core",
	"hub":   "renos-hub",
	"agent": "renos-agent",
}

func CheckServiceHealth(serviceName string) HealthStatus {
	status := HealthStatus{
		Service: serviceName,
		Healthy: false,
	}

	for _, method := range []string{"systemctl", "docker"} {
		switch method {
		case "systemctl":
			cmd := exec.Command("systemctl", "is-active", serviceName)
			output, err := cmd.CombinedOutput()
			if err == nil && strings.TrimSpace(string(output)) == "active" {
				status.Healthy = true
				status.Message = "systemd: active"
				return status
			}
		case "docker":
			cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", serviceName),
				"--format", "{{.Status}}")
			output, err := cmd.CombinedOutput()
			if err == nil && strings.Contains(string(output), "Up") {
				status.Healthy = true
				status.Message = "docker: running"
				return status
			}
		}
	}

	status.Message = "not running (checked systemd, docker)"
	return status
}

func CheckAPIHealth(url string) HealthStatus {
	status := HealthStatus{
		Service: url,
		Healthy: false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		status.Message = fmt.Sprintf("create request: %v", err)
		return status
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		status.Message = fmt.Sprintf("request failed: %v", err)
		return status
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			if ok, exists := result["ok"]; exists {
				if b, ok := ok.(bool); ok && b {
					status.Healthy = true
					status.Message = "health endpoint OK"
					return status
				}
			}
		}
		status.Healthy = true
		status.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	} else {
		status.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return status
}

func CheckDatabase() HealthStatus {
	status := HealthStatus{
		Service: "database",
		Healthy: false,
	}

	if _, err := exec.LookPath("psql"); err != nil {
		status.Message = "postgres client not installed"
		return status
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "psql", "-d", "renos", "-c", "SELECT 1")
	if output, err := cmd.CombinedOutput(); err == nil {
		status.Healthy = true
		status.Message = "postgres: connected"
	} else {
		status.Message = fmt.Sprintf("postgres: %s", strings.TrimSpace(string(output)))
	}

	return status
}

func AllHealthChecks() *HealthReport {
	report := &HealthReport{AllOK: true}

	for component, url := range healthEndpoints {
		status := CheckAPIHealth(url)
		status.Component = component
		report.Statuses = append(report.Statuses, status)
		if !status.Healthy {
			report.AllOK = false
		}
	}

	for component, svc := range serviceNames {
		status := CheckServiceHealth(svc)
		status.Component = component
		report.Statuses = append(report.Statuses, status)
		if !status.Healthy {
			report.AllOK = false
		}
	}

	dbStatus := CheckDatabase()
	dbStatus.Component = "database"
	report.Statuses = append(report.Statuses, dbStatus)
	if !dbStatus.Healthy {
		report.AllOK = false
	}

	return report
}
