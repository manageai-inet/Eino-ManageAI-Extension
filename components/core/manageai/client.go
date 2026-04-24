package manageai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const defaultTimeout = 30 * time.Second
const defaultRetryAttempt = 3
const defaultRetryDelay = 1 * time.Second

var ApiV1PathMap = map[string]string{
	"health":          "/api/v1/health",
	"version":         "/api/v1/version",
	"models":          "/v1/models",
	"validate":        "/v1/validate-token",
	"limits":          "/v1/limits",
	"usage":           "/v1/usage",
	"chat-completion": "/v1/chat/completions",
	"embedding":       "/v1/embeddings",
	"rerank":          "/v1/rerank",
	"tokenize":        "/v1/tokenize",
	"detokenize":      "/v1/detokenize",
}

type Client struct {
	HTTPClient   *http.Client
	Model        *string       `json:"model"`
	BaseURL      string        `json:"base_url"`
	APIKey       string        `json:"api_key"`
	Timeout      time.Duration `json:"timeout"`
	RetryAttempt int           `json:"retry_attempt"`
	RetryDelay   time.Duration `json:"retry_delay"`
}

func NewClient(model *string, apiKey string) (*Client, error) {
	baseUrl := os.Getenv("MAI_BASE_URL")
	if baseUrl == "" {
		return nil, fmt.Errorf("`MAI_BASE_URL` environment variable is not set")
	}
	httpClient := &http.Client{
		Timeout: defaultTimeout,
	}
	if model != nil {
		redirectHelper := GetRedirectHelper()
		redirectedModel, err := redirectHelper.GetModelId(model, false)
		if err != nil {
			return nil, err
		}
		model = redirectedModel
	}
	return &Client{
		HTTPClient:   httpClient,
		Model:        model,
		BaseURL:      baseUrl,
		APIKey:       apiKey,
		Timeout:      defaultTimeout,
		RetryAttempt: defaultRetryAttempt,
		RetryDelay:   defaultRetryDelay,
	}, nil
}

// GetDefaultClient returns a new client with default configuration
// API Key and Model ID are read from environment variables
func GetDefaultClient() (*Client, error) {
	apiKey := os.Getenv("MAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("`MAI_API_KEY` environment variable is not set")
	}
	model := os.Getenv("MAI_MODEL_ID")
	if model == "" {
		return nil, fmt.Errorf("`MAI_MODEL_ID` environment variable is not set")
	}
	return NewClient(&model, apiKey)
}

func (c *Client) WithModel(model *string) (*Client, error) {
	if model != nil {
		redirectHelper := GetRedirectHelper()
		redirectedModel, err := redirectHelper.GetModelId(model, false)
		if err != nil {
			return nil, err
		}
		model = redirectedModel
	}
	c.Model = model
	return c, nil
}

func (c *Client) WithTimeout(timeout time.Duration) (*Client, error) {
	c.Timeout = timeout
	c.HTTPClient.Timeout = timeout
	return c, nil
}

func (c *Client) WithRetryAttempt(retryAttempt int) (*Client, error) {
	c.RetryAttempt = retryAttempt
	return c, nil
}

func (c *Client) WithRetryDelay(retryDelay time.Duration) (*Client, error) {
	c.RetryDelay = retryDelay
	return c, nil
}

func (c *Client) Close() error {
	c.HTTPClient.CloseIdleConnections()
	return nil
}

func (c *Client) NewRequest(ctx context.Context, method string, path string, payload []byte, headers map[string]string) (*http.Request, error) {
	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewBuffer(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

func (c *Client) doRetryableRequest(ctx context.Context, method, path string, payload []byte, headers map[string]string) (*http.Response, error) {
	var res *http.Response
	var err error

	attempts := c.RetryAttempt
	if attempts <= 0 {
		attempts = 1
	}

	for i := 1; i <= attempts; i++ {
		req, reqErr := c.NewRequest(ctx, method, path, payload, headers)
		if reqErr != nil {
			return nil, reqErr
		}

		res, err = c.HTTPClient.Do(req)

		// Retry เฉพาะ Network Error หรือ 5xx Server Error
		if err != nil || res.StatusCode >= 500 {
			if i == attempts {
				break
			}

			// เคลียร์สาย Connection
			if res != nil && res.Body != nil {
				io.Copy(io.Discard, res.Body)
				res.Body.Close()
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err() // ออกทันทีถ้า Context โดน Cancel/Timeout
			case <-time.After(c.RetryDelay):
				continue
			}
		} else {
			break
		}
	}

	return res, err
}

func (c *Client) raiseOnStatus(res *http.Response) error {
	if res.StatusCode >= 400 {
		var respContent string
		bodyBytes, err := io.ReadAll(res.Body)
		if err == nil {
			respContent = string(bodyBytes)
		} else {
			respContent = fmt.Sprintf("failed to read response body: %v", err)
		}
		return &APIError{
			Code:    res.StatusCode,
			Message: "API request failed",
			Details: fmt.Sprintf("[status code: %d]: %s", res.StatusCode, respContent),
		}
	}
	return nil
}

// ============================================================================
// Health
// ============================================================================
func (c *Client) RequestHealth(ctx context.Context) (*http.Response, error) {
	return c.doRetryableRequest(ctx, "GET", ApiV1PathMap["health"], nil, nil)
}

func (c *Client) HealthWithContext(ctx context.Context) (bool, error) {
	res, err := c.RequestHealth(ctx)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	return res.StatusCode == http.StatusOK, nil
}

func (c *Client) Health() (bool, error) {
	return c.HealthWithContext(context.Background())
}

// ============================================================================
// Version
// ============================================================================
func (c *Client) RequestVersion(ctx context.Context) (*http.Response, error) {
	return c.doRetryableRequest(ctx, "GET", ApiV1PathMap["version"], nil, nil)
}

func (c *Client) VersionWithContext(ctx context.Context) (string, error) {
	res, err := c.RequestVersion(ctx)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if err := c.raiseOnStatus(res); err != nil {
		return "", err
	}
	version, err := NewGetVersionResponse(res)
	if err != nil {
		return "", err
	}
	return version.Data, nil
}

func (c *Client) Version() (string, error) {
	return c.VersionWithContext(context.Background())
}

// ============================================================================
// Models
// ============================================================================
func (c *Client) RequestListModels(ctx context.Context) (*http.Response, error) {
	return c.doRetryableRequest(ctx, "GET", ApiV1PathMap["models"], nil, nil)
}

func (c *Client) ListModelsWithContext(ctx context.Context) (*[]ModelInfo, error) {
	res, err := c.RequestListModels(ctx)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := c.raiseOnStatus(res); err != nil {
		return nil, err
	}
	models, err := NewGetListModelsResponse(res)
	if err != nil {
		return nil, err
	}
	return &models.Data, nil
}

func (c *Client) ListModels() (*[]ModelInfo, error) {
	return c.ListModelsWithContext(context.Background())
}

// ============================================================================
// Validate Token
// ============================================================================
func (c *Client) RequestValidateToken(ctx context.Context) (*http.Response, error) {
	return c.doRetryableRequest(ctx, "GET", ApiV1PathMap["validate"], nil, nil)
}

func (c *Client) ValidateTokenWithContext(ctx context.Context) (*TokenValidationInfo, error) {
	res, err := c.RequestValidateToken(ctx)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := c.raiseOnStatus(res); err != nil {
		return nil, err
	}
	validateToken, err := NewValidateTokenResponse(res)
	if err != nil {
		return nil, err
	}
	return &validateToken.Data, nil
}

func (c *Client) ValidateToken() (*TokenValidationInfo, error) {
	return c.ValidateTokenWithContext(context.Background())
}

// ============================================================================
// Rate Limits
// ============================================================================
func (c *Client) RequestInspectRateLimits(ctx context.Context) (*http.Response, error) {
	return c.doRetryableRequest(ctx, "GET", ApiV1PathMap["limits"], nil, nil)
}

func (c *Client) InspectRateLimitsWithContext(ctx context.Context) (*RateLimitsInfo, error) {
	res, err := c.RequestInspectRateLimits(ctx)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := c.raiseOnStatus(res); err != nil {
		return nil, err
	}
	rateLimits, err := NewGetRateLimitsResponse(res)
	if err != nil {
		return nil, err
	}
	return &rateLimits.Data, nil
}

func (c *Client) InspectRateLimits() (*RateLimitsInfo, error) {
	return c.InspectRateLimitsWithContext(context.Background())
}

// ============================================================================
// Tokens Usage
// ============================================================================
func (c *Client) RequestInspectTokensUsage(ctx context.Context) (*http.Response, error) {
	return c.doRetryableRequest(ctx, "GET", ApiV1PathMap["usage"], nil, nil)
}

func (c *Client) InspectTokensUsageWithContext(ctx context.Context) (*TokensUsageInfo, error) {
	res, err := c.RequestInspectTokensUsage(ctx)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := c.raiseOnStatus(res); err != nil {
		return nil, err
	}
	tokensUsage, err := NewGetTokensUsageResponse(res)
	if err != nil {
		return nil, err
	}
	return &tokensUsage.Data, nil
}

func (c *Client) InspectTokensUsage() (*TokensUsageInfo, error) {
	return c.InspectTokensUsageWithContext(context.Background())
}

// ============================================================================
// Chat Completion
// ============================================================================
func (c *Client) RequestChatCompletion(ctx context.Context, messages []any, options *ChatCompletionOptions) (*http.Response, error) {
	if options != nil {
		model := options.Model
		if model == nil {
			model = c.Model
		}
		if model != nil {
			redirectHelper := GetRedirectHelper()
			ifMultiTools := options.Tools != nil && len(*options.Tools) > 1
			redirectedModel, err := redirectHelper.GetModelId(model, ifMultiTools)
			if err != nil {
				return nil, err
			}
			options.Model = redirectedModel
		} else {
			return nil, fmt.Errorf("model id is required in options or client configuration for chat completion")
		}
	}

	chatCompletion, err := CreateChatCompletion(messages, options)
	if err != nil {
		return nil, err
	}
	payload, err := chatCompletion.Marshal()
	if err != nil {
		return nil, err
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return c.doRetryableRequest(ctx, "POST", ApiV1PathMap["chat-completion"], bodyBytes, nil)
}

func (c *Client) ChatCompletionWithContext(ctx context.Context, messages []any, options *ChatCompletionOptions) (*ChatCompletionResponse, error) {
	res, err := c.RequestChatCompletion(ctx, messages, options)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := c.raiseOnStatus(res); err != nil {
		return nil, err
	}

	chatCompletionResponse, err := NewChatCompletionResponse(*res)
	if err != nil {
		return nil, err
	}
	return chatCompletionResponse, nil
}

func (c *Client) ChatCompletion(messages []any, options *ChatCompletionOptions) (*ChatCompletionResponse, error) {
	return c.ChatCompletionWithContext(context.Background(), messages, options)
}

// ============================================================================
// Embedding
// ============================================================================
func (c *Client) RequestEmbedding(ctx context.Context, input []string, options *EmbeddingOptions) (*http.Response, error) {
	if options == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	model := options.Model
	if model == nil {
		model = c.Model
	}
	if model != nil {
		redirectHelper := GetRedirectHelper()
		redirectedModel, err := redirectHelper.GetModelId(model, false)
		if err != nil {
			return nil, err
		}
		options.Model = redirectedModel
	} else {
		return nil, fmt.Errorf("modelId is required in options or client configuration for embedding")
	}

	embeddingRequest, err := CreateEmbeddingRequest(input, options)
	if err != nil {
		return nil, err
	}
	payload, err := embeddingRequest.Marshal()
	if err != nil {
		return nil, err
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return c.doRetryableRequest(ctx, "POST", ApiV1PathMap["embedding"], bodyBytes, nil)
}

func (c *Client) EmbeddingWithContext(ctx context.Context, input []string, options *EmbeddingOptions) (*EmbeddingResponse, error) {
	res, err := c.RequestEmbedding(ctx, input, options)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := c.raiseOnStatus(res); err != nil {
		return nil, err
	}

	embeddingResponse, err := NewEmbeddingResponse(res)
	if err != nil {
		return nil, err
	}
	return embeddingResponse, nil
}

func (c *Client) Embedding(input []string, options *EmbeddingOptions) (*EmbeddingResponse, error) {
	return c.EmbeddingWithContext(context.Background(), input, options)
}

// ============================================================================
// Reranker
// ============================================================================
func (c *Client) RequestRerank(ctx context.Context, query string, documents []string, options *RerankOptions) (*http.Response, error) {
	if options == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	model := options.Model
	if model == nil {
		model = c.Model
	}
	if model != nil {
		redirectHelper := GetRedirectHelper()
		redirectedModel, err := redirectHelper.GetModelId(model, false)
		if err != nil {
			return nil, err
		}
		options.Model = redirectedModel
	} else {
		return nil, fmt.Errorf("model id is required in options or client configuration for rerank")
	}

	rerankRequest, err := CreateRerankRequest(query, documents, options)
	if err != nil {
		return nil, err
	}
	payload, err := rerankRequest.Marshal()
	if err != nil {
		return nil, err
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return c.doRetryableRequest(ctx, "POST", ApiV1PathMap["rerank"], bodyBytes, nil)
}

func (c *Client) RerankWithContext(ctx context.Context, query string, documents []string, options *RerankOptions) (*RerankResponse, error) {
	res, err := c.RequestRerank(ctx, query, documents, options)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := c.raiseOnStatus(res); err != nil {
		return nil, err
	}

	rerankResponse, err := NewRerankResponse(res)
	if err != nil {
		return nil, err
	}
	return rerankResponse, nil
}

func (c *Client) Rerank(query string, documents []string, options *RerankOptions) (*RerankResponse, error) {
	return c.RerankWithContext(context.Background(), query, documents, options)
}

// ============================================================================
// Tokenize
// ============================================================================
func (c *Client) RequestTokenize(ctx context.Context, text string, optionsModel *string) (*http.Response, error) {
	model := c.Model
	if optionsModel != nil {
		model = optionsModel
	}
	if model == nil {
		return nil, fmt.Errorf("model id is required for tokenize")
	}

	redirectHelper := GetRedirectHelper()
	redirectedModel, err := redirectHelper.GetModelId(model, false)
	if err == nil {
		model = redirectedModel
	}

	chatData, err := CreateChatCompletion([]any{text}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create messages: %v", err)
	}

	payload := TokenizeRequest{
		Model:    *model,
		Messages: chatData.Messages,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return c.doRetryableRequest(ctx, "POST", ApiV1PathMap["tokenize"], bodyBytes, nil)
}

func (c *Client) TokenizeWithContext(ctx context.Context, text string, optionsModel *string) (*TokenizeResponse, error) {
	res, err := c.RequestTokenize(ctx, text, optionsModel)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := c.raiseOnStatus(res); err != nil {
		return nil, err
	}

	return NewTokenizeResponse(res)
}

func (c *Client) Tokenize(text string, optionsModel *string) (*TokenizeResponse, error) {
	return c.TokenizeWithContext(context.Background(), text, optionsModel)
}

// ============================================================================
// Chat Completion Stream
// ============================================================================
func (c *Client) RequestChatCompletionStream(ctx context.Context, input []any, options *ChatCompletionOptions) (*http.Response, error) {
	chatData, err := CreateChatCompletion(input, options)
	if err != nil {
		return nil, err
	}

	model := c.Model
	redirectHelper := GetRedirectHelper()
	redirectedModel, err := redirectHelper.GetModelId(model, false)
	if err == nil {
		model = redirectedModel
	}

	payload := map[string]any{
		"messages": chatData.Messages,
		"model":    *model,
		"stream":   true,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Accept": "text/event-stream",
	}

	return c.doRetryableRequest(ctx, "POST", ApiV1PathMap["chat-completion"], bodyBytes, headers)
}

func (c *Client) ChatCompletionStream(ctx context.Context, input []any, options *ChatCompletionOptions) (*ChatCompletionStreamReader, error) {
	res, err := c.RequestChatCompletionStream(ctx, input, options)
	if err != nil {
		return nil, err
	}

	if err := c.raiseOnStatus(res); err != nil {
		res.Body.Close()
		return nil, err
	}

	return StreamChatCompletionResponse(res)
}

// ============================================================================
// Detokenize
// ============================================================================
func (c *Client) RequestDetokenize(ctx context.Context, tokens []int, options *DetokenizeOptions) (*http.Response, error) {
	if options == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	model := options.Model
	if model == nil {
		model = c.Model
	}
	if model != nil {
		redirectHelper := GetRedirectHelper()
		redirectedModel, err := redirectHelper.GetModelId(model, false)
		if err != nil {
			return nil, err
		}
		options.Model = redirectedModel
	} else {
		return nil, fmt.Errorf("modelId is required in options or client configuration for detokenize")
	}

	detokenizeRequest, err := CreateDetokenizeRequest(tokens, options)
	if err != nil {
		return nil, err
	}
	payload, err := detokenizeRequest.Marshal()
	if err != nil {
		return nil, err
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return c.doRetryableRequest(ctx, "POST", ApiV1PathMap["detokenize"], bodyBytes, nil)
}

func (c *Client) DetokenizeWithContext(ctx context.Context, tokens []int, options *DetokenizeOptions) (*DetokenizeResponse, error) {
	res, err := c.RequestDetokenize(ctx, tokens, options)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := c.raiseOnStatus(res); err != nil {
		return nil, err
	}

	detokenizeResponse, err := NewDetokenizeResponse(res)
	if err != nil {
		return nil, err
	}
	return detokenizeResponse, nil
}

func (c *Client) Detokenize(tokens []int, options *DetokenizeOptions) (*DetokenizeResponse, error) {
	return c.DetokenizeWithContext(context.Background(), tokens, options)
}
