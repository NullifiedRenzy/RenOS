package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TerminalSession struct {
	ID           string     `json:"id"`
	DeviceID     string     `json:"device_id"`
	UserID       string     `json:"user_id"`
	SessionToken string     `json:"-"`
	Status       string     `json:"status"`
	Cols         int        `json:"cols"`
	Rows         int        `json:"rows"`
	Shell        string     `json:"shell"`
	StartedAt    time.Time  `json:"started_at"`
	LastActiveAt time.Time  `json:"last_active_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	CloseReason  string     `json:"close_reason"`
	IPAddress    string     `json:"ip_address"`
	UserAgent    string     `json:"user_agent"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type TerminalRecording struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	Direction  string    `json:"direction"`
	Data       []byte    `json:"-"`
	SeqNum     int64     `json:"seq_num"`
	RecordedAt time.Time `json:"recorded_at"`
}

type TerminalSessionRepository struct {
	pool *pgxpool.Pool
}

func NewTerminalSessionRepository(pool *pgxpool.Pool) *TerminalSessionRepository {
	return &TerminalSessionRepository{pool: pool}
}

func (r *TerminalSessionRepository) Create(ctx context.Context, deviceID, userID, sessionToken, shell, ipAddress, userAgent string, cols, rows int) (*TerminalSession, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO terminal_sessions (device_id, user_id, session_token, cols, rows, shell, ip_address, user_agent)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, device_id, user_id, session_token, status, cols, rows, shell, started_at, last_active_at, ended_at, close_reason, ip_address, user_agent, created_at, updated_at`,
		deviceID, userID, sessionToken, cols, rows, shell, ipAddress, userAgent,
	)
	return scanTerminalSession(row)
}

func (r *TerminalSessionRepository) FindByID(ctx context.Context, id string) (*TerminalSession, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, device_id, user_id, session_token, status, cols, rows, shell, started_at, last_active_at, ended_at, close_reason, ip_address, user_agent, created_at, updated_at
		 FROM terminal_sessions WHERE id = $1`, id,
	)
	return scanTerminalSession(row)
}

func (r *TerminalSessionRepository) ListByDevice(ctx context.Context, deviceID string, limit int) ([]*TerminalSession, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, user_id, session_token, status, cols, rows, shell, started_at, last_active_at, ended_at, close_reason, ip_address, user_agent, created_at, updated_at
		 FROM terminal_sessions WHERE device_id = $1 ORDER BY started_at DESC LIMIT $2`, deviceID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list terminal sessions: %w", err)
	}
	defer rows.Close()
	var list []*TerminalSession
	for rows.Next() {
		s, err := scanTerminalSession(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *TerminalSessionRepository) ListByUser(ctx context.Context, userID string, limit int) ([]*TerminalSession, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, user_id, session_token, status, cols, rows, shell, started_at, last_active_at, ended_at, close_reason, ip_address, user_agent, created_at, updated_at
		 FROM terminal_sessions WHERE user_id = $1 ORDER BY started_at DESC LIMIT $2`, userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list terminal sessions by user: %w", err)
	}
	defer rows.Close()
	var list []*TerminalSession
	for rows.Next() {
		s, err := scanTerminalSession(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *TerminalSessionRepository) UpdateStatus(ctx context.Context, id, status, closeReason string) error {
	endedAt := (*time.Time)(nil)
	if status == "closed" || status == "timeout" || status == "error" {
		now := time.Now()
		endedAt = &now
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE terminal_sessions SET status = $1, close_reason = $2, ended_at = COALESCE($3, ended_at), updated_at = NOW() WHERE id = $4`,
		status, closeReason, endedAt, id,
	)
	return err
}

func (r *TerminalSessionRepository) UpdateLastActive(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE terminal_sessions SET last_active_at = NOW(), updated_at = NOW() WHERE id = $1`, id,
	)
	return err
}

func (r *TerminalSessionRepository) UpdateSize(ctx context.Context, id string, cols, rows int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE terminal_sessions SET cols = $1, rows = $2, updated_at = NOW() WHERE id = $3`, cols, rows, id,
	)
	return err
}

func (r *TerminalSessionRepository) CloseStaleSessions(ctx context.Context, timeout time.Duration) (int64, error) {
	result, err := r.pool.Exec(ctx,
		`UPDATE terminal_sessions SET status = 'timeout', close_reason = 'idle timeout', ended_at = NOW(), updated_at = NOW()
		 WHERE status = 'active' AND last_active_at < NOW() - $1::interval`,
		fmt.Sprintf("%d seconds", int(timeout.Seconds())),
	)
	if err != nil {
		return 0, fmt.Errorf("close stale sessions: %w", err)
	}
	return result.RowsAffected(), nil
}

// --- Recordings ---

type TerminalRecordingRepository struct {
	pool *pgxpool.Pool
}

func NewTerminalRecordingRepository(pool *pgxpool.Pool) *TerminalRecordingRepository {
	return &TerminalRecordingRepository{pool: pool}
}

func (r *TerminalRecordingRepository) Insert(ctx context.Context, sessionID, direction string, data []byte) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO terminal_recordings (session_id, direction, data) VALUES ($1, $2, $3)`,
		sessionID, direction, data,
	)
	return err
}

func (r *TerminalRecordingRepository) ListBySession(ctx context.Context, sessionID string) ([]*TerminalRecording, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, session_id, direction, data, seq_num, recorded_at
		 FROM terminal_recordings WHERE session_id = $1 ORDER BY seq_num`, sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list terminal recordings: %w", err)
	}
	defer rows.Close()
	var list []*TerminalRecording
	for rows.Next() {
		rec := &TerminalRecording{}
		if err := rows.Scan(&rec.ID, &rec.SessionID, &rec.Direction, &rec.Data, &rec.SeqNum, &rec.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan recording: %w", err)
		}
		list = append(list, rec)
	}
	return list, rows.Err()
}

func scanTerminalSession(row pgx.Row) (*TerminalSession, error) {
	s := &TerminalSession{}
	err := row.Scan(&s.ID, &s.DeviceID, &s.UserID, &s.SessionToken, &s.Status, &s.Cols, &s.Rows, &s.Shell, &s.StartedAt, &s.LastActiveAt, &s.EndedAt, &s.CloseReason, &s.IPAddress, &s.UserAgent, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan terminal session: %w", err)
	}
	return s, nil
}
