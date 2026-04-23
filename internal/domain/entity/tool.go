package entity

import (
	"time"
)

// Tool represents a tool exposed by an MCP server.
type Tool struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	ServerName  string                 `json:"server_name"` // Which server provides this tool
	InputSchema map[string]interface{} `json:"input_schema"`
	OutputSchema map[string]interface{} `json:"output_schema"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

// NewTool creates a new Tool entity.
func NewTool(id, name, description, serverName string) *Tool {
	return &Tool{
		ID:          id,
		Name:        name,
		Description: description,
		ServerName:  serverName,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// WithInputSchema sets the input schema for the tool.
func (t *Tool) WithInputSchema(schema map[string]interface{}) *Tool {
	t.InputSchema = schema
	return t
}

// WithOutputSchema sets the output schema for the tool.
func (t *Tool) WithOutputSchema(schema map[string]interface{}) *Tool {
	t.OutputSchema = schema
	return t
}

// ValidateInput validates input parameters against the tool's input schema.
func (t *Tool) ValidateInput(params map[string]interface{}) error {
	// Basic validation - check required fields from input schema
	if t.InputSchema == nil {
		return nil
	}

	required, ok := t.InputSchema["required"].([]interface{})
	if !ok {
		return nil
	}

	for _, field := range required {
		fieldName, ok := field.(string)
		if !ok {
			continue
		}
		if _, exists := params[fieldName]; !exists {
			return &ValidationError{
				Field:   fieldName,
				Message: "required field is missing",
			}
		}
	}
	return nil
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}