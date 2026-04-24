# Eino ManageAI Extension

A Go library that provides ManageAI integration for the [CloudWeGo Eino](https://github.com/cloudwego/eino) framework. This extension enables seamless connection to ManageAI AI Router services with full support for chat completions, streaming, tool calling, and multi-modal interactions.

## Project Structure

```
eino-ext/
├── components/
│   ├── core/
│   │   └── manageai/      # ManageAI client library
│   └── model/
│       └── manageai/      # Eino ChatModel implementation
```

## Components

### 1. Core Client (`components/core/manageai`)

The core component provides a low-level Go client for interacting with the ManageAI AI Router service. It offers comprehensive API coverage including chat completions, embeddings, reranking, tokenization, and model management.

**Features:**
- Chat completions with multiple model support
- Streaming responses
- Tool/function calling integration
- Multi-modal support (text + images)
- Embedding generation
- Reranking capabilities
- Tokenization/detokenization
- Rate limiting and quota monitoring
- Model alias resolution and automatic redirection
- Comprehensive error handling

**Installation:**
```bash
go get github.com/manageai-inet/Eino-ManageAI-Extension/components/core/manageai
```

**Documentation:** [Core Client README](./components/core/manageai/README.md)

**Quick Start:**
```go
package main

import (
    "fmt"
    "log"
    "github.com/manageai-inet/Eino-ManageAI-Extension/components/core/manageai"
)

func main() {
    client, err := manageai.GetDefaultClient()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    messages := []any{"Hello, how are you today?"}
    options := &manageai.ChatCompletionOptions{}
    
    response, err := client.ChatCompletion(messages, options)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(*response.Choices[0].Message.Content)
}
```

---

### 2. Model Implementation (`components/model/manageai`)

The model component implements the Eino `ChatModel` interface, providing a bridge between Eino's framework abstractions and the ManageAI client. This allows you to use ManageAI models within Eino's component architecture.

**Features:**
- Implements Eino's `model.ChatModel` interface
- Support for `Generate()` and `Stream()` methods
- Tool calling integration via `WithTools()`
- Configuration via `ChatModelConfig`
- Custom options for response format, guided JSON, and regex
- Automatic message format conversion between Eino and ManageAI

**Installation:**
```bash
go get github.com/manageai-inet/Eino-ManageAI-Extension/components/model/manageai
```

**Quick Start:**
```go
package main

import (
    "context"
    "fmt"
    "log"

    manageai "github.com/manageai-inet/Eino-ManageAI-Extension/components/model/manageai"
    "github.com/cloudwego/eino/schema"
)

func main() {
    ctx := context.Background()

    config := &manageai.ChatModelConfig{
        APIKey: "your-api-key",
        Model:  "Qwen3.5",
    }

    chatModel, err := manageai.NewChatModel(ctx, config)
    if err != nil {
        log.Fatal(err)
    }

    messages := []*schema.Message{
        schema.UserMessage("Hello, how are you?"),
    }

    response, err := chatModel.Generate(ctx, messages)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.Content)
}
```

**Configuration Options:**
```go
config := &manageai.ChatModelConfig{
    APIKey:           "your-api-key",
    Model:            "Qwen3.5",
    BaseURL:          "", // Optional custom base URL
    Timeout:          30 * time.Second,
    MaxTokens:        intPtr(2000),
    Temperature:      float32Ptr(0.7),
    TopP:             float32Ptr(0.9),
    ResponseFormat:   &openai.ChatCompletionResponseFormat{Type: "json_object"},
}
```

**Custom Options:**
```go
// Set response format
manageai.WithResponseFormat(manageai.ResponseFormatJson)

// Guided JSON schema generation
manageai.WithGuidedJson(jsonSchema)

// Guided regex pattern
manageai.WithGuidedRegex("^[A-Z].*")
```

---

## Dependencies

- [CloudWeGo Eino](https://github.com/cloudwego/eino) - AI framework
- [eino-contrib/jsonschema](https://github.com/eino-contrib/jsonschema) - JSON schema validation
- [google/uuid](https://github.com/google/uuid) - UUID generation

## Testing

Run tests for each component:

```bash
# Core client tests
go test -v ./components/core/manageai/...

# Model implementation tests
go test -v ./components/model/manageai/...
```

## License

This project is part of the bot_manageai/eino-ext repository.

## Related Links

- [Core Client Documentation](./components/core/manageai/README.md)
- [CloudWeGo Eino Documentation](https://github.com/cloudwego/eino)
Hello World 2 
