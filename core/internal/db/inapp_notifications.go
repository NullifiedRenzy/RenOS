package db

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// InAppNotification is a user-facing notification shown in the notification
// center (M66). Type is one of: alert, workflow, job, ai, system, mention.
type InAppNotification struct {
	ID         string          `json:"id"`
	UserID     string          `json:"user_id"`
	Title      string          `json:"title"`
	Body       string          `json:"body"`
	Type       string          `json:"type"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Metadata   json.RawMessage `json:"metadata"`
	IsRead     bool            `json:"is_read"`
	ReadAt     *time.Time      `json:"read_at,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

type InAppNotificationFilter struct {
	UnreadOnly bool
	Type       string
	Search     string
	Limit      int
	Offset     int
}

type InAppNotificationRepository struct {
	pool *pgxpool.Pool
}

func NewInAppNotificationRepository(pool *pgxpool.Pool) *InAppNotificationRepository {
	return &InAppNotificationRepository{pool: pool}
}

func (r *InAppNotificationRepository) Create(ctx context.Context, n *InAppNotification) error {
	if n.Metadata == nil {
		n.Metadata = json.RawMessage(`{}`)
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO inapp_notifications (user_id, title, body, type, entity_type, entity_id, metadata)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, created_at`,
		n.UserID, n.Title, n.Body, n.Type, n.EntityType, n.EntityID, n.Metadata,
	)
	return row.Scan(&n.ID, &n.CreatedAt)
}

func (r *InAppNotificationRepository) ListForUser(ctx context.Context, userID string, f InAppNotificationFilter) ([]*InAppNotification, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 30
	}
	var b strings.Builder
	var args []interface{}
	b.WriteString(`SELECT id, user_id, title, body, type, entity_type, entity_id, metadata, is_read, read_at, created_at
		FROM inapp_notifications WHERE user_id = $1`)
	args = append(args, userID)
	if f.UnreadOnly {
		args = append(args, false)
		b.WriteString(` AND is_read = $` + itoa(len(args)))
	}
	if f.Type != "" {
		args = append(args, f.Type)
		b.WriteString(` AND type = $` + itoa(len(args)))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		b.WriteString(` AND (title ILIKE $` + itoa(len(args)) + ` OR body ILIKE $` + itoa(len(args)) + `)`)
	}
	args = append(args, f.Limit, f.Offset)
	b.WriteString(` ORDER BY created_at DESC LIMIT $` + itoa(len(args)-1) + ` OFFSET $` + itoa(len(args)))

	rows, err := r.pool.Query(ctx, b.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*InAppNotification
	for rows.Next() {
		var n InAppNotification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Body, &n.Type, &n.EntityType,
			&n.EntityID, &n.Metadata, &n.IsRead, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &n)
	}
	return list, rows.Err()
}

func (r *InAppNotificationRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM inapp_notifications WHERE user_id = $1 AND is_read = FALSE`, userID,
	).Scan(&count)
	return count, err
}

func (r *InAppNotificationRepository) MarkRead(ctx context.Context, userID, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE inapp_notifications SET is_read = TRUE, read_at = NOW()
		 WHERE id = $1 AND user_id = $2`, id, userID,
	)
	return err
}

func (r *InAppNotificationRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE inapp_notifications SET is_read = TRUE, read_at = NOW()
		 WHERE user_id = $1 AND is_read = FALSE`, userID,
	)
	return err
}

func (r *InAppNotificationRepository) FindByID(ctx context.Context, id string) (*InAppNotification, error) {
	var n InAppNotification
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, title, body, type, entity_type, entity_id, metadata, is_read, read_at, created_at
		 FROM inapp_notifications WHERE id = $1`, id,
	).Scan(&n.ID, &n.UserID, &n.Title, &n.Body, &n.Type, &n.EntityType,
		&n.EntityID, &n.Metadata, &n.IsRead, &n.ReadAt, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits [8]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}
