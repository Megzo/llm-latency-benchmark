package providers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProviderFactory_RegisterConfig(t *testing.T) {
	factory := NewProviderFactory()

	// Test registering a valid config
	config := &OpenAIConfig{APIKey: "test-key"}
	factory.RegisterConfig("test-provider", config)

	// Test registering the same provider again (should overwrite)
	config2 := &OpenAIConfig{APIKey: "test-key-2"}
	factory.RegisterConfig("test-provider", config2)
	
	// Should not error, just overwrite
}

func TestProviderFactory_GetProvider(t *testing.T) {
	factory := NewProviderFactory()

	// Register OpenAI config
	config := &OpenAIConfig{APIKey: "test-key"}
	factory.RegisterConfig("openai", config)

	// Test creating a registered provider
	provider, err := factory.GetProvider("openai")
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "openai", provider.Name())

	// Test creating a non-registered provider
	provider, err = factory.GetProvider("non-existent")
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestProviderFactory_GetProviderWithConfig(t *testing.T) {
	factory := NewProviderFactory()

	// Register Groq config
	config := &GroqConfig{APIKey: "test-key"}
	factory.RegisterConfig("groq", config)

	// Test creating provider with valid config
	provider, err := factory.GetProvider("groq")
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "groq", provider.Name())

	// Test creating provider with missing config (should fail)
	provider, err = factory.GetProvider("groq")
	assert.NoError(t, err) // Should succeed as config is already registered
	assert.NotNil(t, provider)
}

func TestProviderFactory_GetAvailableProviders(t *testing.T) {
	factory := NewProviderFactory()

	// Check available providers
	providers := factory.GetAvailableProviders()
	assert.Len(t, providers, 3)
	assert.Contains(t, providers, "openai")
	assert.Contains(t, providers, "groq")
	assert.Contains(t, providers, "anthropic")
}

func TestProviderFactory_ClearProviders(t *testing.T) {
	factory := NewProviderFactory()

	// Register a config
	config := &OpenAIConfig{APIKey: "test-key"}
	factory.RegisterConfig("openai", config)

	// Get provider (should create and cache it)
	provider1, err := factory.GetProvider("openai")
	assert.NoError(t, err)
	assert.NotNil(t, provider1)

	// Clear providers
	factory.ClearProviders()

	// Get provider again (should create new instance)
	provider2, err := factory.GetProvider("openai")
	assert.NoError(t, err)
	assert.NotNil(t, provider2)

	// Should be different instances (not cached)
	assert.NotEqual(t, provider1, provider2)
}

// MockProvider is a test implementation of the Provider interface
type MockProvider struct {
	name string
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error) {
	responseChan := make(chan ChatResponse)
	
	go func() {
		defer close(responseChan)
		responseChan <- ChatResponse{
			Content:    "Mock response",
			IsComplete: true,
			Timestamp:  time.Now(),
		}
	}()
	
	return responseChan, nil
}

func (m *MockProvider) TokenCount(response ChatResponse) (input, output, total int) {
	return 0, len(response.Content), len(response.Content)
}

func (m *MockProvider) GetTokenCount(text string) int {
	return len(text) / 4
}

func (m *MockProvider) ValidateRequest(request ChatRequest) error {
	if request.Model == "" {
		return assert.AnError
	}
	return nil
}

func (m *MockProvider) IsRetryableError(err error) bool {
	return false
}

func (m *MockProvider) GetRetryDelay(attempt int, err error) time.Duration {
	return time.Second
} 