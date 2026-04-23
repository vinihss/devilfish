package valueobject

import (
	"time"
)

// MCPConnection represents a connection to an MCP server.
type MCPConnection struct {
	serverName    string
	state        ConnectionState
	connectedAt  *time.Time
	lastActive  *time.Time
	err         error
}

// ConnectionState represents the state of a connection.
type ConnectionState string

const (
	// StateDisconnected means no connection.
	StateDisconnected ConnectionState = "disconnected"
	// StateConnecting means connection in progress.
	StateConnecting ConnectionState = "connecting"
	// StateConnected means successfully connected.
	StateConnected ConnectionState = "connected"
	// StateReconnecting means attempting to reconnect.
	StateReconnecting ConnectionState = "reconnecting"
	// StateError means connection has an error.
	StateError ConnectionState = "error"
)

// NewMCPConnection creates a new MCP connection value object.
func NewMCPConnection(serverName string) *MCPConnection {
	return &MCPConnection{
		serverName: serverName,
		state:    StateDisconnected,
	}
}

// ServerName returns the server name.
func (c *MCPConnection) ServerName() string {
	return c.serverName
}

// State returns the connection state.
func (c *MCPConnection) State() ConnectionState {
	return c.state
}

// IsConnected returns whether the connection is active.
func (c *MCPConnection) IsConnected() bool {
	return c.state == StateConnected
}

// Connect marks the connection as established.
func (c *MCPConnection) Connect() {
	now := time.Now()
	c.state = StateConnected
	c.connectedAt = &now
	c.lastActive = &now
	c.err = nil
}

// Disconnect marks the connection as closed.
func (c *MCPConnection) Disconnect() {
	c.state = StateDisconnected
}

// SetError sets an error state.
func (c *MCPConnection) SetError(err error) {
	c.state = StateError
	c.err = err
}

// Error returns the connection error.
func (c *MCPConnection) Error() error {
	return c.err
}

// ConnectedAt returns when the connection was established.
func (c *MCPConnection) ConnectedAt() *time.Time {
	return c.connectedAt
}

// LastActive returns the last activity time.
func (c *MCPConnection) LastActive() *time.Time {
	return c.lastActive
}

// Touch updates the last active time.
func (c *MCPConnection) Touch() {
	now := time.Now()
	c.lastActive = &now
}