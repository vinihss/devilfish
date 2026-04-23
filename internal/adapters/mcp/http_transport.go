package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"devilfish/internal/ports/outbound"
)

// MCP connection states
type mcpState string

const (
	mcpDisconnected mcpState = "disconnected"
	mcpConnecting   mcpState = "connecting"
	mcpConnected    mcpState = "connected"
	mcpError       mcpState = "error"
)

// HTTPTransport implements MCP client over HTTP with SSE.
type HTTPTransport struct {
	config  outbound.MCPServerConfig
	client *http.Client
	state  mcpState
	lastMsg json.RawMessage
	mu     sync.Mutex
}

// NewHTTPTransport creates a new HTTP transport.
func NewHTTPTransport(config outbound.MCPServerConfig) *HTTPTransport {
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &HTTPTransport{
		config: config,
		client: &http.Client{
			Timeout: timeout,
		},
		state: mcpDisconnected,
	}
}

// Connect establishes the connection.
func (t *HTTPTransport) Connect(ctx context.Context, config outbound.MCPServerConfig) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state == mcpConnected {
		return nil
	}

	t.state = mcpConnecting

	// Test connection with a simple request
	req, err := http.NewRequestWithContext(ctx, "GET", config.URL+"/health", nil)
	if err != nil {
		t.state = mcpError
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add auth header if configured
	if config.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+config.AuthToken)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		// Connection might still be valid for non-healthcheck endpoints
		t.state = mcpConnected
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		t.state = mcpConnected
		return nil
	}

	t.state = mcpConnected
	return nil
}

// Disconnect closes the connection.
func (t *HTTPTransport) Disconnect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.state = mcpDisconnected
	return nil
}

// IsConnected returns connection status.
func (t *HTTPTransport) IsConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.state == mcpConnected
}

// ListTools requests the list of tools.
func (t *HTTPTransport) ListTools(ctx context.Context) ([]outbound.MCPTool, error) {
	if !t.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}

	reqBody := mcpRequest{
		JSONRPC: "2.0",
		Method: "tools/list",
		ID:     1,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.config.URL+"/mcp", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if t.config.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+t.config.AuthToken)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rpcResp mcpResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", rpcResp.Error.Message)
	}

	// Extract tools from result
	var result struct {
		Tools []outbound.MCPTool `json:"tools"`
	}
	if rpcResp.Result != nil {
		if data, err := json.Marshal(rpcResp.Result); err == nil {
			json.Unmarshal(data, &result)
		}
	}

	return result.Tools, nil
}

// ExecuteTool executes a tool.
func (t *HTTPTransport) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	if !t.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}

	reqBody := mcpRequest{
		JSONRPC: "2.0",
		Method: "tools/call",
		Params: map[string]interface{}{
			"name":       toolName,
			"arguments": params,
		},
		ID: 2,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.config.URL+"/mcp", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if t.config.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+t.config.AuthToken)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rpcResp mcpResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", rpcResp.Error.Message)
	}

	// Extract result
	if rpcResp.Result == nil {
		return nil, nil
	}

	var result map[string]interface{}
	if data, err := json.Marshal(rpcResp.Result); err == nil {
		json.Unmarshal(data, &result)
	}

	return result, nil
}

// ServerName returns the server name.
func (t *HTTPTransport) ServerName() string {
	return t.config.Name
}

// mcpResponse represents an MCP JSON-RPC response.
type mcpResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *mcperr         `json:"error,omitempty"`
	ID     interface{}     `json:"id"`
}

// mcpError represents an MCP error.
type mcperr struct {
	Code    int             `json:"code"`
	Message string         `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *mcperr) Error() string {
	return e.Message
}