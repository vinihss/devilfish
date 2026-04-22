package valueobject_test

import (
	"testing"

	"devilfish/internal/domain/valueobject"
)

func TestNewProvider_WithValidInput_ReturnsProvider(t *testing.T) {
	provider, err := valueobject.NewProvider("openai", "gpt-4", "https://api.openai.com/v1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected provider, got nil")
	}
	if provider.Name != "openai" {
		t.Errorf("expected Name openai, got %s", provider.Name)
	}
	if provider.Model != "gpt-4" {
		t.Errorf("expected Model gpt-4, got %s", provider.Model)
	}
	if provider.Endpoint != "https://api.openai.com/v1" {
		t.Errorf("expected Endpoint, got %s", provider.Endpoint)
	}
}

func TestNewProvider_NameIsLowercased(t *testing.T) {
	provider, err := valueobject.NewProvider("OPENAI", "gpt-4", "https://api.openai.com/v1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected provider, got nil")
	}
	if provider.Name != "openai" {
		t.Errorf("expected Name openai, got %s", provider.Name)
	}
}

func TestNewProvider_WithEmptyName_ReturnsError(t *testing.T) {
	provider, err := valueobject.NewProvider("", "gpt-4", "https://api.openai.com/v1")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if provider != nil {
		t.Errorf("expected nil provider, got %v", provider)
	}
}

func TestNewProvider_WithEmptyModel_ReturnsError(t *testing.T) {
	provider, err := valueobject.NewProvider("openai", "", "https://api.openai.com/v1")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if provider != nil {
		t.Errorf("expected nil provider, got %v", provider)
	}
}

func TestNewProvider_WithEmptyEndpoint_ReturnsError(t *testing.T) {
	provider, err := valueobject.NewProvider("openai", "gpt-4", "")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if provider != nil {
		t.Errorf("expected nil provider, got %v", provider)
	}
}

func TestProvider_Equals_WithEqualProviders_ReturnsTrue(t *testing.T) {
	p1, _ := valueobject.NewProvider("openai", "gpt-4", "https://api.openai.com/v1")
	p2, _ := valueobject.NewProvider("openai", "gpt-4", "https://api.openai.com/v1")

	if !p1.Equals(p2) {
		t.Error("expected providers to be equal")
	}
}

func TestProvider_Equals_WithDifferentProviders_ReturnsFalse(t *testing.T) {
	p1, _ := valueobject.NewProvider("openai", "gpt-4", "https://api.openai.com/v1")
	p2, _ := valueobject.NewProvider("groq", "llama-3", "https://api.groq.com/v1")

	if p1.Equals(p2) {
		t.Error("expected providers to not be equal")
	}
}

func TestProvider_Equals_WithNilProvider_ReturnsFalse(t *testing.T) {
	p1, _ := valueobject.NewProvider("openai", "gpt-4", "https://api.openai.com/v1")

	if p1.Equals(nil) {
		t.Error("expected comparison to return false")
	}
}

func TestProvider_Equals_WhenNilCompared_ReturnsTrue(t *testing.T) {
	var p *valueobject.Provider

	if !p.Equals(nil) {
		t.Error("expected comparison to return true")
	}
}

func TestProvider_Equals_BothNil_ReturnsTrue(t *testing.T) {
	var p1, p2 *valueobject.Provider

	if !p1.Equals(p2) {
		t.Error("expected comparison to return true")
	}
}

func TestProvider_String_ReturnsFormatNameModel(t *testing.T) {
	provider, _ := valueobject.NewProvider("openai", "gpt-4", "https://api.openai.com/v1")

	if provider.String() != "openai/gpt-4" {
		t.Errorf("expected openai/gpt-4, got %s", provider.String())
	}
}

func TestProvider_String_WithNilProvider_ReturnsEmpty(t *testing.T) {
	var provider *valueobject.Provider

	result := provider.String()
	if result != "" {
		t.Errorf("expected empty string, got %s", result)
	}
}

func TestProvider_IsValid_WithCompleteData_ReturnsTrue(t *testing.T) {
	provider, _ := valueobject.NewProvider("openai", "gpt-4", "https://api.openai.com/v1")

	if !provider.IsValid() {
		t.Error("expected provider to be valid")
	}
}

func TestProvider_IsValid_WithNilProvider_ReturnsFalse(t *testing.T) {
	var provider *valueobject.Provider

	if provider.IsValid() {
		t.Error("expected provider to not be valid")
	}
}

func TestProvider_IsValid_WithEmptyName_ReturnsFalse(t *testing.T) {
	provider := &valueobject.Provider{
		Name:     "",
		Model:    "gpt-4",
		Endpoint: "https://api.openai.com/v1",
	}

	if provider.IsValid() {
		t.Error("expected provider to not be valid")
	}
}

func TestProvider_WithAPIKey_ReturnsCopyWithAPIKey(t *testing.T) {
	provider, _ := valueobject.NewProvider("openai", "gpt-4", "https://api.openai.com/v1")

	result := provider.WithAPIKey("sk-secret-key")

	if result.APIKey != "sk-secret-key" {
		t.Errorf("expected API key, got %s", result.APIKey)
	}
	if result.Name != provider.Name {
		t.Errorf("expected Name %s, got %s", provider.Name, result.Name)
	}
	if result.Model != provider.Model {
		t.Errorf("expected Model %s, got %s", provider.Model, result.Model)
	}
	if provider.APIKey != "" {
		t.Error("expected original API key to be empty")
	}
}

func TestProvider_WithAPIKey_WithNilProvider_ReturnsNil(t *testing.T) {
	var provider *valueobject.Provider

	result := provider.WithAPIKey("sk-secret-key")

	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestProvider_GetName_ReturnsName(t *testing.T) {
	provider, _ := valueobject.NewProvider("openai", "gpt-4", "https://api.openai.com/v1")

	if provider.GetName() != "openai" {
		t.Errorf("expected openai, got %s", provider.GetName())
	}
}

func TestProvider_GetName_WithNilProvider_ReturnsEmpty(t *testing.T) {
	var provider *valueobject.Provider

	if provider.GetName() != "" {
		t.Errorf("expected empty string, got %s", provider.GetName())
	}
}

func TestProvider_GetModel_ReturnsModel(t *testing.T) {
	provider, _ := valueobject.NewProvider("openai", "gpt-4", "https://api.openai.com/v1")

	if provider.GetModel() != "gpt-4" {
		t.Errorf("expected gpt-4, got %s", provider.GetModel())
	}
}

func TestProvider_GetModel_WithNilProvider_ReturnsEmpty(t *testing.T) {
	var provider *valueobject.Provider

	if provider.GetModel() != "" {
		t.Errorf("expected empty string, got %s", provider.GetModel())
	}
}

func TestProvider_GetEndpoint_ReturnsEndpoint(t *testing.T) {
	provider, _ := valueobject.NewProvider("openai", "gpt-4", "https://api.openai.com/v1")

	if provider.GetEndpoint() != "https://api.openai.com/v1" {
		t.Errorf("expected endpoint, got %s", provider.GetEndpoint())
	}
}

func TestProvider_GetEndpoint_WithNilProvider_ReturnsEmpty(t *testing.T) {
	var provider *valueobject.Provider

	if provider.GetEndpoint() != "" {
		t.Errorf("expected empty string, got %s", provider.GetEndpoint())
	}
}