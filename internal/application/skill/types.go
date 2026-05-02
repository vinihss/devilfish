package skill

import "context"

// ToolCall represents a tool invocation with its name and parameters
type ToolCall struct {
	// Name is the name of the tool being called (e.g., "send_email", "file_write")
	Name string

	// Parameters contains the tool's input parameters as key-value pairs
	Parameters map[string]interface{}
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	// Output contains the tool's output data
	Output interface{}

	// Error contains any error that occurred during execution
	Error error
}

// ToolInfo represents tool metadata for system prompt generation.
// This is used by the agent package to build system prompts with tool descriptions.
type ToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema,omitempty"`
}

// Skill interface defines the contract for skill implementations.
// Skills provide a higher-level abstraction over MCP tools,
// offering domain-specific operations with schema validation.
type Skill interface {
	// Name returns the unique name of the skill (e.g., "list_files", "send_email")
	Name() string

	// Description returns a human-readable description of what the skill does
	Description() string

	// Schema returns the JSON schema for the skill's input parameters
	Schema() map[string]interface{}

	// Execute runs the skill with the given arguments
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}
