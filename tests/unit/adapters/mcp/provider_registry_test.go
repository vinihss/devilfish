package mcp_test

import (
	"errors"
	"testing"

	mcpadapter "devilfish/internal/adapters/mcp"
	"devilfish/internal/ports/outbound"
)

// --- test doubles -----------------------------------------------------------

// stubProvider is a minimal MCP provider that does not expose any capability.
type stubProvider struct{ name string }

func (s *stubProvider) Name() string { return s.name }

// searchProvider is a provider that implements SearchCapability.
type searchProvider struct {
	name    string
	results []outbound.SearchResult
	err     error
}

func (s *searchProvider) Name() string { return s.name }

func (s *searchProvider) Search(query string, limit int) ([]outbound.SearchResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	if limit > 0 && limit < len(s.results) {
		return s.results[:limit], nil
	}
	return s.results, nil
}

// --- MCPRegistry tests -------------------------------------------------------

func TestMCPRegistry_Register_And_Get_ReturnsProvider(t *testing.T) {
	r := mcpadapter.NewMCPRegistry()
	p := &stubProvider{name: "my-provider"}

	r.Register(p)

	got, ok := r.Get("my-provider")
	if !ok {
		t.Fatal("expected provider to be found")
	}
	if got.Name() != "my-provider" {
		t.Errorf("expected name my-provider, got %s", got.Name())
	}
}

func TestMCPRegistry_Get_UnknownName_ReturnsFalse(t *testing.T) {
	r := mcpadapter.NewMCPRegistry()

	_, ok := r.Get("unknown")
	if ok {
		t.Fatal("expected ok=false for unknown provider")
	}
}

func TestMCPRegistry_Register_ReplacesExistingProvider(t *testing.T) {
	r := mcpadapter.NewMCPRegistry()
	first := &stubProvider{name: "p"}
	second := &stubProvider{name: "p"}

	r.Register(first)
	r.Register(second)

	got, ok := r.Get("p")
	if !ok {
		t.Fatal("expected provider to be found")
	}
	if got != second {
		t.Error("expected second registration to replace the first")
	}
}

func TestMCPRegistry_MustGet_ReturnsErrorForUnknownProvider(t *testing.T) {
	r := mcpadapter.NewMCPRegistry()

	_, err := r.MustGet("missing")
	if err == nil {
		t.Fatal("expected an error for missing provider")
	}
	if !errors.Is(err, outbound.ErrProviderNotFound) {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestMCPRegistry_MustGet_ReturnsProviderWhenFound(t *testing.T) {
	r := mcpadapter.NewMCPRegistry()
	p := &stubProvider{name: "found"}
	r.Register(p)

	got, err := r.MustGet("found")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name() != "found" {
		t.Errorf("expected name found, got %s", got.Name())
	}
}

func TestMCPRegistry_List_ReturnsAllNames(t *testing.T) {
	r := mcpadapter.NewMCPRegistry()
	r.Register(&stubProvider{name: "a"})
	r.Register(&stubProvider{name: "b"})
	r.Register(&stubProvider{name: "c"})

	names := r.List()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
}

func TestMCPRegistry_List_Empty_ReturnsEmptySlice(t *testing.T) {
	r := mcpadapter.NewMCPRegistry()

	names := r.List()
	if len(names) != 0 {
		t.Fatalf("expected empty list, got %d items", len(names))
	}
}

// --- GetSearchProvider resolver tests ----------------------------------------

func TestGetSearchProvider_WithSearchCapability_ReturnsTrueAndInterface(t *testing.T) {
	sp := &searchProvider{
		name:    "web-search",
		results: []outbound.SearchResult{{Title: "Go", URL: "https://go.dev", Snippet: "The Go language"}},
	}

	sc, ok := mcpadapter.GetSearchProvider(sp)
	if !ok {
		t.Fatal("expected ok=true for a provider with SearchCapability")
	}
	if sc == nil {
		t.Fatal("expected non-nil SearchCapability")
	}
}

func TestGetSearchProvider_WithoutSearchCapability_ReturnsFalse(t *testing.T) {
	p := &stubProvider{name: "no-search"}

	_, ok := mcpadapter.GetSearchProvider(p)
	if ok {
		t.Fatal("expected ok=false for a provider without SearchCapability")
	}
}

func TestGetSearchProvider_Search(t *testing.T) {
	threeResults := []outbound.SearchResult{
		{Title: "A", URL: "https://a.com", Snippet: "a"},
		{Title: "B", URL: "https://b.com", Snippet: "b"},
		{Title: "C", URL: "https://c.com", Snippet: "c"},
	}
	errNetwork := errors.New("network error")

	cases := []struct {
		name     string
		provider *searchProvider
		query    string
		limit    int
		wantLen  int
		wantErr  error
	}{
		{
			name:     "respects limit",
			provider: &searchProvider{name: "web-search", results: threeResults},
			query:    "go lang",
			limit:    2,
			wantLen:  2,
		},
		{
			name:     "returns all when limit exceeds results",
			provider: &searchProvider{name: "web-search", results: threeResults},
			query:    "go lang",
			limit:    10,
			wantLen:  3,
		},
		{
			name:     "propagates error",
			provider: &searchProvider{name: "failing-search", err: errNetwork},
			query:    "anything",
			limit:    5,
			wantErr:  errNetwork,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sc, ok := mcpadapter.GetSearchProvider(tc.provider)
			if !ok {
				t.Fatal("expected SearchCapability")
			}

			results, err := sc.Search(tc.query, tc.limit)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(results) != tc.wantLen {
				t.Errorf("expected %d results, got %d", tc.wantLen, len(results))
			}
		})
	}
}

// --- Integration: registry + resolver ----------------------------------------

func TestMCPRegistry_IntegrationWithResolver_SearchProviderRetrievedAndUsed(t *testing.T) {
	r := mcpadapter.NewMCPRegistry()
	sp := &searchProvider{
		name:    "duckduckgo",
		results: []outbound.SearchResult{{Title: "Duck", URL: "https://ddg.gg", Snippet: "private search"}},
	}
	r.Register(sp)

	provider, ok := r.Get("duckduckgo")
	if !ok {
		t.Fatal("expected provider to be found")
	}

	sc, ok := mcpadapter.GetSearchProvider(provider)
	if !ok {
		t.Fatal("expected SearchCapability")
	}

	results, err := sc.Search("privacy", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least one result")
	}
	if results[0].Title != "Duck" {
		t.Errorf("unexpected result title: %s", results[0].Title)
	}
}
