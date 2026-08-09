package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/ws"
)

// Dangerous commands that should be blocked even for operators.

func isBlockedCommand(cmd string) bool {
	lower := strings.ToLower(strings.TrimSpace(cmd))

	// Always block fork bombs and disk wipe commands
	alwaysBlock := []string{":(){:|:&};:", "mkfs", "> /dev/sd"}
	for _, b := range alwaysBlock {
		if strings.Contains(lower, b) {
			return true
		}
	}

	// Block dd if=/dev/zero or dd if=/dev/random targeting block devices
	if strings.Contains(lower, "dd if=/dev/zero of=/dev/sd") || strings.Contains(lower, "dd if=/dev/random of=/dev/sd") {
		return true
	}

	// Block chmod/chown -R 777 / on root
	if (strings.Contains(lower, "chmod -r 777 /") || strings.Contains(lower, "chown -r 777 /")) && hasRootPath(lower) {
		return true
	}

	// Block rm -rf on root, but allow /tmp and /var/tmp
	if strings.Contains(lower, "rm ") && strings.Contains(lower, " -rf ") {
		if hasRootPath(lower) {
			return true
		}
		// Also block bare "rm -rf /" (exactly root)
		rest := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(lower, "rm", ""), "-rf", ""))
		if rest == "/" || rest == "/*" || rest == "~" || rest == "$home" {
			return true
		}
	}
	// Handle "rm -rf /path" without space before -rf (e.g., "rm -rf/etc")
	if strings.Contains(lower, "rm ") && strings.HasPrefix(strings.TrimPrefix(lower, "rm "), "-rf/") {
		path := strings.TrimPrefix(strings.TrimPrefix(lower, "rm "), "-rf")
		if isDangerousRootPath(path) {
			return true
		}
	}

	return false
}

// hasRootPath checks if the command targets a dangerous root-level path.
func hasRootPath(lower string) bool {
	dangerous := []string{
		" /", " /*", "/etc", "/usr", "/bin", "/sbin", "/boot",
		"/lib", "/lib64", "/root", "/home", "/opt", "/sys", "/proc", "/dev",
	}
	safe := []string{"/tmp", "/var/tmp"}
	for _, d := range dangerous {
		if strings.Contains(lower, d) {
			for _, s := range safe {
				if strings.Contains(lower, s) {
					return false
				}
			}
			return true
		}
	}
	return false
}

func isDangerousRootPath(path string) bool {
	safe := []string{"/tmp", "/var/tmp"}
	for _, s := range safe {
		if strings.HasPrefix(path, s) {
			return false
		}
	}
	return strings.HasPrefix(path, "/") || path == "~" || path == "$home"
}

// isRiskyCommand flags commands that modify system state (operator-level warning).
func isRiskyCommand(cmd string) bool {
	lower := strings.ToLower(strings.TrimSpace(cmd))
	risky := []string{
		"rm ", "rmdir", "mkfs", "fdisk", "dd ", "chmod ", "chown ",
		"systemctl restart", "systemctl stop", "kill ", "pkill ",
		"shutdown", "reboot", "service ", "mount ", "umount ",
		"iptables", "ufw ", "apt ", "yum ", "dnf ", "pacman ",
	}
	for _, r := range risky {
		if strings.Contains(lower, r) {
			return true
		}
	}
	return false
}

const (
	maxLimit     = 200
	maxOffset    = 10000
	defLimit     = 50
	listDefLimit = 500
	listMaxLimit = 1000
)

func parseListLimit(s string) int {
	return parseLimitWith(s, listDefLimit, listMaxLimit)
}

func parseLimitWith(s string, def, max int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

func parseLimit(s string) int {
	return parseLimitWith(s, defLimit, maxLimit)
}

func parseOffset(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0
	}
	if n > maxOffset {
		return maxOffset
	}
	return n
}

type createExecutionPayload struct {
	Command        string `json:"command"`
	Shell          string `json:"shell"`
	TimeoutMs      int64  `json:"timeout_ms"`
	MaxOutputBytes int64  `json:"max_output_bytes"`
}

type executionMessage struct {
	ExecutionID string `json:"execution_id"`
	Command     string `json:"command"`
	Shell       string `json:"shell"`
	TimeoutMs   int64  `json:"timeout_ms"`
	MaxOutput   int64  `json:"max_output_bytes"`
}

type executionOutputMessage struct {
	ExecutionID string `json:"execution_id"`
	Stream      string `json:"stream"`
	Content     string `json:"content"`
	Seq         int    `json:"seq"`
}

type executionCompleteMessage struct {
	ExecutionID string `json:"execution_id"`
	ExitCode    int    `json:"exit_code"`
}

type executionCancelMessage struct {
	ExecutionID string `json:"execution_id"`
}

func createExecutionHandler(
	executions *db.ExecutionRepository,
	devices *db.DeviceRepository,
	audits *db.AuditRepository,
	nc *nats.Conn,
	hub *ws.Hub,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		device, err := devices.FindByID(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Device not found")
			return
		}
		if device.Status != "online" {
			writeError(w, http.StatusConflict, "DEVICE_OFFLINE", "Device is not online")
			return
		}

		claims := r.Context().Value(claimsKey).(*auth.Claims)

		var req createExecutionPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.Command == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "command is required")
			return
		}
		if isBlockedCommand(req.Command) {
			writeError(w, http.StatusForbidden, "COMMAND_BLOCKED", "This command is blocked by security policy")
			return
		}
		if req.Shell == "" {
			req.Shell = "auto"
		}
		if req.TimeoutMs <= 0 {
			req.TimeoutMs = 300000
		}
		if req.MaxOutputBytes <= 0 {
			req.MaxOutputBytes = 1048576
		}

		// Reject concurrent executions on same device
		hasRunning, err := executions.HasRunning(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to check running executions")
			return
		}
		if hasRunning {
			writeError(w, http.StatusConflict, "CONFLICT", "Another execution is already running on this device")
			return
		}

		exec, err := executions.Create(r.Context(), deviceID, claims.UserID, req.Command, req.Shell, req.TimeoutMs, req.MaxOutputBytes)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create execution")
			return
		}

		// Publish execution job to NATS for the agent
		natsSubject := "exec.run." + deviceID
		msg := executionMessage{
			ExecutionID: exec.ID,
			Command:     exec.Command,
			Shell:       exec.Shell,
			TimeoutMs:   exec.TimeoutMs,
			MaxOutput:   exec.MaxOutputBytes,
		}
		if err := nc.Publish(natsSubject, mustMarshal(msg)); err != nil {
			// Mark as failed if we can't reach agent
			executions.UpdateStatus(r.Context(), exec.ID, "failed")
			writeError(w, http.StatusServiceUnavailable, "NATS_ERROR", "Failed to dispatch execution to agent")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "execution.created", "device", deviceID)
		hub.Broadcast(ws.Message{
			Type:     "execution.created",
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"execution_id": exec.ID,
				"command":      exec.Command,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"execution": exec,
		})
	}
}

func listExecutionsHandler(executions *db.ExecutionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		limit := 50
		offset := 0
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		if v := r.URL.Query().Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 10000 {
				offset = n
			}
		}

		list, total, err := executions.ListByDevice(r.Context(), deviceID, limit, offset)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list executions")
			return
		}
		if list == nil {
			list = []*db.Execution{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"executions": list,
			"total":      total,
			"limit":      limit,
			"offset":     offset,
		})
	}
}

func getExecutionHandler(executions *db.ExecutionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		execID := chi.URLParam(r, "execId")
		if execID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "execution id required")
			return
		}

		exec, err := executions.FindByID(r.Context(), execID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Execution not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"execution": exec,
		})
	}
}

func getExecutionOutputHandler(executions *db.ExecutionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		execID := chi.URLParam(r, "execId")
		if execID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "execution id required")
			return
		}

		output, err := executions.GetOutput(r.Context(), execID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to get execution output")
			return
		}
		if output == nil {
			output = []*db.ExecutionOutput{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"output": output,
		})
	}
}

func cancelExecutionHandler(executions *db.ExecutionRepository, nc *nats.Conn, audits *db.AuditRepository, hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		execID := chi.URLParam(r, "execId")

		exec, err := executions.FindByID(r.Context(), execID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Execution not found")
			return
		}
		if exec.DeviceID != deviceID {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Execution not found for this device")
			return
		}
		if exec.Status != "pending" && exec.Status != "running" {
			writeError(w, http.StatusConflict, "CONFLICT", "Execution is not in a cancellable state")
			return
		}

		// Send cancel signal to agent via NATS
		cancelSubject := "exec.cancel." + execID
		cancelMsg := executionCancelMessage{ExecutionID: execID}
		if err := nc.Publish(cancelSubject, mustMarshal(cancelMsg)); err != nil {
			writeError(w, http.StatusServiceUnavailable, "NATS_ERROR", "Failed to send cancel signal")
			return
		}

		// Also update DB status
		if err := executions.MarkCancelled(r.Context(), execID); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to cancel execution")
			return
		}

		claims := r.Context().Value(claimsKey).(*auth.Claims)
		audits.Create(r.Context(), "user", claims.UserID, "execution.cancelled", "device", deviceID)
		hub.Broadcast(ws.Message{
			Type:     "execution.cancelled",
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"execution_id": execID,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"execution": exec,
		})
	}
}

func HandleExecutionOutput(nc *nats.Conn, executions *db.ExecutionRepository, hub *ws.Hub) func(*nats.Msg) {
	return func(msg *nats.Msg) {
		var outMsg executionOutputMessage
		if err := json.Unmarshal(msg.Data, &outMsg); err != nil {
			return
		}

		ctx := context.Background()

		// Persist output chunk
		if err := executions.AppendOutput(ctx, outMsg.ExecutionID, outMsg.Stream, outMsg.Content, outMsg.Seq); err != nil {
			return
		}

		// Broadcast to WebSocket clients
		exec, err := executions.FindByID(ctx, outMsg.ExecutionID)
		if err != nil {
			return
		}

		hub.Broadcast(ws.Message{
			Type:     "execution.output",
			DeviceID: exec.DeviceID,
			Data: map[string]interface{}{
				"execution_id": outMsg.ExecutionID,
				"stream":       outMsg.Stream,
				"content":      outMsg.Content,
				"seq":          outMsg.Seq,
			},
		})
	}
}

func HandleExecutionComplete(nc *nats.Conn, executions *db.ExecutionRepository, audits *db.AuditRepository, hub *ws.Hub) func(*nats.Msg) {
	return func(msg *nats.Msg) {
		var completeMsg executionCompleteMessage
		if err := json.Unmarshal(msg.Data, &completeMsg); err != nil {
			return
		}

		ctx := context.Background()

		exec, err := executions.FindByID(ctx, completeMsg.ExecutionID)
		if err != nil {
			return
		}

		if completeMsg.ExitCode == 0 {
			executions.MarkCompleted(ctx, completeMsg.ExecutionID, 0)
		} else {
			executions.MarkFailed(ctx, completeMsg.ExecutionID, completeMsg.ExitCode)
		}

		audits.Create(ctx, "device", exec.DeviceID, "execution.completed", "execution", completeMsg.ExecutionID)
		hub.Broadcast(ws.Message{
			Type:     "execution.completed",
			DeviceID: exec.DeviceID,
			Data: map[string]interface{}{
				"execution_id": completeMsg.ExecutionID,
				"exit_code":    completeMsg.ExitCode,
			},
		})
	}
}

func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
