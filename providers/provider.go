package providers

import (
	"context"
	"time"
)

// Provider defines the interface for LLM providers
type Provider interface {
	// Name returns the provider name (e.g., "openai", "groq", "anthropic")
	Name() string
	
	// StreamChat performs a streaming chat completion
	StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error)
	
	// TokenCount returns the token counts for a response
	TokenCount(response ChatResponse) (input, output, total int)
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model       string    `json:"model"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	UserPrompt  string    `json:"user_prompt"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
}

// ChatResponse represents a streaming chat response
type ChatResponse struct {
	Content     string    `json:"content"`
	IsComplete  bool      `json:"is_complete"`
	Timestamp   time.Time `json:"timestamp"`
	Error       error     `json:"error,omitempty"`
}

// BenchmarkResult holds the complete result of a benchmark run
type BenchmarkResult struct {
	Provider        string    `json:"provider"`
	Model           string    `json:"model"`
	PromptFile      string    `json:"prompt_file"`
	
	// Timing metrics
	StartTime       time.Time `json:"start_time"`
	FirstTokenTime  time.Time `json:"first_token_time"`
	EndTime         time.Time `json:"end_time"`
	TTFT            time.Duration `json:"ttft"`           // Time to first token
	TotalTime       time.Duration `json:"total_time"`     // Total response time
	
	// Token metrics
	InputTokens     int       `json:"input_tokens"`
	OutputTokens    int       `json:"output_tokens"`
	TotalTokens     int       `json:"total_tokens"`
	TokensPerSecond float64   `json:"tokens_per_second"`
	
	// Cost metrics
	Cost            float64   `json:"cost"`
	
	// Response content
	Response        string    `json:"response"`
	
	// Error information
	Error           error     `json:"error,omitempty"`
	Success         bool      `json:"success"`
}

// CalculateMetrics calculates derived metrics from the benchmark result
func (r *BenchmarkResult) CalculateMetrics() {
	if !r.FirstTokenTime.IsZero() {
		r.TTFT = r.FirstTokenTime.Sub(r.StartTime)
	}
	
	if !r.EndTime.IsZero() {
		r.TotalTime = r.EndTime.Sub(r.StartTime)
	}
	
	r.TotalTokens = r.InputTokens + r.OutputTokens
	
	if r.TotalTime > 0 && r.OutputTokens > 0 {
		r.TokensPerSecond = float64(r.OutputTokens) / r.TotalTime.Seconds()
	}
} 