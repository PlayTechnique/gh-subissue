package api

import (
	"fmt"
	"net/http"
)

// APIError represents a GitHub API error with context.
type APIError struct {
	StatusCode int
	Message    string
	Operation  string // e.g., "create issue", "link sub-issue"
	Hint       string // actionable suggestion
}

func (e *APIError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("%s: %s\n\nHint: %s", e.Operation, e.Message, e.Hint)
	}
	return fmt.Sprintf("%s: %s", e.Operation, e.Message)
}

// newAPIError creates an APIError with appropriate hints based on status code.
func newAPIError(statusCode int, message, operation string) *APIError {
	e := &APIError{
		StatusCode: statusCode,
		Message:    message,
		Operation:  operation,
	}

	switch statusCode {
	case http.StatusUnauthorized: // 401
		e.Hint = "Run 'gh auth login' to authenticate"
	case http.StatusForbidden: // 403
		e.Hint = "Check that you have write access to this repository"
	case http.StatusNotFound: // 404
		e.Hint = "Verify the repository exists and you have access to it"
	case http.StatusGone: // 410
		e.Hint = "Issues are disabled for this repository. Enable them in repository Settings > Features"
	case http.StatusUnprocessableEntity: // 422
		e.Hint = "Check that all required fields are provided and valid"
	case http.StatusTooManyRequests: // 429
		e.Hint = "Rate limited by GitHub. Wait a moment and try again"
	}

	return e
}

// IsNotFound returns true if the error is a 404 Not Found.
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// IsDisabled returns true if the error is a 410 Gone (feature disabled).
func IsDisabled(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusGone
	}
	return false
}

// IsAuthError returns true if the error is authentication-related.
func IsAuthError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden
	}
	return false
}

// IsRateLimited returns true if the error is a rate limit error.
func IsRateLimited(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusTooManyRequests
	}
	return false
}
