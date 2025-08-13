package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file for testing
	tempFile := "test_config.yaml"
	defer os.Remove(tempFile)

	// Test valid config
    validConfig := `
openai:
  gpt-4o-mini:
    token_price:
      input: 0.15
      output: 0.6
    parameters:
      text:
        format:
          type: text
        verbosity: low
      reasoning:
        effort: minimal
        summary: null
  gpt-3.5-turbo:
    token_price:
      input: 0.5
      output: 1.5
    parameters: {}
groq:
  llama-3.1-8b:
    token_price:
      input: 0.05
      output: 0.1
    parameters: {}
`

	err := os.WriteFile(tempFile, []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := LoadConfig(tempFile)
	if err != nil {
		t.Fatalf("LoadConfig() failed with valid config: %v", err)
	}

	if config == nil {
		t.Fatal("LoadConfig() returned nil config")
	}

	if config.Models == nil {
		t.Fatal("Models config is nil")
	}

	// Check OpenAI models
    if len(config.Models.OpenAI) != 2 {
		t.Errorf("Expected 2 OpenAI models, got %d", len(config.Models.OpenAI))
	}

	// Check specific model configuration
    gpt4oMini, exists := config.Models.OpenAI["gpt-4o-mini"]
	if !exists {
		t.Fatal("gpt-4o-mini model not found in config")
	}

    if gpt4oMini.TokenPrice.Input != 0.15 {
        t.Errorf("Expected input cost 0.15, got %f", gpt4oMini.TokenPrice.Input)
    }

    if gpt4oMini.TokenPrice.Output != 0.6 {
        t.Errorf("Expected output cost 0.6, got %f", gpt4oMini.TokenPrice.Output)
    }
}

func TestLoadConfig_InvalidFile(t *testing.T) {
	_, err := LoadConfig("nonexistent_file.yaml")
	if err == nil {
		t.Error("LoadConfig() should fail with nonexistent file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tempFile := "test_invalid_config.yaml"
	defer os.Remove(tempFile)

    invalidConfig := `
openai:
  gpt-4o-mini:
    token_price:
      input: "invalid"  # Should be float
      output: 0.6
`

	err := os.WriteFile(tempFile, []byte(invalidConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err = LoadConfig(tempFile)
	if err == nil {
		t.Error("LoadConfig() should fail with invalid YAML")
	}
}

func TestLoadConfig_EmptyFile(t *testing.T) {
	tempFile := "test_empty_config.yaml"
	defer os.Remove(tempFile)

	err := os.WriteFile(tempFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := LoadConfig(tempFile)
	if err != nil {
		t.Fatalf("LoadConfig() failed with empty file: %v", err)
	}

	if config == nil {
		t.Fatal("LoadConfig() returned nil config for empty file")
	}

	if config.Models == nil {
		t.Fatal("Models config is nil for empty file")
	}
}

func TestModelPricing_CalculateCost(t *testing.T) {
	tests := []struct {
		name          string
		pricing       ModelPricing
		inputTokens   int
		outputTokens  int
		expectedCost  float64
	}{
		{
			name: "basic cost calculation",
			pricing: ModelPricing{
				Input:  0.15,  // $0.15 per million tokens
				Output: 0.6,   // $0.6 per million tokens
			},
			inputTokens:  1000000,  // 1 million tokens
			outputTokens: 500000,   // 0.5 million tokens
			expectedCost: 0.45,     // (1 * 0.15) + (0.5 * 0.6) = 0.15 + 0.3 = 0.45
		},
		{
			name: "zero tokens",
			pricing: ModelPricing{
				Input:  0.15,
				Output: 0.6,
			},
			inputTokens:  0,
			outputTokens: 0,
			expectedCost: 0.0,
		},
		{
			name: "small token count",
			pricing: ModelPricing{
				Input:  0.15,
				Output: 0.6,
			},
			inputTokens:  1000,   // 0.001 million tokens
			outputTokens: 500,    // 0.0005 million tokens
			expectedCost: 0.00045, // (0.001 * 0.15) + (0.0005 * 0.6) = 0.00015 + 0.0003 = 0.00045
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := tt.pricing.CalculateCost(tt.inputTokens, tt.outputTokens)
			assert.InDelta(t, tt.expectedCost, cost, 0.0001, "Cost calculation mismatch")
		})
	}
}

func TestModelsConfig_GetModelPricing(t *testing.T) {
    config := &ModelsConfig{
        OpenAI: map[string]ModelSpec{
            "gpt-4o-mini": {
                TokenPrice: ModelPricing{Input: 0.15, Output: 0.6},
                Parameters: nil,
            },
        },
        Groq: map[string]ModelSpec{
            "llama-3.1-8b": {
                TokenPrice: ModelPricing{Input: 0.05, Output: 0.1},
                Parameters: nil,
            },
        },
    }

	tests := []struct {
		name     string
		provider string
		model    string
		wantErr  bool
	}{
		{
			name:     "existing OpenAI model",
			provider: "openai",
			model:    "gpt-4o-mini",
			wantErr:  false,
		},
		{
			name:     "existing Groq model",
			provider: "groq",
			model:    "llama-3.1-8b",
			wantErr:  false,
		},
		{
			name:     "non-existing model",
			provider: "openai",
			model:    "gpt-5",
			wantErr:  true,
		},
		{
			name:     "non-existing provider",
			provider: "unknown",
			model:    "gpt-4o-mini",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing, err := config.GetModelPricing(tt.provider, tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("ModelsConfig.GetModelPricing() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && pricing == nil {
				t.Error("ModelsConfig.GetModelPricing() returned nil when no error expected")
			}
			if tt.wantErr && err == nil {
				t.Error("ModelsConfig.GetModelPricing() expected error but got none")
			}
		})
	}
} 