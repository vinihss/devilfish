package websearch

import (
	"fmt"

	"devilfish/internal/ports/outbound"
)

const (
	// SkillName is the registered identifier for this skill.
	SkillName = "web_search"
	// SkillDescription describes what the skill does.
	SkillDescription = "Search the web for information on a given query."

	defaultLimit    = 10
	websearchServer = "websearch"
)

// MCPProvider abstracts access to MCP clients by server name.
// *mcp.Registry satisfies this interface, but any implementation works,
// which keeps the skill decoupled from a concrete registry type.
type MCPProvider interface {
	Get(serverName string) (outbound.MCPClient, bool)
}

// WebSearchSkill executes web searches by resolving the "websearch" MCP
// provider and delegating to its SearchCapability.
type WebSearchSkill struct {
	Provider MCPProvider
}

// NewWebSearchSkill creates a WebSearchSkill backed by the given MCPProvider.
func NewWebSearchSkill(provider MCPProvider) *WebSearchSkill {
	return &WebSearchSkill{Provider: provider}
}

// Name returns the skill identifier.
func (s *WebSearchSkill) Name() string { return SkillName }

// Description returns a human-readable description of the skill.
func (s *WebSearchSkill) Description() string { return SkillDescription }

// Execute performs a web search.
//
// Expected input keys:
//   - "query" (string, required)  — the search terms
//   - "limit" (int or float64, optional) — maximum number of results
func (s *WebSearchSkill) Execute(input map[string]interface{}) (string, error) {
	query, err := extractQuery(input)
	if err != nil {
		return "", err
	}
	limit := extractLimit(input)

	client, ok := s.Provider.Get(websearchServer)
	if !ok {
		return "", fmt.Errorf("websearch: MCP provider %q not registered", websearchServer)
	}

	search, ok := outbound.GetSearchProvider(client)
	if !ok {
		return "", fmt.Errorf("websearch: MCP provider %q does not support SearchCapability", websearchServer)
	}

	results, err := search.Search(query, limit)
	if err != nil {
		return "", fmt.Errorf("websearch: search failed: %w", err)
	}

	return FormatResults(results), nil
}

// extractQuery pulls the required "query" string from the input map.
func extractQuery(input map[string]interface{}) (string, error) {
	v, ok := input["query"]
	if !ok {
		return "", fmt.Errorf("websearch: missing required input \"query\"")
	}
	q, ok := v.(string)
	if !ok || q == "" {
		return "", fmt.Errorf("websearch: \"query\" must be a non-empty string")
	}
	return q, nil
}

// extractLimit pulls the optional "limit" value from the input map.
// It accepts int and float64 (JSON numbers decode as float64). Non-positive
// values are ignored and the default is returned.
func extractLimit(input map[string]interface{}) int {
	v, ok := input["limit"]
	if !ok {
		return defaultLimit
	}
	switch lv := v.(type) {
	case int:
		if lv > 0 {
			return lv
		}
	case float64:
		if lv > 0 {
			return int(lv)
		}
	}
	return defaultLimit
}
