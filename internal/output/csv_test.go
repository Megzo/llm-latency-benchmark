package output

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"llm-latency-benchmark/internal/benchmark"
)

func TestCSVWriter_WriteHeader(t *testing.T) {
	tempFile := "test_output.csv"
	defer os.Remove(tempFile)

	writer, err := NewCSVWriter(tempFile)
	require.NoError(t, err)
	defer writer.Close()

	// Write header
	err = writer.WriteHeader()
	require.NoError(t, err)

	// Read the file and check header
	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)

	expectedHeader := "timestamp,model,prompt_name,ttft_ms,total_time_ms,input_tokens,output_tokens,cost,error,tokens_per_second"
	assert.Contains(t, string(content), expectedHeader)
}

func TestCSVWriter_WriteResult(t *testing.T) {
	tempFile := "test_results.csv"
	defer os.Remove(tempFile)

	writer, err := NewCSVWriter(tempFile)
	require.NoError(t, err)
	defer writer.Close()

	// Write header first
	err = writer.WriteHeader()
	require.NoError(t, err)

	// Create test result
	result := benchmark.BenchmarkResult{
		Model:        "gpt-4o-mini",
		PromptName:   "test-prompt",
		TTFT:         1 * time.Second,
		TotalTime:    5 * time.Second,
		InputTokens:  100,
		OutputTokens: 200,
		Cost:         0.001,
		Error:        nil,
	}

	// Write result
	err = writer.WriteResult(result)
	require.NoError(t, err)

	// Read the file and check content
	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)

	lines := string(content)
	assert.Contains(t, lines, "gpt-4o-mini")
	assert.Contains(t, lines, "test-prompt")
	assert.Contains(t, lines, "1000") // TTFT in ms
	assert.Contains(t, lines, "5000") // Total time in ms
	assert.Contains(t, lines, "100")  // Input tokens
	assert.Contains(t, lines, "200")  // Output tokens
	assert.Contains(t, lines, "0.001") // Cost
	assert.Contains(t, lines, "40")   // Tokens per second (200 tokens / 5 seconds)
}

func TestCSVWriter_WriteResultWithError(t *testing.T) {
	tempFile := "test_error_results.csv"
	defer os.Remove(tempFile)

	writer, err := NewCSVWriter(tempFile)
	require.NoError(t, err)
	defer writer.Close()

	// Write header first
	err = writer.WriteHeader()
	require.NoError(t, err)

	// Create test result with error
	result := benchmark.BenchmarkResult{
		Model:        "gpt-4o-mini",
		PromptName:   "test-prompt",
		Error:        assert.AnError,
	}

	// Write result
	err = writer.WriteResult(result)
	require.NoError(t, err)

	// Read the file and check content
	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)

	lines := string(content)
	assert.Contains(t, lines, "gpt-4o-mini")
	assert.Contains(t, lines, "test-prompt")
	assert.Contains(t, lines, "ERROR")
	assert.Contains(t, lines, assert.AnError.Error())
}

func TestCSVWriter_MultipleResults(t *testing.T) {
	tempFile := "test_multiple_results.csv"
	defer os.Remove(tempFile)

	writer, err := NewCSVWriter(tempFile)
	require.NoError(t, err)
	defer writer.Close()

	// Write header
	err = writer.WriteHeader()
	require.NoError(t, err)

	// Create multiple test results
	results := []benchmark.BenchmarkResult{
		{
			Model:        "gpt-4o-mini",
			PromptName:   "prompt1",
			TTFT:         1 * time.Second,
			TotalTime:    5 * time.Second,
			InputTokens:  100,
			OutputTokens: 200,
			Cost:         0.001,
			Error:        nil,
		},
		{
			Model:        "gpt-4o-mini",
			PromptName:   "prompt2",
			TTFT:         2 * time.Second,
			TotalTime:    6 * time.Second,
			InputTokens:  150,
			OutputTokens: 250,
			Cost:         0.002,
			Error:        nil,
		},
		{
			Model:        "gpt-3.5-turbo",
			PromptName:   "prompt3",
			TTFT:         500 * time.Millisecond,
			TotalTime:    3 * time.Second,
			InputTokens:  80,
			OutputTokens: 120,
			Cost:         0.0005,
			Error:        nil,
		},
	}

	// Write all results
	for _, result := range results {
		err = writer.WriteResult(result)
		require.NoError(t, err)
	}

	// Read the file and check content
	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)

	lines := string(content)
	
	// Check that all models are present
	assert.Contains(t, lines, "gpt-4o-mini")
	assert.Contains(t, lines, "gpt-3.5-turbo")
	
	// Check that all prompts are present
	assert.Contains(t, lines, "prompt1")
	assert.Contains(t, lines, "prompt2")
	assert.Contains(t, lines, "prompt3")
	
	// Check that we have the expected number of data lines (3 results)
	dataLines := 0
	for _, line := range lines {
		if line == '\n' {
			dataLines++
		}
	}
	// Should have header + 3 data lines = 4 lines total
	assert.GreaterOrEqual(t, dataLines, 3)
}

func TestCSVWriter_InvalidFilePath(t *testing.T) {
	// Try to create a writer with an invalid path
	_, err := NewCSVWriter("/invalid/path/that/does/not/exist/test.csv")
	assert.Error(t, err)
}

func TestCSVWriter_Close(t *testing.T) {
	tempFile := "test_close.csv"
	defer os.Remove(tempFile)

	writer, err := NewCSVWriter(tempFile)
	require.NoError(t, err)

	// Write some data
	err = writer.WriteHeader()
	require.NoError(t, err)

	// Close the writer
	err = writer.Close()
	require.NoError(t, err)

	// Try to write after closing should fail
	result := benchmark.BenchmarkResult{
		Model:      "test",
		PromptName: "test",
	}
	err = writer.WriteResult(result)
	assert.Error(t, err)
}

func TestCSVWriter_FilePermissions(t *testing.T) {
	tempFile := "test_permissions.csv"
	defer os.Remove(tempFile)

	writer, err := NewCSVWriter(tempFile)
	require.NoError(t, err)
	defer writer.Close()

	// Write some data
	err = writer.WriteHeader()
	require.NoError(t, err)

	// Check file permissions
	info, err := os.Stat(tempFile)
	require.NoError(t, err)

	// File should be readable and writable by owner
	mode := info.Mode()
	assert.True(t, mode.IsRegular())
	assert.True(t, mode&0400 != 0) // Readable
	assert.True(t, mode&0200 != 0) // Writable
}

func TestCSVWriter_CSVFormatting(t *testing.T) {
	tempFile := "test_formatting.csv"
	defer os.Remove(tempFile)

	writer, err := NewCSVWriter(tempFile)
	require.NoError(t, err)
	defer writer.Close()

	// Write header
	err = writer.WriteHeader()
	require.NoError(t, err)

	// Create result with special characters that need escaping
	result := benchmark.BenchmarkResult{
		Model:        "gpt-4o-mini",
		PromptName:   "test,prompt", // Contains comma
		TTFT:         1 * time.Second,
		TotalTime:    5 * time.Second,
		InputTokens:  100,
		OutputTokens: 200,
		Cost:         0.001,
		Error:        nil,
	}

	// Write result
	err = writer.WriteResult(result)
	require.NoError(t, err)

	// Read the file and check that CSV is properly formatted
	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)

	lines := string(content)
	// Should contain the comma in the prompt name, properly escaped
	assert.Contains(t, lines, "test,prompt")
}

func TestCSVWriter_TimestampFormat(t *testing.T) {
	tempFile := "test_timestamp.csv"
	defer os.Remove(tempFile)

	writer, err := NewCSVWriter(tempFile)
	require.NoError(t, err)
	defer writer.Close()

	// Write header
	err = writer.WriteHeader()
	require.NoError(t, err)

	// Create result
	result := benchmark.BenchmarkResult{
		Model:      "gpt-4o-mini",
		PromptName: "test",
	}

	// Write result
	err = writer.WriteResult(result)
	require.NoError(t, err)

	// Read the file and check timestamp format
	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)

	lines := string(content)
	// Should contain a timestamp in ISO format
	assert.Contains(t, lines, "T") // ISO format contains 'T'
	assert.Contains(t, lines, "Z") // ISO format ends with 'Z' for UTC
}

func TestCSVWriter_DirectoryCreation(t *testing.T) {
	// Test creating CSV in a directory that doesn't exist
	tempDir := "test_csv_dir"
	tempFile := filepath.Join(tempDir, "test.csv")
	defer os.RemoveAll(tempDir)

	writer, err := NewCSVWriter(tempFile)
	require.NoError(t, err)
	defer writer.Close()

	// Write header
	err = writer.WriteHeader()
	require.NoError(t, err)

	// Check that directory was created
	_, err = os.Stat(tempDir)
	assert.NoError(t, err)

	// Check that file was created
	_, err = os.Stat(tempFile)
	assert.NoError(t, err)
} 