// Package connection provides the connection and request handling layer for ADT.
//
// It defines Request / AdtResponse as the universal request / response types,
// the Connection interface for sending requests, and concrete implementations
// for HTTP (HttpConnection) and JCo sidecar (JcoConnection) connectivity.
package connection

import (
	"net/http"
	"net/url"
)

// Request describes a single request to the SAP ADT backend.
// It is connection-agnostic: both HTTP and JCo/STDIO connections consume it.
type Request struct {
	// Path is the ADT endpoint path (e.g., "/sap/bc/adt/programs/programs/ZTEST/source/main").
	Path string

	// Method is the HTTP method (GET, POST, PUT, DELETE, PATCH). Defaults to GET.
	Method string

	// Query holds optional URL query parameters.
	Query url.Values

	// Body is the optional request body (XML, plain text, JSON, …).
	Body []byte

	// ContentType for the body. Defaults to "application/xml" when Body is set.
	ContentType string

	// Accept header value. Defaults to "*/*".
	Accept string

	// Headers holds additional request headers.
	Headers map[string]string
}

// AdtResponse is the connection-agnostic response returned for every Request.
// It intentionally does NOT expose http.Header so that non-HTTP transports
// (JCo/STDIO) are first-class citizens.
type AdtResponse struct {
	// StatusCode is the HTTP-style status code (200, 404, …).
	// JCo transports map the SAP status onto the same code space.
	StatusCode int

	// Headers are the response headers as a simple string map.
	// For HTTP connections the canonical header names are used.
	// For JCo connections the sidecar-provided headers are forwarded.
	headers map[string]string

	// Body is the raw response body.
	Body []byte
}

// NewAdtResponse creates an AdtResponse, converting http.Header to a flat map.
func NewAdtResponse(statusCode int, httpHeaders http.Header, body []byte) *AdtResponse {
	h := make(map[string]string, len(httpHeaders))
	for k, vals := range httpHeaders {
		if len(vals) > 0 {
			h[k] = vals[0]
		}
	}
	return &AdtResponse{StatusCode: statusCode, headers: h, Body: body}
}

// NewAdtResponseFromMap creates an AdtResponse from a flat header map.
func NewAdtResponseFromMap(statusCode int, headers map[string]string, body []byte) *AdtResponse {
	if headers == nil {
		headers = make(map[string]string)
	}
	return &AdtResponse{StatusCode: statusCode, headers: headers, Body: body}
}

// Header returns the value of the first header matching key (case-sensitive).
func (r *AdtResponse) Header(key string) string {
	if r.headers == nil {
		return ""
	}
	return r.headers[key]
}

// AllHeaders returns a copy of all response headers.
func (r *AdtResponse) AllHeaders() map[string]string {
	out := make(map[string]string, len(r.headers))
	for k, v := range r.headers {
		out[k] = v
	}
	return out
}

// ProxyRequest is the JSON payload sent to the Java sidecar's /rfc-proxy endpoint.
type ProxyRequest struct {
	Method  string            `json:"method"`
	URI     string            `json:"uri"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// ProxyResponse is the JSON payload returned by the Java sidecar's /rfc-proxy endpoint.
type ProxyResponse struct {
	StatusCode   int               `json:"statusCode"`
	ReasonPhrase string            `json:"reasonPhrase"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         string            `json:"body,omitempty"`
}
