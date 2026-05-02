package policy

import (
	"devilfish/internal/application/skill"
	"fmt"
)

// ToolCall is an alias for skill.ToolCall for convenience
type ToolCall = skill.ToolCall

// ExecutionPolicy interface for validating tool calls before execution.
// Implementations should check if a tool call is safe and allowed.
type ExecutionPolicy interface {
	// Allow checks if a tool call is allowed.
	// Returns nil if the call is allowed, or an error explaining why it's denied.
	Allow(call skill.ToolCall) error
}

// PolicySet is a collection of policies that are all checked.
// A tool call is allowed only if ALL policies allow it.
type PolicySet struct {
	policies []ExecutionPolicy
}

// NewPolicySet creates a new empty PolicySet.
func NewPolicySet() *PolicySet {
	return &PolicySet{
		policies: make([]ExecutionPolicy, 0),
	}
}

// Add adds a policy to the set.
// Policies are checked in the order they were added.
func (ps *PolicySet) Add(policy ExecutionPolicy) {
	ps.policies = append(ps.policies, policy)
}

// Allow checks all policies in the set.
// Returns nil only if ALL policies allow the call.
// Returns the first error encountered if any policy denies the call.
func (ps *PolicySet) Allow(call skill.ToolCall) error {
	for _, policy := range ps.policies {
		if err := policy.Allow(call); err != nil {
			return fmt.Errorf("policy denied: %w", err)
		}
	}
	return nil
}
