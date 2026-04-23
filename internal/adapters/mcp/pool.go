package mcp

import (
	"context"
	"sync"

	"devilfish/internal/ports/outbound"
	"devilfish/internal/domain/valueobject"
)

// Pool manages multiple MCP server connections.
type Pool struct {
	mu       sync.RWMutex
	clients  map[string]outbound.MCPClient
	cons     map[string]*valueobject.MCPConnection
	toolIdx  map[string]ToolRef // toolName -> server+tool
}

// ToolRef references a tool on a specific server.
type ToolRef struct {
	ServerName string
	Tool      outbound.MCPTool
}

// NewPool creates a new connection pool.
func NewPool() *Pool {
	return &Pool{
		clients: make(map[string]outbound.MCPClient),
		cons:    make(map[string]*valueobject.MCPConnection),
		toolIdx: make(map[string]ToolRef),
	}
}

// Add adds a server to the pool.
func (p *Pool) Add(ctx context.Context, config outbound.MCPServerConfig) (outbound.MCPClient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if already exists
	if client, exists := p.clients[config.Name]; exists {
		return client, nil
	}

	// Create new client based on transport
	var client outbound.MCPClient
	var err error

	switch config.Transport {
	case "http":
		client = NewHTTPTransport(config)
	case "stdio":
		client = NewStdIOTransport(config)
	default:
		client = NewStdIOTransport(config)
	}

	// Connect
	err = client.Connect(ctx, config)
	if err != nil {
		return nil, err
	}

	p.clients[config.Name] = client
	p.cons[config.Name] = valueobject.NewMCPConnection(config.Name)

	return client, nil
}

// Remove removes a server from the pool.
func (p *Pool) Remove(ctx context.Context, serverName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	client, exists := p.clients[serverName]
	if !exists {
		return nil
	}

	client.Disconnect(ctx)
	delete(p.clients, serverName)
	delete(p.cons, serverName)

	// Remove tools for this server from index
	for name, ref := range p.toolIdx {
		if ref.ServerName == serverName {
			delete(p.toolIdx, name)
		}
	}

	return nil
}

// Get returns a client by server name.
func (p *Pool) Get(serverName string) (outbound.MCPClient, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	client, exists := p.clients[serverName]
	return client, exists
}

// ListServers returns all connected server names.
func (p *Pool) ListServers() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	names := make([]string, 0, len(p.clients))
	for name := range p.clients {
		names = append(names, name)
	}
	return names
}

// FindTool searches for a tool across all servers.
func (p *Pool) FindTool(toolName string) (outbound.MCPClient, outbound.MCPTool, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ref, exists := p.toolIdx[toolName]
	if !exists {
		return nil, outbound.MCPTool{}, false
	}

	client, exists := p.clients[ref.ServerName]
	if !exists {
		return nil, outbound.MCPTool{}, false
	}

	return client, ref.Tool, true
}

// UpdateToolIndex updates the tool index for a server.
func (p *Pool) UpdateToolIndex(serverName string, tools []outbound.MCPTool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove old tools for this server
	for name, ref := range p.toolIdx {
		if ref.ServerName == serverName {
			delete(p.toolIdx, name)
		}
	}

	// Add new tools
	for _, tool := range tools {
		p.toolIdx[tool.Name] = ToolRef{
			ServerName: serverName,
			Tool:      tool,
		}
	}
}

// Connection returns the connection state for a server.
func (p *Pool) Connection(serverName string) *valueobject.MCPConnection {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.cons[serverName]
}

// Close closes all connections.
func (p *Pool) Close(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for name, client := range p.clients {
		if err := client.Disconnect(ctx); err != nil {
			errs = append(errs, err)
		}
		delete(p.clients, name)
		delete(p.cons, name)
	}

	p.toolIdx = make(map[string]ToolRef)

	if len(errs) > 0 {
		return errs[0] // Return first error
	}
	return nil
}