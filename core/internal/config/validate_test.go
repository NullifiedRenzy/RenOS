package config

import "testing"

func TestValidateValidConfig(t *testing.T) {
	cfg := &Config{
		Port:          8443,
		DatabaseURL:   "postgres://renos:renos@localhost:5432/renos?sslmode=disable",
		NATSUrl:       "nats://localhost:4222",
		JWTSecret:     "this-is-a-very-long-and-secure-jwt-secret-key-32-chars",
		EncryptionKey: "this-is-a-very-long-and-secure-encryption-key-32-chars",
		AdminPassword: "secure-admin-password-123",
	}

	if err := Validate(cfg); err != nil {
		t.Errorf("expected valid config to pass validation, got: %v", err)
	}
}

func TestValidateMissingDatabaseURL(t *testing.T) {
	cfg := &Config{
		Port:        8443,
		DatabaseURL: "",
		NATSUrl:     "nats://localhost:4222",
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("expected error for missing database URL")
	}
}

func TestValidateMissingNATS(t *testing.T) {
	cfg := &Config{
		Port:        8443,
		DatabaseURL: "postgres://localhost:5432/renos",
		NATSUrl:     "",
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("expected error for missing NATS URL")
	}
}

func TestValidateInvalidPort(t *testing.T) {
	cfg := &Config{
		Port:        0,
		DatabaseURL: "postgres://localhost:5432/renos",
		NATSUrl:     "nats://localhost:4222",
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("expected error for invalid port")
	}
}

func TestValidateDevSecretsAllowed(t *testing.T) {
	cfg := &Config{
		Port:          8443,
		DatabaseURL:   "postgres://localhost:5432/renos",
		NATSUrl:       "nats://localhost:4222",
		JWTSecret:     "dev-secret-do-not-use-in-production",
		EncryptionKey: "dev-encryption-key-change-in-production",
	}

	if err := Validate(cfg); err != nil {
		t.Errorf("expected dev secrets to be allowed, got: %v", err)
	}
}

func TestIsProduction(t *testing.T) {
	dev := &Config{
		JWTSecret:     "dev-secret-do-not-use-in-production",
		EncryptionKey: "dev-encryption-key-change-in-production",
	}
	if dev.IsProduction() {
		t.Error("expected dev config to not be production")
	}

	prod := &Config{
		JWTSecret:     "real-secret-key-that-is-at-least-32-characters",
		EncryptionKey: "real-encryption-key-that-is-at-least-32-chars",
	}
	if !prod.IsProduction() {
		t.Error("expected prod config to be production")
	}
}
