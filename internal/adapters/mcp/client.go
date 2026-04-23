package mcp

import (
	"context"
	"fmt"
	"sync"

	"devilfish/internal/ports/outbound"
	"devilfish/internal/domain/valueobject"
)

// Client is the main MCP client adapter.
type Client struct {
	config  outbound.MCPServerConfig
	conn    *valueobject.MCPConnection
	transport outbound.MCPClient
	mu      sync.Mutex
}

// NewClient creates a new MCP client.
func NewClient(config outbound.MCPServerConfig) *Client {
	return &Client{
		config: config,
		conn:   valueobject.NewMCPConnection(config.Name),
	}
}

// Connect establishes a connection to the MCP server.
func (c *Client) Connect(ctx context.Context, config outbound.MCPServerConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn.IsConnected() {
		return nil
	}

	c.config = config

	// Choose transport based on config
	var transport outbound.MCPClient
	switch config.Transport {
	case "http":
		transport = NewHTTPTransport(config)
	case "stdio":
		transport = NewStdIOTransport(config)
	default:
		transport = NewStdIOTransport(config)
	}

	// Connect
	if err := transport.Connect(ctx, config); err != nil {
		c.conn.SetError(err)
		return fmt.Errorf("failed to connect to MCP server %s: %w", config.Name, err)
	}

	c.transport = transport
	c.conn.Connect()

	return nil
}

// Disconnect closes the connection.
func (c *Client) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.transport == nil {
		return nil
	}

	err := c.transport.Disconnect(ctx)
	c.conn.Disconnect()
	c.transport = nil

	return err
}

// IsConnected returns whether the client is connected.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.conn.IsConnected() && c.transport != nil && c.transport.IsConnected()
}

// ListTools returns the list of available tools.
func (c *Client) ListTools(ctx context.Context) ([]outbound.MCPTool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.conn.IsConnected() || c.transport == nil {
		return nil, fmt.Errorf("not connected to MCP server %s", c.config.Name)
	}

	c.conn.Touch()
	tools, err := c.transport.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	return tools, nil
}

// ExecuteTool executes a tool on the MCP server.
func (c *Client) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.conn.IsConnected() || c.transport == nil {
		return nil, fmt.Errorf("not connected to MCP server %s", c.config.Name)
	}

	c.conn.Touch()
	result, err := c.transport.ExecuteTool(ctx, toolName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute tool %s: %w", toolName, err)
	}

	return result, nil
}

// ServerName returns the server name.
func (c *Client) ServerName() string {
	return c.config.Name
}

// Ensure Client implements MCPClient interface
var _ outbound.MCPClient = (*Client)(nil)