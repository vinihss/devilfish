package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"devilfish/internal/application/skill"
)

// DefaultSystemPrompt returns the default system prompt.
func DefaultSystemPrompt() string {
	return `You are an AI agent with access to tools (skills). You help users by reasoning about their requests and using appropriate tools when needed.

RULES:
1. Think step by step before taking action
2. Only call tools when necessary to fulfill the user's request
3. Do not send emails, messages, or perform destructive actions without explicit user intent
4. Prefer reading data before acting on it
5. Avoid repeated or unnecessary tool calls
6. Be concise, accurate, and helpful in your responses
7. When you have enough information, provide a final answer

RESPONSE FORMAT:
When you need to use a tool, respond in this JSON format:
{
  "thought": "your reasoning about what to do next",
  "action": "tool_name",
  "action_input": {
    "param1": "value1",
    "param2": "value2"
  }
}

When you want to provide a final answer without using tools, just respond with the answer directly, or use:
{
  "thought": "I have enough information to answer",
  "final_answer": "your response to the user"
}

AVAILABLE TOOLS:
{{TOOLS}}

Remember: Always think before acting, and be helpful while being safe.`
}

// BuildSystemPrompt creates a system prompt with available tools.
// It injects the tool schemas into the prompt template.
func BuildSystemPrompt(skills *skill.Registry, basePrompt ...string) string {
	prompt := DefaultSystemPrompt()
	if len(basePrompt) > 0 && basePrompt[0] != "" {
		prompt = basePrompt[0]
	}

	// Get all registered skills
	if skills == nil {
		return replaceToolsPlaceholder(prompt, "No tools available.")
	}

	skillList := skills.List()
	if len(skillList) == 0 {
		return replaceToolsPlaceholder(prompt, "No tools available.")
	}

	// Build tool descriptions
	var sb strings.Builder
	for _, s := range skillList {
		sb.WriteString(fmt.Sprintf("- %s: %s", s.Name(), s.Description()))

		// Add input schema if available
		if schema := s.Schema(); schema != nil {
			schemaJSON, err := json.MarshalIndent(schema, "", "  ")
			if err == nil {
				sb.WriteString(fmt.Sprintf("\n  Parameters: %s", string(schemaJSON)))
			}
		}
		sb.WriteString("\n")
	}

	return replaceToolsPlaceholder(prompt, sb.String())
}

// replaceToolsPlaceholder replaces the {{TOOLS}} placeholder in the prompt.
func replaceToolsPlaceholder(prompt, toolsStr string) string {
	return strings.ReplaceAll(prompt, "{{TOOLS}}", toolsStr)
}

// BuildToolDescriptionsForLLM creates a structured tool description for LLM consumption.
// This can be used with LLM APIs that support tool/function calling natively.
func BuildToolDescriptionsForLLM(skills *skill.Registry) string {
	if skills == nil {
		return "[]"
	}

	skillList := skills.List()
	if len(skillList) == 0 {
		return "[]"
	}

	// Build OpenAI-style function definitions
	type FunctionDef struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Parameters  map[string]interface{} `json:"parameters,omitempty"`
	}

	type ToolDef struct {
		Type     string      `json:"type"`
		Function FunctionDef `json:"function"`
	}

	toolDefs := make([]ToolDef, 0, len(skillList))
	for _, s := range skillList {
		td := ToolDef{
			Type: "function",
			Function: FunctionDef{
				Name:        s.Name(),
				Description: s.Description(),
			},
		}

		if schema := s.Schema(); schema != nil {
			td.Function.Parameters = schema
		} else {
			// Default parameters schema
			td.Function.Parameters = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}

		toolDefs = append(toolDefs, td)
	}

	data, err := json.MarshalIndent(toolDefs, "", "  ")
	if err != nil {
		return "[]"
	}

	return string(data)
}

// FormatConversationForPrompt formats the conversation messages for inclusion in a prompt.
func FormatConversationForPrompt(messages []Message) string {
	var sb strings.Builder
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			sb.WriteString(fmt.Sprintf("System: %s\n", msg.Content))
		case "user":
			sb.WriteString(fmt.Sprintf("Human: %s\n", msg.Content))
		case "assistant":
			sb.WriteString(fmt.Sprintf("Assistant: %s\n", msg.Content))
		case "tool":
			sb.WriteString(fmt.Sprintf("Tool Result: %s\n", msg.Content))
		}
	}
	return sb.String()
}

// ParseJSONResponse attempts to parse a JSON response from the LLM.
// Returns the parsed map, a boolean indicating if it was valid JSON, and any error.
func ParseJSONResponse(response string) (map[string]interface{}, bool, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(response), &result)
	if err != nil {
		return nil, false, err
	}
	return result, true, nil
}

// HasToolCall checks if the response contains a tool call.
// This is a simple heuristic - can be extended based on actual LLM response patterns.
func HasToolCall(response string) bool {
	// Check for JSON format with action or tool_calls
	if strings.Contains(response, `"action"`) || strings.Contains(response, `"tool_calls"`) {
		return true
	}

	// Check for text format
	if strings.Contains(response, "Action:") && strings.Contains(response, "Action Input:") {
		return true
	}

	return false
}

// ExtractFinalAnswer attempts to extract a final answer from the response.
// Returns the answer if found, or empty string if not.
func ExtractFinalAnswer(response string) string {
	// Try JSON format first
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err == nil {
		if finalAns, ok := result["final_answer"].(string); ok {
			return finalAns
		}
		// If it's JSON but no final_answer and no action, treat content as answer
		if _, hasAction := result["action"]; !hasAction {
			return response
		}
	}

	// Not a tool call - treat as final answer
	if !HasToolCall(response) {
		return response
	}

	return ""
}
