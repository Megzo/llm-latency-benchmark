package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/megzo/llm-latency-benchmark/internal/benchmark"
	"github.com/megzo/llm-latency-benchmark/internal/config"
	"github.com/megzo/llm-latency-benchmark/internal/output"
	"github.com/megzo/llm-latency-benchmark/providers"
)

const version = "0.1.0"

func main() {
	// Parse command line flags
	var (
		concurrent = flag.Int("concurrent", 1, "Number of concurrent requests")
		promptsDir = flag.String("prompts", "prompts", "Directory containing prompt files")
		outputFile = flag.String("output", "", "Output CSV file (default: results/benchmark_TIMESTAMP.csv)")
		modelsFile = flag.String("models", "models.yaml", "Models configuration file (default: models.yaml)")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
		showHelp   = flag.Bool("help", false, "Show help message")
		showVersion = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	// Handle help and version flags
	if *showHelp {
		printHelp()
		return
	}

	if *showVersion {
		fmt.Printf("llm-benchmark v%s\n", version)
		return
	}

	// Load configuration
	fmt.Printf("Loading configuration from %s...\n", *modelsFile)
	cfg, err := config.LoadConfig(*modelsFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	fmt.Printf("Configuration loaded successfully\n")

	// Override config with CLI flags
	cfg.Concurrent = *concurrent
	cfg.PromptsDir = *promptsDir
	cfg.OutputFile = *outputFile
	cfg.Verbose = *verbose

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
		cancel()
	}()

	// Initialize provider factory
	fmt.Printf("Initializing provider factory...\n")
	factory := providers.NewProviderFactory()
	
	// Register provider configurations
	fmt.Printf("Registering provider configurations...\n")
	factory.RegisterConfig("openai", cfg.GetOpenAIConfig())
	factory.RegisterConfig("groq", cfg.GetGroqConfig())
	factory.RegisterConfig("anthropic", cfg.GetAnthropicConfig())
	
	// Create provider instances for all configured providers
	providerMap := make(map[string]providers.Provider)
	
	// Initialize OpenAI provider if API key is available
	fmt.Printf("Checking OpenAI API key...\n")
	if cfg.OpenAIAPIKey != "" {
		fmt.Printf("OpenAI API key found, creating provider...\n")
		provider, err := factory.GetProvider("openai")
		if err != nil {
			log.Printf("Warning: Failed to create OpenAI provider: %v", err)
		} else {
			providerMap["openai"] = provider
			fmt.Printf("OpenAI provider created successfully\n")
		}
	} else {
		fmt.Printf("No OpenAI API key found\n")
	}
	
	// Initialize Groq provider if API key is available
	fmt.Printf("Checking Groq API key...\n")
	if cfg.GroqAPIKey != "" {
		fmt.Printf("Groq API key found, creating provider...\n")
		provider, err := factory.GetProvider("groq")
		if err != nil {
			log.Printf("Warning: Failed to create Groq provider: %v", err)
		} else {
			providerMap["groq"] = provider
			fmt.Printf("Groq provider created successfully\n")
		}
	} else {
		fmt.Printf("No Groq API key found\n")
	}
	
	// Initialize Anthropic provider if API key is available
	fmt.Printf("Checking Anthropic API key...\n")
	if cfg.AnthropicAPIKey != "" {
		fmt.Printf("Anthropic API key found, creating provider...\n")
		provider, err := factory.GetProvider("anthropic")
		if err != nil {
			log.Printf("Warning: Failed to create Anthropic provider: %v", err)
		} else {
			providerMap["anthropic"] = provider
			fmt.Printf("Anthropic provider created successfully\n")
		}
	} else {
		fmt.Printf("No Anthropic API key found\n")
	}
	
	if len(providerMap) == 0 {
		log.Fatal("No valid providers could be initialized")
	}
	
	fmt.Printf("Providers initialized: %d\n", len(providerMap))
	
	// Create and run benchmark
	runner := benchmark.NewRunner(cfg, providerMap, cfg.Verbose)
	
	fmt.Printf("LLM Benchmark Tool v%s\n", version)
	fmt.Printf("Configuration loaded successfully\n")
	fmt.Printf("Concurrent requests: %d\n", cfg.Concurrent)
	fmt.Printf("Prompts directory: %s\n", cfg.PromptsDir)
	fmt.Printf("Models file: %s\n", *modelsFile)
	fmt.Printf("Output file: %s\n", cfg.GetOutputFile())
	fmt.Printf("Verbose mode: %t\n", cfg.Verbose)
	fmt.Printf("Providers initialized: %d\n", len(providerMap))
	
	// Run the benchmark
	if err := runner.Run(ctx); err != nil {
		log.Fatalf("Benchmark failed: %v", err)
	}
	
	// Get results and write to CSV
	results := runner.GetResults()
	if len(results) == 0 {
		log.Println("No benchmark results generated")
		return
	}
	
	// Write results to CSV
	csvWriter := output.NewCSVWriter(cfg.GetOutputFile())
	if err := csvWriter.WriteResults(results); err != nil {
		log.Fatalf("Failed to write CSV results: %v", err)
	}
	
	// Print summary
	summary := runner.GetSummary()
	fmt.Printf("\nBenchmark completed successfully!\n")
	fmt.Printf("Results written to: %s\n", cfg.GetOutputFile())
	fmt.Printf("Total runs: %d\n", summary.TotalRuns)
	fmt.Printf("Successful runs: %d\n", summary.SuccessfulRuns)
	fmt.Printf("Failed runs: %d\n", summary.FailedRuns)
	fmt.Printf("Error rate: %.2f%%\n", summary.ErrorRate*100)
	if summary.SuccessfulRuns > 0 {
		fmt.Printf("Average TTFT: %v\n", summary.AvgTTFT)
		fmt.Printf("Average total time: %v\n", summary.AvgTotalTime)
		fmt.Printf("Total cost: $%.6f\n", summary.TotalCost)
	}
}

func printHelp() {
	fmt.Printf(`LLM Benchmark Tool v%s

A Go-based command-line tool for measuring LLM latency and performance metrics 
across multiple providers, specifically designed for real-time use cases.

Usage:
  llm-benchmark [flags]

Flags:
  -concurrent int
        Number of concurrent requests (default 1)
  -prompts string
        Directory containing prompt files (default "prompts")
  -output string
        Output CSV file (default: results/benchmark_TIMESTAMP.csv)
  -models string
        Models configuration file (default "models.yaml")
  -verbose
        Enable verbose logging
  -help
        Show this help message
  -version
        Show version information

Examples:
  # Basic usage (sequential)
  llm-benchmark

  # Concurrent execution
  llm-benchmark -concurrent 4

  # Specify prompts directory
  llm-benchmark -prompts ./custom-prompts

  # Custom output file
  llm-benchmark -output results/my-benchmark.csv

  # Use custom models file
  llm-benchmark -models mymodels.yaml

  # Verbose logging
  llm-benchmark -verbose

Configuration:
  Create a .env file with your API keys:
    OPENAI_API_KEY=sk-...
    GROQ_API_KEY=gsk_...
    ANTHROPIC_API_KEY=sk-ant-...

  The models.yaml file contains pricing information for different models.
`, version)
} 