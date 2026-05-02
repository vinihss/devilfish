package skills_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"devilfish/internal/skills"
)

// stubSkill is a minimal Skill implementation for registry tests.
type stubSkill struct {
	name string
}

func (s *stubSkill) Name() string                                         { return s.name }
func (s *stubSkill) Description() string                                  { return "stub" }
func (s *stubSkill) Execute(_ map[string]interface{}) (string, error)     { return "ok", nil }

func TestNewSkillRegistry_Empty(t *testing.T) {
	r := skills.NewSkillRegistry()
	assert.Empty(t, r.List())
}

func TestSkillRegistry_Register_And_Get(t *testing.T) {
	r := skills.NewSkillRegistry()
	r.Register(&stubSkill{name: "foo"})

	s, ok := r.Get("foo")
	require.True(t, ok)
	assert.Equal(t, "foo", s.Name())
}

func TestSkillRegistry_Get_Missing_ReturnsFalse(t *testing.T) {
	r := skills.NewSkillRegistry()
	_, ok := r.Get("nonexistent")
	assert.False(t, ok)
}

func TestSkillRegistry_Register_Overwrites_Existing(t *testing.T) {
	r := skills.NewSkillRegistry()
	r.Register(&stubSkill{name: "foo"})
	r.Register(&stubSkill{name: "foo"})

	assert.Len(t, r.List(), 1)
}

func TestSkillRegistry_List_ReturnsAllNames(t *testing.T) {
	r := skills.NewSkillRegistry()
	r.Register(&stubSkill{name: "a"})
	r.Register(&stubSkill{name: "b"})
	r.Register(&stubSkill{name: "c"})

	names := r.List()
	assert.Len(t, names, 3)
	assert.ElementsMatch(t, []string{"a", "b", "c"}, names)
}
