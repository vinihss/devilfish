package websearch_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"devilfish/internal/ports/outbound"
	skillws "devilfish/internal/skills/websearch"
)

// --- fakes ---

// fakeClient is a minimal outbound.MCPClient + outbound.SearchCapability.
type fakeClient struct {
	results []outbound.SearchResult
	err     error
}

func (f *fakeClient) Connect(_ context.Context, _ outbound.MCPServerConfig) error { return nil }
func (f *fakeClient) Disconnect(_ context.Context) error                          { return nil }
func (f *fakeClient) IsConnected() bool                                           { return true }
func (f *fakeClient) ServerName() string                                          { return "websearch" }
func (f *fakeClient) ListTools(_ context.Context) ([]outbound.MCPTool, error)     { return nil, nil }
func (f *fakeClient) ExecuteTool(_ context.Context, _ string, _ map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}
func (f *fakeClient) Search(_ string, _ int) ([]outbound.SearchResult, error) {
	return f.results, f.err
}

// noSearchClient implements MCPClient but NOT SearchCapability.
type noSearchClient struct{}

func (n *noSearchClient) Connect(_ context.Context, _ outbound.MCPServerConfig) error { return nil }
func (n *noSearchClient) Disconnect(_ context.Context) error                          { return nil }
func (n *noSearchClient) IsConnected() bool                                           { return true }
func (n *noSearchClient) ServerName() string                                          { return "websearch" }
func (n *noSearchClient) ListTools(_ context.Context) ([]outbound.MCPTool, error)     { return nil, nil }
func (n *noSearchClient) ExecuteTool(_ context.Context, _ string, _ map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// fakeRegistry is a minimal MCPProvider implementation.
type fakeRegistry struct {
	client outbound.MCPClient
	found  bool
}

func (r *fakeRegistry) Get(_ string) (outbound.MCPClient, bool) {
	return r.client, r.found
}

// --- tests ---

func TestWebSearchSkill_Name(t *testing.T) {
	s := skillws.NewWebSearchSkill(&fakeRegistry{})
	assert.Equal(t, "web_search", s.Name())
}

func TestWebSearchSkill_Description_NotEmpty(t *testing.T) {
	s := skillws.NewWebSearchSkill(&fakeRegistry{})
	assert.NotEmpty(t, s.Description())
}

func TestWebSearchSkill_ProviderFieldSet(t *testing.T) {
	reg := &fakeRegistry{}
	s := skillws.NewWebSearchSkill(reg)
	assert.Equal(t, reg, s.Provider)
}

func TestWebSearchSkill_Execute_ReturnsFormattedResults(t *testing.T) {
	client := &fakeClient{
		results: []outbound.SearchResult{
			{Title: "Go Blog", URL: "https://go.dev/blog", Snippet: "News from the Go team"},
		},
	}
	reg := &fakeRegistry{client: client, found: true}
	s := skillws.NewWebSearchSkill(reg)

	out, err := s.Execute(map[string]interface{}{"query": "golang"})
	require.NoError(t, err)
	assert.Contains(t, out, "[TOOL RESULT - web_search]")
	assert.Contains(t, out, "Go Blog")
}

func TestWebSearchSkill_Execute_MissingQuery_ReturnsError(t *testing.T) {
	s := skillws.NewWebSearchSkill(&fakeRegistry{found: true, client: &fakeClient{}})
	_, err := s.Execute(map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query")
}

func TestWebSearchSkill_Execute_EmptyQuery_ReturnsError(t *testing.T) {
	s := skillws.NewWebSearchSkill(&fakeRegistry{found: true, client: &fakeClient{}})
	_, err := s.Execute(map[string]interface{}{"query": ""})
	require.Error(t, err)
}

func TestWebSearchSkill_Execute_ProviderNotFound_ReturnsError(t *testing.T) {
	s := skillws.NewWebSearchSkill(&fakeRegistry{found: false})
	_, err := s.Execute(map[string]interface{}{"query": "test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestWebSearchSkill_Execute_NoSearchCapability_ReturnsError(t *testing.T) {
	reg := &fakeRegistry{client: &noSearchClient{}, found: true}
	s := skillws.NewWebSearchSkill(reg)
	_, err := s.Execute(map[string]interface{}{"query": "test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SearchCapability")
}

func TestWebSearchSkill_Execute_SearchError_ReturnsError(t *testing.T) {
	client := &fakeClient{err: fmt.Errorf("network timeout")}
	reg := &fakeRegistry{client: client, found: true}
	s := skillws.NewWebSearchSkill(reg)

	_, err := s.Execute(map[string]interface{}{"query": "golang"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network timeout")
}

func TestWebSearchSkill_Execute_CustomLimit(t *testing.T) {
	client := &fakeClient{results: []outbound.SearchResult{}}
	reg := &fakeRegistry{client: client, found: true}
	s := skillws.NewWebSearchSkill(reg)

	out, err := s.Execute(map[string]interface{}{"query": "golang", "limit": float64(3)})
	require.NoError(t, err)
	assert.Contains(t, out, "No results found.")
}

func TestWebSearchSkill_Execute_EmptyResults_ReturnsNoResultsMessage(t *testing.T) {
	client := &fakeClient{results: []outbound.SearchResult{}}
	reg := &fakeRegistry{client: client, found: true}
	s := skillws.NewWebSearchSkill(reg)

	out, err := s.Execute(map[string]interface{}{"query": "very obscure query"})
	require.NoError(t, err)
	assert.Contains(t, out, "No results found.")
}
