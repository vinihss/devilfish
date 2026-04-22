package i18n

import (
	"fmt"
	"os"
	"path/filepath"
)

// LocaleLoader handles loading translations from TOML files.
type LocaleLoader struct {
	localesPath string
}

// NewLocaleLoader creates a new LocaleLoader instance.
func NewLocaleLoader(localesPath string) *LocaleLoader {
	return &LocaleLoader{
		localesPath: localesPath,
	}
}

// LoadAll loads all locale files from the locales directory.
func (l *LocaleLoader) LoadAll() (map[string]map[string]string, error) {
	locales := make(map[string]map[string]string)

	// Load English first (used as fallback)
	en, err := l.loadLocale("en")
	if err != nil {
		return nil, fmt.Errorf("failed to load English locale: %w", err)
	}
	locales["en"] = en

	// Load other locales
	localesToLoad := []string{"pt", "es", "zh"}

	for _, locale := range localesToLoad {
		translation, err := l.loadLocale(locale)
		if err != nil {
			// If a locale fails to load, continue with others
			fmt.Printf("Warning: failed to load %s locale: %v\n", locale, err)
			continue
		}
		locales[locale] = translation
	}

	return locales, nil
}

// loadLocale loads a single locale TOML file.
func (l *LocaleLoader) loadLocale(locale string) (map[string]string, error) {
	filePath := filepath.Join(l.localesPath, locale+".toml")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse TOML content into map
	translations, err := parseTOML(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	return translations, nil
}

// parseTOML parses simple TOML key-value pairs into a map.
// This is a simple parser for flat TOML structures.
func parseTOML(content string) (map[string]string, error) {
	result := make(map[string]string)
	lines := splitLines(content)

	for _, line := range lines {
		line = trimSpaces(line)

		// Skip empty lines and comments
		if line == "" || startsWith(line, "#") {
			continue
		}

		// Parse key = "value" format
		if contains(line, "=") {
			parts := splitAt(line, "=")
			if len(parts) != 2 {
				continue
			}

			key := trimSpaces(parts[0])
			value := trimSpaces(parts[1])

			// Remove surrounding quotes if present
			value = stripQuotes(value)

			result[key] = value
		}
	}

	return result, nil
}

// splitLines splits content into lines.
func splitLines(content string) []string {
	var lines []string
	start := 0

	for i, r := range content {
		if r == '\n' {
			lines = append(lines, content[start:i])
			start = i + 1
		}
	}

	if start < len(content) {
		lines = append(lines, content[start:])
	}

	return lines
}

// contains checks if s contains substring.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// splitAt splits string at first occurrence of separator.
func splitAt(s, sep string) []string {
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			return []string{s[:i], s[i+len(sep):]}
		}
	}
	return []string{s, ""}
}

// trimSpaces removes leading and trailing spaces.
func trimSpaces(s string) string {
	start := 0
	end := len(s)

	for start < end && s[start] == ' ' {
		start++
	}

	for end > start && s[end-1] == ' ' {
		end--
	}

	return s[start:end]
}

// startsWith checks if s starts with prefix.
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// stripQuotes removes surrounding quotes from string.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// LoadTranslator loads translations and creates a new Translator with config.
func LoadTranslator(localesPath string, defaultLocale, fallbackLocale string) (*Translator, error) {
	loader := NewLocaleLoader(localesPath)
	locales, err := loader.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to load translations: %w", err)
	}

	t := &Translator{
		locales:        locales,
		locale:         defaultLocale,
		fallbackLocale: fallbackLocale,
	}

	return t, nil
}
