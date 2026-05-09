package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Environment variable pattern: ${VAR} or ${VAR:-default}
var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:-([^}]*))?\}`)
var markdownLinkPattern = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)

// Loader handles loading configuration from YAML files and environment variables.
type Loader struct {
	configPaths []string
}

// NewLoader creates a new config loader with the given config file paths.
func NewLoader(configPaths ...string) *Loader {
	return &Loader{
		configPaths: configPaths,
	}
}

// Load loads the configuration from YAML files and resolves environment variables.
func (l *Loader) Load() (*Config, error) {
	if len(l.configPaths) == 0 {
		return nil, fmt.Errorf("no config file path provided")
	}

	// Load and merge all config files
	var merged map[string]interface{}
	for _, path := range l.configPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
		}

		// Resolve environment variables in YAML content
		content := resolveEnvVars(string(data))

		var cfg map[string]interface{}
		if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
		}

		if err := resolveAIReferencesInMap(cfg, filepath.Dir(path)); err != nil {
			return nil, fmt.Errorf("failed to process config file %s: %w", path, err)
		}

		// Merge configurations
		merged = mergeConfigs(merged, cfg)
	}

	// Convert to Config struct
	data, err := yaml.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// LoadFromFile loads configuration from a single YAML file.
func LoadFromFile(path string) (*Config, error) {
	return NewLoader(path).Load()
}

// LoadFromFiles loads configuration from multiple YAML files (merged).
func LoadFromFiles(paths ...string) (*Config, error) {
	return NewLoader(paths...).Load()
}

// resolveEnvVars replaces ${VAR} and ${VAR:-default} patterns with environment variable values.
func resolveEnvVars(content string) string {
	return envVarPattern.ReplaceAllStringFunc(content, func(match string) string {
		parts := envVarPattern.FindStringSubmatch(match)
		if parts == nil {
			return match
		}

		varName := parts[1]
		defaultValue := parts[3]

		// Try to get from environment
		value := os.Getenv(varName)
		if value != "" {
			return value
		}

		// Use default value if provided
		if defaultValue != "" {
			return defaultValue
		}

		// Return empty string if no value found
		return ""
	})
}

// resolveSystemPromptReference resolves ai.system_prompt when it references a file.
// Supported formats are "file://<path>" and "@<path>".
func resolveSystemPromptReference(systemPrompt, baseDir string) (string, error) {
	value := strings.TrimSpace(systemPrompt)
	if value == "" {
		return value, nil
	}

	filePath, isReference := normalizeFileReference(value)
	if !isReference {
		return systemPrompt, nil
	}

	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(baseDir, filePath)
	}
	filePath = filepath.Clean(filePath)

	return resolvePromptFile(filePath, []string{}, map[string]int{})
}

func resolvePromptFile(filePath string, stack []string, visiting map[string]int) (string, error) {
	if idx, exists := visiting[filePath]; exists {
		chain := append(append([]string{}, stack[idx:]...), filePath)
		return "", fmt.Errorf("circular system prompt file reference detected: %s", strings.Join(chain, " -> "))
	}

	visiting[filePath] = len(stack)
	stack = append(stack, filePath)
	defer delete(visiting, filePath)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read system prompt file %s: %w", filePath, err)
	}

	content := string(data)
	if !isMarkdownFile(filePath) {
		return content, nil
	}

	return resolveMarkdownLinks(content, filepath.Dir(filePath), stack, visiting)
}

func resolveMarkdownLinks(content, currentDir string, stack []string, visiting map[string]int) (string, error) {
	matches := markdownLinkPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return content, nil
	}

	seen := make(map[string]struct{})
	blocks := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		target := normalizeMarkdownLinkTarget(match[1])
		if target == "" || isExternalLink(target) {
			continue
		}

		targetPath := target
		if !filepath.IsAbs(targetPath) {
			targetPath = filepath.Join(currentDir, targetPath)
		}
		targetPath = filepath.Clean(targetPath)

		if _, exists := seen[targetPath]; exists {
			continue
		}
		seen[targetPath] = struct{}{}

		refContent, err := resolvePromptFile(targetPath, stack, visiting)
		if err != nil {
			return "", fmt.Errorf("failed to resolve markdown link %q: %w", target, err)
		}

		blocks = append(blocks, fmt.Sprintf("### Referenced file: %s\n%s", target, refContent))
	}

	if len(blocks) == 0 {
		return content, nil
	}

	var sb strings.Builder
	sb.WriteString(content)
	sb.WriteString("\n\n---\nResolved markdown references:\n\n")
	sb.WriteString(strings.Join(blocks, "\n\n"))
	return sb.String(), nil
}

func normalizeFileReference(value string) (string, bool) {
	if path, ok := strings.CutPrefix(value, "file://"); ok {
		return path, true
	}
	if path, ok := strings.CutPrefix(value, "@"); ok {
		return path, true
	}
	return value, false
}

func normalizeMarkdownLinkTarget(target string) string {
	value := strings.TrimSpace(target)
	if value == "" {
		return ""
	}

	if strings.HasPrefix(value, "<") && strings.HasSuffix(value, ">") {
		value = strings.TrimPrefix(strings.TrimSuffix(value, ">"), "<")
	}

	if idx := strings.IndexAny(value, " \t"); idx >= 0 {
		value = value[:idx]
	}

	if idx := strings.Index(value, "#"); idx >= 0 {
		value = value[:idx]
	}

	return strings.TrimSpace(value)
}

func isExternalLink(target string) bool {
	lowered := strings.ToLower(strings.TrimSpace(target))
	return lowered == "" ||
		strings.HasPrefix(lowered, "#") ||
		strings.HasPrefix(lowered, "http://") ||
		strings.HasPrefix(lowered, "https://") ||
		strings.HasPrefix(lowered, "mailto:") ||
		strings.HasPrefix(lowered, "tel:") ||
		strings.Contains(lowered, "://")
}

func isMarkdownFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown"
}

func resolveToolsFileReference(toolsFile, baseDir string) (string, error) {
	value := strings.TrimSpace(toolsFile)
	if value == "" {
		return "", nil
	}

	path, isReference := normalizeFileReference(value)
	if !isReference {
		path = value
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to read tools file %s: %w", path, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("tools file path points to a directory: %s", path)
	}

	return path, nil
}

// resolveAIReferencesInMap resolves ai.system_prompt and ai.tools_file in a raw YAML map before merge.
func resolveAIReferencesInMap(cfg map[string]interface{}, baseDir string) error {
	aiRaw, ok := cfg["ai"]
	if !ok {
		return nil
	}

	ai, ok := aiRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	systemPromptRaw, ok := ai["system_prompt"]
	if ok {
		systemPrompt, ok := systemPromptRaw.(string)
		if ok {
			resolved, err := resolveSystemPromptReference(systemPrompt, baseDir)
			if err != nil {
				return err
			}
			ai["system_prompt"] = resolved
		}
	}

	toolsFileRaw, ok := ai["tools_file"]
	if ok {
		toolsFile, ok := toolsFileRaw.(string)
		if ok {
			resolvedPath, err := resolveToolsFileReference(toolsFile, baseDir)
			if err != nil {
				return err
			}
			ai["tools_file"] = resolvedPath
		}
	}

	return nil
}

// resolveSystemPromptInMap resolves ai.system_prompt in a raw YAML map before merge.
func resolveSystemPromptInMap(cfg map[string]interface{}, baseDir string) error {
	return resolveAIReferencesInMap(cfg, baseDir)
}

// mergeConfigs merges two configuration maps.
// Values from cfg2 take precedence over cfg1.
func mergeConfigs(cfg1, cfg2 map[string]interface{}) map[string]interface{} {
	if cfg1 == nil {
		return cfg2
	}
	if cfg2 == nil {
		return cfg1
	}

	result := make(map[string]interface{})
	for k, v1 := range cfg1 {
		result[k] = v1
	}
	for k, v2 := range cfg2 {
		v1, exists := result[k]
		if !exists {
			result[k] = v2
			continue
		}

		// Both have this key - check if both are maps to merge recursively
		m1, ok1 := v1.(map[string]interface{})
		m2, ok2 := v2.(map[string]interface{})
		if ok1 && ok2 {
			result[k] = mergeConfigs(m1, m2)
		} else {
			result[k] = v2
		}
	}

	return result
}

// Validate validates the configuration and returns an error if invalid.
func (c *Config) Validate() error {
	// Validate Server config
	if c.Server.Port < 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Server.WSPort < 0 || c.Server.WSPort > 65535 {
		return fmt.Errorf("invalid WebSocket port: %d", c.Server.WSPort)
	}
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0" // Default host
	}

	// Validate AI config
	if c.AI.DefaultProvider == "" && len(c.AI.Providers) > 0 {
		return fmt.Errorf("default provider not specified")
	}

	// Validate provider names
	providerNames := make(map[string]bool)
	for _, p := range c.AI.Providers {
		if p.Name == "" {
			return fmt.Errorf("provider name is required")
		}
		if providerNames[p.Name] {
			return fmt.Errorf("duplicate provider name: %s", p.Name)
		}
		providerNames[p.Name] = true
	}

	// Check that default provider exists
	if c.AI.DefaultProvider != "" && !providerNames[c.AI.DefaultProvider] {
		return fmt.Errorf("default provider %s not found in providers list", c.AI.DefaultProvider)
	}

	// Validate Messaging config
	if err := c.Messaging.Validate(); err != nil {
		return err
	}

	// Validate I18n config
	if err := c.I18n.Validate(); err != nil {
		return err
	}

	// Validate Logging config
	if err := c.Logging.Validate(); err != nil {
		return err
	}

	// Validate Security config
	if err := c.Security.Validate(); err != nil {
		return err
	}

	return nil
}

// Validate validates the Messaging configuration.
func (m *MessagingConfig) Validate() error {
	// This is now optional - WebSocket can work without messaging channels
	// At least one channel can be enabled, or none (just WebSocket)
	return nil
}

// Validate validates the I18n configuration.
func (i *I18nConfig) Validate() error {
	if i.DefaultLocale == "" {
		return fmt.Errorf("default locale is required")
	}
	if len(i.SupportedLocales) == 0 {
		return fmt.Errorf("at least one supported locale is required")
	}
	if i.FallbackLocale == "" {
		i.FallbackLocale = "en" // Default fallback
	}

	// Check that default locale is in supported locales
	found := false
	for _, loc := range i.SupportedLocales {
		if loc == i.DefaultLocale {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("default locale %s not in supported locales", i.DefaultLocale)
	}

	// Check that fallback locale is in supported locales
	found = false
	for _, loc := range i.SupportedLocales {
		if loc == i.FallbackLocale {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("fallback locale %s not in supported locales", i.FallbackLocale)
	}

	return nil
}

// Validate validates the Logging configuration.
func (l *LoggingConfig) Validate() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[l.Level] {
		return fmt.Errorf("invalid log level: %s", l.Level)
	}

	validFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validFormats[l.Format] {
		return fmt.Errorf("invalid log format: %s", l.Format)
	}

	validOutputs := map[string]bool{
		"stdout": true,
		"stderr": true,
	}
	if !validOutputs[l.Output] && !strings.HasPrefix(l.Output, "/") && !strings.HasPrefix(l.Output, "./") {
		return fmt.Errorf("invalid log output: %s", l.Output)
	}

	return nil
}

// Validate validates the Security configuration.
func (s *SecurityConfig) Validate() error {
	return s.RateLimit.Validate()
}

// Validate validates the RateLimit configuration.
func (r *RateLimitConfig) Validate() error {
	if r.Enabled {
		if r.RequestsPerMinute <= 0 {
			return fmt.Errorf("requests per minute must be greater than 0")
		}
		if r.Burst <= 0 {
			return fmt.Errorf("burst must be greater than 0")
		}
	}
	return nil
}

// GetConfigDir returns the directory containing the config file.
func GetConfigDir(configPath string) string {
	dir, _ := filepath.Split(configPath)
	return strings.TrimSuffix(dir, "/")
}

// Load is a convenience function that loads config from default location.
// It looks for config.yaml in the current directory.
func Load() (*Config, error) {
	// Try default paths
	defaultPaths := []string{
		"config.yaml",
		"configs/config.yaml",
		"/etc/devilfish/config.yaml",
	}

	var lastErr error
	for _, path := range defaultPaths {
		if _, err := os.Stat(path); err == nil {
			cfg, err := LoadFromFile(path)
			if err == nil {
				return cfg, nil
			}
			lastErr = err
			continue
		}
	}

	// If no file found, return default config
	if lastErr == nil {
		return DefaultConfig(), nil
	}

	return nil, lastErr
}
