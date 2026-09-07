package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
)

type loginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	TOTPCode   string `json:"totp_code"`
	RememberMe bool   `json:"remember_me"`
}

type loginResponse struct {
	Token       string `json:"token,omitempty"`
	MFARequired bool   `json:"mfa_required,omitempty"`
	User        struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
		TOTPEnabled bool   `json:"totp_enabled"`
	} `json:"user"`
}

type mfaSetupResponse struct {
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauth_url"`
}

func loginHandler(users *db.UserRepository, jwtSecret string, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		if req.Email == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Email and password required")
			return
		}

		user, err := users.FindByEmail(r.Context(), req.Email)
		if err != nil {
			slog.Warn("login failed: user not found", "email", req.Email)
			if err := audits.Create(r.Context(), "user", req.Email, "login.failed", "user", ""); err != nil {
				slog.Error("failed to audit login failure", "error", err)
			}
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid email or password")
			return
		}

		if !auth.VerifyPassword(user.Password, req.Password) {
			slog.Warn("login failed: wrong password", "email", req.Email, "user_id", user.ID)
			if err := audits.Create(r.Context(), "user", req.Email, "login.failed", "user", user.ID); err != nil {
				slog.Error("failed to audit login failure", "error", err)
			}
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid email or password")
			return
		}

		if user.TOTPEnabled {
			if req.TOTPCode == "" {
				if err := audits.Create(r.Context(), "user", user.ID, "login.mfa_required", "user", user.ID); err != nil {
					slog.Error("failed to audit mfa challenge", "error", err)
				}
				writeJSONResponse(w, http.StatusAccepted, loginResponse{MFARequired: true})
				return
			}
			if user.TOTPSecret == nil || !auth.ValidateTOTP(*user.TOTPSecret, req.TOTPCode, time.Now()) {
				slog.Warn("login failed: invalid mfa code", "email", req.Email, "user_id", user.ID)
				if err := audits.Create(r.Context(), "user", user.ID, "login.mfa_failed", "user", user.ID); err != nil {
					slog.Error("failed to audit mfa failure", "error", err)
				}
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authentication code")
				return
			}
		}

		tokenTTL := 24 * time.Hour
		if req.RememberMe {
			tokenTTL = 30 * 24 * time.Hour
		}
		token, err := auth.GenerateTokenWithTTL(jwtSecret, user.ID, user.Email, user.Role, tokenTTL)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate token")
			return
		}

		slog.Info("login success", "email", req.Email, "user_id", user.ID, "role", user.Role)
		if err := audits.Create(r.Context(), "user", user.ID, "login.success", "user", user.ID); err != nil {
			slog.Error("failed to audit login success", "error", err)
		}

		resp := loginResponse{Token: token}
		resp.User.ID = user.ID
		resp.User.Email = user.Email
		resp.User.DisplayName = user.DisplayName
		resp.User.Role = user.Role
		resp.User.TOTPEnabled = user.TOTPEnabled

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func setupMFAHandler(users *db.UserRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		user, err := users.FindByID(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}
		if user.TOTPEnabled {
			writeError(w, http.StatusConflict, "MFA_ALREADY_ENABLED", "MFA is already enabled")
			return
		}

		secret, err := auth.GenerateTOTPSecret()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate MFA secret")
			return
		}
		if err := users.SetTOTPSecret(r.Context(), userID, secret); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to save MFA secret")
			return
		}
		if err := audits.Create(r.Context(), "user", userID, "mfa.setup", "user", userID); err != nil {
			slog.Error("failed to audit mfa setup", "error", err)
		}
		writeJSON(w, http.StatusOK, mfaSetupResponse{
			Secret:     secret,
			OTPAuthURL: auth.TOTPAuthURL("RenOS", user.Email, secret),
		})
	}
}

func enableMFAHandler(users *db.UserRepository, audits *db.AuditRepository) http.HandlerFunc {
	type request struct {
		Code string `json:"code"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		userID := getUserID(r)
		user, err := users.FindByID(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}
		if user.TOTPSecret == nil || !auth.ValidateTOTP(*user.TOTPSecret, req.Code, time.Now()) {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authentication code")
			return
		}
		if err := users.EnableTOTP(r.Context(), userID); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to enable MFA")
			return
		}
		if err := audits.Create(r.Context(), "user", userID, "mfa.enable", "user", userID); err != nil {
			slog.Error("failed to audit mfa enable", "error", err)
		}
		writeJSON(w, http.StatusOK, map[string]bool{"totp_enabled": true})
	}
}

func disableMFAHandler(users *db.UserRepository, audits *db.AuditRepository) http.HandlerFunc {
	type request struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		userID := getUserID(r)
		user, err := users.FindByID(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}
		if !auth.VerifyPassword(user.Password, req.Password) {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid password")
			return
		}
		if user.TOTPEnabled && (user.TOTPSecret == nil || !auth.ValidateTOTP(*user.TOTPSecret, req.Code, time.Now())) {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authentication code")
			return
		}
		if err := users.DisableTOTP(r.Context(), userID); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to disable MFA")
			return
		}
		if err := audits.Create(r.Context(), "user", userID, "mfa.disable", "user", userID); err != nil {
			slog.Error("failed to audit mfa disable", "error", err)
		}
		writeJSON(w, http.StatusOK, map[string]bool{"totp_enabled": false})
	}
}
