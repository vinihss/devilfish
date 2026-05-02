package skill

import (
	"context"
	"encoding/json"
	"fmt"

	"devilfish/internal/adapters/mcp"
	"devilfish/internal/infra/logging"
	"devilfish/internal/ports/outbound"
)

// Drive skill server name constant
const driveServerName = "google-drive"

// ListFilesSkill lists files from Google Drive.
// It implements the Skill interface and provides a granular abstraction
// for the list_files capability.
type ListFilesSkill struct {
	registry *mcp.Registry
	logger   *logging.ZLogger
}

// NewListFilesSkill creates a new ListFilesSkill.
// The registry is used to resolve the Google Drive MCP server.
func NewListFilesSkill(registry *mcp.Registry) *ListFilesSkill {
	return &ListFilesSkill{
		registry: registry,
		logger:   logging.NewLogger(nil).Named("skill.list_files"),
	}
}

// Name returns the unique identifier for this skill.
func (s *ListFilesSkill) Name() string {
	return "list_files"
}

// Description returns a human-readable description of what the skill does.
func (s *ListFilesSkill) Description() string {
	return "List files in Google Drive. Supports Google Drive search syntax (e.g., \"name contains 'report'\", \"mimeType='application/pdf'\")."
}

// Schema returns the JSON schema for the skill's input parameters.
// This schema is used by the LLM to understand what arguments are required.
func (s *ListFilesSkill) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Google Drive search query (e.g., \"name contains 'report'\", \"mimeType='application/pdf'\")",
			},
		},
		"required": []string{"query"},
	}
}

// Execute lists files matching the query via the Google Drive MCP server.
// Required arguments: query (string) - Google Drive search syntax.
func (s *ListFilesSkill) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	s.logger.Info("executing list_files skill")

	// 1. Validate input
	query, ok := args["query"].(string)
	if !ok || query == "" {
		// Use default query if not provided
		query = ""
		s.logger.Info("no query provided, listing all files")
	}

	// 2. Get Google Drive MCP client from registry
	client, found := s.registry.Get(driveServerName)
	if !found {
		return "", fmt.Errorf("google-drive MCP server not found in registry")
	}

	// 3. Assert DriveCapability
	driveCap, ok := client.(outbound.DriveCapability)
	if !ok {
		return "", fmt.Errorf("MCP client does not implement DriveCapability")
	}

	// 4. Call capability.ListFiles()
	s.logger.Infof("listing files with query: %s", query)
	files, err := driveCap.ListFiles(ctx, query)
	if err != nil {
		s.logger.Errorf("failed to list files: %v", err)
		return "", fmt.Errorf("failed to list files: %w", err)
	}

	// 5. Return structured output
	if len(files) == 0 {
		return "No files found matching the query.", nil
	}

	result := fmt.Sprintf("Found %d file(s):\n", len(files))
	for i, file := range files {
		if i >= 10 {
			result += fmt.Sprintf("... and %d more file(s)\n", len(files)-10)
			break
		}
		result += fmt.Sprintf("- ID: %s | Name: %s | Type: %s\n", file.ID, file.Name, file.MIMEType)
	}

	s.logger.Infof("listed %d files", len(files))
	return result, nil
}

// ReadFileSkill reads a file from Google Drive.
// It implements the Skill interface and provides a granular abstraction
// for the read_file capability.
type ReadFileSkill struct {
	registry *mcp.Registry
	logger   *logging.ZLogger
}

// NewReadFileSkill creates a new ReadFileSkill.
func NewReadFileSkill(registry *mcp.Registry) *ReadFileSkill {
	return &ReadFileSkill{
		registry: registry,
		logger:   logging.NewLogger(nil).Named("skill.read_file"),
	}
}

// Name returns the unique identifier for this skill.
func (s *ReadFileSkill) Name() string {
	return "read_file"
}

// Description returns a human-readable description of what the skill does.
func (s *ReadFileSkill) Description() string {
	return "Read the content of a file from Google Drive by its ID. Returns the file content as text."
}

// Schema returns the JSON schema for the skill's input parameters.
func (s *ReadFileSkill) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_id": map[string]interface{}{
				"type":        "string",
				"description": "Google Drive file ID to read",
			},
		},
		"required": []string{"file_id"},
	}
}

// Execute reads a file by its ID via the Google Drive MCP server.
// Required arguments: file_id (string) - the file ID.
func (s *ReadFileSkill) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	s.logger.Info("executing read_file skill")

	// 1. Validate input
	fileID, ok := args["file_id"].(string)
	if !ok || fileID == "" {
		return "", fmt.Errorf("invalid or missing 'file_id' parameter: must be a non-empty string")
	}

	// 2. Get Google Drive MCP client from registry
	client, found := s.registry.Get(driveServerName)
	if !found {
		return "", fmt.Errorf("google-drive MCP server not found in registry")
	}

	// 3. Assert DriveCapability
	driveCap, ok := client.(outbound.DriveCapability)
	if !ok {
		return "", fmt.Errorf("MCP client does not implement DriveCapability")
	}

	// 4. Call capability.ReadFile()
	s.logger.Infof("reading file with ID: %s", fileID)
	content, err := driveCap.ReadFile(ctx, fileID)
	if err != nil {
		s.logger.Errorf("failed to read file: %v", err)
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// 5. Return structured output
	// Truncate content if too long
	const maxLength = 4000
	if len(content) > maxLength {
		truncated := content[:maxLength]
		result := fmt.Sprintf("File Content (truncated to %d characters):\n\n%s\n\n... (content truncated, total length: %d characters)",
			maxLength, truncated, len(content))

		s.logger.Infof("file read successfully (truncated): %s", fileID)
		return result, nil
	}

	result := fmt.Sprintf("File Content:\n\n%s", content)
	s.logger.Infof("file read successfully: %s", fileID)
	return result, nil
}

// GetFileInfoSkill retrieves metadata about a file from Google Drive.
// This is a bonus skill that provides file metadata without reading content.
type GetFileInfoSkill struct {
	registry *mcp.Registry
	logger   *logging.ZLogger
}

// NewGetFileInfoSkill creates a new GetFileInfoSkill.
func NewGetFileInfoSkill(registry *mcp.Registry) *GetFileInfoSkill {
	return &GetFileInfoSkill{
		registry: registry,
		logger:   logging.NewLogger(nil).Named("skill.get_file_info"),
	}
}

// Name returns the unique identifier for this skill.
func (s *GetFileInfoSkill) Name() string {
	return "get_file_info"
}

// Description returns a human-readable description of what the skill does.
func (s *GetFileInfoSkill) Description() string {
	return "Get metadata about a file in Google Drive (name, type, size, modified time) without reading its content."
}

// Schema returns the JSON schema for the skill's input parameters.
func (s *GetFileInfoSkill) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_id": map[string]interface{}{
				"type":        "string",
				"description": "Google Drive file ID",
			},
		},
		"required": []string{"file_id"},
	}
}

// Execute gets file metadata via the Google Drive MCP server.
// This uses ListFiles with a query to find the specific file by ID.
func (s *GetFileInfoSkill) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	s.logger.Info("executing get_file_info skill")

	// 1. Validate input
	fileID, ok := args["file_id"].(string)
	if !ok || fileID == "" {
		return "", fmt.Errorf("invalid or missing 'file_id' parameter: must be a non-empty string")
	}

	// 2. Get Google Drive MCP client from registry
	client, found := s.registry.Get(driveServerName)
	if !found {
		return "", fmt.Errorf("google-drive MCP server not found in registry")
	}

	// 3. Assert DriveCapability
	driveCap, ok := client.(outbound.DriveCapability)
	if !ok {
		return "", fmt.Errorf("MCP client does not implement DriveCapability")
	}

	// 4. Use ListFiles with a query to find the specific file
	// Google Drive API query syntax: "id='fileId'"
	query := fmt.Sprintf("id='%s'", fileID)
	s.logger.Infof("getting file info with query: %s", query)

	files, err := driveCap.ListFiles(ctx, query)
	if err != nil {
		s.logger.Errorf("failed to get file info: %v", err)
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("file not found with ID: %s", fileID)
	}

	file := files[0]

	// 5. Return structured output as JSON for easy parsing
	fileInfo := map[string]interface{}{
		"id":            file.ID,
		"name":          file.Name,
		"mime_type":     file.MIMEType,
		"size":          file.Size,
		"modified_time":  file.ModifiedTime,
		"created_time":   file.CreatedTime,
		"parents":        file.Parents,
		"web_view_link":  file.WebViewLink,
	}

	jsonBytes, err := json.MarshalIndent(fileInfo, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal file info: %w", err)
	}

	result := fmt.Sprintf("File Information:\n%s", string(jsonBytes))
	s.logger.Infof("file info retrieved successfully: %s", fileID)
	return result, nil
}

// Ensure skill types implement the Skill interface
var _ Skill = (*ListFilesSkill)(nil)
var _ Skill = (*ReadFileSkill)(nil)
var _ Skill = (*GetFileInfoSkill)(nil)
