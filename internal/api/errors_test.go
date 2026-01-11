package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestAPIError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		message        string
		operation      string
		wantHintSubstr string
	}{
		{
			name:           "401 unauthorized",
			statusCode:     http.StatusUnauthorized,
			message:        "Requires authentication",
			operation:      "create issue",
			wantHintSubstr: "gh auth login",
		},
		{
			name:           "403 forbidden",
			statusCode:     http.StatusForbidden,
			message:        "Must have write access",
			operation:      "create issue",
			wantHintSubstr: "gh repo view",
		},
		{
			name:           "404 not found",
			statusCode:     http.StatusNotFound,
			message:        "Not Found",
			operation:      "create issue",
			wantHintSubstr: "repository exists",
		},
		{
			name:           "410 gone (issues disabled)",
			statusCode:     http.StatusGone,
			message:        "Issues has been disabled",
			operation:      "create issue",
			wantHintSubstr: "Settings > Features",
		},
		{
			name:           "422 validation error",
			statusCode:     http.StatusUnprocessableEntity,
			message:        "Validation Failed",
			operation:      "create issue",
			wantHintSubstr: "required fields",
		},
		{
			name:           "429 rate limited",
			statusCode:     http.StatusTooManyRequests,
			message:        "Rate limit exceeded",
			operation:      "create issue",
			wantHintSubstr: "Rate limited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newAPIError(tt.statusCode, tt.message, tt.operation)

			if err.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", err.StatusCode, tt.statusCode)
			}

			if err.Message != tt.message {
				t.Errorf("Message = %q, want %q", err.Message, tt.message)
			}

			if err.Operation != tt.operation {
				t.Errorf("Operation = %q, want %q", err.Operation, tt.operation)
			}

			if !strings.Contains(err.Hint, tt.wantHintSubstr) {
				t.Errorf("Hint = %q, want to contain %q", err.Hint, tt.wantHintSubstr)
			}

			// Verify Error() includes hint
			errStr := err.Error()
			if !strings.Contains(errStr, "Hint:") {
				t.Errorf("Error() = %q, want to contain 'Hint:'", errStr)
			}
		})
	}
}

func TestAPIErrorHelpers(t *testing.T) {
	t.Run("IsNotFound", func(t *testing.T) {
		err := newAPIError(http.StatusNotFound, "Not Found", "test")
		if !IsNotFound(err) {
			t.Error("IsNotFound() = false, want true")
		}

		otherErr := newAPIError(http.StatusForbidden, "Forbidden", "test")
		if IsNotFound(otherErr) {
			t.Error("IsNotFound() = true for 403, want false")
		}
	})

	t.Run("IsDisabled", func(t *testing.T) {
		err := newAPIError(http.StatusGone, "Issues disabled", "test")
		if !IsDisabled(err) {
			t.Error("IsDisabled() = false, want true")
		}

		otherErr := newAPIError(http.StatusNotFound, "Not Found", "test")
		if IsDisabled(otherErr) {
			t.Error("IsDisabled() = true for 404, want false")
		}
	})

	t.Run("IsAuthError", func(t *testing.T) {
		err401 := newAPIError(http.StatusUnauthorized, "Unauthorized", "test")
		if !IsAuthError(err401) {
			t.Error("IsAuthError() = false for 401, want true")
		}

		err403 := newAPIError(http.StatusForbidden, "Forbidden", "test")
		if !IsAuthError(err403) {
			t.Error("IsAuthError() = false for 403, want true")
		}

		otherErr := newAPIError(http.StatusNotFound, "Not Found", "test")
		if IsAuthError(otherErr) {
			t.Error("IsAuthError() = true for 404, want false")
		}
	})

	t.Run("IsRateLimited", func(t *testing.T) {
		err := newAPIError(http.StatusTooManyRequests, "Rate limited", "test")
		if !IsRateLimited(err) {
			t.Error("IsRateLimited() = false, want true")
		}

		otherErr := newAPIError(http.StatusForbidden, "Forbidden", "test")
		if IsRateLimited(otherErr) {
			t.Error("IsRateLimited() = true for 403, want false")
		}
	})
}

func TestAPIErrorNoHint(t *testing.T) {
	// Test unknown status code doesn't get a hint
	err := newAPIError(http.StatusInternalServerError, "Server error", "test")
	if err.Hint != "" {
		t.Errorf("Hint = %q, want empty for unknown status", err.Hint)
	}

	// Error() should not include "Hint:" when there's no hint
	errStr := err.Error()
	if strings.Contains(errStr, "Hint:") {
		t.Errorf("Error() = %q, should not contain 'Hint:' when no hint", errStr)
	}
}
