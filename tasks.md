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

## Phase 3: First Provider Implementations

### 3.1 OpenAI Provider ✅
- [x] Install the official OpenAI Go SDK: `https://github.com/openai/openai-go`
- [x] Create `providers/openai.go`
- [x] Implement streaming chat completions
- [x] Add token counting from response
- [x] Handle OpenAI-specific errors and retries

## Phase 4: Metrics & Timing ✅

### 4.1 Precise Timing ✅
- [x] Implement high-resolution timing in `internal/benchmark/metrics.go`
- [x] Add TTFT measurement with streaming
- [x] Calculate tokens per second from streaming data
- [x] Add total response time tracking

### 4.2 Cost Calculation ✅
- [x] Implement pricing calculation based on models.yaml
- [x] Add cost per request and total cost tracking
- [x] Handle different pricing models (input/output tokens)
- [x] Add cost reporting in results

### 4.3 Error Handling & Retries ✅
- [x] Implement exponential backoff retry logic
- [x] Add timeout handling for requests
- [x] Create detailed error logging
- [x] Add failure rate tracking

## Phase 5: Output & Logging ✅

### 5.1 CSV Output ✅
- [x] Create `internal/output/csv.go`
- [x] Define CSV schema with all metrics
- [x] Implement file writing with proper headers
- [x] Add timestamp and metadata to filename

### 5.2 Console Logging ✅
- [x] Create `internal/output/logger.go`
- [x] Implement verbose logging with progress indicators
- [x] Add colored output for better UX
- [x] Create real-time statistics display

### 5.3 Results Summary ✅
- [x] Add summary statistics (avg, median, p95, p99)
- [x] Create comparison between models
- [x] Add cost analysis summary
- [x] Implement error rate reporting

## Phase 6: Testing & Validation ✅

### 6.1 Unit Tests ✅
- [x] Write tests for configuration parsing
- [x] Test provider interface implementations
- [x] Add benchmark runner tests
- [x] Test metrics calculations

### 6.2 Integration Tests ✅
- [x] Test with real API calls (using test keys)
- [x] Validate timing accuracy
- [x] Test concurrent execution
- [x] Verify CSV output format

### 6.3 Performance Testing ✅
- [x] Test with high concurrency
- [x] Validate memory usage
- [x] Test with large prompt files
- [x] Benchmark the benchmarker itself

## Phase 7: Other provider implementations

### 7.1 Groq Provider
- [ ] Research Groq API (OpenAI-compatible)
- [ ] Create `providers/groq.go`
- [ ] Implement streaming with custom HTTP client if needed
- [ ] Add Groq-specific configuration (base URL, etc.)
- [ ] Test token counting accuracy

### 7.2 Anthropic Provider ✅
- [x] Install Anthropic SDK at `https://github.com/anthropics/anthropic-sdk-go`
- [x] Create `providers/anthropic.go`
- [x] Handle Claude-specific streaming format
- [x] Implement token counting
- [x] Add Anthropic-specific retry logic

### 7.3 Azure OpenAI Provider
- [ ] Install the official Azure OpenAI SDK: `https://github.com/Azure/azure-sdk-for-go/`
- [ ] Create `providers/azure_openai.go`
- [ ] Implement streaming chat completions with Azure endpoints
- [ ] Add Azure-specific configuration (endpoint, deployment name, API version)
- [ ] Handle Azure-specific authentication and error handling
- [ ] Test token counting accuracy with Azure models

### 7.4 Google Gemini Provider
- [ ] Install the official Google Gemini SDK: `https://github.com/googleapis/go-genai`
- [ ] Create `providers/gemini.go`
- [ ] Implement streaming with Gemini's specific format
- [ ] Handle Gemini-specific model names and parameters
- [ ] Add Google Cloud authentication handling
- [ ] Implement token counting for Gemini models

### 7.5 Fireworks.ai Provider
- [ ] Create `providers/fireworks.go`
- [ ] Implement OpenAI-compatible API client (no additional SDK needed)
- [ ] Add Fireworks.ai specific configuration (base URL, API key)
- [ ] Handle Fireworks.ai model names and parameters
- [ ] Implement streaming with OpenAI-compatible format
- [ ] Test token counting accuracy

## Phase 8: Polish & Documentation

### 8.1 Error Messages & UX
- [ ] Improve error messages for common issues
- [ ] Add validation for configuration files
- [ ] Create helpful CLI help text
- [ ] Add progress bars for long runs

### 8.2 Documentation
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