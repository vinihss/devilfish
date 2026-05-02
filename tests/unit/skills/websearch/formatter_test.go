package websearch_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"devilfish/internal/ports/outbound"
	skillws "devilfish/internal/skills/websearch"
)

func TestFormatResults_EmptySlice(t *testing.T) {
	out := skillws.FormatResults([]outbound.SearchResult{})
	assert.Contains(t, out, "[TOOL RESULT - web_search]")
	assert.Contains(t, out, "No results found.")
}

func TestFormatResults_SingleResult(t *testing.T) {
	results := []outbound.SearchResult{
		{Title: "Go Language", URL: "https://go.dev", Snippet: "The Go programming language."},
	}
	out := skillws.FormatResults(results)
	assert.Contains(t, out, "[TOOL RESULT - web_search]")
	assert.Contains(t, out, "1. Title: Go Language")
	assert.Contains(t, out, "URL: https://go.dev")
	assert.Contains(t, out, "Snippet: The Go programming language.")
}

func TestFormatResults_MultipleResults_NumberedInOrder(t *testing.T) {
	results := []outbound.SearchResult{
		{Title: "First", URL: "https://first.com", Snippet: "first snippet"},
		{Title: "Second", URL: "https://second.com", Snippet: "second snippet"},
		{Title: "Third", URL: "https://third.com", Snippet: "third snippet"},
	}
	out := skillws.FormatResults(results)
	assert.Contains(t, out, "1. Title: First")
	assert.Contains(t, out, "2. Title: Second")
	assert.Contains(t, out, "3. Title: Third")
}
