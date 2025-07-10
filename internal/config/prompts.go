package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Prompt represents a single prompt configuration
type Prompt struct {
	System string `yaml:"system"`
	User   string `yaml:"user"`
}

// PromptFile represents a prompt file with metadata
type PromptFile struct {
	Name   string
	Path   string
	Prompt Prompt
}

// LoadPrompts loads all prompt files from the specified directory
func LoadPrompts(promptsDir string) ([]PromptFile, error) {
	var promptFiles []PromptFile

	// Walk through the prompts directory
	err := filepath.Walk(promptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-YAML files
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".yaml") && !strings.HasSuffix(strings.ToLower(path), ".yml") {
			return nil
		}

		// Load the prompt file
		prompt, err := loadPromptFile(path)
		if err != nil {
			return fmt.Errorf("failed to load prompt file %s: %w", path, err)
		}

		// Validate the prompt
		if err := validatePrompt(prompt); err != nil {
			return fmt.Errorf("invalid prompt in %s: %w", path, err)
		}

		promptFiles = append(promptFiles, PromptFile{
			Name:   strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
			Path:   path,
			Prompt: prompt,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk prompts directory: %w", err)
	}

	if len(promptFiles) == 0 {
		return nil, fmt.Errorf("no valid prompt files found in %s", promptsDir)
	}

	return promptFiles, nil
}

// loadPromptFile loads a single prompt file
func loadPromptFile(path string) (Prompt, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Prompt{}, fmt.Errorf("failed to read file: %w", err)
	}

	var prompt Prompt
	if err := yaml.Unmarshal(data, &prompt); err != nil {
		return Prompt{}, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return prompt, nil
}

// validatePrompt validates a prompt configuration
func validatePrompt(prompt Prompt) error {
	if prompt.User == "" {
		return fmt.Errorf("user prompt cannot be empty")
	}

	// System prompt is optional, so no validation needed

	return nil
}

// GetPromptText returns the full prompt text (system + user)
func (p *Prompt) GetPromptText() string {
	if p.System == "" {
		return p.User
	}
	return p.System + "\n\n" + p.User
} 