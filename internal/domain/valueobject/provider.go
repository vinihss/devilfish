package valueobject

import (
	"errors"
	"fmt"
	"strings"
)

// Provider represents an AI provider as a value object.
type Provider struct {
	Name     string // Provider name (e.g., "openai", "groq", "gemini")
	Model    string // Model name (e.g., "gpt-4", "llama-3")
	Endpoint string // API endpoint URL
	APIKey   string // API key (should be handled securely)
}

// ErrInvalidProvider is returned when provider data is invalid.
var ErrInvalidProvider = errors.New("invalid provider")

// ErrMissingField is returned when a required field is missing.
var ErrMissingField = errors.New("provider field is required")

// NewProvider creates a new Provider value object.
func NewProvider(name, model, endpoint string) (*Provider, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: name", ErrMissingField)
	}
	if model == "" {
		return nil, fmt.Errorf("%w: model", ErrMissingField)
	}
	if endpoint == "" {
		return nil, fmt.Errorf("%w: endpoint", ErrMissingField)
	}

	return &Provider{
		Name:     strings.ToLower(name),
		Model:    model,
		Endpoint: endpoint,
	}, nil
}

// Equals compares two Provider values for equality.
func (p *Provider) Equals(other *Provider) bool {
	if p == nil || other == nil {
		return p == other
	}
	return p.Name == other.Name &&
		p.Model == other.Model &&
		p.Endpoint == other.Endpoint
}

// String returns a string representation of the provider.
func (p *Provider) String() string {
	return p.Name + "/" + p.Model
}

// IsValid checks if the provider has valid configuration.
func (p *Provider) IsValid() bool {
	return p != nil &&
		p.Name != "" &&
		p.Model != "" &&
		p.Endpoint != ""
}

// WithAPIKey returns a copy of the provider with the API key set.
func (p *Provider) WithAPIKey(apiKey string) *Provider {
	if p == nil {
		return nil
	}
	// Return a copy to maintain immutability
	cp := *p
	cp.APIKey = apiKey
	return &cp
}

// GetName returns the provider name.
func (p *Provider) GetName() string {
	if p == nil {
		return ""
	}
	return p.Name
}

// GetModel returns the model name.
func (p *Provider) GetModel() string {
	if p == nil {
		return ""
	}
	return p.Model
}

// GetEndpoint returns the endpoint URL.
func (p *Provider) GetEndpoint() string {
	if p == nil {
		return ""
	}
	return p.Endpoint
}
