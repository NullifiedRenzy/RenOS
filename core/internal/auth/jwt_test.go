package auth

import (
	"testing"
	"time"
)

func TestGenerateTokenWithTTL(t *testing.T) {
	before := time.Now()
	token, err := GenerateTokenWithTTL("test-secret", "user-1", "admin@example.test", "admin", 30*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := ValidateToken("test-secret", token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-1" || claims.Email != "admin@example.test" || claims.Role != "admin" {
		t.Fatalf("unexpected claims: %#v", claims)
	}

	wantMin := before.Add(30 * 24 * time.Hour).Add(-time.Minute)
	wantMax := before.Add(30 * 24 * time.Hour).Add(time.Minute)
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(wantMin) || claims.ExpiresAt.Time.After(wantMax) {
		t.Fatalf("unexpected expiry: %v", claims.ExpiresAt)
	}
}
