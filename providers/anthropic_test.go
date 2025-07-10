package providers

import (
	"testing"
	"time"
)

func TestNewAnthropicProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *AnthropicConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &AnthropicConfig{
				APIKey:  "test-key",
				BaseURL: "https://api.anthropic.com",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: &AnthropicConfig{
				APIKey:  "",
				BaseURL: "https://api.anthropic.com",
			},
			wantErr: true,
		},
		{
			name: "empty base URL (should use default)",
			config: &AnthropicConfig{
				APIKey:  "test-key",
				BaseURL: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewAnthropicProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAnthropicProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewAnthropicProvider() returned nil provider when no error expected")
			}
			if !tt.wantErr && provider.Name() != "anthropic" {
				t.Errorf("NewAnthropicProvider() provider name = %s, want 'anthropic'", provider.Name())
			}
		})
	}
}

func TestAnthropicProvider_ValidateRequest(t *testing.T) {
	provider, err := NewAnthropicProvider(&AnthropicConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.anthropic.com",
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
				Model:      "claude-3-sonnet",
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
				Model:      "claude-3-sonnet",
				UserPrompt: "",
			},
			wantErr: true,
		},
		{
			name: "negative max tokens",
			request: ChatRequest{
				Model:      "claude-3-sonnet",
				UserPrompt: "Hello, world!",
				MaxTokens:  -1,
			},
			wantErr: true,
		},
		{
			name: "invalid temperature (too high)",
			request: ChatRequest{
				Model:       "claude-3-sonnet",
				UserPrompt:  "Hello, world!",
				Temperature: 1.5,
			},
			wantErr: true,
		},
		{
			name: "invalid temperature (negative)",
			request: ChatRequest{
				Model:       "claude-3-sonnet",
				UserPrompt:  "Hello, world!",
				Temperature: -0.1,
			},
			wantErr: true,
		},
		{
			name: "invalid top_p",
			request: ChatRequest{
				Model:      "claude-3-sonnet",
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

func TestAnthropicProvider_TokenCount(t *testing.T) {
	provider, err := NewAnthropicProvider(&AnthropicConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.anthropic.com",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name     string
		response ChatResponse
		wantInput, wantOutput, wantTotal int
	}{
		{
			name: "empty content",
			response: ChatResponse{
				Content: "",
			},
			wantInput:  0,
			wantOutput: 0,
			wantTotal:  0,
		},
		{
			name: "short content",
			response: ChatResponse{
				Content: "Hello",
			},
			wantInput:  0,
			wantOutput: 1,
			wantTotal:  1,
		},
		{
			name: "longer content",
			response: ChatResponse{
				Content: "This is a longer response with more tokens to count",
			},
			wantInput:  0,
			wantOutput: 12, // 48 chars / 4 = 12
			wantTotal:  12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, output, total := provider.TokenCount(tt.response)
			if input != tt.wantInput {
				t.Errorf("TokenCount() input = %v, want %v", input, tt.wantInput)
			}
			if output != tt.wantOutput {
				t.Errorf("TokenCount() output = %v, want %v", output, tt.wantOutput)
			}
			if total != tt.wantTotal {
				t.Errorf("TokenCount() total = %v, want %v", total, tt.wantTotal)
			}
		})
	}
}

func TestAnthropicProvider_GetTokenCount(t *testing.T) {
	provider, err := NewAnthropicProvider(&AnthropicConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.anthropic.com",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

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
			want: 11, // 44 chars / 4 = 11
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := provider.GetTokenCount(tt.text)
			if got != tt.want {
				t.Errorf("GetTokenCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnthropicProvider_IsRetryableError(t *testing.T) {
	provider, err := NewAnthropicProvider(&AnthropicConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.anthropic.com",
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
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "rate limit error",
			err:  &ProviderError{Provider: "anthropic", Message: "rate_limit exceeded"},
			want: true,
		},
		{
			name: "429 error",
			err:  &ProviderError{Provider: "anthropic", Message: "429 too many requests"},
			want: true,
		},
		{
			name: "server error",
			err:  &ProviderError{Provider: "anthropic", Message: "500 internal server error"},
			want: true,
		},
		{
			name: "timeout error",
			err:  &ProviderError{Provider: "anthropic", Message: "timeout after 30s"},
			want: true,
		},
		{
			name: "connection error",
			err:  &ProviderError{Provider: "anthropic", Message: "connection refused"},
			want: true,
		},
		{
			name: "non-retryable error",
			err:  &ProviderError{Provider: "anthropic", Message: "invalid API key"},
			want: false,
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

func TestAnthropicProvider_GetRetryDelay(t *testing.T) {
	provider, err := NewAnthropicProvider(&AnthropicConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.anthropic.com",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name    string
		attempt int
		err     error
		wantMin time.Duration
		wantMax time.Duration
	}{
		{
			name:     "first attempt",
			attempt:  1,
			err:      &ProviderError{Provider: "anthropic", Message: "rate_limit"},
			wantMin:  time.Second,
			wantMax:  2 * time.Second,
		},
		{
			name:     "second attempt",
			attempt:  2,
			err:      &ProviderError{Provider: "anthropic", Message: "500 error"},
			wantMin:  4 * time.Second,
			wantMax:  5 * time.Second,
		},
		{
			name:     "high attempt (should be capped)",
			attempt:  10,
			err:      &ProviderError{Provider: "anthropic", Message: "timeout"},
			wantMin:  30 * time.Second,
			wantMax:  31 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := provider.GetRetryDelay(tt.attempt, tt.err)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("GetRetryDelay() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
} 