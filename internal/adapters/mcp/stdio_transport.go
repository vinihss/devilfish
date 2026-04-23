package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"devilfish/internal/ports/outbound"
)

// StdIOTransport implements MCP client over stdio.
type StdIOTransport struct {
	config outbound.MCPServerConfig
	conn   *stdioConnection
	mu     sync.Mutex
	state  mcpState
}

type stdioConnection struct {
	cmd    *exec.Cmd
	stdin  *os.File
	stdout *os.File
}

// NewStdIOTransport creates a new stdio transport.
func NewStdIOTransport(config outbound.MCPServerConfig) *StdIOTransport {
	return &StdIOTransport{
		config: config,
		state:  mcpDisconnected,
	}
}

// Connect establishes the connection.
func (t *StdIOTransport) Connect(ctx context.Context, config outbound.MCPServerConfig) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state == mcpConnected {
		return nil
	}

	t.state = mcpConnecting

	// Build command
	cmd := exec.Command(config.Command, config.Args...)
	if config.Env != nil {
		for k, v := range config.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.state = mcpError
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.state = mcpError
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		t.state = mcpError
		return fmt.Errorf("failed to start process: %w", err)
	}

	t.conn = &stdioConnection{
		cmd:    cmd,
		stdin:  stdin.(*os.File),
		stdout: stdout.(*os.File),
	}
	t.state = mcpConnected

	return nil
}

// Disconnect closes the connection.
func (t *StdIOTransport) Disconnect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn == nil {
		return nil
	}

	if t.conn.cmd != nil && t.conn.cmd.Process != nil {
		t.conn.cmd.Process.Kill()
		t.conn.cmd.Wait()
	}

	t.conn = nil
	t.state = mcpDisconnected

	return nil
}

// IsConnected returns connection status.
func (t *StdIOTransport) IsConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.state == mcpConnected && t.conn != nil && t.conn.cmd != nil && t.conn.cmd.Process != nil
}

// ListTools requests the list of tools.
func (t *StdIOTransport) ListTools(ctx context.Context) ([]outbound.MCPTool, error) {
	if !t.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}

	req := mcpRequest{
		JSONRPC: "2.0",
		Method: "tools/list",
		ID:     1,
	}

	resp, err := t.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// Parse tools from response
	var result struct {
		Tools []outbound.MCPTool `json:"tools"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tools: %w", err)
	}

	return result.Tools, nil
}

// ExecuteTool executes a tool.
func (t *StdIOTransport) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	if !t.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}

	req := mcpRequest{
		JSONRPC: "2.0",
		Method: "tools/call",
		Params: map[string]interface{}{
			"name":       toolName,
			"arguments": params,
		},
		ID: 2,
	}

	resp, err := t.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// ServerName returns the server name.
func (t *StdIOTransport) ServerName() string {
	return t.config.Name
}

// sendRequest sends an MCP request over stdio.
func (t *StdIOTransport) sendRequest(ctx context.Context, req mcpRequest) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn == nil || t.conn.stdin == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Encode request
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Write to stdin
	_, err = t.conn.stdin.Write(data)
	t.conn.stdin.Write([]byte("\n"))
	if err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Set timeout
	timeout := time.Duration(t.config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	done := make(chan error, 1)
	go func() {
		var buf [4098]byte
		_, err := t.conn.stdout.Read(buf[:])
		if err != nil {
			done <- err
			return
		}
		done <- nil
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout")
	case err := <-done:
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// mcpRequest represents an MCP JSON-RPC request.
type mcpRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
	ID     interface{} `json:"id"`
}