package policy

import (
	"fmt"
	"testing"
	"time"

	"devilfish/internal/application/skill"
)

// Helper function to create a ToolCall
func makeToolCall(name string, params map[string]interface{}) skill.ToolCall {
	if params == nil {
		params = make(map[string]interface{})
	}
	return skill.ToolCall{
		Name:       name,
		Parameters: params,
	}
}

// Test PolicySet

func TestPolicySet_Allow_AllPoliciesPass(t *testing.T) {
	ps := NewPolicySet()

	// Add a policy that always allows
	ps.Add(&mockPolicy{err: nil})

	// Add another policy that always allows
	ps.Add(&mockPolicy{err: nil})

	call := makeToolCall("test_tool", nil)
	if err := ps.Allow(call); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestPolicySet_Allow_FirstPolicyDenies(t *testing.T) {
	ps := NewPolicySet()

	// Add a policy that denies
	ps.Add(&mockPolicy{err: fmt.Errorf("denied by policy 1")})

	// This should not be reached
	ps.Add(&mockPolicy{err: nil})

	call := makeToolCall("test_tool", nil)
	err := ps.Allow(call)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	expected := "policy denied: denied by policy 1"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestPolicySet_Allow_SecondPolicyDenies(t *testing.T) {
	ps := NewPolicySet()

	// First policy allows
	ps.Add(&mockPolicy{err: nil})

	// Second policy denies
	ps.Add(&mockPolicy{err: fmt.Errorf("denied by policy 2")})

	call := makeToolCall("test_tool", nil)
	err := ps.Allow(call)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	expected := "policy denied: denied by policy 2"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestPolicySet_Allow_EmptySet(t *testing.T) {
	ps := NewPolicySet()

	call := makeToolCall("test_tool", nil)
	if err := ps.Allow(call); err != nil {
		t.Errorf("Expected no error for empty policy set, got %v", err)
	}
}

// Test EmailPolicy

func TestEmailPolicy_Allow_NonEmailTool(t *testing.T) {
	policy := NewEmailPolicy()

	call := makeToolCall("file_write", nil)
	if err := policy.Allow(call); err != nil {
		t.Errorf("Expected no error for non-email tool, got %v", err)
	}
}

func TestEmailPolicy_Allow_ConfirmationRequired_NoConfirmation(t *testing.T) {
	policy := NewEmailPolicy()
	policy.RequireConfirmation = true

	call := makeToolCall("send_email", map[string]interface{}{
		"to": "test@example.com",
	})

	err := policy.Allow(call)
	if err == nil {
		t.Error("Expected error when confirmation is required but not provided")
	}
}

func TestEmailPolicy_Allow_ConfirmationRequired_WithConfirmation(t *testing.T) {
	policy := NewEmailPolicy()
	policy.RequireConfirmation = true

	call := makeToolCall("send_email", map[string]interface{}{
		"to":        "test@example.com",
		"confirmed": true,
	})

	if err := policy.Allow(call); err != nil {
		t.Errorf("Expected no error with confirmation, got %v", err)
	}
}

func TestEmailPolicy_Allow_BlockedRecipient(t *testing.T) {
	policy := NewEmailPolicy()
	policy.BlockedRecipients = []string{"spam@example.com", "blocked@test.com"}

	call := makeToolCall("send_email", map[string]interface{}{
		"to": "spam@example.com",
	})

	err := policy.Allow(call)
	if err == nil {
		t.Error("Expected error for blocked recipient")
	}
}

func TestEmailPolicy_Allow_BlockedRecipient_CaseInsensitive(t *testing.T) {
	policy := NewEmailPolicy()
	policy.BlockedRecipients = []string{"spam@example.com"}

	call := makeToolCall("send_email", map[string]interface{}{
		"to": "SPAM@EXAMPLE.COM",
	})

	err := policy.Allow(call)
	if err == nil {
		t.Error("Expected error for blocked recipient (case insensitive)")
	}
}

func TestEmailPolicy_Allow_AllowedDomains(t *testing.T) {
	policy := NewEmailPolicy()
	policy.AllowedDomains = []string{"example.com", "test.org"}

	// Should allow
	call1 := makeToolCall("send_email", map[string]interface{}{
		"to": "user@example.com",
	})
	if err := policy.Allow(call1); err != nil {
		t.Errorf("Expected no error for allowed domain, got %v", err)
	}

	// Should deny
	call2 := makeToolCall("send_email", map[string]interface{}{
		"to": "user@evil.com",
	})
	if err := policy.Allow(call2); err == nil {
		t.Error("Expected error for non-allowed domain")
	}
}

func TestEmailPolicy_Allow_MissingRecipient(t *testing.T) {
	policy := NewEmailPolicy()

	call := makeToolCall("send_email", nil)
	err := policy.Allow(call)
	if err == nil {
		t.Error("Expected error for missing recipient")
	}
}

func TestEmailPolicy_Allow_InvalidRecipientType(t *testing.T) {
	policy := NewEmailPolicy()

	call := makeToolCall("send_email", map[string]interface{}{
		"to": 123, // Not a string
	})
	err := policy.Allow(call)
	if err == nil {
		t.Error("Expected error for invalid recipient type")
	}
}

// Test RateLimitPolicy

func TestRateLimitPolicy_Allow_WithinLimit(t *testing.T) {
	policy := NewRateLimitPolicy(3, 1*time.Minute)

	// First call - should allow
	call := makeToolCall("test_tool", nil)
	if err := policy.Allow(call); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Second call - should allow
	if err := policy.Allow(call); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Third call - should allow
	if err := policy.Allow(call); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check call count
	if count := policy.GetCallCount("test_tool"); count != 3 {
		t.Errorf("Expected 3 calls, got %d", count)
	}
}

func TestRateLimitPolicy_Allow_ExceedsLimit(t *testing.T) {
	policy := NewRateLimitPolicy(2, 1*time.Minute)

	call := makeToolCall("test_tool", nil)

	// First call - should allow
	if err := policy.Allow(call); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Second call - should allow
	if err := policy.Allow(call); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Third call - should deny
	err := policy.Allow(call)
	if err == nil {
		t.Error("Expected error for exceeding rate limit")
	}
}

func TestRateLimitPolicy_Allow_WindowExpiry(t *testing.T) {
	policy := NewRateLimitPolicy(2, 50*time.Millisecond)

	call := makeToolCall("test_tool", nil)

	// First call
	if err := policy.Allow(call); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Second call
	if err := policy.Allow(call); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Wait for window to expire
	time.Sleep(100 * time.Millisecond)

	// Should allow again after window expires
	if err := policy.Allow(call); err != nil {
		t.Errorf("Expected no error after window expiry, got %v", err)
	}
}

func TestRateLimitPolicy_Allow_DifferentTools(t *testing.T) {
	policy := NewRateLimitPolicy(2, 1*time.Minute)

	call1 := makeToolCall("tool_a", nil)
	call2 := makeToolCall("tool_b", nil)

	// Both tools should have separate limits
	if err := policy.Allow(call1); err != nil {
		t.Errorf("Expected no error for tool_a, got %v", err)
	}
	if err := policy.Allow(call1); err != nil {
		t.Errorf("Expected no error for tool_a, got %v", err)
	}
	// tool_a now at limit
	if err := policy.Allow(call1); err == nil {
		t.Error("Expected error for tool_a exceeding limit")
	}

	// tool_b should still be allowed
	if err := policy.Allow(call2); err != nil {
		t.Errorf("Expected no error for tool_b, got %v", err)
	}
}

func TestRateLimitPolicy_Reset(t *testing.T) {
	policy := NewRateLimitPolicy(1, 1*time.Minute)

	call := makeToolCall("test_tool", nil)

	// First call
	if err := policy.Allow(call); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Reset
	policy.Reset()

	// Should allow again after reset
	if err := policy.Allow(call); err != nil {
		t.Errorf("Expected no error after reset, got %v", err)
	}
}

// Test AllowedToolsPolicy

func TestAllowedToolsPolicy_Allow_ToolInList(t *testing.T) {
	policy := NewAllowedToolsPolicy([]string{"tool_a", "tool_b", "tool_c"})

	call := makeToolCall("tool_b", nil)
	if err := policy.Allow(call); err != nil {
		t.Errorf("Expected no error for allowed tool, got %v", err)
	}
}

func TestAllowedToolsPolicy_Allow_ToolNotInList(t *testing.T) {
	policy := NewAllowedToolsPolicy([]string{"tool_a", "tool_b"})

	call := makeToolCall("tool_c", nil)
	err := policy.Allow(call)
	if err == nil {
		t.Error("Expected error for non-allowed tool")
	}
}

func TestAllowedToolsPolicy_Allow_EmptyList(t *testing.T) {
	policy := NewAllowedToolsPolicy([]string{})

	call := makeToolCall("any_tool", nil)
	err := policy.Allow(call)
	if err == nil {
		t.Error("Expected error for empty allowed list")
	}
}

func TestAllowedToolsPolicy_IsAllowed(t *testing.T) {
	policy := NewAllowedToolsPolicy([]string{"tool_a", "tool_b"})

	if !policy.IsAllowed("tool_a") {
		t.Error("Expected tool_a to be allowed")
	}
	if policy.IsAllowed("tool_c") {
		t.Error("Expected tool_c to not be allowed")
	}
}

// Test BlockedToolsPolicy

func TestBlockedToolsPolicy_Allow_ToolNotInList(t *testing.T) {
	policy := NewBlockedToolsPolicy([]string{"tool_a", "tool_b"})

	call := makeToolCall("tool_c", nil)
	if err := policy.Allow(call); err != nil {
		t.Errorf("Expected no error for non-blocked tool, got %v", err)
	}
}

func TestBlockedToolsPolicy_Allow_ToolInList(t *testing.T) {
	policy := NewBlockedToolsPolicy([]string{"tool_a", "tool_b"})

	call := makeToolCall("tool_a", nil)
	err := policy.Allow(call)
	if err == nil {
		t.Error("Expected error for blocked tool")
	}
}

func TestBlockedToolsPolicy_Allow_EmptyList(t *testing.T) {
	policy := NewBlockedToolsPolicy([]string{})

	call := makeToolCall("any_tool", nil)
	if err := policy.Allow(call); err != nil {
		t.Errorf("Expected no error for empty blocked list, got %v", err)
	}
}

func TestBlockedToolsPolicy_IsBlocked(t *testing.T) {
	policy := NewBlockedToolsPolicy([]string{"tool_a", "tool_b"})

	if !policy.IsBlocked("tool_a") {
		t.Error("Expected tool_a to be blocked")
	}
	if policy.IsBlocked("tool_c") {
		t.Error("Expected tool_c to not be blocked")
	}
}

// Test combined policies in PolicySet

func TestPolicySet_CombinedPolicies(t *testing.T) {
	ps := NewPolicySet()

	// Allow only specific tools
	ps.Add(NewAllowedToolsPolicy([]string{"send_email", "file_write"}))

	// Rate limit
	ps.Add(NewRateLimitPolicy(5, 1*time.Minute))

	// Block specific tools
	ps.Add(NewBlockedToolsPolicy([]string{"file_delete"}))

	// Should allow
	call1 := makeToolCall("send_email", nil)
	if err := ps.Allow(call1); err != nil {
		t.Errorf("Expected no error for allowed tool, got %v", err)
	}

	// Should deny - not in allowed list
	call2 := makeToolCall("file_delete", nil)
	if err := ps.Allow(call2); err == nil {
		t.Error("Expected error for tool not in allowed list")
	}
}

// Mock policy for testing
type mockPolicy struct {
	err error
}

func (m *mockPolicy) Allow(call skill.ToolCall) error {
	return m.err
}
