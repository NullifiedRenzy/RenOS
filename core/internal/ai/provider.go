package ai

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function FuncCall `json:"function"`
}

type FuncCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolDefinition struct {
	Type     string       `json:"type"`
	Function FuncDef      `json:"function"`
}

type FuncDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type ChatRequest struct {
	Messages      []Message       `json:"messages"`
	SystemPrompt  string          `json:"system_prompt,omitempty"`
	Model         string          `json:"model,omitempty"`
	Temperature   float64         `json:"temperature,omitempty"`
	MaxTokens     int             `json:"max_tokens,omitempty"`
	Tools         []ToolDefinition `json:"tools,omitempty"`
	Stream        bool            `json:"stream,omitempty"`
	StopSequences []string        `json:"stop,omitempty"`
}

type ChatResponse struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	Usage        Usage      `json:"usage"`
	Model        string     `json:"model"`
	FinishReason string     `json:"finish_reason"`
}

type ChatChunk struct {
	Delta        string     `json:"delta"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	Done         bool       `json:"done"`
	Usage        *Usage     `json:"usage,omitempty"`
	FinishReason string     `json:"finish_reason,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type AIProvider interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
	ProviderName() string
	IsAvailable(ctx context.Context) bool
	ListModels(ctx context.Context) ([]string, error)
}

type ProviderConfig struct {
	Type      string            `json:"type"`
	APIKey    string            `json:"api_key,omitempty"`
	BaseURL   string            `json:"base_url,omitempty"`
	Model     string            `json:"model,omitempty"`
	Models    []string          `json:"models,omitempty"`
	Extra     map[string]string `json:"extra,omitempty"`
	Timeout   time.Duration     `json:"timeout,omitempty"`
	MaxTokens int              `json:"max_tokens,omitempty"`
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]func(ProviderConfig) AIProvider)
)

func RegisterProvider(name string, factory func(ProviderConfig) AIProvider) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[strings.ToLower(name)] = factory
}

func NewProvider(name string, cfg ProviderConfig) (AIProvider, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	factory, ok := registry[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
	return factory(cfg), nil
}

func ListProviders() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
