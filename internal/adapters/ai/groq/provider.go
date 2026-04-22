package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"devilfish/internal/infra/config"
	"devilfish/internal/ports/outbound"
)

// Provider is the Groq provider implementation.
// Groq uses the OpenAI-compatible API.
type Provider struct {
	client  groqClient
	config  config.ProviderConfig
	logger  *zerolog.Logger
	baseURL string
}

type groqClient interface {
	CreateChatCompletion(ctx context.Context, req chatCompletionRequest) (chatCompletionResponse, error)
	CreateChatCompletionStream(ctx context.Context, req chatCompletionRequest) (groqStream, error)
	ListModels(ctx context.Context) (listModelsResponse, error)
}

// chatCompletionRequest represents a chat completion request to Groq.
type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      *bool         `json:"stream,omitempty"`
}

// chatMessage represents a message in a chat completion request.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionResponse represents a chat completion response from Groq.
type chatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

// choice represents a choice in a chat completion response.
type choice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// usage represents token usage.
type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// listModelsResponse represents a list models response.
type listModelsResponse struct {
	// Empty for availability check
}

// groqStream represents a stream of chat completion chunks.
type groqStream interface {
	Close() error
	Recv() (chatCompletionChunk, error)
}

// chatCompletionChunk represents a chunk in a streaming response.
type chatCompletionChunk struct {
	ID      string  `json:"id"`
	Object  string  `json:"object"`
	Created int64   `json:"created"`
	Model   string  `json:"model"`
	Choices []delta `json:"choices"`
}

// delta represents the delta in a streaming choice.
type delta struct {
	Index        int          `json:"index"`
	Delta        contentDelta `json:"delta"`
	FinishReason string       `json:"finish_reason"`
}

// contentDelta represents the content delta.
type contentDelta struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NewProvider creates a new Groq provider.
func NewProvider(cfg config.ProviderConfig, logger *zerolog.Logger) (*Provider, error) {
	if cfg.APIKey == "" {
		return nil, ErrMissingAPIKey
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1"
	}

	client := newGroqClient(cfg.APIKey, baseURL)

	return &Provider{
		client:  client,
		config:  cfg,
		logger:  logger,
		baseURL: baseURL,
	}, nil
}

// newGroqClient creates a new Groq HTTP client.
func newGroqClient(apiKey, baseURL string) groqClient {
	return &httpGroqClient{
		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

// httpGroqClient is an HTTP client for Groq API.
type httpGroqClient struct {
	apiKey  string
	baseURL string
}

func (c *httpGroqClient) CreateChatCompletion(ctx context.Context, req chatCompletionRequest) (chatCompletionResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return chatCompletionResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return chatCompletionResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return chatCompletionResponse{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return chatCompletionResponse{}, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return chatCompletionResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

func (c *httpGroqClient) CreateChatCompletionStream(ctx context.Context, req chatCompletionRequest) (groqStream, error) {
	streamVal := true
	req.Stream = &streamVal

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return &groqStreamReader{
		reader:  resp.Body,
		decoder: json.NewDecoder(resp.Body),
	}, nil
}

func (c *httpGroqClient) ListModels(ctx context.Context) (listModelsResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
	if err != nil {
		return listModelsResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return listModelsResponse{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return listModelsResponse{}, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return listModelsResponse{}, nil
}

// groqStreamReader reads streaming responses from Groq.
type groqStreamReader struct {
	reader  io.Reader
	decoder *json.Decoder
}

func (s *groqStreamReader) Close() error {
	if rc, ok := s.reader.(io.ReadCloser); ok {
		return rc.Close()
	}
	return nil
}

func (s *groqStreamReader) Recv() (chatCompletionChunk, error) {
	var raw json.RawMessage
	for {
		token, err := s.decoder.Token()
		if err != nil {
			return chatCompletionChunk{}, err
		}

		delim, ok := token.(json.Delim)
		if !ok {
			continue
		}

		if delim == json.Delim('[') || delim == json.Delim(']') {
			continue
		}

		if delim == json.Delim('{') {
			if err := s.decoder.Decode(&raw); err != nil {
				return chatCompletionChunk{}, err
			}

			var chunk chatCompletionChunk
			if err := json.Unmarshal(raw, &chunk); err != nil {
				if strings.Contains(string(raw), "data: ") {
					return chatCompletionChunk{}, io.EOF
				}
				continue
			}

			return chunk, nil
		}

		break
	}

	return chatCompletionChunk{}, io.EOF
}

// Chat sends a chat request to Groq and returns a response.
func (p *Provider) Chat(ctx context.Context, req *outbound.ChatRequest) (*outbound.ChatResponse, error) {
	if req == nil {
		return nil, ErrNilRequest
	}

	model := req.Model
	if model == "" {
		model = p.config.Model
		if model == "" {
			model = "llama-3.1-70b-versatile"
		}
	}

	messages := buildMessages(req)

	p.logger.Debug().Msg("sending chat request to Groq")

	resp, err := p.client.CreateChatCompletion(
		ctx,
		chatCompletionRequest{
			Model:       model,
			Messages:    messages,
			Temperature: floatPtr(req.Temperature),
			MaxTokens:   req.MaxTokens,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, ErrEmptyResponse
	}

	choice := resp.Choices[0]
	usage := &outbound.Usage{
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		TotalTokens:  resp.Usage.TotalTokens,
	}

	p.logger.Debug().Msg("received chat response from Groq")

	return &outbound.ChatResponse{
		Content:      choice.Message.Content,
		Model:        resp.Model,
		FinishReason: choice.FinishReason,
		Usage:        usage,
	}, nil
}

// StreamChat sends a streaming chat request to Groq.
func (p *Provider) StreamChat(ctx context.Context, req *outbound.ChatRequest, onChunk func(string)) error {
	if req == nil {
		return ErrNilRequest
	}

	if onChunk == nil {
		return ErrNilChunkHandler
	}

	model := req.Model
	if model == "" {
		model = p.config.Model
		if model == "" {
			model = "llama-3.1-70b-versatile"
		}
	}

	messages := buildMessages(req)

	p.logger.Debug().Msg("starting streaming chat request to Groq")

	stream, err := p.client.CreateChatCompletionStream(
		ctx,
		chatCompletionRequest{
			Model:       model,
			Messages:    messages,
			Temperature: floatPtr(req.Temperature),
			MaxTokens:   req.MaxTokens,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create chat completion stream: %w", err)
	}
	defer stream.Close()

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to receive stream chunk: %w", err)
		}

		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				onChunk(content)
			}
		}
	}

	p.logger.Debug().Msg("streaming chat completed")

	return nil
}

// IsAvailable returns whether the Groq provider is available.
func (p *Provider) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := p.client.ListModels(ctx)
	return err == nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "groq"
}

// buildMessages converts outbound.ChatRequest to Groq messages.
func buildMessages(req *outbound.ChatRequest) []chatMessage {
	var messages []chatMessage

	if req.SystemPrompt != "" {
		messages = append(messages, chatMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	for _, msg := range req.Messages {
		messages = append(messages, chatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return messages
}

func floatPtr(v float64) *float64 {
	return &v
}

// Compile-time check that Provider implements outbound.AIProvider.
var _ outbound.AIProvider = (*Provider)(nil)

// Custom errors for Groq provider.
var (
	ErrMissingAPIKey   = fmt.Errorf("groq provider: missing API key")
	ErrNilRequest      = fmt.Errorf("groq provider: nil request")
	ErrNilChunkHandler = fmt.Errorf("groq provider: nil chunk handler")
	ErrEmptyResponse   = fmt.Errorf("groq provider: empty response")
)
