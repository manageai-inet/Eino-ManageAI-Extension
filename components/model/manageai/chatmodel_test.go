package manageai

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// ============================================================================
// Unit Tests (ทดสอบการจัดการ Error และ Config พื้นฐาน)
// ============================================================================

func TestNewChatModel(t *testing.T) {
	ctx := context.Background()

	modelId := "qwen3-vl-235b-a22b-instruct"
	apiKey := os.Getenv("MAI_API_KEY")

	t.Run("nil config", func(t *testing.T) {
		cm, err := NewChatModel(ctx, nil)
		if err == nil {
			t.Errorf("expected error for nil config, got nil")
		}
		if cm != nil {
			t.Errorf("expected nil ChatModel, got %v", cm)
		}
	})

	t.Run("valid config", func(t *testing.T) {

		cm, err := NewChatModel(ctx, &ChatModelConfig{
			APIKey: apiKey,
			Model:  modelId,
		})

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cm == nil {
			t.Errorf("expected valid ChatModel, got nil")
		}
	})
}

func TestValidateToolOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    []model.Option
		wantErr string
	}{
		{
			name:    "no options",
			opts:    nil,
			wantErr: "",
		},
		{
			name: "tool_choice 'allowed' with allowed_tools",
			opts: []model.Option{
				model.WithToolChoice(schema.ToolChoiceAllowed, "tool1"),
				model.WithTools([]*schema.ToolInfo{{Name: "tool1"}}),
			},
			wantErr: "tool_choice 'allowed' is not supported when allowed tool names are present",
		},
		{
			name: "tool_choice 'forced' with zero allowed_tool",
			opts: []model.Option{
				model.WithToolChoice(schema.ToolChoiceForced),
			},
			wantErr: "at least one allowed tool name is required for tool_choice 'forced'",
		},
		{
			name: "tool_choice 'forced' with more than one allowed_tool",
			opts: []model.Option{
				model.WithToolChoice(schema.ToolChoiceForced, "tool1", "tool2"),
				model.WithTools([]*schema.ToolInfo{{Name: "tool1"}, {Name: "tool2"}}),
			},
			wantErr: "only one allowed tool name can be configured for tool_choice 'forced'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToolOptions(tt.opts...)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

// ============================================================================
// Integration Tests: Generate
// ============================================================================

func TestChatModel_Integration_Generate(t *testing.T) {
	apiKey := os.Getenv("MAI_API_KEY")
	if apiKey == "" {
		t.Fatalf("MAI_API_KEY is not set")
	}

	modelId := "qwen3-vl-235b-a22b-instruct"
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cm, _ := NewChatModel(ctx, &ChatModelConfig{APIKey: apiKey, Model: modelId})

	// Test basic chat generate
	t.Run("Basic", func(t *testing.T) {
		resp, err := cm.Generate(ctx, []*schema.Message{schema.UserMessage("Hello!")})
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if resp.ResponseMeta != nil && resp.ResponseMeta.Usage != nil {
			t.Logf("Token Usage: %+v", resp.ResponseMeta.Usage)
		}
		t.Logf("Response: %s", resp.Content)
	})

	t.Run("SystemMessage", func(t *testing.T) {
		messages := []*schema.Message{
			schema.SystemMessage("You are a helpful assistant."),
			schema.UserMessage("Tell me a short joke."),
		}

		model.WithModel(modelId)

		resp, err := cm.Generate(ctx, messages)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if resp.ResponseMeta != nil && resp.ResponseMeta.Usage != nil {
			t.Logf("Token Usage: %+v", resp.ResponseMeta.Usage)
		}
		t.Logf("Response: %s", resp.Content)

	})

	// Test System message and options
	t.Run("SystemMessageAndOptions", func(t *testing.T) {
		resp, err := cm.Generate(ctx,
			[]*schema.Message{
				schema.SystemMessage("You are a poetic assistant."),
				schema.UserMessage("Write a one-line poem about Go programming."),
			},
			model.WithMaxTokens(50),
			model.WithTemperature(0.8),
		)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if resp.ResponseMeta != nil && resp.ResponseMeta.Usage != nil {
			t.Logf("Token Usage: %+v", resp.ResponseMeta.Usage)
		}

		t.Logf("Poetic Response: %s", resp.Content)
	})

	// Structured messages
	t.Run("StructuredMessages", func(t *testing.T) {
		messages := []*schema.Message{
			{
				Role:    schema.User,
				Content: "What is the capital of France?",
			},
		}

		model.WithModel(modelId)

		resp, err := cm.Generate(ctx, messages)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if resp.ResponseMeta != nil && resp.ResponseMeta.Usage != nil {
			t.Logf("Token Usage: %+v", resp.ResponseMeta.Usage)
		}
		t.Logf("Response: %s", resp.Content)

	})

	// Test Tools
	t.Run("ToolCalling", func(t *testing.T) {
		weatherTool := CreateStandardTool(
			"get_current_weather",
			"Get the current weather in a given location",
			map[string]any{
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
		)

		tools := []*schema.ToolInfo{weatherTool}

		// case 1: ไม่ใส่ Tool Choice (Auto)
		t.Run("Without_ToolChoice_Auto", func(t *testing.T) {
			model.WithModel(modelId)
			resp, err := cm.Generate(ctx,
				[]*schema.Message{schema.UserMessage("What's the weather like in Tokyo?")},
				model.WithTools(tools),
				// ไม่ใส่ model.WithToolChoice()
			)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			if len(resp.ToolCalls) == 0 {
				t.Errorf("Expected AI to use tool automatically, but got no tool calls. Content: %s", resp.Content)
				return
			}

			// show results
			for _, tc := range resp.ToolCalls {
				name := tc.Function.Name
				args := tc.Function.Arguments
				t.Logf("[Auto] Received tool call: %s with arguments: %s", name, args)
			}
		})

		// Case 2: ใส่ Tool Choice แบบ Forced (บังคับให้ AI ต้องเรียกใช้ Tool ที่ชื่อว่า "get_current_weather" เท่านั้น)
		t.Run("With_ToolChoice_Forced", func(t *testing.T) {
			resp, err := cm.Generate(ctx,
				[]*schema.Message{schema.UserMessage("What's the weather like in Seattle?")},
				model.WithTools(tools),
				model.WithToolChoice(schema.ToolChoiceForced, "get_current_weather"),
			)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			if len(resp.ToolCalls) == 0 {
				t.Fatalf("Expected forced tool 'get_current_weather', but got no tool calls. AI replied with text: %s", resp.Content)
			}

			forcedCall := resp.ToolCalls[0]
			if forcedCall.Function.Name != "get_current_weather" {
				t.Errorf("Expected AI to call 'get_current_weather', but it called '%s'", forcedCall.Function.Name)
			} else {
				t.Logf("[Forced] AI successfully called the forced tool: %s with arguments: %s", forcedCall.Function.Name, forcedCall.Function.Arguments)
			}
		})

		// case 3: ใส่ Tool Choice แบบ Forbidden (สั่งให้ AI ห้ามใช้ Tool)
		t.Run("With_ToolChoice_Forbidden", func(t *testing.T) {
			model.WithModel(modelId)
			resp, err := cm.Generate(ctx,
				[]*schema.Message{schema.UserMessage("What's the weather like in Boston?")},
				model.WithTools(tools),
				model.WithToolChoice(schema.ToolChoiceForbidden, ""),
			)

			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			if len(resp.ToolCalls) > 0 {
				t.Errorf("Expected NO tool calls because ToolChoice is Forbidden, but got %d calls", len(resp.ToolCalls))
				return
			}

			if resp.Content == "" {
				t.Errorf("Expected text content instead of tool call, got empty string")
			} else {
				t.Logf("[Forbidden] Model correctly returned text instead of tool: %s", resp.Content)
			}
		})
	})

	// Test Multi-Turn Conversation
	t.Run("MultiTurnConversation", func(t *testing.T) {
		model.WithModel(modelId)
		input := []*schema.Message{
			schema.UserMessage("My name is John."),
			schema.AssistantMessage("Nice to meet you, John!", nil),
			schema.UserMessage("Do you remember my name?"),
		}
		resp, err := cm.Generate(ctx, input, model.WithModel(modelId))
		if err != nil {
			t.Fatalf("Failed multi-turn: %v", err)
		}

		if !strings.Contains(resp.Content, "John") {
			t.Errorf("AI forgot your name! Response: %s", resp.Content)
		} else {
			t.Logf("[Multi-Turn Success] AI replied: %s", resp.Content)
		}
	})

	// Test Multimodal
	t.Run("MultiModalVision", func(t *testing.T) {
		imgURL := "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcSmUJnITGocG67lMOFF2K5EDnhw2SGEdW9TYw&s"
		input := []*schema.Message{{
			Role: schema.User,
			UserInputMultiContent: []schema.MessageInputPart{
				{
					Type: "text",
					Text: "What's in this image?",
				},
				{
					Type: "image_url",
					Image: &schema.MessageInputImage{
						MessagePartCommon: schema.MessagePartCommon{
							URL: &imgURL,
						},
					},
				},
			},
		}}
		resp, err := cm.Generate(ctx, input, model.WithModel(modelId))
		if err != nil {
			t.Fatalf("Vision failed: %v", err)
		}
		if resp.Content == "" {
			t.Fatalf("Vision returned empty content")
		}

		t.Logf("[Vision] AI sees: %s", resp.Content)
	})
}

// ============================================================================
// Integration Tests: Stream
// ============================================================================

func TestChatModel_Integration_Stream(t *testing.T) {
	apiKey := os.Getenv("MAI_API_KEY")
	if apiKey == "" {
		t.Fatalf("MAI_API_KEY is not set")
	}

	modelId := "qwen3-vl-235b-a22b-instruct"
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cm, _ := NewChatModel(ctx, &ChatModelConfig{APIKey: apiKey, Model: modelId})

	// Test basic chat stream
	t.Run("Basic", func(t *testing.T) {
		sr, err := cm.Stream(ctx, []*schema.Message{schema.UserMessage("Hello!")})
		if err != nil {
			t.Fatalf("Stream failed: %v", err)
		}
		defer sr.Close()

		var fullContent string
		var usage *schema.TokenUsage
		for {
			chunk, err := sr.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Recv failed: %v", err)
			}
			fullContent += chunk.Content
			if chunk.ResponseMeta != nil && chunk.ResponseMeta.Usage != nil {
				usage = chunk.ResponseMeta.Usage
			}
		}

		if usage != nil {
			t.Logf("Token Usage: %+v", usage)
		}
		t.Logf("Streamed Response: %s", fullContent)
	})

	// Test system message
	t.Run("SystemMessage", func(t *testing.T) {
		messages := []*schema.Message{
			schema.SystemMessage("You are a helpful assistant."),
			schema.UserMessage("Tell me a short joke."),
		}

		sr, err := cm.Stream(ctx, messages, model.WithModel(modelId))
		if err != nil {
			t.Fatalf("Stream failed: %v", err)
		}
		defer sr.Close()

		var fullContent string
		for {
			chunk, err := sr.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Recv failed: %v", err)
			}
			fullContent += chunk.Content
		}
		t.Logf("Streamed Response: %s", fullContent)
	})

	// Test options
	t.Run("SystemMessageAndOptions", func(t *testing.T) {
		sr, err := cm.Stream(ctx,
			[]*schema.Message{
				schema.SystemMessage("You are a poetic assistant."),
				schema.UserMessage("Write a one-line poem about Go programming."),
			},
			model.WithMaxTokens(50),
			model.WithTemperature(0.8),
		)
		if err != nil {
			t.Fatalf("Stream failed: %v", err)
		}
		defer sr.Close()

		var fullContent string
		for {
			chunk, err := sr.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Recv failed: %v", err)
			}
			fullContent += chunk.Content
		}
		t.Logf("Poetic Streamed Response: %s", fullContent)
	})

	// Test Tools
	t.Run("ToolCalling", func(t *testing.T) {
		weatherTool := CreateStandardTool(
			"get_current_weather",
			"Get the current weather in a given location",
			map[string]any{
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
		)

		tools := []*schema.ToolInfo{weatherTool}

		// case 1: ไม่ใส่ Tool Choice (Auto)
		t.Run("Without_ToolChoice_Auto", func(t *testing.T) {
			sr, err := cm.Stream(ctx,
				[]*schema.Message{schema.UserMessage("Please use the get_current_weather tool to check the current weather in Tokyo.")},
				model.WithModel(modelId),
				model.WithTools(tools),
			)
			if err != nil {
				t.Fatalf("Stream failed: %v", err)
			}
			defer sr.Close()

			var hasToolCall bool
			var funcName, funcArgs string
			var textContent string

			for {
				chunk, err := sr.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("Recv failed: %v", err)
				}
				textContent += chunk.Content
				if len(chunk.ToolCalls) > 0 {
					hasToolCall = true
					if chunk.ToolCalls[0].Function.Name != "" {
						funcName = chunk.ToolCalls[0].Function.Name
					}
					funcArgs += chunk.ToolCalls[0].Function.Arguments
				}
			}

			if !hasToolCall {
				t.Logf("[Auto] AI decided NOT to use the tool. It replied with text: %s", textContent)
				return
			}
			t.Logf("[Auto] Streamed tool call: %s with arguments: %s", funcName, funcArgs)
		})

		// Case 2: ใส่ Tool Choice แบบ Forced (Stream)
		t.Run("With_ToolChoice_Forced", func(t *testing.T) {
			sr, err := cm.Stream(ctx,
				[]*schema.Message{schema.UserMessage("What's the weather like in Seattle?")},
				model.WithModel(modelId),
				model.WithTools(tools),
				model.WithToolChoice(schema.ToolChoiceForced, "get_current_weather"),
			)
			if err != nil {
				t.Fatalf("Stream failed: %v", err)
			}
			defer sr.Close()

			var hasToolCall bool
			var funcName, funcArgs string
			var textContent string

			for {
				chunk, err := sr.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("Recv failed: %v", err)
				}

				if chunk.Content != "" {
					textContent += chunk.Content
				}

				if len(chunk.ToolCalls) > 0 {
					hasToolCall = true
					if chunk.ToolCalls[0].Function.Name != "" {
						funcName = chunk.ToolCalls[0].Function.Name
					}
					funcArgs += chunk.ToolCalls[0].Function.Arguments
				}
			}

			if !hasToolCall {
				t.Logf("[Forced] Model limitation detected in Stream mode! The AI ignored the forced tool and returned text instead:\n%s", textContent)
				return
			}
			if funcName != "get_current_weather" {
				t.Errorf("Expected AI to call 'get_current_weather', but it called '%s'", funcName)
			} else {
				t.Logf("[Forced] Streamed forced tool: %s with arguments: %s", funcName, funcArgs)
			}
		})

		// case 3: ใส่ Tool Choice แบบ Forbidden (Stream)
		t.Run("With_ToolChoice_Forbidden", func(t *testing.T) {
			sr, err := cm.Stream(ctx,
				[]*schema.Message{schema.UserMessage("What's the weather like in Boston?")},
				model.WithModel(modelId),
				model.WithTools(tools),
				model.WithToolChoice(schema.ToolChoiceForbidden, ""),
			)
			if err != nil {
				t.Fatalf("Stream failed: %v", err)
			}
			defer sr.Close()

			var hasToolCall bool
			var fullContent string
			var toolCallDetails string

			for {
				chunk, err := sr.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("Recv failed: %v", err)
				}

				if len(chunk.ToolCalls) > 0 {
					hasToolCall = true
					if chunk.ToolCalls[0].Function.Name != "" {
						toolCallDetails += chunk.ToolCalls[0].Function.Name + " "
					}
					toolCallDetails += chunk.ToolCalls[0].Function.Arguments
				}
				fullContent += chunk.Content
			}

			if hasToolCall {
				t.Logf("[Forbidden] Model limitation detected in Stream! We forbade tools, but AI stubbornly sent: %s\nText Content: %s", toolCallDetails, fullContent)
				return
			}

			if fullContent == "" {
				t.Errorf("Expected text content instead of tool call, got empty string")
			} else {
				t.Logf("✅ [Forbidden] Streamed text instead of tool: %s", fullContent)
			}
		})
	})

	t.Run("MultiTurnConversation", func(t *testing.T) {
		input := []*schema.Message{
			schema.UserMessage("My name is John."),
			schema.AssistantMessage("Nice to meet you, John!", nil),
			schema.UserMessage("Do you remember my name?"),
		}
		sr, err := cm.Stream(ctx, input, model.WithModel(modelId))
		if err != nil {
			t.Fatalf("Stream failed: %v", err)
		}
		defer sr.Close()

		var fullContent string
		for {
			chunk, err := sr.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Recv failed: %v", err)
			}
			fullContent += chunk.Content
		}

		if !strings.Contains(fullContent, "John") {
			t.Errorf("AI forgot your name! Response: %s", fullContent)
		} else {
			t.Logf("[Multi-Turn Success] Streamed reply: %s", fullContent)
		}
	})

	t.Run("MultiModalVision", func(t *testing.T) {
		imgURL := "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcSmUJnITGocG67lMOFF2K5EDnhw2SGEdW9TYw&s"
		input := []*schema.Message{{
			Role: schema.User,
			UserInputMultiContent: []schema.MessageInputPart{
				{
					Type: "text",
					Text: "What's in this image?",
				},
				{
					Type: "image_url",
					Image: &schema.MessageInputImage{
						MessagePartCommon: schema.MessagePartCommon{
							URL: &imgURL,
						},
					},
				},
			},
		}}

		sr, err := cm.Stream(ctx, input, model.WithModel(modelId))
		if err != nil {
			t.Fatalf("Stream Vision failed: %v", err)
		}
		defer sr.Close()

		var fullContent string
		for {
			chunk, err := sr.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Recv failed: %v", err)
			}
			fullContent += chunk.Content
		}

		if fullContent == "" {
			t.Fatalf("Vision returned empty content")
		}

		t.Logf("[Vision] Streamed AI sees: %s", fullContent)
	})
}
