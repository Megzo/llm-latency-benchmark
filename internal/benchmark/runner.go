package benchmark

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/megzo/llm-latency-benchmark/internal/config"
	"github.com/megzo/llm-latency-benchmark/providers"
)

// Runner handles the execution of benchmark tests
type Runner struct {
	config     *config.Config
	providers  map[string]providers.Provider
	results    []BenchmarkResult
	resultsMu  sync.RWMutex
	verbose    bool
}

// NewRunner creates a new benchmark runner
func NewRunner(cfg *config.Config, providers map[string]providers.Provider, verbose bool) *Runner {
	return &Runner{
		config:    cfg,
		providers: providers,
		results:   make([]BenchmarkResult, 0),
		verbose:   verbose,
	}
}

// Run executes the benchmark according to configuration
func (r *Runner) Run(ctx context.Context) error {
	// Load prompts
	promptFiles, err := config.LoadPrompts(r.config.PromptsDir)
	if err != nil {
		return fmt.Errorf("failed to load prompts: %w", err)
	}

	if r.verbose {
		log.Printf("Loaded %d prompt files", len(promptFiles))
	}

	// Create a cancellable context for the entire run
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start the benchmark based on concurrency setting
	if r.config.Concurrent <= 1 {
		return r.runSequential(runCtx, promptFiles)
	} else {
		return r.runConcurrent(runCtx, promptFiles)
	}
}

// runSequential executes benchmarks sequentially
func (r *Runner) runSequential(ctx context.Context, promptFiles []config.PromptFile) error {
	if r.verbose {
		log.Println("Running benchmarks sequentially")
	}

	for _, promptFile := range promptFiles {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if r.verbose {
			log.Printf("Processing prompt file: %s", promptFile.Name)
		}

		// Test each provider and their models
		for providerName, provider := range r.providers {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Get models for this provider
			models, err := r.config.Models.ListModels(providerName)
			if err != nil {
				log.Printf("Warning: Failed to get models for provider %s: %v", providerName, err)
				continue
			}

			for _, modelName := range models {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				if r.verbose {
					log.Printf("  Testing model: %s (%d runs)", modelName, r.config.Runs)
				}

				// Run the benchmark multiple times
				for run := 1; run <= r.config.Runs; run++ {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}

					if r.verbose && r.config.Runs > 1 {
						log.Printf("    Run %d/%d", run, r.config.Runs)
					}

					// Run the benchmark
					result := r.runSingleBenchmark(ctx, provider, modelName, promptFile)
					r.addResult(result)
				}
			}
		}
	}

	return nil
}

// runConcurrent executes benchmarks with worker pools
func (r *Runner) runConcurrent(ctx context.Context, promptFiles []config.PromptFile) error {
	if r.verbose {
		log.Printf("Running benchmarks with %d concurrent workers", r.config.Concurrent)
	}

	// Create a channel to receive work items
	// Estimate work items: promptFiles * providers * models per provider * runs
	estimatedWorkItems := len(promptFiles) * len(r.providers) * 5 * r.config.Runs // Assume ~5 models per provider
	workChan := make(chan workItem, estimatedWorkItems)
	defer close(workChan)

	// Create a wait group to track worker completion
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < r.config.Concurrent; i++ {
		wg.Add(1)
		go r.worker(ctx, &wg, workChan, i+1)
	}

	// Send work items
	go func() {
		defer close(workChan)
		for _, promptFile := range promptFiles {
			for providerName, provider := range r.providers {
				// Get models for this provider
				models, err := r.config.Models.ListModels(providerName)
				if err != nil {
					log.Printf("Warning: Failed to get models for provider %s: %v", providerName, err)
					continue
				}

				for _, modelName := range models {
					for run := 1; run <= r.config.Runs; run++ {
						select {
						case <-ctx.Done():
							return
						case workChan <- workItem{promptFile: promptFile, provider: provider, modelName: modelName, run: run}:
						}
					}
				}
			}
		}
	}()

	// Wait for all workers to complete
	wg.Wait()

	return nil
}

// workItem represents a single benchmark task
type workItem struct {
	promptFile config.PromptFile
	provider   providers.Provider
	modelName  string
	run        int
}

// worker processes work items from the channel
func (r *Runner) worker(ctx context.Context, wg *sync.WaitGroup, workChan <-chan workItem, workerID int) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case work, ok := <-workChan:
			if !ok {
				return
			}

			if r.verbose {
				if r.config.Runs > 1 {
					log.Printf("Worker %d: Processing %s with model %s (run %d/%d)", workerID, work.promptFile.Name, work.modelName, work.run, r.config.Runs)
				} else {
					log.Printf("Worker %d: Processing %s with model %s", workerID, work.promptFile.Name, work.modelName)
				}
			}

			// Run the benchmark
			result := r.runSingleBenchmark(ctx, work.provider, work.modelName, work.promptFile)
			r.addResult(result)
		}
	}
}

// runSingleBenchmark executes a single benchmark test
func (r *Runner) runSingleBenchmark(ctx context.Context, provider providers.Provider, modelName string, promptFile config.PromptFile) BenchmarkResult {
	// Create metrics for this run
	metrics := NewMetrics()

    // Create the chat request
    req := providers.ChatRequest{
		Model:        modelName,
		SystemPrompt: promptFile.Prompt.System,
		UserPrompt:   promptFile.Prompt.User,
		MaxTokens:    1000, // Default max tokens
		Temperature:  0.7,  // Default temperature
		TopP:         1.0,  // Default top_p
	}

    // Apply per-model parameters from config (if present)
    if params, err := r.config.Models.GetModelParameters(provider.Name(), modelName); err == nil && params != nil {
        // Merge into ExtraParams map
        req.ExtraParams = make(map[string]interface{}, len(params))
        for k, v := range params {
            req.ExtraParams[k] = v
        }
    }

    // Add Groq-specific parameters for reasoning models (only if not already provided via model parameters)
	if provider.Name() == "groq" {
		// Check if this is a reasoning model that supports reasoning_effort
		if isReasoningModel(modelName) {
            if req.ExtraParams == nil || req.ExtraParams["reasoning_effort"] == nil {
                if req.ExtraParams == nil { req.ExtraParams = make(map[string]interface{}) }
                req.ExtraParams["reasoning_effort"] = "none"
            }
		}
	}

	// Create a timeout context for this request
	timeoutCtx, cancel := context.WithTimeout(ctx, r.config.RequestTimeout)
	defer cancel()

	// Start the streaming request
	responseChan, err := provider.StreamChat(timeoutCtx, req)
	if err != nil {
		metrics.SetError(&providers.ProviderError{
			Provider: provider.Name(),
			Message:  "failed to start streaming chat",
			Cause:    err,
		})
		return metrics.ToBenchmarkResult(provider.Name(), modelName, promptFile.Name)
	}

	// Process the streaming response
	var firstTokenReceived bool
	var fullResponse string
	for {
		select {
		case <-timeoutCtx.Done():
			metrics.SetError(&providers.TimeoutError{
				Operation: "streaming response",
				Duration:  r.config.RequestTimeout,
			})
			return metrics.ToBenchmarkResult(provider.Name(), modelName, promptFile.Name)

		case response, ok := <-responseChan:
			if !ok {
				// Stream completed successfully
				metrics.Complete()
				
				// Calculate costs
				cost := r.calculateCost(provider.Name(), modelName, metrics.InputTokens, metrics.OutputTokens)
				metrics.SetCost(cost)
				
				return metrics.ToBenchmarkResult(provider.Name(), modelName, promptFile.Name)
			}

					// Check for errors in the response
		if response.Error != nil {
			metrics.SetError(&providers.ProviderError{
				Provider: provider.Name(),
				Message:  "error in streaming response",
				Cause:    response.Error,
			})
			return metrics.ToBenchmarkResult(provider.Name(), modelName, promptFile.Name)
		}

			// Record first token time
			if !firstTokenReceived && response.Content != "" {
				metrics.RecordFirstToken()
				firstTokenReceived = true
			}

			// Add response content
			if response.Content != "" {
				fullResponse += response.Content
				metrics.AddResponseContent(response.Content)
			}

			// Calculate token counts if response is complete
			if response.IsComplete {
				// Estimate input tokens from the request
				inputTokens := provider.GetTokenCount(req.SystemPrompt + req.UserPrompt)
				// Estimate output tokens from the response
				outputTokens := provider.GetTokenCount(fullResponse)
				
				metrics.AddTokens(inputTokens, outputTokens)
			}
		}
	}
}

// calculateCost calculates the cost for a benchmark run
func (r *Runner) calculateCost(providerName, modelName string, inputTokens, outputTokens int) float64 {
	// Get pricing from the model configuration
	pricing, err := r.config.Models.GetModelPricing(providerName, modelName)
	if err != nil {
		// Return 0 cost if pricing not found
		return 0.0
	}
	
	return pricing.CalculateCost(inputTokens, outputTokens)
}

// addResult adds a result to the results slice in a thread-safe manner
func (r *Runner) addResult(result BenchmarkResult) {
	r.resultsMu.Lock()
	defer r.resultsMu.Unlock()
	r.results = append(r.results, result)
}

// GetResults returns a copy of all benchmark results
func (r *Runner) GetResults() []BenchmarkResult {
	r.resultsMu.RLock()
	defer r.resultsMu.RUnlock()
	
	results := make([]BenchmarkResult, len(r.results))
	copy(results, r.results)
	return results
}

// GetSummary returns a summary of all benchmark results
func (r *Runner) GetSummary() Summary {
	results := r.GetResults()
	return CalculateSummary(results)
}

// isReasoningModel checks if a Groq model supports the reasoning_effort parameter
func isReasoningModel(modelName string) bool {
	// List of Groq models that support reasoning_effort parameter
	reasoningModels := []string{
		"qwen/qwen3-32b",
		"qwen/qwen3-110b",
		"qwen/qwen3.5-110b",
		"qwen/qwen3.5-32b",
		"qwen/qwen3.5-7b",
		"qwen/qwen3.5-14b",
		"qwen/qwen3.5-72b",
		"qwen/qwen3.5-32b-instruct",
		"qwen/qwen3.5-110b-instruct",
		"qwen/qwen3.5-7b-instruct",
		"qwen/qwen3.5-14b-instruct",
		"qwen/qwen3.5-72b-instruct",
		"qwen/qwen3-32b-instruct",
		"qwen/qwen3-110b-instruct",
		"qwen/qwen3.5-32b-chat",
		"qwen/qwen3.5-110b-chat",
		"qwen/qwen3.5-7b-chat",
		"qwen/qwen3.5-14b-chat",
		"qwen/qwen3.5-72b-chat",
		"qwen/qwen3-32b-chat",
		"qwen/qwen3-110b-chat",
	}

	for _, model := range reasoningModels {
		if model == modelName {
			return true
		}
	}
	return false
} 