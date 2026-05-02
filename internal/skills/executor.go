package skills

import "fmt"

// ErrSkillNotFound is returned when a requested skill is not in the registry.
var ErrSkillNotFound = fmt.Errorf("skill not found")

// ExecuteSkill resolves the skill identified by call.Name from the registry
// and executes it with call.Arguments.
//
// It returns an error wrapping ErrSkillNotFound when the skill is absent, or
// the underlying skill error on execution failure.
func ExecuteSkill(call ToolCall, registry *SkillRegistry) (string, error) {
	if registry == nil {
		return "", fmt.Errorf("skill registry is required")
	}

	skill, ok := registry.Get(call.Name)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrSkillNotFound, call.Name)
	}

	result, err := skill.Execute(call.Arguments)
	if err != nil {
		return "", fmt.Errorf("skill %q execution failed: %w", call.Name, err)
	}

	return result, nil
}
