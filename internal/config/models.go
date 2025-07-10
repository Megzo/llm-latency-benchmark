package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ModelsConfig holds the pricing configuration for all models
type ModelsConfig struct {
	OpenAI   map[string]ModelPricing `yaml:"openai"`
	Groq     map[string]ModelPricing `yaml:"groq"`
	Anthropic map[string]ModelPricing `yaml:"anthropic"`
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
	var pricing map[string]ModelPricing

	switch provider {
	case "openai":
		pricing = c.OpenAI
	case "groq":
		pricing = c.Groq
	case "anthropic":
		pricing = c.Anthropic
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	if modelPricing, exists := pricing[model]; exists {
		return &modelPricing, nil
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
	var models map[string]ModelPricing

	switch provider {
	case "openai":
		models = c.OpenAI
	case "groq":
		models = c.Groq
	case "anthropic":
		models = c.Anthropic
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	modelNames := make([]string, 0, len(models))
	for modelName := range models {
		modelNames = append(modelNames, modelName)
	}

	return modelNames, nil
} 