package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/megzo/llm-latency-benchmark/providers"
)

// Config holds all application configuration
type Config struct {
	// API Keys
	OpenAIAPIKey    string
	GroqAPIKey      string
	AnthropicAPIKey string
	AzureOpenAIAPIKey string
	GoogleAPIKey    string

	// Provider Base URLs
	OpenAIBaseURL    string
	GroqBaseURL      string
	AnthropicBaseURL string
	AzureOpenAIEndpoint string
	AzureOpenAIAPIVersion string

	// Models configuration
	Models *ModelsConfig

	// CLI flags
	Concurrent int
	Runs       int
	PromptsDir string
	OutputFile string
	Verbose    bool

	// Benchmark settings
	Timeout        time.Duration
	RequestTimeout time.Duration
	Retries        int
}

// LoadConfig loads configuration from environment variables and files
func LoadConfig(modelsFile string) (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// It's okay if .env doesn't exist
	}

	config := &Config{
		OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
		GroqAPIKey:      os.Getenv("GROQ_API_KEY"),
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		AzureOpenAIAPIKey: os.Getenv("AZURE_OPENAI_API_KEY"),
		GoogleAPIKey:    os.Getenv("GOOGLE_API_KEY"),

		OpenAIBaseURL:    getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		GroqBaseURL:      getEnvOrDefault("GROQ_BASE_URL", "https://api.groq.com/openai/v1"),
		AnthropicBaseURL: getEnvOrDefault("ANTHROPIC_BASE_URL", "https://api.anthropic.com"),
		AzureOpenAIEndpoint: os.Getenv("AZURE_OPENAI_ENDPOINT"),
		AzureOpenAIAPIVersion: getEnvOrDefault("AZURE_OPENAI_API_VERSION", "2024-02-15-preview"),

		Concurrent: 1,
		Runs:       1,
		PromptsDir: "prompts",
		OutputFile: "",
		Verbose:    false,

		Timeout:        30 * time.Second,
		RequestTimeout: 60 * time.Second,
		Retries:        3,
	}

	// Load models configuration
	modelsConfig, err := LoadModelsConfig(modelsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load models config: %w", err)
	}
	config.Models = modelsConfig

	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Concurrent < 1 {
		return fmt.Errorf("concurrent must be at least 1")
	}

	if c.Runs < 1 {
		return fmt.Errorf("runs must be at least 1")
	}

	if c.PromptsDir == "" {
		return fmt.Errorf("prompts directory cannot be empty")
	}

	if _, err := os.Stat(c.PromptsDir); os.IsNotExist(err) {
		return fmt.Errorf("prompts directory does not exist: %s", c.PromptsDir)
	}

	return nil
}

// GetOutputFile returns the output file path, generating a default if not specified
func (c *Config) GetOutputFile() string {
	if c.OutputFile != "" {
		return c.OutputFile
	}

	// Generate default filename with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	return filepath.Join("results", fmt.Sprintf("benchmark_%s.csv", timestamp))
}

// GetOpenAIConfig returns OpenAI provider configuration
func (c *Config) GetOpenAIConfig() *providers.OpenAIConfig {
	return &providers.OpenAIConfig{
		APIKey:  c.OpenAIAPIKey,
		BaseURL: c.OpenAIBaseURL,
	}
}

// GetGroqConfig returns Groq provider configuration
func (c *Config) GetGroqConfig() *providers.GroqConfig {
	return &providers.GroqConfig{
		APIKey:  c.GroqAPIKey,
		BaseURL: c.GroqBaseURL,
	}
}

// GetAnthropicConfig returns Anthropic provider configuration
func (c *Config) GetAnthropicConfig() *providers.AnthropicConfig {
	return &providers.AnthropicConfig{
		APIKey:  c.AnthropicAPIKey,
		BaseURL: c.AnthropicBaseURL,
	}
}

// GetAzureOpenAIConfig returns Azure OpenAI provider configuration
func (c *Config) GetAzureOpenAIConfig() *providers.AzureOpenAIConfig {
	return &providers.AzureOpenAIConfig{
		Endpoint:       c.AzureOpenAIEndpoint,
		APIKey:         c.AzureOpenAIAPIKey,
		APIVersion:     c.AzureOpenAIAPIVersion,
	}
}

// GetGeminiConfig returns Gemini provider configuration
func (c *Config) GetGeminiConfig() *providers.GeminiConfig {
	return &providers.GeminiConfig{
		APIKey: c.GoogleAPIKey,
	}
}

// Helper function to get environment variable with default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
} 