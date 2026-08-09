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

type OpenAIProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewOpenAIProvider(cfg ai.ProviderConfig) ai.AIProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	model := cfg.Model
	if model == "" {
		model = "gpt-4o"
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	return &OpenAIProvider{
		apiKey:  cfg.APIKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: timeout},
	}
}

type openAIRequest struct {
	Model       string           `json:"model"`
	Messages    []openAIMessage   `json:"messages"`
	Stream      bool             `json:"stream"`
	Temperature *float64         `json:"temperature,omitempty"`
	MaxTokens   *int             `json:"max_tokens,omitempty"`
	Tools       []openAITool     `json:"tools,omitempty"`
	Stop        []string         `json:"stop,omitempty"`
}

type openAIMessage struct {
	Role       string            `json:"role"`
	Content    string            `json:"content,omitempty"`
	ToolCalls  []openAIToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	Name       string            `json:"name,omitempty"`
}

type openAITool struct {
	Type     string          `json:"type"`
	Function openAIFuncDef   `json:"function"`
}

type openAIFuncDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type openAIToolCall struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Function openAIFuncCall    `json:"function"`
}

type openAIFuncCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content   string            `json:"content"`
			ToolCalls []openAIToolCall  `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string            `json:"content"`
			ToolCalls []openAIToolCall  `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

func (p *OpenAIProvider) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	oaiReq := openAIRequest{
		Model:    model,
		Messages: p.buildMessages(req),
		Stream:   false,
	}
	if req.Temperature > 0 {
		oaiReq.Temperature = &req.Temperature
	}
	if req.MaxTokens > 0 {
		oaiReq.MaxTokens = &req.MaxTokens
	}
	if len(req.Tools) > 0 {
		oaiReq.Tools = p.convertTools(req.Tools)
	}
	if len(req.StopSequences) > 0 {
		oaiReq.Stop = req.StopSequences
	}
	payload, err := json.Marshal(oaiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal openai request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai error (%d): %s", resp.StatusCode, string(raw))
	}
	var result openAIResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}
	choice := result.Choices[0]
	response := &ai.ChatResponse{
		Content:      choice.Message.Content,
		Model:        model,
		FinishReason: choice.FinishReason,
		Usage: ai.Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
	}
	for _, tc := range choice.Message.ToolCalls {
		response.ToolCalls = append(response.ToolCalls, ai.ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: ai.FuncCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}
	return response, nil
}

func (p *OpenAIProvider) StreamChat(ctx context.Context, req ai.ChatRequest) (<-chan ai.ChatChunk, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	oaiReq := openAIRequest{
		Model:    model,
		Messages: p.buildMessages(req),
		Stream:   true,
	}
	if req.Temperature > 0 {
		oaiReq.Temperature = &req.Temperature
	}
	if req.MaxTokens > 0 {
		oaiReq.MaxTokens = &req.MaxTokens
	}
	payload, err := json.Marshal(oaiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal openai request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai stream request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("openai error (%d): %s", resp.StatusCode, string(raw))
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
			if rawLine.Data == "[DONE]" {
				ch <- ai.ChatChunk{Done: true, FinishReason: "stop"}
				return
			}
			var chunk openAIStreamChunk
			if err := json.Unmarshal([]byte(rawLine.Data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) == 0 {
				continue
			}
			c := chunk.Choices[0]
			cnk := ai.ChatChunk{}
			if c.Delta.Content != "" {
				cnk.Delta = c.Delta.Content
			}
			if c.FinishReason != nil {
				cnk.Done = true
				cnk.FinishReason = *c.FinishReason
			}
			if chunk.Usage != nil {
				cnk.Usage = &ai.Usage{
					PromptTokens:     chunk.Usage.PromptTokens,
					CompletionTokens: chunk.Usage.CompletionTokens,
					TotalTokens:      chunk.Usage.TotalTokens,
				}
			}
			ch <- cnk
		}
	}()
	return ch, nil
}

func (p *OpenAIProvider) ProviderName() string { return "openai" }

func (p *OpenAIProvider) IsAvailable(ctx context.Context) bool {
	if p.apiKey == "" {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (p *OpenAIProvider) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	var models []string
	for _, m := range result.Data {
		models = append(models, m.ID)
	}
	return models, nil
}

func (p *OpenAIProvider) buildMessages(req ai.ChatRequest) []openAIMessage {
	var msgs []openAIMessage
	if req.SystemPrompt != "" {
		msgs = append(msgs, openAIMessage{Role: "system", Content: req.SystemPrompt})
	}
	for _, m := range req.Messages {
		om := openAIMessage{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID, Name: m.Name}
		for _, tc := range m.ToolCalls {
			om.ToolCalls = append(om.ToolCalls, openAIToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: openAIFuncCall{Name: tc.Function.Name, Arguments: tc.Function.Arguments},
			})
		}
		msgs = append(msgs, om)
	}
	return msgs
}

func (p *OpenAIProvider) convertTools(tools []ai.ToolDefinition) []openAITool {
	result := make([]openAITool, len(tools))
	for i, t := range tools {
		result[i] = openAITool{
			Type: "function",
			Function: openAIFuncDef{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		}
	}
	return result
}
