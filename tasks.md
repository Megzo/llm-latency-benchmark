# Development Tasks

## Phase 1: Project Setup & Core Structure ✅

### 1.1 Initialize Project
- [x] Create Go module: `go mod init llm-benchmark`
- [x] Set up directory structure according to project.md
- [x] Create basic .gitignore file
- [x] Add sample .env file with placeholder keys

### 1.2 Configuration System
- [x] Install dependencies: `go get gopkg.in/yaml.v3 github.com/joho/godotenv`
- [x] Create `internal/config/config.go` with main config struct
- [x] Implement `internal/config/models.go` for parsing models.yaml
- [x] Add environment variable loading
- [x] Create sample models.yaml file

### 1.3 Basic CLI Structure
- [x] Create `main.go` with flag parsing
- [x] Add flags: `--concurrent`, `--prompts`, `--output`, `--verbose`
- [x] Implement basic help and version commands
- [x] Add graceful shutdown handling

## Phase 2: Provider Interface & Core Types

### 2.1 Define Core Types
- [x] Create `providers/provider.go` with Provider interface
- [x] Define ChatRequest and ChatResponse structs
- [x] Add metrics collection structs in `internal/benchmark/metrics.go`
- [x] Create error types for different failure modes

### 2.2 Prompt Loading
- [x] Create prompt loader in `internal/config/prompts.go`
- [x] Add validation for prompt YAML format
- [x] Implement directory scanning for prompt files
- [x] Add sample prompt files in prompts/ directory

### 2.3 Benchmark Runner Core
- [x] Create `internal/benchmark/runner.go`
- [x] Implement sequential execution logic
- [x] Add concurrent execution with worker pools
- [x] Create context handling for cancellation

## Phase 3: Provider Implementations

### 3.1 OpenAI Provider
- [ ] Install OpenAI Go SDK: `go get github.com/sashabaranov/go-openai`
- [ ] Create `providers/openai.go`
- [ ] Implement streaming chat completions
- [ ] Add token counting from response
- [ ] Handle OpenAI-specific errors and retries

### 3.2 Groq Provider
- [ ] Research Groq API (likely OpenAI-compatible)
- [ ] Create `providers/groq.go`
- [ ] Implement streaming with custom HTTP client if needed
- [ ] Add Groq-specific configuration (base URL, etc.)
- [ ] Test token counting accuracy

### 3.3 Anthropic Provider
- [ ] Install Anthropic SDK or implement HTTP client
- [ ] Create `providers/anthropic.go`
- [ ] Handle Claude-specific streaming format
- [ ] Implement token counting
- [ ] Add Anthropic-specific retry logic

## Phase 4: Metrics & Timing

### 4.1 Precise Timing
- [ ] Implement high-resolution timing in `internal/benchmark/metrics.go`
- [ ] Add TTFT measurement with streaming
- [ ] Calculate tokens per second from streaming data
- [ ] Add total response time tracking

### 4.2 Cost Calculation
- [ ] Implement pricing calculation based on models.yaml
- [ ] Add cost per request and total cost tracking
- [ ] Handle different pricing models (input/output tokens)
- [ ] Add cost reporting in results

### 4.3 Error Handling & Retries
- [ ] Implement exponential backoff retry logic
- [ ] Add timeout handling for requests
- [ ] Create detailed error logging
- [ ] Add failure rate tracking

## Phase 5: Output & Logging

### 5.1 CSV Output
- [ ] Create `internal/output/csv.go`
- [ ] Define CSV schema with all metrics
- [ ] Implement file writing with proper headers
- [ ] Add timestamp and metadata to filename

### 5.2 Console Logging
- [ ] Create `internal/output/logger.go`
- [ ] Implement verbose logging with progress indicators
- [ ] Add colored output for better UX
- [ ] Create real-time statistics display

### 5.3 Results Summary
- [ ] Add summary statistics (avg, median, p95, p99)
- [ ] Create comparison between models
- [ ] Add cost analysis summary
- [ ] Implement error rate reporting

## Phase 6: Testing & Validation

### 6.1 Unit Tests
- [ ] Write tests for configuration parsing
- [ ] Test provider interface implementations
- [ ] Add benchmark runner tests
- [ ] Test metrics calculations

### 6.2 Integration Tests
- [ ] Test with real API calls (using test keys)
- [ ] Validate timing accuracy
- [ ] Test concurrent execution
- [ ] Verify CSV output format

### 6.3 Performance Testing
- [ ] Test with high concurrency
- [ ] Validate memory usage
- [ ] Test with large prompt files
- [ ] Benchmark the benchmarker itself

## Phase 7: Polish & Documentation

### 7.1 Error Messages & UX
- [ ] Improve error messages for common issues
- [ ] Add validation for configuration files
- [ ] Create helpful CLI help text
- [ ] Add progress bars for long runs

### 7.2 Documentation
- [ ] Create comprehensive README.md
- [ ] Add usage examples
- [ ] Document configuration options
- [ ] Add troubleshooting guide

### 7.3 Build & Release
- [ ] Add Makefile for building
- [ ] Create release scripts
- [ ] Add version information
- [ ] Test cross-platform builds

## Implementation Notes

### Key Design Decisions
1. **Streaming First**: Always use streaming to measure TTFT accurately
2. **Provider Abstraction**: Clean interface for easy provider addition
3. **Minimal Dependencies**: Use standard library where possible
4. **Precise Timing**: Use `time.Now()` with high resolution
5. **Graceful Failures**: Continue benchmarking even if some calls fail

### Critical Path
1. Basic CLI + Config loading
2. Provider interface + OpenAI implementation
3. Benchmark runner with metrics
4. CSV output
5. Add remaining providers

### Testing Strategy
- Unit tests for core logic
- Integration tests with real APIs
- Performance validation
- Error condition testing

### Performance Considerations
- Use worker pools for concurrency
- Minimize allocations in hot paths
- Efficient CSV writing
- Memory-conscious streaming handling