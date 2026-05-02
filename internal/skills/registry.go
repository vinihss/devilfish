package skills

// SkillRegistry holds registered skills indexed by name.
type SkillRegistry struct {
	skills map[string]Skill
}

// NewSkillRegistry creates a new, empty SkillRegistry.
func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{skills: make(map[string]Skill)}
}

// Register adds a skill to the registry. If a skill with the same name
// already exists it is replaced.
func (r *SkillRegistry) Register(skill Skill) {
	r.skills[skill.Name()] = skill
}

// Get returns the skill registered under the given name.
func (r *SkillRegistry) Get(name string) (Skill, bool) {
	s, ok := r.skills[name]
	return s, ok
}

// List returns the names of all registered skills.
func (r *SkillRegistry) List() []string {
	names := make([]string, 0, len(r.skills))
	for name := range r.skills {
		names = append(names, name)
	}
	return names
}
