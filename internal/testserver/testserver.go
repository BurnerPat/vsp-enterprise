//go:build testserver

package testserver

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"
)

// csrfToken is the static token issued and validated by the server.
// Clients must send X-CSRF-Token: Fetch on a HEAD/GET to /core/discovery
// before any mutating request.
const csrfToken = "testserver-csrf-token-001"

func StartTestServer(sysID *string, client *string, user *string, password *string, globs []string, port *int) {
	state := NewState(*sysID, *client, *user, *password)
	routes, err := loadGlobs(globs, state)
	if err != nil {
		log.Fatalf("loading fixtures: %v", err)
	}
	log.Printf("Loaded %d routes", len(routes))

	handler := loggingMiddleware(
		authMiddleware(*user, *password,
			csrfMiddleware(csrfToken,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					rt, vars := matchRoute(routes, r)
					if rt == nil {
						http.NotFound(w, r)
						return
					}
					serveRoute(w, r, rt, vars, state, csrfToken)
				}),
			),
		),
	)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("ADT test server listening on http://localhost%s  (sys=%s  client=%s  user=%s)",
		addr, *sysID, *client, *user)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// authMiddleware rejects requests whose Basic Auth credentials do not match.
func authMiddleware(user, password string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || strings.ToLower(u) != strings.ToLower(user) || p != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="ADT Test Server"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// csrfMiddleware validates X-CSRF-Token on mutating requests (POST, PUT, DELETE, PATCH).
// Safe methods (GET, HEAD, OPTIONS) pass through without validation.
func csrfMiddleware(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("X-CSRF-Token") != token {
			http.Error(w, "CSRF token missing or invalid", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture the response body for logging.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func logHttp(direction string, info string, headers http.Header, body []byte) {
	log.Printf("%s: %s", direction, info)

	indent1 := strings.Repeat(" ", utf8.RuneCountInString(direction))

	log.Printf("%sHeaders:", indent1)

	for key, values := range headers {
		for _, value := range values {
			log.Printf("%s  - %s: %s", indent1, key, value)
		}
	}

	if len(body) > 0 {
		log.Printf("%sBody:", indent1)

		strBody := string(body)
		lines := strings.Split(strBody, "\n")

		for _, line := range lines {
			log.Printf("%s  %s", indent1, line)
		}
	}
}

// loggingMiddleware logs all requests and responses with full details.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read request body
		var requestBody []byte
		if r.Body != nil {
			requestBody, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Log request
		logHttp(">>>", fmt.Sprintf("REQUEST: %s %s", r.Method, r.URL.String()), r.Header, requestBody)

		// Wrap response writer to capture response
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           &bytes.Buffer{},
		}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Log response
		logHttp("<<<", fmt.Sprintf("RESPONSE: %d", wrapped.statusCode), wrapped.ResponseWriter.Header(), wrapped.body.Bytes())
	})
}
