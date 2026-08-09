package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Policy struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        string          `json:"type"`
	Conditions  json.RawMessage `json:"conditions"`
	Action      string          `json:"action"`
	Priority    int             `json:"priority"`
	Enabled     bool            `json:"enabled"`
	CreatedBy   *string         `json:"created_by,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type PolicyConditions struct {
	MaxDevices            int      `json:"max_devices,omitempty"`
	RequiresConfirmation  bool     `json:"requires_confirmation,omitempty"`
	RestrictCronFrequency int      `json:"restrict_cron_frequency,omitempty"`
	AllowedCommands       []string `json:"allowed_commands,omitempty"`
	RestrictedTargets     []string `json:"restricted_targets,omitempty"`
}

type PolicyEvalResult struct {
	Policy   *Policy
	Action   string // "allow", "deny", "warn"
	Message  string
	Triggers []string
}

type PolicyEvalContext struct {
	ActionType    string
	DeviceIDs     []string
	TargetIDs     []string
	CronExpr      string
	Command       string
	ContainerID   string
	Action        string
	UnitName      string
	UpdateAction  string
	ScheduleType  string
}

type PolicyRepository struct {
	pool *pgxpool.Pool
}

func NewPolicyRepository(pool *pgxpool.Pool) *PolicyRepository {
	return &PolicyRepository{pool: pool}
}

func (r *PolicyRepository) Create(ctx context.Context, name, description, policyType string, conditions json.RawMessage, action string, priority int, enabled bool, createdBy *string) (*Policy, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO policies (name, description, type, conditions, action, priority, enabled, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, name, description, type, conditions, action, priority, enabled, created_by, created_at, updated_at`,
		name, description, policyType, conditions, action, priority, enabled, createdBy,
	)
	return scanPolicy(row)
}

func (r *PolicyRepository) FindByID(ctx context.Context, id string) (*Policy, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, description, type, conditions, action, priority, enabled, created_by, created_at, updated_at
		 FROM policies WHERE id = $1`, id,
	)
	return scanPolicy(row)
}

func (r *PolicyRepository) FindAll(ctx context.Context) ([]*Policy, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, type, conditions, action, priority, enabled, created_by, created_at, updated_at
		 FROM policies ORDER BY priority DESC, created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list policies: %w", err)
	}
	defer rows.Close()

	var list []*Policy
	for rows.Next() {
		p, err := scanPolicy(rows)
		if err != nil {
			return nil, fmt.Errorf("scan policy: %w", err)
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *PolicyRepository) Update(ctx context.Context, id, name, description, policyType string, conditions json.RawMessage, action string, priority int, enabled bool) (*Policy, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE policies
		 SET name = $2, description = $3, type = $4, conditions = $5, action = $6,
		     priority = $7, enabled = $8, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, name, description, type, conditions, action, priority, enabled, created_by, created_at, updated_at`,
		id, name, description, policyType, conditions, action, priority, enabled,
	)
	return scanPolicy(row)
}

func (r *PolicyRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM policies WHERE id = $1`, id)
	return err
}

func (r *PolicyRepository) SetEnabled(ctx context.Context, id string, enabled bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE policies SET enabled = $2, updated_at = NOW() WHERE id = $1`, id, enabled,
	)
	return err
}

// FindMatching returns all enabled policies that could apply to the given action type.
// Returns policies matching the specific type plus wildcard "*" policies, ordered by priority.
func (r *PolicyRepository) CountPolicyBlocks(ctx context.Context) (int64, error) {
	// Count policy blocks from the audit trail (policy.blocked events)
	var count int64
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audits WHERE action = 'policy.blocked'`,
	).Scan(&count)
	return count, err
}

func (r *PolicyRepository) FindMatching(ctx context.Context, actionType string) ([]*Policy, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, type, conditions, action, priority, enabled, created_by, created_at, updated_at
		 FROM policies
		 WHERE enabled = true AND (type = $1 OR type = '*')
		 ORDER BY priority DESC, created_at ASC`,
		actionType,
	)
	if err != nil {
		return nil, fmt.Errorf("find matching policies: %w", err)
	}
	defer rows.Close()

	var list []*Policy
	for rows.Next() {
		p, err := scanPolicy(rows)
		if err != nil {
			return nil, fmt.Errorf("scan matching policy: %w", err)
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// ParseConditions decodes a policy's conditions JSON into a typed struct.
func ParseConditions(raw json.RawMessage) (*PolicyConditions, error) {
	var c PolicyConditions
	if len(raw) == 0 {
		return &c, nil
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse conditions: %w", err)
	}
	return &c, nil
}

// Evaluate checks all matching policies and returns the first deny or warn result.
// If no policy denies or warns, returns an allow result.
func (r *PolicyRepository) Evaluate(ctx context.Context, ectx *PolicyEvalContext) (*PolicyEvalResult, error) {
	policies, err := r.FindMatching(ctx, ectx.ActionType)
	if err != nil {
		return nil, err
	}

	for _, p := range policies {
		conds, err := ParseConditions(p.Conditions)
		if err != nil {
			continue
		}

		result := evaluatePolicy(p, conds, ectx)
		if result != nil {
			return result, nil
		}
	}

	return &PolicyEvalResult{
		Action:  "allow",
		Message: "No matching policies",
	}, nil
}

func evaluatePolicy(p *Policy, conds *PolicyConditions, ectx *PolicyEvalContext) *PolicyEvalResult {
	triggers := []string{}

	// Check max_devices
	if conds.MaxDevices > 0 {
		totalDevices := len(ectx.DeviceIDs)
		if totalDevices > conds.MaxDevices {
			triggers = append(triggers, fmt.Sprintf("max_devices: %d > %d", totalDevices, conds.MaxDevices))
		}
	}

	// Check requires_confirmation
	if conds.RequiresConfirmation && len(ectx.DeviceIDs) > conds.MaxDevices && conds.MaxDevices > 0 {
		// Already handled by max_devices — no separate trigger needed
	} else if conds.RequiresConfirmation && len(ectx.DeviceIDs) > 0 {
		triggers = append(triggers, "requires_confirmation")
	}

	// Check restrict_cron_frequency (for schedules)
	if conds.RestrictCronFrequency > 0 && ectx.CronExpr != "" {
		if isCronTooFrequent(ectx.CronExpr, conds.RestrictCronFrequency) {
			triggers = append(triggers, fmt.Sprintf("restrict_cron_frequency: min %d min", conds.RestrictCronFrequency))
		}
	}

	// Check allowed_commands
	if len(conds.AllowedCommands) > 0 && ectx.Command != "" {
		if !isCommandAllowed(ectx.Command, conds.AllowedCommands) {
			triggers = append(triggers, "command_not_allowed")
		}
	}

	// Check restricted_targets
	if len(conds.RestrictedTargets) > 0 && len(ectx.TargetIDs) > 0 {
		for _, target := range ectx.TargetIDs {
			for _, restricted := range conds.RestrictedTargets {
				if target == restricted {
					triggers = append(triggers, fmt.Sprintf("restricted_target: %s", target))
					break
				}
			}
		}
	}

	if len(triggers) == 0 {
		return nil
	}

	return &PolicyEvalResult{
		Policy:   p,
		Action:   p.Action,
		Message:  p.Name + ": " + joinStrings(triggers, "; "),
		Triggers: triggers,
	}
}

func isCommandAllowed(cmd string, allowed []string) bool {
	for _, a := range allowed {
		if cmd == a {
			return true
		}
	}
	return false
}

func isCronTooFrequent(expr string, minMinutes int) bool {
	// Simple heuristic: check if the first field (minutes) is *, */N where N < minMinutes, or a list
	parts := splitAndTrim(expr, " ")
	if len(parts) < 1 {
		return false
	}
	minField := parts[0]

	if minField == "*" {
		return true
	}
	if len(minField) > 2 && minField[:2] == "*/" {
		var interval int
		if _, err := fmt.Sscanf(minField, "*/%d", &interval); err == nil && interval < minMinutes {
			return true
		}
	}
	return false
}

func splitAndTrim(s, sep string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if string(c) == sep {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func joinStrings(items []string, sep string) string {
	result := ""
	for i, item := range items {
		if i > 0 {
			result += sep
		}
		result += item
	}
	return result
}

func scanPolicy(s scanner) (*Policy, error) {
	var p Policy
	var conditions []byte
	err := s.Scan(&p.ID, &p.Name, &p.Description, &p.Type, &conditions,
		&p.Action, &p.Priority, &p.Enabled, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan policy: %w", err)
	}
	if conditions != nil {
		p.Conditions = json.RawMessage(conditions)
	}
	return &p, nil
}
