# ManageAI Go Client - Release Notes

## Version 1.0.0 (2026-03-20)

We're excited to announce the first stable release of the ManageAI Go Client! This comprehensive library provides everything you need to integrate with the ManageAI AI Router service and build powerful AI applications.

### 🎉 Major Features

#### Core Client Functionality
- **Complete API Integration**: Full support for all ManageAI service endpoints
- **Configurable HTTP Client**: Customizable timeouts, retry attempts, and delays
- **Smart Model Management**: Automatic model alias resolution and redirection

#### Chat Completions API
- **Full Chat Completion Support**: Send and receive AI-generated responses
- **Real-time Streaming**: Server-Sent Events (SSE) streaming for immediate responses
- **Rich Message Types**: System, human, assistant, and tool messages
- **Multi-modal Content**: Handle text and image content seamlessly

#### Tokenization & Detokenization
- **Text Tokenization**: Convert text to token arrays for AI processing
- **Text Detokenization**: Convert token arrays back to readable text
- **Model-specific Processing**: Optimized tokenization for different AI models

#### Embedding Services
- **High-performance Embeddings**: Generate vector representations of text
- **Batch Processing**: Efficiently process multiple texts at once
- **Flexible Encoding**: Support for different encoding formats

#### Reranking Services
- **Intelligent Document Ranking**: Sort documents by relevance to queries
- **Configurable Scoring**: Customize reranking parameters
- **Performance Optimization**: Get the most relevant results first

#### Advanced Tool Integration
- **Struct-to-Tool Conversion**: Automatically generate tool definitions from Go structs
- **JSON Schema Enhancement**: Add constraints and validations to tool parameters
- **Tool Call Handling**: Parse and validate tool call arguments

### 🚀 Getting Started

```go
package main

import (
    "fmt"
    "log"
    "github.com/manageai/ai-assistant/eino-manageai-ext/components/core/manageai"
)

func main() {
    // Initialize client
    modelId := "qwen3-vl-235b-a22b-instruct"
    apiKey := "your-api-key-here"
    client, err := manageai.NewClient(&modelId, apiKey)
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

### 🛠 Key Improvements

#### Developer Experience
- **Comprehensive Documentation**: Detailed API reference and usage examples
- **Builder Pattern Options**: Fluent interface for configuring chat completions
- **Environment-based Configuration**: Easy setup with environment variables
- **Comprehensive Error Handling**: Clear error types and validation messages

#### Performance & Reliability
- **Built-in Retry Logic**: Automatic retry with configurable delays
- **Rate Limit Monitoring**: Track and respect API quotas
- **Memory Management**: Proper cleanup and connection handling
- **Production Ready**: Extensively tested with real API endpoints

#### Testing & Quality
- **Full Test Coverage**: Comprehensive tests for all features
- **Integration Testing**: Verified against real API endpoints
- **Streaming Verification**: Thorough testing of real-time functionality

### 📚 Documentation

- **Quick Start Guides**: Get up and running quickly
- **API Reference**: Complete documentation for all functions
- **Advanced Usage**: Examples for complex scenarios
- **Best Practices**: Guidelines for production use

### 🎯 Use Cases

- **AI Chat Applications**: Build conversational AI experiences
- **Content Generation**: Automate text creation workflows
- **Data Processing**: Analyze and transform large text datasets
- **Tool Integration**: Connect AI models with external functions
- **Enterprise Solutions**: Scale AI capabilities across organizations

### 🤝 Migration Guide

This is the first stable release. For new projects, simply add the dependency:

```bash
go get github.com/manageai/ai-assistant/eino-manageai-ext/components/core/manageai@v1.0.0
```

### 📞 Support

For issues, feature requests, or questions:
- Check the [documentation](./README.md)
- Review the [changelog](./CHANGELOG.md)
- Contact the development team

---

**Thank you for choosing the ManageAI Go Client!** We're committed to helping you build amazing AI-powered applications.