package manageai

import (
	"fmt"
	"strings"

	"github.com/eino-contrib/jsonschema"
)

func getProp(tdef *ToolDefinition, key string, expectedTypes ...string) (map[string]any, error) {
	if tdef.Parameters == nil {
		return nil, fmt.Errorf("tool %s has no parameters", tdef.Name)
	}

	keyMap := strings.Split(key, ".")
	currentSchema := tdef.Parameters

	// Navigate through the property path
	for _, k := range keyMap {
		properties, ok := currentSchema["properties"]
		if !ok {
			return nil, fmt.Errorf("tool %s schema has no properties field", tdef.Name)
		}

		propertiesMap, ok := properties.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("tool %s properties is not a valid object", tdef.Name)
		}

		property, ok := propertiesMap[k]
		if !ok {
			return nil, fmt.Errorf("tool %s has no property %s", tdef.Name, k)
		}

		currentSchema, ok = property.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("tool %s property %s is not a valid schema object", tdef.Name, k)
		}
	}

	if len(expectedTypes) > 0 {
		propType, ok := currentSchema["type"]
		if !ok {
			return nil, fmt.Errorf("tool %s property %s has no type field", tdef.Name, key)
		}
		propTypeStr, ok := propType.(string)
		if !ok {
			return nil, fmt.Errorf("tool %s property %s type is not a string", tdef.Name, key)
		}
		for _, expectedType := range expectedTypes {
			if propTypeStr == expectedType {
				return currentSchema, nil
			}
		}
		return nil, fmt.Errorf("tool %s has property %s with type %s, expected %s", tdef.Name, key, propTypeStr, strings.Join(expectedTypes, ", "))
	}
	return currentSchema, nil
}

type Tools struct {
	Type     string         `json:"type"` // 'function'
	Function ToolDefinition `json:"function,omitzero"`
}

type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitzero"`
	Parameters  map[string]any `json:"parameters,omitzero"`
	Strict      bool           `json:"strict,omitzero"`
}

// To set enum constraint for a parameter with given key
// The key can be a dot-separated path to the parameter (e.g. "user.profile.email")
//
// Expected types: `string`, `number`, `integer`, `boolean`
//
// **NOTE**: If error occurs, it will be ignored and doesn't apply the constraint.
func (tdef *ToolDefinition) WithEnum(key string, values []any) *ToolDefinition {
	prop, err := getProp(tdef, key, "string", "number", "integer", "boolean")
	if err != nil {
		return tdef
	}
	prop["enum"] = values
	return tdef
}

// To set range constraint for a parameter with given key
// The key can be a dot-separated path to the parameter (e.g. "user.profile.age")
//
// Expected types: `number`, `integer`
//
// **NOTE**: If error occurs, it will be ignored and doesn't apply the constraint.
func (tdef *ToolDefinition) WithRange(key string, le, ge *float64) *ToolDefinition {
	prop, err := getProp(tdef, key, "number", "integer")
	if err != nil {
		return tdef
	}
	if le != nil {
		prop["maximum"] = *le
	}
	if ge != nil {
		prop["minimum"] = *ge
	}
	return tdef
}

// To set exclusive range constraint for a parameter with given key
// The key can be a dot-separated path to the parameter (e.g. "user.profile.age")
//
// Expected types: `number`, `integer`
//
// **NOTE**: If error occurs, it will be ignored and doesn't apply the constraint.
func (tdef *ToolDefinition) WithExclusiveRange(key string, lt, gt *float64) *ToolDefinition {
	prop, err := getProp(tdef, key, "number", "integer")
	if err != nil {
		return tdef
	}
	if lt != nil {
		prop["exclusiveMaximum"] = *lt
	}
	if gt != nil {
		prop["exclusiveMinimum"] = *gt
	}
	return tdef
}

// To set multiple of constraint for a parameter with given key
// The key can be a dot-separated path to the parameter (e.g. "user.profile.age")
//
// Expected types: `number`, `integer`
//
// **NOTE**: If error occurs, it will be ignored and doesn't apply the constraint.
func (tdef *ToolDefinition) WithMultipleOf(key string, multipleOf float64) *ToolDefinition {
	prop, err := getProp(tdef, key, "number", "integer")
	if err != nil {
		return tdef
	}
	prop["multipleOf"] = multipleOf
	return tdef
}

// To set length constraint for a parameter with given key
// The key can be a dot-separated path to the parameter (e.g. "user.profile.name")
//
// Expected types: `string`
//
// **NOTE**: If error occurs, it will be ignored and doesn't apply the constraint.
func (tdef *ToolDefinition) WithLength(key string, min, max *int) *ToolDefinition {
	prop, err := getProp(tdef, key, "string")
	if err != nil {
		return tdef
	}
	if min != nil {
		prop["minLength"] = *min
	}
	if max != nil {
		prop["maxLength"] = *max
	}
	return tdef
}

// To set pattern constraint for a regex parameter with given key
// The key can be a dot-separated path to the parameter (e.g. "user.profile.name")
//
// Expected types: `string`
//
// **NOTE**: If error occurs, it will be ignored and doesn't apply the constraint.
func (tdef *ToolDefinition) WithPattern(key string, pattern string) *ToolDefinition {
	prop, err := getProp(tdef, key, "string")
	if err != nil {
		return tdef
	}
	prop["pattern"] = pattern
	return tdef
}

// To set length constraint for an array parameter with given key
// The key can be a dot-separated path to the parameter (e.g. "user.profile.name")
//
// Expected types: `array`
//
// **NOTE**: If error occurs, it will be ignored and doesn't apply the constraint.
func (tdef *ToolDefinition) WithLengthItems(key string, min, max *int) *ToolDefinition {
	prop, err := getProp(tdef, key, "array")
	if err != nil {
		return tdef
	}
	if min != nil {
		prop["minItems"] = *min
	}
	if max != nil {
		prop["maxItems"] = *max
	}
	return tdef
}

// To set uniqueness constraint for an array parameter with given key
// The key can be a dot-separated path to the parameter (e.g. "user.profile.name")
//
// Expected types: `array`
//
// **NOTE**: If error occurs, it will be ignored and doesn't apply the constraint.
func (tdef *ToolDefinition) WithUniqueness(key string) *ToolDefinition {
	prop, err := getProp(tdef, key, "array")
	if err != nil {
		return tdef
	}
	prop["uniqueItems"] = true
	return tdef
}

type ToolChoiceReference struct {
	Name string `json:"name"`
}

type ToolChoice struct {
	Type     string              `json:"type"` // 'function'
	Function ToolChoiceReference `json:"function"`
}

type ResponseFormat struct {
	Type string `json:"type"` // 'text', 'json_object'
}

type ChatCompletionOptions struct {
	Model              *string             `json:"model,omitzero"`
	Stop               *[]string           `json:"stop,omitzero"`
	MaxTokens          *int64              `json:"max_tokens,omitzero"`
	N                  *int64              `json:"n,omitzero"`
	Logprobs           *bool               `json:"logprobs,omitzero"`
	TopLogprobs        *int64              `json:"top_logprobs,omitzero"`
	LogitBias          *map[string]float64 `json:"logit_bias,omitzero"`
	Seed               *int64              `json:"seed,omitzero"`
	Temperature        *float64            `json:"temperature,omitzero"`
	TopP               *float64            `json:"top_p,omitzero"`
	FrequencyPenalty   *float64            `json:"frequency_penalty,omitzero"`
	PresencePenalty    *float64            `json:"presence_penalty,omitzero"`
	ResponseFormat     *ResponseFormat     `json:"response_format,omitzero"`
	ChatTemplateKwargs *map[string]any     `json:"chat_template_kwargs,omitzero"`

	Tools             *[]Tools    `json:"tools,omitzero"`
	ToolChoice        *string     `json:"tool_choice,omitzero"`
	ToolChoiceRef     *ToolChoice `json:"tool_choice_ref,omitzero"`
	ParallelToolCalls *bool       `json:"parallel_tool_calls,omitzero"`

	GuidedJson              *jsonschema.Schema `json:"guided_json,omitzero"`
	GuidedRegex             *string            `json:"guided_regex,omitzero"`
	GuidedChoice            *[]string          `json:"guided_choice,omitzero"`
	GuidedGrammar           *string            `json:"guided_grammar,omitzero"`
	GuidedWhitespacePattern *string            `json:"guided_whitespace_pattern,omitzero"`
}

func (c *ChatCompletionOptions) WithModel(model string) *ChatCompletionOptions {
	c.Model = &model
	return c
}

func (c *ChatCompletionOptions) WithStop(stop []string) *ChatCompletionOptions {
	c.Stop = &stop
	return c
}

func (c *ChatCompletionOptions) WithMaxTokens(maxTokens int64) *ChatCompletionOptions {
	c.MaxTokens = &maxTokens
	return c
}

func (c *ChatCompletionOptions) WithN(n int64) *ChatCompletionOptions {
	c.N = &n
	return c
}

func (c *ChatCompletionOptions) WithLogprobs(logprobs bool) *ChatCompletionOptions {
	c.Logprobs = &logprobs
	return c
}

func (c *ChatCompletionOptions) WithTopLogprobs(topLogprobs int64) *ChatCompletionOptions {
	c.TopLogprobs = &topLogprobs
	return c
}

func (c *ChatCompletionOptions) WithLogitBias(logitBias map[string]float64) *ChatCompletionOptions {
	c.LogitBias = &logitBias
	return c
}

func (c *ChatCompletionOptions) WithSeed(seed int64) *ChatCompletionOptions {
	c.Seed = &seed
	return c
}

func (c *ChatCompletionOptions) WithTemperature(temperature float64) *ChatCompletionOptions {
	c.Temperature = &temperature
	return c
}

func (c *ChatCompletionOptions) WithTopP(topP float64) *ChatCompletionOptions {
	c.TopP = &topP
	return c
}

func (c *ChatCompletionOptions) WithFrequencyPenalty(frequencyPenalty float64) *ChatCompletionOptions {
	c.FrequencyPenalty = &frequencyPenalty
	return c
}

func (c *ChatCompletionOptions) WithPresencePenalty(presencePenalty float64) *ChatCompletionOptions {
	c.PresencePenalty = &presencePenalty
	return c
}

func (c *ChatCompletionOptions) WithResponseFormat(responseFormat *string) *ChatCompletionOptions {
	if responseFormat != nil {
		switch *responseFormat {
		case "json_object":
			c.ResponseFormat = &ResponseFormat{
				Type: "json_object",
			}
		case "text":
			c.ResponseFormat = &ResponseFormat{
				Type: "text",
			}
		default:
			return c
		}
	} else {
		c.ResponseFormat = nil
	}
	return c
}

func (c *ChatCompletionOptions) WithChatTemplateKwargs(chatTemplateKwargs map[string]any) *ChatCompletionOptions {
	c.ChatTemplateKwargs = &chatTemplateKwargs
	return c
}

func (c *ChatCompletionOptions) WithTools(tools []Tools, toolChoice *string) *ChatCompletionOptions {
	c.Tools = &tools
	if toolChoice != nil {
		setToolChoiceByName := false
		for _, tool := range tools {
			if tool.Function.Name == *toolChoice {
				c.ToolChoiceRef = &ToolChoice{
					Type: "function",
					Function: ToolChoiceReference{
						Name: tool.Function.Name,
					},
				}
				setToolChoiceByName = true
				break
			}
		}
		if !setToolChoiceByName {
			c.ToolChoice = toolChoice
		}
	}
	return c
}

func (c *ChatCompletionOptions) WithParallelToolCalls(parallelToolCalls bool) *ChatCompletionOptions {
	c.ParallelToolCalls = &parallelToolCalls
	return c
}

func (c *ChatCompletionOptions) WithGuidedJson(guidedJson jsonschema.Schema) *ChatCompletionOptions {
	c.GuidedJson = &guidedJson
	c.ResponseFormat = &ResponseFormat{
		Type: "json_object",
	}
	return c
}

func (c *ChatCompletionOptions) WithGuidedRegex(guidedRegex string) *ChatCompletionOptions {
	c.GuidedRegex = &guidedRegex
	c.ResponseFormat = &ResponseFormat{
		Type: "text",
	}
	return c
}

func (c *ChatCompletionOptions) WithGuidedChoice(guidedChoice []string) *ChatCompletionOptions {
	c.GuidedChoice = &guidedChoice
	c.ResponseFormat = &ResponseFormat{
		Type: "text",
	}
	return c
}

func (c *ChatCompletionOptions) WithGuidedGrammar(guidedGrammar string) *ChatCompletionOptions {
	c.GuidedGrammar = &guidedGrammar
	c.ResponseFormat = &ResponseFormat{
		Type: "text",
	}
	return c
}

func (c *ChatCompletionOptions) WithGuidedWhitespacePattern(guidedWhitespacePattern string) *ChatCompletionOptions {
	c.GuidedWhitespacePattern = &guidedWhitespacePattern
	c.ResponseFormat = &ResponseFormat{
		Type: "text",
	}
	return c
}

func (c *ChatCompletionOptions) Marshal() (map[string]any, error) {
	result := map[string]any{}
	if c.Model != nil {
		result["model"] = *c.Model
	}
	if c.Stop != nil {
		result["stop"] = *c.Stop
	}
	if c.MaxTokens != nil {
		result["max_tokens"] = *c.MaxTokens
	}
	if c.N != nil {
		result["n"] = *c.N
	}
	if c.Logprobs != nil {
		result["logprobs"] = *c.Logprobs
	}
	if c.TopLogprobs != nil {
		result["top_logprobs"] = *c.TopLogprobs
	}
	if c.LogitBias != nil {
		result["logit_bias"] = *c.LogitBias
	}
	if c.Seed != nil {
		result["seed"] = *c.Seed
	}
	if c.Temperature != nil {
		result["temperature"] = *c.Temperature
	}
	if c.TopP != nil {
		result["top_p"] = *c.TopP
	}
	if c.FrequencyPenalty != nil {
		result["frequency_penalty"] = *c.FrequencyPenalty
	}
	if c.PresencePenalty != nil {
		result["presence_penalty"] = *c.PresencePenalty
	}
	if c.ResponseFormat != nil {
		result["response_format"] = c.ResponseFormat
	}
	if c.ChatTemplateKwargs != nil {
		result["chat_template_kwargs"] = *c.ChatTemplateKwargs
	}
	if c.Tools != nil {
		result["tools"] = *c.Tools
	}
	if c.ToolChoiceRef != nil {
		result["tool_choice"] = c.ToolChoiceRef
	} else if c.ToolChoice != nil {
		result["tool_choice"] = *c.ToolChoice
	}
	if c.ParallelToolCalls != nil {
		result["parallel_tool_calls"] = *c.ParallelToolCalls
	}
	if c.GuidedJson != nil {
		result["guided_json"] = c.GuidedJson
	}
	if c.GuidedRegex != nil {
		result["guided_regex"] = *c.GuidedRegex
	}
	if c.GuidedChoice != nil {
		result["guided_choice"] = *c.GuidedChoice
	}
	if c.GuidedGrammar != nil {
		result["guided_grammar"] = *c.GuidedGrammar
	}
	if c.GuidedWhitespacePattern != nil {
		result["guided_whitespace_pattern"] = *c.GuidedWhitespacePattern
	}
	return result, nil
}

type ChatCompletion struct {
	Messages []Message              `json:"messages"`
	Options  *ChatCompletionOptions `json:"options,omitzero"`
}

func CreateChatCompletion(messages []any, options *ChatCompletionOptions) (*ChatCompletion, error) {
	mes := []Message{}
	for i, m := range messages {
		mStr, ok := m.(string)
		if ok {
			mes = append(mes, NewHumanMessage(mStr, nil))
			continue
		} else {
			mInterface, ok := m.(Message)
			if ok {
				mes = append(mes, mInterface)
				continue
			}
			mMap, ok := m.(map[string]any)
			if ok {
				role, ok := getFirst(mMap, "role")
				if !ok {
					return nil, fmt.Errorf("message at position %d is missing 'role' field", i)
				}
				switch role {
				case "system", "developer":
					content, _ := getFirst(mMap, "content")
					id, _ := getFirst(mMap, "id")
					contentStr, ok := content.(string)
					if !ok {
						return nil, fmt.Errorf("content of system message should be type string at position %d, got %T", i, contentStr)
					}
					idStr, ok := id.(*string)
					if !ok {
						idStr = nil
					}

					mes = append(mes, NewSystemMessage(contentStr, idStr))

				case "human", "user":
					id, _ := getFirst(mMap, "id")
					idStr, ok := id.(*string)
					if !ok {
						idStr = nil
					}

					content, _ := getFirst(mMap, "content")
					if contentStr, ok := content.(string); ok {
						mes = append(mes, NewHumanMessage(contentStr, idStr))
					} else if contentSlice, ok := content.([]any); ok {
						contentBlocks, err := NewContentBlocks(contentSlice)
						if err != nil {
							return nil, fmt.Errorf("error creating content blocks for human message at position %d: %v", i, err)
						}
						mes = append(mes, NewMultiModalHumanMessage(*contentBlocks, idStr))
					} else if contentMap, ok := content.(map[string]any); ok {
						contentBlock, err := NewContentBlocks([]any{contentMap})
						if err != nil {
							return nil, fmt.Errorf("error creating content block for human message at position %d: %v", i, err)
						}
						mes = append(mes, NewMultiModalHumanMessage(*contentBlock, idStr))
					} else {
						return nil, fmt.Errorf("content of human message should be type string or []any or map[string]any at position %d, got %T", i, content)
					}

				case "assistant", "ai":
					content, _ := getFirst(mMap, "content")
					id, _ := getFirst(mMap, "id")

					idStr, ok := id.(*string)
					if !ok {
						idStr = nil
					}

					toolCallsAny, _ := getFirst(mMap, "tool_calls")
					toolCalls := []ToolCall{}
					if toolCallsAny != nil {
						toolCallsSlice, ok := toolCallsAny.([]any)
						if !ok {
							return nil, fmt.Errorf("tool_calls should be type []any at position %d, got %T", i, toolCallsAny)
						}
						for _, tc := range toolCallsSlice {
							toolCall, err := NewToolCall(tc.(map[string]any))
							if err != nil {
								return nil, fmt.Errorf("error creating tool call for assistant message at position %d: %v", i, err)
							}
							if ok {
								toolCalls = append(toolCalls, *toolCall)
							}
						}
					}
					mes = append(mes, NewAssistantMessage(content.(string), idStr, &toolCalls, nil))

				case "tool", "function":
					content, _ := getFirst(mMap, "content")
					contentStr, ok := content.(string)
					if !ok {
						return nil, fmt.Errorf("content of tool message should be type string at position %d, got %T", i, content)
					}
					id, _ := getFirst(mMap, "id")
					toolId, _ := getFirst(mMap, "extra", "tool_id")

					idStr, ok := id.(*string)
					if !ok {
						idStr = nil
					}
					toolIdStr, ok := toolId.(*string)
					if !ok {
						toolIdStr = nil
					}

					mes = append(mes, NewToolMessage(contentStr, idStr, toolIdStr))

				default:
					return nil, fmt.Errorf("unsupported message roleat position %d: %s", i, role)
				}
			} else {
				return nil, fmt.Errorf("message at position %d should be type string or Message struct or map[string]any, got %T", i, m)
			}
		}
	}
	return &ChatCompletion{
		Messages: mes,
		Options:  options,
	}, nil
}

func (c *ChatCompletion) Marshal() (map[string]any, error) {
	result := map[string]any{
		"messages": c.Messages,
	}

	if c.Options != nil {
		optionsMap, err := c.Options.Marshal()
		if err != nil {
			return nil, err
		}
		for k, v := range optionsMap {
			result[k] = v
		}
	}
	return result, nil
}

// ====================== Embedding ==========================
type EmbeddingOptions struct {
	Model          *string `json:"model,omitzero"`
	EncodingFormat *string `json:"encoding_format,omitzero"`
}

func (e *EmbeddingOptions) WithModel(model string) *EmbeddingOptions {
	e.Model = &model
	return e
}

func (e *EmbeddingOptions) WithEncodingFormat(encodingFormat string) *EmbeddingOptions {
	e.EncodingFormat = &encodingFormat
	return e
}

type EmbeddingRequest struct {
	Input          []string `json:"input"`
	Model          string   `json:"model"`
	EncodingFormat *string  `json:"encoding_format,omitzero"`
}

func CreateEmbeddingRequest(input []string, options *EmbeddingOptions) (*EmbeddingRequest, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("input cannot be empty")
	}
	if options == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}
	if options.Model == nil {
		return nil, fmt.Errorf("model is required in options")
	}

	return &EmbeddingRequest{
		Input:          input,
		Model:          *options.Model,
		EncodingFormat: options.EncodingFormat,
	}, nil
}

func (e *EmbeddingRequest) Marshal() (map[string]any, error) {
	result := map[string]any{
		"input": e.Input,
		"model": e.Model,
	}
	if e.EncodingFormat != nil {
		result["encoding_format"] = *e.EncodingFormat
	}
	return result, nil
}

// ====================== Reranker ==========================
type RerankOptions struct {
	Model           *string `json:"model,omitzero"`
	TopN            *int    `json:"top_n,omitzero"`
	ReturnDocuments *bool   `json:"return_documents,omitzero"`
	MaxChunksPerDoc *int    `json:"max_chunks_per_doc,omitzero"`
}

func (r *RerankOptions) WithModel(model string) *RerankOptions {
	r.Model = &model
	return r
}

func (r *RerankOptions) WithTopN(topN int) *RerankOptions {
	r.TopN = &topN
	return r
}

func (r *RerankOptions) WithReturnDocuments(returnDocuments bool) *RerankOptions {
	r.ReturnDocuments = &returnDocuments
	return r
}

func (r *RerankOptions) WithMaxChunksPerDoc(maxChunksPerDoc int) *RerankOptions {
	r.MaxChunksPerDoc = &maxChunksPerDoc
	return r
}

type RerankRequest struct {
	Model           string   `json:"model"`
	Query           string   `json:"query"`
	Documents       []string `json:"documents"`
	TopN            *int     `json:"top_n,omitzero"`
	ReturnDocuments *bool    `json:"return_documents,omitzero"`
	MaxChunksPerDoc *int     `json:"max_chunks_per_doc,omitzero"`
}

func CreateRerankRequest(query string, documents []string, options *RerankOptions) (*RerankRequest, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	if len(documents) == 0 {
		return nil, fmt.Errorf("documents cannot be empty")
	}
	if options == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}
	if options.Model == nil {
		return nil, fmt.Errorf("model is required in options")
	}

	return &RerankRequest{
		Query:           query,
		Documents:       documents,
		Model:           *options.Model,
		TopN:            options.TopN,
		ReturnDocuments: options.ReturnDocuments,
		MaxChunksPerDoc: options.MaxChunksPerDoc,
	}, nil
}

func (r *RerankRequest) Marshal() (map[string]any, error) {
	result := map[string]any{
		"query":     r.Query,
		"documents": r.Documents,
		"model":     r.Model,
	}
	if r.TopN != nil {
		result["top_n"] = *r.TopN
	}
	if r.ReturnDocuments != nil {
		result["return_documents"] = *r.ReturnDocuments
	}
	if r.MaxChunksPerDoc != nil {
		result["max_chunks_per_doc"] = *r.MaxChunksPerDoc
	}
	return result, nil
}

// ====================== Detokenize ==========================
type DetokenizeOptions struct {
	Model *string `json:"model,omitzero"`
}

func (d *DetokenizeOptions) WithModel(model string) *DetokenizeOptions {
	d.Model = &model
	return d
}

type DetokenizeRequest struct {
	Model  string `json:"model"`
	Tokens []int  `json:"tokens"`
}

func CreateDetokenizeRequest(tokens []int, options *DetokenizeOptions) (*DetokenizeRequest, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("tokens cannot be empty")
	}
	if options == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}
	if options.Model == nil {
		return nil, fmt.Errorf("model is required in options")
	}
	return &DetokenizeRequest{
		Tokens: tokens,
		Model:  *options.Model,
	}, nil
}

func (d *DetokenizeRequest) Marshal() (map[string]any, error) {
	result := map[string]any{
		"tokens": d.Tokens,
		"model":  d.Model,
	}
	return result, nil
}
