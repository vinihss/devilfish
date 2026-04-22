package usecase

import (
	"context"
	"fmt"

	"devilfish/internal/domain/valueobject"
	"devilfish/internal/infra/logging"
	"devilfish/internal/ports/outbound"
)

// RouteMessageUseCase routes messages to appropriate AI providers.
// It selects the best provider based on availability and configuration.
type RouteMessageUseCase struct {
	providers map[string]outbound.AIProvider
	logger    logging.Logger
}

// NewRouteMessageUseCase creates a new RouteMessageUseCase instance.
//
// Parameters:
//   - providers: Map of provider name to AI provider implementation
//   - logger: The logger instance
//
// Returns a new RouteMessageUseCase instance.
func NewRouteMessageUseCase(
	providers map[string]outbound.AIProvider,
	logger logging.Logger,
) *RouteMessageUseCase {
	return &RouteMessageUseCase{
		providers: providers,
		logger:    logger,
	}
}

// RouteInput represents the input for routing a message.
type RouteInput struct {
	UserID            string // User identifier
	Message           string // Message content
	PreferredProvider string // Preferred provider (optional)
	Channel           string // Channel name
}

// RouteOutput represents the output from routing a message.
type RouteOutput struct {
	Provider     *valueobject.Provider
	ProviderName string
	Model        string
	Available    bool
}

// Route selects an appropriate AI provider for the message.
//
// Parameters:
//   - ctx: The context
//   - input: The route input
//
// Returns the route output with selected provider or an error.
func (uc *RouteMessageUseCase) Route(ctx context.Context, input *RouteInput) (*RouteOutput, error) {
	if err := uc.validateRouteInput(input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	uc.logger.With(map[string]interface{}{
		"user_id":            input.UserID,
		"preferred_provider": input.PreferredProvider,
		"channel":            input.Channel,
	}).Info("routing message")

	// Try preferred provider first
	if input.PreferredProvider != "" {
		provider, ok := uc.providers[input.PreferredProvider]
		if ok && provider.IsAvailable() {
			uc.logger.With(map[string]interface{}{
				"provider": input.PreferredProvider,
			}).Info("using preferred provider")
			return &RouteOutput{
				Provider: &valueobject.Provider{
					Name:     provider.Name(),
					Model:    uc.getDefaultModel(provider.Name()),
					Endpoint: "",
				},
				ProviderName: provider.Name(),
				Model:        uc.getDefaultModel(provider.Name()),
				Available:    true,
			}, nil
		}
		uc.logger.With(map[string]interface{}{
			"provider": input.PreferredProvider,
		}).Warn("preferred provider not available")
	}

	// Find first available provider
	for name, provider := range uc.providers {
		if provider.IsAvailable() {
			uc.logger.With(map[string]interface{}{
				"provider": name,
			}).Info("routed to available provider")
			return &RouteOutput{
				Provider: &valueobject.Provider{
					Name:     name,
					Model:    uc.getDefaultModel(name),
					Endpoint: "",
				},
				ProviderName: name,
				Model:        uc.getDefaultModel(name),
				Available:    true,
			}, nil
		}
	}

	uc.logger.Error("no available provider found")
	return nil, fmt.Errorf("no available AI provider found")
}

// RouteToProvider routes a message to a specific provider.
//
// Parameters:
//   - ctx: The context
//   - providerName: The name of the provider to route to
//   - input: The route input
//
// Returns the route output or an error.
func (uc *RouteMessageUseCase) RouteToProvider(ctx context.Context, providerName string, input *RouteInput) (*RouteOutput, error) {
	if err := uc.validateRouteInput(input); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	if providerName == "" {
		return nil, fmt.Errorf("provider name is required")
	}

	provider, ok := uc.providers[providerName]
	if !ok {
		uc.logger.With(map[string]interface{}{
			"provider": providerName,
		}).Error("provider not found")
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}

	available := provider.IsAvailable()
	if !available {
		uc.logger.With(map[string]interface{}{
			"provider": providerName,
		}).Warn("provider not available")
		return &RouteOutput{
			Provider: &valueobject.Provider{
				Name:     provider.Name(),
				Model:    uc.getDefaultModel(providerName),
				Endpoint: "",
			},
			ProviderName: provider.Name(),
			Model:        uc.getDefaultModel(providerName),
			Available:    false,
		}, nil
	}

	uc.logger.With(map[string]interface{}{
		"provider": providerName,
	}).Info("routed to specific provider")

	return &RouteOutput{
		Provider: &valueobject.Provider{
			Name:     provider.Name(),
			Model:    uc.getDefaultModel(providerName),
			Endpoint: "",
		},
		ProviderName: provider.Name(),
		Model:        uc.getDefaultModel(providerName),
		Available:    true,
	}, nil
}

// GetAvailableProviders returns all available providers.
//
// Returns a slice of provider information.
func (uc *RouteMessageUseCase) GetAvailableProviders() []*valueobject.Provider {
	providers := make([]*valueobject.Provider, 0)

	for name := range uc.providers {
		providers = append(providers, &valueobject.Provider{
			Name:     name,
			Model:    uc.getDefaultModel(name),
			Endpoint: "",
		})
	}

	return providers
}

// GetProviderNames returns all registered provider names.
//
// Returns a slice of provider names.
func (uc *RouteMessageUseCase) GetProviderNames() []string {
	names := make([]string, 0, len(uc.providers))
	for name := range uc.providers {
		names = append(names, name)
	}
	return names
}

// IsProviderAvailable checks if a specific provider is available.
//
// Parameters:
//   - providerName: The name of the provider
//
// Returns true if the provider is available.
func (uc *RouteMessageUseCase) IsProviderAvailable(providerName string) bool {
	provider, ok := uc.providers[providerName]
	if !ok {
		return false
	}
	return provider.IsAvailable()
}

// GetProvider returns a specific provider by name.
//
// Parameters:
//   - providerName: The name of the provider
//
// Returns the AI provider interface.
func (uc *RouteMessageUseCase) GetProvider(providerName string) (outbound.AIProvider, bool) {
	provider, ok := uc.providers[providerName]
	return provider, ok
}

// AddProvider adds a new provider to the routing.
//
// Parameters:
//   - name: The provider name
//   - provider: The AI provider implementation
func (uc *RouteMessageUseCase) AddProvider(name string, provider outbound.AIProvider) {
	uc.providers[name] = provider
	uc.logger.With(map[string]interface{}{
		"provider": name,
	}).Info("provider added to routing")
}

// RemoveProvider removes a provider from the routing.
//
// Parameters:
//   - name: The provider name to remove
func (uc *RouteMessageUseCase) RemoveProvider(name string) {
	delete(uc.providers, name)
	uc.logger.With(map[string]interface{}{
		"provider": name,
	}).Info("provider removed from routing")
}

// validateRouteInput validates the route input.
func (uc *RouteMessageUseCase) validateRouteInput(input *RouteInput) error {
	if input == nil {
		return fmt.Errorf("input is required")
	}
	if input.Message == "" {
		return fmt.Errorf("message is required")
	}
	if input.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	return nil
}

// getDefaultModel returns the default model for a provider.
func (uc *RouteMessageUseCase) getDefaultModel(providerName string) string {
	defaultModels := map[string]string{
		"openai": "gpt-4o-mini",
		"groq":   "llama-3-70b-8192",
		"gemini": "gemini-1.5-flash",
		"ollama": "llama3",
	}

	model, ok := defaultModels[providerName]
	if !ok {
		return "default"
	}
	return model
}
