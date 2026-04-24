package manageai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"

	manageai_internal "github.com/manageai-inet/Eino-ManageAI-Extension/components/core/manageai"
)

type ChatModelConfig struct {
	APIKey           string                               `json:"api_key"`
	Timeout          time.Duration                        `json:"timeout"`
	HTTPClient       *http.Client                         `json:"http_client"`
	BaseURL          string                               `json:"base_url"`
	Model            string                               `json:"model"`
	MaxTokens        *int                                 `json:"max_tokens,omitempty"`
	Temperature      *float32                             `json:"temperature,omitempty"`
	TopP             *float32                             `json:"top_p,omitempty"`
	Stop             []string                             `json:"stop,omitempty"`
	PresencePenalty  *float32                             `json:"presence_penalty,omitempty"`
	ResponseFormat   *openai.ChatCompletionResponseFormat `json:"response_format,omitempty"`
	Seed             *int                                 `json:"seed,omitempty"`
	FrequencyPenalty *float32                             `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]int                       `json:"logit_bias,omitempty"`
	GuidedJson       *jsonschema.Schema                   `json:"guided_json,omitempty"`
	GuidedRegex      *string                              `json:"guided_regex,omitempty"`
}

type ChatModel struct {
	cli    *manageai_internal.Client
	config *ChatModelConfig
	tools  []*schema.ToolInfo
}

// =============================================================================================
// Constructors & Factory
// =============================================================================================
func NewChatModel(ctx context.Context, config *ChatModelConfig) (*ChatModel, error) {
	if config == nil {
		return nil, fmt.Errorf("[NewChatModel] config not provided")
	}

	cli, err := manageai_internal.NewClient(&config.Model, config.APIKey)
	if err != nil {
		return nil, err
	}

	if config.Timeout > 0 {
		cli.Timeout = config.Timeout
		if cli.HTTPClient != nil {
			cli.HTTPClient.Timeout = config.Timeout
		}
	}

	return &ChatModel{
		cli:    cli,
		config: config,
	}, nil
}

// CreateStandardTool เป็น Factory Function สำหรับสร้าง schema.ToolInfo
func CreateStandardTool(name string, description string, parameters map[string]any) *schema.ToolInfo {
	// ถ้าไม่ได้ส่ง parameters มา ให้สร้าง Object ว่าง ป้องกัน API Error
	if parameters == nil {
		parameters = map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	// Standard JSON Map
	b, _ := json.Marshal(parameters)
	var jSchema jsonschema.Schema
	_ = json.Unmarshal(b, &jSchema)

	return &schema.ToolInfo{
		Name:        name,
		Desc:        description,
		ParamsOneOf: schema.NewParamsOneOfByJSONSchema(&jSchema),
	}
}

// =============================================================================================
// Core Interface Implementation
// =============================================================================================

func (cm *ChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	modelName := cm.GetModelName()
	ctx = callbacks.EnsureRunInfo(ctx, modelName, cm.GetType())
	if err := validateToolOptions(opts...); err != nil {
		return nil, fmt.Errorf("invalid tool options: %w", err)
	}

	nConfig := cm.buildOptions(opts...)

	internalOpts := cm.toInternalOptions(nConfig)

	manageAIReqMessages := toManageAIMessages(input)

	resp, err := cm.cli.ChatCompletion(manageAIReqMessages, internalOpts)
	if err != nil {
		return nil, fmt.Errorf("manageai generate failed: %w", err)
	}

	if resp == nil {
		return nil, fmt.Errorf("manageai returned nil response")
	}

	return toEinoMessage(resp), nil
}

func (cm *ChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	modelName := cm.GetModelName()
	ctx = callbacks.EnsureRunInfo(ctx, modelName, cm.GetType())

	nConfig := cm.buildOptions(opts...)
	internalOpts := cm.toInternalOptions(nConfig) // ใช้ Helper

	manageAIReqMessages := toManageAIMessages(input)

	stream, err := cm.cli.ChatCompletionStream(ctx, manageAIReqMessages, internalOpts)
	if err != nil {
		return nil, fmt.Errorf("manageai stream failed: %w", err)
	}

	sr, sw := schema.Pipe[*schema.Message](1)

	go func() {
		defer sw.Close()

		for {
			chunk, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					return
				}
				sw.Send(nil, err)
				return
			}

			sw.Send(toEinoChunk(chunk), nil)
		}
	}()

	return sr, nil
}

func (cm *ChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	if cm == nil {
		return nil, fmt.Errorf("chat model is nil")
	}
	return &ChatModel{
		cli:    cm.cli,
		config: cm.config,
		tools:  tools,
	}, nil
}

func (cm *ChatModel) GetType() components.Component {
	return components.ComponentOfChatModel
}

// =============================================================================================
// Data Adapters / Mappers
// =============================================================================================
func toManageAIMessages(in []*schema.Message) []any {
	var out []any

	for _, msg := range in {
		m := map[string]any{
			"role": string(msg.Role),
		}

		if len(msg.UserInputMultiContent) > 0 {
			var parts []any
			for _, p := range msg.UserInputMultiContent {
				if string(p.Type) == "text" {
					parts = append(parts, map[string]any{"type": "text", "text": p.Text})
				} else if string(p.Type) == "image_url" && p.Image != nil {
					urlStr := ""
					if p.Image.URL != nil {
						urlStr = *p.Image.URL
					}

					parts = append(parts, map[string]any{
						"type":      "image_url",
						"image_url": urlStr,
					})
				}
			}
			m["content"] = parts

		} else if len(msg.MultiContent) > 0 {
			var parts []any
			for _, p := range msg.MultiContent {
				if string(p.Type) == "text" {
					parts = append(parts, map[string]any{"type": "text", "text": p.Text})
				} else if string(p.Type) == "image_url" && p.ImageURL != nil {
					parts = append(parts, map[string]any{
						"type":      "image_url",
						"image_url": map[string]any{"url": p.ImageURL.URL},
					})
				}
			}
			m["content"] = parts

		} else {
			m["content"] = msg.Content
		}

		if msg.Role == schema.Tool {
			m["tool_call_id"] = msg.ToolCallID
		}

		if len(msg.ToolCalls) > 0 {
			var tcs []any
			for _, tc := range msg.ToolCalls {
				tcs = append(tcs, map[string]any{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				})
			}
			m["tool_calls"] = tcs
		}

		out = append(out, m)
	}

	return out
}

func toEinoMessage(resp *manageai_internal.ChatCompletionResponse) *schema.Message {
	if resp == nil || len(resp.Choices) == 0 {
		return &schema.Message{Role: schema.Assistant}
	}

	choice := resp.Choices[0]
	einoMsg := &schema.Message{
		Role: schema.Assistant,
	}

	if resp.Usage != nil {
		einoMsg.ResponseMeta = &schema.ResponseMeta{
			Usage: &schema.TokenUsage{
				PromptTokens:     int(resp.Usage.PromptTokens),
				CompletionTokens: int(resp.Usage.CompletionTokens),
				TotalTokens:      int(resp.Usage.TotalTokens),
			},
		}
	}

	if choice.Message.Content != nil {
		einoMsg.Content = *choice.Message.Content
	}

	if choice.Message.ToolsCalls != nil {
		for _, tc := range *choice.Message.ToolsCalls {
			name := ""
			if tc.Function.Name != nil {
				name = *tc.Function.Name
			}
			args := ""
			if tc.Function.Arguments != nil {
				args = *tc.Function.Arguments
			}

			einoMsg.ToolCalls = append(einoMsg.ToolCalls, schema.ToolCall{
				Function: schema.FunctionCall{
					Name:      name,
					Arguments: args,
				},
			})
		}
	}

	return einoMsg
}

func toEinoChunk(chunk *manageai_internal.ChatCompletionStreamChunk) *schema.Message {
	if chunk == nil || len(chunk.Choices) == 0 {
		return &schema.Message{Role: schema.Assistant, Content: ""}
	}

	msg := &schema.Message{
		Role: schema.Assistant,
	}

	delta := chunk.Choices[0].Delta

	if delta.Content != nil {
		msg.Content = *delta.Content
	}

	if delta.ToolsCalls != nil {
		for _, tc := range *delta.ToolsCalls {
			einoTc := schema.ToolCall{}

			if tc.Id != nil {
				einoTc.ID = *tc.Id
			}

			funcCall := schema.FunctionCall{}
			if tc.Function.Name != nil {
				funcCall.Name = *tc.Function.Name
			}
			if tc.Function.Arguments != nil {
				funcCall.Arguments = *tc.Function.Arguments
			}
			einoTc.Function = funcCall

			if tc.Index != nil {
				idx := int(*tc.Index)
				einoTc.Index = &idx
			}

			msg.ToolCalls = append(msg.ToolCalls, einoTc)
		}
	}

	if chunk.Usage != nil {
		msg.ResponseMeta = &schema.ResponseMeta{
			Usage: &schema.TokenUsage{
				PromptTokens:     int(chunk.Usage.PromptTokens),
				CompletionTokens: int(chunk.Usage.CompletionTokens),
				TotalTokens:      int(chunk.Usage.TotalTokens),
			},
		}
	}

	return msg
}

func (cm *ChatModel) buildOptions(opts ...model.Option) *openai.Config {
	common := model.GetCommonOptions(&model.Options{}, opts...)

	var defaultResponseFormat *string
	if cm.config.ResponseFormat != nil {
		s := string(cm.config.ResponseFormat.Type)
		defaultResponseFormat = &s
	}

	extra := model.GetImplSpecificOptions(&manageAIOptions{
		ResponseFormat: defaultResponseFormat,
		GuidedJson:     cm.config.GuidedJson,
		GuidedRegex:    cm.config.GuidedRegex,
	}, opts...)

	nConfig := &openai.Config{
		Model:            cm.config.Model,
		MaxTokens:        cm.config.MaxTokens,
		Temperature:      cm.config.Temperature,
		TopP:             cm.config.TopP,
		Stop:             cm.config.Stop,
		PresencePenalty:  cm.config.PresencePenalty,
		ResponseFormat:   cm.config.ResponseFormat,
		Seed:             cm.config.Seed,
		FrequencyPenalty: cm.config.FrequencyPenalty,
		LogitBias:        cm.config.LogitBias,
	}

	if common != nil {
		if common.MaxTokens != nil {
			nConfig.MaxTokens = common.MaxTokens
		}
		if common.Temperature != nil {
			nConfig.Temperature = common.Temperature
		}
		if common.TopP != nil {
			nConfig.TopP = common.TopP
		}
		if len(common.Stop) > 0 {
			nConfig.Stop = common.Stop
		}
	}

	extraFields := make(map[string]any)

	activeTools := cm.tools
	if common != nil && len(common.Tools) > 0 {
		activeTools = common.Tools
	}

	var internalTools []manageai_internal.Tools
	if len(activeTools) > 0 {
		for _, t := range activeTools {
			var params map[string]any
			if t.ParamsOneOf != nil {
				schemaObj, err := t.ParamsOneOf.ToJSONSchema()

				if err == nil && schemaObj != nil {
					bSchema, _ := json.Marshal(schemaObj)
					_ = json.Unmarshal(bSchema, &params)
				}
			}

			if params == nil {
				params = map[string]any{"type": "object", "properties": map[string]any{}}
			}

			internalTools = append(internalTools, manageai_internal.Tools{
				Type: "function",
				Function: manageai_internal.ToolDefinition{
					Name:        t.Name,
					Description: t.Desc,
					Parameters:  params,
				},
			})
		}

		extraFields["tools"] = internalTools

		if common != nil && common.ToolChoice != nil {
			switch *common.ToolChoice {
			case schema.ToolChoiceForced:
				if len(common.AllowedToolNames) > 0 {
					extraFields["tool_choice"] = map[string]any{
						"type": "function",
						"function": map[string]any{
							"name": common.AllowedToolNames[0],
						},
					}
				}
			case schema.ToolChoiceForbidden:
				extraFields["tool_choice"] = "none"
			}
		}
	}

	if extra != nil {
		if extra.GuidedJson != nil {
			extraFields["guided_json"] = extra.GuidedJson
		}
		if extra.GuidedRegex != nil {
			extraFields["guided_regex"] = *extra.GuidedRegex
		}
	}

	if len(extraFields) > 0 {
		nConfig.ExtraFields = extraFields
	}

	return nConfig
}

func (cm *ChatModel) toInternalOptions(nConfig *openai.Config) *manageai_internal.ChatCompletionOptions {
	opts := &manageai_internal.ChatCompletionOptions{
		MaxTokens:        intToInt64Ptr(nConfig.MaxTokens),
		Temperature:      float32To64Ptr(nConfig.Temperature),
		TopP:             float32To64Ptr(nConfig.TopP),
		PresencePenalty:  float32To64Ptr(nConfig.PresencePenalty),
		FrequencyPenalty: float32To64Ptr(nConfig.FrequencyPenalty),
		Seed:             intToInt64Ptr(nConfig.Seed),
	}

	if len(nConfig.Stop) > 0 {
		stopSeqs := nConfig.Stop
		opts.Stop = &stopSeqs
	}

	if nConfig.ResponseFormat != nil {
		strVal := string(nConfig.ResponseFormat.Type)
		internalFormat := manageai_internal.ResponseFormat{
			Type: strVal,
		}
		opts.ResponseFormat = &internalFormat
	}

	if nConfig.ExtraFields != nil {
		if toolsRaw, exists := nConfig.ExtraFields["tools"]; exists {
			if internalTools, ok := toolsRaw.([]manageai_internal.Tools); ok {
				opts.WithTools(internalTools, nil)
			}
		}

		if tcRaw, exists := nConfig.ExtraFields["tool_choice"]; exists {
			if tcStr, isStr := tcRaw.(string); isStr {
				if tcStr == "none" {
					opts.ToolChoice = &tcStr
				}
			} else if tcObj, isObj := tcRaw.(map[string]any); isObj {
				// บังคับ Object ToolChoiceRef ลงไปโดยตรง
				if fnMap, hasFn := tcObj["function"].(map[string]any); hasFn {
					if name, hasName := fnMap["name"].(string); hasName {
						opts.ToolChoiceRef = &manageai_internal.ToolChoice{
							Type: "function",
							Function: manageai_internal.ToolChoiceReference{
								Name: name,
							},
						}
						opts.ToolChoice = nil
					}
				}
			}
		}
	}

	return opts
}

// =============================================================================================
// Utilities & Validation
// =============================================================================================

func validateToolOptions(opts ...model.Option) error {
	modelOptions := model.GetCommonOptions(&model.Options{}, opts...)
	if modelOptions.ToolChoice != nil {
		if *modelOptions.ToolChoice == schema.ToolChoiceAllowed && len(modelOptions.AllowedToolNames) > 0 {
			return fmt.Errorf("tool_choice 'allowed' is not supported when allowed tool names are present")
		}

		// เช็คว่าต้องมี Tool Name อย่างน้อย 1 ชื่อเมื่อใช้ 'forced'
		if *modelOptions.ToolChoice == schema.ToolChoiceForced {
			if len(modelOptions.AllowedToolNames) == 0 {
				return fmt.Errorf("at least one allowed tool name is required for tool_choice 'forced'")
			}
			if len(modelOptions.AllowedToolNames) > 1 {
				return fmt.Errorf("only one allowed tool name can be configured for tool_choice 'forced'")
			}
		}
	}
	return nil
}

func (cm *ChatModel) GetModelName() string {
	if cm.config != nil && cm.config.Model != "" {
		return cm.config.Model
	}
	return ""
}

func intToInt64Ptr(i *int) *int64 {
	if i == nil {
		return nil
	}
	v := int64(*i)
	return &v
}

func float32To64Ptr(f *float32) *float64 {
	if f == nil {
		return nil
	}
	v := float64(*f)
	return &v
}
