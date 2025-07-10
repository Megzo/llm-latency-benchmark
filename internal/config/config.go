package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	// API Keys
	OpenAIAPIKey    string
	GroqAPIKey      string
	AnthropicAPIKey string

	// Provider Base URLs
	OpenAIBaseURL    string
	GroqBaseURL      string
	AnthropicBaseURL string

	// Models configuration
	Models *ModelsConfig

	// CLI flags
	Concurrent int
	PromptsDir string
	OutputFile string
	Verbose    bool

	// Benchmark settings
	Timeout time.Duration
	Retries int
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

		OpenAIBaseURL:    getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		GroqBaseURL:      getEnvOrDefault("GROQ_BASE_URL", "https://api.groq.com/openai/v1"),
		AnthropicBaseURL: getEnvOrDefault("ANTHROPIC_BASE_URL", "https://api.anthropic.com"),

		Concurrent: 1,
		PromptsDir: "prompts",
		OutputFile: "",
		Verbose:    false,

		Timeout: 30 * time.Second,
		Retries: 3,
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

// Helper function to get environment variable with default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
} 