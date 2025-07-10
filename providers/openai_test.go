package providers

import (
	"testing"
	"time"
)

func TestNewOpenAIProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *OpenAIConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &OpenAIConfig{
				APIKey:  "test-key",
				BaseURL: "https://api.openai.com/v1",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: &OpenAIConfig{
				APIKey:  "",
				BaseURL: "https://api.openai.com/v1",
			},
			wantErr: true,
		},
		{
			name: "custom base URL",
			config: &OpenAIConfig{
				APIKey:  "test-key",
				BaseURL: "https://custom.openai.com/v1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewOpenAIProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOpenAIProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewOpenAIProvider() returned nil provider when no error expected")
			}
			if !tt.wantErr && provider.Name() != "openai" {
				t.Errorf("NewOpenAIProvider() provider name = %s, want 'openai'", provider.Name())
			}
		})
	}
}

func TestOpenAIProvider_ValidateRequest(t *testing.T) {
	provider, err := NewOpenAIProvider(&OpenAIConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.openai.com/v1",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name    string
		request ChatRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: ChatRequest{
				Model:      "gpt-3.5-turbo",
				UserPrompt: "Hello, world!",
			},
			wantErr: false,
		},
		{
			name: "missing model",
			request: ChatRequest{
				Model:      "",
				UserPrompt: "Hello, world!",
			},
			wantErr: true,
		},
		{
			name: "missing user prompt",
			request: ChatRequest{
				Model:      "gpt-3.5-turbo",
				UserPrompt: "",
			},
			wantErr: true,
		},
		{
			name: "negative max tokens",
			request: ChatRequest{
				Model:      "gpt-3.5-turbo",
				UserPrompt: "Hello, world!",
				MaxTokens:  -1,
			},
			wantErr: true,
		},
		{
			name: "invalid temperature",
			request: ChatRequest{
				Model:       "gpt-3.5-turbo",
				UserPrompt:  "Hello, world!",
				Temperature: 3.0,
			},
			wantErr: true,
		},
		{
			name: "invalid top_p",
			request: ChatRequest{
				Model:      "gpt-3.5-turbo",
				UserPrompt: "Hello, world!",
				TopP:       1.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.ValidateRequest(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenAIProvider_TokenCount(t *testing.T) {
	provider, err := NewOpenAIProvider(&OpenAIConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.openai.com/v1",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name     string
		response ChatResponse
		want     int
	}{
		{
			name: "empty content",
			response: ChatResponse{
				Content: "",
			},
			want: 0,
		},
		{
			name: "short content",
			response: ChatResponse{
				Content: "Hello",
			},
			want: 1,
		},
		{
			name: "longer content",
			response: ChatResponse{
				Content: "This is a longer response with more tokens to count.",
			},
			want: 12, // 48 characters / 4 = 12 tokens
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, output, total := provider.TokenCount(tt.response)
			if output != tt.want {
				t.Errorf("TokenCount() output = %v, want %v", output, tt.want)
			}
			if total != tt.want {
				t.Errorf("TokenCount() total = %v, want %v", total, tt.want)
			}
		})
	}
}

func TestOpenAIProvider_IsRetryableError(t *testing.T) {
	provider, err := NewOpenAIProvider(&OpenAIConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.openai.com/v1",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name    string
		err     error
		want    bool
	}{
		{
			name:    "nil error",
			err:     nil,
			want:    false,
		},
		{
			name:    "rate limit error",
			err:     &ProviderError{Provider: "openai", Message: "rate_limit exceeded"},
			want:    true,
		},
		{
			name:    "429 error",
			err:     &ProviderError{Provider: "openai", Message: "429 too many requests"},
			want:    true,
		},
		{
			name:    "500 error",
			err:     &ProviderError{Provider: "openai", Message: "500 internal server error"},
			want:    true,
		},
		{
			name:    "timeout error",
			err:     &ProviderError{Provider: "openai", Message: "timeout"},
			want:    true,
		},
		{
			name:    "authentication error",
			err:     &ProviderError{Provider: "openai", Message: "401 unauthorized"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := provider.IsRetryableError(tt.err)
			if got != tt.want {
				t.Errorf("IsRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOpenAIProvider_GetRetryDelay(t *testing.T) {
	provider, err := NewOpenAIProvider(&OpenAIConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.openai.com/v1",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name    string
		attempt int
		wantMin time.Duration
		wantMax time.Duration
	}{
		{
			name:     "first attempt",
			attempt:  1,
			wantMin:  time.Second,
			wantMax:  2 * time.Second,
		},
		{
			name:     "second attempt",
			attempt:  2,
			wantMin:  4 * time.Second,
			wantMax:  5 * time.Second,
		},
		{
			name:     "high attempt",
			attempt:  10,
			wantMin:  30 * time.Second,
			wantMax:  31 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := provider.GetRetryDelay(tt.attempt, nil)
			if delay < tt.wantMin || delay > tt.wantMax {
				t.Errorf("GetRetryDelay() = %v, want between %v and %v", delay, tt.wantMin, tt.wantMax)
			}
		})
	}
} 