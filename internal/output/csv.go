package output

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/megzo/llm-latency-benchmark/internal/benchmark"
)

// CSVWriter handles writing benchmark results to CSV files
type CSVWriter struct {
	filepath string
}

// NewCSVWriter creates a new CSV writer
func NewCSVWriter(filepath string) *CSVWriter {
	return &CSVWriter{
		filepath: filepath,
	}
}

// WriteResults writes benchmark results to a CSV file
func (w *CSVWriter) WriteResults(results []benchmark.BenchmarkResult) error {
	// Ensure the directory exists
	dir := filepath.Dir(w.filepath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create the CSV file
	file, err := os.Create(w.filepath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Provider",
		"Model",
		"PromptFile",
		"StartTime",
		"FirstTokenTime",
		"EndTime",
		"TTFT_MS",
		"TotalTime_MS",
		"InputTokens",
		"OutputTokens",
		"TotalTokens",
		"TokensPerSecond",
		"Cost",
		"Success",
		"Error",
		"Response",
	}

	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, result := range results {
		row := []string{
			result.Provider,
			result.Model,
			result.PromptFile,
			result.StartTime.Format(time.RFC3339),
			result.FirstTokenTime.Format(time.RFC3339),
			result.EndTime.Format(time.RFC3339),
			fmt.Sprintf("%.2f", float64(result.TTFT.Microseconds())/1000.0), // Convert to milliseconds
			fmt.Sprintf("%.2f", float64(result.TotalTime.Microseconds())/1000.0), // Convert to milliseconds
			fmt.Sprintf("%d", result.InputTokens),
			fmt.Sprintf("%d", result.OutputTokens),
			fmt.Sprintf("%d", result.TotalTokens),
			fmt.Sprintf("%.2f", result.TokensPerSecond),
			fmt.Sprintf("%.6f", result.Cost),
			fmt.Sprintf("%t", result.Success),
			getErrorMessage(result.Error),
			truncateResponse(result.Response),
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// getErrorMessage safely extracts error message
func getErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// truncateResponse truncates response to reasonable length for CSV
func truncateResponse(response string) string {
	if len(response) > 1000 {
		return response[:1000] + "..."
	}
	return response
}

// GenerateOutputFilename generates a timestamped output filename
func GenerateOutputFilename(baseDir, prefix string) string {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.csv", prefix, timestamp)
	return filepath.Join(baseDir, filename)
} 