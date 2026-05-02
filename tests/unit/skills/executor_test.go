package skills_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"devilfish/internal/skills"
)

// echoSkill returns its "text" input argument as the result.
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
	return "", fmt.Errorf("something went wrong")
}

func newRegistryWith(ss ...skills.Skill) *skills.SkillRegistry {
	r := skills.NewSkillRegistry()
	for _, s := range ss {
		r.Register(s)
	}
	return r
}

func TestExecuteSkill_Success(t *testing.T) {
	r := newRegistryWith(&echoSkill{})
	result, err := skills.ExecuteSkill(
		skills.ToolCall{Name: "echo", Arguments: map[string]interface{}{"text": "hello"}},
		r,
	)
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestExecuteSkill_MissingSkill_ReturnsErrSkillNotFound(t *testing.T) {
	r := skills.NewSkillRegistry()
	_, err := skills.ExecuteSkill(skills.ToolCall{Name: "ghost"}, r)
	require.Error(t, err)
	assert.True(t, errors.Is(err, skills.ErrSkillNotFound))
	assert.Contains(t, err.Error(), "ghost")
}

func TestExecuteSkill_SkillError_WrapsError(t *testing.T) {
	r := newRegistryWith(&failSkill{})
	_, err := skills.ExecuteSkill(skills.ToolCall{Name: "fail"}, r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fail")
	assert.Contains(t, err.Error(), "something went wrong")
}

func TestExecuteSkill_NilRegistry_ReturnsError(t *testing.T) {
	_, err := skills.ExecuteSkill(skills.ToolCall{Name: "echo"}, nil)
	require.Error(t, err)
}
