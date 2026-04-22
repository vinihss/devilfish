package outbound

import (
	"context"
)

// AIProvider is the interface for AI service providers (OpenAI, Groq, Gemini, Ollama, etc.).
type AIProvider interface {
	// Chat sends a chat request to the AI provider and returns a response.
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// StreamChat sends a streaming chat request to the AI provider.
	// The onChunk callback is called for each chunk of the response.
	StreamChat(ctx context.Context, req *ChatRequest, onChunk func(string)) error

	// IsAvailable returns whether the AI provider is available.
	IsAvailable() bool

	// Name returns the name of the AI provider.
	Name() string
}

// ChatRequest represents a request to the AI provider.
type ChatRequest struct {
	Model        string
	Messages     []ChatMessage
	SystemPrompt string
	Temperature  float64
	MaxTokens    int
}

// ChatMessage represents a message in a chat exchange.
type ChatMessage struct {
	Role    string // "system", "user", "assistant"
	Content string
}

// ChatResponse represents a response from the AI provider.
type ChatResponse struct {
	Content      string
	Model        string
	FinishReason string
	Usage        *Usage
}

// Usage represents token usage statistics.
type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}
