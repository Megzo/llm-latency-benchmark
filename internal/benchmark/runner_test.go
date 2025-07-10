package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"llm-latency-benchmark/internal/config"
	"llm-latency-benchmark/providers"
)

// MockProvider for testing
type MockProvider struct {
	name string
	delay time.Duration
	shouldFail bool
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Chat(request providers.ChatRequest) (*providers.ChatResponse, error) {
	if m.shouldFail {
		return nil, assert.AnError
	}
	
	// Simulate processing delay
	time.Sleep(m.delay)
	
	return &providers.ChatResponse{
		Content: "Mock response for " + request.UserPrompt,
	}, nil
}

func (m *MockProvider) ValidateRequest(request providers.ChatRequest) error {
	if request.Model == "" {
		return assert.AnError
	}
	return nil
}

func (m *MockProvider) TokenCount(response providers.ChatResponse) (int, int, int) {
	return 10, len(response.Content), 10 + len(response.Content)
}

func (m *MockProvider) IsRetryableError(err error) bool {
	return false
}

func (m *MockProvider) GetRetryDelay(attempt int) time.Duration {
	return time.Second
}

func TestBenchmarkRunner_SequentialExecution(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Models: map[string]config.ModelConfig{
			"mock-model": {
				Provider:         "mock",
				InputCostPer1K:   0.001,
				OutputCostPer1K:  0.002,
				MaxTokens:        1000,
				ContextLength:    4000,
			},
		},
	}

	// Create test prompts
	prompts := map[string]config.Prompt{
		"test1": {
			Name:        "Test Prompt 1",
			Description: "First test prompt",
			Prompt:      "Hello, world!",
		},
		"test2": {
			Name:        "Test Prompt 2",
			Description: "Second test prompt",
			Prompt:      "How are you today?",
		},
	}

	// Create mock provider
	provider := &MockProvider{
		name:  "mock",
		delay: 10 * time.Millisecond,
	}

	// Create provider factory and register mock provider
	factory := providers.NewProviderFactory()
	err := factory.RegisterProvider("mock", func(config interface{}) (providers.Provider, error) {
		return provider, nil
	})
	require.NoError(t, err)

	// Create benchmark runner
	runner := NewBenchmarkRunner(cfg, prompts, factory)

	// Run benchmarks sequentially
	results := make(chan BenchmarkResult, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		defer close(results)
		err := runner.RunSequential(ctx, results)
		assert.NoError(t, err)
	}()

	// Collect results
	var allResults []BenchmarkResult
	for result := range results {
		allResults = append(allResults, result)
	}

	// Verify results
	assert.Len(t, allResults, 2) // 2 prompts

	for _, result := range allResults {
		assert.Equal(t, "mock-model", result.Model)
		assert.True(t, result.IsSuccessful())
		assert.Greater(t, result.TTFT, time.Duration(0))
		assert.Greater(t, result.TotalTime, time.Duration(0))
		assert.Equal(t, 10, result.InputTokens)
		assert.Greater(t, result.OutputTokens, 0)
		assert.Greater(t, result.Cost, 0.0)
	}
}

func TestBenchmarkRunner_ConcurrentExecution(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Models: map[string]config.ModelConfig{
			"mock-model": {
				Provider:         "mock",
				InputCostPer1K:   0.001,
				OutputCostPer1K:  0.002,
				MaxTokens:        1000,
				ContextLength:    4000,
			},
		},
	}

	// Create test prompts
	prompts := map[string]config.Prompt{
		"test1": {
			Name:        "Test Prompt 1",
			Description: "First test prompt",
			Prompt:      "Hello, world!",
		},
		"test2": {
			Name:        "Test Prompt 2",
			Description: "Second test prompt",
			Prompt:      "How are you today?",
		},
		"test3": {
			Name:        "Test Prompt 3",
			Description: "Third test prompt",
			Prompt:      "What is the weather like?",
		},
	}

	// Create mock provider
	provider := &MockProvider{
		name:  "mock",
		delay: 50 * time.Millisecond,
	}

	// Create provider factory and register mock provider
	factory := providers.NewProviderFactory()
	err := factory.RegisterProvider("mock", func(config interface{}) (providers.Provider, error) {
		return provider, nil
	})
	require.NoError(t, err)

	// Create benchmark runner
	runner := NewBenchmarkRunner(cfg, prompts, factory)

	// Run benchmarks concurrently
	results := make(chan BenchmarkResult, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()
	go func() {
		defer close(results)
		err := runner.RunConcurrent(ctx, results, 2) // 2 concurrent workers
		assert.NoError(t, err)
	}()

	// Collect results
	var allResults []BenchmarkResult
	for result := range results {
		allResults = append(allResults, result)
	}

	// Verify results
	assert.Len(t, allResults, 3) // 3 prompts

	// Check that concurrent execution was faster than sequential
	// Sequential would take: 3 * 50ms = 150ms
	// Concurrent should take: ~100ms (with 2 workers)
	executionTime := time.Since(startTime)
	assert.Less(t, executionTime, 150*time.Millisecond, "Concurrent execution should be faster than sequential")

	for _, result := range allResults {
		assert.Equal(t, "mock-model", result.Model)
		assert.True(t, result.IsSuccessful())
		assert.Greater(t, result.TTFT, time.Duration(0))
		assert.Greater(t, result.TotalTime, time.Duration(0))
		assert.Equal(t, 10, result.InputTokens)
		assert.Greater(t, result.OutputTokens, 0)
		assert.Greater(t, result.Cost, 0.0)
	}
}

func TestBenchmarkRunner_ErrorHandling(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Models: map[string]config.ModelConfig{
			"mock-model": {
				Provider:         "mock",
				InputCostPer1K:   0.001,
				OutputCostPer1K:  0.002,
				MaxTokens:        1000,
				ContextLength:    4000,
			},
		},
	}

	// Create test prompts
	prompts := map[string]config.Prompt{
		"test1": {
			Name:        "Test Prompt 1",
			Description: "First test prompt",
			Prompt:      "Hello, world!",
		},
		"test2": {
			Name:        "Test Prompt 2",
			Description: "Second test prompt",
			Prompt:      "How are you today?",
		},
	}

	// Create mock provider that fails
	provider := &MockProvider{
		name:        "mock",
		delay:       10 * time.Millisecond,
		shouldFail:  true,
	}

	// Create provider factory and register mock provider
	factory := providers.NewProviderFactory()
	err := factory.RegisterProvider("mock", func(config interface{}) (providers.Provider, error) {
		return provider, nil
	})
	require.NoError(t, err)

	// Create benchmark runner
	runner := NewBenchmarkRunner(cfg, prompts, factory)

	// Run benchmarks
	results := make(chan BenchmarkResult, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		defer close(results)
		err := runner.RunSequential(ctx, results)
		assert.NoError(t, err)
	}()

	// Collect results
	var allResults []BenchmarkResult
	for result := range results {
		allResults = append(allResults, result)
	}

	// Verify results
	assert.Len(t, allResults, 2) // 2 prompts

	for _, result := range allResults {
		assert.Equal(t, "mock-model", result.Model)
		assert.False(t, result.IsSuccessful())
		assert.NotNil(t, result.Error)
		assert.Equal(t, 0, result.InputTokens)
		assert.Equal(t, 0, result.OutputTokens)
		assert.Equal(t, 0.0, result.Cost)
	}
}

func TestBenchmarkRunner_ContextCancellation(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Models: map[string]config.ModelConfig{
			"mock-model": {
				Provider:         "mock",
				InputCostPer1K:   0.001,
				OutputCostPer1K:  0.002,
				MaxTokens:        1000,
				ContextLength:    4000,
			},
		},
	}

	// Create test prompts
	prompts := map[string]config.Prompt{
		"test1": {
			Name:        "Test Prompt 1",
			Description: "First test prompt",
			Prompt:      "Hello, world!",
		},
		"test2": {
			Name:        "Test Prompt 2",
			Description: "Second test prompt",
			Prompt:      "How are you today?",
		},
	}

	// Create mock provider with long delay
	provider := &MockProvider{
		name:  "mock",
		delay: 1 * time.Second, // Long delay
	}

	// Create provider factory and register mock provider
	factory := providers.NewProviderFactory()
	err := factory.RegisterProvider("mock", func(config interface{}) (providers.Provider, error) {
		return provider, nil
	})
	require.NoError(t, err)

	// Create benchmark runner
	runner := NewBenchmarkRunner(cfg, prompts, factory)

	// Run benchmarks with short timeout
	results := make(chan BenchmarkResult, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond) // Short timeout
	defer cancel()

	go func() {
		defer close(results)
		err := runner.RunSequential(ctx, results)
		assert.NoError(t, err)
	}()

	// Collect results
	var allResults []BenchmarkResult
	for result := range results {
		allResults = append(allResults, result)
	}

	// Should have fewer results due to timeout
	assert.LessOrEqual(t, len(allResults), 2)
}

func TestBenchmarkRunner_InvalidModel(t *testing.T) {
	// Create test configuration with non-existent model
	cfg := &config.Config{
		Models: map[string]config.ModelConfig{
			"non-existent-model": {
				Provider:         "non-existent-provider",
				InputCostPer1K:   0.001,
				OutputCostPer1K:  0.002,
				MaxTokens:        1000,
				ContextLength:    4000,
			},
		},
	}

	// Create test prompts
	prompts := map[string]config.Prompt{
		"test1": {
			Name:        "Test Prompt 1",
			Description: "First test prompt",
			Prompt:      "Hello, world!",
		},
	}

	// Create provider factory (no providers registered)
	factory := providers.NewProviderFactory()

	// Create benchmark runner
	runner := NewBenchmarkRunner(cfg, prompts, factory)

	// Run benchmarks
	results := make(chan BenchmarkResult, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		defer close(results)
		err := runner.RunSequential(ctx, results)
		assert.NoError(t, err)
	}()

	// Collect results
	var allResults []BenchmarkResult
	for result := range results {
		allResults = append(allResults, result)
	}

	// Should have results but they should be failed
	assert.Len(t, allResults, 1)

	for _, result := range allResults {
		assert.Equal(t, "non-existent-model", result.Model)
		assert.False(t, result.IsSuccessful())
		assert.NotNil(t, result.Error)
	}
}

func TestBenchmarkRunner_EmptyPrompts(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Models: map[string]config.ModelConfig{
			"mock-model": {
				Provider:         "mock",
				InputCostPer1K:   0.001,
				OutputCostPer1K:  0.002,
				MaxTokens:        1000,
				ContextLength:    4000,
			},
		},
	}

	// Create empty prompts
	prompts := map[string]config.Prompt{}

	// Create mock provider
	provider := &MockProvider{
		name:  "mock",
		delay: 10 * time.Millisecond,
	}

	// Create provider factory and register mock provider
	factory := providers.NewProviderFactory()
	err := factory.RegisterProvider("mock", func(config interface{}) (providers.Provider, error) {
		return provider, nil
	})
	require.NoError(t, err)

	// Create benchmark runner
	runner := NewBenchmarkRunner(cfg, prompts, factory)

	// Run benchmarks
	results := make(chan BenchmarkResult, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		defer close(results)
		err := runner.RunSequential(ctx, results)
		assert.NoError(t, err)
	}()

	// Collect results
	var allResults []BenchmarkResult
	for result := range results {
		allResults = append(allResults, result)
	}

	// Should have no results
	assert.Len(t, allResults, 0)
} 