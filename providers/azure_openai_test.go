package providers

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAzureOpenAIProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *AzureOpenAIConfig
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: &AzureOpenAIConfig{
				Endpoint:       "https://test.openai.azure.com/",
				APIKey:         "test-key",
				APIVersion:     "2024-02-15-preview",
				DeploymentName: "test-deployment",
			},
			wantErr: false,
		},
		{
			name: "missing endpoint",
			config: &AzureOpenAIConfig{
				APIKey:         "test-key",
				APIVersion:     "2024-02-15-preview",
				DeploymentName: "test-deployment",
			},
			wantErr: true,
		},
		{
			name: "missing API key",
			config: &AzureOpenAIConfig{
				Endpoint:       "https://test.openai.azure.com/",
				APIVersion:     "2024-02-15-preview",
				DeploymentName: "test-deployment",
			},
			wantErr: true,
		},
		{
			name: "missing deployment name",
			config: &AzureOpenAIConfig{
				Endpoint:   "https://test.openai.azure.com/",
				APIKey:     "test-key",
				APIVersion: "2024-02-15-preview",
			},
			wantErr: true,
		},
		{
			name: "empty configuration",
			config: &AzureOpenAIConfig{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewAzureOpenAIProvider(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, "azure_openai", provider.Name())
			}
		})
	}
}

func TestAzureOpenAIProvider_Name(t *testing.T) {
	config := &AzureOpenAIConfig{
		Endpoint:       "https://test.openai.azure.com/",
		APIKey:         "test-key",
		APIVersion:     "2024-02-15-preview",
		DeploymentName: "test-deployment",
	}

	provider, err := NewAzureOpenAIProvider(config)
	require.NoError(t, err)
	assert.Equal(t, "azure_openai", provider.Name())
}

func TestAzureOpenAIProvider_ValidateRequest(t *testing.T) {
	config := &AzureOpenAIConfig{
		Endpoint:       "https://test.openai.azure.com/",
		APIKey:         "test-key",
		APIVersion:     "2024-02-15-preview",
		DeploymentName: "test-deployment",
	}

	provider, err := NewAzureOpenAIProvider(config)
	require.NoError(t, err)

	tests := []struct {
		name    string
		request ChatRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: ChatRequest{
				Model:      "gpt-4",
				UserPrompt: "Hello, world!",
			},
			wantErr: false,
		},
		{
			name: "missing model",
			request: ChatRequest{
				UserPrompt: "Hello, world!",
			},
			wantErr: true,
		},
		{
			name: "missing user prompt",
			request: ChatRequest{
				Model: "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "negative max tokens",
			request: ChatRequest{
				Model:      "gpt-4",
				UserPrompt: "Hello, world!",
				MaxTokens:  -1,
			},
			wantErr: true,
		},
		{
			name: "invalid temperature",
			request: ChatRequest{
				Model:       "gpt-4",
				UserPrompt:  "Hello, world!",
				Temperature: 3.0,
			},
			wantErr: true,
		},
		{
			name: "invalid top_p",
			request: ChatRequest{
				Model:      "gpt-4",
				UserPrompt: "Hello, world!",
				TopP:       1.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.ValidateRequest(tt.request)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAzureOpenAIProvider_TokenCount(t *testing.T) {
	config := &AzureOpenAIConfig{
		Endpoint:       "https://test.openai.azure.com/",
		APIKey:         "test-key",
		APIVersion:     "2024-02-15-preview",
		DeploymentName: "test-deployment",
	}

	provider, err := NewAzureOpenAIProvider(config)
	require.NoError(t, err)

	tests := []struct {
		name     string
		response ChatResponse
		wantInput int
		wantOutput int
		wantTotal int
	}{
		{
			name: "empty response",
			response: ChatResponse{
				Content: "",
			},
			wantInput:  0,
			wantOutput: 0,
			wantTotal:  0,
		},
		{
			name: "short response",
			response: ChatResponse{
				Content: "Hello",
			},
			wantInput:  0,
			wantOutput: 1,
			wantTotal:  1,
		},
		{
			name: "longer response",
			response: ChatResponse{
				Content: "This is a longer response with more tokens to count",
			},
			wantInput:  0,
			wantOutput: 11, // 44 characters / 4 = 11
			wantTotal:  11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, output, total := provider.TokenCount(tt.response)
			assert.Equal(t, tt.wantInput, input)
			assert.Equal(t, tt.wantOutput, output)
			assert.Equal(t, tt.wantTotal, total)
		})
	}
}

func TestAzureOpenAIProvider_GetTokenCount(t *testing.T) {
	config := &AzureOpenAIConfig{
		Endpoint:       "https://test.openai.azure.com/",
		APIKey:         "test-key",
		APIVersion:     "2024-02-15-preview",
		DeploymentName: "test-deployment",
	}

	provider, err := NewAzureOpenAIProvider(config)
	require.NoError(t, err)

	tests := []struct {
		name string
		text string
		want int
	}{
		{
			name: "empty text",
			text: "",
			want: 0,
		},
		{
			name: "short text",
			text: "Hello",
			want: 1,
		},
		{
			name: "longer text",
			text: "This is a longer text with more tokens to count",
			want: 9, // 36 characters / 4 = 9
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := provider.GetTokenCount(tt.text)
			assert.Equal(t, tt.want, count)
		})
	}
}

func TestAzureOpenAIProvider_IsRetryableError(t *testing.T) {
	config := &AzureOpenAIConfig{
		Endpoint:       "https://test.openai.azure.com/",
		APIKey:         "test-key",
		APIVersion:     "2024-02-15-preview",
		DeploymentName: "test-deployment",
	}

	provider, err := NewAzureOpenAIProvider(config)
	require.NoError(t, err)

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "rate limit error",
			err:  &ProviderError{Message: "rate_limit exceeded"},
			want: true,
		},
		{
			name: "server error",
			err:  &ProviderError{Message: "500 Internal Server Error"},
			want: true,
		},
		{
			name: "timeout error",
			err:  &ProviderError{Message: "context deadline exceeded"},
			want: true,
		},
		{
			name: "non-retryable error",
			err:  &ProviderError{Message: "invalid request"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.IsRetryableError(tt.err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestAzureOpenAIProvider_GetRetryDelay(t *testing.T) {
	config := &AzureOpenAIConfig{
		Endpoint:       "https://test.openai.azure.com/",
		APIKey:         "test-key",
		APIVersion:     "2024-02-15-preview",
		DeploymentName: "test-deployment",
	}

	provider, err := NewAzureOpenAIProvider(config)
	require.NoError(t, err)

	tests := []struct {
		name    string
		attempt int
		err     error
		wantMin time.Duration
		wantMax time.Duration
	}{
		{
			name:    "first attempt",
			attempt: 1,
			err:     &ProviderError{Message: "rate_limit"},
			wantMin: time.Second,
			wantMax: 2 * time.Second,
		},
		{
			name:    "second attempt",
			attempt: 2,
			err:     &ProviderError{Message: "500 error"},
			wantMin: 4 * time.Second,
			wantMax: 5 * time.Second,
		},
		{
			name:    "high attempt number",
			attempt: 10,
			err:     &ProviderError{Message: "timeout"},
			wantMin: 30 * time.Second,
			wantMax: 31 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := provider.GetRetryDelay(tt.attempt, tt.err)
			assert.GreaterOrEqual(t, delay, tt.wantMin)
			assert.LessOrEqual(t, delay, tt.wantMax)
		})
	}
}

// Integration test - only runs if AZURE_OPENAI_API_KEY is set
func TestAzureOpenAIProvider_Integration(t *testing.T) {
	apiKey := os.Getenv("AZURE_OPENAI_API_KEY")
	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	deploymentName := os.Getenv("AZURE_OPENAI_DEPLOYMENT_NAME")

	if apiKey == "" || endpoint == "" || deploymentName == "" {
		t.Skip("AZURE_OPENAI_API_KEY, AZURE_OPENAI_ENDPOINT, or AZURE_OPENAI_DEPLOYMENT_NAME not set")
	}

	config := &AzureOpenAIConfig{
		Endpoint:       endpoint,
		APIKey:         apiKey,
		APIVersion:     "2024-02-15-preview",
		DeploymentName: deploymentName,
	}

	provider, err := NewAzureOpenAIProvider(config)
	require.NoError(t, err)

	// Test streaming chat
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	request := ChatRequest{
		Model:      "gpt-4",
		UserPrompt: "Say hello in one word",
		MaxTokens:  10,
	}

	responseChan, err := provider.StreamChat(ctx, request)
	require.NoError(t, err)

	var response string
	var isComplete bool
	var responseError error

	for resp := range responseChan {
		if resp.Error != nil {
			responseError = resp.Error
			break
		}
		if resp.IsComplete {
			isComplete = true
			break
		}
		response += resp.Content
	}

	assert.NoError(t, responseError)
	assert.True(t, isComplete)
	assert.NotEmpty(t, response)
	assert.LessOrEqual(t, len(response), 50) // Should be short due to max tokens
} 