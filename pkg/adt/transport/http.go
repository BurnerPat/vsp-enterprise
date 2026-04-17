package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

// SessionType defines how the connection manages server sessions.
type SessionType string

const (
	SessionStateful  SessionType = "stateful"
	SessionStateless SessionType = "stateless"
	SessionKeep      SessionType = "keep"
)

// HttpConnectionConfig holds the parameters needed to create an HttpConnection.
// The root adt.Config is translated into this struct by the Client constructor
// so that the transport package has no dependency on adt.Config.
type HttpConnectionConfig struct {
	BaseURL            string
	Username           string
	Password           string
	Client             string // SAP client number (e.g., "001")
	Language           string // SAP language (e.g., "EN")
	InsecureSkipVerify bool
	SessionType        SessionType
	Timeout            time.Duration
	Cookies            map[string]string // Cookie-based auth (alternative to basic auth)
}

// HasBasicAuth returns true if username and password are both non-empty.
func (c *HttpConnectionConfig) HasBasicAuth() bool {
	return c.Username != "" && c.Password != ""
}

// HTTPDoer is an interface for executing HTTP requests.
// Useful for testing with mock implementations.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// HttpConnection implements Connection for direct HTTP access to SAP ADT.
// It manages CSRF tokens, sessions, basic/cookie authentication, and automatic
// retry on 401/403 and session expiry.
type HttpConnection struct {
	config     *HttpConnectionConfig
	httpClient HTTPDoer
	SessionState
}

// Ensure interface compliance at compile time.
var _ Connection = (*HttpConnection)(nil)

// NewHttpConnection creates a new HTTP-based connection.
func NewHttpConnection(cfg *HttpConnectionConfig) *HttpConnection {
	return &HttpConnection{
		config:     cfg,
		httpClient: newHTTPClient(cfg),
	}
}

// NewHttpConnectionWithClient creates a new HTTP-based connection with a custom HTTP client.
// Useful for testing with mock HTTP clients.
func NewHttpConnectionWithClient(cfg *HttpConnectionConfig, client HTTPDoer) *HttpConnection {
	return &HttpConnection{
		config:     cfg,
		httpClient: client,
	}
}

// SendRequest performs an HTTP request to the ADT API.
func (c *HttpConnection) SendRequest(ctx context.Context, req *Request) (*AdtResponse, error) {
	if req.Method == "" {
		req.Method = http.MethodGet
	}

	// Build URL.
	reqURL, err := BuildFullURL(c.config.BaseURL, req.Path, req.Query, c.config.Client, c.config.Language)
	if err != nil {
		return nil, fmt.Errorf("building URL: %w", err)
	}

	// Create the http.Request.
	var bodyReader io.Reader
	if req.Body != nil {
		bodyReader = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Authentication — basic auth or cookies.
	if c.config.HasBasicAuth() {
		httpReq.SetBasicAuth(c.config.Username, c.config.Password)
	}
	for name, value := range c.config.Cookies {
		httpReq.AddCookie(&http.Cookie{Name: name, Value: value})
	}

	// Default headers.
	c.applyHeaders(httpReq, req)

	// CSRF token for modifying requests.
	if IsModifyingMethod(req.Method) {
		token := c.GetCSRFToken()
		if token == "" {
			if err := c.fetchCSRFToken(ctx); err != nil {
				return nil, fmt.Errorf("fetching CSRF token: %w", err)
			}
			token = c.GetCSRFToken()
		}
		httpReq.Header.Set("X-CSRF-Token", token)
	}

	// Execute.
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	// Handle CSRF token refresh on 403.
	if resp.StatusCode == http.StatusForbidden && IsModifyingMethod(req.Method) {
		if err := c.fetchCSRFToken(ctx); err != nil {
			return nil, fmt.Errorf("refreshing CSRF token: %w", err)
		}
		return c.retryRequest(ctx, req)
	}

	// Store CSRF token from response.
	if token := resp.Header.Get("X-CSRF-Token"); token != "" && token != "Required" {
		c.SetCSRFToken(token)
	}

	// Store session ID.
	if sessionID := c.extractSessionID(resp); sessionID != "" {
		c.SetSessionCookie(sessionID)
	}

	// Error status codes.
	if resp.StatusCode >= 400 {
		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
			Path:       req.Path,
		}

		// Session timeout → refresh and retry.
		if apiErr.IsSessionExpired() {
			c.ClearSession()
			if err := c.fetchCSRFToken(ctx); err != nil {
				return nil, fmt.Errorf("refreshing session after timeout: %w", err)
			}
			return c.retryRequest(ctx, req)
		}

		// 401 Unauthorized → re-authenticate and retry.
		if resp.StatusCode == http.StatusUnauthorized {
			c.ClearSession()
			if err := c.fetchCSRFToken(ctx); err != nil {
				return nil, fmt.Errorf("re-authenticating after 401 on %s: %w (original error: %v)", req.Path, err, apiErr)
			}
			return c.retryRequest(ctx, req)
		}

		return nil, apiErr
	}

	return NewAdtResponse(resp.StatusCode, resp.Header, body), nil
}

// Ping sends a lightweight HEAD request to keep the session alive.
func (c *HttpConnection) Ping(ctx context.Context) error {
	return c.fetchCSRFToken(ctx)
}

// Close is a no-op for HTTP connections (no external resources to release).
func (c *HttpConnection) Close() error {
	return nil
}

// Config returns the connection configuration (useful for service layer access to client/language).
func (c *HttpConnection) Config() *HttpConnectionConfig {
	return c.config
}

// --------------------------------------------------------------------------
// internal helpers
// --------------------------------------------------------------------------

// retryRequest retries a request after CSRF token refresh.
func (c *HttpConnection) retryRequest(ctx context.Context, req *Request) (*AdtResponse, error) {
	reqURL, err := BuildFullURL(c.config.BaseURL, req.Path, req.Query, c.config.Client, c.config.Language)
	if err != nil {
		return nil, fmt.Errorf("building URL: %w", err)
	}

	var bodyReader io.Reader
	if req.Body != nil {
		bodyReader = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if c.config.HasBasicAuth() {
		httpReq.SetBasicAuth(c.config.Username, c.config.Password)
	}
	for name, value := range c.config.Cookies {
		httpReq.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	c.applyHeaders(httpReq, req)
	httpReq.Header.Set("X-CSRF-Token", c.GetCSRFToken())

	if c.config.SessionType == SessionStateful {
		httpReq.Header.Set("X-sap-adt-sessiontype", "stateful")
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing retry request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &APIError{StatusCode: resp.StatusCode, Message: string(body), Path: req.Path}
	}

	return NewAdtResponse(resp.StatusCode, resp.Header, body), nil
}

// fetchCSRFToken retrieves a CSRF token via HEAD /sap/bc/adt/core/discovery.
func (c *HttpConnection) fetchCSRFToken(ctx context.Context) error {
	reqURL, err := BuildFullURL(c.config.BaseURL, "/sap/bc/adt/core/discovery", nil, c.config.Client, c.config.Language)
	if err != nil {
		return fmt.Errorf("building URL: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodHead, reqURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if c.config.HasBasicAuth() {
		httpReq.SetBasicAuth(c.config.Username, c.config.Password)
	}
	for name, value := range c.config.Cookies {
		httpReq.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	httpReq.Header.Set("X-CSRF-Token", "fetch")
	httpReq.Header.Set("Accept", "*/*")

	if c.config.SessionType == SessionStateful {
		httpReq.Header.Set("X-sap-adt-sessiontype", "stateful")
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	token := resp.Header.Get("X-CSRF-Token")
	if token == "" || token == "Required" {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed (401): check username/password")
		case http.StatusForbidden:
			return fmt.Errorf("access forbidden (403): check user authorizations")
		default:
			return fmt.Errorf("no CSRF token in response (HTTP %d)", resp.StatusCode)
		}
	}

	c.SetCSRFToken(token)
	return nil
}

// applyHeaders sets default + custom headers on an http.Request.
func (c *HttpConnection) applyHeaders(httpReq *http.Request, req *Request) {
	accept := req.Accept
	if accept == "" {
		accept = "*/*"
	}
	httpReq.Header.Set("Accept", accept)

	if req.Body != nil {
		ct := req.ContentType
		if ct == "" {
			ct = "application/xml"
		}
		httpReq.Header.Set("Content-Type", ct)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	switch c.config.SessionType {
	case SessionStateful:
		httpReq.Header.Set("X-sap-adt-sessiontype", "stateful")
	case SessionStateless:
		httpReq.Header.Set("X-sap-adt-sessiontype", "stateless")
	}
}

// extractSessionID extracts the session ID from response cookies.
func (c *HttpConnection) extractSessionID(resp *http.Response) string {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "sap-contextid" || cookie.Name == "SAP_SESSIONID" {
			return cookie.Value
		}
	}
	return ""
}

// --------------------------------------------------------------------------
// HTTP client factory
// --------------------------------------------------------------------------

// newHTTPClient creates an *http.Client configured for the given connection config.
func newHTTPClient(cfg *HttpConnectionConfig) *http.Client {
	jar, _ := cookiejar.New(nil)

	base := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec
		},
	}

	var roundTripper http.RoundTripper = base
	if strings.HasPrefix(strings.ToLower(cfg.BaseURL), "http://") {
		roundTripper = &stripSecureCookieTransport{base: base}
	}

	return &http.Client{
		Jar:       jar,
		Transport: roundTripper,
		Timeout:   cfg.Timeout,
	}
}

// stripSecureCookieTransport wraps an http.RoundTripper and removes the Secure
// flag from Set-Cookie headers so that Go's cookie jar persists SAP session
// cookies when connecting over plain HTTP.
type stripSecureCookieTransport struct {
	base http.RoundTripper
}

func (t *stripSecureCookieTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if cookies := resp.Header.Values("Set-Cookie"); len(cookies) > 0 {
		resp.Header.Del("Set-Cookie")
		for _, c := range cookies {
			resp.Header.Add("Set-Cookie", stripSecureFlag(c))
		}
	}
	return resp, err
}

func stripSecureFlag(cookie string) string {
	parts := strings.Split(cookie, ";")
	filtered := parts[:0]
	for _, p := range parts {
		if !strings.EqualFold(strings.TrimSpace(p), "secure") {
			filtered = append(filtered, p)
		}
	}
	return strings.Join(filtered, ";")
}
