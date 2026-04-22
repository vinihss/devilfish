package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"devilfish/internal/infra/config"
	"devilfish/internal/infra/logging"
	"devilfish/internal/ports/outbound"
)

// Provider is the Ollama provider implementation.
// Ollama is a local AI runtime that runs models locally.
type Provider struct {
	client  ollamaClient
	config  config.ProviderConfig
	logger  logging.Logger
	baseURL string
}

type ollamaClient interface {
	Generate(ctx context.Context, req *generateRequest) (*generateResponse, error)
	GenerateStream(ctx context.Context, req *generateRequest, onChunk func(string)) error
	ListModels(ctx context.Context) (*listModelsResponse, error)
}

// generateRequest represents a request to Ollama API.
type generateRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	System      string   `json:"system,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Stream      bool     `json:"stream,omitempty"`
	Options     options  `json:"options,omitempty"`
}

// options represents additional options for Ollama.
type options struct {
	Temperature *float64 `json:"temperature,omitempty"`
	NumCtx      int      `json:"num_ctx,omitempty"`
	NumPredict  int      `json:"num_predict,omitempty"`
}

// generateResponse represents a response from Ollama API.
type generateResponse struct {
	Model           string       `json:"model"`
	Response        string       `json:"response"`
	Done            bool         `json:"done"`
	Context         *contextInfo `json:"context,omitempty"`
	TotalDuration   int64        `json:"total_duration,omitempty"`
	LoadDuration    int64        `json:"load_duration,omitempty"`
	PromptEvalCount int          `json:"prompt_eval_count,omitempty"`
	EvalCount       int          `json:"eval_count,omitempty"`
}

// contextInfo represents context information.
type contextInfo struct {
	// Fields for context (not fully specified)
}

// listModelsResponse represents a list models response from Ollama.
type listModelsResponse struct {
	Models []modelInfo `json:"models"`
}

// modelInfo represents information about a model.
type modelInfo struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Modified string `json:"modified_at"`
}

const defaultModel = "llama3.2"

// NewProvider creates a new Ollama provider.
func NewProvider(cfg config.ProviderConfig, logger logging.Logger) (*Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	client := newOllamaClient(baseURL)

	return &Provider{
		client:  client,
		config:  cfg,
		logger:  logger,
		baseURL: baseURL,
	}, nil
}

// newOllamaClient creates a new Ollama HTTP client.
func newOllamaClient(baseURL string) ollamaClient {
	return &httpOllamaClient{
		baseURL: baseURL,
	}
}

// httpOllamaClient is an HTTP client for Ollama API.
type httpOllamaClient struct {
	baseURL string
}

func (c *httpOllamaClient) Generate(ctx context.Context, req *generateRequest) (*generateResponse, error) {
	req.Stream = false

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/api/generate"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *httpOllamaClient) GenerateStream(ctx context.Context, req *generateRequest, onChunk func(string)) error {
	req.Stream = true

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/api/generate"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Skip "[DONE]" message
		if strings.TrimSpace(line) == "done" {
			continue
		}

		// Parse as JSON
		var chunk generateResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}

		if chunk.Response != "" {
			onChunk(chunk.Response)
		}

		if chunk.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read stream: %w", err)
	}

	return nil
}

func (c *httpOllamaClient) ListModels(ctx context.Context) (*listModelsResponse, error) {
	url := c.baseURL + "/api/tags"
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result listModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// Chat sends a chat request to Ollama and returns a response.
func (p *Provider) Chat(ctx context.Context, req *outbound.ChatRequest) (*outbound.ChatResponse, error) {
	if req == nil {
		return nil, ErrNilRequest
	}

	model := req.Model
	if model == "" {
		model = p.config.Model
		if model == "" {
			model = defaultModel
		}
	}

	// Convert messages to a single prompt for Ollama
	prompt, systemPrompt := buildPrompt(req)

	apiReq := &generateRequest{
		Model:       model,
		Prompt:      prompt,
		System:      systemPrompt,
		Temperature: floatPtr(req.Temperature),
		MaxTokens:   req.MaxTokens,
	}

	p.logger.Debug("sending chat request to Ollama")

	resp, err := p.client.Generate(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate: %w", err)
	}

	usage := &outbound.Usage{
		InputTokens:  resp.PromptEvalCount,
		OutputTokens: resp.EvalCount,
		TotalTokens:  resp.PromptEvalCount + resp.EvalCount,
	}

	finishReason := "stop"
	if resp.Done == false {
		finishReason = "length"
	}

	p.logger.Debug("received chat response from Ollama")

	return &outbound.ChatResponse{
		Content:      resp.Response,
		Model:        resp.Model,
		FinishReason: finishReason,
		Usage:        usage,
	}, nil
}

// StreamChat sends a streaming chat request to Ollama.
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
			model = defaultModel
		}
	}

	prompt, systemPrompt := buildPrompt(req)

	apiReq := &generateRequest{
		Model:       model,
		Prompt:      prompt,
		System:      systemPrompt,
		Temperature: floatPtr(req.Temperature),
		MaxTokens:   req.MaxTokens,
	}

	p.logger.Debug("starting streaming chat request to Ollama")

	if err := p.client.GenerateStream(ctx, apiReq, onChunk); err != nil {
		return fmt.Errorf("failed to stream generate: %w", err)
	}

	p.logger.Debug("streaming chat completed")

	return nil
}

// IsAvailable returns whether the Ollama provider is available.
func (p *Provider) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := p.client.ListModels(ctx)
	return err == nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "ollama"
}

// buildPrompt converts outbound.ChatRequest to a single prompt for Ollama.
func buildPrompt(req *outbound.ChatRequest) (prompt, systemPrompt string) {
	var sb strings.Builder

	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			sb.WriteString("System: ")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		case "user":
			sb.WriteString("User: ")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		case "assistant":
			sb.WriteString("Assistant: ")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		}
	}

	prompt = sb.String()
	systemPrompt = req.SystemPrompt

	return prompt, systemPrompt
}

func floatPtr(v float64) *float64 {
	return &v
}

// Compile-time check that Provider implements outbound.AIProvider.
var _ outbound.AIProvider = (*Provider)(nil)

// Custom errors for Ollama provider.
var (
	ErrNilRequest      = fmt.Errorf("ollama provider: nil request")
	ErrNilChunkHandler = fmt.Errorf("ollama provider: nil chunk handler")
)
