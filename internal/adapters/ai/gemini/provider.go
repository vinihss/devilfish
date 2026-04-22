package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"devilfish/internal/infra/config"
	"devilfish/internal/infra/logging"
	"devilfish/internal/ports/outbound"
)

// Provider is the Google Gemini provider implementation.
type Provider struct {
	client  geminiClient
	config  config.ProviderConfig
	logger  logging.Logger
	baseURL string
}

type geminiClient interface {
	GenerateContent(ctx context.Context, req *generateContentRequest) (*generateContentResponse, error)
	StreamGenerateContent(ctx context.Context, req *generateContentRequest, onChunk func(string)) error
}

// generateContentRequest represents a request to Gemini API.
type generateContentRequest struct {
	Contents          []content         `json:"contents"`
	SystemInstruction *instruction      `json:"systemInstruction,omitempty"`
	GenerationConfig  *generationConfig `json:"generationConfig,omitempty"`
}

// content represents a content in a request.
type content struct {
	Role  string `json:"role"`
	Parts []part `json:"parts"`
}

// part represents a part of content.
type part struct {
	Text string `json:"text,omitempty"`
}

// instruction represents system instruction.
type instruction struct {
	Parts []part `json:"parts"`
}

// generationConfig represents generation configuration.
type generationConfig struct {
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   int      `json:"maxOutputTokens,omitempty"`
	Model       string   `json:"-"` // Not sent to API
}

// generateContentResponse represents a response from Gemini API.
type generateContentResponse struct {
	Candidates    []candidate    `json:"candidates"`
	UsageMetadata *usageMetadata `json:"usageMetadata,omitempty"`
}

// candidate represents a candidate in a response.
type candidate struct {
	Content      content `json:"content"`
	FinishReason string  `json:"finishReason"`
}

// usageMetadata represents token usage metadata.
type usageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// streamGenerateContentResponse represents a streaming chunk.
type streamGenerateContentResponse struct {
	Candidates []candidate `json:"candidates"`
}

const defaultModel = "gemini-2.0-flash"

// NewProvider creates a new Gemini provider.
func NewProvider(cfg config.ProviderConfig, logger logging.Logger) (*Provider, error) {
	if cfg.APIKey == "" {
		return nil, ErrMissingAPIKey
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}

	client := newGeminiClient(cfg.APIKey, baseURL)

	return &Provider{
		client:  client,
		config:  cfg,
		logger:  logger,
		baseURL: baseURL,
	}, nil
}

// newGeminiClient creates a new Gemini HTTP client.
func newGeminiClient(apiKey, baseURL string) geminiClient {
	return &httpGeminiClient{
		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

// httpGeminiClient is an HTTP client for Gemini API.
type httpGeminiClient struct {
	apiKey  string
	baseURL string
}

func (c *httpGeminiClient) GenerateContent(ctx context.Context, req *generateContentRequest) (*generateContentResponse, error) {
	model := req.GenerationConfig.Model
	if model == "" {
		model = defaultModel
	}

	// Clear model from config as it's sent in URL
	req.GenerationConfig.Model = ""

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/v1beta/models/" + model + ":generateContent?key=" + c.apiKey
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

	var result generateContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *httpGeminiClient) StreamGenerateContent(ctx context.Context, req *generateContentRequest, onChunk func(string)) error {
	model := req.GenerationConfig.Model
	if model == "" {
		model = defaultModel
	}

	req.GenerationConfig.Model = ""

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/v1beta/models/" + model + ":streamGenerateContent?alt=sse&key=" + c.apiKey
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

	decoder := json.NewDecoder(resp.Body)
	for {
		var chunk streamGenerateContentResponse
		if err := decoder.Decode(&chunk); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("failed to decode chunk: %w", err)
		}

		if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
			text := chunk.Candidates[0].Content.Parts[0].Text
			if text != "" {
				onChunk(text)
			}
		}
	}

	return nil
}

// Chat sends a chat request to Gemini and returns a response.
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

	contents := buildContents(req)

	genConfig := &generationConfig{
		Temperature: floatPtr(req.Temperature),
		MaxTokens:   req.MaxTokens,
		Model:       model,
	}

	var systemInstruction *instruction
	if req.SystemPrompt != "" {
		systemInstruction = &instruction{
			Parts: []part{{Text: req.SystemPrompt}},
		}
	}

	apiReq := &generateContentRequest{
		Contents:          contents,
		SystemInstruction: systemInstruction,
		GenerationConfig:  genConfig,
	}

	p.logger.Debug("sending chat request to Gemini")

	resp, err := p.client.GenerateContent(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, ErrEmptyResponse
	}

	candidate := resp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, ErrEmptyResponse
	}

	content := candidate.Content.Parts[0].Text

	var usage *outbound.Usage
	if resp.UsageMetadata != nil {
		usage = &outbound.Usage{
			InputTokens:  resp.UsageMetadata.PromptTokenCount,
			OutputTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:  resp.UsageMetadata.TotalTokenCount,
		}
	}

	p.logger.Debug("received chat response from Gemini")

	return &outbound.ChatResponse{
		Content:      content,
		Model:        model,
		FinishReason: candidate.FinishReason,
		Usage:        usage,
	}, nil
}

// StreamChat sends a streaming chat request to Gemini.
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

	contents := buildContents(req)

	genConfig := &generationConfig{
		Temperature: floatPtr(req.Temperature),
		MaxTokens:   req.MaxTokens,
		Model:       model,
	}

	var systemInstruction *instruction
	if req.SystemPrompt != "" {
		systemInstruction = &instruction{
			Parts: []part{{Text: req.SystemPrompt}},
		}
	}

	apiReq := &generateContentRequest{
		Contents:          contents,
		SystemInstruction: systemInstruction,
		GenerationConfig:  genConfig,
	}

	p.logger.Debug("starting streaming chat request to Gemini")

	if err := p.client.StreamGenerateContent(ctx, apiReq, onChunk); err != nil {
		return fmt.Errorf("failed to stream generate content: %w", err)
	}

	p.logger.Debug("streaming chat completed")

	return nil
}

// IsAvailable returns whether the Gemini provider is available.
func (p *Provider) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &generateContentRequest{
		Contents: []content{
			{Role: "user", Parts: []part{{Text: "ping"}}},
		},
		GenerationConfig: &generationConfig{
			MaxTokens: 1,
		},
	}

	_, err := p.client.GenerateContent(ctx, req)
	return err == nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "gemini"
}

// buildContents converts outbound.ChatRequest to Gemini contents.
func buildContents(req *outbound.ChatRequest) []content {
	var contents []content

	for _, msg := range req.Messages {
		role := msg.Role
		// Gemini uses "model" instead of "assistant"
		if role == "assistant" {
			role = "model"
		}

		contents = append(contents, content{
			Role:  role,
			Parts: []part{{Text: msg.Content}},
		})
	}

	return contents
}

func floatPtr(v float64) *float64 {
	return &v
}

// Compile-time check that Provider implements outbound.AIProvider.
var _ outbound.AIProvider = (*Provider)(nil)

// Custom errors for Gemini provider.
var (
	ErrMissingAPIKey   = fmt.Errorf("gemini provider: missing API key")
	ErrNilRequest      = fmt.Errorf("gemini provider: nil request")
	ErrNilChunkHandler = fmt.Errorf("gemini provider: nil chunk handler")
	ErrEmptyResponse   = fmt.Errorf("gemini provider: empty response")
)
