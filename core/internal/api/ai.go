package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type ollamaResponse struct {
	Message chatMessage `json:"message"`
	Done    bool        `json:"done"`
}

func getOllamaURL() string {
	if u := os.Getenv("OLLAMA_URL"); u != "" {
		return u
	}
	return "http://localhost:11434"
}

func getOllamaModel() string {
	if m := os.Getenv("OLLAMA_MODEL"); m != "" {
		return m
	}
	return "llama3.2"
}

func callOllama(messages []chatMessage) (*chatMessage, error) {
	body := ollamaRequest{
		Model:    getOllamaModel(),
		Messages: messages,
		Stream:   false,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(getOllamaURL()+"/api/chat", "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama error (%d): %s", resp.StatusCode, string(raw))
	}

	var result ollamaResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &result.Message, nil
}

func chatHandler(convs *db.AIConversationRepository) http.HandlerFunc {
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

		var conv *db.AIConversation
		if req.ConversationID != "" {
			var err error
			conv, err = convs.FindByID(r.Context(), req.ConversationID)
			if err != nil {
				writeError(w, http.StatusNotFound, "NOT_FOUND", "Conversation not found")
				return
			}
			if conv.UserID != claims.UserID {
				writeError(w, http.StatusForbidden, "FORBIDDEN", "not your conversation")
				return
			}
		} else {
			var err error
			title := req.Message
			if len(title) > 80 {
				title = title[:80]
			}
			conv, err = convs.Create(r.Context(), claims.UserID, title)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create conversation")
				return
			}
		}

		var messages []chatMessage
		if len(conv.Messages) > 0 {
			json.Unmarshal(conv.Messages, &messages)
		}

		messages = append(messages, chatMessage{Role: "user", Content: req.Message})

		reply, err := callOllama(messages)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "AI_ERROR", fmt.Sprintf("AI service error: %v", err))
			return
		}

		messages = append(messages, *reply)

		raw, _ := json.Marshal(messages)
		if err := convs.AppendMessage(r.Context(), conv.ID, raw); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to save messages")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"conversation_id": conv.ID,
			"reply":           reply.Content,
		})
	}
}

func listConversationsHandler(convs *db.AIConversationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)

		list, err := convs.ListByUser(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list conversations")
			return
		}

		if list == nil {
			list = []*db.AIConversation{}
		}

		type convSummary struct {
			ID        string    `json:"id"`
			Title     string    `json:"title"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
		}

		var summaries []convSummary
		for _, c := range list {
			summaries = append(summaries, convSummary{
				ID: c.ID, Title: c.Title,
				CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"conversations": summaries,
		})
	}
}

func getConversationHandler(convs *db.AIConversationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)
		convID := chi.URLParam(r, "id")

		conv, err := convs.FindByID(r.Context(), convID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Conversation not found")
			return
		}
		if conv.UserID != claims.UserID {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "not your conversation")
			return
		}

		var messages []chatMessage
		if len(conv.Messages) > 0 {
			json.Unmarshal(conv.Messages, &messages)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       conv.ID,
			"title":    conv.Title,
			"messages": messages,
		})
	}
}
