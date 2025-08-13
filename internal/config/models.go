package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ModelsConfig holds the pricing and parameter configuration for all models
type ModelsConfig struct {
	OpenAI       map[string]ModelSpec `yaml:"openai"`
    OpenAIResponses map[string]ModelSpec `yaml:"openai_responses"`
	Groq         map[string]ModelSpec `yaml:"groq"`
	Anthropic    map[string]ModelSpec `yaml:"anthropic"`
	AzureOpenAI  map[string]ModelSpec `yaml:"azure_openai"`
	Gemini       map[string]ModelSpec `yaml:"gemini"`
}

// ModelSpec defines token pricing and optional provider-specific parameters
type ModelSpec struct {
	TokenPrice ModelPricing            `yaml:"token_price"`
	Parameters map[string]interface{} `yaml:"parameters"`
}

// ModelPricing holds the pricing information for a specific model
type ModelPricing struct {
	Input  float64 `yaml:"input"`  // $ per million input tokens
	Output float64 `yaml:"output"` // $ per million output tokens
}

// LoadModelsConfig loads the models configuration from a YAML file
func LoadModelsConfig(filename string) (*ModelsConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read models config file: %w", err)
	}

	var config ModelsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse models config YAML: %w", err)
	}

	return &config, nil
}

// GetModelPricing returns the pricing for a specific model
func (c *ModelsConfig) GetModelPricing(provider, model string) (*ModelPricing, error) {
	var specs map[string]ModelSpec

	switch provider {
	case "openai":
		specs = c.OpenAI
	case "openai_responses":
        specs = c.OpenAIResponses
	case "groq":
		specs = c.Groq
	case "anthropic":
		specs = c.Anthropic
	case "azure_openai":
		specs = c.AzureOpenAI
	case "gemini":
		specs = c.Gemini
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	if spec, exists := specs[model]; exists {
		return &spec.TokenPrice, nil
	}

	return nil, fmt.Errorf("model %s not found for provider %s", model, provider)
}

// GetModelParameters returns the parameters map for a specific model (may be nil)
func (c *ModelsConfig) GetModelParameters(provider, model string) (map[string]interface{}, error) {
	var specs map[string]ModelSpec

	switch provider {
	case "openai":
		specs = c.OpenAI
	case "openai_responses":
        specs = c.OpenAIResponses
	case "groq":
		specs = c.Groq
	case "anthropic":
		specs = c.Anthropic
	case "azure_openai":
		specs = c.AzureOpenAI
	case "gemini":
		specs = c.Gemini
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	if spec, exists := specs[model]; exists {
		return spec.Parameters, nil
	}

	return nil, fmt.Errorf("model %s not found for provider %s", model, provider)
}

// CalculateCost calculates the cost for a given number of input and output tokens
func (p *ModelPricing) CalculateCost(inputTokens, outputTokens int) float64 {
	inputCost := (float64(inputTokens) / 1_000_000) * p.Input
	outputCost := (float64(outputTokens) / 1_000_000) * p.Output
	return inputCost + outputCost
}

// ListModels returns all available models for a provider
func (c *ModelsConfig) ListModels(provider string) ([]string, error) {
	var specs map[string]ModelSpec

	switch provider {
	case "openai":
		specs = c.OpenAI
	case "openai_responses":
        specs = c.OpenAIResponses
	case "groq":
		specs = c.Groq
	case "anthropic":
		specs = c.Anthropic
	case "azure_openai":
		specs = c.AzureOpenAI
	case "gemini":
		specs = c.Gemini
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	modelNames := make([]string, 0, len(specs))
	for modelName := range specs {
		modelNames = append(modelNames, modelName)
	}

	return modelNames, nil
}