package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/renos-project/renos/core/internal/ai"
)

type AnthropicProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewAnthropicProvider(cfg ai.ProviderConfig) ai.AIProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	model := cfg.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	return &AnthropicProvider{
		apiKey:  cfg.APIKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: timeout},
	}
}

type anthropicRequest struct {
	Model       string              `json:"model"`
	Messages    []anthropicMessage  `json:"messages"`
	System      string              `json:"system,omitempty"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature *float64            `json:"temperature,omitempty"`
	Tools       []anthropicTool     `json:"tools,omitempty"`
	Stream      bool                `json:"stream"`
	StopSequences []string          `json:"stop_sequences,omitempty"`
}

type anthropicMessage struct {
	Role       string               `json:"role"`
	Content    json.RawMessage      `json:"content"`
}

type anthropicTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type anthropicResponse struct {
	ID         string `json:"id"`
	Content    []struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
		ID   string `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Input json.RawMessage `json:"input,omitempty"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicStreamEvent struct {
	Type  string          `json:"type"`
	Delta *struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta,omitempty"`
	ContentBlock *struct {
		Type  string `json:"type"`
		ID    string `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
	} `json:"content_block,omitempty"`
	Usage *struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
}

func (p *AnthropicProvider) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	messages, systemPrompt := p.buildMessages(req)
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	anthReq := anthropicRequest{
		Model:       model,
		Messages:    messages,
		System:      systemPrompt,
		MaxTokens:   maxTokens,
		Stream:      false,
	}
	if req.Temperature > 0 {
		anthReq.Temperature = &req.Temperature
	}
	if len(req.Tools) > 0 {
		anthReq.Tools = p.convertTools(req.Tools)
	}
	if len(req.StopSequences) > 0 {
		anthReq.StopSequences = req.StopSequences
	}
	payload, err := json.Marshal(anthReq)
	if err != nil {
		return nil, fmt.Errorf("marshal anthropic request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic error (%d): %s", resp.StatusCode, string(raw))
	}
	var result anthropicResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	response := &ai.ChatResponse{
		Model:        model,
		FinishReason: result.StopReason,
		Usage: ai.Usage{
			PromptTokens:     result.Usage.InputTokens,
			CompletionTokens: result.Usage.OutputTokens,
			TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
		},
	}
	for _, block := range result.Content {
		switch block.Type {
		case "text":
			response.Content += block.Text
		case "tool_use":
			inputJSON, _ := json.Marshal(block.Input)
			response.ToolCalls = append(response.ToolCalls, ai.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: ai.FuncCall{
					Name:      block.Name,
					Arguments: string(inputJSON),
				},
			})
		}
	}
	return response, nil
}

func (p *AnthropicProvider) StreamChat(ctx context.Context, req ai.ChatRequest) (<-chan ai.ChatChunk, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	messages, systemPrompt := p.buildMessages(req)
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	anthReq := anthropicRequest{
		Model:       model,
		Messages:    messages,
		System:      systemPrompt,
		MaxTokens:   maxTokens,
		Stream:      true,
	}
	if req.Temperature > 0 {
		anthReq.Temperature = &req.Temperature
	}
	payload, err := json.Marshal(anthReq)
	if err != nil {
		return nil, fmt.Errorf("marshal anthropic request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic stream request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("anthropic error (%d): %s", resp.StatusCode, string(raw))
	}
	ch := make(chan ai.ChatChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		for {
			var rawLine struct {
				Data string `json:"data"`
			}
			if err := decoder.Decode(&rawLine); err != nil {
				if err == io.EOF {
					ch <- ai.ChatChunk{Done: true, FinishReason: "stop"}
					return
				}
				ch <- ai.ChatChunk{Done: true}
				return
			}
			if rawLine.Data == "" {
				continue
			}
			var event anthropicStreamEvent
			if err := json.Unmarshal([]byte(rawLine.Data), &event); err != nil {
				continue
			}
			switch event.Type {
			case "content_block_delta":
				if event.Delta != nil && event.Delta.Type == "text_delta" {
					ch <- ai.ChatChunk{Delta: event.Delta.Text}
				}
			case "message_start":
			case "message_delta":
				ch <- ai.ChatChunk{Done: true, FinishReason: "stop"}
				return
			case "message_stop":
				ch <- ai.ChatChunk{Done: true, FinishReason: "stop"}
				return
			}
		}
	}()
	return ch, nil
}

func (p *AnthropicProvider) ProviderName() string { return "anthropic" }

func (p *AnthropicProvider) IsAvailable(ctx context.Context) bool {
	return p.apiKey != ""
}

func (p *AnthropicProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{
		"claude-sonnet-4-20250514",
		"claude-3-5-haiku-20241022",
		"claude-3-5-sonnet-20241022",
		"claude-3-opus-20240229",
		"claude-3-haiku-20240307",
	}, nil
}

func (p *AnthropicProvider) buildMessages(req ai.ChatRequest) ([]anthropicMessage, string) {
	var msgs []anthropicMessage
	for _, m := range req.Messages {
		if m.Role == "system" {
			continue
		}
		if m.Role == "tool" {
			msgs = append(msgs, anthropicMessage{
				Role:    "user",
				Content: json.RawMessage(fmt.Sprintf(`[{"type":"tool_result","tool_use_id":"%s","content":"%s"}]`, m.ToolCallID, escapeJSON(m.Content))),
			})
			continue
		}
		content := fmt.Sprintf(`[{"type":"text","text":"%s"}]`, escapeJSON(m.Content))
		if len(m.ToolCalls) > 0 {
			blocks := []interface{}{}
			if m.Content != "" {
				blocks = append(blocks, map[string]string{"type": "text", "text": m.Content})
			}
			for _, tc := range m.ToolCalls {
				var input map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &input)
				blocks = append(blocks, map[string]interface{}{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Function.Name,
					"input": input,
				})
			}
			raw, _ := json.Marshal(blocks)
			content = string(raw)
		}
		msgs = append(msgs, anthropicMessage{Role: m.Role, Content: json.RawMessage(content)})
	}
	return msgs, req.SystemPrompt
}

func (p *AnthropicProvider) convertTools(tools []ai.ToolDefinition) []anthropicTool {
	result := make([]anthropicTool, len(tools))
	for i, t := range tools {
		result[i] = anthropicTool{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			InputSchema: t.Function.Parameters,
		}
	}
	return result
}

func escapeJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b[1 : len(b)-1])
}
