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

type GoogleProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewGoogleProvider(cfg ai.ProviderConfig) ai.AIProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	model := cfg.Model
	if model == "" {
		model = "gemini-2.0-flash"
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	return &GoogleProvider{
		apiKey:  cfg.APIKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: timeout},
	}
}

type googleRequest struct {
	Contents         []googleContent     `json:"contents"`
	SystemInstruction *googleContent     `json:"systemInstruction,omitempty"`
	GenerationConfig *googleGenConfig    `json:"generationConfig,omitempty"`
	Tools            []googleTool        `json:"tools,omitempty"`
}

type googleContent struct {
	Role  string         `json:"role,omitempty"`
	Parts []googlePart   `json:"parts"`
}

type googlePart struct {
	Text         string           `json:"text,omitempty"`
	FunctionCall *googleFuncCall  `json:"functionCall,omitempty"`
	FunctionResponse *googleFuncResponse `json:"functionResponse,omitempty"`
}

type googleFuncCall struct {
	Name string      `json:"name"`
	Args interface{} `json:"args"`
}

type googleFuncResponse struct {
	Name   string      `json:"name"`
	Result interface{} `json:"result"`
}

type googleGenConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type googleTool struct {
	FunctionDeclarations []googleFuncDecl `json:"functionDeclarations"`
}

type googleFuncDecl struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type googleResponse struct {
	Candidates []struct {
		Content struct {
			Parts []googlePart `json:"parts"`
			Role  string       `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

func (p *GoogleProvider) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	googleReq := p.buildRequest(req, false)
	payload, err := json.Marshal(googleReq)
	if err != nil {
		return nil, fmt.Errorf("marshal google request: %w", err)
	}
	url := fmt.Sprintf("%s/models/%s:generateContent", p.baseURL, model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", p.apiKey)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("google request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google error (%d): %s", resp.StatusCode, string(raw))
	}
	var result googleResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}
	candidate := result.Candidates[0]
	response := &ai.ChatResponse{
		Model:        model,
		FinishReason: candidate.FinishReason,
		Usage: ai.Usage{
			PromptTokens:     result.UsageMetadata.PromptTokenCount,
			CompletionTokens: result.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      result.UsageMetadata.TotalTokenCount,
		},
	}
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			response.Content += part.Text
		}
		if part.FunctionCall != nil {
			argsJSON, _ := json.Marshal(part.FunctionCall.Args)
			response.ToolCalls = append(response.ToolCalls, ai.ToolCall{
				ID:   part.FunctionCall.Name,
				Type: "function",
				Function: ai.FuncCall{
					Name:      part.FunctionCall.Name,
					Arguments: string(argsJSON),
				},
			})
		}
	}
	return response, nil
}

func (p *GoogleProvider) StreamChat(ctx context.Context, req ai.ChatRequest) (<-chan ai.ChatChunk, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	googleReq := p.buildRequest(req, true)
	payload, err := json.Marshal(googleReq)
	if err != nil {
		return nil, fmt.Errorf("marshal google request: %w", err)
	}
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", p.baseURL, model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", p.apiKey)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("google stream request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("google error (%d): %s", resp.StatusCode, string(raw))
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
			var result googleResponse
			if err := json.Unmarshal([]byte(rawLine.Data), &result); err != nil {
				continue
			}
			if len(result.Candidates) == 0 {
				continue
			}
			for _, part := range result.Candidates[0].Content.Parts {
				if part.Text != "" {
					ch <- ai.ChatChunk{Delta: part.Text}
				}
			}
			if result.Candidates[0].FinishReason != "" {
				ch <- ai.ChatChunk{
					Done: true,
					FinishReason: result.Candidates[0].FinishReason,
					Usage: &ai.Usage{
						PromptTokens:     result.UsageMetadata.PromptTokenCount,
						CompletionTokens: result.UsageMetadata.CandidatesTokenCount,
						TotalTokens:      result.UsageMetadata.TotalTokenCount,
					},
				}
				return
			}
		}
	}()
	return ch, nil
}

func (p *GoogleProvider) ProviderName() string { return "google" }

func (p *GoogleProvider) IsAvailable(ctx context.Context) bool {
	return p.apiKey != ""
}

func (p *GoogleProvider) ListModels(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/models", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-goog-api-key", p.apiKey)
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
		name := strings.TrimPrefix(m.Name, "models/")
		models = append(models, name)
	}
	return models, nil
}

func (p *GoogleProvider) buildRequest(req ai.ChatRequest, stream bool) googleRequest {
	googleReq := googleRequest{}
	if req.SystemPrompt != "" {
		googleReq.SystemInstruction = &googleContent{
			Parts: []googlePart{{Text: req.SystemPrompt}},
		}
	}
	for _, m := range req.Messages {
		role := "user"
		if m.Role == "assistant" {
			role = "model"
		}
		if m.Role == "system" || m.Role == "tool" {
			continue
		}
		part := googlePart{Text: m.Content}
		googleReq.Contents = append(googleReq.Contents, googleContent{
			Role:  role,
			Parts: []googlePart{part},
		})
	}
	if req.Temperature > 0 || req.MaxTokens > 0 {
		googleReq.GenerationConfig = &googleGenConfig{}
		if req.Temperature > 0 {
			googleReq.GenerationConfig.Temperature = &req.Temperature
		}
		if req.MaxTokens > 0 {
			googleReq.GenerationConfig.MaxOutputTokens = &req.MaxTokens
		}
	}
	if len(req.Tools) > 0 {
		var funcDecls []googleFuncDecl
		for _, t := range req.Tools {
			funcDecls = append(funcDecls, googleFuncDecl{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			})
		}
		googleReq.Tools = []googleTool{{FunctionDeclarations: funcDecls}}
	}
	return googleReq
}
