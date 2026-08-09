package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertRuleRepository struct {
	pool *pgxpool.Pool
}

func NewAlertRuleRepository(pool *pgxpool.Pool) *AlertRuleRepository {
	return &AlertRuleRepository{pool: pool}
}

func (r *AlertRuleRepository) Create(ctx context.Context, a *AlertRule) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO alert_rules (name, description, metric_name, condition, threshold, duration_seconds, severity, enabled, notification_channels)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, created_at, updated_at`,
		a.Name, a.Description, a.MetricName, a.Condition, a.Threshold, a.DurationSeconds, a.Severity, a.Enabled, a.NotificationChannels,
	)
	return row.Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *AlertRuleRepository) FindByID(ctx context.Context, id string) (*AlertRule, error) {
	a := &AlertRule{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, metric_name, condition, threshold, duration_seconds, severity, enabled, notification_channels, created_at, updated_at
		 FROM alert_rules WHERE id = $1`, id,
	).Scan(&a.ID, &a.Name, &a.Description, &a.MetricName, &a.Condition, &a.Threshold, &a.DurationSeconds, &a.Severity, &a.Enabled, &a.NotificationChannels, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *AlertRuleRepository) ListAll(ctx context.Context) ([]*AlertRule, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, metric_name, condition, threshold, duration_seconds, severity, enabled, notification_channels, created_at, updated_at
		 FROM alert_rules ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*AlertRule
	for rows.Next() {
		a := &AlertRule{}
		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.MetricName, &a.Condition, &a.Threshold, &a.DurationSeconds, &a.Severity, &a.Enabled, &a.NotificationChannels, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (r *AlertRuleRepository) ListEnabled(ctx context.Context) ([]*AlertRule, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, metric_name, condition, threshold, duration_seconds, severity, enabled, notification_channels, created_at, updated_at
		 FROM alert_rules WHERE enabled = true ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*AlertRule
	for rows.Next() {
		a := &AlertRule{}
		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.MetricName, &a.Condition, &a.Threshold, &a.DurationSeconds, &a.Severity, &a.Enabled, &a.NotificationChannels, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (r *AlertRuleRepository) Update(ctx context.Context, a *AlertRule) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE alert_rules SET name=$1, description=$2, metric_name=$3, condition=$4, threshold=$5, duration_seconds=$6, severity=$7, enabled=$8, notification_channels=$9, updated_at=NOW()
		 WHERE id=$10`,
		a.Name, a.Description, a.MetricName, a.Condition, a.Threshold, a.DurationSeconds, a.Severity, a.Enabled, a.NotificationChannels, a.ID,
	)
	return err
}

func (r *AlertRuleRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM alert_rules WHERE id = $1`, id)
	return err
}

type AlertInstanceRepository struct {
	pool *pgxpool.Pool
}

func NewAlertInstanceRepository(pool *pgxpool.Pool) *AlertInstanceRepository {
	return &AlertInstanceRepository{pool: pool}
}

func (r *AlertInstanceRepository) Create(ctx context.Context, a *AlertInstance) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO alert_instances (rule_id, device_id, status, value, message)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id, started_at, created_at, updated_at`,
		a.RuleID, a.DeviceID, a.Status, a.Value, a.Message,
	)
	return row.Scan(&a.ID, &a.StartedAt, &a.CreatedAt, &a.UpdatedAt)
}

func (r *AlertInstanceRepository) ListActive(ctx context.Context) ([]*AlertInstance, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, rule_id, device_id, status, value, message, started_at, resolved_at, acknowledged_by, acknowledged_at, created_at, updated_at
		 FROM alert_instances WHERE status = 'firing' ORDER BY started_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*AlertInstance
	for rows.Next() {
		a := &AlertInstance{}
		if err := rows.Scan(&a.ID, &a.RuleID, &a.DeviceID, &a.Status, &a.Value, &a.Message, &a.StartedAt, &a.ResolvedAt, &a.AcknowledgedBy, &a.AcknowledgedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (r *AlertInstanceRepository) ListRecent(ctx context.Context, limit int) ([]*AlertInstance, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, rule_id, device_id, status, value, message, started_at, resolved_at, acknowledged_by, acknowledged_at, created_at, updated_at
		 FROM alert_instances ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*AlertInstance
	for rows.Next() {
		a := &AlertInstance{}
		if err := rows.Scan(&a.ID, &a.RuleID, &a.DeviceID, &a.Status, &a.Value, &a.Message, &a.StartedAt, &a.ResolvedAt, &a.AcknowledgedBy, &a.AcknowledgedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (r *AlertInstanceRepository) Resolve(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE alert_instances SET status = 'resolved', resolved_at = NOW(), updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *AlertInstanceRepository) Acknowledge(ctx context.Context, id, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE alert_instances SET acknowledged_by = $1, acknowledged_at = NOW(), updated_at = NOW() WHERE id = $2`, userID, id)
	return err
}

func (r *AlertInstanceRepository) GetSummary(ctx context.Context) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT status, COUNT(*) FROM alert_instances GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	summary := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		summary[status] = count
	}
	return summary, rows.Err()
}
