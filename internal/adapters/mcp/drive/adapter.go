package drive

import (
	"context"
	"fmt"
	"sync"

	"devilfish/internal/infra/logging"
	"devilfish/internal/ports/outbound"
)

const (
	// tool names expected to be provided by the Google Drive MCP server
	toolListFiles = "list_files"
	toolReadFile  = "read_file"

	// serverName is the fixed name for this adapter
	serverName = "google-drive"
)

// DriveMCP is a concrete MCP adapter that provides Google Drive operations via an
// MCP server. It implements both outbound.MCPClient (so it can be registered in
// the Registry) and outbound.DriveCapability (so callers can use it directly
// through the typed interface).
//
// The adapter wraps an existing outbound.MCPClient that connects to a Google
// Drive MCP server, delegating tool execution to that client.
type DriveMCP struct {
	config    outbound.MCPServerConfig
	client    outbound.MCPClient
	connected bool
	mu        sync.RWMutex
	logger    *logging.ZLogger
}

// NewDriveMCP creates a new Drive MCP adapter.
// The config should specify the MCP server that provides Drive functionality.
// The actual MCPClient is set via the Connect method or SetClient.
func NewDriveMCP(config outbound.MCPServerConfig) *DriveMCP {
	return &DriveMCP{
		config: config,
		logger: logging.Named("drive-mcp"),
	}
}

// SetClient sets the underlying MCP client (useful for testing or when the
// client is created externally).
func (d *DriveMCP) SetClient(client outbound.MCPClient) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.client = client
	if client != nil && client.IsConnected() {
		d.connected = true
	}
}

// --- outbound.MCPClient implementation ---

// Connect establishes a connection to the MCP server.
// If d.client is already set, it delegates to that client's Connect method.
// Otherwise, it marks the adapter as connected (the actual transport should
// be set up before or after this call).
func (d *DriveMCP) Connect(ctx context.Context, config outbound.MCPServerConfig) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.logger.Infof("connecting to Google Drive MCP server %s", config.Name)

	if d.connected {
		d.logger.Infof("already connected to Google Drive MCP server %s", config.Name)
		return nil
	}

	d.config = config

	// If we have a client, delegate to it
	if d.client != nil {
		if err := d.client.Connect(ctx, config); err != nil {
			d.logger.Errorf("failed to connect to Google Drive MCP server: %v", err)
			return fmt.Errorf("failed to connect to Google Drive MCP server %s: %w", config.Name, err)
		}
	}

	d.connected = true

	d.logger.Infof("successfully connected to Google Drive MCP server %s", config.Name)
	return nil
}

// Disconnect closes the connection to the MCP server.
func (d *DriveMCP) Disconnect(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.logger.Infof("disconnecting from Google Drive MCP server %s", d.config.Name)

	if !d.connected {
		d.logger.Infof("not connected to Google Drive MCP server %s", d.config.Name)
		return nil
	}

	// If we have a client, delegate to it
	if d.client != nil {
		if err := d.client.Disconnect(ctx); err != nil {
			d.logger.Errorf("error disconnecting from Google Drive MCP server: %v", err)
			// Continue with cleanup even if disconnect has errors
		}
	}

	d.connected = false

	d.logger.Infof("successfully disconnected from Google Drive MCP server %s", d.config.Name)
	return nil
}

// IsConnected returns whether the adapter is connected to the MCP server.
func (d *DriveMCP) IsConnected() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.connected {
		return false
	}
	if d.client != nil {
		return d.client.IsConnected()
	}
	return d.connected
}

// ListTools returns the list of tools provided by the Drive MCP server.
func (d *DriveMCP) ListTools(ctx context.Context) ([]outbound.MCPTool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if !d.connected {
		return nil, fmt.Errorf("not connected to Google Drive MCP server %s", d.config.Name)
	}

	if d.client == nil {
		return nil, fmt.Errorf("no MCP client available for Google Drive server %s", d.config.Name)
	}

	d.logger.Debugf("listing tools from Google Drive MCP server %s", d.config.Name)

	tools, err := d.client.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools from Google Drive MCP server: %w", err)
	}

	return tools, nil
}

// ExecuteTool executes a tool on the Drive MCP server.
func (d *DriveMCP) ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if !d.connected {
		return nil, fmt.Errorf("not connected to Google Drive MCP server %s", d.config.Name)
	}

	if d.client == nil {
		return nil, fmt.Errorf("no MCP client available for Google Drive server %s", d.config.Name)
	}

	d.logger.Debugf("executing tool %s on Google Drive MCP server %s", toolName, d.config.Name)

	result, err := d.client.ExecuteTool(ctx, toolName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute tool %s on Google Drive MCP server: %w", toolName, err)
	}

	return result, nil
}

// ServerName returns the server name.
func (d *DriveMCP) ServerName() string {
	return serverName
}

// Name returns the provider name (alias for ServerName).
func (d *DriveMCP) Name() string {
	return serverName
}

// --- outbound.DriveCapability implementation ---

// ListFiles lists files matching the given query.
// The query parameter follows Google Drive API query syntax.
func (d *DriveMCP) ListFiles(ctx context.Context, query string) ([]outbound.File, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	d.logger.Infof("listing Drive files with query: %s", query)

	if !d.connected {
		return nil, fmt.Errorf("not connected to Google Drive MCP server %s", d.config.Name)
	}

	if d.client == nil {
		return nil, fmt.Errorf("no MCP client available for Google Drive server %s", d.config.Name)
	}

	// Prepare parameters for the list_files tool
	params := map[string]interface{}{
		"query": query,
	}

	// Execute the list_files tool on the MCP server
	result, err := d.client.ExecuteTool(ctx, toolListFiles, params)
	if err != nil {
		d.logger.Errorf("failed to list files: %v", err)
		return nil, fmt.Errorf("failed to list Drive files: %w", err)
	}

	// Parse the result
	files, err := parseFileList(result)
	if err != nil {
		d.logger.Errorf("failed to parse file list: %v", err)
		return nil, fmt.Errorf("failed to parse Drive file list: %w", err)
	}

	d.logger.Infof("successfully listed %d Drive files", len(files))
	return files, nil
	}

// ReadFile reads the content of a file by its ID.
// Returns the file content as a string.
func (d *DriveMCP) ReadFile(ctx context.Context, fileID string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	d.logger.Infof("reading Drive file with ID: %s", fileID)

	if !d.connected {
		return "", fmt.Errorf("not connected to Google Drive MCP server %s", d.config.Name)
	}

	if d.client == nil {
		return "", fmt.Errorf("no MCP client available for Google Drive server %s", d.config.Name)
	}

	if fileID == "" {
		return "", fmt.Errorf("file ID must not be empty")
	}

	// Prepare parameters for the read_file tool
	params := map[string]interface{}{
		"file_id": fileID,
	}

	// Execute the read_file tool on the MCP server
	result, err := d.client.ExecuteTool(ctx, toolReadFile, params)
	if err != nil {
		d.logger.Errorf("failed to read file %s: %v", fileID, err)
		return "", fmt.Errorf("failed to read Drive file %s: %w", fileID, err)
	}

	// Parse the result
	content, err := parseFileContent(result)
	if err != nil {
		d.logger.Errorf("failed to parse file content: %v", err)
		return "", fmt.Errorf("failed to parse Drive file content: %w", err)
	}

	d.logger.Infof("successfully read Drive file %s", fileID)
	return content, nil
}

// --- helper functions ---

// parseFileList parses the result from list_files tool into File slice.
func parseFileList(result map[string]interface{}) ([]outbound.File, error) {
	files := make([]outbound.File, 0)

	// The result should contain a "files" key with an array of file objects
	filesRaw, ok := result["files"]
	if !ok {
		// Try alternative key names
		filesRaw, ok = result["results"]
		if !ok {
			return files, nil // Return empty slice if no files found
		}
	}

	// Convert to slice
	filesSlice, ok := filesRaw.([]interface{})
	if !ok {
		// Invalid format, return empty slice
		return files, nil
	}

	for _, f := range filesSlice {
		fileMap, ok := f.(map[string]interface{})
		if !ok {
			continue
		}

		file := outbound.File{}

		if id, ok := fileMap["id"].(string); ok {
			file.ID = id
		}

		if name, ok := fileMap["name"].(string); ok {
			file.Name = name
		}

		files = append(files, file)
	}

	return files, nil
}

// parseFileContent parses the result from read_file tool into a string.
func parseFileContent(result map[string]interface{}) (string, error) {
	// Try different possible keys for content
	contentKeys := []string{"content", "data", "text", "file_content"}

	for _, key := range contentKeys {
		if val, ok := result[key]; ok {
			switch v := val.(type) {
			case string:
				return v, nil
			case []byte:
				return string(v), nil
			default:
				return fmt.Sprintf("%v", v), nil
			}
		}
	}

	return "", fmt.Errorf("no content found in result")
}

// Compile-time assertions.
var _ outbound.MCPClient = (*DriveMCP)(nil)
var _ outbound.DriveCapability = (*DriveMCP)(nil)
