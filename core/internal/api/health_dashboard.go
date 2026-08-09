package api

import (
	"bufio"
	"context"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Health check statuses. Each maps to a numeric score used in the weighted
// health score.
const (
	healthHealthy       = "healthy"
	healthDegraded      = "degraded"
	healthWarning       = "warning"
	healthNotConfigured = "not_configured"
	healthDisabled      = "disabled"
	healthUnknown       = "unknown"
	healthCritical      = "critical"
	healthUnhealthy     = "unhealthy"
)

// healthCheckResult describes a single subsystem's health.
type healthCheckResult struct {
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
}

// healthDashboard is the payload returned by the health dashboard endpoint.
type healthDashboard struct {
	Score  int                          `json:"score"`
	Status string                       `json:"status"`
	Checks map[string]healthCheckResult `json:"checks"`
}

// healthWeight maps each check name to its weight (sum = 100).
var healthWeight = map[string]int{
	"database":     25,
	"redis":        15,
	"nats":         15,
	"cpu":          10,
	"memory":       10,
	"disk":         10,
	"certificates": 10,
	"backups":      5,
}

// dashboardScore converts a check status to its numeric score.
func dashboardScore(status string) int {
	switch status {
	case healthHealthy:
		return 100
	case healthDegraded:
		return 70
	case healthWarning:
		return 50
	case healthNotConfigured:
		return 60
	case healthDisabled:
		return 100
	case healthUnknown:
		return 60
	case healthCritical:
		return 20
	case healthUnhealthy:
		return 0
	default:
		return 60
	}
}

// healthDashboardHandler returns a weighted health summary of all core
// subsystems: database, redis, NATS, host resources, certificates and backups.
func healthDashboardHandler(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := map[string]healthCheckResult{
			"database":     checkDashboardDatabase(deps),
			"redis":        checkDashboardRedis(deps),
			"nats":         checkDashboardNATS(deps),
			"cpu":          checkDashboardCPU(),
			"memory":       checkDashboardMemory(),
			"disk":         checkDashboardDisk(),
			"certificates": checkDashboardCertificates(deps),
			"backups":      checkDashboardBackups(deps),
		}

		total := 0
		for name, check := range checks {
			total += healthWeight[name] * dashboardScore(check.Status)
		}
		score := int(math.Round(float64(total) / 100))

		writeJSONResponse(w, http.StatusOK, healthDashboard{
			Score:  score,
			Status: overallHealthStatus(score),
			Checks: checks,
		})
	}
}

func overallHealthStatus(score int) string {
	switch {
	case score >= 90:
		return healthHealthy
	case score >= 70:
		return healthDegraded
	case score >= 50:
		return healthWarning
	default:
		return healthCritical
	}
}

func checkDashboardDatabase(deps *Dependencies) healthCheckResult {
	if deps.Pool == nil {
		return healthCheckResult{Status: healthUnknown, Detail: "database not configured"}
	}
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := deps.Pool.Ping(ctx)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return healthCheckResult{Status: healthUnhealthy, Detail: "database ping failed", LatencyMs: latency}
	}
	return healthCheckResult{Status: healthHealthy, Detail: "database reachable", LatencyMs: latency}
}

func checkDashboardRedis(deps *Dependencies) healthCheckResult {
	if deps.RDB == nil {
		return healthCheckResult{Status: healthDisabled, Detail: "redis not configured"}
	}
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := deps.RDB.Ping(ctx).Err()
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return healthCheckResult{Status: healthUnhealthy, Detail: "redis ping failed", LatencyMs: latency}
	}
	return healthCheckResult{Status: healthHealthy, Detail: "redis reachable", LatencyMs: latency}
}

func checkDashboardNATS(deps *Dependencies) healthCheckResult {
	if deps.NC == nil {
		return healthCheckResult{Status: healthUnknown, Detail: "nats not configured"}
	}
	done := make(chan struct{})
	var rtt time.Duration
	var err error
	go func() {
		rtt, err = deps.NC.RTT()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		return healthCheckResult{Status: healthUnhealthy, Detail: "nats ping timed out"}
	}
	if err != nil {
		return healthCheckResult{Status: healthUnhealthy, Detail: "nats ping failed: " + err.Error()}
	}
	return healthCheckResult{Status: healthHealthy, Detail: "nats reachable", LatencyMs: rtt.Milliseconds()}
}

func checkDashboardCertificates(deps *Dependencies) healthCheckResult {
	if deps.CA == nil || deps.CA.Cert == nil {
		return healthCheckResult{Status: healthUnknown, Detail: "no CA certificate loaded"}
	}
	now := time.Now()
	expiry := deps.CA.Cert.NotAfter
	if now.After(expiry) {
		return healthCheckResult{Status: healthCritical, Detail: "CA certificate expired " + expiry.Format(time.RFC3339)}
	}
	if expiry.Before(now.Add(30 * 24 * time.Hour)) {
		return healthCheckResult{Status: healthWarning, Detail: "CA certificate expires soon " + expiry.Format(time.RFC3339)}
	}
	return healthCheckResult{Status: healthHealthy, Detail: "CA certificate valid until " + expiry.Format(time.RFC3339)}
}

func checkDashboardBackups(deps *Dependencies) healthCheckResult {
	dir := deps.BackupsDir
	if dir == "" {
		dir = "/var/lib/renos/backups"
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return healthCheckResult{Status: healthUnknown, Detail: "no backups directory"}
		}
		return healthCheckResult{Status: healthUnknown, Detail: "backups directory unreadable"}
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(cutoff) {
			return healthCheckResult{Status: healthHealthy, Detail: "recent backup found: " + entry.Name()}
		}
	}
	return healthCheckResult{Status: healthWarning, Detail: "no backup in the last 24 hours"}
}

// checkDashboardCPU samples /proc/stat twice and computes the busy ratio.
func checkDashboardCPU() healthCheckResult {
	first, ok := readProcCPU()
	if !ok {
		return healthCheckResult{Status: healthUnknown, Detail: "cannot read /proc/stat"}
	}
	time.Sleep(200 * time.Millisecond)
	second, ok := readProcCPU()
	if !ok {
		return healthCheckResult{Status: healthUnknown, Detail: "cannot read /proc/stat"}
	}

	idle := second.idle - first.idle
	total := second.total - first.total
	if total <= 0 {
		return healthCheckResult{Status: healthUnknown, Detail: "cannot sample CPU"}
	}
	usage := 100 * (1 - float64(idle)/float64(total))
	if usage > 90 {
		return healthCheckResult{Status: healthWarning, Detail: "cpu usage high: " + strconv.Itoa(int(usage)) + "%"}
	}
	return healthCheckResult{Status: healthHealthy, Detail: "cpu usage normal: " + strconv.Itoa(int(usage)) + "%"}
}

type cpuSample struct {
	idle  uint64
	total uint64
}

func readProcCPU() (cpuSample, bool) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuSample{}, false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)[1:]
		if len(fields) < 4 {
			return cpuSample{}, false
		}
		var sample cpuSample
		for i, fld := range fields {
			val, err := strconv.ParseUint(fld, 10, 64)
			if err != nil {
				return cpuSample{}, false
			}
			sample.total += val
			if i == 3 {
				sample.idle = val
			}
		}
		return sample, true
	}
	return cpuSample{}, false
}

// checkDashboardMemory reads MemTotal and MemAvailable from /proc/meminfo.
func checkDashboardMemory() healthCheckResult {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return healthCheckResult{Status: healthUnknown, Detail: "cannot read /proc/meminfo"}
	}
	defer f.Close()

	var total, available uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			total, err = parseMeminfoKb(line)
		case strings.HasPrefix(line, "MemAvailable:"):
			available, err = parseMeminfoKb(line)
		}
		if err != nil {
			return healthCheckResult{Status: healthUnknown, Detail: "cannot parse /proc/meminfo"}
		}
		if total > 0 && available > 0 {
			break
		}
	}
	if total == 0 || available == 0 {
		return healthCheckResult{Status: healthUnknown, Detail: "cannot parse /proc/meminfo"}
	}
	usage := 100 * (1 - float64(available)/float64(total))
	if usage > 90 {
		return healthCheckResult{Status: healthWarning, Detail: "memory usage high: " + strconv.Itoa(int(usage)) + "%"}
	}
	return healthCheckResult{Status: healthHealthy, Detail: "memory usage normal: " + strconv.Itoa(int(usage)) + "%"}
}

func parseMeminfoKb(line string) (uint64, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, nil
	}
	return strconv.ParseUint(fields[1], 10, 64)
}

// checkDashboardDisk reports usage of the filesystem at "/".
func checkDashboardDisk() healthCheckResult {
	var st syscall.Statfs_t
	if err := syscall.Statfs("/", &st); err != nil {
		return healthCheckResult{Status: healthUnknown, Detail: "cannot stat filesystem"}
	}
	if st.Blocks == 0 {
		return healthCheckResult{Status: healthUnknown, Detail: "cannot compute disk usage"}
	}
	total := st.Blocks * uint64(st.Bsize)
	free := st.Bavail * uint64(st.Bsize)
	if total == 0 {
		return healthCheckResult{Status: healthUnknown, Detail: "cannot compute disk usage"}
	}
	usage := 100 * (1 - float64(free)/float64(total))
	if usage > 90 {
		return healthCheckResult{Status: healthWarning, Detail: "disk usage high: " + strconv.Itoa(int(usage)) + "%"}
	}
	return healthCheckResult{Status: healthHealthy, Detail: "disk usage normal: " + strconv.Itoa(int(usage)) + "%"}
}
