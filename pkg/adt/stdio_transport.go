package adt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
)

// StdioRfcTransport implements Requester by sending JSON messages
// to the Java JCo sidecar via STDIO (stdin/stdout) instead of HTTP.
type StdioRfcTransport struct {
	sidecar *SidecarManager
	config  *Config

	// Session management — forward sap-contextid from proxy responses
	sessionCookie string
	sessionMu     sync.RWMutex

	// CSRF token from proxy responses
	csrfToken string
	csrfMu    sync.RWMutex

	// Concurrency control
	semaphore chan struct{}

	// Request ID counter
	nextID atomic.Int64
}

// NewStdioRfcTransport creates a new STDIO-based RFC transport.
func NewStdioRfcTransport(sidecar *SidecarManager, cfg *Config, maxConcurrent int) *StdioRfcTransport {
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}
	return &StdioRfcTransport{
		sidecar:   sidecar,
		config:    cfg,
		semaphore: make(chan struct{}, maxConcurrent),
	}
}

// Request implements Requester by converting the ADT request into a STDIO JSON message.
func (r *StdioRfcTransport) Request(ctx context.Context, path string, opts *RequestOptions) (*Response, error) {
	if opts == nil {
		opts = &RequestOptions{}
	}
	if opts.Method == "" {
		opts.Method = http.MethodGet
	}

	// Acquire semaphore slot
	select {
	case r.semaphore <- struct{}{}:
		defer func() { <-r.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Build the full URI with query params
	uri, err := r.buildURI(path, opts.Query)
	if err != nil {
		return nil, fmt.Errorf("building URI: %w", err)
	}

	// Build proxy request headers
	headers := make(map[string]string)

	accept := opts.Accept
	if accept == "" {
		accept = "*/*"
	}
	headers["Accept"] = accept

	if opts.Body != nil {
		ct := opts.ContentType
		if ct == "" {
			ct = "application/xml"
		}
		headers["Content-Type"] = ct
	}

	for k, v := range opts.Headers {
		headers[k] = v
	}

	switch r.config.SessionType {
	case SessionStateful:
		headers["X-sap-adt-sessiontype"] = "stateful"
	case SessionStateless:
		headers["X-sap-adt-sessiontype"] = "stateless"
	}

	if cookie := r.getSessionCookie(); cookie != "" {
		headers["Cookie"] = "sap-contextid=" + cookie
	}

	if isModifyingMethod(opts.Method) {
		if token := r.getCSRFToken(); token != "" {
			headers["X-CSRF-Token"] = token
		}
	}

	// Build the proxy request
	proxyReq := ProxyRequest{
		Method:  opts.Method,
		URI:     uri,
		Headers: headers,
	}
	if opts.Body != nil {
		proxyReq.Body = string(opts.Body)
	}

	// Send via STDIO
	proxyResp, err := r.sendToSidecar(ctx, &proxyReq)
	if err != nil {
		return nil, err
	}

	// Extract session cookie
	if cookieHeader, ok := proxyResp.Headers["Set-Cookie"]; ok {
		if id := extractContextID(cookieHeader); id != "" {
			r.setSessionCookie(id)
		}
	}
	if cookieHeader, ok := proxyResp.Headers["set-cookie"]; ok {
		if id := extractContextID(cookieHeader); id != "" {
			r.setSessionCookie(id)
		}
	}

	// Extract CSRF token
	if token, ok := proxyResp.Headers["X-CSRF-Token"]; ok && token != "" && token != "Required" {
		r.setCSRFToken(token)
	}
	if token, ok := proxyResp.Headers["x-csrf-token"]; ok && token != "" && token != "Required" {
		r.setCSRFToken(token)
	}

	// Convert to Response
	respHeaders := http.Header{}
	for k, v := range proxyResp.Headers {
		respHeaders.Set(k, v)
	}

	if proxyResp.StatusCode >= 400 {
		return nil, &APIError{
			StatusCode: proxyResp.StatusCode,
			Message:    proxyResp.Body,
			Path:       path,
		}
	}

	return &Response{
		StatusCode: proxyResp.StatusCode,
		Headers:    respHeaders,
		Body:       []byte(proxyResp.Body),
	}, nil
}

// sendToSidecar sends a proxy request via STDIO and parses the response.
func (r *StdioRfcTransport) sendToSidecar(ctx context.Context, proxyReq *ProxyRequest) (*ProxyResponse, error) {
	id := fmt.Sprintf("%d", r.nextID.Add(1))

	msg := map[string]interface{}{
		"id":      id,
		"type":    "proxy",
		"request": proxyReq,
	}

	resp, err := r.sidecar.SendSTDIO(msg)
	if err != nil {
		return nil, fmt.Errorf("STDIO proxy request failed: %w", err)
	}

	// Extract the "response" field and parse as ProxyResponse
	respData, ok := resp["response"]
	if !ok {
		return nil, fmt.Errorf("STDIO response missing 'response' field")
	}

	// Re-marshal and unmarshal to get the typed ProxyResponse
	respJSON, err := json.Marshal(respData)
	if err != nil {
		return nil, fmt.Errorf("marshaling response data: %w", err)
	}

	var proxyResp ProxyResponse
	if err := json.Unmarshal(respJSON, &proxyResp); err != nil {
		return nil, fmt.Errorf("parsing proxy response: %w", err)
	}

	return &proxyResp, nil
}

// buildURI constructs the URI path with query parameters for the proxy request.
func (r *StdioRfcTransport) buildURI(path string, query url.Values) (string, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	u, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	q := u.Query()
	if r.config.Client != "" {
		q.Set("sap-client", r.config.Client)
	}
	if r.config.Language != "" {
		q.Set("sap-language", r.config.Language)
	}
	for k, v := range query {
		for _, val := range v {
			q.Add(k, val)
		}
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// Session cookie accessors
func (r *StdioRfcTransport) getSessionCookie() string {
	r.sessionMu.RLock()
	defer r.sessionMu.RUnlock()
	return r.sessionCookie
}

func (r *StdioRfcTransport) setSessionCookie(cookie string) {
	r.sessionMu.Lock()
	defer r.sessionMu.Unlock()
	r.sessionCookie = cookie
}

// CSRF token accessors
func (r *StdioRfcTransport) getCSRFToken() string {
	r.csrfMu.RLock()
	defer r.csrfMu.RUnlock()
	return r.csrfToken
}

func (r *StdioRfcTransport) setCSRFToken(token string) {
	r.csrfMu.Lock()
	defer r.csrfMu.Unlock()
	r.csrfToken = token
}
