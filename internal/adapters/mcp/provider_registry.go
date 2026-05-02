package mcp

import (
	"fmt"
	"sync"

	"devilfish/internal/ports/outbound"
)

// MCPRegistry holds registered capability providers and allows lookup by name.
// It is safe for concurrent use.
type MCPRegistry struct {
	mu        sync.RWMutex
	providers map[string]outbound.MCP
}

// NewMCPRegistry creates an empty, ready-to-use MCPRegistry.
func NewMCPRegistry() *MCPRegistry {
	return &MCPRegistry{
		providers: make(map[string]outbound.MCP),
	}
}

// Register adds a provider to the registry.  If a provider with the same name
// already exists it is replaced by the new one.  Callers that need idempotency
// should call Get first to check whether a name is already taken.
func (r *MCPRegistry) Register(mcp outbound.MCP) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[mcp.Name()] = mcp
}

// Get returns the provider registered under name together with a boolean flag
// that reports whether the lookup succeeded.
func (r *MCPRegistry) Get(name string) (outbound.MCP, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[name]
	return p, ok
}

// MustGet returns the provider registered under name or an error if it is not
// present.  This is a convenience wrapper around Get.
func (r *MCPRegistry) MustGet(name string) (outbound.MCP, error) {
	p, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("%w: %s", outbound.ErrProviderNotFound, name)
	}
	return p, nil
}

// List returns the names of all registered providers in an unspecified order.
func (r *MCPRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// GetSearchProvider performs a safe type-assertion to check whether the given
// provider implements SearchCapability.  The second return value is false when
// the provider does not support search.
func GetSearchProvider(mcp outbound.MCP) (outbound.SearchCapability, bool) {
	sc, ok := mcp.(outbound.SearchCapability)
	return sc, ok
}
