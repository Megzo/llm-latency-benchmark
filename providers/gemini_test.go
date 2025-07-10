package providers

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewGeminiProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *GeminiConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &GeminiConfig{
				APIKey: "test-api-key",
			},
			wantErr: false,
		},
		{
			name: "empty API key",
			config: &GeminiConfig{
				APIKey: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGeminiProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGeminiProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewGeminiProvider() returned nil provider when no error expected")
			}
		})
	}
}

func TestGeminiProvider_Name(t *testing.T) {
	provider := &GeminiProvider{}
	if got := provider.Name(); got != "gemini" {
		t.Errorf("GeminiProvider.Name() = %v, want %v", got, "gemini")
	}
}

func TestGeminiProvider_ValidateRequest(t *testing.T) {
	provider := &GeminiProvider{}

	tests := []struct {
		name    string
		req     ChatRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: ChatRequest{
				Model:      "gemini-2.0-flash",
				UserPrompt: "Hello, world!",
			},
			wantErr: false,
		},
		{
			name: "missing model",
			req: ChatRequest{
				UserPrompt: "Hello, world!",
			},
			wantErr: true,
		},
		{
			name: "missing user prompt",
			req: ChatRequest{
				Model: "gemini-2.0-flash",
			},
			wantErr: true,
		},
		{
			name: "negative max tokens",
			req: ChatRequest{
				Model:      "gemini-2.0-flash",
				UserPrompt: "Hello, world!",
				MaxTokens:  -1,
			},
			wantErr: true,
		},
		{
			name: "invalid temperature",
			req: ChatRequest{
				Model:       "gemini-2.0-flash",
				UserPrompt:  "Hello, world!",
				Temperature: 3.0,
			},
			wantErr: true,
		},
		{
			name: "invalid top_p",
			req: ChatRequest{
				Model:      "gemini-2.0-flash",
				UserPrompt: "Hello, world!",
				TopP:       1.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.ValidateRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("GeminiProvider.ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGeminiProvider_TokenCount(t *testing.T) {
	provider := &GeminiProvider{}

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
				t.Errorf("GeminiProvider.TokenCount() input = %v, want %v", input, tt.wantInput)
			}
			if output != tt.wantOutput {
				t.Errorf("GeminiProvider.TokenCount() output = %v, want %v", output, tt.wantOutput)
			}
			if total != tt.wantTotal {
				t.Errorf("GeminiProvider.TokenCount() total = %v, want %v", total, tt.wantTotal)
			}
		})
	}
}

func TestGeminiProvider_GetTokenCount(t *testing.T) {
	provider := &GeminiProvider{}

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
			if got := provider.GetTokenCount(tt.text); got != tt.want {
				t.Errorf("GeminiProvider.GetTokenCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeminiProvider_IsRetryableError(t *testing.T) {
	provider := &GeminiProvider{}

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
			err:  fmt.Errorf("rate_limit exceeded"),
			want: true,
		},
		{
			name: "server error",
			err:  fmt.Errorf("500 internal server error"),
			want: true,
		},
		{
			name: "network error",
			err:  fmt.Errorf("connection timeout"),
			want: true,
		},
		{
			name: "non-retryable error",
			err:  fmt.Errorf("invalid request"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := provider.IsRetryableError(tt.err); got != tt.want {
				t.Errorf("GeminiProvider.IsRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeminiProvider_GetRetryDelay(t *testing.T) {
	provider := &GeminiProvider{}

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
			err:      fmt.Errorf("test error"),
			wantMin:  time.Second,
			wantMax:  2 * time.Second,
		},
		{
			name:     "second attempt",
			attempt:  2,
			err:      fmt.Errorf("test error"),
			wantMin:  2 * time.Second,
			wantMax:  3 * time.Second,
		},
		{
			name:     "high attempt number",
			attempt:  10,
			err:      fmt.Errorf("test error"),
			wantMin:  30 * time.Second,
			wantMax:  31 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := provider.GetRetryDelay(tt.attempt, tt.err)
			if delay < tt.wantMin || delay > tt.wantMax {
				t.Errorf("GeminiProvider.GetRetryDelay() = %v, want between %v and %v", delay, tt.wantMin, tt.wantMax)
			}
		})
	}
} 