package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterToolsFromFile_WithMarkdownRegistersBashSkill(t *testing.T) {
	tmpDir := t.TempDir()
	toolsPath := filepath.Join(tmpDir, "tools.md")

	content := `# Tools

` + "```yaml" + `
tools:
  - type: bash
    name: run_bash
    description: Run safe commands
    allowed_commands:
      - echo
` + "```" + `
`

	require.NoError(t, os.WriteFile(toolsPath, []byte(content), 0o644))

	registry := NewRegistry()
	err := RegisterToolsFromFile(registry, toolsPath)
	require.NoError(t, err)
	assert.True(t, registry.Has("run_bash"))

	result, err := registry.ExecuteSkill(context.Background(), "run_bash", map[string]interface{}{
		"command": "echo devilfish",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "devilfish")
}

func TestRegisterToolsFromFile_BashSkillRejectsDisallowedCommand(t *testing.T) {
	tmpDir := t.TempDir()
	toolsPath := filepath.Join(tmpDir, "tools.yaml")

	content := `
tools:
  - type: bash
    name: run_bash
    allowed_commands:
      - echo
`
	require.NoError(t, os.WriteFile(toolsPath, []byte(content), 0o644))

	registry := NewRegistry()
	require.NoError(t, RegisterToolsFromFile(registry, toolsPath))

	_, err := registry.ExecuteSkill(context.Background(), "run_bash", map[string]interface{}{
		"command": "pwd",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in allowed_commands")
}

func TestRegisterToolsFromFile_WithInvalidMarkdownReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	toolsPath := filepath.Join(tmpDir, "tools.md")
	require.NoError(t, os.WriteFile(toolsPath, []byte("# no fenced blocks"), 0o644))

	registry := NewRegistry()
	err := RegisterToolsFromFile(registry, toolsPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid fenced")
}
