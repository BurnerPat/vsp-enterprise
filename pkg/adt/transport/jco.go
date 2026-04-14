package transport

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// JcoConnectionConfig holds the parameters needed to create an AdtJcoConnection.
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

// AdtJcoConnection implements AdtConnection by delegating to an AdtJcoTransport.
// It owns:
//   - An AdtJcoTransport (HTTP or STDIO) for the low-level sidecar communication
//   - The shared proxy-request building and response-parsing logic
//   - The SidecarLifecycle for stopping the sidecar on Close()
//   - Shared SessionState for CSRF and session-cookie management
//   - A concurrency semaphore
type AdtJcoConnection struct {
	transport AdtJcoTransport
	sidecar   SidecarLifecycle
	config    *JcoConnectionConfig
	SessionState
	semaphore chan struct{}
}

var _ AdtConnection = (*AdtJcoConnection)(nil)

// NewAdtJcoConnection creates a JCo-based connection.
// The sidecar parameter may be nil if the sidecar is managed externally.
func NewAdtJcoConnection(jcoTransport AdtJcoTransport, sidecar SidecarLifecycle, cfg *JcoConnectionConfig) *AdtJcoConnection {
	maxConcurrent := cfg.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}
	return &AdtJcoConnection{
		transport: jcoTransport,
		sidecar:   sidecar,
		config:    cfg,
		semaphore: make(chan struct{}, maxConcurrent),
	}
}

// SendRequest converts an AdtRequest into a ProxyRequest, sends it through the
// AdtJcoTransport, and converts the ProxyResponse back into an AdtResponse.
// This centralises logic that was previously duplicated across RfcTransport and
// StdioRfcTransport.
func (c *AdtJcoConnection) SendRequest(ctx context.Context, req *AdtRequest) (*AdtResponse, error) {
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
func (c *AdtJcoConnection) Ping(_ context.Context) error { return nil }

// Close stops the owned sidecar and releases the JCo transport.
func (c *AdtJcoConnection) Close() error {
	var firstErr error
	if err := c.transport.Close(); err != nil && firstErr == nil {
		firstErr = fmt.Errorf("closing JCo transport: %w", err)
	}
	if c.sidecar != nil {
		if err := c.sidecar.Stop(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("stopping sidecar: %w", err)
		}
	}
	return firstErr
}

// Sidecar returns the managed sidecar lifecycle, or nil.
// This is useful during server bootstrapping for health-checks or direct RFC calls.
func (c *AdtJcoConnection) Sidecar() SidecarLifecycle {
	return c.sidecar
}

// Transport returns the underlying JCo transport.
func (c *AdtJcoConnection) Transport() AdtJcoTransport {
	return c.transport
}

// --------------------------------------------------------------------------
// internal helpers
// --------------------------------------------------------------------------

func (c *AdtJcoConnection) buildHeaders(req *AdtRequest) map[string]string {
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

// IsJcoConnection returns true if the given AdtConnection is a *AdtJcoConnection.
// Helper for callers that need to detect the connection type (e.g., IsRfcMode).
func IsJcoConnection(conn AdtConnection) bool {
	_, ok := conn.(*AdtJcoConnection)
	return ok
}

// GetSidecarFromConnection returns the SidecarLifecycle from a JCo connection, or nil.
// Helper for bootstrap code that needs access to the sidecar for health-checks or RFC calls.
func GetSidecarFromConnection(conn AdtConnection) SidecarLifecycle {
	if jco, ok := conn.(*AdtJcoConnection); ok {
		return jco.Sidecar()
	}
	return nil
}

// SidecarURL is a helper to reconstruct the sidecar base URL from a JCo HTTP transport.
func SidecarURL(conn AdtConnection) string {
	jco, ok := conn.(*AdtJcoConnection)
	if !ok {
		return ""
	}
	if ht, ok := jco.Transport().(*AdtJcoHttpTransport); ok {
		return strings.TrimSuffix(ht.sidecarURL, "/")
	}
	return ""
}
