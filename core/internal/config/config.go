package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port int

	DatabaseURL string
	NATSUrl     string
	RedisURL    string

	JWTSecret       string
	EncryptionKey   string
	ExternalURL     string
	ExternalURLPath string

	LogLevel slog.Level

	GRPCPort int

	CACertDir    string
	CACertTTL    time.Duration
	EnrollTokenTTL time.Duration

	AgentHeartbeatInterval time.Duration
	AgentOfflineTimeout    time.Duration

	AdminEmail    string
	AdminPassword string
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func Load() *Config {
	logLevel := slog.LevelInfo
	switch strings.ToLower(getEnv("RENOS_LOG_LEVEL", "info")) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	return &Config{
		Port: getEnvInt("RENOS_PORT", 8443),
		GRPCPort: getEnvInt("RENOS_GRPC_PORT", 8444),

		DatabaseURL: getEnv("RENOS_DATABASE_URL", "postgres://renos:renos@localhost:5432/renos?sslmode=disable"),
		NATSUrl:     getEnv("RENOS_NATS_URL", "nats://localhost:4222"),
		RedisURL:    getEnv("RENOS_REDIS_URL", "redis://localhost:6379/0"),

		JWTSecret:     getEnv("RENOS_JWT_SECRET", "dev-secret-do-not-use-in-production"),
		EncryptionKey: getEnv("RENOS_ENCRYPTION_KEY", "dev-encryption-key-change-in-production"),
		ExternalURL:   getEnv("RENOS_EXTERNAL_URL", "https://localhost:8443"),

		LogLevel: logLevel,

		CACertDir:    getEnv("RENOS_CA_CERT_DIR", "/var/lib/renos/ca"),
		CACertTTL:    365 * 24 * time.Hour,
		EnrollTokenTTL: 15 * time.Minute,

		AgentHeartbeatInterval: 30 * time.Second,
		AgentOfflineTimeout:    90 * time.Second,

		AdminEmail:    getEnv("RENOS_ADMIN_EMAIL", "admin@renos.local"),
		AdminPassword: getEnv("RENOS_ADMIN_PASSWORD", "admin"),
	}
}
