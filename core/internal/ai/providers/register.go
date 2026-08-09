package providers

import "github.com/renos-project/renos/core/internal/ai"

func init() {
	ai.RegisterProvider("ollama", NewOllamaProvider)
	ai.RegisterProvider("openai", NewOpenAIProvider)
	ai.RegisterProvider("anthropic", NewAnthropicProvider)
	ai.RegisterProvider("google", NewGoogleProvider)
	ai.RegisterProvider("lmstudio", NewOpenAICompatProvider)
	ai.RegisterProvider("vllm", NewOpenAICompatProvider)
	ai.RegisterProvider("openrouter", NewOpenAICompatProvider)
}
