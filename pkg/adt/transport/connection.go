package transport

import "context"

// Connection is the central abstraction for communicating with an SAP system.
// Implementations exist for direct HTTP (HttpConnection) and for JCo sidecar
// proxy (JcoConnection, which in turn delegates to an JcoTransport).
type Connection interface {
	// SendRequest sends an Request and returns an AdtResponse.
	// The implementation handles authentication, CSRF tokens, session management,
	// retries on 401/403, and translation to/from the underlying protocol.
	SendRequest(ctx context.Context, req *Request) (*AdtResponse, error)

	// Ping performs a lightweight health-check / keep-alive against the backend.
	// For HTTP connections this fetches a CSRF token; for JCo connections it is a no-op.
	Ping(ctx context.Context) error

	// Close releases resources held by the connection (e.g., stops the JCo sidecar).
	Close() error
}

// Sender is a minimal interface consumed by the service layer.
// It is intentionally small so that service implementations only depend on the
// ability to send a request, not on the full connection lifecycle.
type Sender interface {
	SendRequest(ctx context.Context, req *Request) (*AdtResponse, error)
}
