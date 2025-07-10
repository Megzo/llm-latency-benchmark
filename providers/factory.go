package providers

import (
	"fmt"
	"sync"
)

// ProviderFactory manages provider creation and caching
type ProviderFactory struct {
	configs map[string]interface{}
	providers map[string]Provider
	mutex   sync.RWMutex
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{
		configs:   make(map[string]interface{}),
		providers: make(map[string]Provider),
	}
}

// RegisterConfig registers configuration for a provider
func (f *ProviderFactory) RegisterConfig(providerName string, config interface{}) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.configs[providerName] = config
}

// GetProvider returns a provider instance, creating it if necessary
func (f *ProviderFactory) GetProvider(providerName string) (Provider, error) {
	f.mutex.RLock()
	if provider, exists := f.providers[providerName]; exists {
		f.mutex.RUnlock()
		return provider, nil
	}
	f.mutex.RUnlock()

	f.mutex.Lock()
	defer f.mutex.Unlock()

	// Double-check after acquiring write lock
	if provider, exists := f.providers[providerName]; exists {
		return provider, nil
	}

	// Create new provider
	provider, err := f.createProvider(providerName)
	if err != nil {
		return nil, err
	}

	f.providers[providerName] = provider
	return provider, nil
}

// createProvider creates a new provider instance based on the provider name
func (f *ProviderFactory) createProvider(providerName string) (Provider, error) {
	switch providerName {
	case "openai":
		config, ok := f.configs[providerName].(*OpenAIConfig)
		if !ok {
			return nil, &ConfigurationError{
				Field:   "openai_config",
				Message: "OpenAI configuration not found or invalid",
			}
		}
		return NewOpenAIProvider(config)

	case "groq":
		config, ok := f.configs[providerName].(*GroqConfig)
		if !ok {
			return nil, &ConfigurationError{
				Field:   "groq_config",
				Message: "Groq configuration not found or invalid",
			}
		}
		return NewGroqProvider(config)

	case "anthropic":
		// TODO: Implement when Anthropic provider is added
		return nil, &ConfigurationError{
			Field:   "anthropic_provider",
			Message: "Anthropic provider not yet implemented",
		}

	default:
		return nil, &ConfigurationError{
			Field:   "provider_name",
			Message: fmt.Sprintf("unknown provider: %s", providerName),
		}
	}
}

// ClearProviders clears all cached providers
func (f *ProviderFactory) ClearProviders() {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.providers = make(map[string]Provider)
}

// GetAvailableProviders returns a list of available provider names
func (f *ProviderFactory) GetAvailableProviders() []string {
	return []string{
		"openai",
		"groq",
		// "anthropic", // TODO: Add when implemented
	}
} 