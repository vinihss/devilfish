package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

var markdownCodeBlockPattern = regexp.MustCompile("(?s)```(?:yaml|yml|json)\\s*\\n(.*?)\\n```")

// ToolDefinitionFile is the structured tool definition root object.
type ToolDefinitionFile struct {
	Tools []ToolDefinition `yaml:"tools" json:"tools"`
}

// ToolDefinition describes a tool loaded from YAML/JSON/Markdown file content.
type ToolDefinition struct {
	Type            string   `yaml:"type" json:"type"`
	Name            string   `yaml:"name" json:"name"`
	Description     string   `yaml:"description" json:"description"`
	AllowedCommands []string `yaml:"allowed_commands" json:"allowed_commands"`
	WorkingDir      string   `yaml:"working_dir" json:"working_dir"`
	TimeoutSeconds  int      `yaml:"timeout_seconds" json:"timeout_seconds"`
	MaxOutputBytes  int      `yaml:"max_output_bytes" json:"max_output_bytes"`
}

// RegisterToolsFromFile loads tool definitions from file and registers supported tools.
// Supported formats: YAML, JSON, and Markdown with fenced yaml/json blocks.
func RegisterToolsFromFile(registry *Registry, path string) error {
	if registry == nil {
		return fmt.Errorf("skill registry is required")
	}
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("tool definition path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read tool definition file %s: %w", path, err)
	}

	definitions, err := parseToolDefinitions(path, data)
	if err != nil {
		return err
	}
	if len(definitions) == 0 {
		return fmt.Errorf("no tools found in definition file %s", path)
	}

	for _, def := range definitions {
		switch strings.ToLower(strings.TrimSpace(def.Type)) {
		case "bash":
			skill, err := NewBashCommandSkill(def)
			if err != nil {
				return err
			}
			registry.Register(skill)
		default:
			return fmt.Errorf("unsupported tool type %q in %s", def.Type, path)
		}
	}

	return nil
}

func parseToolDefinitions(path string, data []byte) ([]ToolDefinition, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".md" || ext == ".markdown" {
		return parseToolDefinitionsFromMarkdown(path, string(data))
	}
	return parseToolDefinitionsFromStructured(path, data)
}

func parseToolDefinitionsFromMarkdown(path, content string) ([]ToolDefinition, error) {
	blocks := markdownCodeBlockPattern.FindAllStringSubmatch(content, -1)
	for _, block := range blocks {
		if len(block) < 2 {
			continue
		}
		defs, err := parseToolDefinitionsFromStructured(path, []byte(block[1]))
		if err == nil && len(defs) > 0 {
			return defs, nil
		}
	}
	return nil, fmt.Errorf("no valid fenced yaml/json tool definitions found in markdown file %s", path)
}

func parseToolDefinitionsFromStructured(path string, data []byte) ([]ToolDefinition, error) {
	var file ToolDefinitionFile

	if err := yaml.Unmarshal(data, &file); err == nil && len(file.Tools) > 0 {
		return file.Tools, nil
	}

	if err := json.Unmarshal(data, &file); err == nil && len(file.Tools) > 0 {
		return file.Tools, nil
	}

	return nil, fmt.Errorf("failed to parse tool definitions from %s", path)
}

// BashCommandSkill executes configured bash commands safely through allowlist rules.
type BashCommandSkill struct {
	name            string
	description     string
	allowedCommands map[string]struct{}
	workingDir      string
	timeout         time.Duration
	maxOutputBytes  int
}

// NewBashCommandSkill creates a new bash skill from a tool definition.
func NewBashCommandSkill(def ToolDefinition) (*BashCommandSkill, error) {
	name := strings.TrimSpace(def.Name)
	if name == "" {
		return nil, fmt.Errorf("bash tool name is required")
	}

	description := strings.TrimSpace(def.Description)
	if description == "" {
		description = "Execute allowed bash commands."
	}

	allowed := make(map[string]struct{}, len(def.AllowedCommands))
	for _, command := range def.AllowedCommands {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}
		allowed[command] = struct{}{}
	}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("bash tool %q requires at least one allowed command", name)
	}

	timeout := 30 * time.Second
	if def.TimeoutSeconds > 0 {
		timeout = time.Duration(def.TimeoutSeconds) * time.Second
	}

	maxOutputBytes := 8192
	if def.MaxOutputBytes > 0 {
		maxOutputBytes = def.MaxOutputBytes
	}

	workingDir := strings.TrimSpace(def.WorkingDir)
	if workingDir != "" {
		workingDir = filepath.Clean(workingDir)
	}

	return &BashCommandSkill{
		name:            name,
		description:     description,
		allowedCommands: allowed,
		workingDir:      workingDir,
		timeout:         timeout,
		maxOutputBytes:  maxOutputBytes,
	}, nil
}

func (s *BashCommandSkill) Name() string {
	return s.name
}

func (s *BashCommandSkill) Description() string {
	return s.description
}

func (s *BashCommandSkill) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Command to execute (first token must be in allowed_commands).",
			},
			"timeout_seconds": map[string]interface{}{
				"type":        "number",
				"description": "Optional timeout override in seconds.",
			},
		},
		"required": []string{"command"},
	}
}

func (s *BashCommandSkill) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	command, ok := args["command"].(string)
	if !ok || strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("invalid or missing 'command' parameter: must be a non-empty string")
	}
	command = strings.TrimSpace(command)

	if containsUnsafeShellChars(command) {
		return "", fmt.Errorf("command contains unsafe shell characters")
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid command")
	}
	baseCommand := parts[0]
	if _, allowed := s.allowedCommands[baseCommand]; !allowed {
		return "", fmt.Errorf("command %q is not in allowed_commands", baseCommand)
	}

	timeout := s.timeout
	if override, ok := args["timeout_seconds"]; ok {
		if seconds, ok := toPositiveInt(override); ok {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, parts[0], parts[1:]...)
	if s.workingDir != "" {
		cmd.Dir = s.workingDir
	}

	output, err := cmd.CombinedOutput()
	outputText := string(output)
	if len(outputText) > s.maxOutputBytes {
		outputText = truncateUTF8(outputText, s.maxOutputBytes) + "\n... output truncated ..."
	}

	if execCtx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("bash command timed out after %s", timeout)
	}
	if err != nil {
		return "", fmt.Errorf("bash command failed: %w\noutput:\n%s", err, outputText)
	}
	if outputText == "" {
		outputText = "(no output)"
	}
	return outputText, nil
}

func containsUnsafeShellChars(command string) bool {
	return strings.ContainsAny(command, ";&|><$`\n\r(){}")
}

func toPositiveInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, v > 0
	case int64:
		return int(v), v > 0
	case float64:
		return int(v), v > 0
	default:
		return 0, false
	}
}

func truncateUTF8(text string, maxBytes int) string {
	if maxBytes <= 0 || len(text) <= maxBytes {
		return text
	}

	truncated := text[:maxBytes]
	for len(truncated) > 0 && !utf8.ValidString(truncated) {
		_, size := utf8.DecodeLastRuneInString(truncated)
		if size <= 0 || size > len(truncated) {
			size = 1
		}
		truncated = truncated[:len(truncated)-size]
	}
	return truncated
}

var _ Skill = (*BashCommandSkill)(nil)
