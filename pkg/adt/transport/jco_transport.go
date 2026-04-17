package transport

import (
	"context"
	"encoding/json"
	"fmt"
)

// JcoTransport is the low-level transport interface for communicating with the
// Java JCo sidecar. Two implementations exist:
//   - JcoHttpTransport  — sends JSON via HTTP POST to the sidecar's /rfc-proxy endpoint
//   - AdtJcoStdioTransport — sends JSON via stdin/stdout pipes to the sidecar process
type JcoTransport interface {
	// Send transmits a ProxyRequest and returns the ProxyResponse.
	Send(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error)
}

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

// NewJcoStdioTransport creates a transport that sends requests via STDIO.
func NewJcoStdioTransport(sidecar SidecarIO) *AdtJcoStdioTransport {
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
