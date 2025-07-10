package providers

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderFactory_RegisterProvider(t *testing.T) {
	factory := NewProviderFactory()

	// Test registering a valid provider
	err := factory.RegisterProvider("test-provider", func(config interface{}) (Provider, error) {
		return &MockProvider{name: "test-provider"}, nil
	})
	assert.NoError(t, err)

	// Test registering the same provider again (should fail)
	err = factory.RegisterProvider("test-provider", func(config interface{}) (Provider, error) {
		return &MockProvider{name: "test-provider-2"}, nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestProviderFactory_CreateProvider(t *testing.T) {
	factory := NewProviderFactory()

	// Register a test provider
	err := factory.RegisterProvider("test-provider", func(config interface{}) (Provider, error) {
		return &MockProvider{name: "test-provider"}, nil
	})
	require.NoError(t, err)

	// Test creating a registered provider
	provider, err := factory.CreateProvider("test-provider", nil)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "test-provider", provider.Name())

	// Test creating a non-registered provider
	provider, err = factory.CreateProvider("non-existent", nil)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "not registered")
}

func TestProviderFactory_CreateProviderWithConfig(t *testing.T) {
	factory := NewProviderFactory()

	// Register a provider that uses config
	err := factory.RegisterProvider("config-provider", func(config interface{}) (Provider, error) {
		if config == nil {
			return nil, assert.AnError
		}
		cfg, ok := config.(string)
		if !ok {
			return nil, assert.AnError
		}
		return &MockProvider{name: cfg}, nil
	})
	require.NoError(t, err)

	// Test creating provider with valid config
	provider, err := factory.CreateProvider("config-provider", "test-config")
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "test-config", provider.Name())

	// Test creating provider with nil config (should fail)
	provider, err = factory.CreateProvider("config-provider", nil)
	assert.Error(t, err)
	assert.Nil(t, provider)
}

func TestProviderFactory_ListProviders(t *testing.T) {
	factory := NewProviderFactory()

	// Initially should be empty
	providers := factory.ListProviders()
	assert.Empty(t, providers)

	// Register some providers
	err := factory.RegisterProvider("provider1", func(config interface{}) (Provider, error) {
		return &MockProvider{name: "provider1"}, nil
	})
	require.NoError(t, err)

	err = factory.RegisterProvider("provider2", func(config interface{}) (Provider, error) {
		return &MockProvider{name: "provider2"}, nil
	})
	require.NoError(t, err)

	// Check that both providers are listed
	providers = factory.ListProviders()
	assert.Len(t, providers, 2)
	assert.Contains(t, providers, "provider1")
	assert.Contains(t, providers, "provider2")
}

func TestProviderFactory_ProviderExists(t *testing.T) {
	factory := NewProviderFactory()

	// Initially no providers exist
	assert.False(t, factory.ProviderExists("test-provider"))

	// Register a provider
	err := factory.RegisterProvider("test-provider", func(config interface{}) (Provider, error) {
		return &MockProvider{name: "test-provider"}, nil
	})
	require.NoError(t, err)

	// Now the provider should exist
	assert.True(t, factory.ProviderExists("test-provider"))
	assert.False(t, factory.ProviderExists("non-existent"))
}

func TestProviderFactory_ConcurrentAccess(t *testing.T) {
	factory := NewProviderFactory()

	// Test concurrent registration and creation
	done := make(chan bool, 2)

	// Goroutine 1: Register providers
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 10; i++ {
			name := fmt.Sprintf("provider-%d", i)
			err := factory.RegisterProvider(name, func(config interface{}) (Provider, error) {
				return &MockProvider{name: name}, nil
			})
			assert.NoError(t, err)
		}
	}()

	// Goroutine 2: Create providers
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 10; i++ {
			name := fmt.Sprintf("provider-%d", i)
			provider, err := factory.CreateProvider(name, nil)
			if err == nil {
				assert.NotNil(t, provider)
				assert.Equal(t, name, provider.Name())
			}
		}
	}()

	// Wait for both goroutines to complete
	<-done
	<-done
}

func TestProviderFactory_ErrorHandling(t *testing.T) {
	factory := NewProviderFactory()

	// Register a provider that always returns an error
	err := factory.RegisterProvider("error-provider", func(config interface{}) (Provider, error) {
		return nil, assert.AnError
	})
	require.NoError(t, err)

	// Test creating the provider (should return error)
	provider, err := factory.CreateProvider("error-provider", nil)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Equal(t, assert.AnError, err)
}

func TestProviderFactory_EmptyName(t *testing.T) {
	factory := NewProviderFactory()

	// Test registering provider with empty name
	err := factory.RegisterProvider("", func(config interface{}) (Provider, error) {
		return &MockProvider{name: "empty"}, nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty provider name")

	// Test creating provider with empty name
	provider, err := factory.CreateProvider("", nil)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "empty provider name")
}

func TestProviderFactory_NilConstructor(t *testing.T) {
	factory := NewProviderFactory()

	// Test registering provider with nil constructor
	err := factory.RegisterProvider("nil-provider", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil constructor")
}

// MockProvider is a test implementation of the Provider interface
type MockProvider struct {
	name string
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Chat(request ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{
		Content: "Mock response",
	}, nil
}

func (m *MockProvider) ValidateRequest(request ChatRequest) error {
	if request.Model == "" {
		return assert.AnError
	}
	return nil
}

func (m *MockProvider) TokenCount(response ChatResponse) (int, int, int) {
	return 0, len(response.Content), len(response.Content)
}

func (m *MockProvider) IsRetryableError(err error) bool {
	return false
}

func (m *MockProvider) GetRetryDelay(attempt int) time.Duration {
	return time.Second
} 