package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type CheckStatus string

const (
	StatusPass    CheckStatus = "pass"
	StatusWarn    CheckStatus = "warn"
	StatusFail    CheckStatus = "fail"
	StatusSkipped CheckStatus = "skipped"
)

type CheckResult struct {
	Name    string      `json:"name"`
	Category string     `json:"category"`
	Status  CheckStatus `json:"status"`
	Message string      `json:"message"`
	Details string      `json:"details,omitempty"`
}

type PreflightReport struct {
	OS          string        `json:"os"`
	Arch        string        `json:"arch"`
	Hostname    string        `json:"hostname"`
	Timestamp   time.Time     `json:"timestamp"`
	Results     []CheckResult `json:"results"`
	PassCount   int           `json:"pass_count"`
	WarnCount   int           `json:"warn_count"`
	FailCount   int           `json:"fail_count"`
	AllPassed   bool          `json:"all_passed"`
}

func RunPreflightChecks(ctx context.Context) (*PreflightReport, error) {
	hostname, _ := os.Hostname()
	report := &PreflightReport{
		OS:        detectOS(),
		Arch:      runtime.GOARCH,
		Hostname:  hostname,
		Timestamp: time.Now(),
	}

	checks := []func(context.Context) CheckResult{
		checkDocker,
		checkDockerCompose,
		checkDockerDaemon,
		checkCPU,
		checkRAM,
		checkDisk,
		checkKernel,
		checkFirewall,
		checkVirtualization,
		checkNetworking,
		checkDNS,
		checkPorts,
		checkSELinux,
		checkSystemd,
		checkOpenFiles,
		checkUser,
	}

	for _, check := range checks {
		result := check(ctx)
		report.Results = append(report.Results, result)
		switch result.Status {
		case StatusPass:
			report.PassCount++
		case StatusWarn:
			report.WarnCount++
		case StatusFail:
			report.FailCount++
		}
	}

	report.AllPassed = report.FailCount == 0
	return report, nil
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
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}

func runCmd(ctx context.Context, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func checkDocker(ctx context.Context) CheckResult {
	out, err := runCmd(ctx, "docker", "--version")
	if err != nil {
		return CheckResult{Name: "Docker", Category: "docker", Status: StatusFail, Message: "Docker is not installed", Details: "Install Docker from https://docs.docker.com/get-docker/"}
	}
	return CheckResult{Name: "Docker", Category: "docker", Status: StatusPass, Message: out}
}

func checkDockerCompose(ctx context.Context) CheckResult {
	out, err := runCmd(ctx, "docker", "compose", "version")
	if err != nil {
		out, err = runCmd(ctx, "docker-compose", "--version")
		if err != nil {
			return CheckResult{Name: "Docker Compose", Category: "docker", Status: StatusFail, Message: "Docker Compose is not installed"}
		}
	}
	return CheckResult{Name: "Docker Compose", Category: "docker", Status: StatusPass, Message: out}
}

func checkDockerDaemon(ctx context.Context) CheckResult {
	out, err := runCmd(ctx, "docker", "info")
	if err != nil {
		return CheckResult{Name: "Docker Daemon", Category: "docker", Status: StatusFail, Message: "Docker daemon is not running", Details: "Start Docker: systemctl start docker"}
	}
	if strings.Contains(out, "Server Version") {
		return CheckResult{Name: "Docker Daemon", Category: "docker", Status: StatusPass, Message: "Docker daemon is running"}
	}
	return CheckResult{Name: "Docker Daemon", Category: "docker", Status: StatusFail, Message: "Docker daemon not responding"}
}

func checkCPU(ctx context.Context) CheckResult {
	cpus := runtime.NumCPU()
	if cpus < 2 {
		return CheckResult{Name: "CPU", Category: "hardware", Status: StatusFail, Message: fmt.Sprintf("Only %d CPU core(s) available", cpus), Details: "Minimum 2 cores required"}
	}
	if cpus < 4 {
		return CheckResult{Name: "CPU", Category: "hardware", Status: StatusWarn, Message: fmt.Sprintf("%d CPU core(s) available (4+ recommended)", cpus)}
	}
	return CheckResult{Name: "CPU", Category: "hardware", Status: StatusPass, Message: fmt.Sprintf("%d CPU core(s) available", cpus)}
}

func checkRAM(ctx context.Context) CheckResult {
	out, err := runCmd(ctx, "free", "-m")
	if err != nil {
		return CheckResult{Name: "RAM", Category: "hardware", Status: StatusSkipped, Message: "Could not detect RAM"}
	}
	var totalMB int
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "Mem:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				fmt.Sscanf(fields[1], "%d", &totalMB)
			}
			break
		}
	}
	if totalMB < 1024 {
		return CheckResult{Name: "RAM", Category: "hardware", Status: StatusFail, Message: fmt.Sprintf("Only %dMB RAM available", totalMB), Details: "Minimum 1GB required"}
	}
	if totalMB < 2048 {
		return CheckResult{Name: "RAM", Category: "hardware", Status: StatusWarn, Message: fmt.Sprintf("%dMB RAM available (2GB+ recommended)", totalMB)}
	}
	return CheckResult{Name: "RAM", Category: "hardware", Status: StatusPass, Message: fmt.Sprintf("%dMB RAM available", totalMB)}
}

func checkDisk(ctx context.Context) CheckResult {
	out, err := runCmd(ctx, "df", "-BM", "/")
	if err != nil {
		return CheckResult{Name: "Disk", Category: "hardware", Status: StatusSkipped, Message: "Could not detect disk space"}
	}
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		return CheckResult{Name: "Disk", Category: "hardware", Status: StatusSkipped, Message: "Could not parse disk info"}
	}
	fields := strings.Fields(lines[1])
	var availMB int
	if len(fields) >= 4 {
		fmt.Sscanf(fields[3], "%d", &availMB)
	}
	if availMB < 5120 {
		return CheckResult{Name: "Disk", Category: "hardware", Status: StatusFail, Message: fmt.Sprintf("Only %dMB disk space available", availMB), Details: "Minimum 5GB free required"}
	}
	if availMB < 10240 {
		return CheckResult{Name: "Disk", Category: "hardware", Status: StatusWarn, Message: fmt.Sprintf("%dMB disk space available (10GB+ recommended)", availMB)}
	}
	return CheckResult{Name: "Disk", Category: "hardware", Status: StatusPass, Message: fmt.Sprintf("%dMB disk space available", availMB)}
}

func checkKernel(ctx context.Context) CheckResult {
	out, err := runCmd(ctx, "uname", "-r")
	if err != nil {
		return CheckResult{Name: "Kernel", Category: "system", Status: StatusSkipped, Message: "Could not detect kernel version"}
	}
	return CheckResult{Name: "Kernel", Category: "system", Status: StatusPass, Message: out}
}

func checkFirewall(ctx context.Context) CheckResult {
	// Check for common firewalls
	for _, fw := range []string{"ufw", "firewall-cmd", "iptables"} {
		if _, err := runCmd(ctx, "which", fw); err == nil {
			return CheckResult{Name: "Firewall", Category: "network", Status: StatusWarn, Message: fmt.Sprintf("%s detected — ensure ports 8443, 5432, 4222, 6379 are open", fw)}
		}
	}
	return CheckResult{Name: "Firewall", Category: "network", Status: StatusPass, Message: "No firewall detected (ports should be accessible)"}
}

func checkVirtualization(ctx context.Context) CheckResult {
	out, err := runCmd(ctx, "systemd-detect-virt")
	if err != nil || out == "none" {
		return CheckResult{Name: "Virtualization", Category: "system", Status: StatusPass, Message: "Running on bare metal or undetected virtualization"}
	}
	return CheckResult{Name: "Virtualization", Category: "system", Status: StatusPass, Message: fmt.Sprintf("Running in %s", out)}
}

func checkNetworking(ctx context.Context) CheckResult {
	out, err := runCmd(ctx, "ip", "route", "show", "default")
	if err != nil {
		return CheckResult{Name: "Networking", Category: "network", Status: StatusWarn, Message: "Could not detect default route"}
	}
	return CheckResult{Name: "Networking", Category: "network", Status: StatusPass, Message: "Default route found: " + out}
}

func checkDNS(ctx context.Context) CheckResult {
	out, err := runCmd(ctx, "nslookup", "github.com")
	if err != nil {
		return CheckResult{Name: "DNS", Category: "network", Status: StatusWarn, Message: "DNS resolution may not work", Details: "External access may be limited"}
	}
	if strings.Contains(out, "Address") {
		return CheckResult{Name: "DNS", Category: "network", Status: StatusPass, Message: "DNS resolution working"}
	}
	return CheckResult{Name: "DNS", Category: "network", Status: StatusWarn, Message: "DNS resolution uncertain"}
}

func checkPorts(ctx context.Context) CheckResult {
	ports := []int{8443, 5432, 4222, 6379}
	var conflicts []string
	for _, port := range ports {
		out, err := runCmd(ctx, "sh", "-c", fmt.Sprintf("ss -tlnp 'sport = :%d' 2>/dev/null || netstat -tlnp 2>/dev/null | grep ':%d '", port, port))
		if err == nil && strings.Contains(out, fmt.Sprintf(":%d", port)) {
			conflicts = append(conflicts, fmt.Sprintf(":%d", port))
		}
	}
	if len(conflicts) > 0 {
		return CheckResult{Name: "Port Availability", Category: "network", Status: StatusWarn, Message: fmt.Sprintf("Ports in use: %s — may conflict with RenOS", strings.Join(conflicts, ", "))}
	}
	return CheckResult{Name: "Port Availability", Category: "network", Status: StatusPass, Message: "All required ports are available"}
}

func checkSELinux(ctx context.Context) CheckResult {
	out, err := runCmd(ctx, "getenforce")
	if err != nil {
		return CheckResult{Name: "SELinux", Category: "system", Status: StatusPass, Message: "SELinux not present"}
	}
	if strings.EqualFold(strings.TrimSpace(out), "Enforcing") {
		return CheckResult{Name: "SELinux", Category: "system", Status: StatusWarn, Message: "SELinux is enforcing — may interfere with Docker volumes", Details: "Consider: setenforce 0 or configure SELinux policies"}
	}
	return CheckResult{Name: "SELinux", Category: "system", Status: StatusPass, Message: "SELinux is " + strings.TrimSpace(out)}
}

func checkSystemd(ctx context.Context) CheckResult {
	if _, err := runCmd(ctx, "systemctl", "--version"); err != nil {
		return CheckResult{Name: "Systemd", Category: "system", Status: StatusWarn, Message: "Systemd not found — service management may differ"}
	}
	return CheckResult{Name: "Systemd", Category: "system", Status: StatusPass, Message: "Systemd available"}
}

func checkOpenFiles(ctx context.Context) CheckResult {
	out, err := runCmd(ctx, "sh", "-c", "ulimit -n")
	if err != nil {
		return CheckResult{Name: "Open Files Limit", Category: "system", Status: StatusSkipped, Message: "Could not determine open files limit"}
	}
	var limit int
	fmt.Sscanf(out, "%d", &limit)
	if limit < 65535 {
		return CheckResult{Name: "Open Files Limit", Category: "system", Status: StatusWarn, Message: fmt.Sprintf("Open files limit: %d (65535+ recommended)", limit)}
	}
	return CheckResult{Name: "Open Files Limit", Category: "system", Status: StatusPass, Message: fmt.Sprintf("Open files limit: %d", limit)}
}

func checkUser(ctx context.Context) CheckResult {
	out, err := runCmd(ctx, "id", "-u")
	if err != nil {
		return CheckResult{Name: "User", Category: "system", Status: StatusSkipped, Message: "Could not determine current user"}
	}
	if strings.TrimSpace(out) == "0" {
		return CheckResult{Name: "User", Category: "system", Status: StatusPass, Message: "Running as root"}
	}
	return CheckResult{Name: "User", Category: "system", Status: StatusWarn, Message: "Not running as root — some operations may need sudo"}
}
