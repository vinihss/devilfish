package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"devilfish/internal/application/policy"
	"devilfish/internal/application/skill"
	"devilfish/internal/infra/logging"
	"devilfish/internal/ports/outbound"
)

// Agent is the main agent that uses LLM to make decisions and execute tools.
type Agent struct {
	config   Config
	skills   *skill.Registry
	policies *policy.PolicySet
	steps    []Step
	messages []Message
	provider outbound.AIProvider
	logger   logging.Logger
}

// NewAgent creates a new agent with the given configuration.
func NewAgent(
	config Config,
	skills *skill.Registry,
	policies *policy.PolicySet,
	provider outbound.AIProvider,
	logger logging.Logger,
) *Agent {
	// Use default system prompt if not provided
	if config.SystemPrompt == "" {
		config.SystemPrompt = DefaultSystemPrompt()
	}

	// Set default max iterations if not configured
	if config.MaxIterations <= 0 {
		config.MaxIterations = 10
	}

	// Set default temperature if not configured
	if config.Temperature <= 0 {
		config.Temperature = 0.7
	}

	return &Agent{
		config:   config,
		skills:   skills,
		policies: policies,
		steps:    make([]Step, 0),
		messages: make([]Message, 0),
		provider: provider,
		logger:   logger,
	}
}

// Run executes the agent loop until a final response is generated or max iterations reached.
func (a *Agent) Run(ctx context.Context, userInput string) (string, error) {
	startTime := time.Now()

	a.logger.With(map[string]interface{}{
		"user_input_length": len(userInput),
		"max_iterations":    a.config.MaxIterations,
		"model":             a.config.Model,
	}).Info("agent loop started")

	// 1. Add user message to context
	a.messages = append(a.messages, Message{
		Role:    "user",
		Content: userInput,
	})

	// 2. Loop until max iterations
	for iteration := 0; iteration < a.config.MaxIterations; iteration++ {
		iterStart := time.Now()

		a.logger.With(map[string]interface{}{
			"iteration": iteration + 1,
			"max":       a.config.MaxIterations,
		}).Debug("agent iteration started")

		// Call LLM with messages
		response, err := a.callLLM(ctx)
		if err != nil {
			a.recordStep("", nil, "", fmt.Errorf("LLM call failed: %w", err), time.Since(iterStart))
			return "", fmt.Errorf("LLM call failed at iteration %d: %w", iteration+1, err)
		}

		// Parse response for tool calls
		toolCalls, thought, err := a.parseResponse(response)
		if err != nil {
			a.recordStep(thought, nil, "", fmt.Errorf("failed to parse response: %w", err), time.Since(iterStart))
			return "", fmt.Errorf("failed to parse LLM response: %w", err)
		}

		// If no tool calls, return the response as final
		if len(toolCalls) == 0 {
			a.recordStep(thought, nil, response, nil, time.Since(iterStart))
			a.logger.With(map[string]interface{}{
				"total_duration": time.Since(startTime).String(),
				"iterations":     iteration + 1,
			}).Info("agent loop completed with direct response")
			return response, nil
		}

		// Process tool calls
		for _, tc := range toolCalls {
			stepStart := time.Now()

			a.logger.With(map[string]interface{}{
				"tool_name": tc.Name,
				"iteration": iteration + 1,
			}).Info("executing tool")

			// Apply ExecutionPolicy BEFORE tool execution
			if a.policies != nil {
				if err := a.policies.Allow(tc); err != nil {
					errMsg := fmt.Sprintf("policy denied tool %q: %v", tc.Name, err)
					a.recordStep(thought, &tc, errMsg, err, time.Since(stepStart))
					a.logger.With(map[string]interface{}{
						"tool_name": tc.Name,
						"error":      err.Error(),
					}).Warn("tool execution denied by policy")
					continue
				}
			}

			// Execute skill
			result, err := a.executeTool(ctx, tc)
			if err != nil {
				errMsg := fmt.Sprintf("tool %q execution failed: %v", tc.Name, err)
				a.recordStep(thought, &tc, errMsg, err, time.Since(stepStart))
				a.logger.With(map[string]interface{}{
					"tool_name": tc.Name,
					"error":      err.Error(),
				}).Error("tool execution failed")
				continue
			}

			// Record successful step
			a.recordStep(thought, &tc, result, nil, time.Since(stepStart))

			// Append tool result as Message{Role: "tool"}
			a.messages = append(a.messages, Message{
				Role:    "tool",
				Content: result,
			})

			a.logger.With(map[string]interface{}{
				"tool_name": tc.Name,
				"duration":  time.Since(stepStart).String(),
			}).Info("tool executed successfully")
		}

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			a.logger.Warn("agent loop cancelled")
			return "", ctx.Err()
		default:
			// Continue
		}
	}

	// Max iterations reached
	a.logger.With(map[string]interface{}{
		"max_iterations": a.config.MaxIterations,
		"total_duration": time.Since(startTime).String(),
	}).Warn("agent loop reached max iterations")

	// Try to get a final response from LLM with the accumulated context
	finalResponse, err := a.callLLM(ctx)
	if err != nil {
		return "", fmt.Errorf("max iterations reached and failed to get final response: %w", err)
	}

	return finalResponse, nil
}

// callLLM sends the current messages to the LLM and returns the response.
func (a *Agent) callLLM(ctx context.Context) (string, error) {
	// Build chat messages for the AI provider
	chatMessages := make([]outbound.ChatMessage, 0, len(a.messages)+1)

	// Add system message if we have a system prompt
	if a.config.SystemPrompt != "" {
		chatMessages = append(chatMessages, outbound.ChatMessage{
			Role:    "system",
			Content: a.config.SystemPrompt,
		})
	}

	// Add all conversation messages
	for _, msg := range a.messages {
		chatMessages = append(chatMessages, outbound.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Build tool definitions if skills are available
	systemPrompt := a.config.SystemPrompt
	if a.skills != nil {
		systemPrompt = BuildSystemPrompt(a.skills, a.config.SystemPrompt)
	}

	// Create chat request
	req := &outbound.ChatRequest{
		Model:        a.config.Model,
		Messages:     chatMessages,
		SystemPrompt: systemPrompt,
		Temperature:  float64(a.config.Temperature),
		MaxTokens:    4096,
	}

	// Call the AI provider
	resp, err := a.provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("AI provider call failed: %w", err)
	}

	if resp == nil || resp.Content == "" {
		return "", fmt.Errorf("AI provider returned empty response")
	}

	return resp.Content, nil
}

// parseResponse parses the LLM response to extract tool calls and thought.
// Expected format from LLM:
// Thought: <reasoning>
// Action: <tool_name>
// Action Input: <json_params>
// OR for final answer:
// Final Answer: <response>
func (a *Agent) parseResponse(response string) ([]skill.ToolCall, string, error) {
	toolCalls := make([]skill.ToolCall, 0)
	thought := ""

	// Try to extract tool calls from the response
	// Format 1: JSON tool call format
	var jsonCall struct {
		Thought    string                 `json:"thought"`
		ToolCalls  []skill.ToolCall       `json:"tool_calls"`
		Action     string                 `json:"action"`
		ActionInput map[string]interface{} `json:"action_input"`
	}

	if err := json.Unmarshal([]byte(response), &jsonCall); err == nil {
		if jsonCall.Thought != "" {
			thought = jsonCall.Thought
		}

		// Check for tool_calls array format
		if len(jsonCall.ToolCalls) > 0 {
			return jsonCall.ToolCalls, thought, nil
		}

		// Check for single action format
		if jsonCall.Action != "" {
			tc := skill.ToolCall{
				Name:       jsonCall.Action,
				Parameters: jsonCall.ActionInput,
			}
			return []skill.ToolCall{tc}, thought, nil
		}
	}

	// Format 2: Parse text-based format (Thought/Action/Action Input)
	if contains(response, "Thought:") {
		if idx := indexOf(response, "Thought:"); idx >= 0 {
			rest := response[idx+len("Thought:"):]
			if idx2 := indexOf(rest, "Action:"); idx2 >= 0 {
				thought = rest[:idx2]
				thought = trimWhitespace(thought)
			} else {
				thought = trimWhitespace(rest)
			}
		}
	}

	// Try to find Action: and Action Input: patterns
	if contains(response, "Action:") && contains(response, "Action Input:") {
		action := extractBetween(response, "Action:", "Action Input:")
		actionInput := extractBetween(response, "Action Input:", "")

		if action != "" {
			var params map[string]interface{}
			if actionInput != "" {
				if err := json.Unmarshal([]byte(actionInput), &params); err != nil {
					// If not valid JSON, use as string parameter
					params = map[string]interface{}{"input": actionInput}
				}
			} else {
				params = make(map[string]interface{})
			}

			tc := skill.ToolCall{
				Name:       trimWhitespace(action),
				Parameters: params,
			}
			toolCalls = append(toolCalls, tc)
			return toolCalls, thought, nil
		}
	}

	// No tool calls found - this is a final answer
	return toolCalls, thought, nil
}

// executeTool executes a tool call using the skill registry.
func (a *Agent) executeTool(ctx context.Context, tc skill.ToolCall) (string, error) {
	if a.skills == nil {
		return "", fmt.Errorf("no skill registry available")
	}

	result, err := a.skills.Execute(ctx, tc)
	if err != nil {
		return "", fmt.Errorf("skill execution failed: %w", err)
	}

	if result.Error != nil {
		return "", result.Error
	}

	// Convert output to string
	switch v := result.Output.(type) {
	case string:
		return v, nil
	case nil:
		return "null", nil
	default:
		// Try to marshal to JSON
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v), nil
		}
		return string(data), nil
	}
}

// recordStep records a step for observability.
func (a *Agent) recordStep(thought string, toolCall *skill.ToolCall, result string, err error, duration time.Duration) {
	step := Step{
		Thought:  thought,
		ToolCall: toolCall,
		Result:   result,
		Error:    err,
		Duration: duration,
	}
	a.steps = append(a.steps, step)
}

// GetSteps returns the execution steps for observability.
func (a *Agent) GetSteps() []Step {
	return a.steps
}

// GetMessages returns the current conversation messages.
func (a *Agent) GetMessages() []Message {
	return a.messages
}

// Reset clears the agent's state (steps and messages).
func (a *Agent) Reset() {
	a.steps = make([]Step, 0)
	a.messages = make([]Message, 0)
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(s)-len(substr)+1] == substr || contains(s[1:], substr)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func trimWhitespace(s string) string {
	// Simple whitespace trim
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func extractBetween(s, start, end string) string {
	idx1 := indexOf(s, start)
	if idx1 < 0 {
		return ""
	}
	idx1 += len(start)

	if end == "" {
		return trimWhitespace(s[idx1:])
	}

	idx2 := indexOf(s[idx1:], end)
	if idx2 < 0 {
		return trimWhitespace(s[idx1:])
	}

	return trimWhitespace(s[idx1 : idx1+idx2])
}
