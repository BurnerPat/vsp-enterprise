package adt

import "github.com/oisee/vibing-steampunk/pkg/adt/connection"

// APIError is a type alias for connection.APIError, re-exported for backward
// compatibility so that callers can continue to use adt.APIError.
type APIError = connection.APIError

// IsNotFoundError checks if err wraps an APIError with status 404.
func IsNotFoundError(err error) bool {
	return connection.IsNotFoundError(err)
}

// IsSessionExpiredError checks if err wraps an APIError that indicates session timeout.
func IsSessionExpiredError(err error) bool {
	return connection.IsSessionExpiredError(err)
}
