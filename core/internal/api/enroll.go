package api

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/ca"
	"github.com/renos-project/renos/core/internal/config"
	"github.com/renos-project/renos/core/internal/db"
)

// M06 uses HTTP enrollment as a temporary v0.1 bridge.
// Real gRPC streaming + mTLS comes in M07.
// This endpoint returns a long-term device_secret used by /connect
// until mTLS replaces it.

var tokenChars = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randomToken() string {
	b := make([]byte, 32)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(tokenChars))))
		b[i] = tokenChars[n.Int64()]
	}
	return string(b)
}

func randomHexSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}

type enrollmentToken struct {
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"-"`
}

var (
	tokenMu      sync.RWMutex
	activeTokens = make(map[string]*enrollmentToken)
	maxTokens    = 1000
)

func generateEnrollTokenHandler(cfg *config.Config, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := randomToken()
		et := &enrollmentToken{
			Token:     token,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(cfg.EnrollTokenTTL),
		}

		tokenMu.Lock()
		// Evict expired tokens to prevent unbounded growth
		now := time.Now()
		for t, et := range activeTokens {
			if now.After(et.ExpiresAt) || et.Used {
				delete(activeTokens, t)
			}
		}
		// Enforce max tokens limit
		if len(activeTokens) >= maxTokens {
			// Remove oldest token
			var oldestKey string
			var oldestTime time.Time
			for t, tok := range activeTokens {
				if oldestKey == "" || tok.CreatedAt.Before(oldestTime) {
					oldestKey = t
					oldestTime = tok.CreatedAt
				}
			}
			if oldestKey != "" {
				delete(activeTokens, oldestKey)
			}
		}
		activeTokens[token] = et
		tokenMu.Unlock()

		claims, _ := r.Context().Value(claimsKey).(*auth.Claims)
		var actorID string
		if claims != nil {
			actorID = claims.UserID
		}
		if err := audits.Create(r.Context(), "user", actorID, "enroll_token.generated", "system", ""); err != nil {
			slog.Error("failed to audit token generation", "error", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token":      token,
			"expires_at": et.ExpiresAt,
			"ttl_seconds": int(cfg.EnrollTokenTTL.Seconds()),
		})
	}
}

type enrollRequestBody struct {
	Token          string `json:"token"`
	PublicKeyBase64 string `json:"public_key_base64"`
	Hostname       string `json:"hostname"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	AgentVersion   string `json:"agent_version"`
}

func enrollHandler(caCert *ca.CA, devices *db.DeviceRepository, audits *db.AuditRepository, cfg *config.Config, adminUserID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req enrollRequestBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		if req.Token == "" || req.PublicKeyBase64 == "" || req.Hostname == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "token, public_key_base64, and hostname are required")
			return
		}

		tokenMu.Lock()
		et, exists := activeTokens[req.Token]
		if !exists {
			tokenMu.Unlock()
			writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Enrollment token not found")
			return
		}
		if et.Used {
			tokenMu.Unlock()
			writeError(w, http.StatusUnauthorized, "TOKEN_USED", "Enrollment token already used")
			return
		}
		if time.Now().After(et.ExpiresAt) {
			delete(activeTokens, req.Token)
			tokenMu.Unlock()
			writeError(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "Enrollment token expired")
			return
		}
		et.Used = true
		delete(activeTokens, req.Token)
		tokenMu.Unlock()

		pubKeyDER, err := base64.StdEncoding.DecodeString(req.PublicKeyBase64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_KEY", "Failed to decode public key base64")
			return
		}

		pubKey, err := x509.ParsePKIXPublicKey(pubKeyDER)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_KEY", "Failed to parse public key DER")
			return
		}

		params := ca.SigningParams{
			CommonName:   req.Hostname,
			Organization: "RenOS",
			NotAfter:     time.Now().Add(cfg.CACertTTL),
		}
		certDER, err := caCert.CreateDeviceCert(pubKey, params)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "SIGN_ERROR", "Failed to sign device certificate")
			return
		}

		fingerprint := ca.SHA256Fingerprint(certDER)

		device, err := devices.Create(r.Context(), adminUserID, req.Hostname, req.OS, req.Arch, req.AgentVersion, fingerprint)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create device record")
			return
		}

		deviceSecret, err := randomHexSecret()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "SECRET_ERROR", "Failed to generate device secret")
			return
		}

		secretHash, err := auth.HashPassword(deviceSecret)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "SECRET_ERROR", "Failed to hash device secret")
			return
		}

		metadataJSON := fmt.Sprintf(`{"device_secret_hash":"%s"}`, secretHash)
		if err := devices.UpdateMetadata(r.Context(), device.ID, metadataJSON); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update device metadata")
			return
		}

		audits.Create(r.Context(), "device", device.ID, "device.enrolled", "device", device.ID)

		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
		caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert.CertDER})

		// The device_secret is returned only once during enrollment.
		// The agent must store it securely. The Core only keeps the bcrypt hash.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"device_id":       device.ID,
			"cert_pem":        string(certPEM),
			"ca_cert_pem":     string(caCertPEM),
			"cert_expiry_unix": params.NotAfter.Unix(),
			"device_secret":    deviceSecret,
		})
	}
}
