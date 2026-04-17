package connection

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// --------------------------------------------------------------------------
// AdtRequest / AdtResponse
// --------------------------------------------------------------------------

func TestResponse_Header(t *testing.T) {
	resp := NewAdtResponseFromMap(200, map[string]string{
		"Content-Type": "application/xml",
		"X-Custom":     "value",
	}, []byte("body"))

	if got := resp.Header("Content-Type"); got != "application/xml" {
		t.Errorf("Header(Content-Type) = %q, want application/xml", got)
	}
	if got := resp.Header("Missing"); got != "" {
		t.Errorf("Header(Missing) = %q, want empty", got)
	}
}

func TestResponse_AllHeaders(t *testing.T) {
	orig := map[string]string{"A": "1", "B": "2"}
	resp := NewAdtResponseFromMap(200, orig, nil)

	headers := resp.AllHeaders()
	if len(headers) != 2 {
		t.Errorf("AllHeaders() returned %d entries, want 2", len(headers))
	}

	// Mutating the returned map should not affect the response.
	headers["C"] = "3"
	if resp.Header("C") != "" {
		t.Error("AllHeaders() returned mutable reference to internal map")
	}
}

func TestNewResponse_FromHTTPHeader(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Type", "text/plain")
	h.Add("X-Multi", "val1")

	resp := NewAdtResponse(201, h, []byte("ok"))
	if resp.StatusCode != 201 {
		t.Errorf("StatusCode = %d, want 201", resp.StatusCode)
	}
	if got := resp.Header("Content-Type"); got != "text/plain" {
		t.Errorf("Header(Content-Type) = %q, want text/plain", got)
	}
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func TestIsModifyingMethod(t *testing.T) {
	tests := []struct {
		method string
		want   bool
	}{
		{"GET", false},
		{"HEAD", false},
		{"POST", true},
		{"PUT", true},
		{"DELETE", true},
		{"PATCH", true},
	}
	for _, tt := range tests {
		if got := IsModifyingMethod(tt.method); got != tt.want {
			t.Errorf("IsModifyingMethod(%q) = %v, want %v", tt.method, got, tt.want)
		}
	}
}

func TestExtractContextID(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"sap-contextid=abc123; path=/", "abc123"},
		{"other=x; sap-contextid=def456; Secure", "def456"},
		{"no-match=true", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := ExtractContextID(tt.header); got != tt.want {
			t.Errorf("ExtractContextID(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestBuildURI(t *testing.T) {
	q := url.Values{}
	q.Set("foo", "bar")

	uri, err := BuildURI("/sap/bc/adt/test", q, "001", "EN")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(uri, "sap-client=001") {
		t.Errorf("URI missing sap-client: %s", uri)
	}
	if !strings.Contains(uri, "sap-language=EN") {
		t.Errorf("URI missing sap-language: %s", uri)
	}
	if !strings.Contains(uri, "foo=bar") {
		t.Errorf("URI missing custom query param: %s", uri)
	}
}

func TestBuildFullURL(t *testing.T) {
	u, err := BuildFullURL("https://sap.example.com:44300", "/sap/bc/adt/test", nil, "100", "DE")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(u, "https://sap.example.com:44300/sap/bc/adt/test") {
		t.Errorf("unexpected URL: %s", u)
	}
}

// --------------------------------------------------------------------------
// SessionState
// --------------------------------------------------------------------------

func TestSessionState(t *testing.T) {
	var s SessionState

	if s.GetCSRFToken() != "" {
		t.Error("initial CSRF token should be empty")
	}
	s.SetCSRFToken("tok123")
	if s.GetCSRFToken() != "tok123" {
		t.Errorf("CSRF token = %q, want tok123", s.GetCSRFToken())
	}

	s.SetSessionCookie("sess456")
	if s.GetSessionCookie() != "sess456" {
		t.Errorf("session cookie = %q, want sess456", s.GetSessionCookie())
	}

	s.ClearSession()
	if s.GetCSRFToken() != "" || s.GetSessionCookie() != "" {
		t.Error("ClearSession should reset both token and cookie")
	}
}

func TestSessionState_UpdateFromProxyResponse(t *testing.T) {
	var s SessionState
	headers := map[string]string{
		"Set-Cookie":   "sap-contextid=ctx789; path=/",
		"X-CSRF-Token": "csrf-abc",
	}
	s.UpdateFromProxyResponse(headers)

	if s.GetSessionCookie() != "ctx789" {
		t.Errorf("session cookie = %q, want ctx789", s.GetSessionCookie())
	}
	if s.GetCSRFToken() != "csrf-abc" {
		t.Errorf("CSRF token = %q, want csrf-abc", s.GetCSRFToken())
	}
}

// --------------------------------------------------------------------------
// APIError
// --------------------------------------------------------------------------

func TestAPIError(t *testing.T) {
	err := &APIError{StatusCode: 404, Message: "not found", Path: "/test"}
	if !err.IsNotFound() {
		t.Error("IsNotFound() should be true for 404")
	}
	if err.IsSessionExpired() {
		t.Error("IsSessionExpired() should be false for 404")
	}

	expired := &APIError{StatusCode: 400, Message: "ICMENOSESSION"}
	if !expired.IsSessionExpired() {
		t.Error("IsSessionExpired() should be true for ICMENOSESSION")
	}

	if !IsNotFoundError(err) {
		t.Error("IsNotFoundError should be true")
	}
	if IsNotFoundError(nil) {
		t.Error("IsNotFoundError(nil) should be false")
	}
}

// --------------------------------------------------------------------------
// AdtHttpConnection
// --------------------------------------------------------------------------

func TestHttpConnection_SendRequest_GET(t *testing.T) {
	// Set up a test server that returns a CSRF token on HEAD and XML on GET.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("X-CSRF-Token", "test-token")
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(200)
		w.Write([]byte("<result>ok</result>"))
	}))
	defer server.Close()

	cfg := &HttpConnectionConfig{
		BaseURL:     server.URL,
		Username:    "user",
		Password:    "pass",
		Client:      "001",
		Language:    "EN",
		SessionType: SessionStateful,
	}
	conn := NewHttpConnection(cfg)

	resp, err := conn.SendRequest(context.Background(), &Request{
		Path:   "/sap/bc/adt/test",
		Method: http.MethodGet,
		Accept: "application/xml",
	})
	if err != nil {
		t.Fatalf("SendRequest failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if string(resp.Body) != "<result>ok</result>" {
		t.Errorf("Body = %q, want <result>ok</result>", string(resp.Body))
	}
}

func TestHttpConnection_SendRequest_POST_WithCSRF(t *testing.T) {
	csrfToken := "server-csrf-token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("X-CSRF-Token", csrfToken)
			return
		}
		// POST must include the CSRF token.
		if r.Method == http.MethodPost {
			if got := r.Header.Get("X-CSRF-Token"); got != csrfToken {
				w.WriteHeader(403)
				w.Write([]byte("missing CSRF token"))
				return
			}
			body, _ := io.ReadAll(r.Body)
			w.WriteHeader(200)
			w.Write(body)
			return
		}
	}))
	defer server.Close()

	cfg := &HttpConnectionConfig{
		BaseURL:  server.URL,
		Username: "user",
		Password: "pass",
		Client:   "001",
	}
	conn := NewHttpConnection(cfg)

	resp, err := conn.SendRequest(context.Background(), &Request{
		Path:        "/sap/bc/adt/test",
		Method:      http.MethodPost,
		Body:        []byte("hello"),
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatalf("SendRequest POST failed: %v", err)
	}
	if string(resp.Body) != "hello" {
		t.Errorf("POST Body = %q, want hello", string(resp.Body))
	}
}

func TestHttpConnection_Ping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-CSRF-Token", "ping-token")
	}))
	defer server.Close()

	conn := NewHttpConnection(&HttpConnectionConfig{BaseURL: server.URL, Username: "u", Password: "p"})
	if err := conn.Ping(context.Background()); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if conn.GetCSRFToken() != "ping-token" {
		t.Errorf("CSRF token after ping = %q, want ping-token", conn.GetCSRFToken())
	}
}

func TestHttpConnection_Close(t *testing.T) {
	conn := NewHttpConnection(&HttpConnectionConfig{BaseURL: "http://localhost"})
	if err := conn.Close(); err != nil {
		t.Fatalf("Close should be a no-op for HTTP: %v", err)
	}
}
