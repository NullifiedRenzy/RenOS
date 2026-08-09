package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

var (
	ErrMissingDatabaseURL = errors.New("RENOS_DATABASE_URL is required")
	ErrMissingNATSURL     = errors.New("RENOS_NATS_URL is required")
	ErrWeakJWTSecret      = errors.New("RENOS_JWT_SECRET must be at least 32 characters in production")
	ErrWeakEncryptionKey  = errors.New("RENOS_ENCRYPTION_KEY must be at least 32 characters in production")
	ErrEmptyJWTSecret     = errors.New("RENOS_JWT_SECRET must not be empty in production")
	ErrEmptyEncryptionKey = errors.New("RENOS_ENCRYPTION_KEY must not be empty in production")
	ErrWeakAdminPassword  = errors.New("RENOS_ADMIN_PASSWORD must be at least 8 characters in production")
	ErrDefaultAdminPass   = errors.New("RENOS_ADMIN_PASSWORD must not be 'admin' in production")
	ErrInvalidPort        = errors.New("RENOS_PORT must be between 1 and 65535")
	ErrInvalidLogLevel    = errors.New("RENOS_LOG_LEVEL must be one of: debug, info, warn, error")
	ErrProductionDefaults = errors.New("using default secrets in production — change RENOS_JWT_SECRET and RENOS_ENCRYPTION_KEY")
)

func Validate(cfg *Config) error {
	var errs []error

	if cfg.DatabaseURL == "" {
		errs = append(errs, ErrMissingDatabaseURL)
	}

	if cfg.NATSUrl == "" {
		errs = append(errs, ErrMissingNATSURL)
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		errs = append(errs, ErrInvalidPort)
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	levelStr := strings.ToLower(os.Getenv("RENOS_LOG_LEVEL"))
	if levelStr != "" && !validLevels[levelStr] {
		errs = append(errs, fmt.Errorf("%w: got %q", ErrInvalidLogLevel, levelStr))
	}

	if !cfg.isDevSecrets() && cfg.LogLevel != slog.LevelDebug {
		if cfg.JWTSecret == "" {
			errs = append(errs, ErrEmptyJWTSecret)
		} else if len(cfg.JWTSecret) < 32 {
			errs = append(errs, ErrWeakJWTSecret)
		}
		if cfg.EncryptionKey == "" {
			errs = append(errs, ErrEmptyEncryptionKey)
		} else if len(cfg.EncryptionKey) < 32 {
			errs = append(errs, ErrWeakEncryptionKey)
		}
		if cfg.AdminPassword == "" {
			errs = append(errs, ErrWeakAdminPassword)
		} else if cfg.AdminPassword == "admin" {
			errs = append(errs, ErrDefaultAdminPass)
		} else if len(cfg.AdminPassword) < 8 {
			errs = append(errs, ErrWeakAdminPassword)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration errors:\n  - %w", errors.Join(errs...))
	}
	return nil
}

func (c *Config) isDevSecrets() bool {
	return c.JWTSecret == "dev-secret-do-not-use-in-production" ||
		c.EncryptionKey == "dev-encryption-key-change-in-production"
}

func (c *Config) IsProduction() bool {
	return !c.isDevSecrets()
}
