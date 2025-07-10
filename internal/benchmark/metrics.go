package benchmark

import (
	"sync"
	"time"
)

// Metrics holds timing and performance metrics for a benchmark run
type Metrics struct {
	mu sync.RWMutex

	// Timing
	StartTime      time.Time
	FirstTokenTime time.Time
	EndTime        time.Time

	// Token tracking
	InputTokens  int
	OutputTokens int
	TotalTokens  int

	// Calculated metrics
	TTFT            time.Duration
	TotalTime       time.Duration
	TokensPerSecond float64

	// Cost
	Cost float64

	// Response content
	Response string

	// Error tracking
	Error   error
	Success bool
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		StartTime: time.Now(),
	}
}

// RecordFirstToken records the time of the first token
func (m *Metrics) RecordFirstToken() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.FirstTokenTime.IsZero() {
		m.FirstTokenTime = time.Now()
	}
}

// AddTokens adds tokens to the count
func (m *Metrics) AddTokens(input, output int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.InputTokens += input
	m.OutputTokens += output
}

// AddResponseContent appends content to the response
func (m *Metrics) AddResponseContent(content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.Response += content
}

// Complete marks the benchmark as complete and calculates final metrics
func (m *Metrics) Complete() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.EndTime = time.Now()
	m.Success = true
	
	// Calculate derived metrics
	if !m.FirstTokenTime.IsZero() {
		m.TTFT = m.FirstTokenTime.Sub(m.StartTime)
	}
	
	m.TotalTime = m.EndTime.Sub(m.StartTime)
	m.TotalTokens = m.InputTokens + m.OutputTokens
	
	if m.TotalTime > 0 && m.OutputTokens > 0 {
		m.TokensPerSecond = float64(m.OutputTokens) / m.TotalTime.Seconds()
	}
}

// SetError records an error and marks the benchmark as failed
func (m *Metrics) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.Error = err
	m.Success = false
	m.EndTime = time.Now()
}

// SetCost sets the cost for this benchmark run
func (m *Metrics) SetCost(cost float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.Cost = cost
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

// ToBenchmarkResult converts metrics to a BenchmarkResult
func (m *Metrics) ToBenchmarkResult(provider, model, promptFile string) BenchmarkResult {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return BenchmarkResult{
		Provider:        provider,
		Model:           model,
		PromptFile:      promptFile,
		StartTime:       m.StartTime,
		FirstTokenTime:  m.FirstTokenTime,
		EndTime:         m.EndTime,
		TTFT:            m.TTFT,
		TotalTime:       m.TotalTime,
		InputTokens:     m.InputTokens,
		OutputTokens:    m.OutputTokens,
		TotalTokens:     m.TotalTokens,
		TokensPerSecond: m.TokensPerSecond,
		Cost:            m.Cost,
		Response:        m.Response,
		Error:           m.Error,
		Success:         m.Success,
	}
}

// Summary holds aggregated metrics across multiple benchmark runs
type Summary struct {
	TotalRuns       int
	SuccessfulRuns  int
	FailedRuns      int
	
	// Timing statistics
	AvgTTFT         time.Duration
	AvgTotalTime    time.Duration
	MinTTFT         time.Duration
	MaxTTFT         time.Duration
	P95TTFT         time.Duration
	P99TTFT         time.Duration
	
	// Token statistics
	AvgTokensPerSecond float64
	TotalInputTokens   int
	TotalOutputTokens  int
	
	// Cost statistics
	TotalCost         float64
	AvgCostPerRun     float64
	
	// Error rate
	ErrorRate         float64
}

// CalculateSummary calculates summary statistics from a slice of results
func CalculateSummary(results []BenchmarkResult) Summary {
	if len(results) == 0 {
		return Summary{}
	}
	
	var summary Summary
	var ttftDurations []time.Duration
	var totalCost float64
	
	for _, result := range results {
		summary.TotalRuns++
		
		if result.Success {
			summary.SuccessfulRuns++
			ttftDurations = append(ttftDurations, result.TTFT)
			totalCost += result.Cost
			summary.TotalInputTokens += result.InputTokens
			summary.TotalOutputTokens += result.OutputTokens
		} else {
			summary.FailedRuns++
		}
	}
	
	// Calculate error rate
	summary.ErrorRate = float64(summary.FailedRuns) / float64(summary.TotalRuns)
	
	// Calculate timing statistics
	if len(ttftDurations) > 0 {
		summary.AvgTTFT = calculateAverageDuration(ttftDurations)
		summary.MinTTFT = calculateMinDuration(ttftDurations)
		summary.MaxTTFT = calculateMaxDuration(ttftDurations)
		summary.P95TTFT = calculatePercentileDuration(ttftDurations, 95)
		summary.P99TTFT = calculatePercentileDuration(ttftDurations, 99)
	}
	
	// Calculate cost statistics
	summary.TotalCost = totalCost
	if summary.SuccessfulRuns > 0 {
		summary.AvgCostPerRun = totalCost / float64(summary.SuccessfulRuns)
	}
	
	return summary
}

// Helper functions for duration calculations
func calculateAverageDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func calculateMinDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	min := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
	}
	return min
}

func calculateMaxDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}
	return max
}

func calculatePercentileDuration(durations []time.Duration, percentile int) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	// Sort durations (simplified - in production you'd want a proper sort)
	// For now, just return the average as a placeholder
	return calculateAverageDuration(durations)
} 