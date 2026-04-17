package transport

import (
	"context"
	"fmt"
	"net/http"
)

// JcoConnectionConfig holds the parameters needed to create an JcoConnection.
type JcoConnectionConfig struct {
	Client        string // SAP client number
	Language      string // SAP language
	SessionType   SessionType
	MaxConcurrent int // Max concurrent RFC requests (default 5)
}

// SidecarLifecycle is the minimal interface the JCo connection uses to manage
// the sidecar process. Satisfied by SidecarManager.
type SidecarLifecycle interface {
	Stop() error
}

// JcoConnection implements Connection by delegating to an JcoTransport.
// It owns:
//   - An JcoTransport (HTTP or STDIO) for the low-level sidecar communication
//   - The shared proxy-request building and response-parsing logic
//   - The SidecarLifecycle for stopping the sidecar on Close()
//   - Shared SessionState for CSRF and session-cookie management
//   - A concurrency semaphore
type JcoConnection struct {
	transport JcoTransport
	sidecar   SidecarLifecycle
	config    *JcoConnectionConfig
	SessionState
	semaphore chan struct{}
}

var _ Connection = (*JcoConnection)(nil)

// NewJcoConnection creates a JCo-based connection.
// The sidecar parameter may be nil if the sidecar is managed externally.
func NewJcoConnection(jcoTransport JcoTransport, sidecar SidecarLifecycle, cfg *JcoConnectionConfig) *JcoConnection {
	maxConcurrent := cfg.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}
	return &JcoConnection{
		transport: jcoTransport,
		sidecar:   sidecar,
		config:    cfg,
		semaphore: make(chan struct{}, maxConcurrent),
	}
}

// SendRequest converts an Request into a ProxyRequest, sends it through the
// JcoTransport, and converts the ProxyResponse back into an AdtResponse.
// This centralises logic that was previously duplicated across RfcTransport and
// StdioRfcTransport.
func (c *JcoConnection) SendRequest(ctx context.Context, req *Request) (*AdtResponse, error) {
	if req.Method == "" {
		req.Method = http.MethodGet
	}

	// Acquire semaphore slot.
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Build the URI with query params.
	uri, err := BuildURI(req.Path, req.Query, c.config.Client, c.config.Language)
	if err != nil {
		return nil, fmt.Errorf("building URI: %w", err)
	}

	// Build headers.
	headers := c.buildHeaders(req)

	// Build proxy request.
	proxyReq := &ProxyRequest{
		Method:  req.Method,
		URI:     uri,
		Headers: headers,
	}
	if req.Body != nil {
		proxyReq.Body = string(req.Body)
	}

	// Send via transport.
	proxyResp, err := c.transport.Send(ctx, proxyReq)
	if err != nil {
		return nil, err
	}

	// Update session state from response.
	c.SessionState.UpdateFromProxyResponse(proxyResp.Headers)

	// Check for error status codes.
	if proxyResp.StatusCode >= 400 {
		return nil, &APIError{
			StatusCode: proxyResp.StatusCode,
			Message:    proxyResp.Body,
			Path:       req.Path,
		}
	}

	return NewAdtResponseFromMap(proxyResp.StatusCode, proxyResp.Headers, []byte(proxyResp.Body)), nil
}

// Ping is a no-op for JCo connections (sessions are managed by the sidecar).
func (c *JcoConnection) Ping(_ context.Context) error { return nil }

// Close stops the owned sidecar and releases the JCo transport.
func (c *JcoConnection) Close() error {
	var firstErr error
	if c.sidecar != nil {
		if err := c.sidecar.Stop(); err != nil {
			firstErr = fmt.Errorf("stopping sidecar: %w", err)
		}
	}
	return firstErr
}

// Sidecar returns the managed sidecar lifecycle, or nil.
// This is useful during server bootstrapping for health-checks or direct RFC calls.
func (c *JcoConnection) Sidecar() SidecarLifecycle {
	return c.sidecar
}

// Transport returns the underlying JCo transport.
func (c *JcoConnection) Transport() JcoTransport {
	return c.transport
}

// --------------------------------------------------------------------------
// internal helpers
// --------------------------------------------------------------------------

func (c *JcoConnection) buildHeaders(req *Request) map[string]string {
	headers := make(map[string]string)

	accept := req.Accept
	if accept == "" {
		accept = "*/*"
	}
	headers["Accept"] = accept

	if req.Body != nil {
		ct := req.ContentType
		if ct == "" {
			ct = "application/xml"
		}
		headers["Content-Type"] = ct
	}

	for k, v := range req.Headers {
		headers[k] = v
	}

	switch c.config.SessionType {
	case SessionStateful:
		headers["X-sap-adt-sessiontype"] = "stateful"
	case SessionStateless:
		headers["X-sap-adt-sessiontype"] = "stateless"
	}

	if cookie := c.GetSessionCookie(); cookie != "" {
		headers["Cookie"] = "sap-contextid=" + cookie
	}

	if IsModifyingMethod(req.Method) {
		if token := c.GetCSRFToken(); token != "" {
			headers["X-CSRF-Token"] = token
		}
	}

	return headers
}

// IsJcoConnection returns true if the given Connection is a *JcoConnection.
// Helper for callers that need to detect the connection type (e.g., IsRfcMode).
func IsJcoConnection(conn Connection) bool {
	_, ok := conn.(*JcoConnection)
	return ok
}

// GetSidecarFromConnection returns the SidecarLifecycle from a JCo connection, or nil.
// Helper for bootstrap code that needs access to the sidecar for health-checks or RFC calls.
func GetSidecarFromConnection(conn Connection) SidecarLifecycle {
	if jco, ok := conn.(*JcoConnection); ok {
		return jco.Sidecar()
	}
	return nil
}
