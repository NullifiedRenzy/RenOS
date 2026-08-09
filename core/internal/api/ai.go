package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/ai"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/notify"
)

func chatHandler(renai *ai.RenAI, audits *db.AuditRepository, notifier *notify.Notifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)

		var req struct {
			ConversationID string `json:"conversation_id"`
			Message        string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.Message == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "message required")
			return
		}

		slog.Info("ai chat", "user_id", claims.UserID, "role", claims.Role)
		if err := audits.Create(r.Context(), "user", claims.UserID, "ai.chat", "conversation", req.ConversationID); err != nil {
			slog.Error("failed to audit ai chat", "error", err)
		}

		result, err := renai.Chat(r.Context(), claims.UserID, req.ConversationID, req.Message, claims.Role)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "AI_ERROR", err.Error())
			return
		}

		if notifier != nil && len(result.SuggestedActions) > 0 {
			notifier.Notify(r.Context(), claims.UserID, "ai", "AI suggestions ready",
				"Your AI assistant produced actionable suggestions for your request.", "conversation", result.ConversationID,
				map[string]any{"actions": len(result.SuggestedActions)})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"conversation_id":   result.ConversationID,
			"content":           result.Content,
			"suggested_actions": result.SuggestedActions,
			"usage":             result.Usage,
			"model":             result.Model,
		})
	}
}

func streamChatHandler(renai *ai.RenAI, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)

		var req struct {
			ConversationID string `json:"conversation_id"`
			Message        string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.Message == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "message required")
			return
		}

		if err := audits.Create(r.Context(), "user", claims.UserID, "ai.stream_chat", "conversation", req.ConversationID); err != nil {
			slog.Error("failed to audit ai stream chat", "error", err)
		}

		streamCh, convID, providerName, err := renai.StreamChat(r.Context(), claims.UserID, req.ConversationID, req.Message, claims.Role)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "AI_ERROR", err.Error())
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeError(w, http.StatusInternalServerError, "STREAM_ERROR", "streaming not supported")
			return
		}

		initialMsg, _ := json.Marshal(map[string]interface{}{
			"type":           "started",
			"conversation_id": convID,
			"provider":        providerName,
		})
		w.Write(append(initialMsg, '\n'))
		flusher.Flush()

		for chunk := range streamCh {
			if chunk.Done {
				doneMsg, _ := json.Marshal(map[string]interface{}{
					"type":          "done",
					"finish_reason": chunk.FinishReason,
					"usage":         chunk.Usage,
				})
				w.Write(append(doneMsg, '\n'))
				flusher.Flush()
				return
			}
			if chunk.Delta != "" {
				deltaMsg, _ := json.Marshal(map[string]interface{}{
					"type":  "delta",
					"delta": chunk.Delta,
				})
				w.Write(append(deltaMsg, '\n'))
				flusher.Flush()
			}
		}
	}
}

func listConversationsHandler(renai *ai.RenAI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		convs, err := renai.ListConversations(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list conversations")
			return
		}
		if convs == nil {
			convs = []ai.ConversationSummary{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"conversations": convs,
		})
	}
}

func getConversationHandler(renai *ai.RenAI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		convID := chi.URLParam(r, "id")
		conv, err := renai.GetConversation(r.Context(), claims.UserID, convID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":       conv.ID,
			"title":    conv.Title,
			"messages": conv.Messages,
		})
	}
}

func deleteConversationHandler(renai *ai.RenAI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		convID := chi.URLParam(r, "id")
		if err := renai.DeleteConversation(r.Context(), claims.UserID, convID); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "conversation deleted",
		})
	}
}

func searchConversationsHandler(renai *ai.RenAI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		query := r.URL.Query().Get("q")
		if query == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "query parameter q required")
			return
		}
		results, err := renai.SearchConversations(r.Context(), claims.UserID, query)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to search conversations")
			return
		}
		if results == nil {
			results = []ai.ConversationSummary{}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"conversations": results,
		})
	}
}

func aiSettingsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"providers": ai.ListProviders(),
		})
	}
}
