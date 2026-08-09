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

type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllamaProvider(cfg ai.ProviderConfig) ai.AIProvider {
	url := cfg.BaseURL
	if url == "" {
		url = "http://localhost:11434"
	}
	model := cfg.Model
	if model == "" {
		model = "llama3.2"
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	return &OllamaProvider{
		baseURL: strings.TrimRight(url, "/"),
		model:   model,
		client:  &http.Client{Timeout: timeout},
	}
}

type ollamaRequest struct {
	Model    string           `json:"model"`
	Messages []ollamaMessage  `json:"messages"`
	Stream   bool             `json:"stream"`
	Options  *ollamaOptions   `json:"options,omitempty"`
	Tools    []ollamaTool     `json:"tools,omitempty"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type ollamaMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []ollamaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type ollamaTool struct {
	Type     string         `json:"type"`
	Function ollamaFuncDef  `json:"function"`
}

type ollamaFuncDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type ollamaToolCall struct {
	Type     string          `json:"type"`
	Function ollamaFuncCall  `json:"function"`
}

type ollamaFuncCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ollamaResponse struct {
	Message struct {
		Content    string           `json:"content"`
		ToolCalls  []ollamaToolCall `json:"tool_calls,omitempty"`
	} `json:"message"`
	Done              bool `json:"done"`
	TotalDuration     int  `json:"total_duration"`
	PromptEvalCount   int  `json:"prompt_eval_count"`
	EvalCount         int  `json:"eval_count"`
	PromptEvalDuration int `json:"prompt_eval_duration"`
	EvalDuration      int  `json:"eval_duration"`
}

func (p *OllamaProvider) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	messages := p.buildMessages(req)
	ollamaReq := ollamaRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
	}
	if req.Temperature > 0 || req.MaxTokens > 0 {
		ollamaReq.Options = &ollamaOptions{}
		if req.Temperature > 0 {
			ollamaReq.Options.Temperature = req.Temperature
		}
		if req.MaxTokens > 0 {
			ollamaReq.Options.NumPredict = req.MaxTokens
		}
	}
	if len(req.Tools) > 0 {
		ollamaReq.Tools = convertTools(req.Tools)
	}
	payload, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("marshal ollama request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(httpReq)
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
	response := &ai.ChatResponse{
		Content:      result.Message.Content,
		Model:        model,
		FinishReason: "stop",
		Usage: ai.Usage{
			PromptTokens:     result.PromptEvalCount,
			CompletionTokens: result.EvalCount,
			TotalTokens:      result.PromptEvalCount + result.EvalCount,
		},
	}
	for _, tc := range result.Message.ToolCalls {
		response.ToolCalls = append(response.ToolCalls, ai.ToolCall{
			ID:   tc.Function.Name,
			Type: "function",
			Function: ai.FuncCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}
	return response, nil
}

func (p *OllamaProvider) StreamChat(ctx context.Context, req ai.ChatRequest) (<-chan ai.ChatChunk, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	messages := p.buildMessages(req)
	ollamaReq := ollamaRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}
	if req.Temperature > 0 || req.MaxTokens > 0 {
		ollamaReq.Options = &ollamaOptions{}
		if req.Temperature > 0 {
			ollamaReq.Options.Temperature = req.Temperature
		}
		if req.MaxTokens > 0 {
			ollamaReq.Options.NumPredict = req.MaxTokens
		}
	}
	payload, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("marshal ollama request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama stream request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("ollama error (%d): %s", resp.StatusCode, string(raw))
	}
	ch := make(chan ai.ChatChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		for {
			var chunk ollamaResponse
			if err := decoder.Decode(&chunk); err != nil {
				if err == io.EOF {
					ch <- ai.ChatChunk{Done: true, FinishReason: "stop"}
					return
				}
				ch <- ai.ChatChunk{Done: true}
				return
			}
			if chunk.Done {
				ch <- ai.ChatChunk{
					Done: true,
					Usage: &ai.Usage{
						PromptTokens:     chunk.PromptEvalCount,
						CompletionTokens: chunk.EvalCount,
						TotalTokens:      chunk.PromptEvalCount + chunk.EvalCount,
					},
					FinishReason: "stop",
				}
				return
			}
			ch <- ai.ChatChunk{Delta: chunk.Message.Content}
		}
	}()
	return ch, nil
}

func (p *OllamaProvider) ProviderName() string { return "ollama" }

func (p *OllamaProvider) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (p *OllamaProvider) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	var models []string
	for _, m := range result.Models {
		models = append(models, m.Name)
	}
	return models, nil
}

func (p *OllamaProvider) buildMessages(req ai.ChatRequest) []ollamaMessage {
	var msgs []ollamaMessage
	if req.SystemPrompt != "" {
		msgs = append(msgs, ollamaMessage{Role: "system", Content: req.SystemPrompt})
	}
	for _, m := range req.Messages {
		om := ollamaMessage{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}
		for _, tc := range m.ToolCalls {
			om.ToolCalls = append(om.ToolCalls, ollamaToolCall{
				Type: "function",
				Function: ollamaFuncCall{Name: tc.Function.Name, Arguments: tc.Function.Arguments},
			})
		}
		msgs = append(msgs, om)
	}
	return msgs
}

func convertTools(tools []ai.ToolDefinition) []ollamaTool {
	result := make([]ollamaTool, len(tools))
	for i, t := range tools {
		result[i] = ollamaTool{
			Type: "function",
			Function: ollamaFuncDef{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		}
	}
	return result
}
