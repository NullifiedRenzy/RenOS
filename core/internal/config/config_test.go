package config

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	os.Clearenv()

	cfg := Load()

	if cfg.Port != 8443 {
		t.Errorf("expected default port 8443, got %d", cfg.Port)
	}

	if cfg.GRPCPort != 8444 {
		t.Errorf("expected default gRPC port 8444, got %d", cfg.GRPCPort)
	}

	if cfg.DatabaseURL != "postgres://renos:renos@localhost:5432/renos?sslmode=disable" {
		t.Errorf("unexpected default database URL: %s", cfg.DatabaseURL)
	}

	if cfg.NATSUrl != "nats://localhost:4222" {
		t.Errorf("unexpected default NATS URL: %s", cfg.NATSUrl)
	}

	if cfg.RedisURL != "redis://localhost:6379/0" {
		t.Errorf("unexpected default Redis URL: %s", cfg.RedisURL)
	}

	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("expected default log level info, got %v", cfg.LogLevel)
	}

	if cfg.ExternalURL != "https://localhost:8443" {
		t.Errorf("unexpected default external URL: %s", cfg.ExternalURL)
	}

	if cfg.AgentHeartbeatInterval != 30*time.Second {
		t.Errorf("expected default heartbeat interval 30s, got %v", cfg.AgentHeartbeatInterval)
	}

	if cfg.AgentOfflineTimeout != 90*time.Second {
		t.Errorf("expected default offline timeout 90s, got %v", cfg.AgentOfflineTimeout)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Clearenv()

	os.Setenv("RENOS_PORT", "9090")
	os.Setenv("RENOS_DATABASE_URL", "postgres://test:test@db:5432/test")
	os.Setenv("RENOS_NATS_URL", "nats://nats:4222")
	os.Setenv("RENOS_REDIS_URL", "redis://redis:6379/1")
	os.Setenv("RENOS_LOG_LEVEL", "debug")
	os.Setenv("RENOS_EXTERNAL_URL", "https://renos.example.com")
	os.Setenv("RENOS_JWT_SECRET", "custom-secret")
	os.Setenv("RENOS_ENCRYPTION_KEY", "custom-key")

	defer os.Clearenv()

	cfg := Load()

	if cfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Port)
	}

	if cfg.DatabaseURL != "postgres://test:test@db:5432/test" {
		t.Errorf("unexpected database URL: %s", cfg.DatabaseURL)
	}

	if cfg.NATSUrl != "nats://nats:4222" {
		t.Errorf("unexpected NATS URL: %s", cfg.NATSUrl)
	}

	if cfg.RedisURL != "redis://redis:6379/1" {
		t.Errorf("unexpected Redis URL: %s", cfg.RedisURL)
	}

	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("expected debug log level, got %v", cfg.LogLevel)
	}

	if cfg.ExternalURL != "https://renos.example.com" {
		t.Errorf("unexpected external URL: %s", cfg.ExternalURL)
	}

	if cfg.JWTSecret != "custom-secret" {
		t.Errorf("unexpected JWT secret: %s", cfg.JWTSecret)
	}

	if cfg.EncryptionKey != "custom-key" {
		t.Errorf("unexpected encryption key: %s", cfg.EncryptionKey)
	}
}

func TestLoadLogLevelWarn(t *testing.T) {
	os.Clearenv()
	os.Setenv("RENOS_LOG_LEVEL", "warn")
	defer os.Clearenv()

	cfg := Load()

	if cfg.LogLevel != slog.LevelWarn {
		t.Errorf("expected warn log level, got %v", cfg.LogLevel)
	}
}

func TestLoadLogLevelError(t *testing.T) {
	os.Clearenv()
	os.Setenv("RENOS_LOG_LEVEL", "error")
	defer os.Clearenv()

	cfg := Load()

	if cfg.LogLevel != slog.LevelError {
		t.Errorf("expected error log level, got %v", cfg.LogLevel)
	}
}

func TestLoadInvalidPortFallsBackToDefault(t *testing.T) {
	os.Clearenv()
	os.Setenv("RENOS_PORT", "not-a-number")
	defer os.Clearenv()

	cfg := Load()

	if cfg.Port != 8443 {
		t.Errorf("expected fallback port 8443 for invalid env, got %d", cfg.Port)
	}
}

func TestValidateProductionSecrets(t *testing.T) {
	tests := []struct {
		name        string
		jwtSecret   string
		encKey      string
		adminPass   string
		logLevel    slog.Level
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid production config",
			jwtSecret: "a]secure-jwt-secret-that-is-long-enough-for-validation-000",
			encKey:    "a-secure-encryption-key-that-is-long-enough-for-validation-00",
			adminPass: "strongpassword123",
			logLevel:  slog.LevelInfo,
			wantErr:   false,
		},
		{
			name:        "empty JWT secret in production",
			jwtSecret:   "",
			encKey:      "a-secure-encryption-key-that-is-long-enough-for-validation-00",
			adminPass:   "strongpassword123",
			logLevel:    slog.LevelInfo,
			wantErr:     true,
			errContains: "RENOS_JWT_SECRET must not be empty",
		},
		{
			name:        "empty encryption key in production",
			jwtSecret:   "a]secure-jwt-secret-that-is-long-enough-for-validation-000",
			encKey:      "",
			adminPass:   "strongpassword123",
			logLevel:    slog.LevelInfo,
			wantErr:     true,
			errContains: "RENOS_ENCRYPTION_KEY must not be empty",
		},
		{
			name:        "weak admin password in production",
			jwtSecret:   "a]secure-jwt-secret-that-is-long-enough-for-validation-000",
			encKey:      "a-secure-encryption-key-that-is-long-enough-for-validation-00",
			adminPass:   "short",
			logLevel:    slog.LevelInfo,
			wantErr:     true,
			errContains: "at least 8 characters",
		},
		{
			name:        "default admin password in production",
			jwtSecret:   "a]secure-jwt-secret-that-is-long-enough-for-validation-000",
			encKey:      "a-secure-encryption-key-that-is-long-enough-for-validation-00",
			adminPass:   "admin",
			logLevel:    slog.LevelInfo,
			wantErr:     true,
			errContains: "must not be 'admin'",
		},
		{
			name:      "dev secrets bypass validation",
			jwtSecret: "dev-secret-do-not-use-in-production",
			encKey:    "dev-encryption-key-change-in-production",
			adminPass: "admin",
			logLevel:  slog.LevelDebug,
			wantErr:   false,
		},
		{
			name:        "short JWT secret in production",
			jwtSecret:   "tooshort",
			encKey:      "a-secure-encryption-key-that-is-long-enough-for-validation-00",
			adminPass:   "strongpassword123",
			logLevel:    slog.LevelInfo,
			wantErr:     true,
			errContains: "at least 32 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				DatabaseURL:  "postgres://test:test@localhost/test",
				NATSUrl:      "nats://localhost:4222",
				Port:         8443,
				LogLevel:     tt.logLevel,
				JWTSecret:    tt.jwtSecret,
				EncryptionKey: tt.encKey,
				AdminPassword: tt.adminPass,
			}

			err := Validate(cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
