package agent

import (
	"fmt"

	"devilfish/internal/skills"
)

// Message represents a single entry in the agent conversation context.
type Message struct {
	// Role is one of "user", "assistant", or "tool".
	Role    string
	Content string
}

// LLMResponse is the response produced by a single LLM call.
//
// If ToolCall is non-nil the agent must execute the referenced skill and
// continue the loop. If ToolCall is nil, Content is the final answer.
type LLMResponse struct {
	Content  string
	ToolCall *skills.ToolCall
}

// LLM is the interface for any language model that participates in the agent
// loop. The agent does not know how the LLM works internally; it only
// communicates via messages.
type LLM interface {
	Generate(messages []Message) (LLMResponse, error)
}

// ErrMaxIterationsReached is returned when the loop exhausts its iteration
// budget before the LLM produces a final answer.
var ErrMaxIterationsReached = fmt.Errorf("agent: maximum iterations reached without a final answer")

// RunAgent drives the agent execution loop.
//
// The loop:
//  1. Calls the LLM with the current message context.
//  2. If the response contains a ToolCall, the corresponding skill is executed,
//     the result is appended as a "tool" message, and the loop continues.
//  3. If the response contains no ToolCall, the content is returned as the
//     final answer.
//
// RunAgent returns ErrMaxIterationsReached when maxIterations is exhausted
// before a final answer is produced.
func RunAgent(
	llm LLM,
	context []Message,
	registry *skills.SkillRegistry,
	maxIterations int,
) (string, error) {
	if llm == nil {
		return "", fmt.Errorf("agent: LLM is required")
	}
	if registry == nil {
		return "", fmt.Errorf("agent: skill registry is required")
	}
	if maxIterations <= 0 {
		return "", fmt.Errorf("agent: maxIterations must be greater than zero")
	}

	msgs := make([]Message, len(context))
	copy(msgs, context)

	for i := 0; i < maxIterations; i++ {
		resp, err := llm.Generate(msgs)
		if err != nil {
			return "", fmt.Errorf("agent: LLM error on iteration %d: %w", i+1, err)
		}

		if resp.ToolCall == nil {
			return resp.Content, nil
		}

		result, err := skills.ExecuteSkill(*resp.ToolCall, registry)
		if err != nil {
			return "", fmt.Errorf("agent: tool execution error on iteration %d: %w", i+1, err)
		}

		msgs = append(msgs, Message{
			Role:    "tool",
			Content: result,
		})
	}

	return "", ErrMaxIterationsReached
}
