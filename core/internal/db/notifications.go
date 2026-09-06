package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationProvider struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	ProviderType string          `json:"provider_type"`
	Config       json.RawMessage `json:"config"`
	Enabled      bool            `json:"enabled"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type NotificationTemplate struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Subject      string          `json:"subject"`
	Body         string          `json:"body"`
	TemplateType string          `json:"template_type"`
	Variables    json.RawMessage `json:"variables"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type NotificationRule struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	EventType       string          `json:"event_type"`
	Conditions      json.RawMessage `json:"conditions"`
	ProviderIDs     json.RawMessage `json:"provider_ids"`
	TemplateID      *string         `json:"template_id,omitempty"`
	CooldownSeconds int             `json:"cooldown_seconds"`
	Enabled         bool            `json:"enabled"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type NotificationHistoryEntry struct {
	ID           string     `json:"id"`
	RuleID       *string    `json:"rule_id,omitempty"`
	ProviderID   *string    `json:"provider_id,omitempty"`
	TemplateID   *string    `json:"template_id,omitempty"`
	EventType    string     `json:"event_type"`
	Subject      string     `json:"subject"`
	Body         string     `json:"body"`
	Status       string     `json:"status"`
	ErrorMessage string     `json:"error_message"`
	SentAt       *time.Time `json:"sent_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type NotificationProviderRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationProviderRepository(pool *pgxpool.Pool) *NotificationProviderRepository {
	return &NotificationProviderRepository{pool: pool}
}

func (r *NotificationProviderRepository) Create(ctx context.Context, p *NotificationProvider) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO notification_providers (name, provider_type, config, enabled)
		 VALUES ($1,$2,$3,$4) RETURNING id, created_at, updated_at`,
		p.Name, p.ProviderType, p.Config, p.Enabled,
	)
	return row.Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *NotificationProviderRepository) ListAll(ctx context.Context) ([]*NotificationProvider, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, provider_type, config, enabled, created_at, updated_at
		 FROM notification_providers ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*NotificationProvider
	for rows.Next() {
		p := &NotificationProvider{}
		if err := rows.Scan(&p.ID, &p.Name, &p.ProviderType, &p.Config, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *NotificationProviderRepository) ListEnabled(ctx context.Context) ([]*NotificationProvider, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, provider_type, config, enabled, created_at, updated_at
		 FROM notification_providers WHERE enabled = true ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*NotificationProvider
	for rows.Next() {
		p := &NotificationProvider{}
		if err := rows.Scan(&p.ID, &p.Name, &p.ProviderType, &p.Config, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *NotificationProviderRepository) FindByID(ctx context.Context, id string) (*NotificationProvider, error) {
	p := &NotificationProvider{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, provider_type, config, enabled, created_at, updated_at
		 FROM notification_providers WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.ProviderType, &p.Config, &p.Enabled, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *NotificationProviderRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM notification_providers WHERE id = $1`, id)
	return err
}

func (r *NotificationProviderRepository) ToggleEnabled(ctx context.Context, id string, enabled bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE notification_providers SET enabled = $1, updated_at = NOW() WHERE id = $2`, enabled, id)
	return err
}

type NotificationTemplateRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationTemplateRepository(pool *pgxpool.Pool) *NotificationTemplateRepository {
	return &NotificationTemplateRepository{pool: pool}
}

func (r *NotificationTemplateRepository) Create(ctx context.Context, t *NotificationTemplate) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO notification_templates (name, subject, body, template_type, variables)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at, updated_at`,
		t.Name, t.Subject, t.Body, t.TemplateType, t.Variables,
	)
	return row.Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *NotificationTemplateRepository) ListAll(ctx context.Context) ([]*NotificationTemplate, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, subject, body, template_type, variables, created_at, updated_at
		 FROM notification_templates ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*NotificationTemplate
	for rows.Next() {
		t := &NotificationTemplate{}
		if err := rows.Scan(&t.ID, &t.Name, &t.Subject, &t.Body, &t.TemplateType, &t.Variables, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (r *NotificationTemplateRepository) FindByID(ctx context.Context, id string) (*NotificationTemplate, error) {
	t := &NotificationTemplate{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, subject, body, template_type, variables, created_at, updated_at
		 FROM notification_templates WHERE id = $1`, id,
	).Scan(&t.ID, &t.Name, &t.Subject, &t.Body, &t.TemplateType, &t.Variables, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *NotificationTemplateRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM notification_templates WHERE id = $1`, id)
	return err
}

type NotificationRuleRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRuleRepository(pool *pgxpool.Pool) *NotificationRuleRepository {
	return &NotificationRuleRepository{pool: pool}
}

func (r *NotificationRuleRepository) Create(ctx context.Context, nr *NotificationRule) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO notification_rules (name, description, event_type, conditions, provider_ids, template_id, cooldown_seconds, enabled)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, created_at, updated_at`,
		nr.Name, nr.Description, nr.EventType, nr.Conditions, nr.ProviderIDs, nr.TemplateID, nr.CooldownSeconds, nr.Enabled,
	)
	return row.Scan(&nr.ID, &nr.CreatedAt, &nr.UpdatedAt)
}

func (r *NotificationRuleRepository) ListAll(ctx context.Context) ([]*NotificationRule, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, event_type, conditions, provider_ids, template_id, cooldown_seconds, enabled, created_at, updated_at
		 FROM notification_rules ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*NotificationRule
	for rows.Next() {
		nr := &NotificationRule{}
		if err := rows.Scan(&nr.ID, &nr.Name, &nr.Description, &nr.EventType, &nr.Conditions, &nr.ProviderIDs, &nr.TemplateID, &nr.CooldownSeconds, &nr.Enabled, &nr.CreatedAt, &nr.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, nr)
	}
	return list, rows.Err()
}

func (r *NotificationRuleRepository) ListEnabled(ctx context.Context) ([]*NotificationRule, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, event_type, conditions, provider_ids, template_id, cooldown_seconds, enabled, created_at, updated_at
		 FROM notification_rules WHERE enabled = true ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*NotificationRule
	for rows.Next() {
		nr := &NotificationRule{}
		if err := rows.Scan(&nr.ID, &nr.Name, &nr.Description, &nr.EventType, &nr.Conditions, &nr.ProviderIDs, &nr.TemplateID, &nr.CooldownSeconds, &nr.Enabled, &nr.CreatedAt, &nr.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, nr)
	}
	return list, rows.Err()
}

func (r *NotificationRuleRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM notification_rules WHERE id = $1`, id)
	return err
}

type NotificationHistoryRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationHistoryRepository(pool *pgxpool.Pool) *NotificationHistoryRepository {
	return &NotificationHistoryRepository{pool: pool}
}

func (r *NotificationHistoryRepository) Create(ctx context.Context, h *NotificationHistoryEntry) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO notification_history (rule_id, provider_id, template_id, event_type, subject, body, status, error_message, sent_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, created_at`,
		h.RuleID, h.ProviderID, h.TemplateID, h.EventType, h.Subject, h.Body, h.Status, h.ErrorMessage, h.SentAt,
	)
	return row.Scan(&h.ID, &h.CreatedAt)
}

func (r *NotificationHistoryRepository) ListRecent(ctx context.Context, limit int) ([]*NotificationHistoryEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, rule_id, provider_id, template_id, event_type, subject, body, status, COALESCE(error_message, ''), sent_at, created_at
		 FROM notification_history ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*NotificationHistoryEntry
	for rows.Next() {
		h := &NotificationHistoryEntry{}
		if err := rows.Scan(&h.ID, &h.RuleID, &h.ProviderID, &h.TemplateID, &h.EventType, &h.Subject, &h.Body, &h.Status, &h.ErrorMessage, &h.SentAt, &h.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, rows.Err()
}

func (r *NotificationHistoryRepository) GetStats(ctx context.Context) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT status, COUNT(*) FROM notification_history GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}
	return stats, rows.Err()
}
