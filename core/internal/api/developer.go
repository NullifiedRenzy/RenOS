package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/renos-project/renos/core/internal/developer"
)

func listWebhookSubscriptionsHandler(subs *developer.WebhookSubscriptionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		list, err := subs.ListByUser(r.Context(), userID)
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", err.Error()); return }
		if list == nil { list = []*developer.WebhookSubscription{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func createWebhookSubscriptionHandler(subs *developer.WebhookSubscriptionRepository) http.HandlerFunc {
	type request struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid body")
			return
		}
		if req.URL == "" || len(req.Events) == 0 {
			writeError(w, http.StatusBadRequest, "invalid_request", "url and events required")
			return
		}

		secret := generateWebhookSecret()
		ws := &developer.WebhookSubscription{
			UserID:  userID,
			URL:     req.URL,
			Events:  req.Events,
			Secret:  secret,
			Active:  true,
			Headers: map[string]string{},
			RetryPolicy: 3,
		}
		if err := subs.Create(r.Context(), ws); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]interface{}{"subscription": ws, "secret": secret})
	}
}

func deleteWebhookSubscriptionHandler(subs *developer.WebhookSubscriptionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" { writeError(w, http.StatusBadRequest, "invalid_request", "id required"); return }
		if err := subs.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func toggleWebhookSubscriptionHandler(subs *developer.WebhookSubscriptionRepository) http.HandlerFunc {
	type request struct {
		Active bool `json:"active"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		var req request
		json.NewDecoder(r.Body).Decode(&req)
		if id == "" { writeError(w, http.StatusBadRequest, "invalid_request", "id required"); return }
		if err := subs.Toggle(r.Context(), id, req.Active); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"active": req.Active})
	}
}

func webhookEventsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		events := developer.GetAvailableEvents()
		writeJSON(w, http.StatusOK, events)
	}
}

func webhookDeliveriesHandler(dlvs *developer.WebhookDeliveryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subID := r.URL.Query().Get("subscription_id")
		if subID == "" { writeError(w, http.StatusBadRequest, "invalid_request", "subscription_id required"); return }
		list, err := dlvs.ListBySubscription(r.Context(), subID, 50)
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", err.Error()); return }
		if list == nil { list = []*developer.WebhookDelivery{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func listAPIKeysHandler(keys *developer.APIKeyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		list, err := keys.ListByUser(r.Context(), userID)
		if err != nil { writeError(w, http.StatusInternalServerError, "internal_error", err.Error()); return }
		if list == nil { list = []*developer.APIKey{} }
		writeJSON(w, http.StatusOK, list)
	}
}

func createAPIKeyHandler(keys *developer.APIKeyRepository) http.HandlerFunc {
	type request struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid body")
			return
		}
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "name required")
			return
		}

		rawKey := generateAPIKey()
		keyHash := hashAPIKey(rawKey)
		prefix := rawKey[:8]

		apiKey := &developer.APIKey{
			UserID:    userID,
			Name:      req.Name,
			KeyHash:   keyHash,
			KeyPrefix: prefix,
			Scopes:    req.Scopes,
			Active:    true,
		}
		if err := keys.Create(r.Context(), apiKey); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]interface{}{"api_key": apiKey, "raw_key": rawKey})
	}
}

func deleteAPIKeyHandler(keys *developer.APIKeyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" { writeError(w, http.StatusBadRequest, "invalid_request", "id required"); return }
		if err := keys.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func toggleAPIKeyHandler(keys *developer.APIKeyRepository) http.HandlerFunc {
	type request struct {
		Active bool `json:"active"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		var req request
		json.NewDecoder(r.Body).Decode(&req)
		if id == "" { writeError(w, http.StatusBadRequest, "invalid_request", "id required"); return }
		if err := keys.Toggle(r.Context(), id, req.Active); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"active": req.Active})
	}
}

func generateWebhookSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "whsec_" + hex.EncodeToString(b)
}

func generateAPIKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "renos_" + hex.EncodeToString(b)
}

func hashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
