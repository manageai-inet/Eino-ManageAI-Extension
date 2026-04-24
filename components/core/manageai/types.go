package manageai

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type ContentBlock struct {
	Type string `json:"type"`
	TextContentBlock
	ImageUrlContentBlock
}

type TextContentBlock struct {
	Text string `json:"text"`
}

func NewTextContentBlock(text string) ContentBlock {
	return ContentBlock{
		Type: "text",
		TextContentBlock: TextContentBlock{
			Text: text,
		},
	}
}

type ImageUrlContentBlock struct {
	ImageUrl string `json:"image_url"`
}

func NewImageUrlContentBlock(imageUrl string) ContentBlock {
	return ContentBlock{
		Type: "image_url",
		ImageUrlContentBlock: ImageUrlContentBlock{
			ImageUrl: imageUrl,
		},
	}
}

func NewContentBlocks(content_blocks []any) (*[]ContentBlock, error) {
	var blocks []ContentBlock

	for _, cb := range content_blocks {
		if cbStr, ok := cb.(string); ok {
			// If the content block is a string, treat it as a text content block
			textBlock := NewTextContentBlock(cbStr)
			blocks = append(blocks, textBlock)
		} else if cbMap, ok := cb.(map[string]any); ok {
			cbType, ok := cbMap["type"]
			if !ok {
				return nil, fmt.Errorf("content block missing 'type' field")
			}
			cbTypeStr, ok := cbType.(string)
			if !ok {
				return nil, fmt.Errorf("content block 'type' field is not a string, got %T", cbTypeStr)
			}
			switch cbTypeStr {
			case "text":
				text, ok := cbMap["text"].(string)
				if !ok {
					return nil, fmt.Errorf("content block of type 'text' is missing 'text' field")
				}
				textBlock := NewTextContentBlock(text)
				blocks = append(blocks, textBlock)
			case "image_url":
				imageUrl, ok := getFirst(cbMap, "image_url", "url")
				if !ok {
					return nil, fmt.Errorf("content block of type 'image_url' is missing 'image_url' or 'url' field")
				}
				imageUrlBlock := NewImageUrlContentBlock(imageUrl.(string))
				blocks = append(blocks, imageUrlBlock)
			default:
				return nil, fmt.Errorf("unsupported content block type: %s", cbType)
			}
		}
	}

	return &blocks, nil
}

type Message struct {
	Id      string         `json:"id"`
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`

	// only for AssistantMessage
	ToolCalls []ToolCall      `json:"tool_calls,omitzero"`
	Extra     *map[string]any `json:"extra,omitzero"`
}

func (m *Message) Marshal() map[string]any {
	contents := []map[string]any{}
	result := map[string]any{
		"id":      m.Id,
		"role":    m.Role,
		"content": contents,
	}
	if len(m.Content) > 0 {
		for _, tb := range m.Content {
			switch tb.Type {
			case "text":
				contents = append(contents, map[string]any{
					"type": tb.Type,
					"text": tb.Text,
				})
			case "image_url":
				contents = append(contents, map[string]any{
					"type":      tb.Type,
					"image_url": tb.ImageUrl,
				})
			default:
				continue
			}
		}
	}

	if len(m.ToolCalls) > 0 {
		result["tool_calls"] = m.ToolCalls
	}
	return result
}

func NewSystemMessage(content string, id *string) Message {
	if id == nil {
		newId := uuid.New().String()
		id = &newId
	}
	contentBlocks, err := NewContentBlocks([]any{content})
	if err != nil {
		// If there's an error creating content blocks, fallback to using the original string as a text content block
		contentBlocks = &[]ContentBlock{NewTextContentBlock(content)}
	}
	return Message{
		Id:      *id,
		Role:    "system",
		Content: *contentBlocks,
	}
}

func NewHumanMessage(content string, id *string) Message {
	contentBlocks, err := NewContentBlocks([]any{content})
	if err != nil {
		// If there's an error creating content blocks, fallback to using the original string as a text content block
		contentBlocks = &[]ContentBlock{NewTextContentBlock(content)}
	}
	if id == nil {
		newId := uuid.New().String()
		id = &newId
	}
	return Message{
		Id:      *id,
		Role:    "user",
		Content: *contentBlocks,
	}
}

func NewMultiModalHumanMessage(contentBlocks []ContentBlock, id *string) Message {
	if id == nil {
		newId := uuid.New().String()
		id = &newId
	}
	return Message{
		Id:      *id,
		Role:    "user",
		Content: contentBlocks,
	}
}

func NewHumanMessageWithImage(content *string, imageUrl *string, id *string) Message {
	if id == nil {
		newId := uuid.New().String()
		id = &newId
	}
	if content == nil {
		content = new(string)
	}
	textContentBlock := NewTextContentBlock(*content)
	if imageUrl == nil || strings.TrimSpace(*imageUrl) == "" {
		// If imageUrl is nil or empty, return a message with just the text content block
		return Message{
			Id:      *id,
			Role:    "user",
			Content: []ContentBlock{textContentBlock},
		}
	}
	imageContentBlock := NewImageUrlContentBlock(*imageUrl)
	contentBlocks := []ContentBlock{textContentBlock, imageContentBlock}
	return Message{
		Id:      *id,
		Role:    "user",
		Content: contentBlocks,
	}
}

func NewAssistantMessage(content string, id *string, toolCalls *[]ToolCall, usage *TokenUsage) Message {
	if id == nil {
		newId := uuid.New().String()
		id = &newId
	}
	contentBlocks, err := NewContentBlocks([]any{content})
	if err != nil {
		// If there's an error creating content blocks, fallback to using the original string as a text content block
		contentBlocks = &[]ContentBlock{NewTextContentBlock(content)}
	}
	toolCallsValue := []ToolCall{}
	if toolCalls != nil {
		toolCallsValue = *toolCalls
	}
	extra := make(map[string]any)
	if usage != nil {
		extra["token_usage"] = usage
	}
	return Message{
		Id:        *id,
		Role:      "assistant",
		Content:   *contentBlocks,
		ToolCalls: toolCallsValue,
		Extra:     &extra,
	}
}

func NewToolMessage(content string, id *string, toolId *string) Message {
	if id == nil {
		newId := uuid.New().String()
		id = &newId
	}
	contentBlocks, err := NewContentBlocks([]any{content})
	if err != nil {
		// If there's an error creating content blocks, fallback to using the original string as a text content block
		contentBlocks = &[]ContentBlock{NewTextContentBlock(content)}
	}
	extra := make(map[string]any)
	if toolId != nil {
		extra["tool_id"] = *toolId
	}

	return Message{
		Id:      *id,
		Role:    "tool",
		Content: *contentBlocks,
		Extra:   &extra,
	}
}
