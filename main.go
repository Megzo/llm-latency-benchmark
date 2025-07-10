package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/megzo/llm-latency-benchmark/internal/config"
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
	cfg, err := config.LoadConfig(*modelsFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

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
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
		cancel()
	}()

	// TODO: Initialize and run benchmark
	fmt.Printf("LLM Benchmark Tool v%s\n", version)
	fmt.Printf("Configuration loaded successfully\n")
	fmt.Printf("Concurrent requests: %d\n", cfg.Concurrent)
	fmt.Printf("Prompts directory: %s\n", cfg.PromptsDir)
	fmt.Printf("Models file: %s\n", *modelsFile)
	fmt.Printf("Output file: %s\n", cfg.GetOutputFile())
	fmt.Printf("Verbose mode: %t\n", cfg.Verbose)

	// Placeholder for benchmark execution
	fmt.Println("Benchmark execution not yet implemented")
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