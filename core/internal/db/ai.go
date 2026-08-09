package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AIConversation struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Messages  []byte    `json:"messages"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AIConversationRepository struct {
	pool *pgxpool.Pool
}

func NewAIConversationRepository(pool *pgxpool.Pool) *AIConversationRepository {
	return &AIConversationRepository{pool: pool}
}

func (r *AIConversationRepository) Create(ctx context.Context, userID, title string) (*AIConversation, error) {
	conv := &AIConversation{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO ai_conversations (user_id, title) VALUES ($1, $2)
		 RETURNING id, user_id, title, messages, created_at, updated_at`,
		userID, title,
	).Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.Messages, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create ai_conversation: %w", err)
	}
	return conv, nil
}

func (r *AIConversationRepository) FindByID(ctx context.Context, id string) (*AIConversation, error) {
	conv := &AIConversation{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, title, messages, created_at, updated_at
		 FROM ai_conversations WHERE id = $1`, id,
	).Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.Messages, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("find ai_conversation: %w", err)
	}
	return conv, nil
}

func (r *AIConversationRepository) ListByUser(ctx context.Context, userID string) ([]*AIConversation, error) {	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, title, messages, created_at, updated_at
		 FROM ai_conversations WHERE user_id = $1 ORDER BY updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list ai_conversations: %w", err)
	}
	defer rows.Close()

	var convs []*AIConversation
	for rows.Next() {
		c := &AIConversation{}
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.Messages, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan ai_conversation: %w", err)
		}
		convs = append(convs, c)
	}
	return convs, rows.Err()
}

func (r *AIConversationRepository) AppendMessage(ctx context.Context, id string, messages []byte) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE ai_conversations SET messages = $1, updated_at = NOW() WHERE id = $2`,
		messages, id,
	)
	if err != nil {
		return fmt.Errorf("append message: %w", err)
	}
	return nil
}

func (r *AIConversationRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM ai_conversations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete ai_conversation: %w", err)
	}
	return nil
}

// Count returns the total number of AI conversations across all users.
func (r *AIConversationRepository) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM ai_conversations`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count ai_conversations: %w", err)
	}
	return n, nil
}
