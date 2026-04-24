# ManageAI Go Client

A Go client library for interacting with the ManageAI AI Router service. This package provides a comprehensive interface for working with various AI models and services through the ManageAI platform.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Components](#core-components)
  - [Client](#client)
  - [Messages](#messages)
  - [Message Creation Utilities](#message-creation-utilities)
  - [Chat Completion Options](#chat-completion-options)
- [API Endpoints](#api-endpoints)
- [Message Types](#message-types)
- [Model Aliases and Redirection](#model-aliases-and-redirection)
- [Error Handling](#error-handling)
- [Advanced Features](#advanced-features)
- [Response Structures](#response-structures)
- [Testing](#testing)
- [Dependencies](#dependencies)
- [Environment Variables](#environment-variables)
- [Best Practices](#best-practices)
- [Changelog](#changelog)
- [Release Notes](#release-notes)

## Features

- **Chat Completions**: Full support for chat completions with multiple models
- **Model Management**: List available models and validate tokens
- **Rate Limiting**: Built-in rate limit handling and quota monitoring
- **Model Redirection**: Automatic model alias resolution and redirection
- **Multi-modal Support**: Handle text and image content in messages
- **Tool Integration**: Support for function calling and tool usage
- **Streaming**: Real-time response streaming capabilities
- **Error Handling**: Comprehensive error types and validation

## Installation

To install the ManageAI Go client, use `go get` to add it to your project:

```bash
go get github.com/manageai-inet/Eino-ManageAI-Extension/components/core/manageai@latest
```

Or to use a specific version:

```bash
go get github.com/manageai-inet/Eino-ManageAI-Extension/components/core/manageai@v1.0.0
```

After installation, run `go mod tidy` to ensure all dependencies are properly resolved:

```bash
go mod tidy
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "github.com/manageai-inet/Eino-ManageAI-Extension/components/core/manageai"
)

func main() {
    // Initialize client from environment variables
    client, err := manageai.GetDefaultClient()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Simple chat completion
    messages := []any{
        "Hello, how are you today?",
    }

    options := &manageai.ChatCompletionOptions{}
    response, err := client.ChatCompletion(messages, options)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(*response.Choices[0].Message.Content)
}
```

## Core Components

### Client

The main entry point for all API interactions:

```go
modelId := "Qwen3.5"
apiKey := "your-api-key-here"

// Create a new client
client, err := manageai.NewClient(&modelId, apiKey)

// Configure client options
client.WithTimeout(30 * time.Second)
client.WithRetryAttempt(3)
client.WithRetryDelay(1 * time.Second)
```

### Messages

Support for various message types and roles:

```go
// Simple text messages
messages := []any{
    "Hello!",
    "How can you help me today?",
}

// Structured messages with roles
messages := []any{
    manageai.NewSystemMessage("You are a helpful assistant.", nil),
    manageai.NewHumanMessage("What is the weather like?", nil),
}

// Multi-modal messages with images
contentBlocks := []manageai.ContentBlock{
    manageai.NewTextContentBlock("Describe this image:"),
    manageai.NewImageUrlContentBlock("https://example.com/image.jpg"),
}
messages := []any{
    manageai.NewMultiModalHumanMessage(contentBlocks, nil),
}

// Utility functions for common message patterns
messages := []any{
    manageai.NewSystemMessage("You are a helpful assistant.", nil),
    "What is the capital of France?", // Simple human message
    manageai.NewAssistantMessage("The capital of France is Paris.", nil, nil, nil),
}
```

### Message Creation Utilities

The package provides several utility functions for creating different types of messages:

```go
// Create system message
systemMsg := manageai.NewSystemMessage("You are a helpful assistant.", nil)

// Create human/user message from string
humanMsg := manageai.NewHumanMessage("Hello!", nil)

// Create multi-modal message with text and image
contentBlocks := []manageai.ContentBlock{
    manageai.NewTextContentBlock("Describe this image:"),
    manageai.NewImageUrlContentBlock("https://example.com/image.jpg"),
}
multiModalMsg := manageai.NewMultiModalHumanMessage(contentBlocks, nil)

// Create assistant message
assistantMsg := manageai.NewAssistantMessage("I'm doing great!", nil, nil, nil)

// Create tool message
toolCallId := "tool_call_id"
toolMsg := manageai.NewToolMessage("The weather is sunny", nil, &toolCallId)
```

### Chat Completion Options

Extensive configuration options for chat completions:

```go
options := &manageai.ChatCompletionOptions{}

// Model selection
options.WithModel("Qwen3.5")

// Generation parameters
options.WithTemperature(0.7)
options.WithMaxTokens(1000)
options.WithTopP(0.9)

// Sampling controls
options.WithFrequencyPenalty(0.5)
options.WithPresencePenalty(0.5)

// Stop sequences
stop := []string{"\n\n", "User:"}
options.WithStop(stop)

// Response format
format := "json_object"
options.WithResponseFormat(&format)

// Function calling
tools := []manageai.Tools{
    {
        Type: "function",
        Function: manageai.ToolDefinition{
            Name:        "get_weather",
            Description: "Get current weather information",
            Parameters: map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "location": map[string]any{
                        "type":        "string",
                        "description": "The city and state",
                    },
                },
                "required": []string{"location"},
            },
        },
    },
}
toolChoice := "auto"
options.WithTools(tools, &toolChoice)
```

### Tool Definition Enhancement Methods

The `ToolDefinition` struct provides helper methods for enhancing JSON Schema properties:

```go
toolDef := manageai.ToolDefinition{
    Name:        "user_validator",
    Description: "Validate user information",
    Parameters: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "username": map[string]any{
                "type": "string",
            },
            "age": map[string]any{
                "type": "integer",
            },
            "user": map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "profile": map[string]any{
                        "type": "object",
                        "properties": map[string]any{
                            "email": map[string]any{
                                "type": "string",
                            },
                        },
                    },
                },
            },
        },
    },
}

// Add enum restrictions to properties
toolDef.WithEnum("username", []any{"admin", "user", "guest"})

// Add numeric range constraints
var maxAge float64 = 120
var minAge float64 = 0
toolDef.WithRange("age", &maxAge, &minAge)

// Add exclusive range constraints
var maxScore float64 = 100
var minScore float64 = 0
toolDef.WithExclusiveRange("score", &maxScore, &minScore)

// Add multiple-of constraint
var multipleOf float64 = 5
toolDef.WithMultipleOf("points", multipleOf)

// Add string length constraints
var minLength int = 3
var maxLength int = 20
toolDef.WithLength("username", &minLength, &maxLength)

// Add string pattern validation
toolDef.WithPattern("username", "^[a-zA-Z0-9_]+$")
```

#### Nested Property Support

All enhancement methods support dot-separated nested property paths:

```go
// Enhance deeply nested properties
toolDef.WithPattern("user.profile.email", "^[^@]+@[^@]+\\.[^@]+$")
toolDef.WithRange("user.settings.max_attempts", &maxAttempts, nil)
```

#### Property Validation

The underlying `getProp` function ensures robust property validation:
- Proper JSON Schema navigation through nested properties
- Type validation with clear error messages
- Support for all JSON Schema primitive types
- Graceful error handling for missing or malformed properties

### Struct-to-Tool Conversion and Tool Call Handling

The package provides powerful utilities for converting Go structs to tool definitions and handling tool call arguments:

#### StructToTool

Automatically generate tool definitions from Go structs using reflection and JSON schema inference:

```go
// Define your struct with JSON tags and validation
type WeatherRequest struct {
    Location string  `json:"location" jsonschema:"The city and state to get weather for"` // required by default
    Units    string  `json:"units,omitempty" jsonschema:"The temperature units to use"` // omitempty or omitzero will infer as optional
}

// Convert struct to tool definition
tool, err := manageai.StructToTool[WeatherRequest](
    "get_weather", 
    "Get current weather information for a location",
    true, // strict mode
)
if err != nil {
    log.Fatal(err)
}

toolDef.WithEnum("units", []any{"celsius", "fahrenheit"})

// Use the tool in chat completion options
tools := []manageai.Tools{tool}
options := &manageai.ChatCompletionOptions{}
options.WithTools(tools, nil)
```

Benefits of `StructToTool`:
- **Automatic Schema Generation**: No manual JSON schema writing required
- **Type Safety**: Compile-time struct validation
- **Nested Struct Support**: Handles complex nested structures automatically
- **Slice and Map Support**: Works with arrays and maps in structs
- **Validation Tags**: Supports jsonschema tags for additional constraints
- **Strict Mode**: Optional strict JSON schema compliance

#### HandleToolCall

Parse and validate tool call arguments back into Go structs:

```go
// Example tool call arguments from LLM
jsonArgs := `{"location": "New York, NY", "units": "celsius"}`

// Parse arguments into your struct
weatherReq, err := manageai.HandleToolCall[WeatherRequest](jsonArgs)
if err != nil {
    log.Printf("Failed to parse tool arguments: %v", err)
    return
}

// Use the parsed struct
fmt.Printf("Location: %s\n", weatherReq.Location)
fmt.Printf("Units: %s\n", weatherReq.Units)

// Process the request and return results
weatherData := getWeather(weatherReq.Location, weatherReq.Units)
```

Benefits of `HandleToolCall`:
- **Type-Safe Parsing**: Converts JSON to strongly-typed Go structs
- **Automatic Validation**: Validates against struct field types
- **Error Handling**: Clear error messages for malformed JSON
- **Generic Support**: Works with any Go struct type

## API Endpoints

### Chat Completions

```go
response, err := client.ChatCompletion(messages, options)
```

### Tokenization and Detokenization

```go
// Tokenize text into tokens
text := "Hello, world!"
tokensResponse, err := client.Tokenize(text, nil)

// Detokenize tokens back to text
tokens := []int{151644, 872, 198, 25817, 151645}
detokenizeOptions := &DetokenizeOptions{}
detokenizeOptions.WithModel("qwen3-235b-a22b")
detokenizeResponse, err := client.Detokenize(tokens, detokenizeOptions)
```

### Embedding Generation

```go
// Generate embeddings for text
input := []string{"This is a test sentence."}
embeddingOptions := &EmbeddingOptions{}
embeddingOptions.WithModel("Qwen3-Embedding-8B")
embeddingResponse, err := client.Embedding(input, embeddingOptions)
```

### Reranking

```go
// Rerank documents based on query relevance
query := "artificial intelligence"
documents := []string{
    "AI is transforming healthcare.",
    "The weather today is sunny.",
    "Machine learning is a subset of AI.",
}
rerankOptions := &RerankOptions{}
rerankOptions.WithModel("Qwen3-Reranker-8B")
rerankResponse, err := client.Rerank(query, documents, rerankOptions)
```

### Model Management

```go
// List available models
models, err := client.ListModels()

// Validate API token
validationInfo, err := client.ValidateToken()

// Get rate limits
rateLimits, err := client.InspectRateLimits()

// Get token usage
usage, err := client.InspectTokensUsage()

// Health check
healthy, err := client.Health()

// Get version
version, err := client.Version()
```

## Message Types

### System Messages

Provide context and instructions to the AI:

```go
systemMsg := manageai.NewSystemMessage("You are a helpful assistant that speaks like a pirate.", nil)
```

### Human/User Messages

Input from the user:

```go
// Simple text message
humanMsg := manageai.NewHumanMessage("Hello!", nil)

// Message with image
content := "What's in this image?"
imageUrl := "https://example.com/image.jpg"
humanMsg := manageai.NewHumanMessageWithImage(&content, &imageUrl, nil)
```

### Assistant Messages

Responses from the AI:

```go
assistantMsg := manageai.NewAssistantMessage("I'm doing great!", nil, nil, nil)
```

### Tool Messages

Function call results:

```go
toolCallId := "tool_call_id"
toolMsg := manageai.NewToolMessage("The weather is sunny", nil, &toolCallId)
```

## Model Aliases and Redirection

The package supports model aliases and automatic redirection:

### Supported Aliases

| Alias | Resolves To |
|-------|-------------|
| qwen2.5-72b-instruct | qwen/qwen2.5-72b-instruct |
| qwen3 | qwen/qwen3-vl-235b-a22b-instruct |
| qwen3.5 | qwen/qwen3.5-397b-a17b-non_thinking |
| medgemma | google/medgemma-27b-it |

### Automatic Redirection

Models are automatically redirected based on capabilities:
- Multi-tool scenarios redirect to more capable models
- Deprecated models redirect to maintained alternatives
- Fuzzy matching suggests corrections for typos

## Error Handling

Comprehensive error types for different scenarios:

```go
import "github.com/manageai-inet/Eino-ManageAI-Extension/components/core/manageai"

// API errors
if apiErr, ok := err.(*manageai.APIError); ok {
    fmt.Printf("API Error %d: %s\n", apiErr.Code, apiErr.Message)
}

// Validation errors
if valErr, ok := err.(*manageai.ValidationError); ok {
    fmt.Printf("Validation Error in field '%s': %s\n", valErr.Field, valErr.Message)
}
```

## Advanced Features

### Configuration

The client can be configured with various options:

```go
// Set custom timeouts
client.WithTimeout(60 * time.Second)

// Configure retry behavior
client.WithRetryAttempt(5)
client.WithRetryDelay(2 * time.Second)
```

### Streaming Responses

Stream responses in real-time as they're generated:

```go
stream, err := client.ChatCompletionStream(messages, options)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for {
    chunk, err := stream.Recv()
    if err != nil {
        if err == io.EOF {
            break
        }
        log.Fatal(err)
    }
    
    if chunk.Choices[0].Delta.Content != nil {
        fmt.Print(*chunk.Choices[0].Delta.Content)
    }
}
```

### Rate Limit Handling

```go
// Check rate limits
if err := response.ResponseHeaders.IsExceedRateLimit(); err != nil {
    log.Printf("Rate limit exceeded: %v", err)
}

// Check quota status
if err := response.ResponseHeaders.IsInsufficientQuota(); err != nil {
    log.Printf("Quota exceeded: %v", err)
}
```

## Response Structures

### Chat Completion Response

```go
type ChatCompletionResponse struct {
    Id      string                 `json:"id"`
    Object  string                 `json:"object"`
    Model   string                 `json:"model"`
    Created int64                  `json:"created"`
    Choices []ChatCompletionChoice `json:"choices"`
    Usage   *TokenUsage            `json:"usage"`
    // ... additional metadata
}
```

### Usage Tracking

```go
// Access token usage information
if response.Usage != nil {
    fmt.Printf("Prompt tokens: %d\n", response.Usage.PromptTokens)
    fmt.Printf("Completion tokens: %d\n", response.Usage.CompletionTokens)
    fmt.Printf("Total tokens: %d\n", response.Usage.TotalTokens)
}
```

## Testing

The package includes comprehensive tests:

```bash
go test -v ./components/core/manageai/...
```

## Dependencies

- `github.com/eino-contrib/jsonschema` - JSON schema validation
- `github.com/google/uuid` - UUID generation for message IDs

## Environment Variables

No environment variables required. All configuration is passed through the client constructor and methods.

## Best Practices

1. **Always close clients**: Use `defer client.Close()` to clean up connections
2. **Handle errors appropriately**: Check for specific error types when needed
3. **Set appropriate timeouts**: Configure timeouts based on your use case
4. **Monitor rate limits**: Check response headers for rate limit information
5. **Use model aliases**: Take advantage of friendly model names
6. **Validate responses**: Always check that responses contain expected data

## Contributing

This is an core package. For issues and feature requests, please contact the maintainers.

## Changelog

For detailed information about changes and releases, see [CHANGELOG.md](./CHANGELOG.md).

## Release Notes

For comprehensive release information and migration guides, see [RELEASE_NOTES.md](./RELEASE_NOTES.md).

## License

This package is part of the bot_manageai/eino-ext project.
