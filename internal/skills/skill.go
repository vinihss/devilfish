package skills

// Skill defines the execution contract for a named capability.
// Each skill is self-contained and has no knowledge of the agent loop.
type Skill interface {
	// Name returns the unique identifier used to look up the skill.
	Name() string
	// Description returns a human-readable summary of what the skill does.
	Description() string
	// Execute runs the skill with the given input parameters and returns a
	// plain-text result or an error.
	Execute(input map[string]interface{}) (string, error)
}

// ToolCall represents a tool invocation requested by the LLM.
type ToolCall struct {
	Name      string
	Arguments map[string]interface{}
}
