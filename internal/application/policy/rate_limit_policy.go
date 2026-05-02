package policy

import (
	"devilfish/internal/application/skill"
	"fmt"
	"sync"
	"time"
)

// RateLimitPolicy prevents tool call spam by limiting the number of calls per tool within a time window.
type RateLimitPolicy struct {
	mu       sync.Mutex
	calls    map[string][]time.Time // tool name -> list of call timestamps
	maxCalls int
	window   time.Duration
}

// NewRateLimitPolicy creates a new RateLimitPolicy.
// maxCalls: maximum number of calls allowed per tool within the time window.
// window: the time window for rate limiting (e.g., 1*time.Minute).
func NewRateLimitPolicy(maxCalls int, window time.Duration) *RateLimitPolicy {
	return &RateLimitPolicy{
		calls:    make(map[string][]time.Time),
		maxCalls: maxCalls,
		window:   window,
	}
}

// Allow checks if a tool call is allowed based on rate limits.
// It records the call timestamp and cleans up old entries.
func (p *RateLimitPolicy) Allow(call skill.ToolCall) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	toolName := call.Name

	// Get existing calls for this tool
	toolCalls, exists := p.calls[toolName]
	if !exists {
		toolCalls = make([]time.Time, 0)
	}

	// Remove calls outside the time window
	cutoff := now.Add(-p.window)
	validCalls := make([]time.Time, 0, len(toolCalls))
	for _, t := range toolCalls {
		if t.After(cutoff) {
			validCalls = append(validCalls, t)
		}
	}

	// Check if rate limit is exceeded
	if len(validCalls) >= p.maxCalls {
		return fmt.Errorf("rate limit exceeded for tool %q: %d calls allowed per %v (window: %v)",
			toolName, p.maxCalls, p.window, p.window)
	}

	// Record this call
	validCalls = append(validCalls, now)
	p.calls[toolName] = validCalls

	return nil
}

// GetCallCount returns the number of recent calls for a tool (within the time window).
// This is useful for testing and monitoring.
func (p *RateLimitPolicy) GetCallCount(toolName string) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-p.window)

	toolCalls, exists := p.calls[toolName]
	if !exists {
		return 0
	}

	count := 0
	for _, t := range toolCalls {
		if t.After(cutoff) {
			count++
		}
	}
	return count
}

// Reset clears all recorded calls for all tools.
func (p *RateLimitPolicy) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = make(map[string][]time.Time)
}
