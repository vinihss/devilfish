package websearch_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"devilfish/internal/adapters/websearch"
	"devilfish/internal/ports/outbound"
)

// apiResponse builds the JSON body that the fake search API returns.
func apiResponse(results []map[string]string) []byte {
	body := map[string]interface{}{"results": results}
	data, _ := json.Marshal(body)
	return data
}

// newTestServer creates an httptest.Server that returns the given status and body.
func newTestServer(status int, body []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(body) //nolint:errcheck
	}))
}

// newProvider creates a WebSearchMCP pointed at the given server URL.
func newProvider(server *httptest.Server) *websearch.WebSearchMCP {
	return websearch.NewWebSearchMCP(server.URL, "test-key", server.Client())
}

// --- Constructor ---

func TestNewWebSearchMCP_DefaultClient(t *testing.T) {
	w := websearch.NewWebSearchMCP("https://api.example.com", "key", nil)
	assert.NotNil(t, w.Client)
}

// --- MCPClient: Connect / IsConnected / Disconnect ---

func TestWebSearchMCP_Connect_SetsConnected(t *testing.T) {
	server := newTestServer(http.StatusOK, apiResponse(nil))
	defer server.Close()

	w := websearch.NewWebSearchMCP(server.URL, "key", server.Client())
	require.False(t, w.IsConnected())

	err := w.Connect(context.Background(), outbound.MCPServerConfig{})
	require.NoError(t, err)
	assert.True(t, w.IsConnected())
}

func TestWebSearchMCP_Connect_WithConfig_OverridesEndpointAndKey(t *testing.T) {
	server := newTestServer(http.StatusOK, apiResponse(nil))
	defer server.Close()

	w := websearch.NewWebSearchMCP("", "", server.Client())
	err := w.Connect(context.Background(), outbound.MCPServerConfig{
		URL:       server.URL,
		AuthToken: "new-key",
	})
	require.NoError(t, err)
	assert.True(t, w.IsConnected())
	assert.Equal(t, server.URL, w.Endpoint)
	assert.Equal(t, "new-key", w.APIKey)
}

func TestWebSearchMCP_Connect_MissingEndpoint_ReturnsError(t *testing.T) {
	w := websearch.NewWebSearchMCP("", "", nil)
	err := w.Connect(context.Background(), outbound.MCPServerConfig{})
	assert.True(t, errors.Is(err, websearch.ErrMissingEndpoint))
	assert.False(t, w.IsConnected())
}

func TestWebSearchMCP_Disconnect_SetsNotConnected(t *testing.T) {
	server := newTestServer(http.StatusOK, apiResponse(nil))
	defer server.Close()

	w := newProvider(server)
	require.NoError(t, w.Connect(context.Background(), outbound.MCPServerConfig{}))
	require.True(t, w.IsConnected())

	require.NoError(t, w.Disconnect(context.Background()))
	assert.False(t, w.IsConnected())
}

// --- MCPClient: ServerName ---

func TestWebSearchMCP_ServerName(t *testing.T) {
	w := websearch.NewWebSearchMCP("https://api.example.com", "key", nil)
	assert.Equal(t, "websearch", w.ServerName())
}

// --- MCPClient: ListTools ---

func TestWebSearchMCP_ListTools_ReturnsWebSearchTool(t *testing.T) {
	w := websearch.NewWebSearchMCP("https://api.example.com", "key", nil)
	tools, err := w.ListTools(context.Background())
	require.NoError(t, err)
	require.Len(t, tools, 1)
	assert.Equal(t, "web_search", tools[0].Name)
	assert.NotEmpty(t, tools[0].Description)
	assert.NotNil(t, tools[0].InputSchema)
}

// --- MCPClient: ExecuteTool ---

func TestWebSearchMCP_ExecuteTool_ReturnsResults(t *testing.T) {
	body := apiResponse([]map[string]string{
		{"title": "Go Language", "url": "https://go.dev", "snippet": "The Go programming language"},
	})
	server := newTestServer(http.StatusOK, body)
	defer server.Close()

	w := newProvider(server)
	result, err := w.ExecuteTool(context.Background(), "web_search", map[string]interface{}{
		"query": "golang",
		"limit": 5,
	})
	require.NoError(t, err)
	results, ok := result["results"].([]outbound.SearchResult)
	require.True(t, ok)
	require.Len(t, results, 1)
	assert.Equal(t, "Go Language", results[0].Title)
}

func TestWebSearchMCP_ExecuteTool_UnknownTool_ReturnsError(t *testing.T) {
	w := websearch.NewWebSearchMCP("https://api.example.com", "key", nil)
	_, err := w.ExecuteTool(context.Background(), "unknown_tool", map[string]interface{}{"query": "test"})
	assert.Error(t, err)
}

func TestWebSearchMCP_ExecuteTool_MissingQuery_ReturnsError(t *testing.T) {
	w := websearch.NewWebSearchMCP("https://api.example.com", "key", nil)
	_, err := w.ExecuteTool(context.Background(), "web_search", map[string]interface{}{})
	assert.True(t, errors.Is(err, websearch.ErrMissingQuery))
}

func TestWebSearchMCP_ExecuteTool_LimitAsFloat64(t *testing.T) {
	body := apiResponse([]map[string]string{
		{"title": "A", "url": "https://a.com", "snippet": "snippet a"},
	})
	server := newTestServer(http.StatusOK, body)
	defer server.Close()

	w := newProvider(server)
	// JSON numbers are float64 when decoded into interface{}
	result, err := w.ExecuteTool(context.Background(), "web_search", map[string]interface{}{
		"query": "golang",
		"limit": float64(3),
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// --- SearchCapability: Search ---

func TestWebSearchMCP_Search_ReturnsResults(t *testing.T) {
	body := apiResponse([]map[string]string{
		{"title": "Go Blog", "url": "https://go.dev/blog", "snippet": "Latest news"},
		{"title": "Go Docs", "url": "https://pkg.go.dev", "snippet": "Documentation"},
	})
	server := newTestServer(http.StatusOK, body)
	defer server.Close()

	w := newProvider(server)
	results, err := w.Search("golang", 5)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "Go Blog", results[0].Title)
	assert.Equal(t, "https://go.dev/blog", results[0].URL)
	assert.Equal(t, "Latest news", results[0].Snippet)
}

func TestWebSearchMCP_Search_EmptyResults_ReturnsEmptySlice(t *testing.T) {
	server := newTestServer(http.StatusOK, apiResponse(nil))
	defer server.Close()

	w := newProvider(server)
	results, err := w.Search("obscure query", 5)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestWebSearchMCP_Search_EmptyQuery_ReturnsError(t *testing.T) {
	w := websearch.NewWebSearchMCP("https://api.example.com", "key", nil)
	_, err := w.Search("", 5)
	assert.True(t, errors.Is(err, websearch.ErrMissingQuery))
}

func TestWebSearchMCP_Search_ZeroLimit_ReturnsError(t *testing.T) {
	w := websearch.NewWebSearchMCP("https://api.example.com", "key", nil)
	_, err := w.Search("golang", 0)
	assert.True(t, errors.Is(err, websearch.ErrInvalidLimit))
}

func TestWebSearchMCP_Search_Non200Response_ReturnsError(t *testing.T) {
	server := newTestServer(http.StatusInternalServerError, []byte(`{"error":"server error"}`))
	defer server.Close()

	w := newProvider(server)
	_, err := w.Search("golang", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestWebSearchMCP_Search_InvalidJSON_ReturnsError(t *testing.T) {
	server := newTestServer(http.StatusOK, []byte(`not-json`))
	defer server.Close()

	w := newProvider(server)
	_, err := w.Search("golang", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestWebSearchMCP_Search_RequestTimeout_ReturnsError(t *testing.T) {
	// Server that blocks until client times out.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	fastClient := &http.Client{Timeout: 50 * time.Millisecond}
	w := websearch.NewWebSearchMCP(server.URL, "key", fastClient)

	_, err := w.Search("golang", 5)
	require.Error(t, err)
}

// --- GetSearchProvider ---

func TestGetSearchProvider_WithWebSearchMCP_ReturnsCapability(t *testing.T) {
	w := websearch.NewWebSearchMCP("https://api.example.com", "key", nil)
	cap, ok := outbound.GetSearchProvider(w)
	assert.True(t, ok)
	assert.NotNil(t, cap)
}
