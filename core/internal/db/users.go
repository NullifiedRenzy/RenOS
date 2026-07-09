package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	Password     string     `json:"-"`
	DisplayName  string     `json:"display_name"`
	Role         string     `json:"role"`
	Settings     string     `json:"settings"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, email, password, displayName, role string) (*User, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password, display_name, role)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, email, password, display_name, role, settings, created_at, updated_at, deleted_at`,
		email, password, displayName, role,
	)

	u := &User{}
	err := row.Scan(&u.ID, &u.Email, &u.Password, &u.DisplayName, &u.Role, &u.Settings, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, password, display_name, role, settings, created_at, updated_at, deleted_at
		 FROM users WHERE email = $1 AND deleted_at IS NULL`,
		email,
	)

	u := &User{}
	err := row.Scan(&u.ID, &u.Email, &u.Password, &u.DisplayName, &u.Role, &u.Settings, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return u, nil
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)`,
		email,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user exists: %w", err)
	}
	return exists, nil
}
