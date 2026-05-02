package outbound

import "errors"

// ErrCapabilityNotSupported is returned when a provider does not implement
// the requested capability interface.
var ErrCapabilityNotSupported = errors.New("capability not supported by this provider")

// ErrProviderNotFound is returned when no provider is registered under the
// requested name.
var ErrProviderNotFound = errors.New("provider not found in registry")

// MCP is the base interface that every capability provider must satisfy.
// Providers expose additional capabilities through optional interfaces such as
// SearchCapability; callers use the resolver helpers to obtain those.
type MCP interface {
	// Name returns the unique identifier of this provider.
	Name() string
}

// SearchCapability is an optional interface a provider may implement to expose
// web-search or document-search functionality.
type SearchCapability interface {
	// Search executes a search query and returns at most limit results.
	Search(query string, limit int) ([]SearchResult, error)
}

// SearchResult is the shared model returned by SearchCapability.Search.
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}
