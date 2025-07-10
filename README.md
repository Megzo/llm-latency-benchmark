# LLM Benchmark Tool

A Go-based command-line tool for measuring LLM latency and performance metrics across multiple providers, specifically designed for real-time use cases.

## Project Structure

```
llm-benchmark/
├── main.go                 # Entry point and CLI handling
├── .env                    # API keys and secrets
├── models.yaml             # Model definitions and pricing
├── go.mod                  # Go module file
├── go.sum                  # Go dependencies
├── README.md               # This file
├── tasks.md                # Development tasks
├── prompts/                # Test prompts in YAML format
│   ├── simple.yaml
│   ├── complex.yaml
│   └── creative.yaml
├── providers/              # Provider implementations
│   ├── provider.go         # Provider interface
│   ├── openai.go          # OpenAI implementation
│   ├── groq.go            # Groq implementation
│   └── anthropic.go       # Anthropic implementation
├── internal/               # Internal packages
│   ├── config/            # Configuration handling
│   │   ├── config.go      # Main config struct
│   │   └── models.go      # Models config parsing
│   ├── benchmark/         # Benchmarking logic
│   │   ├── runner.go      # Benchmark runner
│   │   └── metrics.go     # Metrics collection
│   └── output/            # Output formatting
│       ├── csv.go         # CSV output
│       └── logger.go      # Console logging
└── results/               # Generated CSV files
    └── benchmark_YYYY-MM-DD_HH-MM-SS.csv
```

## Features

### Core Metrics
- **Time to First Token (TTFT)**: From request start to first streaming token
- **Total Response Time**: Complete request-response cycle
- **Tokens per Second**: Output tokens only (calculated from streaming)
- **Token Counts**: Input, output, and total tokens
- **Cost Calculation**: Based on provider pricing
- **Response Content**: Full LLM response

### Execution Modes
- **Sequential**: One request at a time (`--concurrent 1` or default)
- **Concurrent**: Multiple simultaneous requests (`--concurrent N`)

### Output Formats
- **CSV**: Structured data for analysis
- **Console**: Verbose logging with real-time progress

## Configuration Files

### .env
```env
OPENAI_API_KEY=sk-...
GROQ_API_KEY=gsk_...
ANTHROPIC_API_KEY=sk-ant-...
```

### models.yaml
```yaml
openai:
  gpt-4-turbo:
    input: 10.0   # $ per million tokens
    output: 30.0
  gpt-3.5-turbo:
    input: 0.5
    output: 1.5
groq:
  llama-3-70b:
    input: 0.8
    output: 0.8
  mixtral-8x7b:
    input: 0.27
    output: 0.27
anthropic:
  claude-3-sonnet:
    input: 3.0
    output: 15.0
```

### Prompt Format (prompts/*.yaml)
```yaml
system: |
  You are a helpful assistant.
user: |
  What is your name?
```

## CLI Usage

```bash
# Basic usage (sequential)
./llm-benchmark

# Concurrent execution
./llm-benchmark --concurrent 4

# Specify prompts directory
./llm-benchmark --prompts ./custom-prompts

# Custom output file
./llm-benchmark --output results/my-benchmark.csv

# Verbose logging
./llm-benchmark --verbose
```

## Dependencies

- **Minimal approach**: Use standard library where possible
- **Provider SDKs**: Official Go libraries when available
- **Configuration**: YAML parsing (gopkg.in/yaml.v3)
- **Environment**: godotenv for .env loading

## Architecture

### Provider Interface
```go
type Provider interface {
    Name() string
    StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error)
    TokenCount(response ChatResponse) (input, output, total int)
}
```

### Metrics Collection
- Start timer on request
- Record first token timestamp
- Track streaming tokens
- Calculate final metrics

### Error Handling
- Retry logic with exponential backoff
- Timeout handling
- Graceful degradation
- Detailed error logging

## Development Goals

1. **Performance**: Minimal overhead for accurate latency measurements
2. **Reliability**: Robust error handling and retry mechanisms
3. **Extensibility**: Easy to add new providers
4. **Usability**: Clear CLI interface and comprehensive logging
5. **Accuracy**: Precise timing measurements for real-time applications