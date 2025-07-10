package providers

import (
	"context"
	"strings"
	"time"

	"google.golang.org/genai"
)

// GeminiProvider implements the Provider interface for Google Gemini
type GeminiProvider struct {
	client *genai.Client
	config *GeminiConfig
}

// GeminiConfig holds Gemini-specific configuration
type GeminiConfig struct {
	APIKey string
	// Gemini supports both Gemini API and Vertex AI backends
	// The client will automatically detect which one to use based on the API key
}

// NewGeminiProvider creates a new Gemini provider instance
func NewGeminiProvider(config *GeminiConfig) (*GeminiProvider, error) {
	if config.APIKey == "" {
		return nil, &ConfigurationError{
			Field:   "GOOGLE_API_KEY",
			Message: "Google API key is required for Gemini",
		}
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return nil, &ProviderError{
			Provider: "gemini",
			Message:  "failed to create Gemini client",
			Cause:    err,
		}
	}

	return &GeminiProvider{
		client: client,
		config: config,
	}, nil
}

// Name returns the provider name
func (p *GeminiProvider) Name() string {
	return "gemini"
}

// StreamChat performs a streaming chat completion
func (p *GeminiProvider) StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error) {
	responseChan := make(chan ChatResponse)

	// Validate request
	if err := p.ValidateRequest(req); err != nil {
		return nil, err
	}

	go func() {
		defer close(responseChan)

		// Create chat configuration
		config := &genai.GenerateContentConfig{
			ThinkingConfig: &genai.ThinkingConfig{
				ThinkingBudget: genai.Ptr[int32](0), // Disable thinking mode for faster responses
			},
		}
		if req.Temperature > 0 {
			config.Temperature = genai.Ptr[float32](float32(req.Temperature))
		}
		if req.TopP > 0 {
			config.TopP = genai.Ptr[float32](float32(req.TopP))
		}

		// Create a new chat session
		chat, err := p.client.Chats.Create(ctx, req.Model, config, nil)
		if err != nil {
			responseChan <- ChatResponse{
				Content:    "",
				IsComplete: true,
				Timestamp:  time.Now(),
				Error: &ProviderError{
					Provider: "gemini",
					Message:  "failed to create chat session",
					Cause:    err,
				},
			}
			return
		}

		// Prepare the message content
		messageContent := req.UserPrompt
		if req.SystemPrompt != "" {
			// Gemini doesn't have a separate system prompt, so we prepend it
			messageContent = req.SystemPrompt + "\n\n" + req.UserPrompt
		}

		// Create the message part
		part := genai.Part{Text: messageContent}

		// Send message and stream response
		for result, err := range chat.SendMessageStream(ctx, part) {
			if err != nil {
				responseChan <- ChatResponse{
					Content:    "",
					IsComplete: true,
					Timestamp:  time.Now(),
					Error: &ProviderError{
						Provider: "gemini",
						Message:  "failed to receive stream response",
						Cause:    err,
					},
				}
				return
			}

			// Extract text content from the result
			text := result.Text()
			if text != "" {
				responseChan <- ChatResponse{
					Content:    text,
					IsComplete: false,
					Timestamp:  time.Now(),
				}
			}
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
// Note: Gemini doesn't provide token counts in streaming responses
// This is a simplified implementation - in practice, you might want to
// use a tokenizer library for accurate counting
func (p *GeminiProvider) TokenCount(response ChatResponse) (input, output, total int) {
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
func (p *GeminiProvider) GetTokenCount(text string) int {
	// Rough estimation: ~4 characters per token for English text
	count := len(text) / 4
	if count < 1 {
		count = 1
	}
	return count
}

// ValidateRequest validates the chat request
func (p *GeminiProvider) ValidateRequest(req ChatRequest) error {
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
func (p *GeminiProvider) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Check for rate limit errors
	if strings.Contains(errStr, "rate_limit") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "quota") {
		return true
	}

	// Check for server errors
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") {
		return true
	}

	// Check for network errors
	if strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "network") {
		return true
	}

	// Check for Gemini-specific retryable errors
	if strings.Contains(errStr, "internal") ||
		strings.Contains(errStr, "unavailable") {
		return true
	}

	return false
}

// GetRetryDelay returns the delay before retrying
func (p *GeminiProvider) GetRetryDelay(attempt int, err error) time.Duration {
	// Base delay with exponential backoff
	baseDelay := time.Second * time.Duration(1<<uint(attempt))
	
	// Cap the maximum delay at 30 seconds
	if baseDelay > 30*time.Second {
		baseDelay = 30 * time.Second
	}

	// Add jitter to prevent thundering herd
	jitter := time.Duration(attempt*100) * time.Millisecond
	return baseDelay + jitter
}

// GetBackendInfo returns information about which backend is being used
func (p *GeminiProvider) GetBackendInfo() string {
	if p.client == nil {
		return "unknown"
	}
	
	config := p.client.ClientConfig()
	if config.Backend == genai.BackendVertexAI {
		return "vertex_ai"
	}
	return "gemini_api"
} 