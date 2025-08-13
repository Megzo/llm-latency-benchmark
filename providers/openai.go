package providers

import (
    "bufio"
    "bytes"
    "context"
    "encoding/json"
    "io"
    "net/http"
    "strings"
    "time"

    "github.com/openai/openai-go/v2"
    "github.com/openai/openai-go/v2/option"
)

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	client openai.Client
	config *OpenAIConfig
}

// OpenAIConfig holds OpenAI-specific configuration
type OpenAIConfig struct {
	APIKey  string
	BaseURL string
}

// NewOpenAIProvider creates a new OpenAI provider instance
func NewOpenAIProvider(config *OpenAIConfig) (*OpenAIProvider, error) {
	if config.APIKey == "" {
		return nil, &ConfigurationError{
			Field:   "OPENAI_API_KEY",
			Message: "OpenAI API key is required",
		}
	}

	client := openai.NewClient(option.WithAPIKey(config.APIKey))

	return &OpenAIProvider{
		client: client,
		config: config,
	}, nil
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// StreamChat performs a streaming chat completion
func (p *OpenAIProvider) StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error) {
	responseChan := make(chan ChatResponse)

    // If arbitrary extra params provided, use direct HTTP to allow full passthrough
    if req.ExtraParams != nil && len(req.ExtraParams) > 0 {
        go p.streamChatDirect(ctx, req, responseChan)
        return responseChan, nil
    }

    // Build messages for OpenAI API (SDK path)
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
        if !requiresMaxCompletionTokens(req.Model) {
            chatReq.MaxTokens = openai.Int(int64(req.MaxTokens))
        }
    }
    if req.Temperature > 0 {
        if !disallowsSamplingParameters(req.Model) {
            chatReq.Temperature = openai.Float(req.Temperature)
        }
    }
    if req.TopP > 0 {
        if !disallowsSamplingParameters(req.Model) {
            chatReq.TopP = openai.Float(req.TopP)
        }
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
                    Provider: "openai",
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

// requiresMaxCompletionTokens returns true for models that reject the legacy
// "max_tokens" parameter on the Chat Completions API.
func requiresMaxCompletionTokens(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(m, "gpt-5") ||
		strings.HasPrefix(m, "gpt-4.1") ||
		strings.HasPrefix(m, "gpt-4o") ||
		strings.HasPrefix(m, "o3") ||
		strings.HasPrefix(m, "o4")
}

// disallowsSamplingParameters returns true for models that do not accept
// temperature/top_p overrides and require default values.
func disallowsSamplingParameters(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(m, "gpt-5") ||
		strings.HasPrefix(m, "gpt-4.1") ||
		strings.HasPrefix(m, "gpt-4o") ||
		strings.HasPrefix(m, "o3") ||
		strings.HasPrefix(m, "o4")
}

// streamChatDirect performs streaming chat using direct HTTP API with full parameter passthrough
func (p *OpenAIProvider) streamChatDirect(ctx context.Context, req ChatRequest, responseChan chan<- ChatResponse) {
    defer close(responseChan)

    baseURL := strings.TrimRight(p.getBaseURL(), "/")
    endpoint := baseURL + "/chat/completions"

    // Build messages array
    messages := []map[string]interface{}{}
    if strings.TrimSpace(req.SystemPrompt) != "" {
        messages = append(messages, map[string]interface{}{"role": "system", "content": req.SystemPrompt})
    }
    messages = append(messages, map[string]interface{}{"role": "user", "content": req.UserPrompt})

    // Base payload
    payloadMap := map[string]interface{}{
        "model":   req.Model,
        "messages": messages,
        "stream":  true,
    }

    // Standard params
    if req.MaxTokens > 0 && !requiresMaxCompletionTokens(req.Model) {
        payloadMap["max_tokens"] = req.MaxTokens
    }
    if req.Temperature > 0 && !disallowsSamplingParameters(req.Model) {
        payloadMap["temperature"] = req.Temperature
    }
    if req.TopP > 0 && !disallowsSamplingParameters(req.Model) {
        payloadMap["top_p"] = req.TopP
    }

    // Merge ExtraParams
    if req.ExtraParams != nil {
        for k, v := range req.ExtraParams {
            if k == "model" || k == "stream" || k == "messages" {
                continue
            }
            payloadMap[k] = v
        }
    }

    // Marshal
    body, err := json.Marshal(payloadMap)
    if err != nil {
        responseChan <- ChatResponse{IsComplete: true, Timestamp: time.Now(), Error: &ProviderError{Provider: p.Name(), Message: "failed to marshal request", Cause: err}}
        return
    }

    // HTTP request
    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
    if err != nil {
        responseChan <- ChatResponse{IsComplete: true, Timestamp: time.Now(), Error: &ProviderError{Provider: p.Name(), Message: "failed to create HTTP request", Cause: err}}
        return
    }
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)
    httpReq.Header.Set("Accept", "text/event-stream")

    client := &http.Client{}
    resp, err := client.Do(httpReq)
    if err != nil {
        responseChan <- ChatResponse{IsComplete: true, Timestamp: time.Now(), Error: &ProviderError{Provider: p.Name(), Message: "failed to make HTTP request", Cause: err}}
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        responseChan <- ChatResponse{IsComplete: true, Timestamp: time.Now(), Error: &ProviderError{Provider: p.Name(), Message: strings.TrimSpace(string(b))}}
        return
    }

    reader := bufio.NewReader(resp.Body)
    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            if err == io.EOF { break }
            responseChan <- ChatResponse{IsComplete: true, Timestamp: time.Now(), Error: &ProviderError{Provider: p.Name(), Message: "failed to read response stream", Cause: err}}
            return
        }
        line = strings.TrimSpace(line)
        if line == "" { continue }
        if strings.HasPrefix(line, "data: ") {
            data := strings.TrimPrefix(line, "data: ")
            if data == "[DONE]" { break }
            // Parse minimal fields from Chat API delta
            var s struct {
                Choices []struct {
                    Delta struct {
                        Content string `json:"content"`
                    } `json:"delta"`
                } `json:"choices"`
            }
            if err := json.Unmarshal([]byte(data), &s); err == nil {
                if len(s.Choices) > 0 {
                    if c := s.Choices[0].Delta.Content; c != "" {
                        responseChan <- ChatResponse{Content: c, IsComplete: false, Timestamp: time.Now()}
                    }
                }
            }
        }
    }
    responseChan <- ChatResponse{IsComplete: true, Timestamp: time.Now()}
}

func (p *OpenAIProvider) getBaseURL() string {
    if strings.TrimSpace(p.config.BaseURL) != "" {
        return strings.TrimRight(p.config.BaseURL, "/")
    }
    return "https://api.openai.com/v1"
}

// TokenCount returns the token counts for a response
// Note: OpenAI doesn't provide token counts in streaming responses
// This is a simplified implementation - in practice, you might want to
// use a tokenizer library for accurate counting
func (p *OpenAIProvider) TokenCount(response ChatResponse) (input, output, total int) {
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
func (p *OpenAIProvider) GetTokenCount(text string) int {
	// Rough estimation: ~4 characters per token for English text
	count := len(text) / 4
	if count < 1 {
		count = 1
	}
	return count
}

// ValidateRequest validates the chat request
func (p *OpenAIProvider) ValidateRequest(req ChatRequest) error {
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
func (p *OpenAIProvider) IsRetryableError(err error) bool {
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
		strings.Contains(err.Error(), "context deadline exceeded") {
		return true
	}

	return false
}

// GetRetryDelay calculates the delay before retrying
func (p *OpenAIProvider) GetRetryDelay(attempt int, err error) time.Duration {
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
