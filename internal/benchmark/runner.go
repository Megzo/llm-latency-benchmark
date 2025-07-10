package benchmark

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/megzo/llm-latency-benchmark/internal/config"
	"github.com/megzo/llm-latency-benchmark/providers"
)

// Runner handles the execution of benchmark tests
type Runner struct {
	config     *config.Config
	providers  map[string]providers.Provider
	results    []providers.BenchmarkResult
	resultsMu  sync.RWMutex
	verbose    bool
}

// NewRunner creates a new benchmark runner
func NewRunner(cfg *config.Config, providers map[string]providers.Provider, verbose bool) *Runner {
	return &Runner{
		config:    cfg,
		providers: providers,
		results:   make([]providers.BenchmarkResult, 0),
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

		for _, model := range r.config.Models {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if r.verbose {
				log.Printf("  Testing model: %s", model.Name)
			}

			// Find the provider for this model
			provider, exists := r.providers[model.Provider]
			if !exists {
				log.Printf("Warning: Provider %s not found for model %s", model.Provider, model.Name)
				continue
			}

			// Run the benchmark
			result := r.runSingleBenchmark(ctx, provider, model, promptFile)
			r.addResult(result)
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
	workChan := make(chan workItem, len(promptFiles)*len(r.config.Models))
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
			for _, model := range r.config.Models {
				select {
				case <-ctx.Done():
					return
				case workChan <- workItem{promptFile: promptFile, model: model}:
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
	model      config.Model
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
				log.Printf("Worker %d: Processing %s with model %s", workerID, work.promptFile.Name, work.model.Name)
			}

			// Find the provider for this model
			provider, exists := r.providers[work.model.Provider]
			if !exists {
				log.Printf("Warning: Provider %s not found for model %s", work.model.Provider, work.model.Name)
				continue
			}

			// Run the benchmark
			result := r.runSingleBenchmark(ctx, provider, work.model, work.promptFile)
			r.addResult(result)
		}
	}
}

// runSingleBenchmark executes a single benchmark test
func (r *Runner) runSingleBenchmark(ctx context.Context, provider providers.Provider, model config.Model, promptFile config.PromptFile) providers.BenchmarkResult {
	// Create metrics for this run
	metrics := NewMetrics()

	// Create the chat request
	req := providers.ChatRequest{
		Model:        model.Name,
		SystemPrompt: promptFile.Prompt.System,
		UserPrompt:   promptFile.Prompt.User,
		MaxTokens:    model.MaxTokens,
		Temperature:  model.Temperature,
		TopP:         model.TopP,
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
		return metrics.ToBenchmarkResult(provider.Name(), model.Name, promptFile.Name)
	}

	// Process the streaming response
	var firstTokenReceived bool
	for {
		select {
		case <-timeoutCtx.Done():
			metrics.SetError(&providers.TimeoutError{
				Operation: "streaming response",
				Duration:  r.config.RequestTimeout,
			})
			return metrics.ToBenchmarkResult(provider.Name(), model.Name, promptFile.Name)

		case response, ok := <-responseChan:
			if !ok {
				// Stream completed successfully
				metrics.Complete()
				return metrics.ToBenchmarkResult(provider.Name(), model.Name, promptFile.Name)
			}

			// Check for errors in the response
			if response.Error != nil {
				metrics.SetError(&providers.ProviderError{
					Provider: provider.Name(),
					Message:  "error in streaming response",
					Cause:    response.Error,
				})
				return metrics.ToBenchmarkResult(provider.Name(), model.Name, promptFile.Name)
			}

			// Record first token time
			if !firstTokenReceived {
				metrics.RecordFirstToken()
				firstTokenReceived = true
			}

			// Add response content
			metrics.AddResponseContent(response.Content)

			// Calculate token counts if response is complete
			if response.IsComplete {
				inputTokens, outputTokens, _ := provider.TokenCount(response)
				metrics.AddTokens(inputTokens, outputTokens)
			}
		}
	}
}

// addResult adds a result to the results slice in a thread-safe manner
func (r *Runner) addResult(result providers.BenchmarkResult) {
	r.resultsMu.Lock()
	defer r.resultsMu.Unlock()
	r.results = append(r.results, result)
}

// GetResults returns a copy of all benchmark results
func (r *Runner) GetResults() []providers.BenchmarkResult {
	r.resultsMu.RLock()
	defer r.resultsMu.RUnlock()
	
	results := make([]providers.BenchmarkResult, len(r.results))
	copy(results, r.results)
	return results
}

// GetSummary returns a summary of all benchmark results
func (r *Runner) GetSummary() Summary {
	results := r.GetResults()
	return CalculateSummary(results)
} 