package skill

import (
	"context"
	"fmt"
	"sync"
)

// Registry manages all available skills in the system.
// It provides thread-safe access to skills and their schemas.
// The registry is used by the agent (LLM) to discover and execute skills.
type Registry struct {
	skills map[string]Skill
	mu     sync.RWMutex
}

// NewRegistry creates a new skill registry.
func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]Skill),
	}
}

// Register adds a skill to the registry.
// If a skill with the same name already exists, it will be overwritten.
// The skill name is used as the unique identifier for lookup.
func (r *Registry) Register(skill Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.skills[skill.Name()] = skill
}

// Unregister removes a skill from the registry by name.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.skills, name)
}

// Get retrieves a skill by name.
// Returns the skill and true if found, or nil and false if not found.
func (r *Registry) Get(name string) (Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skill, found := r.skills[name]
	return skill, found
}

// List returns all registered skills.
// Returns a slice of skills for iteration.
func (r *Registry) List() []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]Skill, 0, len(r.skills))
	for _, skill := range r.skills {
		skills = append(skills, skill)
	}
	return skills
}

// GetSchemas returns the JSON schemas for all registered skills.
// This is used by the LLM to understand what skills are available
// and what parameters they expect.
// Returns a slice of schema maps.
func (r *Registry) GetSchemas() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemas := make([]map[string]interface{}, 0, len(r.skills))
	for _, skill := range r.skills {
		schema := map[string]interface{}{
			"name":        skill.Name(),
			"description": skill.Description(),
			"inputSchema": skill.Schema(),
		}
		schemas = append(schemas, schema)
	}
	return schemas
}

// ExecuteSkill executes a skill by name with the given arguments.
// This is a convenience method that combines Get and Execute.
// Returns the result string and any error that occurred.
func (r *Registry) ExecuteSkill(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	skill, found := r.Get(name)
	if !found {
		return "", fmt.Errorf("skill not found: %s", name)
	}

	return skill.Execute(ctx, args)
}

// Count returns the number of registered skills.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.skills)
}

// Has checks if a skill with the given name is registered.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, found := r.skills[name]
	return found
}

// Clear removes all skills from the registry.
// This is primarily used for testing.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.skills = make(map[string]Skill)
}

// Execute executes a skill by name with given parameters (using ToolCall format).
// This provides a compatible interface for the agent package.
func (r *Registry) Execute(ctx context.Context, call ToolCall) (*ToolResult, error) {
	skill, exists := r.Get(call.Name)
	if !exists {
		return nil, fmt.Errorf("skill %q not found in registry", call.Name)
	}

	output, err := skill.Execute(ctx, call.Parameters)
	if err != nil {
		return &ToolResult{
			Output: nil,
			Error:  err,
		}, nil
	}

	return &ToolResult{
		Output: output,
		Error:  nil,
	}, nil
}
