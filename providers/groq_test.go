package providers

import (
	"testing"
	"time"
)

func TestNewGroqProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *GroqConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &GroqConfig{
				APIKey:  "test-key",
				BaseURL: "https://api.groq.com/openai/v1",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: &GroqConfig{
				APIKey:  "",
				BaseURL: "https://api.groq.com/openai/v1",
			},
			wantErr: true,
		},
		{
			name: "empty base URL (should use default)",
			config: &GroqConfig{
				APIKey:  "test-key",
				BaseURL: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGroqProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGroqProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewGroqProvider() returned nil provider when no error expected")
			}
			if !tt.wantErr && provider.Name() != "groq" {
				t.Errorf("NewGroqProvider() provider name = %s, want 'groq'", provider.Name())
			}
		})
	}
}

func TestGroqProvider_ValidateRequest(t *testing.T) {
	provider, err := NewGroqProvider(&GroqConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.groq.com/openai/v1",
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
				Model:      "llama-3-70b",
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
				Model:      "llama-3-70b",
				UserPrompt: "",
			},
			wantErr: true,
		},
		{
			name: "negative max tokens",
			request: ChatRequest{
				Model:      "llama-3-70b",
				UserPrompt: "Hello, world!",
				MaxTokens:  -1,
			},
			wantErr: true,
		},
		{
			name: "invalid temperature",
			request: ChatRequest{
				Model:       "llama-3-70b",
				UserPrompt:  "Hello, world!",
				Temperature: 3.0,
			},
			wantErr: true,
		},
		{
			name: "invalid top_p",
			request: ChatRequest{
				Model:      "llama-3-70b",
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

func TestGroqProvider_TokenCount(t *testing.T) {
	provider, err := NewGroqProvider(&GroqConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.groq.com/openai/v1",
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
			wantOutput: 11, // 44 chars / 4 = 11
			wantTotal:  11,
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

func TestGroqProvider_GetTokenCount(t *testing.T) {
	provider, err := NewGroqProvider(&GroqConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.groq.com/openai/v1",
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
			want: 10, // 40 chars / 4 = 10
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

func TestGroqProvider_IsRetryableError(t *testing.T) {
	provider, err := NewGroqProvider(&GroqConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.groq.com/openai/v1",
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
			err:  &ProviderError{Provider: "groq", Message: "rate_limit exceeded"},
			want: true,
		},
		{
			name: "429 error",
			err:  &ProviderError{Provider: "groq", Message: "429 too many requests"},
			want: true,
		},
		{
			name: "server error",
			err:  &ProviderError{Provider: "groq", Message: "500 internal server error"},
			want: true,
		},
		{
			name: "timeout error",
			err:  &ProviderError{Provider: "groq", Message: "timeout after 30s"},
			want: true,
		},
		{
			name: "connection error",
			err:  &ProviderError{Provider: "groq", Message: "connection refused"},
			want: true,
		},
		{
			name: "non-retryable error",
			err:  &ProviderError{Provider: "groq", Message: "invalid API key"},
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

func TestGroqProvider_GetRetryDelay(t *testing.T) {
	provider, err := NewGroqProvider(&GroqConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.groq.com/openai/v1",
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
			err:      &ProviderError{Provider: "groq", Message: "rate_limit"},
			wantMin:  time.Second,
			wantMax:  2 * time.Second,
		},
		{
			name:     "second attempt",
			attempt:  2,
			err:      &ProviderError{Provider: "groq", Message: "500 error"},
			wantMin:  4 * time.Second,
			wantMax:  5 * time.Second,
		},
		{
			name:     "high attempt (should be capped)",
			attempt:  10,
			err:      &ProviderError{Provider: "groq", Message: "timeout"},
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