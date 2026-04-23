package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"devilfish/internal/ports/outbound"
	"devilfish/internal/domain/valueobject"
)

// Registry manages multiple MCP server connections and routing.
type Registry struct {
	mu       sync.RWMutex
	servers  map[string]*ServerState
	pool     *Pool
	callback func(serverName string, state valueobject.ConnectionState)
}

// ServerState holds state for a managed server.
type ServerState struct {
	Config       outbound.MCPServerConfig
	Connection  *valueobject.MCPConnection
	LastHealth  time.Time
	HealthCheck bool
}

// NewRegistry creates a new server registry.
func NewRegistry(callback func(serverName string, state valueobject.ConnectionState)) *Registry {
	return &Registry{
		servers:  make(map[string]*ServerState),
		pool:    NewPool(),
		callback: callback,
	}
}

// Add adds a server to the registry.
func (r *Registry) Add(ctx context.Context, config outbound.MCPServerConfig) (outbound.MCPClient, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already exists - return existing client
	if _, exists := r.servers[config.Name]; exists {
		client, ok := r.pool.Get(config.Name)
		if ok {
			return client, nil
		}
		// Server exists in registry but not in pool - add it
	}

	// Add to pool
	newClient, err := r.pool.Add(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to add server %s: %w", config.Name, err)
	}

	r.servers[config.Name] = &ServerState{
		Config:      config,
		Connection: valueobject.NewMCPConnection(config.Name),
		LastHealth:  time.Now(),
	}

	// Update tool index
	tools, _ := newClient.ListTools(ctx)
	r.pool.UpdateToolIndex(config.Name, tools)

	return newClient, nil
}

// Remove removes a server from the registry.
func (r *Registry) Remove(ctx context.Context, serverName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.servers[serverName]; !exists {
		return nil
	}

	r.pool.Remove(ctx, serverName)
	delete(r.servers, serverName)

	return nil
}

// Get returns a client by server name.
func (r *Registry) Get(serverName string) (outbound.MCPClient, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.servers[serverName]
	if !exists {
		return nil, false
	}

	return r.pool.Get(serverName)
}

// ListServers returns all registered server names.
func (r *Registry) ListServers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.servers))
	for name := range r.servers {
		names = append(names, name)
	}
	return names
}

// RouteTool finds the appropriate server for a tool.
func (r *Registry) RouteTool(toolName string) (string, outbound.MCPTool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Search for tool across all servers
	for name := range r.servers {
		client, exists := r.pool.Get(name)
		if !exists || !client.IsConnected() {
			continue
		}

		tools, err := client.ListTools(context.Background())
		if err != nil {
			continue
		}

		for _, tool := range tools {
			if tool.Name == toolName {
				return name, tool, true
			}
		}
	}

	return "", outbound.MCPTool{}, false
}

// ResolveConflict resolves duplicate tool name conflicts.
func (r *Registry) ResolveConflict(toolName string) (serverName string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First server wins (deterministic)
	for name := range r.servers {
		return name
	}
	return ""
}

// CheckHealth checks connection health for all servers.
func (r *Registry) CheckHealth(ctx context.Context) map[string]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[string]bool)

	for name, state := range r.servers {
		client, exists := r.pool.Get(name)
		if !exists {
			results[name] = false
			continue
		}

		connected := client.IsConnected()
		state.HealthCheck = connected
		state.LastHealth = time.Now()
		results[name] = connected

		// Notify callback
		if r.callback != nil {
			if connected {
				r.callback(name, valueobject.StateConnected)
			} else {
				r.callback(name, valueobject.StateError)
			}
		}
	}

	return results
}

// Close closes all connections.
func (r *Registry) Close(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.pool.Close(ctx)
}