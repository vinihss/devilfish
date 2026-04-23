package outbound

import (
	"context"
)

// MCPServerConfig represents configuration for an MCP server.
type MCPServerConfig struct {
	Name        string            `yaml:"name"`
	URL         string            `yaml:"url"`          // HTTP URL or "stdio" for local
	AuthToken   string            `yaml:"auth_token"`  // Authentication token
	Transport   string            `yaml:"transport"`   // "stdio" or "http"
	Command     string            `yaml:"command"`      // Command for stdio transport
	Args       []string          `yaml:"args"`        // Arguments for stdio
	Env        map[string]string `yaml:"env"`         // Environment variables
	Timeout    int              `yaml:"timeout"`     // Connection timeout in seconds
	MaxRetries int              `yaml:"max_retries"` // Max reconnection attempts
}

// MCPTool represents a tool exposed by an MCP server.
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	OutputSchema map[string]interface{} `json:"outputSchema"`
}

// MCPClient is the interface for MCP server connections.
type MCPClient interface {
	// Connect establishes a connection to the MCP server.
	Connect(ctx context.Context, config MCPServerConfig) error

	// Disconnect closes the connection to the MCP server.
	Disconnect(ctx context.Context) error

	// IsConnected returns whether the client is connected.
	IsConnected() bool

	// ListTools returns the list of available tools.
	ListTools(ctx context.Context) ([]MCPTool, error)

	// ExecuteTool executes a tool on the MCP server.
	// Parameters should be validated against the tool's input schema.
	ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error)

	// ServerName returns the server name.
	ServerName() string
}

// MCPConnectionState represents the state of an MCP connection.
type MCPConnectionState string

const (
	// MCPStateDisconnected means no connection.
	MCPStateDisconnected MCPConnectionState = "disconnected"
	// MCPStateConnecting means connection in progress.
	MCPStateConnecting MCPConnectionState = "connecting"
	// MCPStateConnected means successfully connected.
	MCPStateConnected MCPConnectionState = "connected"
	// MCPStateReconnecting means attempting to reconnect.
	MCPStateReconnecting MCPConnectionState = "reconnecting"
	// MCPStateError means connection has an error.
	MCPStateError MCPConnectionState = "error"
)

// ConnectionPool manages multiple MCP server connections.
type ConnectionPool interface {
	// Add adds a server to the pool.
	Add(ctx context.Context, config MCPServerConfig) (MCPClient, error)

	// Remove removes a server from the pool.
	Remove(ctx context.Context, serverName string) error

	// Get returns a client by server name.
	Get(serverName string) (MCPClient, bool)

	// ListServers returns all connected server names.
	ListServers() []string

	// FindTool searches for a tool across all servers.
	FindTool(toolName string) (MCPClient, MCPTool, bool)

	// Close closes all connections.
	Close(ctx context.Context) error
}