package providers

import (
	"context"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// GroqProvider implements the Provider interface for Groq
type GroqProvider struct {
	client openai.Client
	config *GroqConfig
}

// GroqConfig holds Groq-specific configuration
type GroqConfig struct {
	APIKey  string
	BaseURL string
}

// NewGroqProvider creates a new Groq provider instance
func NewGroqProvider(config *GroqConfig) (*GroqProvider, error) {
	if config.APIKey == "" {
		return nil, &ConfigurationError{
			Field:   "GROQ_API_KEY",
			Message: "Groq API key is required",
		}
	}

	// Set default base URL if not provided
	if config.BaseURL == "" {
		config.BaseURL = "https://api.groq.com/openai/v1"
	}

	client := openai.NewClient(
		option.WithAPIKey(config.APIKey),
		option.WithBaseURL(config.BaseURL),
	)

	return &GroqProvider{
		client: client,
		config: config,
	}, nil
}

// Name returns the provider name
func (p *GroqProvider) Name() string {
	return "groq"
}

// StreamChat performs a streaming chat completion
func (p *GroqProvider) StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error) {
	responseChan := make(chan ChatResponse)

	// Build messages for Groq API (OpenAI-compatible)
	messages := []openai.ChatCompletionMessageParamUnion{}
	if req.SystemPrompt != "" {
		messages = append(messages, openai.SystemMessage(req.SystemPrompt))
	}
	messages = append(messages, openai.UserMessage(req.UserPrompt))

	chatReq := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(req.Model),
		Messages: messages,
	}
	if req.MaxTokens > 0 {
		chatReq.MaxTokens = openai.Int(int64(req.MaxTokens))
	}
	if req.Temperature > 0 {
		chatReq.Temperature = openai.Float(req.Temperature)
	}
	if req.TopP > 0 {
		chatReq.TopP = openai.Float(req.TopP)
	}

	go func() {
		defer close(responseChan)
		
		// Create streaming completion
		stream := p.client.Chat.Completions.NewStreaming(ctx, chatReq)
		
		for stream.Next() {
			resp := stream.Current()
			if len(resp.Choices) > 0 {
				choice := resp.Choices[0]
				if choice.Delta.Content != "" {
					responseChan <- ChatResponse{
						Content:    choice.Delta.Content,
						IsComplete: false,
						Timestamp:  time.Now(),
					}
				}
			}
		}
		
		// Check for errors
		if err := stream.Err(); err != nil {
			responseChan <- ChatResponse{
				Content:    "",
				IsComplete: true,
				Timestamp:  time.Now(),
				Error: &ProviderError{
					Provider: "groq",
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
// Note: Groq doesn't provide token counts in streaming responses
// This is a simplified implementation - in practice, you might want to
// use a tokenizer library for accurate counting
func (p *GroqProvider) TokenCount(response ChatResponse) (input, output, total int) {
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
func (p *GroqProvider) GetTokenCount(text string) int {
	// Rough estimation: ~4 characters per token for English text
	count := len(text) / 4
	if count < 1 {
		count = 1
	}
	return count
}

// ValidateRequest validates the chat request
func (p *GroqProvider) ValidateRequest(req ChatRequest) error {
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

	if req.Temperature < 0 || req.Temperature > 2 {
		return &ValidationError{
			Field:   "temperature",
			Message: "temperature must be between 0 and 2",
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
func (p *GroqProvider) IsRetryableError(err error) bool {
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
func (p *GroqProvider) GetRetryDelay(attempt int, err error) time.Duration {
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