package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// JcoTransport is the low-level transport interface for communicating with the
// Java JCo sidecar. Two implementations exist:
//   - JcoHttpTransport  — sends JSON via HTTP POST to the sidecar's /rfc-proxy endpoint
//   - AdtJcoStdioTransport — sends JSON via stdin/stdout pipes to the sidecar process
type JcoTransport interface {
	// Send transmits a ProxyRequest and returns the ProxyResponse.
	Send(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error)

	// Close releases resources held by the transport.
	Close() error
}

// --------------------------------------------------------------------------
// AdtJcoHttpTransport
// --------------------------------------------------------------------------

// JcoHttpTransport implements JcoTransport by POSTing JSON to the
// sidecar's /rfc-proxy HTTP endpoint.
type JcoHttpTransport struct {
	sidecarURL string
	httpClient *http.Client
}

var _ JcoTransport = (*JcoHttpTransport)(nil)

// NewJcoHttpTransport creates a transport that sends requests to the sidecar over HTTP.
func NewJcoHttpTransport(sidecarURL string) *JcoHttpTransport {
	return &JcoHttpTransport{
		sidecarURL: sidecarURL,
		httpClient: &http.Client{}, // no hard timeout — context deadline controls per-request
	}
}

func (t *JcoHttpTransport) Send(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling proxy request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.sidecarURL+"/rfc-proxy", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating sidecar request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sidecar request failed (is the sidecar running at %s?): %w", t.sidecarURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading sidecar response: %w", err)
	}

	// The sidecar itself should always return 200; the SAP status is inside the JSON.
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sidecar returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var proxyResp ProxyResponse
	if err := json.Unmarshal(respBody, &proxyResp); err != nil {
		return nil, fmt.Errorf("parsing sidecar response: %w", err)
	}
	return &proxyResp, nil
}

func (t *JcoHttpTransport) Close() error { return nil }

// --------------------------------------------------------------------------
// AdtJcoStdioTransport
// --------------------------------------------------------------------------

// SidecarIO is the minimal interface for communicating with the sidecar via
// stdin/stdout. It is satisfied by SidecarManager.
type SidecarIO interface {
	SendSTDIO(msg map[string]interface{}) (map[string]interface{}, error)
}

// AdtJcoStdioTransport implements JcoTransport by exchanging newline-delimited
// JSON messages with the sidecar process over stdin/stdout.
type AdtJcoStdioTransport struct {
	sidecar SidecarIO
	nextID  int64
}

var _ JcoTransport = (*AdtJcoStdioTransport)(nil)

// NewAdtJcoStdioTransport creates a transport that sends requests via STDIO.
func NewAdtJcoStdioTransport(sidecar SidecarIO) *AdtJcoStdioTransport {
	return &AdtJcoStdioTransport{sidecar: sidecar}
}

func (t *AdtJcoStdioTransport) Send(_ context.Context, req *ProxyRequest) (*ProxyResponse, error) {
	t.nextID++
	msg := map[string]interface{}{
		"id":      fmt.Sprintf("%d", t.nextID),
		"type":    "proxy",
		"request": req,
	}

	resp, err := t.sidecar.SendSTDIO(msg)
	if err != nil {
		return nil, fmt.Errorf("STDIO proxy request failed: %w", err)
	}

	respData, ok := resp["response"]
	if !ok {
		return nil, fmt.Errorf("STDIO response missing 'response' field")
	}

	// Re-marshal and unmarshal to get the typed ProxyResponse.
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

func (t *AdtJcoStdioTransport) Close() error { return nil }
