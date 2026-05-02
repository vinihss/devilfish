package outbound

import "context"

// File represents a file or folder in Google Drive.
type File struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MIMEType  string `json:"mime_type"`
	Size      int64  `json:"size,omitempty"`
	ModifiedTime string `json:"modified_time,omitempty"`
	CreatedTime  string `json:"created_time,omitempty"`
	Parents     []string `json:"parents,omitempty"`
	WebViewLink string `json:"web_view_link,omitempty"`
	Content    string `json:"content,omitempty"` // File content (for readable files)
}

// DriveCapability defines the interface for Google Drive operations
// through an MCP-compatible server (e.g., Google Drive MCP server).
//
// Implementations of this interface should use the MCPClient to execute
// Drive-related tools on the connected MCP server. The capability acts
// as a higher-level abstraction over the raw MCP tool execution.
//
// Example usage with MCP registry:
//
//	client, tool, found := pool.FindTool("list_files")
//	if !found {
//		return fmt.Errorf("list_files tool not available")
//	}
//	// Use client.ExecuteTool to perform the operation
type DriveCapability interface {
	// ListFiles lists files matching the given query string.
	// The query uses Google Drive search syntax (e.g., "name contains 'report'", "mimeType='application/pdf'").
	// Returns a slice of File objects and any error encountered.
	ListFiles(ctx context.Context, query string) ([]File, error)

	// ReadFile reads the content of a file by its ID.
	// For Google Docs/Sheets, this may return exported content in a readable format.
	// Returns the file content as a string and any error encountered.
	ReadFile(ctx context.Context, fileID string) (string, error)
}
