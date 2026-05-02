package outbound

// SearchResult represents a single result from a web search.
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}

// SearchCapability is the interface for MCP providers that support web search.
type SearchCapability interface {
	// Search performs a web search for the given query and returns up to limit results.
	Search(query string, limit int) ([]SearchResult, error)
}

// GetSearchProvider returns the SearchCapability of an MCPClient if it supports it,
// along with a boolean indicating whether the capability is available.
func GetSearchProvider(client MCPClient) (SearchCapability, bool) {
	s, ok := client.(SearchCapability)
	return s, ok
}
