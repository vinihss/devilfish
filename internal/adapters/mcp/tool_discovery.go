package mcp

import (
	"context"
	"fmt"
	"sync"

	"devilfish/internal/domain/entity"
)

// ToolDiscovery handles tool discovery and listing across MCP servers.
type ToolDiscovery struct {
	mu    sync.RWMutex
	tools map[string]map[string]entity.Tool // serverName -> toolName -> Tool
	pool  *Pool
}

// NewToolDiscovery creates a new tool discovery instance.
func NewToolDiscovery(pool *Pool) *ToolDiscovery {
	return &ToolDiscovery{
		pool:  pool,
		tools: make(map[string]map[string]entity.Tool),
	}
}

// DiscoverTools fetches and caches tools from all connected servers.
func (d *ToolDiscovery) DiscoverTools(ctx context.Context) error {
	servers := d.pool.ListServers()

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, serverName := range servers {
		client, exists := d.pool.Get(serverName)
		if !exists {
			continue
		}

		mcpTools, err := client.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("failed to list tools from %s: %w", serverName, err)
		}

		// Convert and cache
		tools := make(map[string]entity.Tool, len(mcpTools))
		for _, mcpTool := range mcpTools {
			tool := entity.NewTool(
				fmt.Sprintf("%s-%s", serverName, mcpTool.Name),
				mcpTool.Name,
				mcpTool.Description,
				serverName,
			)
			if mcpTool.InputSchema != nil {
				tool.WithInputSchema(mcpTool.InputSchema)
			}
			if mcpTool.OutputSchema != nil {
				tool.WithOutputSchema(mcpTool.OutputSchema)
			}
			tools[mcpTool.Name] = *tool
		}

		d.tools[serverName] = tools
		d.pool.UpdateToolIndex(serverName, mcpTools)
	}

	return nil
}

// GetTool returns a tool by name from a specific server.
func (d *ToolDiscovery) GetTool(serverName, toolName string) (entity.Tool, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	serverTools, exists := d.tools[serverName]
	if !exists {
		return entity.Tool{}, false
	}

	tool, exists := serverTools[toolName]
	return tool, exists
}

// GetAllTools returns all cached tools.
func (d *ToolDiscovery) GetAllTools() map[string]map[string]entity.Tool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make(map[string]map[string]entity.Tool)
	for k, v := range d.tools {
		result[k] = v
	}
	return result
}

// ListToolNames returns all unique tool names across all servers.
func (d *ToolDiscovery) ListToolNames() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	unique := make(map[string]struct{})
	for _, serverTools := range d.tools {
		for name := range serverTools {
			unique[name] = struct{}{}
		}
	}

	names := make([]string, 0, len(unique))
	for name := range unique {
		names = append(names, name)
	}
	return names
}

// Invalidate removes all cached tools for a server.
func (d *ToolDiscovery) Invalidate(serverName string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.tools, serverName)
}

// ExecuteTool executes a tool with validation.
func (d *ToolDiscovery) ExecuteTool(ctx context.Context, serverName, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	// Get tool for validation
	tool, exists := d.GetTool(serverName, toolName)
	if !exists {
		return nil, fmt.Errorf("tool %s not found on server %s", toolName, serverName)
	}

	// Validate input
	if err := tool.ValidateInput(params); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Execute via pool
	client, exists := d.pool.Get(serverName)
	if !exists {
		return nil, fmt.Errorf("server %s not connected", serverName)
	}

	return client.ExecuteTool(ctx, toolName, params)
}