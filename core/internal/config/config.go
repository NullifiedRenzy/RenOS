package config

import (
	"log/slog"
	"net/url"
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

	CACertDir      string
	CACertTTL      time.Duration
	EnrollTokenTTL time.Duration

	AgentHeartbeatInterval time.Duration
	AgentOfflineTimeout    time.Duration

	AdminEmail    string
	AdminPassword string

	RateLimitLogin          int
	RateLimitLoginBurst     int
	RateLimitAPI            int
	RateLimitAPIBurst       int
	RateLimitWSConnect      int
	RateLimitWSConnectBurst int
	MaxWSConnsPerUser       int
	MaxBodySize             int

	PoolSize                    int
	MaxConcurrentJobs           int
	MaxConcurrentDeviceActions  int
	DeviceActionCooldown        time.Duration
	CircuitBreakerThreshold     int
	CircuitBreakerResetAfter    time.Duration
	DefaultMaxJobDuration       time.Duration
	RetryableErrorCodes         []string
	DefaultBatchSize            int
	DefaultBatchDelayMs         int
	FailureContainmentThreshold float64

	PluginDir string

	AIProvider       string
	AIOllamaURL      string
	AIOllamaModel    string
	AIOpenAIKey      string
	AIOpenAIModel    string
	AIOpenAIBaseURL  string
	AIAnthropicKey   string
	AIAnthropicModel string
	AIGoogleKey      string
	AIGoogleModel    string
	AIAIEndpoint     string
	AIAIModel        string
	AITemperature    float64
	AIMaxTokens      int
	AITimeout        time.Duration

	// Production readiness
	TelemetryEnabled  bool
	TelemetryEndpoint string
	CrashDir          string
	BackupsDir        string

	// Observability & SRE
	TracingEnabled    bool
	TracingExporter   string
	TracingEndpoint   string
	TracingSampleRate float64
	MetricsEnabled    bool
	MetricsPath       string
	PprofEnabled      bool
	PprofPath         string
	SLOEnabled        bool
	SLOPath           string
	DeepHealthEnabled bool
	DeepHealthPath    string
	ServiceName       string
	Environment       string
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}

func parseStringSliceEnv(key, fallback string) []string {
	v := os.Getenv(key)
	if v == "" {
		v = fallback
	}
	parts := strings.Split(v, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func parseFloatEnv(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
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

func sanitizeDatabaseURL(rawURL string) string {
	schemeEnd := strings.Index(rawURL, "://")
	if schemeEnd < 0 {
		return rawURL
	}
	scheme := rawURL[:schemeEnd+3]
	rest := rawURL[schemeEnd+3:]

	atIdx := strings.LastIndex(rest, "@")
	if atIdx < 0 {
		return rawURL
	}
	userinfo := rest[:atIdx]
	hostDB := rest[atIdx+1:]

	colonIdx := strings.Index(userinfo, ":")
	if colonIdx < 0 {
		return rawURL
	}
	user := userinfo[:colonIdx]
	pass := userinfo[colonIdx+1:]

	return scheme + user + ":" + url.QueryEscape(pass) + "@" + hostDB
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
		Port:     getEnvInt("RENOS_PORT", 8443),
		GRPCPort: getEnvInt("RENOS_GRPC_PORT", 8444),

		DatabaseURL: sanitizeDatabaseURL(getEnv("RENOS_DATABASE_URL", "postgres://renos:renos@localhost:5432/renos?sslmode=disable")),
		NATSUrl:     getEnv("RENOS_NATS_URL", "nats://localhost:4222"),
		RedisURL:    getEnv("RENOS_REDIS_URL", "redis://localhost:6379/0"),

		JWTSecret:     getEnv("RENOS_JWT_SECRET", "dev-secret-do-not-use-in-production"),
		EncryptionKey: getEnv("RENOS_ENCRYPTION_KEY", "dev-encryption-key-change-in-production"),
		ExternalURL:   getEnv("RENOS_EXTERNAL_URL", "https://localhost:8443"),

		LogLevel: logLevel,

		CACertDir:      getEnv("RENOS_CA_CERT_DIR", "/var/lib/renos/ca"),
		CACertTTL:      365 * 24 * time.Hour,
		EnrollTokenTTL: 15 * time.Minute,

		AgentHeartbeatInterval: 30 * time.Second,
		AgentOfflineTimeout:    90 * time.Second,

		AdminEmail:    getEnv("RENOS_ADMIN_EMAIL", "admin@renos.local"),
		AdminPassword: getEnv("RENOS_ADMIN_PASSWORD", "admin"),

		RateLimitLogin:          getEnvInt("RENOS_RATE_LIMIT_LOGIN", 5),
		RateLimitLoginBurst:     getEnvInt("RENOS_RATE_LIMIT_LOGIN_BURST", 10),
		RateLimitAPI:            getEnvInt("RENOS_RATE_LIMIT_API", 50),
		RateLimitAPIBurst:       getEnvInt("RENOS_RATE_LIMIT_API_BURST", 100),
		RateLimitWSConnect:      getEnvInt("RENOS_RATE_LIMIT_WS_CONNECT", 2),
		RateLimitWSConnectBurst: getEnvInt("RENOS_RATE_LIMIT_WS_CONNECT_BURST", 5),
		MaxWSConnsPerUser:       getEnvInt("RENOS_MAX_WS_CONNS_PER_USER", 10),
		MaxBodySize:             getEnvInt("RENOS_MAX_BODY_SIZE", 1048576),

		PoolSize:                    getEnvInt("RENOS_POOL_SIZE", 5),
		MaxConcurrentJobs:           getEnvInt("RENOS_MAX_CONCURRENT_JOBS", 10),
		MaxConcurrentDeviceActions:  getEnvInt("RENOS_MAX_CONCURRENT_DEVICE_ACTIONS", 20),
		DeviceActionCooldown:        parseDurationEnv("RENOS_DEVICE_ACTION_COOLDOWN", 5*time.Second),
		CircuitBreakerThreshold:     getEnvInt("RENOS_CIRCUIT_BREAKER_THRESHOLD", 5),
		CircuitBreakerResetAfter:    parseDurationEnv("RENOS_CIRCUIT_BREAKER_RESET_AFTER", 10*time.Minute),
		DefaultMaxJobDuration:       parseDurationEnv("RENOS_DEFAULT_MAX_JOB_DURATION", 30*time.Minute),
		RetryableErrorCodes:         parseStringSliceEnv("RENOS_RETRYABLE_ERROR_CODES", "timeout,network_unreachable,connection_refused,agent_unavailable,temporary_failure"),
		DefaultBatchSize:            getEnvInt("RENOS_DEFAULT_BATCH_SIZE", 5),
		DefaultBatchDelayMs:         getEnvInt("RENOS_DEFAULT_BATCH_DELAY_MS", 2000),
		FailureContainmentThreshold: parseFloatEnv("RENOS_FAILURE_CONTAINMENT_THRESHOLD", 0.5),

		PluginDir: getEnv("RENOS_PLUGIN_DIR", "./plugins"),

		AIProvider:       getEnv("RENOS_AI_PROVIDER", "ollama"),
		AIOllamaURL:      getEnv("RENOS_AI_OLLAMA_URL", "http://localhost:11434"),
		AIOllamaModel:    getEnv("RENOS_AI_OLLAMA_MODEL", "llama3.2"),
		AIOpenAIKey:      getEnv("RENOS_AI_OPENAI_KEY", ""),
		AIOpenAIModel:    getEnv("RENOS_AI_OPENAI_MODEL", "gpt-4o"),
		AIOpenAIBaseURL:  getEnv("RENOS_AI_OPENAI_BASE_URL", "https://api.openai.com/v1"),
		AIAnthropicKey:   getEnv("RENOS_AI_ANTHROPIC_KEY", ""),
		AIAnthropicModel: getEnv("RENOS_AI_ANTHROPIC_MODEL", "claude-sonnet-4-20250514"),
		AIGoogleKey:      getEnv("RENOS_AI_GOOGLE_KEY", ""),
		AIGoogleModel:    getEnv("RENOS_AI_GOOGLE_MODEL", "gemini-2.0-flash"),
		AIAIEndpoint:     getEnv("RENOS_AI_ENDPOINT", ""),
		AIAIModel:        getEnv("RENOS_AI_MODEL", ""),
		AITemperature:    parseFloatEnv("RENOS_AI_TEMPERATURE", 0.7),
		AIMaxTokens:      getEnvInt("RENOS_AI_MAX_TOKENS", 2048),
		AITimeout:        parseDurationEnv("RENOS_AI_TIMEOUT", 120*time.Second),

		TracingEnabled:  getEnv("RENOS_TRACING_ENABLED", "false") == "true",
		TracingExporter: getEnv("RENOS_TRACING_EXPORTER", "otlp"),
		TracingEndpoint: getEnv("RENOS_TRACING_ENDPOINT", "http://localhost:4318"),

		TelemetryEnabled:  getEnv("RENOS_TELEMETRY_ENABLED", "false") == "true",
		TelemetryEndpoint: getEnv("RENOS_TELEMETRY_ENDPOINT", "https://telemetry.renos.io/v1/ping"),
		CrashDir:          getEnv("RENOS_CRASH_DIR", "/var/lib/renos/crashes"),
		BackupsDir:        getEnv("RENOS_BACKUPS_DIR", "/var/lib/renos/backups"),
		TracingSampleRate: parseFloatEnv("RENOS_TRACING_SAMPLE_RATE", 0.1),
		MetricsEnabled:    getEnv("RENOS_METRICS_ENABLED", "true") == "true",
		MetricsPath:       getEnv("RENOS_METRICS_PATH", "/metrics"),
		PprofEnabled:      getEnv("RENOS_PPROF_ENABLED", "false") == "true",
		PprofPath:         getEnv("RENOS_PPROF_PATH", "/debug/pprof"),
		SLOEnabled:        getEnv("RENOS_SLO_ENABLED", "true") == "true",
		SLOPath:           getEnv("RENOS_SLO_PATH", "/api/v1/slos"),
		DeepHealthEnabled: getEnv("RENOS_DEEP_HEALTH_ENABLED", "true") == "true",
		DeepHealthPath:    getEnv("RENOS_DEEP_HEALTH_PATH", "/api/v1/deephealth"),
		ServiceName:       getEnv("RENOS_SERVICE_NAME", "renos-core"),
		Environment:       getEnv("RENOS_ENVIRONMENT", "production"),
	}
}
