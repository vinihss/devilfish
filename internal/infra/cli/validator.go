package cli

import (
	"fmt"
	"strings"
)

// ConfigValidator provides validation functions for configuration
type ConfigValidator struct{}

// NewConfigValidator creates a new validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// ValidateProviderAPIKey validates an AI provider API key
func (v *ConfigValidator) ValidateProviderAPIKey(provider, apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	switch provider {
	case "openai":
		if !strings.HasPrefix(apiKey, "sk-") {
			return fmt.Errorf("invalid OpenAI API key format (should start with 'sk-')")
		}
	case "groq":
		if !strings.HasPrefix(apiKey, "gsk_") {
			return fmt.Errorf("invalid Groq API key format (should start with 'gsk_')")
		}
	case "gemini":
		if len(apiKey) < 10 {
			return fmt.Errorf("invalid Gemini API key format")
		}
	case "ollama":
		// Ollama might not have an API key for local
	}
	return nil
}

// ValidateBotToken validates a messaging bot token
func (v *ConfigValidator) ValidateBotToken(token string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}
	if len(token) < 10 {
		return fmt.Errorf("invalid token format")
	}
	return nil
}

// ValidateURL validates a URL
func (v *ConfigValidator) ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("invalid URL format (must start with http:// or https://)")
	}
	return nil
}