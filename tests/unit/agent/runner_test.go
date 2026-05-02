package agent_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"devilfish/internal/agent"
	"devilfish/internal/skills"
)

// --- fakes ---

// staticLLM returns a fixed LLMResponse every call.
type staticLLM struct {
	responses []agent.LLMResponse
	calls     int
	err       error
}

func (l *staticLLM) Generate(_ []agent.Message) (agent.LLMResponse, error) {
	if l.err != nil {
		return agent.LLMResponse{}, l.err
	}
	if l.calls >= len(l.responses) {
		return agent.LLMResponse{Content: "fallback"}, nil
	}
	resp := l.responses[l.calls]
	l.calls++
	return resp, nil
}

// echoSkill returns its "text" argument as the result.
type echoSkill struct{}

func (e *echoSkill) Name() string        { return "echo" }
func (e *echoSkill) Description() string { return "echoes input" }
func (e *echoSkill) Execute(input map[string]interface{}) (string, error) {
	text, _ := input["text"].(string)
	return text, nil
}

// failSkill always returns an error.
type failSkill struct{}

func (f *failSkill) Name() string        { return "fail" }
func (f *failSkill) Description() string { return "always fails" }
func (f *failSkill) Execute(_ map[string]interface{}) (string, error) {
	return "", fmt.Errorf("skill error")
}

func newRegistry(ss ...skills.Skill) *skills.SkillRegistry {
	r := skills.NewSkillRegistry()
	for _, s := range ss {
		r.Register(s)
	}
	return r
}

// --- tests ---

func TestRunAgent_DirectAnswer_NoToolCall(t *testing.T) {
	llm := &staticLLM{
		responses: []agent.LLMResponse{
			{Content: "Go is great!"},
		},
	}
	result, err := agent.RunAgent(llm, nil, newRegistry(), 5)
	require.NoError(t, err)
	assert.Equal(t, "Go is great!", result)
}

func TestRunAgent_OneToolCallThenAnswer(t *testing.T) {
	llm := &staticLLM{
		responses: []agent.LLMResponse{
			{ToolCall: &skills.ToolCall{Name: "echo", Arguments: map[string]interface{}{"text": "search result"}}},
			{Content: "Final answer after tool."},
		},
	}
	result, err := agent.RunAgent(llm, nil, newRegistry(&echoSkill{}), 5)
	require.NoError(t, err)
	assert.Equal(t, "Final answer after tool.", result)
}

func TestRunAgent_ToolResultAppendedToContext(t *testing.T) {
	var lastMessages []agent.Message
	// Custom LLM that captures messages on second call.
	capturingLLM := &captureLLM{
		responses: []agent.LLMResponse{
			{ToolCall: &skills.ToolCall{Name: "echo", Arguments: map[string]interface{}{"text": "captured"}}},
			{Content: "done"},
		},
		capture: &lastMessages,
	}
	_, err := agent.RunAgent(capturingLLM, nil, newRegistry(&echoSkill{}), 5)
	require.NoError(t, err)
	// The last call should include the "tool" message with the skill result.
	require.Len(t, lastMessages, 1)
	assert.Equal(t, "tool", lastMessages[0].Role)
	assert.Equal(t, "captured", lastMessages[0].Content)
}

func TestRunAgent_MaxIterationsReached_ReturnsError(t *testing.T) {
	// LLM always requests a tool call.
	llm := &staticLLM{
		responses: []agent.LLMResponse{
			{ToolCall: &skills.ToolCall{Name: "echo", Arguments: map[string]interface{}{"text": "x"}}},
			{ToolCall: &skills.ToolCall{Name: "echo", Arguments: map[string]interface{}{"text": "x"}}},
			{ToolCall: &skills.ToolCall{Name: "echo", Arguments: map[string]interface{}{"text": "x"}}},
		},
	}
	_, err := agent.RunAgent(llm, nil, newRegistry(&echoSkill{}), 3)
	require.Error(t, err)
	assert.True(t, errors.Is(err, agent.ErrMaxIterationsReached))
}

func TestRunAgent_LLMError_ReturnsError(t *testing.T) {
	llm := &staticLLM{err: fmt.Errorf("LLM unavailable")}
	_, err := agent.RunAgent(llm, nil, newRegistry(), 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LLM unavailable")
}

func TestRunAgent_SkillExecutionError_ReturnsError(t *testing.T) {
	llm := &staticLLM{
		responses: []agent.LLMResponse{
			{ToolCall: &skills.ToolCall{Name: "fail"}},
		},
	}
	_, err := agent.RunAgent(llm, nil, newRegistry(&failSkill{}), 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool execution error")
}

func TestRunAgent_UnknownSkill_ReturnsError(t *testing.T) {
	llm := &staticLLM{
		responses: []agent.LLMResponse{
			{ToolCall: &skills.ToolCall{Name: "nonexistent"}},
		},
	}
	_, err := agent.RunAgent(llm, nil, newRegistry(), 5)
	require.Error(t, err)
}

func TestRunAgent_NilLLM_ReturnsError(t *testing.T) {
	_, err := agent.RunAgent(nil, nil, newRegistry(), 5)
	require.Error(t, err)
}

func TestRunAgent_NilRegistry_ReturnsError(t *testing.T) {
	llm := &staticLLM{}
	_, err := agent.RunAgent(llm, nil, nil, 5)
	require.Error(t, err)
}

func TestRunAgent_ZeroMaxIterations_ReturnsError(t *testing.T) {
	llm := &staticLLM{}
	_, err := agent.RunAgent(llm, nil, newRegistry(), 0)
	require.Error(t, err)
}

func TestRunAgent_InitialContextPassedToLLM(t *testing.T) {
	var firstCall []agent.Message
	llm := &captureLLM{
		responses: []agent.LLMResponse{{Content: "done"}},
		capture:   &firstCall,
	}
	initial := []agent.Message{
		{Role: "user", Content: "What are the best Go frameworks?"},
	}
	_, err := agent.RunAgent(llm, initial, newRegistry(), 5)
	require.NoError(t, err)
	require.Len(t, firstCall, 1)
	assert.Equal(t, "user", firstCall[0].Role)
}

// captureLLM saves the messages from the SECOND call onwards.
type captureLLM struct {
	responses []agent.LLMResponse
	calls     int
	capture   *[]agent.Message
}

func (c *captureLLM) Generate(messages []agent.Message) (agent.LLMResponse, error) {
	if c.calls == 0 {
		// First call: capture the full context (used by InitialContextPassedToLLM test).
		*c.capture = messages
	} else {
		// Subsequent calls: capture only the last appended message.
		*c.capture = messages[len(messages)-1:]
	}
	resp := c.responses[c.calls]
	c.calls++
	return resp, nil
}
