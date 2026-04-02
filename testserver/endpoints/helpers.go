package endpoints

import (
	"fmt"
	"net/http"
	"strings"
)

// xmlResponse writes an XML body with status 200.
func xmlResponse(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, body)
}

// textResponse writes a plain-text body with status 200.
func textResponse(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, body)
}

// noContent writes a 204 No Content response.
func noContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// created writes a 201 Created response with a Location header.
func created(w http.ResponseWriter, location string) {
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}

// lockHandleFor generates a deterministic but unique-looking lock handle.
func lockHandleFor(objectURL string) string {
	h := uint32(2166136261)
	for _, c := range objectURL {
		h ^= uint32(c)
		h *= 16777619
	}
	return fmt.Sprintf("TS%016X", h)
}

// objectNameFromURL extracts the last meaningful path segment (before /source, /objectstructure, etc.)
func objectNameFromURL(r *http.Request) string {
	p := r.URL.Path
	// strip trailing /source/main or /objectstructure
	for _, suffix := range []string{"/source/main", "/objectstructure", "/includes/definitions",
		"/includes/implementations", "/includes/macros", "/includes/testclasses"} {
		p = strings.TrimSuffix(p, suffix)
	}
	parts := strings.Split(p, "/")
	return strings.ToUpper(parts[len(parts)-1])
}
