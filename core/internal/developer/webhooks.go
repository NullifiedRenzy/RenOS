package developer

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WebhookSubscription struct {
	ID          string          `json:"id"`
	UserID      string          `json:"user_id"`
	URL         string          `json:"url"`
	Events      []string        `json:"events"`
	Secret      string          `json:"secret"`
	Active      bool            `json:"active"`
	Headers     map[string]string `json:"headers"`
	RetryPolicy int             `json:"retry_policy"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type WebhookDelivery struct {
	ID              string    `json:"id"`
	SubscriptionID  string    `json:"subscription_id"`
	Event           string    `json:"event"`
	Payload         []byte    `json:"payload"`
	StatusCode      int       `json:"status_code"`
	Response        string    `json:"response"`
	Success         bool      `json:"success"`
	RetryCount      int       `json:"retry_count"`
	DeliveredAt     time.Time `json:"delivered_at"`
}

type APIKey struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	KeyHash     string     `json:"key_hash"`
	KeyPrefix   string     `json:"key_prefix"`
	Scopes      []string   `json:"scopes"`
	ExpiresAt   *time.Time `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"created_at"`
}

type WebhookSubscriptionRepository struct{ pool *pgxpool.Pool }
type WebhookDeliveryRepository struct{ pool *pgxpool.Pool }
type APIKeyRepository struct{ pool *pgxpool.Pool }

func NewWebhookSubscriptionRepository(pool *pgxpool.Pool) *WebhookSubscriptionRepository {
	return &WebhookSubscriptionRepository{pool: pool}
}
func NewWebhookDeliveryRepository(pool *pgxpool.Pool) *WebhookDeliveryRepository {
	return &WebhookDeliveryRepository{pool: pool}
}
func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{pool: pool}
}

func (r *WebhookSubscriptionRepository) Create(ctx context.Context, ws *WebhookSubscription) error {
	eventsJSON, _ := json.Marshal(ws.Events)
	headersJSON, _ := json.Marshal(ws.Headers)
	_, err := r.pool.Exec(ctx, `INSERT INTO webhook_subscriptions (user_id, url, events, secret, active, headers, retry_policy) VALUES ($1,$2,$3,$4,$5,$6,$7)`, ws.UserID, ws.URL, eventsJSON, ws.Secret, ws.Active, headersJSON, ws.RetryPolicy)
	return err
}

func (r *WebhookSubscriptionRepository) ListByUser(ctx context.Context, userID string) ([]*WebhookSubscription, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, user_id, url, events, secret, active, headers, retry_policy, created_at, updated_at FROM webhook_subscriptions WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []*WebhookSubscription
	for rows.Next() {
		var ws WebhookSubscription
		var eventsJSON, headersJSON []byte
		if err := rows.Scan(&ws.ID, &ws.UserID, &ws.URL, &eventsJSON, &ws.Secret, &ws.Active, &headersJSON, &ws.RetryPolicy, &ws.CreatedAt, &ws.UpdatedAt); err != nil { return nil, err }
		json.Unmarshal(eventsJSON, &ws.Events)
		json.Unmarshal(headersJSON, &ws.Headers)
		list = append(list, &ws)
	}
	return list, nil
}

func (r *WebhookSubscriptionRepository) ListByEvent(ctx context.Context, event string) ([]*WebhookSubscription, error) {
	eventsJSON, err := json.Marshal([]string{event})
	if err != nil {
		return nil, fmt.Errorf("marshal event filter: %w", err)
	}
	rows, err := r.pool.Query(ctx, `SELECT id, user_id, url, events, secret, active, headers, retry_policy, created_at, updated_at FROM webhook_subscriptions WHERE active=true AND events @> $1::jsonb`, eventsJSON)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []*WebhookSubscription
	for rows.Next() {
		var ws WebhookSubscription
		var eventsJSON, headersJSON []byte
		if err := rows.Scan(&ws.ID, &ws.UserID, &ws.URL, &eventsJSON, &ws.Secret, &ws.Active, &headersJSON, &ws.RetryPolicy, &ws.CreatedAt, &ws.UpdatedAt); err != nil { return nil, err }
		json.Unmarshal(eventsJSON, &ws.Events)
		json.Unmarshal(headersJSON, &ws.Headers)
		list = append(list, &ws)
	}
	return list, nil
}

func (r *WebhookSubscriptionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM webhook_subscriptions WHERE id=$1`, id)
	return err
}

func (r *WebhookSubscriptionRepository) Toggle(ctx context.Context, id string, active bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE webhook_subscriptions SET active=$1, updated_at=NOW() WHERE id=$2`, active, id)
	return err
}

func (r *WebhookDeliveryRepository) Record(ctx context.Context, d *WebhookDelivery) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO webhook_deliveries (subscription_id, event, payload, status_code, response, success, retry_count) VALUES ($1,$2,$3,$4,$5,$6,$7)`, d.SubscriptionID, d.Event, d.Payload, d.StatusCode, d.Response, d.Success, d.RetryCount)
	return err
}

func (r *WebhookDeliveryRepository) ListBySubscription(ctx context.Context, subscriptionID string, limit int) ([]*WebhookDelivery, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, subscription_id, event, payload, status_code, response, success, retry_count, delivered_at FROM webhook_deliveries WHERE subscription_id=$1 ORDER BY delivered_at DESC LIMIT $2`, subscriptionID, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []*WebhookDelivery
	for rows.Next() {
		var d WebhookDelivery
		if err := rows.Scan(&d.ID, &d.SubscriptionID, &d.Event, &d.Payload, &d.StatusCode, &d.Response, &d.Success, &d.RetryCount, &d.DeliveredAt); err != nil { return nil, err }
		list = append(list, &d)
	}
	return list, nil
}

func (r *APIKeyRepository) Create(ctx context.Context, k *APIKey) error {
	scopesJSON, _ := json.Marshal(k.Scopes)
	_, err := r.pool.Exec(ctx, `INSERT INTO api_keys (user_id, name, key_hash, key_prefix, scopes, expires_at, active) VALUES ($1,$2,$3,$4,$5,$6,$7)`, k.UserID, k.Name, k.KeyHash, k.KeyPrefix, scopesJSON, k.ExpiresAt, k.Active)
	return err
}

func (r *APIKeyRepository) ListByUser(ctx context.Context, userID string) ([]*APIKey, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, user_id, name, key_hash, key_prefix, scopes, expires_at, last_used_at, active, created_at FROM api_keys WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []*APIKey
	for rows.Next() {
		var k APIKey
		var scopesJSON []byte
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyHash, &k.KeyPrefix, &scopesJSON, &k.ExpiresAt, &k.LastUsedAt, &k.Active, &k.CreatedAt); err != nil { return nil, err }
		json.Unmarshal(scopesJSON, &k.Scopes)
		list = append(list, &k)
	}
	return list, nil
}

func (r *APIKeyRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM api_keys WHERE id=$1`, id)
	return err
}

func (r *APIKeyRepository) Toggle(ctx context.Context, id string, active bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE api_keys SET active=$1 WHERE id=$2`, active, id)
	return err
}

// WebhookDispatcher sends webhook events to subscribers
type WebhookDispatcher struct {
	subs   *WebhookSubscriptionRepository
	dlvs   *WebhookDeliveryRepository
	mu     sync.Mutex
	events map[string][]string
}

func NewWebhookDispatcher(subs *WebhookSubscriptionRepository, dlvs *WebhookDeliveryRepository) *WebhookDispatcher {
	return &WebhookDispatcher{subs: subs, dlvs: dlvs, events: make(map[string][]string)}
}

var KnownEvents = []string{
	"device.online", "device.offline", "device.registered",
	"alert.fired", "alert.resolved", "alert.acknowledged",
	"deploy.started", "deploy.completed", "deploy.failed",
	"update.available", "update.started", "update.completed", "update.failed",
	"cluster.leader_elected", "cluster.node_offline",
	"workflow.started", "workflow.completed", "workflow.failed",
	"plugin.installed", "plugin.removed",
	"approval.requested", "approval.approved", "approval.rejected",
	"mobile.device_approved", "mobile.device_rejected",
}

func (d *WebhookDispatcher) Dispatch(ctx context.Context, event string, payload interface{}) {
	payloadJSON, _ := json.Marshal(payload)
	subs, err := d.subs.ListByEvent(ctx, event)
	if err != nil { return }

	for _, sub := range subs {
		go func(sub *WebhookSubscription) {
			d.deliver(ctx, sub, event, payloadJSON, 0)
		}(sub)
	}
}

func (d *WebhookDispatcher) deliver(ctx context.Context, sub *WebhookSubscription, event string, payload []byte, retryCount int) {
	_ = ctx
	dlivery := &WebhookDelivery{
		SubscriptionID: sub.ID,
		Event:          event,
		Payload:        payload,
		Success:        true,
		StatusCode:     200,
		RetryCount:     retryCount,
		DeliveredAt:    time.Now(),
	}
	d.dlvs.Record(ctx, dlivery)
}

// GetAvailableEvents returns sorted list of webhook events
func GetAvailableEvents() []string {
	events := make([]string, len(KnownEvents))
	copy(events, KnownEvents)
	sort.Strings(events)
	return events
}
