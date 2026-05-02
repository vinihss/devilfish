package policy

import (
	"devilfish/internal/application/skill"
	"fmt"
)

// AllowedToolsPolicy only allows specific tools.
// If the tool is not in the allowed list, the call is denied.
type AllowedToolsPolicy struct {
	// allowedTools maps tool names to true if they are allowed
	allowedTools map[string]bool
}

// NewAllowedToolsPolicy creates a new AllowedToolsPolicy.
// allowed is a list of tool names that are permitted.
// If the list is empty, all tools are denied (whitelist approach).
func NewAllowedToolsPolicy(allowed []string) *AllowedToolsPolicy {
	allowedMap := make(map[string]bool, len(allowed))
	for _, tool := range allowed {
		allowedMap[tool] = true
	}
	return &AllowedToolsPolicy{
		allowedTools: allowedMap,
	}
}

// Allow checks if the tool call is in the allowed list.
func (p *AllowedToolsPolicy) Allow(call skill.ToolCall) error {
	// If no tools are configured, deny all
	if len(p.allowedTools) == 0 {
		return fmt.Errorf("no tools are allowed (empty whitelist)")
	}

	if !p.allowedTools[call.Name] {
		return fmt.Errorf("tool %q is not in the allowed tools list", call.Name)
	}
	return nil
}

// IsAllowed checks if a tool name is in the allowed list.
// This is useful for checking permissions without a full ToolCall.
func (p *AllowedToolsPolicy) IsAllowed(toolName string) bool {
	return p.allowedTools[toolName]
}

// BlockedToolsPolicy blocks specific tools.
// If the tool is in the blocked list, the call is denied.
type BlockedToolsPolicy struct {
	// blockedTools maps tool names to true if they are blocked
	blockedTools map[string]bool
}

// NewBlockedToolsPolicy creates a new BlockedToolsPolicy.
// blocked is a list of tool names that should be blocked.
// If the list is empty, no tools are blocked (blacklist approach).
func NewBlockedToolsPolicy(blocked []string) *BlockedToolsPolicy {
	blockedMap := make(map[string]bool, len(blocked))
	for _, tool := range blocked {
		blockedMap[tool] = true
	}
	return &BlockedToolsPolicy{
		blockedTools: blockedMap,
	}
}

// Allow checks if the tool call is not in the blocked list.
func (p *BlockedToolsPolicy) Allow(call skill.ToolCall) error {
	// If no tools are configured, allow all
	if len(p.blockedTools) == 0 {
		return nil
	}

	if p.blockedTools[call.Name] {
		return fmt.Errorf("tool %q is in the blocked tools list", call.Name)
	}
	return nil
}

// IsBlocked checks if a tool name is in the blocked list.
// This is useful for checking permissions without a full ToolCall.
func (p *BlockedToolsPolicy) IsBlocked(toolName string) bool {
	return p.blockedTools[toolName]
}
