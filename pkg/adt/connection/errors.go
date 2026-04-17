package connection

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// APIError represents an error returned by the SAP ADT backend.
type APIError struct {
	StatusCode int
	Message    string
	Path       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("ADT API error: status %d at %s: %s", e.StatusCode, e.Path, e.Message)
}

// IsNotFound returns true if the error is a 404 Not Found error.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsSessionExpired returns true if the error indicates an SAP session timeout.
// SAP returns 400 with ICMENOSESSION or "Session Timed Out" when the session expires.
func (e *APIError) IsSessionExpired() bool {
	if e.StatusCode != http.StatusBadRequest {
		return false
	}
	msg := strings.ToLower(e.Message)
	return strings.Contains(msg, "icmenosession") ||
		strings.Contains(msg, "session timed out") ||
		strings.Contains(msg, "session no longer exists")
}

// IsNotFoundError checks if err wraps an APIError with status 404.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsNotFound()
	}
	return false
}

// IsSessionExpiredError checks if err wraps an APIError that indicates session timeout.
func IsSessionExpiredError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsSessionExpired()
	}
	return false
}
