package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	Password    string     `json:"-"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	Settings    string     `json:"settings"`
	TOTPSecret  *string    `json:"-"`
	TOTPEnabled bool       `json:"totp_enabled"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
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
		 RETURNING id, email, password, display_name, role, settings, totp_secret, totp_enabled, created_at, updated_at, deleted_at`,
		email, password, displayName, role,
	)

	u := &User{}
	err := row.Scan(&u.ID, &u.Email, &u.Password, &u.DisplayName, &u.Role, &u.Settings, &u.TOTPSecret, &u.TOTPEnabled, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, password, display_name, role, settings, totp_secret, totp_enabled, created_at, updated_at, deleted_at
		 FROM users WHERE email = $1 AND deleted_at IS NULL`,
		email,
	)

	u := &User{}
	err := row.Scan(&u.ID, &u.Email, &u.Password, &u.DisplayName, &u.Role, &u.Settings, &u.TOTPSecret, &u.TOTPEnabled, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return u, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, password, display_name, role, settings, totp_secret, totp_enabled, created_at, updated_at, deleted_at
		 FROM users WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	u := &User{}
	err := row.Scan(&u.ID, &u.Email, &u.Password, &u.DisplayName, &u.Role, &u.Settings, &u.TOTPSecret, &u.TOTPEnabled, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return u, nil
}

func (r *UserRepository) UpdateRole(ctx context.Context, userID, role string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET role = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		userID, role,
	)
	if err != nil {
		return fmt.Errorf("update user role: %w", err)
	}
	return nil
}

func (r *UserRepository) SetTOTPSecret(ctx context.Context, userID, secret string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET totp_secret = $2, totp_enabled = false, updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`,
		userID, secret,
	)
	if err != nil {
		return fmt.Errorf("set totp secret: %w", err)
	}
	return nil
}

func (r *UserRepository) EnableTOTP(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET totp_enabled = true, updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL AND totp_secret IS NOT NULL`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("enable totp: %w", err)
	}
	return nil
}

func (r *UserRepository) DisableTOTP(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET totp_secret = NULL, totp_enabled = false, updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("disable totp: %w", err)
	}
	return nil
}

func (r *UserRepository) ListAll(ctx context.Context, limit, offset int) ([]*User, error) {
	var lim *int
	if limit > 0 {
		lim = &limit
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, email, display_name, role, settings, totp_enabled, created_at, updated_at, deleted_at
		 FROM users WHERE deleted_at IS NULL ORDER BY created_at ASC LIMIT $1 OFFSET $2`,
		lim, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var list []*User
	for rows.Next() {
		u := &User{}
		err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.Settings, &u.TOTPEnabled, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

func ValidRoles() []string {
	return []string{"admin", "operator", "viewer"}
}

func IsValidRole(role string) bool {
	for _, r := range ValidRoles() {
		if r == role {
			return true
		}
	}
	return false
}

// RoleWeight returns a numeric weight for role comparison (higher = more privileges).
func RoleWeight(role string) int {
	switch role {
	case "admin":
		return 100
	case "operator":
		return 50
	case "viewer":
		return 10
	default:
		return 0
	}
}

// CanAssignRole checks if the caller (with callerRole) can assign the targetRole.
// A user cannot assign a role with higher weight than their own.
func CanAssignRole(callerRole, targetRole string) bool {
	return RoleWeight(callerRole) >= RoleWeight(targetRole)
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
