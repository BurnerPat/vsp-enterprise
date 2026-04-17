package adt

import (
	"context"

	"github.com/oisee/vibing-steampunk/pkg/adt/connection"
)

// ClientInterface is the primary interface for interacting with an SAP ADT backend.
// It provides:
//   - Connect / Close for lifecycle management
//   - SendRequest for low-level request dispatch
//   - Domain-grouped service accessors (Source, DevTools, …)
//
// The concrete implementation is *Client (in client.go).
type ClientInterface interface {
	// Connect eagerly validates credentials and establishes the session.
	// For HTTP connections this fetches a CSRF token; for JCo it is currently a no-op.
	Connect(ctx context.Context) error

	// Close releases all resources: stops the keep-alive goroutine, closes the
	// underlying AdtConnection (which in turn stops the JCo sidecar if applicable).
	Close() error

	// SendRequest dispatches a single request through the underlying connection.
	// Most callers should use the service accessors instead.
	SendRequest(ctx context.Context, req *connection.Request) (*connection.AdtResponse, error)

	// Connection returns the underlying AdtConnection.
	// Useful for advanced callers that need connection-level access.
	Connection() connection.Connection

	// GetConfig returns the client configuration.
	GetConfig() *Config

	// Safety returns the safety configuration.
	Safety() *SafetyConfig
}

// HttpConfigFromConfig converts an adt.Config into a connection.HttpConnectionConfig.
func HttpConfigFromConfig(cfg *Config) *connection.HttpConnectionConfig {
	return &connection.HttpConnectionConfig{
		BaseURL:            cfg.BaseURL,
		Username:           cfg.Username,
		Password:           cfg.Password,
		Client:             cfg.Client,
		Language:           cfg.Language,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		SessionType:        connection.SessionType(cfg.SessionType),
		Timeout:            cfg.Timeout,
		Cookies:            cfg.Cookies,
	}
}
