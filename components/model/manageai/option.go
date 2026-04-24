package manageai

import (
	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/eino-contrib/jsonschema"
)

const (
	ResponseFormatJson = "json_object"
	ResponseFormatText = "text"
)

// CallOption represents configuration options for model calls
type manageAIOptions struct {
	ResponseFormat *string
	GuidedJson     *jsonschema.Schema
	GuidedRegex    *string
}

func WithResponseFormat(format string) model.Option {
	return model.WrapImplSpecificOptFn(func(opt *manageAIOptions) {
		if format != ResponseFormatJson && format != ResponseFormatText {
			panic("invalid response format, only support json_object and text")
		}
		opt.ResponseFormat = &format
	})
}

func WithGuidedJson(schema *jsonschema.Schema) model.Option {
	return model.WrapImplSpecificOptFn(func(opt *manageAIOptions) {
		opt.GuidedJson = schema
		responseFormatJson := ResponseFormatJson
		opt.ResponseFormat = &responseFormatJson
	})
}

func WithGuidedRegex(regex string) model.Option {
	return model.WrapImplSpecificOptFn(func(opt *manageAIOptions) {
		opt.GuidedRegex = &regex
		responseFormatText := ResponseFormatText
		opt.ResponseFormat = &responseFormatText
	})
}

// WithExtraFields is used to set extra body fields for the request.
func WithExtraFields(extraFields map[string]any) model.Option {
	return openai.WithExtraFields(extraFields)
}

// WithExtraHeader is used to set extra headers for the request.
func WithExtraHeader(header map[string]string) model.Option {
	return openai.WithExtraHeader(header)
}
