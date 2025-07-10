package providers

import (
	"context"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
)

// AnthropicProvider implements the Provider interface for Anthropic
type AnthropicProvider struct {
	client anthropic.Client
	config *AnthropicConfig
}

// AnthropicConfig holds Anthropic-specific configuration
type AnthropicConfig struct {
	APIKey  string
	BaseURL string
}

// NewAnthropicProvider creates a new Anthropic provider instance
func NewAnthropicProvider(config *AnthropicConfig) (*AnthropicProvider, error) {
	if config.APIKey == "" {
		return nil, &ConfigurationError{
			Field:   "ANTHROPIC_API_KEY",
			Message: "Anthropic API key is required",
		}
	}

	// Set default base URL if not provided
	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com"
	}

	client := anthropic.NewClient(
		option.WithAPIKey(config.APIKey),
		option.WithBaseURL(config.BaseURL),
	)

	return &AnthropicProvider{
		client: client,
		config: config,
	}, nil
}

// Name returns the provider name
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// StreamChat performs a streaming chat completion
func (p *AnthropicProvider) StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error) {
	responseChan := make(chan ChatResponse)

	// Build messages for Anthropic API
	messages := []anthropic.MessageParam{}
	
	// Add user message
	messages = append(messages, anthropic.NewUserMessage(
		anthropic.NewTextBlock(req.UserPrompt),
	))

	// Create the request parameters
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		Messages:  messages,
		MaxTokens: int64(req.MaxTokens),
	}
	
	if req.Temperature > 0 {
		params.Temperature = param.NewOpt(req.Temperature)
	}
	if req.TopP > 0 {
		params.TopP = param.NewOpt(req.TopP)
	}

	// Add system prompt if provided
	if req.SystemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: req.SystemPrompt},
		}
	}

	go func() {
		defer close(responseChan)
		
		// Create streaming completion
		stream := p.client.Messages.NewStreaming(ctx, params)
		
		message := anthropic.Message{}
		for stream.Next() {
			event := stream.Current()
			err := message.Accumulate(event)
			if err != nil {
				responseChan <- ChatResponse{
					Content:    "",
					IsComplete: true,
					Timestamp:  time.Now(),
					Error: &ProviderError{
						Provider: "anthropic",
						Message:  "failed to accumulate stream event",
						Cause:    err,
					},
				}
				return
			}
			
			// Handle different types of content
			switch eventVariant := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				switch deltaVariant := eventVariant.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					if deltaVariant.Text != "" {
						responseChan <- ChatResponse{
							Content:    deltaVariant.Text,
							IsComplete: false,
							Timestamp:  time.Now(),
						}
					}
				}
			case anthropic.MessageStopEvent:
				responseChan <- ChatResponse{
					Content:    "",
					IsComplete: true,
					Timestamp:  time.Now(),
				}
				return
			}
		}
		
		// Check for errors
		if err := stream.Err(); err != nil {
			responseChan <- ChatResponse{
				Content:    "",
				IsComplete: true,
				Timestamp:  time.Now(),
				Error: &ProviderError{
					Provider: "anthropic",
					Message:  "failed to receive stream response",
					Cause:    err,
				},
			}
			return
		}
		
		// Stream completed successfully
		responseChan <- ChatResponse{
			Content:    "",
			IsComplete: true,
			Timestamp:  time.Now(),
		}
	}()
	return responseChan, nil
}

// TokenCount returns the token counts for a response
// Note: Anthropic doesn't provide token counts in streaming responses
// This is a simplified implementation - in practice, you might want to
// use a tokenizer library for accurate counting
func (p *AnthropicProvider) TokenCount(response ChatResponse) (input, output, total int) {
	// This is a rough estimation - for production use, consider using
	// a proper tokenizer like tiktoken or similar
	if response.Content != "" {
		// Rough estimation: ~4 characters per token for English text
		output = len(response.Content) / 4
		if output < 1 {
			output = 1
		}
	}
	
	return 0, output, output
}

// GetTokenCount estimates token count for input text
// This is a simplified implementation - consider using a proper tokenizer
func (p *AnthropicProvider) GetTokenCount(text string) int {
	// Rough estimation: ~4 characters per token for English text
	count := len(text) / 4
	if count < 1 {
		count = 1
	}
	return count
}

// ValidateRequest validates the chat request
func (p *AnthropicProvider) ValidateRequest(req ChatRequest) error {
	if req.Model == "" {
		return &ValidationError{
			Field:   "model",
			Message: "model name is required",
		}
	}

	if req.UserPrompt == "" {
		return &ValidationError{
			Field:   "user_prompt",
			Message: "user prompt is required",
		}
	}

	if req.MaxTokens < 0 {
		return &ValidationError{
			Field:   "max_tokens",
			Message: "max_tokens must be non-negative",
		}
	}

	if req.Temperature < 0 || req.Temperature > 1 {
		return &ValidationError{
			Field:   "temperature",
			Message: "temperature must be between 0 and 1",
		}
	}

	if req.TopP < 0 || req.TopP > 1 {
		return &ValidationError{
			Field:   "top_p",
			Message: "top_p must be between 0 and 1",
		}
	}

	return nil
}

// IsRetryableError checks if an error is retryable
func (p *AnthropicProvider) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for rate limit errors
	if strings.Contains(err.Error(), "rate_limit") || 
	   strings.Contains(err.Error(), "429") {
		return true
	}

	// Check for server errors
	if strings.Contains(err.Error(), "500") ||
	   strings.Contains(err.Error(), "502") ||
	   strings.Contains(err.Error(), "503") ||
	   strings.Contains(err.Error(), "504") {
		return true
	}

	// Check for timeout errors
	if strings.Contains(err.Error(), "timeout") ||
	   strings.Contains(err.Error(), "deadline exceeded") {
		return true
	}

	// Check for connection errors
	if strings.Contains(err.Error(), "connection refused") ||
	   strings.Contains(err.Error(), "no route to host") ||
	   strings.Contains(err.Error(), "network is unreachable") {
		return true
	}

	return false
}

// GetRetryDelay calculates the delay before retrying
func (p *AnthropicProvider) GetRetryDelay(attempt int, err error) time.Duration {
	// Base delay with exponential backoff
	baseDelay := time.Duration(attempt*attempt) * time.Second
	
	// Cap at 30 seconds
	if baseDelay > 30*time.Second {
		baseDelay = 30 * time.Second
	}
	
	// Add jitter to prevent thundering herd
	jitter := time.Duration(attempt) * 100 * time.Millisecond
	
	return baseDelay + jitter
} 