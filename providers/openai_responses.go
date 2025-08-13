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
)

// OpenAIResponsesProvider implements the Provider interface using OpenAI Responses API (v1/responses)
type OpenAIResponsesProvider struct {
	config *OpenAIConfig
}

// NewOpenAIResponsesProvider creates a new provider instance for the Responses API
func NewOpenAIResponsesProvider(config *OpenAIConfig) (*OpenAIResponsesProvider, error) {
	if config == nil || strings.TrimSpace(config.APIKey) == "" {
		return nil, &ConfigurationError{
			Field:   "OPENAI_API_KEY",
			Message: "OpenAI API key is required",
		}
	}
	return &OpenAIResponsesProvider{config: config}, nil
}

// Name returns the provider name
func (p *OpenAIResponsesProvider) Name() string {
	return "openai_responses"
}

// We build a flexible map payload to allow arbitrary parameters to pass through

// responsesStreamEvent is a partial model of SSE data lines for the Responses API
type responsesStreamEvent struct {
	Type  string  `json:"type"`
	Delta *string `json:"delta,omitempty"`
	// Some events may include a "message" or other fields when errors occur
	Message string `json:"message,omitempty"`
}

// StreamChat performs a streaming call using the Responses API
func (p *OpenAIResponsesProvider) StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error) {
	responseChan := make(chan ChatResponse)

	go func() {
		defer close(responseChan)

		// Determine base URL
		baseURL := strings.TrimRight(p.getBaseURL(), "/")
		endpoint := baseURL + "/responses"

		// Build input string. Responses API accepts string for simple use-cases.
		inputBuilder := strings.Builder{}
		if strings.TrimSpace(req.SystemPrompt) != "" {
			inputBuilder.WriteString(req.SystemPrompt)
			inputBuilder.WriteString("\n\n")
		}
		inputBuilder.WriteString(req.UserPrompt)

        // Build request body (flexible map to allow arbitrary parameters)
        payloadMap := map[string]interface{}{
            "model":  req.Model,
            "input":  inputBuilder.String(),
            "stream": true,
        }

        // Map parameters per model capability
        if req.MaxTokens > 0 {
            payloadMap["max_output_tokens"] = req.MaxTokens
        }
        if req.Temperature > 0 && !disallowsSamplingParameters(req.Model) {
            payloadMap["temperature"] = req.Temperature
        }
        if req.TopP > 0 && !disallowsSamplingParameters(req.Model) {
            payloadMap["top_p"] = req.TopP
        }

        // Merge arbitrary ExtraParams, allowing override of defaults
        if req.ExtraParams != nil {
            for k, v := range req.ExtraParams {
                // Protect required fields only if caller didn't set them explicitly
                if k == "model" || k == "stream" {
                    continue
                }
                // Allow overriding input if provided
                payloadMap[k] = v
            }
        }

        // Marshal to JSON
        payload, err := json.Marshal(payloadMap)
		if err != nil {
			responseChan <- ChatResponse{IsComplete: true, Timestamp: time.Now(), Error: &ProviderError{Provider: p.Name(), Message: "failed to marshal request", Cause: err}}
			return
		}

		// Prepare HTTP request
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			responseChan <- ChatResponse{IsComplete: true, Timestamp: time.Now(), Error: &ProviderError{Provider: p.Name(), Message: "failed to create HTTP request", Cause: err}}
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
        httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)
        httpReq.Header.Set("Accept", "text/event-stream")

		// Execute
		client := &http.Client{}
		resp, err := client.Do(httpReq)
		if err != nil {
			responseChan <- ChatResponse{IsComplete: true, Timestamp: time.Now(), Error: &ProviderError{Provider: p.Name(), Message: "failed to make HTTP request", Cause: err}}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			responseChan <- ChatResponse{IsComplete: true, Timestamp: time.Now(), Error: &ProviderError{Provider: p.Name(), Message: strings.TrimSpace(string(body))}}
			return
		}

		// Parse SSE stream (data: {json}) lines
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				responseChan <- ChatResponse{IsComplete: true, Timestamp: time.Now(), Error: &ProviderError{Provider: p.Name(), Message: "failed to read response stream", Cause: err}}
				return
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					break
				}

				var event responsesStreamEvent
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					// Skip malformed JSON lines
					continue
				}

				// Emit deltas for textual output
				if strings.HasSuffix(event.Type, "output_text.delta") && event.Delta != nil && *event.Delta != "" {
					responseChan <- ChatResponse{Content: *event.Delta, IsComplete: false, Timestamp: time.Now()}
				}

				// If there's an error-type event, surface it
				if strings.Contains(event.Type, "error") && event.Message != "" {
					responseChan <- ChatResponse{IsComplete: true, Timestamp: time.Now(), Error: &ProviderError{Provider: p.Name(), Message: event.Message}}
					return
				}
			}
		}

		// Completed
		responseChan <- ChatResponse{IsComplete: true, Timestamp: time.Now()}
	}()

	return responseChan, nil
}

// TokenCount returns the token counts for a response
func (p *OpenAIResponsesProvider) TokenCount(response ChatResponse) (input, output, total int) {
	if response.Content != "" {
		output = len(response.Content) / 4
		if output < 1 {
			output = 1
		}
	}
	return 0, output, output
}

// GetTokenCount estimates token count for input text
func (p *OpenAIResponsesProvider) GetTokenCount(text string) int {
	count := len(text) / 4
	if count < 1 {
		count = 1
	}
	return count
}

// Helper to determine base URL for Responses API
func (p *OpenAIResponsesProvider) getBaseURL() string {
	if strings.TrimSpace(p.config.BaseURL) != "" {
		// Respect provided base URL as-is (it may already include /v1)
		return strings.TrimRight(p.config.BaseURL, "/")
	}
	return "https://api.openai.com/v1"
}


