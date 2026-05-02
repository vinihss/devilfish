package outbound

import "context"

// Email represents an email message in the system.
// It is used for both listing emails and reading individual messages.
type Email struct {
	ID      string `json:"id"`
	ThreadID string `json:"thread_id,omitempty"`
	From    string `json:"from"`
	To      []string `json:"to"`
	Subject string `json:"subject"`
	Snippet string `json:"snippet,omitempty"`
	Body    string `json:"body,omitempty"`
	Date    string `json:"date,omitempty"`
	Labels  []string `json:"labels,omitempty"`
}

// EmailCapability defines the interface for email operations
// through an MCP-compatible server (e.g., Gmail MCP server).
//
// Implementations of this interface should use the MCPClient to execute
// email-related tools on the connected MCP server. The capability acts
// as a higher-level abstraction over the raw MCP tool execution.
//
// Example usage with MCP registry:
//
//	client, tool, found := pool.FindTool("send_email")
//	if !found {
//		return fmt.Errorf("send_email tool not available")
//	}
//	// Use client.ExecuteTool to perform the operation
type EmailCapability interface {
	// SendEmail sends an email to the specified recipient.
	// Returns an error if the email cannot be sent.
	SendEmail(ctx context.Context, to, subject, body string) error

	// ListEmails lists emails matching the given query string.
	// The query uses Gmail search syntax (e.g., "is:unread", "from:example@gmail.com").
	// Returns a slice of Email objects and any error encountered.
	ListEmails(ctx context.Context, query string) ([]Email, error)

	// ReadEmail reads a single email by its ID.
	// Returns the full Email object including body content and any error encountered.
	ReadEmail(ctx context.Context, id string) (Email, error)
}
