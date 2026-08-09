package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/renos-project/renos/core/internal/db"
)

type SuggestedAction struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	Target      string `json:"target"`
	Description string `json:"description"`
	RiskLevel   string `json:"risk_level"`
	Command     string `json:"command"`
}

type ChatResult struct {
	Content          string            `json:"content"`
	SuggestedActions []SuggestedAction `json:"suggested_actions,omitempty"`
	ConversationID   string            `json:"conversation_id"`
	Usage            Usage             `json:"usage"`
	Model            string            `json:"model"`
}

type RenAI struct {
	provider   AIProvider
	context    *ContextBuilder
	conversations *db.AIConversationRepository
	mu         sync.RWMutex
}

func NewRenAI(provider AIProvider, ctx *ContextBuilder, convs *db.AIConversationRepository) *RenAI {
	return &RenAI{
		provider:      provider,
		context:       ctx,
		conversations: convs,
	}
}

func (r *RenAI) SetProvider(provider AIProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.provider = provider
}

func (r *RenAI) GetProvider() AIProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.provider
}

func (r *RenAI) Chat(ctx context.Context, userID, conversationID, message string, userRole string) (*ChatResult, error) {
	r.mu.RLock()
	provider := r.provider
	r.mu.RUnlock()

	if provider == nil {
		return nil, fmt.Errorf("no AI provider configured")
	}

	aiCtx, err := r.context.Build(ctx)
	if err != nil {
		slog.Warn("failed to build AI context", "error", err)
		aiCtx = &AIContext{}
	}

	var conv *db.AIConversation
	if conversationID != "" {
		conv, err = r.conversations.FindByID(ctx, conversationID)
		if err != nil {
			return nil, fmt.Errorf("conversation not found: %w", err)
		}
		if conv.UserID != userID {
			return nil, fmt.Errorf("not your conversation")
		}
	} else {
		title := message
		if len(title) > 80 {
			title = title[:80]
		}
		conv, err = r.conversations.Create(ctx, userID, title)
		if err != nil {
			return nil, fmt.Errorf("create conversation: %w", err)
		}
	}

	var history []Message
	if len(conv.Messages) > 0 {
		json.Unmarshal(conv.Messages, &history)
	}

	history = append(history, Message{Role: "user", Content: message})

	systemPrompt := BuildSystemPrompt(aiCtx, userRole, nil)

	req := ChatRequest{
		Messages:     history,
		SystemPrompt: systemPrompt,
		Temperature:  0.7,
		MaxTokens:    2048,
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("AI provider error: %w", err)
	}

	history = append(history, Message{Role: "assistant", Content: resp.Content})

	raw, _ := json.Marshal(history)
	if err := r.conversations.AppendMessage(ctx, conv.ID, raw); err != nil {
		slog.Error("failed to save conversation", "error", err)
	}

	suggestedActions := parseSuggestedActions(resp.Content)

	content := cleanResponse(resp.Content)

	return &ChatResult{
		Content:          content,
		SuggestedActions: suggestedActions,
		ConversationID:   conv.ID,
		Usage:            resp.Usage,
		Model:            resp.Model,
	}, nil
}

func (r *RenAI) StreamChat(ctx context.Context, userID, conversationID, message string, userRole string) (<-chan ChatChunk, string, string, error) {
	r.mu.RLock()
	provider := r.provider
	r.mu.RUnlock()

	if provider == nil {
		return nil, "", "", fmt.Errorf("no AI provider configured")
	}

	aiCtx, err := r.context.Build(ctx)
	if err != nil {
		aiCtx = &AIContext{}
	}

	var conv *db.AIConversation
	if conversationID != "" {
		conv, err = r.conversations.FindByID(ctx, conversationID)
		if err != nil {
			return nil, "", "", fmt.Errorf("conversation not found: %w", err)
		}
		if conv.UserID != userID {
			return nil, "", "", fmt.Errorf("not your conversation")
		}
	} else {
		title := message
		if len(title) > 80 {
			title = title[:80]
		}
		conv, err = r.conversations.Create(ctx, userID, title)
		if err != nil {
			return nil, "", "", fmt.Errorf("create conversation: %w", err)
		}
	}

	var history []Message
	if len(conv.Messages) > 0 {
		json.Unmarshal(conv.Messages, &history)
	}

	history = append(history, Message{Role: "user", Content: message})

	systemPrompt := BuildSystemPrompt(aiCtx, userRole, nil)

	req := ChatRequest{
		Messages:     history,
		SystemPrompt: systemPrompt,
		Temperature:  0.7,
		MaxTokens:    2048,
		Stream:       true,
	}

	streamCh, err := provider.StreamChat(ctx, req)
	if err != nil {
		return nil, "", "", fmt.Errorf("AI provider stream error: %w", err)
	}

	adaptedCh := make(chan ChatChunk, 64)
	var fullContent strings.Builder

	go func() {
		defer close(adaptedCh)
		for chunk := range streamCh {
			fullContent.WriteString(chunk.Delta)
			adaptedCh <- chunk
		}

		finalContent := fullContent.String()
		history = append(history, Message{Role: "assistant", Content: finalContent})
		raw, _ := json.Marshal(history)
		if err := r.conversations.AppendMessage(context.Background(), conv.ID, raw); err != nil {
			slog.Error("failed to save streamed conversation", "error", err)
		}
	}()

	return adaptedCh, conv.ID, provider.ProviderName(), nil
}

func (r *RenAI) ListConversations(ctx context.Context, userID string) ([]ConversationSummary, error) {
	convs, err := r.conversations.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	var summaries []ConversationSummary
	for _, c := range convs {
		msgCount := 0
		if len(c.Messages) > 0 {
			var msgs []Message
			json.Unmarshal(c.Messages, &msgs)
			msgCount = len(msgs)
		}
		summaries = append(summaries, ConversationSummary{
			ID:         c.ID,
			Title:      c.Title,
			MessageCount: msgCount,
			CreatedAt:  c.CreatedAt,
			UpdatedAt:  c.UpdatedAt,
		})
	}
	return summaries, nil
}

func (r *RenAI) GetConversation(ctx context.Context, userID, conversationID string) (*ConversationDetail, error) {
	conv, err := r.conversations.FindByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv.UserID != userID {
		return nil, fmt.Errorf("not your conversation")
	}
	var messages []Message
	if len(conv.Messages) > 0 {
		json.Unmarshal(conv.Messages, &messages)
	}
	return &ConversationDetail{
		ID:       conv.ID,
		Title:    conv.Title,
		Messages: messages,
	}, nil
}

func (r *RenAI) DeleteConversation(ctx context.Context, userID, conversationID string) error {
	conv, err := r.conversations.FindByID(ctx, conversationID)
	if err != nil {
		return err
	}
	if conv.UserID != userID {
		return fmt.Errorf("not your conversation")
	}
	return r.conversations.Delete(ctx, conversationID)
}

func (r *RenAI) SearchConversations(ctx context.Context, userID, query string) ([]ConversationSummary, error) {
	all, err := r.conversations.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(query)
	var results []ConversationSummary
	for _, c := range all {
		titleMatch := strings.Contains(strings.ToLower(c.Title), query)
		contentMatch := false
		if len(c.Messages) > 0 {
			contentMatch = strings.Contains(strings.ToLower(string(c.Messages)), query)
		}
		if titleMatch || contentMatch {
			results = append(results, ConversationSummary{
				ID:        c.ID,
				Title:     c.Title,
				CreatedAt: c.CreatedAt,
				UpdatedAt: c.UpdatedAt,
			})
		}
	}
	return results, nil
}

type ConversationSummary struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ConversationDetail struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Messages []Message `json:"messages"`
}

func parseSuggestedActions(content string) []SuggestedAction {
	var actions []SuggestedAction
	lines := strings.Split(content, "\n")
	inSuggestion := false
	var current SuggestedAction

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "### Suggested Action") || strings.HasPrefix(trimmed, "## Suggested Action") {
			if inSuggestion && current.Action != "" {
				current.ID = fmt.Sprintf("action_%d", len(actions)+1)
				actions = append(actions, current)
				current = SuggestedAction{}
			}
			inSuggestion = true
			continue
		}
		if inSuggestion {
			if strings.HasPrefix(trimmed, "**Action:**") {
				current.Action = strings.TrimPrefix(trimmed, "**Action:**")
				current.Action = strings.TrimSpace(current.Action)
			} else if strings.HasPrefix(trimmed, "**Target:**") {
				current.Target = strings.TrimPrefix(trimmed, "**Target:**")
				current.Target = strings.TrimSpace(current.Target)
			} else if strings.HasPrefix(trimmed, "**Description:**") {
				current.Description = strings.TrimPrefix(trimmed, "**Description:**")
				current.Description = strings.TrimSpace(current.Description)
			} else if strings.HasPrefix(trimmed, "**Risk Level:**") {
				current.RiskLevel = strings.TrimPrefix(trimmed, "**Risk Level:**")
				current.RiskLevel = strings.TrimSpace(current.RiskLevel)
			} else if strings.HasPrefix(trimmed, "**Command:**") {
				cmd := strings.TrimPrefix(trimmed, "**Command:**")
				cmd = strings.TrimSpace(cmd)
				cmd = strings.Trim(cmd, "`")
				current.Command = cmd
			} else if strings.HasPrefix(trimmed, "###") || strings.HasPrefix(trimmed, "## ") {
				if current.Action != "" {
					current.ID = fmt.Sprintf("action_%d", len(actions)+1)
					actions = append(actions, current)
					current = SuggestedAction{}
				}
				inSuggestion = false
			}
		}
	}
	if inSuggestion && current.Action != "" {
		current.ID = fmt.Sprintf("action_%d", len(actions)+1)
		actions = append(actions, current)
	}
	return actions
}

func cleanResponse(content string) string {
	lines := strings.Split(content, "\n")
	var cleaned []string
	skipNext := false
	for _, line := range lines {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.TrimSpace(line) == "### Suggested Action" || strings.TrimSpace(line) == "## Suggested Action" {
			continue
		}
		cleaned = append(cleaned, line)
	}
	return strings.Join(cleaned, "\n")
}
