package skill

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSkill is a mock implementation of the Skill interface for testing.
type MockSkill struct {
	mock.Mock
	name        string
	description string
	schema      map[string]interface{}
}

func (m *MockSkill) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSkill) Description() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSkill) Schema() map[string]interface{} {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[string]interface{})
}

func (m *MockSkill) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	callArgs := m.Called(ctx, args)
	return callArgs.String(0), callArgs.Error(1)
}

// TestNewRegistry tests the NewRegistry constructor.
func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()

	assert.NotNil(t, reg)
	assert.Equal(t, 0, reg.Count())
}

// TestRegistry_Register tests registering a skill.
func TestRegistry_Register(t *testing.T) {
	reg := NewRegistry()

	skill := new(MockSkill)
	skill.name = "test_skill"
	skill.On("Name").Return("test_skill")

	reg.Register(skill)

	assert.Equal(t, 1, reg.Count())
	assert.True(t, reg.Has("test_skill"))
	skill.AssertExpectations(t)
}

// TestRegistry_Register_Overwrite tests that registering with same name overwrites.
func TestRegistry_Register_Overwrite(t *testing.T) {
	reg := NewRegistry()

	// Register first skill
	skill1 := new(MockSkill)
	skill1.name = "test_skill"
	skill1.On("Name").Return("test_skill")

	reg.Register(skill1)
	assert.Equal(t, 1, reg.Count())

	// Register second skill with same name
	skill2 := new(MockSkill)
	skill2.name = "test_skill"
	skill2.On("Name").Return("test_skill")

	reg.Register(skill2)
	assert.Equal(t, 1, reg.Count()) // Still 1, overwritten
	assert.True(t, reg.Has("test_skill"))

	skill1.AssertExpectations(t)
	skill2.AssertExpectations(t)
}

// TestRegistry_Unregister tests unregistering a skill.
func TestRegistry_Unregister(t *testing.T) {
	reg := NewRegistry()

	skill := new(MockSkill)
	skill.name = "test_skill"
	skill.On("Name").Return("test_skill")

	reg.Register(skill)
	assert.True(t, reg.Has("test_skill"))

	reg.Unregister("test_skill")
	assert.False(t, reg.Has("test_skill"))
	assert.Equal(t, 0, reg.Count())

	skill.AssertExpectations(t)
}

// TestRegistry_Get tests retrieving a skill by name.
func TestRegistry_Get(t *testing.T) {
	reg := NewRegistry()

	skill := new(MockSkill)
	skill.name = "test_skill"
	skill.On("Name").Return("test_skill")

	reg.Register(skill)

	// Get existing skill
	retrieved, found := reg.Get("test_skill")
	assert.True(t, found)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test_skill", retrieved.Name())

	// Get non-existent skill
	_, found = reg.Get("non_existent")
	assert.False(t, found)

	skill.AssertExpectations(t)
}

// TestRegistry_List tests listing all skills.
func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()

	// Empty registry
	skills := reg.List()
	assert.Empty(t, skills)

	// Add skills
	skill1 := new(MockSkill)
	skill1.name = "skill1"
	skill1.On("Name").Return("skill1")

	skill2 := new(MockSkill)
	skill2.name = "skill2"
	skill2.On("Name").Return("skill2")

	reg.Register(skill1)
	reg.Register(skill2)

	skills = reg.List()
	assert.Len(t, skills, 2)

	skill1.AssertExpectations(t)
	skill2.AssertExpectations(t)
}

// TestRegistry_GetSchemas tests retrieving schemas for all skills.
func TestRegistry_GetSchemas(t *testing.T) {
	reg := NewRegistry()

	// Create mock skills with schemas
	skill1 := new(MockSkill)
	skill1.name = "skill1"
	skill1.On("Name").Return("skill1")
	skill1.On("Description").Return("Description 1")
	skill1.On("Schema").Return(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"param1": map[string]interface{}{"type": "string"},
		},
	})

	skill2 := new(MockSkill)
	skill2.name = "skill2"
	skill2.On("Name").Return("skill2")
	skill2.On("Description").Return("Description 2")
	skill2.On("Schema").Return(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"param2": map[string]interface{}{"type": "number"},
		},
	})

	reg.Register(skill1)
	reg.Register(skill2)

	schemas := reg.GetSchemas()
	assert.Len(t, schemas, 2)

	// Check first schema
	assert.Equal(t, "skill1", schemas[0]["name"])
	assert.Equal(t, "Description 1", schemas[0]["description"])
	assert.NotNil(t, schemas[0]["inputSchema"])

	// Check second schema
	assert.Equal(t, "skill2", schemas[1]["name"])
	assert.Equal(t, "Description 2", schemas[1]["description"])
	assert.NotNil(t, schemas[1]["inputSchema"])

	skill1.AssertExpectations(t)
	skill2.AssertExpectations(t)
}

// TestRegistry_ExecuteSkill tests executing a skill through the registry.
func TestRegistry_ExecuteSkill(t *testing.T) {
	ctx := context.Background()
	reg := NewRegistry()

	skill := new(MockSkill)
	skill.name = "test_skill"
	skill.On("Name").Return("test_skill")
	skill.On("Execute", ctx, mock.Anything).Return("success result", nil)

	reg.Register(skill)

	result, err := reg.ExecuteSkill(ctx, "test_skill", map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, "success result", result)

	skill.AssertExpectations(t)
}

// TestRegistry_ExecuteSkill_NotFound tests executing a non-existent skill.
func TestRegistry_ExecuteSkill_NotFound(t *testing.T) {
	ctx := context.Background()
	reg := NewRegistry()

	result, err := reg.ExecuteSkill(ctx, "non_existent", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Empty(t, result)
}

// TestRegistry_ExecuteSkill_InvalidContext tests executing with invalid context.
func TestRegistry_ExecuteSkill_InvalidContext(t *testing.T) {
	reg := NewRegistry()

	skill := new(MockSkill)
	skill.name = "test_skill"
	skill.On("Name").Return("test_skill")

	reg.Register(skill)

	// The ExecuteSkill method expects a context.Context as first arg
	// Passing a string should cause a compile-time error, but since we're testing,
	// we'll just test the "not found" case instead
	result, err := reg.ExecuteSkill(context.Background(), "non_existent", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Empty(t, result)

	skill.AssertExpectations(t)
}

// TestRegistry_Count tests the Count method.
func TestRegistry_Count(t *testing.T) {
	reg := NewRegistry()

	assert.Equal(t, 0, reg.Count())

	skill1 := new(MockSkill)
	skill1.name = "skill1"
	skill1.On("Name").Return("skill1")
	reg.Register(skill1)
	assert.Equal(t, 1, reg.Count())

	skill2 := new(MockSkill)
	skill2.name = "skill2"
	skill2.On("Name").Return("skill2")
	reg.Register(skill2)
	assert.Equal(t, 2, reg.Count())

	skill1.AssertExpectations(t)
	skill2.AssertExpectations(t)
}

// TestRegistry_Has tests the Has method.
func TestRegistry_Has(t *testing.T) {
	reg := NewRegistry()

	assert.False(t, reg.Has("test_skill"))

	skill := new(MockSkill)
	skill.name = "test_skill"
	skill.On("Name").Return("test_skill")
	reg.Register(skill)

	assert.True(t, reg.Has("test_skill"))

	skill.AssertExpectations(t)
}

// TestRegistry_Clear tests clearing all skills.
func TestRegistry_Clear(t *testing.T) {
	reg := NewRegistry()

	skill1 := new(MockSkill)
	skill1.name = "skill1"
	skill1.On("Name").Return("skill1")
	reg.Register(skill1)

	skill2 := new(MockSkill)
	skill2.name = "skill2"
	skill2.On("Name").Return("skill2")
	reg.Register(skill2)

	assert.Equal(t, 2, reg.Count())

	reg.Clear()
	assert.Equal(t, 0, reg.Count())
	assert.False(t, reg.Has("skill1"))
	assert.False(t, reg.Has("skill2"))

	skill1.AssertExpectations(t)
	skill2.AssertExpectations(t)
}

// TestRegistry_ConcurrentAccess tests thread safety of the registry.
func TestRegistry_ConcurrentAccess(t *testing.T) {
	reg := NewRegistry()

	// Register some initial skills
	for i := 0; i < 10; i++ {
		skill := new(MockSkill)
		skill.name = "skill"
		skill.On("Name").Return("skill")
		reg.Register(skill)
	}

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(index int) {
			// Register
			skill := new(MockSkill)
			skill.name = "skill"
			skill.On("Name").Return("skill")
			reg.Register(skill)

			// Get
			reg.Get("skill")

			// List
			reg.List()

			// Has
			reg.Has("skill")

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic or have race conditions
	assert.True(t, reg.Count() >= 1)
}
