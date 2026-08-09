package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/enc"
)

func listVaultSecretsHandler(secrets *db.VaultSecretRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		var list []*db.VaultSecret
		var err error
		if query != "" {
			list, err = secrets.Search(r.Context(), query)
		} else {
			list, err = secrets.ListAll(r.Context())
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list secrets: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.VaultSecret{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func getVaultSecretHandler(secrets *db.VaultSecretRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		s, err := secrets.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "secret not found")
			return
		}
		writeJSON(w, http.StatusOK, s)
	}
}

func createVaultSecretHandler(secrets *db.VaultSecretRepository, versions *db.VaultSecretVersionRepository, audit *db.VaultAuditRepository, audits *db.AuditRepository, encryptionKey string) http.HandlerFunc {
	type request struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Path        string          `json:"path"`
		SecretType  string          `json:"secret_type"`
		Value       string          `json:"value"`
		Metadata    json.RawMessage `json:"metadata"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if req.Name == "" || req.Path == "" || req.Value == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "name, path, and value are required")
			return
		}
		if req.SecretType == "" {
			req.SecretType = "generic"
		}
		if req.Metadata == nil {
			req.Metadata = json.RawMessage(`{}`)
		}

		// Encrypt the value using AES-256-GCM
		encKey := enc.DeriveKey(encryptionKey)
		encryptedStr, err := enc.Encrypt(encKey, []byte(req.Value))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "encryption_error", "failed to encrypt secret: "+err.Error())
			return
		}
		encrypted := []byte(encryptedStr)

		uid := getUserID(r)
		s := &db.VaultSecret{
			Name:           req.Name,
			Description:    req.Description,
			Path:           req.Path,
			SecretType:     req.SecretType,
			EncryptedValue: encrypted,
			Metadata:       req.Metadata,
			CreatedBy:      &uid,
		}
		if err := secrets.Create(r.Context(), s); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create secret: "+err.Error())
			return
		}

		// Store initial version
		v := &db.VaultSecretVersion{
			SecretID:       s.ID,
			Version:        1,
			EncryptedValue: encrypted,
			Metadata:       req.Metadata,
			ChangedBy:      &uid,
		}
		versions.Create(r.Context(), v)

		audit.Log(r.Context(), &db.VaultAuditEntry{
			SecretID:  &s.ID,
			Action:    "create",
			UserID:    &uid,
			IPAddress: r.RemoteAddr,
			Details:   json.RawMessage(`{}`),
		})

		audits.Create(r.Context(), "user", uid, "vault.secret.create", "secret", s.ID)
		writeJSON(w, http.StatusCreated, s)
	}
}

func updateVaultSecretHandler(secrets *db.VaultSecretRepository, versions *db.VaultSecretVersionRepository, audit *db.VaultAuditRepository, audits *db.AuditRepository, encryptionKey string) http.HandlerFunc {
	type request struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		SecretType  string          `json:"secret_type"`
		Value       string          `json:"value"`
		Metadata    json.RawMessage `json:"metadata"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		existing, err := secrets.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "secret not found")
			return
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}

		if req.Name != "" {
			existing.Name = req.Name
		}
		if req.Description != "" {
			existing.Description = req.Description
		}
		if req.SecretType != "" {
			existing.SecretType = req.SecretType
		}
		if req.Metadata != nil {
			existing.Metadata = req.Metadata
		}

		if req.Value != "" {
			encKey := enc.DeriveKey(encryptionKey)
			encryptedStr, err := enc.Encrypt(encKey, []byte(req.Value))
			if err != nil {
				writeError(w, http.StatusInternalServerError, "encryption_error", "failed to encrypt secret: "+err.Error())
				return
			}
			existing.EncryptedValue = []byte(encryptedStr)
			existing.Version++

			uid := getUserID(r)
			v := &db.VaultSecretVersion{
				SecretID:       id,
				Version:        existing.Version,
				EncryptedValue: existing.EncryptedValue,
				Metadata:       existing.Metadata,
				ChangedBy:      &uid,
			}
			versions.Create(r.Context(), v)
		}

		if err := secrets.Update(r.Context(), existing); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to update secret: "+err.Error())
			return
		}

		uid := getUserID(r)
		audit.Log(r.Context(), &db.VaultAuditEntry{
			SecretID:  &id,
			Action:    "update",
			UserID:    &uid,
			IPAddress: r.RemoteAddr,
			Details:   json.RawMessage(`{}`),
		})

		audits.Create(r.Context(), "user", uid, "vault.secret.update", "secret", id)
		writeJSON(w, http.StatusOK, existing)
	}
}

func deleteVaultSecretHandler(secrets *db.VaultSecretRepository, audit *db.VaultAuditRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		uid := getUserID(r)
		if err := secrets.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete secret: "+err.Error())
			return
		}
		audit.Log(r.Context(), &db.VaultAuditEntry{
			SecretID:  &id,
			Action:    "delete",
			UserID:    &uid,
			IPAddress: r.RemoteAddr,
			Details:   json.RawMessage(`{}`),
		})
		audits.Create(r.Context(), "user", uid, "vault.secret.delete", "secret", id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func listVaultSecretVersionsHandler(versions *db.VaultSecretVersionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		list, err := versions.ListBySecret(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list versions: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.VaultSecretVersion{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func getVaultSecretVersionHandler(versions *db.VaultSecretVersionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		verStr := chi.URLParam(r, "version")
		var ver int
		if err := json.Unmarshal([]byte(verStr), &ver); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid version")
			return
		}
		v, err := versions.FindByVersion(r.Context(), id, ver)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "version not found")
			return
		}
		writeJSON(w, http.StatusOK, v)
	}
}

func listVaultKeysHandler(keys *db.VaultKeyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := keys.ListAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list keys: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.VaultKey{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func toggleVaultKeyHandler(keys *db.VaultKeyRepository, audits *db.AuditRepository) http.HandlerFunc {
	type request struct {
		Enabled bool `json:"enabled"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if err := keys.Enable(r.Context(), id, req.Enabled); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to toggle key: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "vault.key.toggle", "key", id)
		writeJSON(w, http.StatusOK, map[string]bool{"enabled": req.Enabled})
	}
}

func listVaultPoliciesHandler(policies *db.VaultPolicyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := policies.ListAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list policies: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.VaultPolicy{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func vaultAuditLogHandler(audit *db.VaultAuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secretID := r.URL.Query().Get("secret_id")
		var list []*db.VaultAuditEntry
		var err error
		if secretID != "" {
			list, err = audit.ListBySecret(r.Context(), secretID, 50)
		} else {
			list, err = audit.ListRecent(r.Context(), 100)
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list audit log: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.VaultAuditEntry{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}
