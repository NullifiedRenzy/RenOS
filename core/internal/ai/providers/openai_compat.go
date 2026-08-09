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

type OpenAICompatProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
	name    string
}

func NewOpenAICompatProvider(cfg ai.ProviderConfig) ai.AIProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:1234/v1"
	}
	model := cfg.Model
	if model == "" {
		model = "default"
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	name := cfg.Extra["provider_name"]
	if name == "" {
		name = "openai-compat"
	}
	return &OpenAICompatProvider{
		apiKey:  cfg.APIKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: timeout},
		name:    name,
	}
}

type compatRequest struct {
	Model       string           `json:"model"`
	Messages    []openAIMessage   `json:"messages"`
	Stream      bool             `json:"stream"`
	Temperature *float64         `json:"temperature,omitempty"`
	MaxTokens   *int             `json:"max_tokens,omitempty"`
	Tools       []openAITool     `json:"tools,omitempty"`
	Stop        []string         `json:"stop,omitempty"`
}

func (p *OpenAICompatProvider) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	compatReq := compatRequest{
		Model:    model,
		Messages: p.buildMessages(req),
		Stream:   false,
	}
	if req.Temperature > 0 {
		compatReq.Temperature = &req.Temperature
	}
	if req.MaxTokens > 0 {
		compatReq.MaxTokens = &req.MaxTokens
	}
	if len(req.Tools) > 0 {
		compatReq.Tools = p.convertTools(req.Tools)
	}
	if len(req.StopSequences) > 0 {
		compatReq.Stop = req.StopSequences
	}
	payload, err := json.Marshal(compatReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error (%d): %s", resp.StatusCode, string(raw))
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

func (p *OpenAICompatProvider) StreamChat(ctx context.Context, req ai.ChatRequest) (<-chan ai.ChatChunk, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	compatReq := compatRequest{
		Model:    model,
		Messages: p.buildMessages(req),
		Stream:   true,
	}
	if req.Temperature > 0 {
		compatReq.Temperature = &req.Temperature
	}
	if req.MaxTokens > 0 {
		compatReq.MaxTokens = &req.MaxTokens
	}
	payload, err := json.Marshal(compatReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stream request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("error (%d): %s", resp.StatusCode, string(raw))
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

func (p *OpenAICompatProvider) ProviderName() string { return p.name }

func (p *OpenAICompatProvider) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return false
	}
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (p *OpenAICompatProvider) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
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

func (p *OpenAICompatProvider) buildMessages(req ai.ChatRequest) []openAIMessage {
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

func (p *OpenAICompatProvider) convertTools(tools []ai.ToolDefinition) []openAITool {
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
