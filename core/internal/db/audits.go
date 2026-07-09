package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	ActorType string    `json:"actor_type"`
	ActorID   string    `json:"actor_id"`
	Action    string    `json:"action"`
	TargetType string   `json:"target_type"`
	TargetID  string   `json:"target_id"`
	Details   string    `json:"details"`
	Signature *string   `json:"signature,omitempty"`
	PrevHash  *string   `json:"prev_hash,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type AuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

func (r *AuditRepository) Create(ctx context.Context, actorType, actorID, action, targetType, targetID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO audits (actor_type, actor_id, action, target_type, target_id)
		 VALUES ($1, $2, $3, $4, $5)`,
		actorType, actorID, action, targetType, targetID,
	)
	if err != nil {
		return fmt.Errorf("create audit entry: %w", err)
	}
	return nil
}
