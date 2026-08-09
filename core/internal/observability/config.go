package observability

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type TracingConfig struct {
	Enabled     bool
	Exporter    string
	Endpoint    string
	ServiceName string
	Environment string
	SampleRate  float64
}

type MetricsConfig struct {
	Enabled      bool
	Path         string
	Aggregation  time.Duration
}

type PprofConfig struct {
	Enabled bool
	Path    string
}

type SLOConfig struct {
	Enabled bool
	Path    string
}

type HealthConfig struct {
	DeepEnabled bool
	Path        string
}

type ObservabilityConfig struct {
	Tracing  TracingConfig
	Metrics  MetricsConfig
	Pprof    PprofConfig
	SLO      SLOConfig
	Health   HealthConfig
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	switch strings.ToLower(os.Getenv(key)) {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func LoadObservabilityConfig() *ObservabilityConfig {
	cfg := &ObservabilityConfig{
		Tracing: TracingConfig{
			Enabled:     getEnvBool("RENOS_TRACING_ENABLED", false),
			Exporter:    getEnv("RENOS_TRACING_EXPORTER", "otlp"),
			Endpoint:    getEnv("RENOS_TRACING_ENDPOINT", "http://localhost:4318"),
			ServiceName: getEnv("RENOS_SERVICE_NAME", "renos-core"),
			Environment: getEnv("RENOS_ENVIRONMENT", "production"),
			SampleRate:  getEnvFloat("RENOS_TRACING_SAMPLE_RATE", 0.1),
		},
		Metrics: MetricsConfig{
			Enabled:     getEnvBool("RENOS_METRICS_ENABLED", true),
			Path:        getEnv("RENOS_METRICS_PATH", "/metrics"),
			Aggregation: getEnvDuration("RENOS_METRICS_AGGREGATION", 15*time.Second),
		},
		Pprof: PprofConfig{
			Enabled: getEnvBool("RENOS_PPROF_ENABLED", false),
			Path:    getEnv("RENOS_PPROF_PATH", "/debug/pprof"),
		},
		SLO: SLOConfig{
			Enabled: getEnvBool("RENOS_SLO_ENABLED", true),
			Path:    getEnv("RENOS_SLO_PATH", "/api/v1/slos"),
		},
		Health: HealthConfig{
			DeepEnabled: getEnvBool("RENOS_DEEP_HEALTH_ENABLED", true),
			Path:        getEnv("RENOS_DEEP_HEALTH_PATH", "/api/v1/deephealth"),
		},
	}

	slog.Info("observability config loaded",
		"tracing_enabled", cfg.Tracing.Enabled,
		"tracing_exporter", cfg.Tracing.Exporter,
		"metrics_enabled", cfg.Metrics.Enabled,
		"pprof_enabled", cfg.Pprof.Enabled,
		"slo_enabled", cfg.SLO.Enabled,
	)

	return cfg
}
