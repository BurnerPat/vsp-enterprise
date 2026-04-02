// Command adt-testserver is a standalone HTTP server that emulates the SAP ADT REST API
// for testing purposes. It requires no SAP system and returns hardcoded XML responses
// with object-name placeholders.
//
// Usage:
//
//	go run ./testserver [flags]
//
// Flags:
//
//	--sys-id    SAP System ID reported by the server (default: TST)
//	--client    SAP client number (default: 001)
//	--user      Expected Basic Auth username (default: developer)
//	--password  Expected Basic Auth password (default: secret)
//	--port      TCP port to listen on (default: 8080)
//	--fixtures  Optional path to a YAML fixtures file
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"adt-testserver/endpoints"
)

// csrfToken is the static token issued and validated by the server.
// Clients must send X-CSRF-Token: Fetch on a HEAD/GET to /core/discovery
// before any mutating request.
const csrfToken = "testserver-csrf-token-001"

func main() {
	sysID := flag.String("sys-id", "TST", "SAP System ID")
	client := flag.String("client", "001", "SAP Client")
	user := flag.String("user", "developer", "SAP Username")
	password := flag.String("password", "secret", "SAP Password")
	port := flag.Int("port", 8080, "HTTP port to listen on")
	fixtures := flag.String("fixtures", "", "Path to YAML fixtures file (optional)")
	flag.Parse()

	state := endpoints.NewState(*sysID, *client, *user, *password)

	if *fixtures != "" {
		if err := loadFixtures(*fixtures, state); err != nil {
			log.Fatalf("loading fixtures %q: %v", *fixtures, err)
		}
		log.Printf("Loaded fixtures from %s", *fixtures)
	}

	mux := http.NewServeMux()

	endpoints.RegisterCore(mux, state, csrfToken)
	endpoints.RegisterPrograms(mux, state)
	endpoints.RegisterOO(mux, state)
	endpoints.RegisterFunctions(mux, state)
	endpoints.RegisterDDIC(mux, state)
	endpoints.RegisterCheckruns(mux, state)
	endpoints.RegisterActivation(mux, state)
	endpoints.RegisterAbapunit(mux, state)
	endpoints.RegisterATC(mux, state)
	endpoints.RegisterRepository(mux, state)
	endpoints.RegisterDatapreview(mux, state)
	endpoints.RegisterRuntime(mux, state)

	// Wrap with auth (outermost) → CSRF → mux
	handler := authMiddleware(*user, *password, csrfMiddleware(csrfToken, mux))

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
		if !ok || u != user || p != password {
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
