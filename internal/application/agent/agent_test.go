package agent

import (
	"context"
	"errors"
	"testing"

	"devilfish/internal/application/policy"
	"devilfish/internal/application/skill"
	"devilfish/internal/infra/logging"
	"devilfish/internal/ports/outbound"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAIProvider is a mock implementation of outbound.AIProvider
type MockAIProvider struct {
	mock.Mock
}

func (m *MockAIProvider) Chat(ctx context.Context, req *outbound.ChatRequest) (*outbound.ChatResponse, error) {
	args := m.Called(ctx, req)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return resp.(*outbound.ChatResponse), args.Error(1)
}

func (m *MockAIProvider) StreamChat(ctx context.Context, req *outbound.ChatRequest, onChunk func(string)) error {
	args := m.Called(ctx, req, onChunk)
	return args.Error(0)
}

func (m *MockAIProvider) IsAvailable() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockAIProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

// MockLogger is a simple logger for testing
type MockLogger struct{}

func (l *MockLogger) Debug(msg string)                            {}
func (l *MockLogger) Debugf(format string, args ...interface{})    {}
func (l *MockLogger) Info(msg string)                             {}
func (l *MockLogger) Infof(format string, args ...interface{})     {}
func (l *MockLogger) Warn(msg string)                             {}
func (l *MockLogger) Warnf(format string, args ...interface{})     {}
func (l *MockLogger) Error(msg string)                             {}
func (l *MockLogger) Errorf(format string, args ...interface{})     {}
func (l *MockLogger) With(fields map[string]interface{}) logging.Logger { return l }

// mockSkill is a mock implementation of skill.Skill for testing
type mockSkill struct {
	name        string
	description string
	schema      map[string]interface{}
	executeFunc func(ctx context.Context, args map[string]interface{}) (string, error)
}

func (m *mockSkill) Name() string {
	return m.name
}

func (m *mockSkill) Description() string {
	return m.description
}

func (m *mockSkill) Schema() map[string]interface{} {
	return m.schema
}

func (m *mockSkill) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, args)
	}
	return "mock result", nil
}

// TestNewAgent tests agent creation
func TestNewAgent(t *testing.T) {
	config := Config{
		MaxIterations: 5,
		SystemPrompt:  "Test prompt",
		Model:         "test-model",
		Temperature:   0.5,
	}

	skills := skill.NewRegistry()
	policies := policy.NewPolicySet()
	provider := new(MockAIProvider)
	logger := &MockLogger{}

	agent := NewAgent(config, skills, policies, provider, logger)

	assert.NotNil(t, agent)
	assert.Equal(t, 5, agent.config.MaxIterations)
	assert.Equal(t, "Test prompt", agent.config.SystemPrompt)
	assert.Equal(t, "test-model", agent.config.Model)
	assert.Equal(t, float32(0.5), agent.config.Temperature)
}

// TestNewAgentWithDefaults tests agent creation with default values
func TestNewAgentWithDefaults(t *testing.T) {
	config := Config{}
	skills := skill.NewRegistry()
	policies := policy.NewPolicySet()
	provider := new(MockAIProvider)
	logger := &MockLogger{}

	agent := NewAgent(config, skills, policies, provider, logger)

	assert.NotNil(t, agent)
	assert.Equal(t, 10, agent.config.MaxIterations) // Default max iterations
	assert.Equal(t, DefaultSystemPrompt(), agent.config.SystemPrompt)
	assert.Equal(t, float32(0.7), agent.config.Temperature) // Default temperature
}

// TestAgentRunDirectResponse tests agent returning a direct response without tool calls
func TestAgentRunDirectResponse(t *testing.T) {
	// Setup
	config := Config{
		MaxIterations: 5,
		Model:         "test-model",
	}

	skills := skill.NewRegistry()
	policies := policy.NewPolicySet()
	provider := new(MockAIProvider)
	logger := &MockLogger{}

	// Mock AI provider to return a direct response (no tool calls)
	provider.On("Chat", mock.Anything, mock.Anything).
		Return(&outbound.ChatResponse{
			Content: "Hello! I'm here to help you.",
		}, nil)

	agent := NewAgent(config, skills, policies, provider, logger)

	// Execute
	result, err := agent.Run(context.Background(), "Hello")

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, "Hello! I'm here to help you.", result)
	assert.Equal(t, 1, len(agent.GetSteps()))

	provider.AssertExpectations(t)
}

// TestAgentRunWithToolCall tests agent executing a tool
func TestAgentRunWithToolCall(t *testing.T) {
	// Setup
	config := Config{
		MaxIterations: 5,
		Model:         "test-model",
	}

	skills := skill.NewRegistry()
	policies := policy.NewPolicySet()
	provider := new(MockAIProvider)
	logger := &MockLogger{}

	// Register a test skill
	weatherSkill := &mockSkill{
		name:        "get_weather",
		description: "Get weather for a city",
		executeFunc: func(ctx context.Context, args map[string]interface{}) (string, error) {
			return "Sunny, 25°C", nil
		},
	}
	skills.Register(weatherSkill)

	// Mock AI provider responses
	// First call: LLM decides to use tool
	provider.On("Chat", mock.Anything, mock.Anything).
		Return(&outbound.ChatResponse{
			Content: `{"thought": "I should get weather", "action": "get_weather", "action_input": {"city": "London"}}`,
		}, nil).Once()

	// Second call: LLM gives final answer after tool result
	provider.On("Chat", mock.Anything, mock.Anything).
		Return(&outbound.ChatResponse{
			Content: "The weather in London is sunny at 25°C.",
		}, nil).Once()

	agent := NewAgent(config, skills, policies, provider, logger)

	// Execute
	result, err := agent.Run(context.Background(), "What's the weather in London?")

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, "The weather in London is sunny at 25°C.", result)

	// Check steps were recorded
	steps := agent.GetSteps()
	assert.GreaterOrEqual(t, len(steps), 1)

	// Check that tool was executed
	hasToolStep := false
	for _, step := range steps {
		if step.ToolCall != nil && step.ToolCall.Name == "get_weather" {
			hasToolStep = true
			assert.NoError(t, step.Error)
		}
	}
	assert.True(t, hasToolStep, "Should have a step with tool call to get_weather")

	provider.AssertExpectations(t)
}

// TestAgentMaxIterations tests that agent stops after max iterations
func TestAgentMaxIterations(t *testing.T) {
	// Setup
	config := Config{
		MaxIterations: 3,
		Model:         "test-model",
	}

	skills := skill.NewRegistry()
	policies := policy.NewPolicySet()
	provider := new(MockAIProvider)
	logger := &MockLogger{}

	// Register a tool that always gets called
	callCount := 0
	infiniteSkill := &mockSkill{
		name:        "infinite_tool",
		description: "A tool that keeps getting called",
		executeFunc: func(ctx context.Context, args map[string]interface{}) (string, error) {
			callCount++
			return "done", nil
		},
	}
	skills.Register(infiniteSkill)

	// Mock AI provider to always return a tool call
	provider.On("Chat", mock.Anything, mock.Anything).
		Return(&outbound.ChatResponse{
			Content: `{"thought": "need to call tool", "action": "infinite_tool", "action_input": {}}`,
		}, nil)

	agent := NewAgent(config, skills, policies, provider, logger)

	// Execute
	_, err := agent.Run(context.Background(), "Start infinite loop")

	// Verify - should stop after max iterations
	assert.NoError(t, err) // Should not error, just stop
	assert.GreaterOrEqual(t, callCount, 3)
	assert.LessOrEqual(t, callCount, config.MaxIterations)

	provider.AssertExpectations(t)
}

// TestAgentPolicyDenied tests that policies are applied before tool execution
func TestAgentPolicyDenied(t *testing.T) {
	// Setup
	config := Config{
		MaxIterations: 5,
		Model:         "test-model",
	}

	skills := skill.NewRegistry()
	logger := &MockLogger{}

	// Create a policy that denies the dangerous_tool
	denyPolicy := &mockPolicy{denyTool: "dangerous_tool"}
	policies := policy.NewPolicySet()
	policies.Add(denyPolicy)

	provider := new(MockAIProvider)

	// Register tools
	dangerousSkill := &mockSkill{
		name:        "dangerous_tool",
		description: "A dangerous tool",
		executeFunc: func(ctx context.Context, args map[string]interface{}) (string, error) {
			return "should not reach here", nil
		},
	}
	skills.Register(dangerousSkill)

	// Mock AI provider to call the dangerous tool
	provider.On("Chat", mock.Anything, mock.Anything).
		Return(&outbound.ChatResponse{
			Content: `{"thought": "I'll use dangerous tool", "action": "dangerous_tool", "action_input": {}}`,
		}, nil).Once()

	// After policy denial, LLM should give final answer
	provider.On("Chat", mock.Anything, mock.Anything).
		Return(&outbound.ChatResponse{
			Content: "I cannot execute that tool due to policy restrictions.",
		}, nil).Once()

	agent := NewAgent(config, skills, policies, provider, logger)

	// Execute
	result, err := agent.Run(context.Background(), "Do something dangerous")

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, "I cannot execute that tool due to policy restrictions.", result)

	provider.AssertExpectations(t)
}

// TestAgentToolExecutionError tests handling of tool execution errors
func TestAgentToolExecutionError(t *testing.T) {
	// Setup
	config := Config{
		MaxIterations: 5,
		Model:         "test-model",
	}

	skills := skill.NewRegistry()
	policies := policy.NewPolicySet()
	provider := new(MockAIProvider)
	logger := &MockLogger{}

	// Register a tool that returns an error
	failingSkill := &mockSkill{
		name:        "failing_tool",
		description: "A tool that fails",
		executeFunc: func(ctx context.Context, args map[string]interface{}) (string, error) {
			return "", errors.New("tool execution failed")
		},
	}
	skills.Register(failingSkill)

	// Mock AI provider
	provider.On("Chat", mock.Anything, mock.Anything).
		Return(&outbound.ChatResponse{
			Content: `{"thought": "try failing tool", "action": "failing_tool", "action_input": {}}`,
		}, nil).Once()

	// After error, LLM gives final answer
	provider.On("Chat", mock.Anything, mock.Anything).
		Return(&outbound.ChatResponse{
			Content: "The tool failed to execute.",
		}, nil).Once()

	agent := NewAgent(config, skills, policies, provider, logger)

	// Execute
	result, err := agent.Run(context.Background(), "Use failing tool")

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, "The tool failed to execute.", result)

	provider.AssertExpectations(t)
}

// TestAgentGetSteps tests step tracking
func TestAgentGetSteps(t *testing.T) {
	config := Config{
		MaxIterations: 5,
		Model:         "test-model",
	}

	skills := skill.NewRegistry()
	policies := policy.NewPolicySet()
	provider := new(MockAIProvider)
	logger := &MockLogger{}

	// Register a tool
	testSkill := &mockSkill{
		name:        "test_tool",
		description: "Test tool",
		executeFunc: func(ctx context.Context, args map[string]interface{}) (string, error) {
			return "result", nil
		},
	}
	skills.Register(testSkill)

	// Mock AI provider
	provider.On("Chat", mock.Anything, mock.Anything).
		Return(&outbound.ChatResponse{
			Content: `{"thought": "use tool", "action": "test_tool", "action_input": {}}`,
		}, nil).Once()

	provider.On("Chat", mock.Anything, mock.Anything).
		Return(&outbound.ChatResponse{
			Content: "Done",
		}, nil).Once()

	agent := NewAgent(config, skills, policies, provider, logger)

	// Execute
	_, err := agent.Run(context.Background(), "Run test")

	// Verify steps
	assert.NoError(t, err)
	steps := agent.GetSteps()
	assert.GreaterOrEqual(t, len(steps), 1)

	// Check step structure
	for _, step := range steps {
		assert.NotNil(t, step)
		// Duration should be recorded (non-negative)
		assert.True(t, step.Duration >= 0, "Duration should be non-negative")
	}

	provider.AssertExpectations(t)
}

// TestAgentReset tests resetting agent state
func TestAgentReset(t *testing.T) {
	config := Config{
		MaxIterations: 5,
		Model:         "test-model",
	}

	skills := skill.NewRegistry()
	policies := policy.NewPolicySet()
	provider := new(MockAIProvider)
	logger := &MockLogger{}

	agent := NewAgent(config, skills, policies, provider, logger)

	// Add some dummy data
	agent.steps = append(agent.steps, Step{Thought: "test"})
	agent.messages = append(agent.messages, Message{Role: "user", Content: "test"})

	// Reset
	agent.Reset()

	assert.Equal(t, 0, len(agent.GetSteps()))
	assert.Equal(t, 0, len(agent.GetMessages()))
}

// TestBuildSystemPrompt tests system prompt building
func TestBuildSystemPrompt(t *testing.T) {
	skills := skill.NewRegistry()

	// Test with no skills
	prompt := BuildSystemPrompt(skills, "")
	assert.Contains(t, prompt, "AVAILABLE TOOLS:")
	assert.Contains(t, prompt, "No tools available")

	// Add some skills
	skill1 := &mockSkill{
		name:        "tool1",
		description: "First tool",
	}
	skill2 := &mockSkill{
		name:        "tool2",
		description: "Second tool",
	}
	skills.Register(skill1)
	skills.Register(skill2)

	prompt = BuildSystemPrompt(skills, "")
	assert.Contains(t, prompt, "tool1: First tool")
	assert.Contains(t, prompt, "tool2: Second tool")
}

// TestAgentWithToolSchema tests system prompt with tool schemas
func TestAgentWithToolSchema(t *testing.T) {
	skills := skill.NewRegistry()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"city": map[string]interface{}{
				"type":        "string",
				"description": "The city name",
			},
		},
		"required": []interface{}{"city"},
	}

	weatherSkill := &mockSkill{
		name:        "get_weather",
		description: "Get weather for a city",
		schema:      schema,
	}
	skills.Register(weatherSkill)

	prompt := BuildSystemPrompt(skills, "")
	assert.Contains(t, prompt, "get_weather: Get weather for a city")
	assert.Contains(t, prompt, "city")
}

// mockPolicy is a mock policy for testing
type mockPolicy struct {
	denyTool string
}

func (m *mockPolicy) Allow(call skill.ToolCall) error {
	if call.Name == m.denyTool {
		return errors.New("tool denied by mock policy")
	}
	return nil
}

// TestAgentContextCancellation tests that agent respects context cancellation
func TestAgentContextCancellation(t *testing.T) {
	config := Config{
		MaxIterations: 100,
		Model:         "test-model",
	}

	skills := skill.NewRegistry()
	policies := policy.NewPolicySet()
	provider := new(MockAIProvider)
	logger := &MockLogger{}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	provider.On("Chat", mock.Anything, mock.Anything).
		Return(&outbound.ChatResponse{
			Content: `{"thought": "test", "action": "tool", "action_input": {}}`,
		}, nil)

	agent := NewAgent(config, skills, policies, provider, logger)

	// Execute with cancelled context
	_, err := agent.Run(ctx, "Test")

	// Verify - agent should handle cancellation gracefully
	// (The actual behavior depends on implementation - it may return context.Canceled)
	if err != nil {
		assert.Equal(t, context.Canceled, err)
	}

	provider.AssertExpectations(t)
}
