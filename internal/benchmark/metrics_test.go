package benchmark

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBenchmarkResult_CalculateCost(t *testing.T) {
	tests := []struct {
		name           string
		result         BenchmarkResult
		inputCostPer1K float64
		outputCostPer1K float64
		expectedCost   float64
	}{
		{
			name: "basic cost calculation",
			result: BenchmarkResult{
				InputTokens:  1000,
				OutputTokens: 500,
			},
			inputCostPer1K:  0.001,
			outputCostPer1K: 0.002,
			expectedCost:    0.002, // (1000 * 0.001) + (500 * 0.002) = 1 + 1 = 2
		},
		{
			name: "zero tokens",
			result: BenchmarkResult{
				InputTokens:  0,
				OutputTokens: 0,
			},
			inputCostPer1K:  0.001,
			outputCostPer1K: 0.002,
			expectedCost:    0.0,
		},
		{
			name: "high token count",
			result: BenchmarkResult{
				InputTokens:  5000,
				OutputTokens: 3000,
			},
			inputCostPer1K:  0.0001,
			outputCostPer1K: 0.0002,
			expectedCost:    0.0011, // (5000 * 0.0001) + (3000 * 0.0002) = 0.5 + 0.6 = 1.1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := tt.result.CalculateCost(tt.inputCostPer1K, tt.outputCostPer1K)
			assert.InDelta(t, tt.expectedCost, cost, 0.0001, "Cost calculation mismatch")
		})
	}
}

func TestBenchmarkResult_TokensPerSecond(t *testing.T) {
	tests := []struct {
		name           string
		result         BenchmarkResult
		expectedTPS    float64
	}{
		{
			name: "normal response time",
			result: BenchmarkResult{
				OutputTokens: 100,
				TotalTime:    2 * time.Second,
			},
			expectedTPS: 50.0, // 100 tokens / 2 seconds
		},
		{
			name: "fast response",
			result: BenchmarkResult{
				OutputTokens: 50,
				TotalTime:    500 * time.Millisecond,
			},
			expectedTPS: 100.0, // 50 tokens / 0.5 seconds
		},
		{
			name: "zero tokens",
			result: BenchmarkResult{
				OutputTokens: 0,
				TotalTime:    1 * time.Second,
			},
			expectedTPS: 0.0,
		},
		{
			name: "zero time",
			result: BenchmarkResult{
				OutputTokens: 100,
				TotalTime:    0,
			},
			expectedTPS: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tps := tt.result.TokensPerSecond()
			assert.InDelta(t, tt.expectedTPS, tps, 0.1, "Tokens per second calculation mismatch")
		})
	}
}

func TestBenchmarkResult_IsSuccessful(t *testing.T) {
	tests := []struct {
		name     string
		result   BenchmarkResult
		expected bool
	}{
		{
			name: "successful result",
			result: BenchmarkResult{
				Error: nil,
			},
			expected: true,
		},
		{
			name: "failed result",
			result: BenchmarkResult{
				Error: assert.AnError,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			success := tt.result.IsSuccessful()
			assert.Equal(t, tt.expected, success)
		})
	}
}

func TestBenchmarkSummary_CalculateStatistics(t *testing.T) {
	results := []BenchmarkResult{
		{
			Model:        "gpt-4o-mini",
			PromptName:   "test1",
			TTFT:         1 * time.Second,
			TotalTime:    5 * time.Second,
			InputTokens:  100,
			OutputTokens: 200,
			Cost:         0.001,
			Error:        nil,
		},
		{
			Model:        "gpt-4o-mini",
			PromptName:   "test2",
			TTFT:         2 * time.Second,
			TotalTime:    6 * time.Second,
			InputTokens:  150,
			OutputTokens: 250,
			Cost:         0.002,
			Error:        nil,
		},
		{
			Model:        "gpt-4o-mini",
			PromptName:   "test3",
			TTFT:         3 * time.Second,
			TotalTime:    7 * time.Second,
			InputTokens:  200,
			OutputTokens: 300,
			Cost:         0.003,
			Error:        assert.AnError, // This one should fail
		},
	}

	summary := NewBenchmarkSummary()
	for _, result := range results {
		summary.AddResult(result)
	}

	// Test basic statistics
	assert.Equal(t, 3, summary.TotalRuns)
	assert.Equal(t, 2, summary.SuccessfulRuns)
	assert.Equal(t, 1, summary.FailedRuns)
	assert.InDelta(t, 33.33, summary.ErrorRate, 0.1)

	// Test timing statistics
	assert.InDelta(t, 2.0, summary.AverageTTFT.Seconds(), 0.1) // (1+2+3)/3 = 2
	assert.InDelta(t, 6.0, summary.AverageTotalTime.Seconds(), 0.1) // (5+6+7)/3 = 6

	// Test cost statistics
	assert.InDelta(t, 0.002, summary.TotalCost, 0.0001) // Only successful runs count
	assert.InDelta(t, 0.001, summary.AverageCost, 0.0001) // 0.002 / 2 successful runs

	// Test token statistics
	assert.Equal(t, 250, summary.TotalInputTokens) // 100+150 (failed run doesn't count)
	assert.Equal(t, 450, summary.TotalOutputTokens) // 200+250
}

func TestBenchmarkSummary_EmptyResults(t *testing.T) {
	summary := NewBenchmarkSummary()

	assert.Equal(t, 0, summary.TotalRuns)
	assert.Equal(t, 0, summary.SuccessfulRuns)
	assert.Equal(t, 0, summary.FailedRuns)
	assert.Equal(t, 0.0, summary.ErrorRate)
	assert.Equal(t, 0.0, summary.TotalCost)
	assert.Equal(t, 0.0, summary.AverageCost)
	assert.Equal(t, 0, summary.TotalInputTokens)
	assert.Equal(t, 0, summary.TotalOutputTokens)
}

func TestBenchmarkSummary_AllFailed(t *testing.T) {
	results := []BenchmarkResult{
		{
			Model:      "gpt-4o-mini",
			PromptName: "test1",
			Error:      assert.AnError,
		},
		{
			Model:      "gpt-4o-mini",
			PromptName: "test2",
			Error:      assert.AnError,
		},
	}

	summary := NewBenchmarkSummary()
	for _, result := range results {
		summary.AddResult(result)
	}

	assert.Equal(t, 2, summary.TotalRuns)
	assert.Equal(t, 0, summary.SuccessfulRuns)
	assert.Equal(t, 2, summary.FailedRuns)
	assert.Equal(t, 100.0, summary.ErrorRate)
	assert.Equal(t, 0.0, summary.TotalCost)
	assert.Equal(t, 0.0, summary.AverageCost)
}

func TestBenchmarkSummary_Percentiles(t *testing.T) {
	// Create results with known values for percentile testing
	results := []BenchmarkResult{
		{TTFT: 1 * time.Second, TotalTime: 5 * time.Second},
		{TTFT: 2 * time.Second, TotalTime: 6 * time.Second},
		{TTFT: 3 * time.Second, TotalTime: 7 * time.Second},
		{TTFT: 4 * time.Second, TotalTime: 8 * time.Second},
		{TTFT: 5 * time.Second, TotalTime: 9 * time.Second},
	}

	summary := NewBenchmarkSummary()
	for _, result := range results {
		summary.AddResult(result)
	}

	// Test percentiles (these should be calculated from successful runs only)
	// For 5 values: p50 = 3rd value, p95 = 5th value, p99 = 5th value
	assert.InDelta(t, 3.0, summary.TTFTPercentiles[50].Seconds(), 0.1)
	assert.InDelta(t, 5.0, summary.TTFTPercentiles[95].Seconds(), 0.1)
	assert.InDelta(t, 5.0, summary.TTFTPercentiles[99].Seconds(), 0.1)

	assert.InDelta(t, 7.0, summary.TotalTimePercentiles[50].Seconds(), 0.1)
	assert.InDelta(t, 9.0, summary.TotalTimePercentiles[95].Seconds(), 0.1)
	assert.InDelta(t, 9.0, summary.TotalTimePercentiles[99].Seconds(), 0.1)
}

func TestBenchmarkResult_String(t *testing.T) {
	result := BenchmarkResult{
		Model:        "gpt-4o-mini",
		PromptName:   "test-prompt",
		TTFT:         1 * time.Second,
		TotalTime:    5 * time.Second,
		InputTokens:  100,
		OutputTokens: 200,
		Cost:         0.001,
		Error:        nil,
	}

	str := result.String()
	assert.Contains(t, str, "gpt-4o-mini")
	assert.Contains(t, str, "test-prompt")
	assert.Contains(t, str, "1s") // TTFT
	assert.Contains(t, str, "5s") // Total time
	assert.Contains(t, str, "100") // Input tokens
	assert.Contains(t, str, "200") // Output tokens
	assert.Contains(t, str, "0.001") // Cost
}

func TestBenchmarkResult_StringWithError(t *testing.T) {
	result := BenchmarkResult{
		Model:        "gpt-4o-mini",
		PromptName:   "test-prompt",
		Error:        assert.AnError,
	}

	str := result.String()
	assert.Contains(t, str, "gpt-4o-mini")
	assert.Contains(t, str, "test-prompt")
	assert.Contains(t, str, "ERROR")
	assert.Contains(t, str, assert.AnError.Error())
} 