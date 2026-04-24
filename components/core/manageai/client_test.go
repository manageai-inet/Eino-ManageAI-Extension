package manageai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestChatCompletion(t *testing.T) {
	apiKey := os.Getenv("MAI_API_KEY")
	modelId := "qwen3-vl-235b-a22b-instruct"
	// modelId := "Qwen3.5"

	// Create client
	client, err := NewClient(&modelId, apiKey)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test basic chat completion
	t.Run("BasicChatCompletion", func(t *testing.T) {
		messages := []any{
			"Hello, how are you?",
		}

		options := &ChatCompletionOptions{}
		options.WithModel(modelId)

		response, err := client.ChatCompletion(messages, options)
		if err != nil {
			t.Errorf("ChatCompletion failed: %v", err)
			return
		}

		if response == nil {
			t.Error("Expected non-nil response")
			return
		}

		if len(response.Choices) == 0 {
			t.Error("Expected at least one choice in response")
			return
		}

		choice := response.Choices[0]
		if choice.Message.Content == nil || *choice.Message.Content == "" {
			t.Error("Expected non-empty content in response")
			return
		}

		t.Logf("Successfully received chat completion response: %s", *choice.Message.Content)
	})

	// Test chat completion with system message
	t.Run("ChatCompletionWithSystemMessage", func(t *testing.T) {
		messages := []any{
			NewSystemMessage("You are a helpful assistant.", nil),
			"Tell me a short joke.",
		}

		options := &ChatCompletionOptions{}
		options.WithModel(modelId)

		response, err := client.ChatCompletion(messages, options)
		if err != nil {
			t.Errorf("ChatCompletion failed: %v", err)
			return
		}

		if response == nil {
			t.Error("Expected non-nil response")
			return
		}

		if len(response.Choices) == 0 {
			t.Error("Expected at least one choice in response")
			return
		}

		choice := response.Choices[0]
		if choice.Message.Content == nil || *choice.Message.Content == "" {
			t.Error("Expected non-empty content in response")
			return
		}

		t.Logf("Successfully received chat completion response with system message: %s", *choice.Message.Content)
	})

	// Test chat completion with options
	t.Run("ChatCompletionWithOptions", func(t *testing.T) {
		messages := []any{
			"Count from 1 to 5.",
		}

		options := &ChatCompletionOptions{}
		maxTokens := int64(100)
		temperature := 0.7
		options.WithMaxTokens(maxTokens).WithTemperature(temperature).WithModel(modelId)

		response, err := client.ChatCompletion(messages, options)
		if err != nil {
			t.Errorf("ChatCompletion failed: %v", err)
			return
		}

		if response == nil {
			t.Error("Expected non-nil response")
			return
		}

		if len(response.Choices) == 0 {
			t.Error("Expected at least one choice in response")
			return
		}

		choice := response.Choices[0]
		if choice.Message.Content == nil || *choice.Message.Content == "" {
			t.Error("Expected non-empty content in response")
			return
		}

		t.Logf("Successfully received chat completion response with options: %s", *choice.Message.Content)
	})

	// Test chat completion with structured messages
	t.Run("ChatCompletionWithStructuredMessages", func(t *testing.T) {
		messages := []any{
			map[string]any{
				"role":    "user",
				"content": "What is the capital of France?",
			},
		}

		options := &ChatCompletionOptions{}
		options.WithModel(modelId)

		response, err := client.ChatCompletion(messages, options)
		if err != nil {
			t.Errorf("ChatCompletion failed: %v", err)
			return
		}

		if response == nil {
			t.Error("Expected non-nil response")
			return
		}

		if len(response.Choices) == 0 {
			t.Error("Expected at least one choice in response")
			return
		}

		choice := response.Choices[0]
		if choice.Message.Content == nil || *choice.Message.Content == "" {
			t.Error("Expected non-empty content in response")
			return
		}

		t.Logf("Successfully received chat completion response with structured messages: %s", *choice.Message.Content)
	})

	// Test chat completion with tools
	t.Run("ChatCompletionWithTools", func(t *testing.T) {
		messages := []any{
			"What is the weather like in Boston?",
		}

		options := &ChatCompletionOptions{}
		options.WithModel(modelId)

		tools := []Tools{
			{
				Type: "function",
				Function: ToolDefinition{
					Name:        "get_current_weather",
					Description: "Get the current weather in a given location",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"location": map[string]any{
								"type":        "string",
								"description": "The city and state, e.g. San Francisco, CA",
							},
							"unit": map[string]any{
								"type": "string",
								"enum": []string{"celsius", "fahrenheit"},
							},
						},
						"required": []string{"location"},
					},
				},
			},
		}
		options.WithTools(tools, nil)

		response, err := client.ChatCompletion(messages, options)
		if err != nil {
			t.Errorf("ChatCompletion failed: %v", err)
			return
		}

		if response == nil {
			t.Error("Expected non-nil response")
			return
		}

		if len(response.Choices) == 0 {
			t.Error("Expected at least one choice in response")
			return
		}

		choice := response.Choices[0]

		if choice.Message.ToolsCalls == nil || len(*choice.Message.ToolsCalls) == 0 {
			t.Log("Expected tool calls in response, but the model might have returned text instead based on the prompt. Logging its response if any.")
			if choice.Message.Content != nil {
				t.Logf("Model returned content instead: %s", *choice.Message.Content)
			}
			return
		}

		for _, tc := range *choice.Message.ToolsCalls {
			name := ""
			if tc.Function.Name != nil {
				name = *tc.Function.Name
			}
			args := ""
			if tc.Function.Arguments != nil {
				args = *tc.Function.Arguments
			}
			t.Logf("Received tool call: %s with arguments: %s", name, args)
		}
	})

	t.Run("ChatCompletionWithToolsStruct", func(t *testing.T) {
		messages := []any{
			"What is the weather like in Boston? Use celsius",
		}

		options := &ChatCompletionOptions{}
		options.WithModel(modelId)

		type WeatherArgs struct {
			Location string `json:"location" jsonschema:"The city and state, e.g. San Francisco, CA"`
			Unit     string `json:"unit" jsonschema:"The unit of temperature"`
		}
		WeatherToolName := "get_current_weather"
		WeatherToolDescription := "Get the current weather in a given location"
		tool, err := StructToTool[WeatherArgs](WeatherToolName, WeatherToolDescription, true)
		if err != nil {
			t.Errorf("StructToTool failed: %v", err)
			return
		}
		tool.Function.WithEnum("unit", []any{"C", "F", "K"})
		options.WithTools([]Tools{tool}, nil)
		response, err := client.ChatCompletion(messages, options)
		if err != nil {
			t.Errorf("ChatCompletion failed: %v", err)
			return
		}

		if response == nil {
			t.Error("Expected non-nil response")
			return
		}

		if len(response.Choices) == 0 {
			t.Error("Expected at least one choice in response")
			return
		}

		choice := response.Choices[0]

		if choice.Message.ToolsCalls == nil || len(*choice.Message.ToolsCalls) == 0 {
			t.Log("Expected tool calls in response, but the model might have returned text instead based on the prompt. Logging its response if any.")
			if choice.Message.Content != nil {
				t.Logf("Model returned content instead: %s", *choice.Message.Content)
			}
			return
		}

		for _, tc := range *choice.Message.ToolsCalls {
			name := ""
			if tc.Function.Name != nil {
				name = *tc.Function.Name
			}
			args := ""
			if tc.Function.Arguments != nil {
				args = *tc.Function.Arguments
			}
			argsStruct, err := HandleToolCall[WeatherArgs](args)
			if err != nil {
				t.Errorf("HandleToolCall failed: %v", err)
				return
			}
			t.Logf("Received tool call: %s with arguments: %+v", name, *argsStruct)
		}
	})
}

func TestClientConfiguration(t *testing.T) {
	apiKey := os.Getenv("MAI_API_KEY")
	modelId := "Qwen3.5"

	t.Run("CreateClient", func(t *testing.T) {
		client, err := NewClient(&modelId, apiKey)
		if err != nil {
			t.Errorf("Failed to create client: %v", err)
			return
		}

		if client == nil {
			t.Error("Expected non-nil client")
			return
		}

		if client.APIKey != apiKey {
			t.Error("API key mismatch")
			return
		}

		if client.Model == nil {
			t.Error("Model should not be nil")
			return
		}

		t.Log("Successfully created client with correct configuration")
	})

	t.Run("ClientWithCustomSettings", func(t *testing.T) {
		client, err := NewClient(&modelId, apiKey)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		// Test with custom timeout
		client, err = client.WithTimeout(60)
		if err != nil {
			t.Errorf("Failed to set timeout: %v", err)
			return
		}

		// Test with custom retry settings
		client, err = client.WithRetryAttempt(5)
		if err != nil {
			t.Errorf("Failed to set retry attempt: %v", err)
			return
		}

		client, err = client.WithRetryDelay(2)
		if err != nil {
			t.Errorf("Failed to set retry delay: %v", err)
			return
		}

		t.Log("Successfully configured client with custom settings")
	})
}

func TestTokenize(t *testing.T) {
	apiKey := os.Getenv("MAI_API_KEY")
	// modelId := "Qwen/Qwen3-235B-A22B" //used
	modelId := "qwen3-235b-a22b"

	client, err := NewClient(&modelId, apiKey)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	t.Run("TokenizeWithMessagesStructure", func(t *testing.T) {
		testText := "สวัสดี AI"

		// เรียกฟังก์ชัน Tokenize
		resp, err := client.Tokenize(testText, nil)

		if err != nil {
			t.Fatalf("Tokenize failed: %v", err)
		}

		if resp == nil {
			t.Fatal("Expected non-nil response")
		}

		if len(resp.Data) == 0 {
			t.Error("Expected token IDs, but got an empty array")
		}

		t.Logf("--- Tokenize Result ---")
		t.Logf("Input Text: %s", testText)
		t.Logf("Model: %s", resp.Model)
		t.Logf("Token Count: %d", len(resp.Data))
		t.Logf("Token IDs: %v", resp.Data)
		t.Logf("Max Model Length: %d", resp.MaxModelLength)
		t.Logf("-----------------------")
	})

	t.Run("TokenizeWithMessagesStructure", func(t *testing.T) {
		testText := "สวัสดีชาวโลก 🌍✨ @#$%^&*()"

		// เรียกฟังก์ชัน Tokenize
		resp, err := client.Tokenize(testText, nil)

		if err != nil {
			t.Fatalf("Tokenize failed: %v", err)
		}

		if resp == nil {
			t.Fatal("Expected non-nil response")
		}

		if len(resp.Data) == 0 {
			t.Error("Expected token IDs, but got an empty array")
		}

		t.Logf("--- Tokenize Result ---")
		t.Logf("Input Text: %s", testText)
		t.Logf("Model: %s", resp.Model)
		t.Logf("Token Count: %d", len(resp.Data))
		t.Logf("Token IDs: %v", resp.Data)
		t.Logf("Max Model Length: %d", resp.MaxModelLength)
		t.Logf("-----------------------")
	})

	t.Run("TokenizeWithMessagesStructure", func(t *testing.T) {
		testText := ""

		// เรียกฟังก์ชัน Tokenize
		resp, err := client.Tokenize(testText, nil)

		if err != nil {
			t.Fatalf("Tokenize failed: %v", err)
		}

		if resp == nil {
			t.Fatal("Expected non-nil response")
		}

		if len(resp.Data) == 0 {
			t.Error("Expected token IDs, but got an empty array")
		}

		t.Logf("--- Tokenize Result ---")
		t.Logf("Input Text: %s", testText)
		t.Logf("Model: %s", resp.Model)
		t.Logf("Token Count: %d", len(resp.Data))
		t.Logf("Token IDs: %v", resp.Data)
		t.Logf("Max Model Length: %d", resp.MaxModelLength)
		t.Logf("-----------------------")
	})
}

func TestChatCompletionStream(t *testing.T) {
	apiKey := os.Getenv("MAI_API_KEY")
	modelId := "qwen3-vl-235b-a22b-instruct"

	// Create client
	client, err := NewClient(&modelId, apiKey)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	client.WithTimeout(120 * time.Second) // Set client timeout to 2 minutes

	// Test basic chat completion stream
	t.Run("BasicChatCompletionStream", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		messages := []any{
			"Hello, how are you?",
		}

		options := &ChatCompletionOptions{}
		options.WithModel(modelId)

		fmt.Println("\n--- Basic Chat Stream Output ---")
		stream, err := client.ChatCompletionStream(ctx, messages, options)
		if err != nil {
			t.Fatalf("ChatCompletionStream failed: %v", err)
		}
		defer stream.Close()

		var fullContent string
		var chunkCount int

		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Stream interrupted: %v", err)
			}

			chunkCount++
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != nil {
				content := *chunk.Choices[0].Delta.Content
				fmt.Print(content + "|")
				fullContent += content
			}
		}
		fmt.Println("\n--------------------------------")

		if chunkCount == 0 {
			t.Error("Expected at least one chunk in response")
		}
		if fullContent == "" {
			t.Error("Expected non-empty content from stream")
		}

		t.Logf("Successfully received stream with %d chunks", chunkCount)
	})

	// Test chat completion stream with system message
	t.Run("ChatCompletionStreamWithSystemMessage", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		messages := []any{
			NewSystemMessage("You are a helpful assistant.", nil),
			"Tell me a short joke.",
		}

		options := &ChatCompletionOptions{}
		options.WithModel(modelId)

		fmt.Println("\n--- System Message Stream Output ---")
		stream, err := client.ChatCompletionStream(ctx, messages, options)
		if err != nil {
			t.Fatalf("ChatCompletionStream failed: %v", err)
		}
		defer stream.Close()

		var fullContent string
		var chunkCount int

		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Stream interrupted: %v", err)
			}

			chunkCount++
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != nil {
				content := *chunk.Choices[0].Delta.Content
				fmt.Print(content + "|")
				fullContent += content
			}
		}
		fmt.Println("\n------------------------------------")

		if chunkCount == 0 {
			t.Error("Expected at least one chunk in response")
		}
		if fullContent == "" {
			t.Error("Expected non-empty content from stream")
		}

		t.Logf("Successfully received stream with system message, %d chunks", chunkCount)
	})

	// Test chat completion stream with options
	t.Run("ChatCompletionStreamWithOptions", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		messages := []any{
			"Count from 1 to 5.",
		}

		options := &ChatCompletionOptions{}
		maxTokens := int64(100)
		temperature := 0.7
		options.WithMaxTokens(maxTokens).WithTemperature(temperature).WithModel(modelId)

		fmt.Println("\n--- Options Stream Output ---")
		stream, err := client.ChatCompletionStream(ctx, messages, options)
		if err != nil {
			t.Fatalf("ChatCompletionStream failed: %v", err)
		}
		defer stream.Close()

		var fullContent string
		var chunkCount int

		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Stream interrupted: %v", err)
			}

			chunkCount++
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != nil {
				content := *chunk.Choices[0].Delta.Content
				fmt.Print(content + "|")
				fullContent += content
			}
		}
		fmt.Println("\n-----------------------------")

		if chunkCount == 0 {
			t.Error("Expected at least one chunk in response")
		}
		if fullContent == "" {
			t.Error("Expected non-empty content from stream")
		}

		t.Logf("Successfully received stream with options, %d chunks", chunkCount)
	})

	// Test chat completion stream with structured messages
	t.Run("ChatCompletionStreamWithStructuredMessages", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		messages := []any{
			map[string]any{
				"role":    "user",
				"content": "What is the capital of France?",
			},
		}

		options := &ChatCompletionOptions{}
		options.WithModel(modelId)

		fmt.Println("\n--- Structured Messages Stream Output ---")
		stream, err := client.ChatCompletionStream(ctx, messages, options)
		if err != nil {
			t.Fatalf("ChatCompletionStream failed: %v", err)
		}
		defer stream.Close()

		var fullContent string
		var chunkCount int

		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Stream interrupted: %v", err)
			}

			chunkCount++
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != nil {
				content := *chunk.Choices[0].Delta.Content
				fmt.Print(content + "|")
				fullContent += content
			}
		}
		fmt.Println("\n-----------------------------------------")

		if chunkCount == 0 {
			t.Error("Expected at least one chunk in response")
		}
		if fullContent == "" {
			t.Error("Expected non-empty content from stream")
		}

		t.Logf("Successfully received stream with structured messages, %d chunks", chunkCount)
	})
}

// ====================== Embedding ==========================
func TestEmbedding(t *testing.T) {
	// Test configuration
	apiKey := os.Getenv("MAI_API_KEY")
	modelId := "Qwen3-Embedding-8B"
	// Create client
	client, err := NewClient(&modelId, apiKey)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test basic embedding
	t.Run("BasicEmbedding", func(t *testing.T) {
		input := []string{
			"This is a test sentence 1.",
			"This is a test sentence 2.",
		}

		options := &EmbeddingOptions{}
		options.WithModel(modelId)

		response, err := client.Embedding(input, options)
		if err != nil {
			t.Fatalf("Embedding failed: %v", err)
			return
		}

		if response == nil {
			t.Error("Expected non-nil response")
			return
		}

		if len(response.Data) != len(input) {
			t.Errorf("Expected %d embeddings, got %d", len(input), len(response.Data))
			return
		}

		for i, data := range response.Data {
			if len(data.Embedding) == 0 {
				t.Errorf("Expected non-empty embedding for input %d", i)
				return
			}
		}

		t.Logf("Successfully received embedding response with %d embeddings", len(response.Data))
	})

	// Test embedding with options
	t.Run("EmbeddingWithOptions", func(t *testing.T) {
		input := []string{
			"This is another test.",
		}

		options := &EmbeddingOptions{}
		encodingFormat := "float"
		options.WithModel(modelId).WithEncodingFormat(encodingFormat)

		response, err := client.Embedding(input, options)
		if err != nil {
			t.Errorf("Embedding failed: %v", err)
			return
		}

		if response == nil {
			t.Error("Expected non-nil response")
			return
		}

		if len(response.Data) != 1 {
			t.Errorf("Expected 1 embedding, got %d", len(response.Data))
			return
		}

		t.Logf("Successfully received embedding response with custom options")
	})
}

// ====================== Reranker ==========================
func TestRerank(t *testing.T) {
	// Test configuration
	apiKey := os.Getenv("MAI_API_KEY")
	modelId := "Qwen3-Reranker-8B"
	// Create client
	client, err := NewClient(&modelId, apiKey)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	// Test basic rerank
	t.Run("BasicRerank", func(t *testing.T) {
		query := "artificial intelligence"
		documents := []string{
			"AI is transforming healthcare.",
			"The weather today is sunny.",
			"Machine learning is a subset of AI.",
			"Cooking recipes for beginners.",
		}

		options := &RerankOptions{}
		options.WithModel(modelId)

		response, err := client.Rerank(query, documents, options)
		if err != nil {
			t.Errorf("Rerank failed: %v", err)
			return
		}

		if response == nil {
			t.Fatalf("Expected non-nil response")
			return
		}

		if len(response.Data) != len(documents) {
			t.Errorf("Expected %d rerank results, got %d", len(documents), len(response.Data))
			return
		}

		// Check that results are sorted by relevance score (descending)
		for i := 1; i < len(response.Data); i++ {
			if response.Data[i].RelevanceScore > response.Data[i-1].RelevanceScore {
				t.Error("Results should be sorted by relevance score in descending order")
				return
			}
		}

		t.Logf("Successfully received rerank response with %d results", len(response.Data))
	})

	// Test rerank with options
	t.Run("RerankWithOptions", func(t *testing.T) {
		query := "Explain supervised vs unsupervised learning"
		documents := []string{
			"Supervised learning uses labelled data...",
			"Unsupervised learning discovers hidden patterns...",
			"การเรียนรู้แบบมีผู้สอนต้องใช้ป้ายกำกับ",
			"แบบไม่มีผู้สอนไม่ต้องมี label",
			"กินข้าวหรือยัง",
		}

		options := &RerankOptions{}
		topN := 5
		returnDocuments := true
		maxChunksPerDoc := 32
		options.WithModel(modelId).WithTopN(topN).WithReturnDocuments(returnDocuments).WithMaxChunksPerDoc(maxChunksPerDoc)

		response, err := client.Rerank(query, documents, options)
		if err != nil {
			t.Errorf("Rerank failed: %v", err)
			return
		}

		if response == nil {
			t.Error("Expected non-nil response")
			return
		}

		if len(response.Data) != topN {
			t.Errorf("Expected %d rerank results, got %d", topN, len(response.Data))
			return
		}

		t.Logf("Successfully received rerank response with top %d results", topN)
	})
}

// ====================== Detokenize ==========================
func TestDetokenize(t *testing.T) {
	// Test configuration
	apiKey := os.Getenv("MAI_API_KEY")
	modelId := "qwen3-235b-a22b"
	// Create client
	client, err := NewClient(&modelId, apiKey)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test basic detokenize
	t.Run("BasicDetokenize", func(t *testing.T) {
		tokens := []int{
			248045, 846, 198, 9419, 248046, 198, 248045, 74455, 198, 248068, 198,
		}

		options := &DetokenizeOptions{}
		options.WithModel(modelId)

		response, err := client.Detokenize(tokens, options)
		if err != nil {
			t.Fatalf("Detokenize failed: %v", err)
			return
		}

		if response.Data == "" {
			t.Error("Expected non-empty detokenized data")
			return
		}
		t.Logf("Successfully received detokenize response: %s", response.Data)
	})

	// Test Detokenize with empty tokens
	t.Run("DetokenizeWithEmptyTokens", func(t *testing.T) {
		tokens := []int{}
		options := &DetokenizeOptions{}
		options.WithModel(modelId)

		_, err := client.Detokenize(tokens, options)
		if err == nil {
			t.Error("Expected error when sending empty tokens, but got nil")
		} else {
			t.Logf("Received error as expected: %v", err)
		}
	})
}

// ============================================================================
// Mock Transport: ดักจับและจำลอง Response โดยไม่ต้องใช้อินเทอร์เน็ต
// ============================================================================
type mockTransport struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

// ============================================================================
// Unit Tests
// ============================================================================

func TestClient_Config(t *testing.T) {
	model := "qwen3-235b-a22b"
	// apiKey := "test-key"
	apiKey := os.Getenv("MAI_API_KEY")
	client, err := NewClient(&model, apiKey)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.Timeout != defaultTimeout {
		t.Errorf("expected default timeout %v, got %v", defaultTimeout, client.Timeout)
	}

	// ทดสอบการเปลี่ยนค่า Retry
	client.WithRetryAttempt(5)
	if client.RetryAttempt != 5 {
		t.Errorf("expected retry attempt 5, got %d", client.RetryAttempt)
	}
}

func TestClient_NewRequest(t *testing.T) {
	apiKey := os.Getenv("MAI_API_KEY")

	if apiKey == "" {
		t.Skip("Skipping test: MAI_API_KEY environment variable is not set. Please export it before running.")
	}

	client := &Client{
		APIKey: apiKey,
	}
	ctx := context.Background()

	payload := []byte(`{"hello":"world"}`)
	headers := map[string]string{"X-Custom-Header": "custom-value"}

	req, err := client.NewRequest(ctx, "POST", "/api/v1/test", payload, headers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gotAuth := req.Header.Get("Authorization")
	expectedAuth := "Bearer " + apiKey
	if gotAuth != expectedAuth {
		t.Errorf("missing or wrong Authorization header, expected %q, got: %q", expectedAuth, gotAuth)
	}

	gotContentType := req.Header.Get("Content-Type")
	if gotContentType != "application/json" {
		t.Errorf("missing or wrong Content-Type header, got: %q", gotContentType)
	}

	gotCustom := req.Header.Get("X-Custom-Header")
	if gotCustom != "custom-value" {
		t.Errorf("missing custom header, got: %q", gotCustom)
	}
}

func TestClient_RetryLogic(t *testing.T) {
	model := "qwen3-235b-a22b"
	apiKey := os.Getenv("MAI_API_KEY")
	client, _ := NewClient(&model, apiKey)

	client.WithRetryAttempt(3)
	client.WithRetryDelay(10 * time.Millisecond)

	// case 1: ยิงครั้งแรกสำเร็จเลย (ไม่ควรเกิดการ Retry)
	t.Run("Success on first try", func(t *testing.T) {
		attempts := 0
		client.HTTPClient.Transport = &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				attempts++
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok"}`)),
				}, nil
			},
		}

		res, err := client.doRetryableRequest(context.Background(), "GET", "/test", nil, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}
		if res.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", res.StatusCode)
		}
	})

	// case 2: จำลองเซิร์ฟเวอร์พัง 2 ครั้งแรก และกลับมาดีในครั้งที่ 3 (ต้องยิงครบ 3 ครั้งแล้วสำเร็จ)
	t.Run("Retry on Error then success", func(t *testing.T) {
		attempts := 0
		client.HTTPClient.Transport = &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				attempts++
				if attempts < 3 {
					// พัง 2 รอบแรก (สถานะ 500)
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       io.NopCloser(bytes.NewBufferString(`{"error":"server error"}`)),
					}, nil
				}
				// รอบที่ 3 สำเร็จ
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok"}`)),
				}, nil
			},
		}

		res, err := client.doRetryableRequest(context.Background(), "GET", "/test", nil, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
		if res.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", res.StatusCode)
		}
	})

	// case 3: เน็ตหลุด (ต้องพยายามจนครบโควต้า 3 ครั้ง แล้วคืนค่า Error ออกมา)
	t.Run("Max Retries Reached on Network Error", func(t *testing.T) {
		attempts := 0
		client.HTTPClient.Transport = &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				attempts++
				return nil, errors.New("connection reset by peer")
			},
		}

		_, err := client.doRetryableRequest(context.Background(), "GET", "/test", nil, nil)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if attempts != 3 {
			t.Errorf("expected max attempts 3, got %d", attempts)
		}
	})

	// case 4: สั่ง Context Timeout คั่นกลาง (ต้องหยุด Retry ทันทีโดยไม่สนว่าจะครบ 3 โควต้าหรือไม่)
	t.Run("Context Timeout stops retry immediately", func(t *testing.T) {
		attempts := 0
		client.HTTPClient.Transport = &mockTransport{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				attempts++
				// ดึงเวลาถ่วงไว้รอบละ 20ms
				time.Sleep(20 * time.Millisecond)
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString(`{"error":"server error"}`)),
				}, nil
			},
		}

		// สั่งให้ Timeout ใน 15ms (ซึ่งจะระเบิดตั้งแต่รอบแรกยังไม่ทันจบ)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
		defer cancel()

		_, err := client.doRetryableRequest(ctx, "GET", "/test", nil, nil)

		// คาดหวังว่าต้องหลุดออกมาเพราะ Context หมดเวลา
		if err == nil || (!errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled)) {
			t.Errorf("expected context deadline/canceled error, got %v", err)
		}

		// โควต้ามี 3 รอบ แต่มันควรโดนตัดจบตั้งแต่รอบแรก
		if attempts >= 3 {
			t.Errorf("expected less than 3 attempts due to early context cancel, got %d", attempts)
		}
	})
}
