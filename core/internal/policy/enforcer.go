package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/ws"
)

// Enforcer evaluates policies before actions and enforces the result.
type Enforcer struct {
	policyRepo *db.PolicyRepository
	audits     *db.AuditRepository
	logs       *db.SystemLogRepository
	hub        *ws.Hub
}

func NewEnforcer(policyRepo *db.PolicyRepository, audits *db.AuditRepository, logs *db.SystemLogRepository, hub *ws.Hub) *Enforcer {
	return &Enforcer{policyRepo: policyRepo, audits: audits, logs: logs, hub: hub}
}

// EvalContext describes the action being evaluated.
type EvalContext struct {
	ActionType    string
	DeviceIDs     []string
	TargetIDs     []string
	Command       string
	ContainerID   string
	Action        string
	UnitName      string
	UpdateAction  string
	CronExpr      string
	ScheduleType  string
}

// EvalResult is returned from Evaluate — callers must check Action before proceeding.
type EvalResult struct {
	Action  string // "allow", "deny", "warn"
	Message string
	Policy  *db.Policy
}

// Evaluate checks all matching policies. Returns deny/warn/allow.
// For deny — callers MUST abort the action.
// For warn — callers SHOULD surface the warning to the user.
// For allow — proceed normally.
func (e *Enforcer) Evaluate(ctx context.Context, ec *EvalContext) *EvalResult {
	result, err := e.policyRepo.Evaluate(ctx, &db.PolicyEvalContext{
		ActionType:   ec.ActionType,
		DeviceIDs:    ec.DeviceIDs,
		TargetIDs:    ec.TargetIDs,
		CronExpr:     ec.CronExpr,
		Command:      ec.Command,
		ContainerID:  ec.ContainerID,
		Action:       ec.Action,
		UnitName:     ec.UnitName,
		UpdateAction: ec.UpdateAction,
		ScheduleType: ec.ScheduleType,
	})
	if err != nil {
		slog.Error("policy evaluation error, allowing by default", "error", err)
		return &EvalResult{Action: "allow", Message: "Evaluation error, allowed by default"}
	}

	switch result.Action {
	case "deny":
		e.audit(ctx, "policy.blocked", result.Message)
		slog.Warn("policy blocked action", "action_type", ec.ActionType, "reason", result.Message)
		e.log(ctx, "warn", "policy", "Policy blocked action: "+result.Message, ec.ActionType)
		if e.hub != nil {
			e.hub.Broadcast(ws.Message{
				Type: "policy.blocked",
				Data: map[string]interface{}{
					"action_type": ec.ActionType,
					"reason":      result.Message,
				},
			})
		}
		return &EvalResult{Action: "deny", Message: result.Message, Policy: result.Policy}

	case "warn":
		e.audit(ctx, "policy.warning", result.Message)
		slog.Warn("policy warning", "action_type", ec.ActionType, "reason", result.Message)
		e.log(ctx, "warn", "policy", "Policy warning: "+result.Message, ec.ActionType)
		return &EvalResult{Action: "warn", Message: result.Message, Policy: result.Policy}

	default:
		e.audit(ctx, "policy.triggered", "allowed: "+result.Message)
		e.log(ctx, "info", "policy", "Policy allowed: "+result.Message, ec.ActionType)
		return &EvalResult{Action: "allow", Message: result.Message}
	}
}

// EvaluateJobPayload is a convenience that builds EvalContext from a job's type + payload.
func (e *Enforcer) EvaluateJobPayload(ctx context.Context, jobType string, payload json.RawMessage) *EvalResult {
	var p struct {
		DeviceIDs   []string `json:"device_ids"`
		Command     string   `json:"command"`
		ContainerID string   `json:"container_id"`
		Action      string   `json:"action"`
		UnitName    string   `json:"unit_name"`
		UpdateAct   string   `json:"update_action"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return &EvalResult{Action: "allow", Message: "Cannot parse payload for policy eval"}
	}

	return e.Evaluate(ctx, &EvalContext{
		ActionType:   jobType,
		DeviceIDs:    p.DeviceIDs,
		Command:      p.Command,
		ContainerID:  p.ContainerID,
		Action:       p.Action,
		UnitName:     p.UnitName,
		UpdateAction: p.UpdateAct,
	})
}

// BatchConfig holds rolling-execution parameters derived from policies.
type BatchConfig struct {
	MaxDevicesPerBatch int
	DelayBetweenBatches time.Duration
}

// DefaultBatchConfig returns sensible defaults.
func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		MaxDevicesPerBatch:  5,
		DelayBetweenBatches: 2 * time.Second,
	}
}

// EvaluateBatchConfig checks policies for max_devices constraints and returns
// a batch config that respects them. Callers use this to chunk device execution.
func (e *Enforcer) EvaluateBatchConfig(ctx context.Context, actionType string, deviceCount int) BatchConfig {
	cfg := DefaultBatchConfig()

	policies, err := e.policyRepo.FindMatching(ctx, actionType)
	if err != nil {
		return cfg
	}

	for _, p := range policies {
		conds, err := db.ParseConditions(p.Conditions)
		if err != nil {
			continue
		}
		if conds.MaxDevices > 0 && conds.MaxDevices < cfg.MaxDevicesPerBatch {
			cfg.MaxDevicesPerBatch = conds.MaxDevices
		}
		// If the total exceeds max_devices, we still batch but at a safer size
		if deviceCount > cfg.MaxDevicesPerBatch {
			// Use max_devices from policy as batch size
			for _, p2 := range policies {
				c, _ := db.ParseConditions(p2.Conditions)
				if c != nil && c.MaxDevices > 0 && c.MaxDevices < deviceCount {
					cfg.MaxDevicesPerBatch = c.MaxDevices
					break
				}
			}
		}
	}

	return cfg
}

func (e *Enforcer) audit(ctx context.Context, action, message string) {
	if e.audits != nil {
		e.audits.Create(ctx, "system", "policy", action, "policy", "")
	}
}

func (e *Enforcer) log(ctx context.Context, level, source, message, actionType string) {
	if e.logs != nil {
		meta, _ := json.Marshal(map[string]string{"action_type": actionType})
		e.logs.Create(ctx, level, source, message, meta)
	}
}

// ValidatePolicyConditions checks that the conditions JSON is well-formed.
func ValidatePolicyConditions(raw json.RawMessage) error {
	var conds db.PolicyConditions
	if err := json.Unmarshal(raw, &conds); err != nil {
		return fmt.Errorf("invalid policy conditions: %w", err)
	}
	return nil
}
