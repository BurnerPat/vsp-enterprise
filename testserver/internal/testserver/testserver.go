package testserver

import (
	"fmt"
	"log"
	"net/http"
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

	handler := authMiddleware(*user, *password,
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
