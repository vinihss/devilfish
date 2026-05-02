package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"devilfish/internal/ports/outbound"
)

const (
	defaultTimeout    = 10 * time.Second
	toolName          = "web_search"
	toolDescription   = "Performs a web search and returns structured results."
	serverName        = "websearch"
)

// WebSearchMCP is a concrete MCP adapter that performs web searches via an
// external HTTP search API. It implements both outbound.MCPClient (so it can
// be registered in the Registry) and outbound.SearchCapability (so callers can
// use it directly through the typed interface).
type WebSearchMCP struct {
	Endpoint string
	APIKey   string
	Client   *http.Client

	mu        sync.Mutex
	connected bool
}

// NewWebSearchMCP creates a WebSearchMCP with the given endpoint and API key.
// A nil Client is replaced with a default client that uses defaultTimeout.
func NewWebSearchMCP(endpoint, apiKey string, client *http.Client) *WebSearchMCP {
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	return &WebSearchMCP{
		Endpoint: endpoint,
		APIKey:   apiKey,
		Client:   client,
	}
}

// --- outbound.MCPClient implementation ---

// Connect marks the provider as connected. The WebSearch provider has no
// persistent connection; this simply validates that the endpoint is set.
func (w *WebSearchMCP) Connect(_ context.Context, config outbound.MCPServerConfig) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if config.URL != "" {
		w.Endpoint = config.URL
	}
	if config.AuthToken != "" {
		w.APIKey = config.AuthToken
	}
	if w.Endpoint == "" {
		return ErrMissingEndpoint
	}

	w.connected = true
	return nil
}

// Disconnect marks the provider as disconnected.
func (w *WebSearchMCP) Disconnect(_ context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.connected = false
	return nil
}

// IsConnected returns whether the provider is ready to serve requests.
func (w *WebSearchMCP) IsConnected() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.connected
}

// ListTools returns the single "web_search" tool that this provider exposes.
func (w *WebSearchMCP) ListTools(_ context.Context) ([]outbound.MCPTool, error) {
	return []outbound.MCPTool{
		{
			Name:        toolName,
			Description: toolDescription,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results to return",
					},
				},
				"required": []string{"query"},
			},
		},
	}, nil
}

// ExecuteTool dispatches "web_search" calls to the underlying Search method and
// serializes the results into the generic map format expected by MCPClient callers.
func (w *WebSearchMCP) ExecuteTool(ctx context.Context, name string, params map[string]interface{}) (map[string]interface{}, error) {
	if name != toolName {
		return nil, fmt.Errorf("websearch: unknown tool %q", name)
	}

	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, ErrMissingQuery
	}

	limit := 10
	if v, ok := params["limit"]; ok {
		switch lv := v.(type) {
		case int:
			limit = lv
		case float64:
			limit = int(lv)
		case string:
			if n, err := strconv.Atoi(lv); err == nil {
				limit = n
			}
		}
	}

	// Respect the caller's context for the HTTP request.
	results, err := w.searchWithContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{"results": results}, nil
}

// ServerName returns the provider's fixed name.
func (w *WebSearchMCP) ServerName() string {
	return serverName
}

// --- outbound.SearchCapability implementation ---

// Search performs a web search and returns up to limit results.
// It uses a background context with the configured client timeout, falling back
// to defaultTimeout when the client has no explicit timeout set.
func (w *WebSearchMCP) Search(query string, limit int) ([]outbound.SearchResult, error) {
	timeout := w.Client.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return w.searchWithContext(ctx, query, limit)
}

// --- internal helpers ---

// searchRequest represents the JSON body sent to the external search API.
type searchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// externalResult is one item in the external API's response array.
type externalResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// externalResponse is the top-level JSON object returned by the external API.
type externalResponse struct {
	Results []externalResult `json:"results"`
}

func (w *WebSearchMCP) searchWithContext(ctx context.Context, query string, limit int) ([]outbound.SearchResult, error) {
	if query == "" {
		return nil, ErrMissingQuery
	}
	if limit <= 0 {
		return nil, ErrInvalidLimit
	}

	reqURL, err := buildURL(w.Endpoint, query, limit)
	if err != nil {
		return nil, fmt.Errorf("websearch: failed to build request URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("websearch: failed to create request: %w", err)
	}

	if w.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+w.APIKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := w.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("websearch: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) // best-effort; error is intentionally ignored in the error path
		return nil, fmt.Errorf("websearch: API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp externalResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("websearch: failed to decode response: %w", err)
	}

	if len(apiResp.Results) == 0 {
		return []outbound.SearchResult{}, nil
	}

	results := make([]outbound.SearchResult, 0, len(apiResp.Results))
	for _, r := range apiResp.Results {
		results = append(results, outbound.SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Snippet,
		})
	}

	return results, nil
}

// buildURL constructs the request URL with query parameters.
func buildURL(endpoint, query string, limit int) (string, error) {
	base, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint URL: %w", err)
	}

	q := base.Query()
	q.Set("q", query)
	q.Set("limit", strconv.Itoa(limit))
	base.RawQuery = q.Encode()

	return base.String(), nil
}

// Compile-time assertions.
var _ outbound.MCPClient = (*WebSearchMCP)(nil)
var _ outbound.SearchCapability = (*WebSearchMCP)(nil)

// Sentinel errors.
var (
	ErrMissingEndpoint = fmt.Errorf("websearch: endpoint must not be empty")
	ErrMissingQuery    = fmt.Errorf("websearch: query must not be empty")
	ErrInvalidLimit    = fmt.Errorf("websearch: limit must be greater than zero")
)
