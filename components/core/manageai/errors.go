package manageai

import (
	"fmt"
)

// Error types for dynamic errors with context
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

type APIError struct {
	Code    int
	Message string
	Details string
}

func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("API error %d: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

// Usage examples (commented out):
/*
// Returning predefined errors
func validateAPIKey(key string) error {
	if key == "" {
		return ErrInvalidAPIKey
	}
	return nil
}

// Wrapping errors with context
func processRequest(req Request) error {
	if err := validateRequest(req); err != nil {
		return fmt.Errorf("failed to process request: %w", err)
	}
	return nil
}

// Checking for specific errors
func handleResponse(resp *http.Response) error {
	if resp.StatusCode == 401 {
		return ErrUnauthorized
	}
	if resp.StatusCode == 429 {
		return ErrRateLimitExceeded
	}
	return nil
}

// Using custom error types
func validateField(field string, value interface{}) error {
	if value == nil {
		return NewValidationError(field, "value cannot be nil")
	}
	return nil
}
*/