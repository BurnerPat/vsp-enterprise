package adt

import (
	"context"

	"github.com/oisee/vibing-steampunk/pkg/adt/transport"
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
	SendRequest(ctx context.Context, req *transport.Request) (*transport.AdtResponse, error)

	// Connection returns the underlying AdtConnection.
	// Useful for advanced callers that need transport-level access.
	Connection() transport.Connection

	// GetConfig returns the client configuration.
	GetConfig() *Config

	// Safety returns the safety configuration.
	Safety() *SafetyConfig
}

// HttpConfigFromConfig converts an adt.Config into a transport.HttpConnectionConfig.
func HttpConfigFromConfig(cfg *Config) *transport.HttpConnectionConfig {
	return &transport.HttpConnectionConfig{
		BaseURL:            cfg.BaseURL,
		Username:           cfg.Username,
		Password:           cfg.Password,
		Client:             cfg.Client,
		Language:           cfg.Language,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		SessionType:        transport.SessionType(cfg.SessionType),
		Timeout:            cfg.Timeout,
		Cookies:            cfg.Cookies,
	}
}
