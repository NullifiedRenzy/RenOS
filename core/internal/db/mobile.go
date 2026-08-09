package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PushRegistration struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Platform    string     `json:"platform"`
	Token       string     `json:"token"`
	DeviceName  string     `json:"device_name"`
	DeviceModel string     `json:"device_model"`
	OSVersion   string     `json:"os_version"`
	AppVersion  string     `json:"app_version"`
	Enabled     bool       `json:"enabled"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type MobileSession struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	DeviceID         string    `json:"device_id"`
	DeviceName       string    `json:"device_name"`
	DeviceModel      string    `json:"device_model"`
	OSVersion        string    `json:"os_version"`
	AppVersion       string    `json:"app_version"`
	IPAddress        string    `json:"ip_address"`
	UserAgent        string    `json:"user_agent"`
	BiometricEnabled bool      `json:"biometric_enabled"`
	PushEnabled      bool      `json:"push_enabled"`
	LastActiveAt     time.Time `json:"last_active_at"`
	ExpiresAt        time.Time `json:"expires_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type OfflineSyncItem struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	DeviceID     string     `json:"device_id"`
	Action       string     `json:"action"`
	Payload      []byte     `json:"payload"`
	Priority     int        `json:"priority"`
	Status       string     `json:"status"`
	RetryCount   int        `json:"retry_count"`
	MaxRetries   int        `json:"max_retries"`
	ErrorMessage string     `json:"error_message"`
	CreatedAt    time.Time  `json:"created_at"`
	ProcessedAt  *time.Time `json:"processed_at"`
}

type DeviceApproval struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	DeviceID     string     `json:"device_id"`
	DeviceName   string     `json:"device_name"`
	DeviceModel  string     `json:"device_model"`
	IPAddress    string     `json:"ip_address"`
	Fingerprint  string     `json:"fingerprint"`
	Status       string     `json:"status"`
	ApprovedBy   *string    `json:"approved_by"`
	ApprovedAt   *time.Time `json:"approved_at"`
	RejectReason string     `json:"reject_reason"`
	ExpiresAt    time.Time  `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type MobileSummary struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	SummaryType string    `json:"summary_type"`
	Data        []byte    `json:"data"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PushRegistrationRepository struct{ pool *pgxpool.Pool }
type MobileSessionRepository struct{ pool *pgxpool.Pool }
type OfflineSyncRepository struct{ pool *pgxpool.Pool }
type DeviceApprovalRepository struct{ pool *pgxpool.Pool }
type MobileSummaryRepository struct{ pool *pgxpool.Pool }

func NewPushRegistrationRepository(pool *pgxpool.Pool) *PushRegistrationRepository {
	return &PushRegistrationRepository{pool: pool}
}
func NewMobileSessionRepository(pool *pgxpool.Pool) *MobileSessionRepository {
	return &MobileSessionRepository{pool: pool}
}
func NewOfflineSyncRepository(pool *pgxpool.Pool) *OfflineSyncRepository {
	return &OfflineSyncRepository{pool: pool}
}
func NewDeviceApprovalRepository(pool *pgxpool.Pool) *DeviceApprovalRepository {
	return &DeviceApprovalRepository{pool: pool}
}
func NewMobileSummaryRepository(pool *pgxpool.Pool) *MobileSummaryRepository {
	return &MobileSummaryRepository{pool: pool}
}

func (r *PushRegistrationRepository) Register(ctx context.Context, reg *PushRegistration) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO push_registrations (user_id, platform, token, device_name, device_model, os_version, app_version) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (user_id, platform, token) DO UPDATE SET device_name=EXCLUDED.device_name, device_model=EXCLUDED.device_model, os_version=EXCLUDED.os_version, app_version=EXCLUDED.app_version, enabled=true, updated_at=NOW()`, reg.UserID, reg.Platform, reg.Token, reg.DeviceName, reg.DeviceModel, reg.OSVersion, reg.AppVersion)
	return err
}

func (r *PushRegistrationRepository) ListByUser(ctx context.Context, userID string) ([]*PushRegistration, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, user_id, platform, token, device_name, device_model, os_version, app_version, enabled, last_used_at, created_at, updated_at FROM push_registrations WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*PushRegistration
	for rows.Next() {
		var p PushRegistration
		if err := rows.Scan(&p.ID, &p.UserID, &p.Platform, &p.Token, &p.DeviceName, &p.DeviceModel, &p.OSVersion, &p.AppVersion, &p.Enabled, &p.LastUsedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &p)
	}
	return list, nil
}

func (r *PushRegistrationRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM push_registrations WHERE id=$1`, id)
	return err
}

func (r *MobileSessionRepository) Create(ctx context.Context, s *MobileSession) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO mobile_sessions (user_id, device_id, device_name, device_model, os_version, app_version, ip_address, user_agent, biometric_enabled, push_enabled) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT (device_id) DO UPDATE SET last_active_at=NOW(), expires_at=NOW()+INTERVAL '30 days'`, s.UserID, s.DeviceID, s.DeviceName, s.DeviceModel, s.OSVersion, s.AppVersion, s.IPAddress, s.UserAgent, s.BiometricEnabled, s.PushEnabled)
	return err
}

func (r *MobileSessionRepository) ListByUser(ctx context.Context, userID string) ([]*MobileSession, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, user_id, device_id, device_name, device_model, os_version, app_version, ip_address, user_agent, biometric_enabled, push_enabled, last_active_at, expires_at, created_at, updated_at FROM mobile_sessions WHERE user_id=$1 ORDER BY last_active_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*MobileSession
	for rows.Next() {
		var s MobileSession
		if err := rows.Scan(&s.ID, &s.UserID, &s.DeviceID, &s.DeviceName, &s.DeviceModel, &s.OSVersion, &s.AppVersion, &s.IPAddress, &s.UserAgent, &s.BiometricEnabled, &s.PushEnabled, &s.LastActiveAt, &s.ExpiresAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &s)
	}
	return list, nil
}

func (r *MobileSessionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM mobile_sessions WHERE id=$1`, id)
	return err
}

func (r *OfflineSyncRepository) Enqueue(ctx context.Context, item *OfflineSyncItem) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO offline_sync_queue (user_id, device_id, action, payload, priority) VALUES ($1,$2,$3,$4,$5)`, item.UserID, item.DeviceID, item.Action, item.Payload, item.Priority)
	return err
}

func (r *OfflineSyncRepository) ListPending(ctx context.Context, deviceID string, limit int) ([]*OfflineSyncItem, error) {
	rows, err := r.pool.Query(ctx, `UPDATE offline_sync_queue SET status='processing' WHERE id IN (SELECT id FROM offline_sync_queue WHERE device_id=$1 AND status='pending' ORDER BY priority DESC, created_at ASC LIMIT $2 FOR UPDATE SKIP LOCKED) RETURNING id, user_id, device_id, action, payload, priority, status, retry_count, max_retries, error_message, created_at, processed_at`, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*OfflineSyncItem
	for rows.Next() {
		var item OfflineSyncItem
		if err := rows.Scan(&item.ID, &item.UserID, &item.DeviceID, &item.Action, &item.Payload, &item.Priority, &item.Status, &item.RetryCount, &item.MaxRetries, &item.ErrorMessage, &item.CreatedAt, &item.ProcessedAt); err != nil {
			return nil, err
		}
		list = append(list, &item)
	}
	return list, nil
}

func (r *OfflineSyncRepository) Complete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE offline_sync_queue SET status='completed', processed_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *OfflineSyncRepository) Fail(ctx context.Context, id string, errMsg string) error {
	_, err := r.pool.Exec(ctx, `UPDATE offline_sync_queue SET retry_count=retry_count+1, error_message=$2, status=CASE WHEN retry_count+1>=max_retries THEN 'failed' ELSE 'pending' END WHERE id=$1`, id, errMsg)
	return err
}

func (r *DeviceApprovalRepository) Request(ctx context.Context, d *DeviceApproval) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO device_approvals (user_id, device_id, device_name, device_model, ip_address, fingerprint) VALUES ($1,$2,$3,$4,$5,$6)`, d.UserID, d.DeviceID, d.DeviceName, d.DeviceModel, d.IPAddress, d.Fingerprint)
	return err
}

func (r *DeviceApprovalRepository) Approve(ctx context.Context, id, approvedBy string) error {
	_, err := r.pool.Exec(ctx, `UPDATE device_approvals SET status='approved', approved_by=$2, approved_at=NOW(), updated_at=NOW() WHERE id=$1 AND status='pending'`, id, approvedBy)
	return err
}

func (r *DeviceApprovalRepository) Reject(ctx context.Context, id, reason string) error {
	_, err := r.pool.Exec(ctx, `UPDATE device_approvals SET status='rejected', reject_reason=$2, updated_at=NOW() WHERE id=$1 AND status='pending'`, id, reason)
	return err
}

func (r *DeviceApprovalRepository) ListPending(ctx context.Context) ([]*DeviceApproval, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, user_id, device_id, device_name, device_model, ip_address, fingerprint, status, approved_by, approved_at, reject_reason, expires_at, created_at, updated_at FROM device_approvals WHERE status='pending' ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*DeviceApproval
	for rows.Next() {
		var d DeviceApproval
		if err := rows.Scan(&d.ID, &d.UserID, &d.DeviceID, &d.DeviceName, &d.DeviceModel, &d.IPAddress, &d.Fingerprint, &d.Status, &d.ApprovedBy, &d.ApprovedAt, &d.RejectReason, &d.ExpiresAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &d)
	}
	return list, nil
}

func (r *DeviceApprovalRepository) ListByUser(ctx context.Context, userID string) ([]*DeviceApproval, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, user_id, device_id, device_name, device_model, ip_address, fingerprint, status, approved_by, approved_at, reject_reason, expires_at, created_at, updated_at FROM device_approvals WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*DeviceApproval
	for rows.Next() {
		var d DeviceApproval
		if err := rows.Scan(&d.ID, &d.UserID, &d.DeviceID, &d.DeviceName, &d.DeviceModel, &d.IPAddress, &d.Fingerprint, &d.Status, &d.ApprovedBy, &d.ApprovedAt, &d.RejectReason, &d.ExpiresAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &d)
	}
	return list, nil
}

func (r *MobileSummaryRepository) Get(ctx context.Context, userID, summaryType string) (*MobileSummary, error) {
	var s MobileSummary
	err := r.pool.QueryRow(ctx, `SELECT id, user_id, summary_type, data, version, created_at, updated_at FROM mobile_summaries WHERE user_id=$1 AND summary_type=$2`, userID, summaryType).Scan(&s.ID, &s.UserID, &s.SummaryType, &s.Data, &s.Version, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *MobileSummaryRepository) Upsert(ctx context.Context, s *MobileSummary) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO mobile_summaries (user_id, summary_type, data, version) VALUES ($1,$2,$3,$4) ON CONFLICT (user_id, summary_type) DO UPDATE SET data=EXCLUDED.data, version=mobile_summaries.version+1, updated_at=NOW()`, s.UserID, s.SummaryType, s.Data, s.Version)
	return err
}

func (r *MobileSummaryRepository) ListByUser(ctx context.Context, userID string) ([]*MobileSummary, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, user_id, summary_type, data, version, created_at, updated_at FROM mobile_summaries WHERE user_id=$1 ORDER BY summary_type`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*MobileSummary
	for rows.Next() {
		var s MobileSummary
		if err := rows.Scan(&s.ID, &s.UserID, &s.SummaryType, &s.Data, &s.Version, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &s)
	}
	return list, nil
}
