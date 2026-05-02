package websearch

import (
	"fmt"
	"strings"

	"devilfish/internal/ports/outbound"
)

// FormatResults formats a slice of SearchResult into a human-readable block
// that can be appended to the LLM context as a tool result.
//
// Format:
//
//	[TOOL RESULT - web_search]
//
//	1. Title: ...
//	   URL: ...
//	   Snippet: ...
func FormatResults(results []outbound.SearchResult) string {
	if len(results) == 0 {
		return "[TOOL RESULT - web_search]\n\nNo results found."
	}

	var sb strings.Builder
	sb.WriteString("[TOOL RESULT - web_search]\n\n")

	for i, r := range results {
		fmt.Fprintf(&sb, "%d. Title: %s\n   URL: %s\n   Snippet: %s\n\n",
			i+1, r.Title, r.URL, r.Snippet)
	}

	return strings.TrimRight(sb.String(), "\n")
}
