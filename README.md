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
  gpt-4.1:
    token_price:
      input: 2.0   # $ per million tokens
      output: 8.0
    parameters: {}
  gpt-4.1-mini:
    token_price:
      input: 0.4
      output: 1.6
    parameters: {}
  gpt-4.1-nano:
    token_price:
      input: 0.1
      output: 0.4
    parameters: {}

openai_responses:
  gpt-5-mini:
    token_price:
      input: 0.25
      output: 2.0
    parameters:
      text:
        format:
          type: text
        verbosity: low
      reasoning:
        effort: minimal
        summary: null
  gpt-5-chat-latest:
    token_price:
      input: 1.0
      output: 8.0
    parameters:
      temperature: 0.7
      top_p: 0.9
      max_output_tokens: 4096
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

# Number of runs from each prompt
./llm-benchmark --runs 10

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