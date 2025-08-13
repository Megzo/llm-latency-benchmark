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

	case "openai_responses":
		config, ok := f.configs["openai"].(*OpenAIConfig)
		if !ok {
			return nil, &ConfigurationError{
				Field:   "openai_config",
				Message: "OpenAI configuration not found or invalid",
			}
		}
		return NewOpenAIResponsesProvider(config)

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
		config, ok := f.configs[providerName].(*AnthropicConfig)
		if !ok {
			return nil, &ConfigurationError{
				Field:   "anthropic_config",
				Message: "Anthropic configuration not found or invalid",
			}
		}
		return NewAnthropicProvider(config)

	case "azure_openai":
		config, ok := f.configs[providerName].(*AzureOpenAIConfig)
		if !ok {
			return nil, &ConfigurationError{
				Field:   "azure_openai_config",
				Message: "Azure OpenAI configuration not found or invalid",
			}
		}
		return NewAzureOpenAIProvider(config)

	case "gemini":
		config, ok := f.configs[providerName].(*GeminiConfig)
		if !ok {
			return nil, &ConfigurationError{
				Field:   "gemini_config",
				Message: "Gemini configuration not found or invalid",
			}
		}
		return NewGeminiProvider(config)

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
        "openai_responses",
		"groq",
		"anthropic",
		"azure_openai",
		"gemini",
	}
} 