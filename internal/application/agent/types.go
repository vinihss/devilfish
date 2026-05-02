package agent

import (
	"time"

	"devilfish/internal/application/skill"
)

// Step represents a single step in the agent's execution
type Step struct {
	Thought  string          `json:"thought"`
	ToolCall *skill.ToolCall `json:"tool_call,omitempty"`
	Result   string          `json:"result,omitempty"`
	Error    error           `json:"error,omitempty"`
	Duration time.Duration   `json:"duration"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant", "tool"
	Content string `json:"content"`
}

// Config holds agent configuration
type Config struct {
	MaxIterations int
	SystemPrompt  string
	Model         string
	Temperature   float32
}
