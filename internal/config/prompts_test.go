package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrompts(t *testing.T) {
	// Create a temporary prompts directory for testing
	tempDir := "test_prompts"
	defer os.RemoveAll(tempDir)

	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test prompts directory: %v", err)
	}

	// Create test prompt files
	promptFiles := map[string]string{
		"simple.yaml": `
system: "You are a helpful assistant."
user: "Hello, how are you today?"
`,
		"complex.yaml": `
system: "You are an expert analyst."
user: |
  Please analyze the following text and provide a detailed response:
  
  "The quick brown fox jumps over the lazy dog."
  
  Consider:
  1. Grammar and syntax
  2. Literary devices
  3. Historical context
`,
		"creative.yaml": `
user: "Write a short story about a robot learning to paint."
`,
	}

	for filename, content := range promptFiles {
		filepath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filepath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test prompt file %s: %v", filename, err)
		}
	}

	prompts, err := LoadPrompts(tempDir)
	if err != nil {
		t.Fatalf("LoadPrompts() failed: %v", err)
	}

	if len(prompts) != 3 {
		t.Errorf("Expected 3 prompts, got %d", len(prompts))
	}

	// Check specific prompts
	var simplePrompt, complexPrompt, creativePrompt PromptFile
	for _, p := range prompts {
		switch p.Name {
		case "simple":
			simplePrompt = p
		case "complex":
			complexPrompt = p
		case "creative":
			creativePrompt = p
		}
	}

	if simplePrompt.Name == "" {
		t.Fatal("Simple prompt not found")
	}

	if simplePrompt.Prompt.User != "Hello, how are you today?" {
		t.Errorf("Expected user prompt 'Hello, how are you today?', got '%s'", simplePrompt.Prompt.User)
	}

	if simplePrompt.Prompt.System != "You are a helpful assistant." {
		t.Errorf("Expected system prompt 'You are a helpful assistant.', got '%s'", simplePrompt.Prompt.System)
	}

	if complexPrompt.Name == "" {
		t.Fatal("Complex prompt not found")
	}

	// Check that the multiline prompt was loaded correctly
	if len(complexPrompt.Prompt.User) < 100 {
		t.Errorf("Complex prompt seems too short: %d characters", len(complexPrompt.Prompt.User))
	}

	if creativePrompt.Name == "" {
		t.Fatal("Creative prompt not found")
	}

	if creativePrompt.Prompt.System != "" {
		t.Errorf("Expected empty system prompt, got '%s'", creativePrompt.Prompt.System)
	}
}

func TestLoadPrompts_EmptyDirectory(t *testing.T) {
	tempDir := "test_empty_prompts"
	defer os.RemoveAll(tempDir)

	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test prompts directory: %v", err)
	}

	prompts, err := LoadPrompts(tempDir)
	if err != nil {
		t.Fatalf("LoadPrompts() failed with empty directory: %v", err)
	}

	if len(prompts) != 0 {
		t.Errorf("Expected 0 prompts for empty directory, got %d", len(prompts))
	}
}

func TestLoadPrompts_NonexistentDirectory(t *testing.T) {
	_, err := LoadPrompts("nonexistent_directory")
	if err == nil {
		t.Error("LoadPrompts() should fail with nonexistent directory")
	}
}

func TestLoadPrompts_InvalidYAML(t *testing.T) {
	tempDir := "test_invalid_prompts"
	defer os.RemoveAll(tempDir)

	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test prompts directory: %v", err)
	}

	// Create an invalid YAML file
	invalidYAML := `
name: Invalid Test
description: A test with invalid YAML
prompt: "This is a test"
  invalid: indentation: here
`

	filepath := filepath.Join(tempDir, "invalid.yaml")
	err = os.WriteFile(filepath, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid YAML file: %v", err)
	}

	_, err = LoadPrompts(tempDir)
	if err == nil {
		t.Error("LoadPrompts() should fail with invalid YAML")
	}
}

func TestLoadPrompts_NonYAMLFiles(t *testing.T) {
	tempDir := "test_mixed_prompts"
	defer os.RemoveAll(tempDir)

	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test prompts directory: %v", err)
	}

	// Create a valid YAML file
	validYAML := `
system: "You are a helpful assistant."
user: "This is a valid prompt"
`

	validFile := filepath.Join(tempDir, "valid.yaml")
	err = os.WriteFile(validFile, []byte(validYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create valid YAML file: %v", err)
	}

	// Create a non-YAML file
	nonYAMLFile := filepath.Join(tempDir, "readme.txt")
	err = os.WriteFile(nonYAMLFile, []byte("This is not a YAML file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create non-YAML file: %v", err)
	}

	prompts, err := LoadPrompts(tempDir)
	if err != nil {
		t.Fatalf("LoadPrompts() failed: %v", err)
	}

	// Should only load the YAML file
	if len(prompts) != 1 {
		t.Errorf("Expected 1 prompt (only YAML files), got %d", len(prompts))
	}

	found := false
	for _, p := range prompts {
		if p.Name == "valid" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Valid prompt not found")
	}
}

func TestPrompt_Validate(t *testing.T) {
	tests := []struct {
		name    string
		prompt  Prompt
		wantErr bool
	}{
		{
			name: "valid prompt with user only",
			prompt: Prompt{
				User: "Hello, world!",
			},
			wantErr: false,
		},
		{
			name: "valid prompt with system and user",
			prompt: Prompt{
				System: "You are a helpful assistant.",
				User:   "Hello, world!",
			},
			wantErr: false,
		},
		{
			name: "missing user prompt",
			prompt: Prompt{
				System: "You are a helpful assistant.",
				User:   "",
			},
			wantErr: true,
		},
		{
			name: "whitespace-only user prompt",
			prompt: Prompt{
				System: "You are a helpful assistant.",
				User:   "   \n\t   ",
			},
			wantErr: true,
		},
		{
			name: "empty prompt",
			prompt: Prompt{
				System: "",
				User:   "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePrompt(tt.prompt)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePrompt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadPrompts_Subdirectories(t *testing.T) {
	tempDir := "test_subdir_prompts"
	defer os.RemoveAll(tempDir)

	err := os.MkdirAll(filepath.Join(tempDir, "subdir"), 0755)
	if err != nil {
		t.Fatalf("Failed to create test prompts directory: %v", err)
	}

	// Create a prompt in the subdirectory
	subdirPrompt := `
system: "You are a helpful assistant."
user: "This is in a subdirectory"
`

	subdirFile := filepath.Join(tempDir, "subdir", "subdir.yaml")
	err = os.WriteFile(subdirFile, []byte(subdirPrompt), 0644)
	if err != nil {
		t.Fatalf("Failed to create subdir prompt file: %v", err)
	}

	prompts, err := LoadPrompts(tempDir)
	if err != nil {
		t.Fatalf("LoadPrompts() failed: %v", err)
	}

	// Should not load prompts from subdirectories
	if len(prompts) != 0 {
		t.Errorf("Expected 0 prompts (subdirectories should be ignored), got %d", len(prompts))
	}
} 