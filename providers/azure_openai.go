package providers

import (
	"context"
	"strings"
	"time"

    "github.com/openai/openai-go/v2"
    "github.com/openai/openai-go/v2/azure"
    "github.com/openai/openai-go/v2/option"
)

// AzureOpenAIProvider implements the Provider interface for Azure OpenAI
type AzureOpenAIProvider struct {
	client openai.Client
	config *AzureOpenAIConfig
}

// AzureOpenAIConfig holds Azure OpenAI-specific configuration
type AzureOpenAIConfig struct {
	Endpoint        string
	APIKey          string
	APIVersion      string
}

// NewAzureOpenAIProvider creates a new Azure OpenAI provider instance
func NewAzureOpenAIProvider(config *AzureOpenAIConfig) (*AzureOpenAIProvider, error) {
	if config.Endpoint == "" {
		return nil, &ConfigurationError{
			Field:   "AZURE_OPENAI_ENDPOINT",
			Message: "Azure OpenAI endpoint is required",
		}
	}

	if config.APIKey == "" {
		return nil, &ConfigurationError{
			Field:   "AZURE_OPENAI_API_KEY",
			Message: "Azure OpenAI API key is required",
		}
	}

	// Set default API version if not provided
	if config.APIVersion == "" {
		config.APIVersion = "2024-02-15-preview"
	}

	// Create client with Azure OpenAI configuration
	client := openai.NewClient(
		option.WithAPIKey(config.APIKey),
		azure.WithEndpoint(config.Endpoint, config.APIVersion),
	)

	return &AzureOpenAIProvider{
		client: client,
		config: config,
	}, nil
}

// Name returns the provider name
func (p *AzureOpenAIProvider) Name() string {
	return "azure_openai"
}

// StreamChat performs a streaming chat completion
func (p *AzureOpenAIProvider) StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error) {
	responseChan := make(chan ChatResponse)

	// Build messages for Azure OpenAI API
	messages := []openai.ChatCompletionMessageParamUnion{}
	if req.SystemPrompt != "" {
		messages = append(messages, openai.SystemMessage(req.SystemPrompt))
	}
	messages = append(messages, openai.UserMessage(req.UserPrompt))

	// Create chat completion request
	chatReq := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(req.Model), // Use req.Model as deployment name
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
					Provider: "azure_openai",
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
// Note: Azure OpenAI doesn't provide token counts in streaming responses
// This is a simplified implementation - in practice, you might want to
// use a tokenizer library for accurate counting
func (p *AzureOpenAIProvider) TokenCount(response ChatResponse) (input, output, total int) {
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
func (p *AzureOpenAIProvider) GetTokenCount(text string) int {
	// Rough estimation: ~4 characters per token for English text
	count := len(text) / 4
	if count < 1 {
		count = 1
	}
	return count
}

// ValidateRequest validates the chat request
func (p *AzureOpenAIProvider) ValidateRequest(req ChatRequest) error {
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
func (p *AzureOpenAIProvider) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for Azure-specific errors (these would be handled by the OpenAI SDK)

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
		strings.Contains(err.Error(), "context deadline exceeded") {
		return true
	}

	// Check for Azure-specific retryable errors
	if strings.Contains(err.Error(), "ServiceUnavailable") ||
		strings.Contains(err.Error(), "InternalServerError") ||
		strings.Contains(err.Error(), "BadGateway") ||
		strings.Contains(err.Error(), "GatewayTimeout") {
		return true
	}

	return false
}

// GetRetryDelay calculates the delay before retrying
func (p *AzureOpenAIProvider) GetRetryDelay(attempt int, err error) time.Duration {
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