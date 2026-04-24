package manageai

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func getFirst(m map[string]any, keys ...string) (any, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v, true
		}
	}
	return nil, false
}

func decodeBase64Embedding(b64Data string) ([]float32, error) {
	// Step 1: Decode from Base64 to bytes
	data, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return nil, err
	}

	// Step 2: Unpack the bytes into float32s (4 bytes per float)
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("invalid base64 embedding data: length %d is not a multiple of 4", len(data))
	}
	floats := make([]float32, len(data)/4)
	for i := range floats {
		bits := binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
		floats[i] = math.Float32frombits(bits)
	}
	return floats, nil
}

type GetVersionResponse struct {
	Data string `json:"data"`
}

func NewGetVersionResponse(response *http.Response) (*GetVersionResponse, error) {
	var decoded_body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&decoded_body); err != nil {
		return nil, &ValidationError{
			Field:   "body",
			Message: err.Error(),
		}
	}
	if data, ok := decoded_body["data"]; ok {
		data_string, ok := data.(string)
		if !ok {
			return nil, &ValidationError{
				Field:   "data",
				Message: fmt.Sprintf("expected 'data' field to be string, got %T", data),
			}
		}
		return &GetVersionResponse{
			Data: data_string,
		}, nil
	}
	return nil, &ValidationError{
		Field:   "data",
		Message: "expected 'data' field",
	}
}

type GetListModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

func NewGetListModelsResponse(response *http.Response) (*GetListModelsResponse, error) {
	var decoded map[string]any
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, &ValidationError{
			Field:   "body",
			Message: err.Error(),
		}
	}
	if object, ok := decoded["object"]; ok {
		object_string, ok := object.(string)
		if !ok {
			return nil, &ValidationError{
				Field:   "object",
				Message: fmt.Sprintf("expected 'object' field to be string, got %T", object),
			}
		}
		if data, ok := decoded["data"]; ok {
			da, ok := data.([]any)
			if !ok {
				return nil, &ValidationError{
					Field:   "data",
					Message: fmt.Sprintf("expected 'data' field to be array, got %T", data),
				}
			}
			var models []ModelInfo
			for i, model := range da {
				model_map, ok := model.(map[string]any)
				if !ok {
					return nil, &ValidationError{
						Field:   "data",
						Message: fmt.Sprintf("invalid response, expected 'data' field to be array of objects, got %T", model),
					}
				}
				model_info, err := NewModelInfo(model_map)
				if err != nil {
					return nil, &ValidationError{
						Field:   fmt.Sprintf("data[%d]", i),
						Message: err.Error(),
					}
				}
				models = append(models, *model_info)
			}
			return &GetListModelsResponse{
				Object: object_string,
				Data:   models,
			}, nil
		}
		return nil, &ValidationError{
			Field:   "data",
			Message: "expected 'data' field",
		}
	}
	return nil, &ValidationError{
		Field:   "object",
		Message: "expected 'object' field",
	}
}

type ModelInfo struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
	Status  string `json:"status"`
}

func NewModelInfo(model map[string]any) (*ModelInfo, error) {
	typeString, ok := model["type"].(string)
	if !ok {
		return nil, &ValidationError{
			Field:   "type",
			Message: fmt.Sprintf("expected 'type' field to be string, got %T", model["type"]),
		}
	}
	idString, ok := model["id"].(string)
	if !ok {
		return nil, &ValidationError{
			Field:   "id",
			Message: fmt.Sprintf("expected 'id' field to be string, got %T", model["id"]),
		}
	}
	objectString, ok := model["object"].(string)
	if !ok {
		return nil, &ValidationError{
			Field:   "object",
			Message: fmt.Sprintf("expected 'object' field to be string, got %T", model["object"]),
		}
	}
	createdInt, ok := model["created"].(int64)
	if !ok {
		return nil, &ValidationError{
			Field:   "created",
			Message: fmt.Sprintf("expected 'created' field to be int64, got %T", model["created"]),
		}
	}
	ownedByString, ok := model["owned_by"].(string)
	if !ok {
		return nil, &ValidationError{
			Field:   "owned_by",
			Message: fmt.Sprintf("expected 'owned_by' field to be string, got %T", model["owned_by"]),
		}
	}
	statusString, ok := model["status"].(string)
	if !ok {
		return nil, &ValidationError{
			Field:   "status",
			Message: fmt.Sprintf("expected 'status' field to be string, got %T", model["status"]),
		}
	}
	return &ModelInfo{
		Type:    typeString,
		ID:      idString,
		Object:  objectString,
		Created: createdInt,
		OwnedBy: ownedByString,
		Status:  statusString,
	}, nil
}

type TokenValidationDataScope struct {
	Read  []string `json:"read"`
	Write []string `json:"write"`
	Admin []string `json:"admin"`
}

type TokenValidationInfo struct {
	Name           string                     `json:"name"`
	Valid          bool                       `json:"valid"`
	ExpiresAt      int64                      `json:"expires_at"`
	IpRestrictions string                     `json:"ip_restrictions"`
	Scopes         []TokenValidationDataScope `json:"scopes"`
}

func NewTokenValidationDataScope(scope map[string]any) (*TokenValidationDataScope, error) {
	if scope == nil {
		return &TokenValidationDataScope{
			Read:  []string{},
			Write: []string{},
			Admin: []string{},
		}, nil
	}
	readArray, ok := scope["read"].([]string)
	if !ok {
		readArray = []string{}
	}
	writeArray, ok := scope["write"].([]string)
	if !ok {
		writeArray = []string{}
	}
	adminArray, ok := scope["admin"].([]string)
	if !ok {
		adminArray = []string{}
	}
	return &TokenValidationDataScope{
		Read:  readArray,
		Write: writeArray,
		Admin: adminArray,
	}, nil
}

func NewTokenValidationInfo(info map[string]any) (*TokenValidationInfo, error) {
	nameString, ok := info["name"].(string)
	if !ok {
		return nil, &ValidationError{
			Field:   "name",
			Message: fmt.Sprintf("invalid response, expected 'name' field to be string, got %T", info["name"]),
		}
	}
	validBool, ok := info["valid"].(bool)
	if !ok {
		return nil, &ValidationError{
			Field:   "valid",
			Message: fmt.Sprintf("invalid response, expected 'valid' field to be bool, got %T", info["valid"]),
		}
	}
	expiresAtInt, ok := info["expires_at"].(int64)
	if !ok {
		return nil, &ValidationError{
			Field:   "expires_at",
			Message: fmt.Sprintf("invalid response, expected 'expires_at' field to be int64, got %T", info["expires_at"]),
		}
	}
	ipRestrictionsString, ok := info["ip_restrictions"].(string)
	if !ok {
		return nil, &ValidationError{
			Field:   "ip_restrictions",
			Message: fmt.Sprintf("invalid response, expected 'ip_restrictions' field to be string, got %T", info["ip_restrictions"]),
		}
	}
	scopesArray, ok := info["scopes"].([]any)
	if !ok {
		return nil, &ValidationError{
			Field:   "scopes",
			Message: fmt.Sprintf("invalid response, expected 'scopes' field to be array, got %T", info["scopes"]),
		}
	}
	var scopes []TokenValidationDataScope
	for i, scope := range scopesArray {
		scopeMap, ok := scope.(map[string]any)
		if !ok {
			return nil, &ValidationError{
				Field:   "scopes",
				Message: fmt.Sprintf("invalid response, expected 'scopes' field to be array of objects, got %T", scope),
			}
		}
		scopeInfo, err := NewTokenValidationDataScope(scopeMap)
		if err != nil {
			return nil, &ValidationError{
				Field:   fmt.Sprintf("scopes[%d]", i),
				Message: err.Error(),
			}
		}
		scopes = append(scopes, *scopeInfo)
	}
	return &TokenValidationInfo{
		Name:           nameString,
		Valid:          validBool,
		ExpiresAt:      expiresAtInt,
		IpRestrictions: ipRestrictionsString,
		Scopes:         scopes,
	}, nil
}

type ValidateTokenResponse struct {
	Data TokenValidationInfo `json:"data"`
}

func NewValidateTokenResponse(response *http.Response) (*ValidateTokenResponse, error) {
	var decodedBody map[string]any
	if err := json.NewDecoder(response.Body).Decode(&decodedBody); err != nil {
		return nil, &ValidationError{
			Field:   "body",
			Message: err.Error(),
		}
	}
	if data, ok := decodedBody["data"]; ok {
		dataMap, ok := data.(map[string]any)
		if !ok {
			return nil, &ValidationError{
				Field:   "data",
				Message: fmt.Sprintf("invalid response, expected 'data' field to be object, got %T", data),
			}
		}
		tokenValidationInfo, err := NewTokenValidationInfo(dataMap)
		if err != nil {
			return nil, &ValidationError{
				Field:   "data",
				Message: err.Error(),
			}
		}
		return &ValidateTokenResponse{
			Data: *tokenValidationInfo,
		}, nil
	}
	return nil, &ValidationError{
		Field:   "data",
		Message: "invalid response, expected 'data' field",
	}
}

type RateLimits struct {
	RequestsPerMinute int64 `json:"requests_per_minute"`
	RequestsPerHour   int64 `json:"requests_per_hour"`
	RequestsPerDay    int64 `json:"requests_per_day"`
	RequestsPerMonth  int64 `json:"requests_per_month"`
	StrictEnforcement bool  `json:"strict_enforcement"`
}

func NewRateLimits(rateLimitsMap map[string]any) (*RateLimits, error) {
	if rateLimitsMap == nil {
		return &RateLimits{
			RequestsPerMinute: 0,
			RequestsPerHour:   0,
			RequestsPerDay:    0,
			RequestsPerMonth:  0,
			StrictEnforcement: false,
		}, nil
	}
	requestsPerMinuteInt, ok := rateLimitsMap["requests_per_minute"].(int64)
	if !ok {
		requestsPerMinuteInt = 0
	}
	requestsPerHourInt, ok := rateLimitsMap["requests_per_hour"].(int64)
	if !ok {
		requestsPerHourInt = 0
	}
	requestsPerDayInt, ok := rateLimitsMap["requests_per_day"].(int64)
	if !ok {
		requestsPerDayInt = 0
	}
	requestsPerMonthInt, ok := rateLimitsMap["requests_per_month"].(int64)
	if !ok {
		requestsPerMonthInt = 0
	}
	strictEnforcementBool, ok := rateLimitsMap["strict_enforcement"].(bool)
	if !ok {
		strictEnforcementBool = false
	}
	return &RateLimits{
		RequestsPerMinute: requestsPerMinuteInt,
		RequestsPerHour:   requestsPerHourInt,
		RequestsPerDay:    requestsPerDayInt,
		RequestsPerMonth:  requestsPerMonthInt,
		StrictEnforcement: strictEnforcementBool,
	}, nil
}

type RateLimitsInfo struct {
	Name             string     `json:"name"`
	Valid            bool       `json:"valid"`
	ExpiresAt        int64      `json:"expires_at"`
	IpRestrictions   string     `json:"ip_restrictions"`
	RateLimits       RateLimits `json:"rate_limits"`
	CalculatedLimits RateLimits `json:"calculated_limits"`
}

func NewRateLimitsInfo(infoMap map[string]any) (*RateLimitsInfo, error) {
	nameString, ok := infoMap["name"].(string)
	if !ok {
		return nil, &ValidationError{
			Field:   "name",
			Message: fmt.Sprintf("invalid response, expected 'name' field to be string, got %T", infoMap["name"]),
		}
	}
	validBool, ok := infoMap["valid"].(bool)
	if !ok {
		return nil, &ValidationError{
			Field:   "valid",
			Message: fmt.Sprintf("invalid response, expected 'valid' field to be bool, got %T", infoMap["valid"]),
		}
	}
	expiresAtInt, ok := infoMap["expires_at"].(int64)
	if !ok {
		return nil, &ValidationError{
			Field:   "expires_at",
			Message: fmt.Sprintf("invalid response, expected 'expires_at' field to be int64, got %T", infoMap["expires_at"]),
		}
	}
	ipRestrictionsString, ok := infoMap["ip_restrictions"].(string)
	if !ok {
		return nil, &ValidationError{
			Field:   "ip_restrictions",
			Message: fmt.Sprintf("invalid response, expected 'ip_restrictions' field to be string, got %T", infoMap["ip_restrictions"]),
		}
	}
	rateLimitsMap, ok := infoMap["rate_limits"].(map[string]any)
	if !ok {
		return nil, &ValidationError{
			Field:   "rate_limits",
			Message: fmt.Sprintf("invalid response, expected 'rate_limits' field to be object, got %T", infoMap["rate_limits"]),
		}
	}
	calculatedLimitsMap, ok := infoMap["calculated_limits"].(map[string]any)
	if !ok {
		return nil, &ValidationError{
			Field:   "calculated_limits",
			Message: fmt.Sprintf("invalid response, expected 'calculated_limits' field to be object, got %T", infoMap["calculated_limits"]),
		}
	}

	rateLimits, err := NewRateLimits(rateLimitsMap)
	if err != nil {
		return nil, &ValidationError{
			Field:   "rate_limits",
			Message: "failed to get rate limits",
		}
	}
	calculatedLimits, err := NewRateLimits(calculatedLimitsMap)
	if err != nil {
		return nil, &ValidationError{
			Field:   "calculated_limits",
			Message: "failed to get calculated limits",
		}
	}
	return &RateLimitsInfo{
		Name:             nameString,
		Valid:            validBool,
		ExpiresAt:        expiresAtInt,
		IpRestrictions:   ipRestrictionsString,
		RateLimits:       *rateLimits,
		CalculatedLimits: *calculatedLimits,
	}, nil
}

type GetRateLimitsResponse struct {
	Data RateLimitsInfo `json:"data"`
}

func NewGetRateLimitsResponse(response *http.Response) (*GetRateLimitsResponse, error) {
	var decodedBody map[string]any
	if err := json.NewDecoder(response.Body).Decode(&decodedBody); err != nil {
		return nil, &ValidationError{
			Field:   "body",
			Message: err.Error(),
		}
	}
	if data, ok := decodedBody["data"]; ok {
		dataMap, ok := data.(map[string]any)
		if !ok {
			return nil, &ValidationError{
				Field:   "data",
				Message: fmt.Sprintf("invalid response, expected 'data' field to be object, got %T", data),
			}
		}
		rateLimitsInfo, err := NewRateLimitsInfo(dataMap)
		if err != nil {
			return nil, &ValidationError{
				Field:   "data",
				Message: err.Error(),
			}
		}
		return &GetRateLimitsResponse{
			Data: *rateLimitsInfo,
		}, nil
	}
	return nil, &ValidationError{
		Field:   "data",
		Message: "invalid response, expected 'data' field",
	}
}

type TokenUsage struct {
	PromptTokens        int64            `json:"prompt_tokens"`
	TotalTokens         int64            `json:"total_tokens"`
	CompletionTokens    int64            `json:"completion_tokens"`
	PromptTokensDetails []map[string]any `json:"prompt_tokens_details"`
}

func NewTokenUsage(tokenUsageMap map[string]any) (*TokenUsage, error) {
	var promptTokensInt int64
	promptTokens, ok := getFirst(tokenUsageMap, "prompt_tokens", "input_tokens")
	if !ok {
		promptTokensInt = 0
	} else {
		// assert to float64 and then convert to int64.
		if floatVal, isFloat := promptTokens.(float64); isFloat {
			promptTokensInt = int64(floatVal)
		}
	}
	// else {
	// 	promptTokensInt = promptTokens.(int64)
	// }

	var completionTokensInt int64
	completionTokens, ok := getFirst(tokenUsageMap, "completion_tokens", "output_tokens")
	if !ok {
		completionTokensInt = 0
	} else {
		// assert to float64 and then convert to int64.
		if floatval, isFloat := completionTokens.(float64); isFloat {
			completionTokensInt = int64(floatval)
		}
	}
	var totalTokensInt int64
	// assert to float64 and then convert to int64.
	if f, ok := tokenUsageMap["total_tokens"].(float64); ok {
		totalTokensInt = int64(f)
	} else {
		totalTokensInt = promptTokensInt + completionTokensInt
	}
	var promptTokensDetails []map[string]any
	// assert to []any and then assert each item to map[string]any
	if rawDetails, ok := tokenUsageMap["prompt_tokens_details"].([]any); ok {
		for _, rawItem := range rawDetails {
			if detailMap, isMap := rawItem.(map[string]any); isMap {
				promptTokensDetails = append(promptTokensDetails, detailMap)
			}
		}
	}
	if promptTokensDetails == nil {
		promptTokensDetails = []map[string]any{}
	}
	return &TokenUsage{
		PromptTokens:        promptTokensInt,
		TotalTokens:         totalTokensInt,
		CompletionTokens:    completionTokensInt,
		PromptTokensDetails: promptTokensDetails,
	}, nil
}

type ModelTokensUsage struct {
	TotalTokens  int64 `json:"total_tokens"`
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	ImageCount   int64 `json:"image_count"`
	PageCount    int64 `json:"page_count"`
	RequestCount int64 `json:"request_count"`
}

func NewModelTokensUsage(modelTokensUsageMap map[string]any) (*ModelTokensUsage, error) {
	totalTokensInt, ok := modelTokensUsageMap["total_tokens"].(int64)
	if !ok {
		return nil, &ValidationError{
			Field:   "total_tokens",
			Message: fmt.Sprintf("invalid response, expected 'total_tokens' field to be int64, got %T", modelTokensUsageMap["total_tokens"]),
		}
	}
	inputTokensInt, ok := modelTokensUsageMap["input_tokens"].(int64)
	if !ok {
		return nil, &ValidationError{
			Field:   "input_tokens",
			Message: fmt.Sprintf("invalid response, expected 'input_tokens' field to be int64, got %T", modelTokensUsageMap["input_tokens"]),
		}
	}
	outputTokensInt, ok := modelTokensUsageMap["output_tokens"].(int64)
	if !ok {
		return nil, &ValidationError{
			Field:   "output_tokens",
			Message: fmt.Sprintf("invalid response, expected 'output_tokens' field to be int64, got %T", modelTokensUsageMap["output_tokens"]),
		}
	}
	imageCountInt, ok := modelTokensUsageMap["image_count"].(int64)
	if !ok {
		return nil, &ValidationError{
			Field:   "image_count",
			Message: fmt.Sprintf("invalid response, expected 'image_count' field to be int64, got %T", modelTokensUsageMap["image_count"]),
		}
	}
	pageCountInt, ok := modelTokensUsageMap["page_count"].(int64)
	if !ok {
		return nil, &ValidationError{
			Field:   "page_count",
			Message: fmt.Sprintf("invalid response, expected 'page_count' field to be int64, got %T", modelTokensUsageMap["page_count"]),
		}
	}
	requestCountInt, ok := modelTokensUsageMap["request_count"].(int64)
	if !ok {
		return nil, &ValidationError{
			Field:   "request_count",
			Message: fmt.Sprintf("invalid response, expected 'request_count' field to be int64, got %T", modelTokensUsageMap["request_count"]),
		}
	}
	return &ModelTokensUsage{
		TotalTokens:  totalTokensInt,
		InputTokens:  inputTokensInt,
		OutputTokens: outputTokensInt,
		ImageCount:   imageCountInt,
		PageCount:    pageCountInt,
		RequestCount: requestCountInt,
	}, nil
}

type DailyTokensUsage struct {
	Date    string                      `json:"date"`
	ByModel map[string]ModelTokensUsage `json:"by_model"`
}

func NewDailyTokensUsage(dailyTokensUsageMap map[string]any) (*DailyTokensUsage, error) {
	dateString, ok := dailyTokensUsageMap["date"].(string)
	if !ok {
		return nil, &ValidationError{
			Field:   "date",
			Message: fmt.Sprintf("invalid response, expected 'date' field to be string, got %T", dailyTokensUsageMap["date"]),
		}
	}
	byModelMap, ok := dailyTokensUsageMap["by_model"].(map[string]any)
	if !ok {
		return nil, &ValidationError{
			Field:   "by_model",
			Message: fmt.Sprintf("invalid response, expected 'by_model' field to be object, got %T", dailyTokensUsageMap["by_model"]),
		}
	}
	byModel := make(map[string]ModelTokensUsage)
	for modelName, modelTokensUsageMap := range byModelMap {
		modelTokensUsage, err := NewModelTokensUsage(modelTokensUsageMap.(map[string]any))
		if err != nil {
			return nil, &ValidationError{
				Field:   fmt.Sprintf("by_model[%s]", modelName),
				Message: err.Error(),
			}
		}
		byModel[modelName] = *modelTokensUsage
	}
	return &DailyTokensUsage{
		Date:    dateString,
		ByModel: byModel,
	}, nil
}

type TokensUsageInfo struct {
	DataPerDay []DailyTokensUsage          `json:"data_perday"`
	DataSum    map[string]ModelTokensUsage `json:"data_sum"`
}

func NewTokensUsageInfo(tokensUsageInfoMap map[string]any) (*TokensUsageInfo, error) {
	dataPerDaySlice, ok := tokensUsageInfoMap["data_perday"].([]map[string]any)
	if !ok {
		return nil, &ValidationError{
			Field:   "data_perday",
			Message: fmt.Sprintf("invalid response, expected 'data_perday' field to be array, got %T", tokensUsageInfoMap["data_perday"]),
		}
	}
	dataPerDay := make([]DailyTokensUsage, len(dataPerDaySlice))
	for i, dailyTokensUsageMap := range dataPerDaySlice {
		dailyTokensUsage, err := NewDailyTokensUsage(dailyTokensUsageMap)
		if err != nil {
			return nil, &ValidationError{
				Field:   fmt.Sprintf("data_perday[%d]", i),
				Message: err.Error(),
			}
		}
		dataPerDay[i] = *dailyTokensUsage
	}
	dataSumMap, ok := tokensUsageInfoMap["data_sum"].(map[string]any)
	if !ok {
		return nil, &ValidationError{
			Field:   "data_sum",
			Message: fmt.Sprintf("invalid response, expected 'data_sum' field to be object, got %T", tokensUsageInfoMap["data_sum"]),
		}
	}
	dataSum := make(map[string]ModelTokensUsage)
	for modelName, modelTokensUsageMap := range dataSumMap {
		modelTokensUsage, err := NewModelTokensUsage(modelTokensUsageMap.(map[string]any))
		if err != nil {
			return nil, &ValidationError{
				Field:   fmt.Sprintf("data_sum[%s]", modelName),
				Message: err.Error(),
			}
		}
		dataSum[modelName] = *modelTokensUsage
	}
	return &TokensUsageInfo{
		DataPerDay: dataPerDay,
		DataSum:    dataSum,
	}, nil
}

type GetTokensUsageResponse struct {
	Data TokensUsageInfo `json:"data"`
}

func NewGetTokensUsageResponse(response *http.Response) (*GetTokensUsageResponse, error) {
	var decodedBody map[string]any
	if err := json.NewDecoder(response.Body).Decode(&decodedBody); err != nil {
		return nil, &ValidationError{
			Field:   "body",
			Message: err.Error(),
		}
	}
	if data, ok := decodedBody["data"]; ok {
		dataMap, ok := data.(map[string]any)
		if !ok {
			return nil, &ValidationError{
				Field:   "data",
				Message: fmt.Sprintf("invalid response, expected 'data' field to be object, got %T", data),
			}
		}
		tokensUsageInfo, err := NewTokensUsageInfo(dataMap)
		if err != nil {
			return nil, &ValidationError{
				Field:   "data",
				Message: err.Error(),
			}
		}
		return &GetTokensUsageResponse{
			Data: *tokensUsageInfo,
		}, nil
	}
	return nil, &ValidationError{
		Field:   "body",
		Message: "invalid response, expected 'data' field",
	}
}

type EmbeddingResponse struct {
	Object string          `json:"object"`
	Model  string          `json:"model"`
	Data   []EmbeddingData `json:"data"`

	Usage           TokenUsage      `json:"usage"`
	GatewayMetadata GatewayMetadata `json:"gateway_metadata"`
	// extract from headers
	ResponseHeaders ServiceHeaders
}

func NewEmbeddingResponse(response *http.Response) (*EmbeddingResponse, error) {
	var decodedBody map[string]any
	if err := json.NewDecoder(response.Body).Decode(&decodedBody); err != nil {
		return nil, &ValidationError{
			Field:   "body",
			Message: err.Error(),
		}
	}
	object, ok := decodedBody["object"].(string)
	if !ok {
		object = "list"
	}
	model, ok := decodedBody["model"].(string)
	if !ok {
		return nil, &ValidationError{
			Field:   "model",
			Message: fmt.Sprintf("invalid response, expected 'model' field to be string, got %T", decodedBody["model"]),
		}
	}
	var dataSlice []EmbeddingData
	dataSliceInterface, ok := decodedBody["data"]
	if !ok {
		return nil, &ValidationError{
			Field:   "data",
			Message: "invalid response, expected 'data' field",
		}
	} else {
		// assert to []any and then assert each item to map[string]any
		dataSliceInterfaceSlice, ok := dataSliceInterface.([]any)
		if !ok {
			return nil, &ValidationError{
				Field:   "data",
				Message: fmt.Sprintf("invalid response, expected 'data' field to be array, got %T", dataSliceInterface),
			}
		}
		dataSlice = make([]EmbeddingData, len(dataSliceInterfaceSlice))
		for i, rawItem := range dataSliceInterfaceSlice {
			dataSliceInterfaceMap, ok := rawItem.(map[string]any)
			if !ok {
				return nil, &ValidationError{
					Field:   "data",
					Message: fmt.Sprintf("invalid response: expected 'data[%d]' to be an object, got %T", i, rawItem),
				}
			}
			embeddingData, err := NewEmbeddingData(dataSliceInterfaceMap)
			if err != nil {
				return nil, &ValidationError{
					Field:   "data",
					Message: err.Error(),
				}
			}
			dataSlice[i] = *embeddingData
		}
	}

	var tokenUsage TokenUsage
	usage, ok := decodedBody["usage"].(map[string]any)
	if !ok {
		tokenUsage = TokenUsage{
			PromptTokens:        0,
			TotalTokens:         0,
			CompletionTokens:    0,
			PromptTokensDetails: []map[string]any{},
		}
	} else {
		tu, err := NewTokenUsage(usage)
		if err != nil {
			return nil, &ValidationError{
				Field:   "usage",
				Message: err.Error(),
			}
		}
		tokenUsage = *tu
	}

	var gatewayMetadata GatewayMetadata
	metadata, ok := decodedBody["gateway_metadata"].(map[string]any)
	if !ok {
		gatewayMetadata = GatewayMetadata{}
	} else {
		gatewayMetadata = NewGatewayMetadata(metadata)
	}

	serviceHeaders := NewServiceHeaders(response.Header)

	return &EmbeddingResponse{
		Object:          object,
		Model:           model,
		Data:            dataSlice,
		Usage:           tokenUsage,
		GatewayMetadata: gatewayMetadata,
		ResponseHeaders: serviceHeaders,
	}, nil
}

type EmbeddingData struct {
	Index     int       `json:"index"`
	Text      *string   `json:"text"`
	Embedding []float64 `json:"embedding"`
}

func NewEmbeddingData(embeddingData map[string]any) (*EmbeddingData, error) {
	indexFloat, ok := embeddingData["index"].(float64)
	if !ok {
		return nil,
			&ValidationError{
				Field:   "index",
				Message: fmt.Sprintf("invalid response, expected 'index' field to be int, got %T", embeddingData["index"]),
			}
	}
	indexInt := int(indexFloat) // convert float to int
	var textString *string = nil
	textStringInterface, ok := embeddingData["text"]
	if ok {
		s, ok := textStringInterface.(string)
		if ok {
			textString = &s
		}
	}
	var embeddingArray []float64
	embeddingArrayInterface, ok := embeddingData["embedding"]
	if !ok {
		return nil, &ValidationError{
			Field:   "embedding",
			Message: "invalid response, expected 'embedding' field",
		}
	} else {
		// if emembeddingArray is string of base64, decode it
		if s, ok := embeddingArrayInterface.(string); ok {
			decoded, err := decodeBase64Embedding(s)
			if err != nil {
				return nil, &ValidationError{
					Field:   "embedding",
					Message: err.Error(),
				}
			}
			decodedFloat64 := make([]float64, len(decoded))
			for i, v := range decoded {
				decodedFloat64[i] = float64(v)
			}
			embeddingArray = decodedFloat64
			// assert to []any and then convert each item to float
		} else if a, ok := embeddingArrayInterface.([]any); ok {
			embeddingArray = make([]float64, len(a))
			for i, v := range a {
				if floatVal, ok := v.(float64); ok {
					embeddingArray[i] = floatVal
				} else {
					return nil, &ValidationError{
						Field:   "embedding",
						Message: fmt.Sprintf("invalid value in embedding array at index %d, expected float64, got %T", i, v),
					}
				}
			}
		} else {
			return nil, &ValidationError{
				Field:   "embedding",
				Message: fmt.Sprintf("invalid response, expected 'embedding' field to be array of float64 or string of base64, got %T", embeddingArrayInterface),
			}
		}
	}
	return &EmbeddingData{
		Index:     indexInt,
		Text:      textString,
		Embedding: embeddingArray,
	}, nil
}

type RerankResponse struct {
	Object string       `json:"object"`
	Model  string       `json:"model"`
	Data   []RerankData `json:"data"`

	Usage           TokenUsage      `json:"usage"`
	GatewayMetadata GatewayMetadata `json:"gateway_metadata"`
	// extract from headers
	ResponseHeaders ServiceHeaders
}

func NewRerankResponse(response *http.Response) (*RerankResponse, error) {
	var decodedBody map[string]any
	if err := json.NewDecoder(response.Body).Decode(&decodedBody); err != nil {
		return nil, &ValidationError{
			Field:   "body",
			Message: err.Error(),
		}
	}
	object, ok := decodedBody["object"].(string)
	if !ok {
		object = "rerank"
	}
	model, ok := decodedBody["model"].(string)
	if !ok {
		return nil, &ValidationError{
			Field:   "model",
			Message: fmt.Sprintf("invalid response, expected 'model' field to be string, got %T", decodedBody["model"]),
		}
	}
	var dataSlice []RerankData
	dataSliceInterface, ok := decodedBody["data"]
	if !ok {
		return nil, &ValidationError{
			Field:   "data",
			Message: "invalid response, expected 'data' field",
		}
	} else {
		// assert to []any and then assert each item to map[string]any
		dataSliceInterfaceSlice, ok := dataSliceInterface.([]any)
		if !ok {
			return nil, &ValidationError{
				Field:   "data",
				Message: fmt.Sprintf("invalid response, expected 'data' field to be array, got %T", dataSliceInterface),
			}
		}
		dataSlice = make([]RerankData, len(dataSliceInterfaceSlice))
		for i, rawItem := range dataSliceInterfaceSlice {
			dataSliceInterfaceMap, ok := rawItem.(map[string]any)
			if !ok {
				return nil, &ValidationError{
					Field:   "data",
					Message: fmt.Sprintf("invalid response, expected data slice interface map in 'data' to be object, got %T", dataSliceInterfaceSlice),
				}
			}
			rerankData, err := NewRerankData(dataSliceInterfaceMap)
			if err != nil {
				return nil, &ValidationError{
					Field:   "data",
					Message: err.Error(),
				}
			}
			dataSlice[i] = *rerankData
		}
	}

	var tokenUsage TokenUsage
	usage, ok := decodedBody["usage"].(map[string]any)
	if !ok {
		tokenUsage = TokenUsage{
			PromptTokens:        0,
			TotalTokens:         0,
			CompletionTokens:    0,
			PromptTokensDetails: []map[string]any{},
		}
	} else {
		tu, err := NewTokenUsage(usage)
		if err != nil {
			return nil, &ValidationError{
				Field:   "usage",
				Message: err.Error(),
			}
		}
		tokenUsage = *tu
	}

	var gatewayMetadata GatewayMetadata
	metadata, ok := decodedBody["gateway_metadata"].(map[string]any)
	if !ok {
		gatewayMetadata = GatewayMetadata{}
	} else {
		gatewayMetadata = NewGatewayMetadata(metadata)
	}

	serviceHeaders := NewServiceHeaders(response.Header)

	return &RerankResponse{
		Object:          object,
		Model:           model,
		Data:            dataSlice,
		Usage:           tokenUsage,
		GatewayMetadata: gatewayMetadata,
		ResponseHeaders: serviceHeaders,
	}, nil
}

type RerankData struct {
	Index          int                 `json:"index"`
	RelevanceScore float64             `json:"relevance_score"`
	Document       *[]RerankedDocument `json:"document"`
}

func NewRerankData(rerankData map[string]any) (*RerankData, error) {
	indexFloat, ok := rerankData["index"].(float64)
	if !ok {
		return nil,
			&ValidationError{
				Field:   "index",
				Message: fmt.Sprintf("invalid response, expected 'index' field to be int, got %T", rerankData["index"]),
			}
	}
	indexInt := int(indexFloat) // convert float to int
	relevanceScoreFloat, ok := rerankData["relevance_score"].(float64)
	if !ok {
		return nil, &ValidationError{
			Field:   "relevance_score",
			Message: fmt.Sprintf("invalid response, expected 'relevance_score' field to be float64, got %T", rerankData["relevance_score"]),
		}
	}
	var document *[]RerankedDocument = nil
	documentInterface, ok := rerankData["document"]
	if ok {
		documentSlice, ok := documentInterface.([]map[string]any)
		if !ok {
			return nil, &ValidationError{
				Field:   "document",
				Message: fmt.Sprintf("invalid response, expected 'document' field to be array, got %T", documentInterface),
			}
		}
		documentSliceInterface := make([]RerankedDocument, len(documentSlice))
		for i, documentMap := range documentSlice {
			doc, err := NewRerankedDocument(documentMap)
			if err != nil {
				return nil, &ValidationError{
					Field:   fmt.Sprintf("document[%d]", i),
					Message: err.Error(),
				}
			}
			documentSliceInterface[i] = *doc
		}
		document = &documentSliceInterface
	}
	return &RerankData{
		Index:          indexInt,
		RelevanceScore: relevanceScoreFloat,
		Document:       document,
	}, nil
}

type RerankedDocument struct {
	Text string `json:"text"`
}

func NewRerankedDocument(rerankedDocument map[string]any) (*RerankedDocument, error) {
	textString, ok := rerankedDocument["text"].(string)
	if !ok {
		return nil, &ValidationError{
			Field:   "text",
			Message: fmt.Sprintf("invalid response, expected 'text' field to be string, got %T", rerankedDocument["text"]),
		}
	}
	return &RerankedDocument{
		Text: textString,
	}, nil
}

type TokenizeRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type TokenizeResponse struct {
	Object          string          `json:"object"` // always "tokens"
	Model           string          `json:"model"`
	MaxModelLength  int             `json:"max_model_len"`
	Data            []int           `json:"data"`
	TokensString    *[]string       `json:"tokens_string"`
	GatewayMetadata GatewayMetadata `json:"gateway_metadata"`
	// extract from headers
	ResponseHeaders ServiceHeaders
}

func NewTokenizeResponse(response *http.Response) (*TokenizeResponse, error) {
	var decodedBody map[string]any
	if err := json.NewDecoder(response.Body).Decode(&decodedBody); err != nil {
		return nil, &ValidationError{
			Field:   "body",
			Message: err.Error(),
		}
	}
	object, ok := decodedBody["object"].(string)
	if !ok {
		object = "tokens"
	}

	model, _ := decodedBody["model"].(string)
	// model, ok := decodedBody["model"].(string)
	// if !ok {
	// 	return nil, &ValidationError{
	// 		Field:   "model",
	// 		Message: fmt.Sprintf("invalid response, expected 'model' field to be string, got %T", decodedBody["model"]),
	// 	}
	// }

	// ========= แก้เรื่อง float64 ===================
	maxModelLengthInterface, ok := getFirst(decodedBody, "max_model_len", "max_model_length")
	if !ok {
		return nil, &ValidationError{
			Field:   "max_model_len",
			Message: "invalid response, expected 'max_model_len' or 'max_model_length' field",
		}
	}
	var maxModelLengthInt int
	if val, ok := maxModelLengthInterface.(float64); ok {
		maxModelLengthInt = int(val)
	} else {
		return nil, &ValidationError{
			Field:   "max_model_len",
			Message: fmt.Sprintf("expected number, got %T", maxModelLengthInterface),
		}
	}
	// maxModelLengthInt, ok := maxModelLengthInterface.(int)
	// if !ok {
	// 	return nil, &ValidationError{
	// 		Field:   "max_model_len",
	// 		Message: fmt.Sprintf("invalid response, expected 'max_model_len' or 'max_model_length' field to be int, got %T", maxModelLengthInterface),
	// 	}
	// }

	// ========= แก้เรื่อง []any และ float64 ===================
	var dataSlice []int
	// dataSliceInterface, ok := decodedBody["data"]
	dataSliceInterface, ok := getFirst(decodedBody, "data", "tokens")
	if !ok {
		return nil, &ValidationError{Field: "data", Message: "field missing"}
	}

	dataAnySlice, ok := dataSliceInterface.([]any)
	if !ok {
		return nil, &ValidationError{
			Field:   "data",
			Message: fmt.Sprintf("expected array, got %T", dataSliceInterface),
		}
	}
	for _, v := range dataAnySlice {
		if fv, ok := v.(float64); ok {
			dataSlice = append(dataSlice, int(fv))
		}
	}
	// if !ok {
	// 	return nil, &ValidationError{
	// 		Field:   "data",
	// 		Message: "invalid response, expected 'data' field",
	// 	}
	// } else {
	// 	dataSliceInterfaceSlice, ok := dataSliceInterface.([]int)
	// 	if !ok {
	// 		return nil, &ValidationError{
	// 			Field:   "data",
	// 			Message: fmt.Sprintf("invalid response, expected 'data' field to be array, got %T", dataSliceInterface),
	// 		}
	// 	}
	// 	dataSlice = dataSliceInterfaceSlice
	// }

	// ================ แก้เรื่อง []any ========================
	var tokensString *[]string = nil
	if tsInterface, ok := getFirst(decodedBody, "token_strs", "tokens_string"); ok {
		if sAny, ok := tsInterface.([]any); ok {
			tempS := make([]string, 0, len(sAny))
			for _, v := range sAny {
				if str, ok := v.(string); ok {
					tempS = append(tempS, str)
				}
			}
			tokensString = &tempS
		}
	}
	// tokensStringInterface, ok := getFirst(decodedBody, "token_strs", "tokens_string")
	// if ok {
	// 	s, ok := tokensStringInterface.([]string)
	// 	if ok {
	// 		tokensString = &s
	// 	}
	// }

	var gatewayMetadata GatewayMetadata
	metadata, ok := decodedBody["gateway_metadata"].(map[string]any)
	if !ok {
		gatewayMetadata = GatewayMetadata{}
	} else {
		gatewayMetadata = NewGatewayMetadata(metadata)
	}

	return &TokenizeResponse{
		Object:          object,
		Model:           model,
		MaxModelLength:  maxModelLengthInt,
		Data:            dataSlice,
		TokensString:    tokensString,
		GatewayMetadata: gatewayMetadata,
	}, nil
}

type DetokenizeResponse struct {
	Object          string          `json:"object"` // always "prompt"
	Model           string          `json:"model"`
	Data            string          `json:"data"`
	GatewayMetadata GatewayMetadata `json:"gateway_metadata"`
	// extract from headers
	ResponseHeaders ServiceHeaders
}

// extract user message from prompt
func extractData(rawPrompt string) string {
	regexPattern := `(?s)<\|im_start\|>user\s*\r?\n(.*?)(?:<\|im_end\|>|$)`
	re := regexp.MustCompile(regexPattern)
	matches := re.FindStringSubmatch(rawPrompt)
	if len(matches) > 1 {
		return matches[1]
	}
	return rawPrompt
}

func NewDetokenizeResponse(response *http.Response) (*DetokenizeResponse, error) {
	var decodedBody map[string]any
	if err := json.NewDecoder(response.Body).Decode(&decodedBody); err != nil {
		return nil, &ValidationError{
			Field:   "body",
			Message: err.Error(),
		}
	}
	object, ok := decodedBody["object"].(string)
	if !ok {
		object = "prompt"
	}
	model, ok := decodedBody["model"].(string)
	if !ok {
		model = "string" //default
	}
	dataStringInterface, ok := getFirst(decodedBody, "data", "prompt", "text")
	if !ok {
		return nil, &ValidationError{
			Field:   "data",
			Message: fmt.Sprintf("invalid response, expected 'data' or 'prompt' field to be string, got %T", decodedBody["data"]),
		}
	}
	var dataString string
	dataString, ok = dataStringInterface.(string)
	if !ok {
		return nil, &ValidationError{
			Field:   "data",
			Message: fmt.Sprintf("invalid response, expected 'data' or 'prompt' field to be string, got %T", dataStringInterface),
		}
	}
	dataString = extractData(dataString)
	var gatewayMetadata GatewayMetadata
	metadata, ok := decodedBody["gateway_metadata"].(map[string]any)
	if !ok {
		gatewayMetadata = GatewayMetadata{}
	} else {
		gatewayMetadata = NewGatewayMetadata(metadata)
	}

	return &DetokenizeResponse{
		Object:          object,
		Model:           model,
		Data:            dataString,
		GatewayMetadata: gatewayMetadata,
		ResponseHeaders: NewServiceHeaders(response.Header),
	}, nil
}

type ToolCall struct {
	Index    *int              `json:"index,omitzero"`
	Id       *string           `json:"id"`
	Type     string            `json:"type"` // Literal['function']
	Function ToolCallsFunction `json:"function"`
}

func NewToolCall(toolCall map[string]any) (*ToolCall, error) {
	var index *int = nil
	indexInterface, ok := toolCall["index"]
	if ok {
		indexInt, ok := indexInterface.(int)
		if ok {
			index = &indexInt
		}
	}
	id, ok := toolCall["id"].(string)
	if !ok {
		return nil,
			&ValidationError{
				Field:   "id",
				Message: "invalid tool call: id field required",
			}
	}
	function, err := NewToolCallsFunction(toolCall["function"].(map[string]any))
	if err != nil {
		return nil, &ValidationError{
			Field:   "function",
			Message: err.Error(),
		}
	}
	if function == nil {
		return nil, &ValidationError{
			Field:   "function",
			Message: "invalid tool call: function field required",
		}
	}
	if function.Name == nil {
		return nil, &ValidationError{
			Field:   "function.name",
			Message: "invalid tool call: function.name field required",
		}
	}
	if function.Arguments == nil {
		return nil, &ValidationError{
			Field:   "function.arguments",
			Message: "invalid tool call: function.arguments field required",
		}
	}
	return &ToolCall{
		Index:    index,
		Id:       &id,
		Type:     "function",
		Function: *function,
	}, nil
}

func NewToolCallsDelta(toolCall map[string]any) (*ToolCall, error) {
	index, ok := toolCall["index"].(int)
	if !ok {
		return nil, &ValidationError{
			Field:   "index",
			Message: "index is required",
		}
	}
	var id *string = nil
	idStr, ok := toolCall["id"].(string)
	if ok {
		id = &idStr
	}
	function, err := NewToolCallsFunction(toolCall["function"].(map[string]any))
	if err != nil {
		return nil, &ValidationError{
			Field:   "function",
			Message: err.Error(),
		}
	}
	if function == nil {
		return nil, &ValidationError{
			Field:   "function",
			Message: "invalid tool call: function field required",
		}
	}
	return &ToolCall{
		Index:    &index,
		Id:       id,
		Type:     "function",
		Function: *function,
	}, nil
}

type ToolCallsFunction struct {
	Name      *string `json:"name"`      // can nil if streaming
	Arguments *string `json:"arguments"` // can nil if streaming
}

func NewToolCallsFunction(function map[string]any) (*ToolCallsFunction, error) {
	var name *string = nil
	nameStr, ok := function["name"].(string)
	if ok {
		name = &nameStr
	}
	var arguments *string = nil
	argumentsStr, ok := function["arguments"].(string)
	if ok {
		arguments = &argumentsStr
	}
	return &ToolCallsFunction{
		Name:      name,
		Arguments: arguments,
	}, nil
}

type ChatResponseMessage struct {
	Role             string      `json:"role"` // Literal['system', 'user', 'assistant']
	Content          *string     `json:"content"`
	ReasoningContent *string     `json:"reasoning_content"`
	ToolsCalls       *[]ToolCall `json:"tool_calls"`
}

func NewChatResponseMessage(message map[string]any) (*ChatResponseMessage, error) {
	role, ok := message["role"].(string)
	if !ok {
		role = "assistant"
	}
	var reasoningContent *string = nil
	reasoningContentInterface, ok := getFirst(message, "reasoning_content", "reasoning")
	if ok {
		reasoningContentStr, ok := reasoningContentInterface.(string)
		if ok {
			reasoningContent = &reasoningContentStr
		}
	}
	var content *string = nil
	contentStr, ok := message["content"].(string)
	if ok {
		content = &contentStr
	} else if reasoningContent == nil {
		contentStr = ""
		content = &contentStr
	}

	var toolsCalls *[]ToolCall = nil
	toolsCallsArr, ok := message["tool_calls"].([]any)
	if ok {
		toolsCalls = &[]ToolCall{}
		for i, toolCall := range toolsCallsArr {
			toolCall, err := NewToolCall(toolCall.(map[string]any))
			if err != nil {
				return nil, &ValidationError{
					Field:   fmt.Sprintf("tool_calls[%d]", i),
					Message: err.Error(),
				}
			}
			*toolsCalls = append(*toolsCalls, *toolCall)
		}
	}
	return &ChatResponseMessage{
		Role:             role,
		Content:          content,
		ReasoningContent: reasoningContent,
		ToolsCalls:       toolsCalls,
	}, nil
}

type Logprob struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
	Bytes   []int   `json:"bytes"`
}

type ContentLogprobs struct {
	Token       string     `json:"token"`
	Logprob     float64    `json:"logprob"`
	Bytes       []int      `json:"bytes"`
	TopLogprobs *[]Logprob `json:"top_logprobs,omitzero"`
}

func NewContentLogprobs(logprob map[string]any) (*ContentLogprobs, error) {
	token, ok := logprob["token"].(string)
	if !ok {
		return nil,
			&ValidationError{
				Field:   "token",
				Message: "invalid logprob: token field required",
			}
	}
	logprobFloat, ok := logprob["logprob"].(float64)
	if !ok {
		return nil, &ValidationError{
			Field:   "logprob",
			Message: "invalid logprob: logprob field required",
		}
	}
	bytes, ok := logprob["bytes"].([]int)
	if !ok {
		return nil, &ValidationError{
			Field:   "bytes",
			Message: "invalid logprob: bytes field required",
		}
	}
	var topLogprobs *[]Logprob = nil
	topLogprobsSlice, ok := logprob["top_logprobs"].([]any)
	if ok {
		topLogprobs = &[]Logprob{}
		for i, topLogprobInterface := range topLogprobsSlice {
			topLogprob, ok := topLogprobInterface.(Logprob)
			if !ok {
				return nil, &ValidationError{
					Field:   fmt.Sprintf("top_logprobs[%d]", i),
					Message: fmt.Sprintf("invalid top logprob at position %d, got %T", i, topLogprobInterface),
				}
			}
			*topLogprobs = append(*topLogprobs, topLogprob)
		}
	}
	return &ContentLogprobs{
		Token:       token,
		Logprob:     logprobFloat,
		Bytes:       bytes,
		TopLogprobs: topLogprobs,
	}, nil
}

type ChatCompletionLogprobs struct {
	Content *[]ContentLogprobs `json:"content,omitzero"`
}

func NewChatCompletionLogprobs(logprobs map[string]any) (*ChatCompletionLogprobs, error) {
	var content *[]ContentLogprobs = nil
	contentSlice, ok := logprobs["content"].([]any)
	if ok {
		content = &[]ContentLogprobs{}
		for i, contentLogprobInterface := range contentSlice {
			contentLogprob, err := NewContentLogprobs(contentLogprobInterface.(map[string]any))
			if err != nil {
				return nil, &ValidationError{
					Field:   fmt.Sprintf("content[%d]", i),
					Message: fmt.Sprintf("invalid content logprob at position %d, got %T", i, contentLogprobInterface),
				}
			}
			*content = append(*content, *contentLogprob)
		}
	}
	return &ChatCompletionLogprobs{
		Content: content,
	}, nil
}

type ChatCompletionChoice struct {
	Index        int                     `json:"index"`
	Message      ChatResponseMessage     `json:"message"`
	FinishReason *string                 `json:"finish_reason"` // can nil while streaming
	Logprobs     *ChatCompletionLogprobs `json:"logprobs"`
}

func NewChatCompletionChoice(choice map[string]any) (*ChatCompletionChoice, error) {
	// index, ok := choice["index"].(int)
	// if !ok {
	// 	return nil, &ValidationError{
	// 		Field:   "index",
	// 		Message: "invalid choice: index field required",
	// 	}
	// }
	indexFloat, ok := choice["index"].(float64)
	if !ok {
		return nil, &ValidationError{
			Field:   "index",
			Message: "invalid choice: index field required (must be number)",
		}
	}
	index := int(indexFloat)

	message, err := NewChatResponseMessage(choice["message"].(map[string]any))
	if err != nil {
		return nil, &ValidationError{
			Field:   "message",
			Message: err.Error(),
		}
	}
	var finishReason *string = nil
	finishReasonStr, ok := choice["finish_reason"].(string)
	if ok {
		finishReason = &finishReasonStr
	}
	var logprob *ChatCompletionLogprobs = nil
	logprobMap, ok := choice["logprobs"].(map[string]any)
	if ok {
		logprob, err = NewChatCompletionLogprobs(logprobMap)
		if err != nil {
			return nil, &ValidationError{
				Field:   "logprobs",
				Message: err.Error(),
			}
		}
	}
	return &ChatCompletionChoice{
		Index:        index,
		Message:      *message,
		FinishReason: finishReason,
		Logprobs:     logprob,
	}, nil
}

type ChatCompletionResponse struct {
	Id      string                 `json:"id"`
	Object  string                 `json:"object"`
	Model   string                 `json:"model"`
	Created int64                  `json:"created"`
	Choices []ChatCompletionChoice `json:"choices"`

	Usage             *TokenUsage `json:"usage"`
	SystemFingerprint *string     `json:"system_fingerprint"`

	GatewayMetadata  GatewayMetadata `json:"gateway_metadata"`
	ServiceTier      *map[string]any `json:"service_tier"`
	PromptLogprobs   *map[string]any `json:"prompt_logprobs"`
	KvTransferParams *map[string]any `json:"kv_transfer_params"`

	// extract from headers
	ResponseHeaders ServiceHeaders
}

func NewChatCompletionResponse(response http.Response) (*ChatCompletionResponse, error) {
	m := make(map[string]any)
	if err := json.NewDecoder(response.Body).Decode(&m); err != nil {
		return nil, &ValidationError{
			Field:   "body",
			Message: err.Error(),
		}
	}
	id, ok := m["id"].(string)
	if !ok {
		id = "chatcmpl-" + uuid.New().String()
	}
	object, ok := m["object"].(string)
	if !ok {
		object = "chat.completion"
	}
	model, ok := m["model"].(string)
	if !ok {
		return nil, &ValidationError{
			Field:   "model",
			Message: "invalid chat completion response: model field required",
		}
	}
	created, ok := m["created"].(int64)
	if !ok {
		created = time.Now().Unix()
	}
	choices, ok := m["choices"].([]any)
	if !ok {
		return nil, &ValidationError{
			Field:   "choices",
			Message: "invalid chat completion response: choices field required",
		}
	}
	var choicesSlice []ChatCompletionChoice
	for i, choice := range choices {
		choice, err := NewChatCompletionChoice(choice.(map[string]any))
		if err != nil {
			return nil, &ValidationError{
				Field:   fmt.Sprintf("choices[%d]", i),
				Message: err.Error(),
			}
		}
		choicesSlice = append(choicesSlice, *choice)
	}
	var usageStruct *TokenUsage = nil
	usage, ok := m["usage"].(map[string]any)
	if ok {
		us, err := NewTokenUsage(usage)
		if err != nil {
			return nil, &ValidationError{
				Field:   "usage",
				Message: err.Error(),
			}
		}
		usageStruct = us
	}
	var systemFingerprint *string = nil
	systemFingerprintStr, ok := m["system_fingerprint"].(string)
	if ok {
		systemFingerprint = &systemFingerprintStr
	}
	gatewayMetadata, ok := m["gateway_metadata"].(map[string]any)
	if !ok {
		return nil, &ValidationError{
			Field:   "gateway_metadata",
			Message: "invalid chat completion response: gateway_metadata field required",
		}
	}
	gatewayMetadataStruct := NewGatewayMetadata(gatewayMetadata)

	var serviceTier *map[string]any = nil
	serviceTierMap, ok := m["service_tier"].(map[string]any)
	if ok {
		serviceTier = &serviceTierMap
	}
	var promptLogprobs *map[string]any = nil
	promptLogprobsMap, ok := m["prompt_logprobs"].(map[string]any)
	if ok {
		promptLogprobs = &promptLogprobsMap
	}
	var kvTransferParams *map[string]any = nil
	kvTransferParamsMap, ok := m["kv_transfer_params"].(map[string]any)
	if ok {
		kvTransferParams = &kvTransferParamsMap
	}
	responseHeaders := NewServiceHeaders(response.Header)

	return &ChatCompletionResponse{
		Id:                id,
		Object:            object,
		Model:             model,
		Created:           created,
		Choices:           choicesSlice,
		Usage:             usageStruct,
		SystemFingerprint: systemFingerprint,
		GatewayMetadata:   gatewayMetadataStruct,
		ServiceTier:       serviceTier,
		PromptLogprobs:    promptLogprobs,
		KvTransferParams:  kvTransferParams,
		ResponseHeaders:   responseHeaders,
	}, nil
}

type ChatStreamDelta struct {
	Role             *string     `json:"role,omitzero"`
	Content          *string     `json:"content,omitzero"`
	ReasoningContent *string     `json:"reasoning_content,omitzero"`
	ToolsCalls       *[]ToolCall `json:"tools_calls,omitzero"`
}

func NewChatStreamDelta(delta map[string]any) (*ChatStreamDelta, error) {
	defaultRole := "assistant"
	var role *string = &defaultRole
	roleStr, ok := delta["role"].(string)
	if ok {
		role = &roleStr
	}
	var reasoningContent *string = nil
	reasoningContentInterface, ok := getFirst(delta, "reasoning_content", "reasoning")
	if ok {
		reasoningContentStr := reasoningContentInterface.(string)
		reasoningContent = &reasoningContentStr
	}
	var content *string = nil
	contentStr, ok := delta["content"].(string)
	if ok {
		content = &contentStr
	} else if reasoningContent == nil {
		contentStr = ""
		content = &contentStr
	}
	var toolsCalls *[]ToolCall = nil
	toolsCallsSlice, ok := delta["tools_calls"].([]any)
	if ok {
		toolsCalls = &[]ToolCall{}
		for i, toolsCall := range toolsCallsSlice {
			toolsCall, err := NewToolCallsDelta(toolsCall.(map[string]any))
			if err != nil {
				return nil, &ValidationError{
					Field:   fmt.Sprintf("tools_calls[%d]", i),
					Message: err.Error(),
				}
			}
			*toolsCalls = append(*toolsCalls, *toolsCall)
		}
	}
	return &ChatStreamDelta{
		Role:             role,
		Content:          content,
		ReasoningContent: reasoningContent,
		ToolsCalls:       toolsCalls,
	}, nil
}

type ChatCompletionStreamChoice struct {
	Index        int                     `json:"index"`
	Delta        ChatStreamDelta         `json:"delta"`
	FinishReason *string                 `json:"finish_reason"`
	Logprobs     *ChatCompletionLogprobs `json:"logprobs"`
}

func NewChatCompletionStreamChoice(choice map[string]any) (*ChatCompletionStreamChoice, error) {

	indexFloat, ok := choice["index"].(float64)
	if !ok {
		return nil, &ValidationError{Field: "index", Message: "index required"}
	}
	index := int(indexFloat)
	// index, ok := choice["index"].(int)
	// if !ok {
	// 	return nil, &ValidationError{
	// 		Field:   "index",
	// 		Message: "invalid chat completion stream choice: index field required",
	// 	}
	// }

	delta, err := NewChatStreamDelta(choice["delta"].(map[string]any))
	if err != nil {
		return nil, &ValidationError{
			Field:   "delta",
			Message: err.Error(),
		}
	}
	var finishReason *string = nil
	finishReasonStr, ok := choice["finish_reason"].(string)
	if ok {
		finishReason = &finishReasonStr
	}
	var logprob *ChatCompletionLogprobs = nil
	logprobMap, ok := choice["logprobs"].(map[string]any)
	if ok {
		logprob, err = NewChatCompletionLogprobs(logprobMap)
		if err != nil {
			return nil, &ValidationError{
				Field:   "logprobs",
				Message: err.Error(),
			}
		}
	}
	return &ChatCompletionStreamChoice{
		Index:        index,
		Delta:        *delta,
		FinishReason: finishReason,
		Logprobs:     logprob,
	}, nil
}

type ChatCompletionStreamChunk struct {
	Id      string                       `json:"id"`
	Object  string                       `json:"object"`
	Model   string                       `json:"model"`
	Created int64                        `json:"created"`
	Choices []ChatCompletionStreamChoice `json:"choices"`

	Usage             *TokenUsage `json:"usage"`
	SystemFingerprint *string     `json:"system_fingerprint"`

	GatewayMetadata  GatewayMetadata `json:"gateway_metadata"`
	ServiceTier      *map[string]any `json:"service_tier"`
	PromptLogprobs   *map[string]any `json:"prompt_logprobs"`
	KvTransferParams *map[string]any `json:"kv_transfer_params"`

	// extract from headers
	ResponseHeaders *ServiceHeaders
}

func NewChatCompletionStreamChunk(chunk map[string]any) (*ChatCompletionStreamChunk, error) {
	id, ok := chunk["id"].(string)
	if !ok {
		id = "chatcmpl-" + uuid.New().String()
	}
	object, ok := chunk["object"].(string)
	if !ok {
		object = "chat.completion.chunk"
	}
	model, ok := chunk["model"].(string)
	if !ok {
		return nil, &ValidationError{
			Field:   "model",
			Message: "invalid chat completion stream chunk: model field required",
		}
	}
	var created int64
	if cFloat, ok := chunk["created"].(float64); ok {
		created = int64(cFloat)
	} else {
		created = time.Now().Unix()
	}
	// created, ok := chunk["created"].(int64)
	// if !ok {
	// 	created = time.Now().Unix()
	// }
	choices, ok := chunk["choices"].([]any)
	if !ok {
		return nil, &ValidationError{
			Field:   "choices",
			Message: "invalid chat completion stream chunk: choices field required",
		}
	}
	var choicesSlice []ChatCompletionStreamChoice
	for i, choice := range choices {
		choice, err := NewChatCompletionStreamChoice(choice.(map[string]any))
		if err != nil {
			return nil, &ValidationError{
				Field:   fmt.Sprintf("choices[%d]", i),
				Message: err.Error(),
			}
		}
		choicesSlice = append(choicesSlice, *choice)
	}
	var usageStruct *TokenUsage = nil
	usage, ok := chunk["usage"].(map[string]any)
	if ok {
		us, err := NewTokenUsage(usage)
		if err != nil {
			return nil, err
		}
		usageStruct = us
	}
	var systemFingerprint *string = nil
	systemFingerprintStr, ok := chunk["system_fingerprint"].(string)
	if ok {
		systemFingerprint = &systemFingerprintStr
	}

	var gatewayMetadataStruct GatewayMetadata
	if gatewayMetadata, ok := chunk["gateway_metadata"].(map[string]any); ok {
		gatewayMetadataStruct = NewGatewayMetadata(gatewayMetadata)
	} else {
		gatewayMetadataStruct = GatewayMetadata{}
	}
	// gatewayMetadata, ok := chunk["gateway_metadata"].(map[string]any)
	// if !ok {
	// 	return nil, &ValidationError{
	// 		Field:   "gateway_metadata",
	// 		Message: "invalid chat completion stream chunk: gateway_metadata field required",
	// 	}
	// }
	// gatewayMetadataStruct := NewGatewayMetadata(gatewayMetadata)

	var serviceTier *map[string]any = nil
	serviceTierMap, ok := chunk["service_tier"].(map[string]any)
	if ok {
		serviceTier = &serviceTierMap
	}
	var promptLogprobs *map[string]any = nil
	promptLogprobsMap, ok := chunk["prompt_logprobs"].(map[string]any)
	if ok {
		promptLogprobs = &promptLogprobsMap
	}
	var kvTransferParams *map[string]any = nil
	kvTransferParamsMap, ok := chunk["kv_transfer_params"].(map[string]any)
	if ok {
		kvTransferParams = &kvTransferParamsMap
	}

	return &ChatCompletionStreamChunk{
		Id:                id,
		Object:            object,
		Model:             model,
		Created:           created,
		Choices:           choicesSlice,
		Usage:             usageStruct,
		SystemFingerprint: systemFingerprint,
		GatewayMetadata:   gatewayMetadataStruct,
		ServiceTier:       serviceTier,
		PromptLogprobs:    promptLogprobs,
		KvTransferParams:  kvTransferParams,
	}, nil
}

var (
	headerData  = regexp.MustCompile(`^data:\s*`)
	errorPrefix = regexp.MustCompile(`^data:\s*{"error":`)
)

type ChatCompletionStreamReader struct {
	emptyMessagesLimit uint
	isFinished         bool
	reader             *bufio.Reader
	response           *http.Response

	ResponseHeaders *ServiceHeaders
}

func (sr *ChatCompletionStreamReader) Recv() (*ChatCompletionStreamChunk, error) {
	rawLine, err := sr.RecvRaw()
	if err != nil {
		return nil, err
	}
	var response map[string]any
	err = json.Unmarshal(rawLine, &response)
	if err != nil {
		return nil, err
	}
	chatChunk, err := NewChatCompletionStreamChunk(response)
	if err != nil {
		return nil, err
	}
	chatChunk.ResponseHeaders = sr.ResponseHeaders
	return chatChunk, nil
}

func (sr *ChatCompletionStreamReader) RecvRaw() ([]byte, error) {
	if sr.isFinished {
		return nil, io.EOF
	}

	return sr.processLines()
}

func (sr *ChatCompletionStreamReader) processLines() ([]byte, error) {
	var (
		emptyMessagesCount uint
		hasErrorPrefix     bool
	)

	for {
		rawLine, err := sr.reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		noSpaceLine := bytes.TrimSpace(rawLine)
		if errorPrefix.Match(noSpaceLine) {
			hasErrorPrefix = true
		}

		if !headerData.Match(noSpaceLine) || hasErrorPrefix {
			if hasErrorPrefix {
				noSpaceLine = headerData.ReplaceAll(noSpaceLine, nil)
			}

			emptyMessagesCount++
			if emptyMessagesCount > sr.emptyMessagesLimit {
				return nil, fmt.Errorf("exceeded empty messages limit of %d", sr.emptyMessagesLimit)
			}

			continue
		}

		noPrefixLine := headerData.ReplaceAll(noSpaceLine, nil)
		if string(noPrefixLine) == "[DONE]" {
			sr.isFinished = true
			return nil, io.EOF
		}

		return noPrefixLine, nil
	}
}

func (sr *ChatCompletionStreamReader) Close() error {
	return sr.response.Body.Close()
}

func StreamChatCompletionResponse(response *http.Response) (*ChatCompletionStreamReader, error) {
	sc := response.StatusCode
	if sc != http.StatusOK {
		// var content_bytes []byte
		// _, err := response.Body.Read(content_bytes)
		content_bytes, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, &ValidationError{
				Field:   "body",
				Message: fmt.Sprintf("error reading error response body: %s", err.Error()),
			}
		}
		if len(content_bytes) == 0 {
			return nil, &ValidationError{
				Field:   "body",
				Message: fmt.Sprintf("error response body is empty, status code: %d", sc),
			}
		}

		var errorResponse map[string]any
		err = json.Unmarshal(content_bytes, &errorResponse)
		if err != nil {
			return nil, &ValidationError{
				Field:   "body",
				Message: fmt.Sprintf("error parsing error response body: %s", err.Error()),
			}
		}
		return nil, &ValidationError{
			Field:   "body",
			Message: fmt.Sprintf("error reading response body: status code %d", sc),
		}
	}

	ct := response.Header.Get("Content-Type")
	if ct != "text/event-stream" {
		return nil, &ValidationError{
			Field:   "Content-Type",
			Message: fmt.Sprintf("invalid content type for streaming: %s", ct),
		}
	}
	// Create decoder and process events
	serviceHeaders := NewServiceHeaders(response.Header)
	streamReader := &ChatCompletionStreamReader{
		emptyMessagesLimit: 5,
		isFinished:         false,
		reader:             bufio.NewReader(response.Body),
		response:           response,
		ResponseHeaders:    &serviceHeaders,
	}
	return streamReader, nil
}

type GatewayMetadata struct {
	UsageDetails *map[string]any `json:"usage_details"`
	ModelVersion *string         `json:"model_version"`
	GatewayInfo  *string         `json:"gateway_info"`

	ProcessingTime *int    `json:"processing_time"`
	CacheHit       *bool   `json:"cache_hit"`
	ServiceType    *string `json:"service_type"`
}

func NewGatewayMetadata(m map[string]any) GatewayMetadata {
	gm := GatewayMetadata{}
	if v, ok := m["usage_details"].(map[string]any); ok {
		gm.UsageDetails = &v
	}
	if v, ok := m["model_version"].(string); ok {
		gm.ModelVersion = &v
	}
	if v, ok := m["gateway_info"].(string); ok {
		gm.GatewayInfo = &v
	}
	if v, ok := m["processing_time"].(float64); ok {
		i := int(v)
		gm.ProcessingTime = &i
	}
	if v, ok := m["cache_hit"].(bool); ok {
		gm.CacheHit = &v
	}
	if v, ok := m["service_type"].(string); ok {
		gm.ServiceType = &v
	}
	return gm
}

type ServiceHeaders struct {
	RequestId      *string `json:"request_id"`
	CacheHit       *bool   `json:"cache_hit"`
	GatewayVersion *string `json:"gateway_version"`
	ProcessingTime *int    `json:"processing_time"` // nil: in case of error

	RateLimitLimit     *int `json:"ratelimit_limit"`
	RateLimitRemaining *int `json:"ratelimit_remaining"`
	RateLimitReset     *int `json:"ratelimit_reset"`
	RateLimitWindow    *int `json:"ratelimit_window"`

	QuotaDailyUsed        *float64 `json:"quota_daily_used"`
	QuotaDailyBudget      *float64 `json:"quota_daily_budget"`
	QuotaDailyRemaining   *float64 `json:"quota_daily_remaining"`
	QuotaMonthlyUsed      *float64 `json:"quota_monthly_used"`
	QuotaMonthlyBudget    *float64 `json:"quota_monthly_budget"`
	QuotaMonthlyRemaining *float64 `json:"quota_monthly_remaining"`
	QuotaStatus           *string  `json:"quota_status"` // Literal['OK', 'WARNING', 'EXCEEDED']
}

func NewServiceHeaders(headers http.Header) ServiceHeaders {
	serHeaders := ServiceHeaders{}
	reqId := headers.Get("x-request-id")
	if reqId != "" {
		serHeaders.RequestId = &reqId
	}
	cacheHit, err := strconv.ParseBool(headers.Get("x-cache-hit"))
	if err == nil {
		serHeaders.CacheHit = &cacheHit
	}
	gatewayVersion := headers.Get("x-gateway-version")
	if gatewayVersion != "" {
		serHeaders.GatewayVersion = &gatewayVersion
	}
	processingTime, err := strconv.Atoi(headers.Get("x-processing-time"))
	if err == nil {
		serHeaders.ProcessingTime = &processingTime
	}
	rll, err := strconv.Atoi(headers.Get("x-ratelimit-limit"))
	if err == nil {
		serHeaders.RateLimitLimit = &rll
	}
	rlr, err := strconv.Atoi(headers.Get("x-ratelimit-remaining"))
	if err == nil {
		serHeaders.RateLimitRemaining = &rlr
	}
	rlrst, err := strconv.Atoi(headers.Get("x-ratelimit-reset"))
	if err == nil {
		serHeaders.RateLimitReset = &rlrst
	}
	rlw, err := strconv.Atoi(headers.Get("x-ratelimit-window"))
	if err == nil {
		serHeaders.RateLimitWindow = &rlw
	}
	qdu, err := strconv.ParseFloat(headers.Get("x-quota-daily-used"), 64)
	if err == nil {
		serHeaders.QuotaDailyUsed = &qdu
	}
	qdb, err := strconv.ParseFloat(headers.Get("x-quota-daily-budget"), 64)
	if err == nil {
		serHeaders.QuotaDailyBudget = &qdb
	}
	qdr, err := strconv.ParseFloat(headers.Get("x-quota-daily-remaining"), 64)
	if err == nil {
		serHeaders.QuotaDailyRemaining = &qdr
	}
	qmu, err := strconv.ParseFloat(headers.Get("x-quota-monthly-used"), 64)
	if err == nil {
		serHeaders.QuotaMonthlyUsed = &qmu
	}
	qmb, err := strconv.ParseFloat(headers.Get("x-quota-monthly-budget"), 64)
	if err == nil {
		serHeaders.QuotaMonthlyBudget = &qmb
	}
	qmr, err := strconv.ParseFloat(headers.Get("x-quota-monthly-remaining"), 64)
	if err == nil {
		serHeaders.QuotaMonthlyRemaining = &qmr
	}
	qs := headers.Get("x-quota-status")
	if qs != "" {
		serHeaders.QuotaStatus = &qs
	}
	return serHeaders
}

func (h *ServiceHeaders) IsExceedRateLimit() error {
	if (h.RateLimitRemaining != nil && *h.RateLimitRemaining <= 0) && (h.RateLimitLimit != nil && *h.RateLimitLimit >= 0) {
		return fmt.Errorf("rate limit exceeded")
	}
	return nil
}

func (h *ServiceHeaders) IsInsufficientQuota() error {
	if (h.QuotaDailyBudget != nil && *h.QuotaDailyBudget >= 0) && (h.QuotaDailyRemaining != nil && *h.QuotaDailyRemaining <= 0) ||
		(h.QuotaMonthlyBudget != nil && *h.QuotaMonthlyBudget >= 0) && (h.QuotaMonthlyRemaining != nil && *h.QuotaMonthlyRemaining <= 0) ||
		(h.QuotaStatus != nil && *h.QuotaStatus == "EXCEEDED") {
		return fmt.Errorf("quota exceeded")
	}
	return nil
}
