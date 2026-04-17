package transport

import "sync"

// SessionState holds CSRF token and session cookie state.
// It is embedded by both HttpConnection and JcoConnection to
// eliminate the session-management duplication that previously existed
// across Transport, RfcTransport, and StdioRfcTransport.
type SessionState struct {
	csrfToken string
	csrfMu    sync.RWMutex

	sessionCookie string
	sessionMu     sync.RWMutex
}

// GetCSRFToken returns the current CSRF token.
func (s *SessionState) GetCSRFToken() string {
	s.csrfMu.RLock()
	defer s.csrfMu.RUnlock()
	return s.csrfToken
}

// SetCSRFToken stores a new CSRF token.
func (s *SessionState) SetCSRFToken(token string) {
	s.csrfMu.Lock()
	defer s.csrfMu.Unlock()
	s.csrfToken = token
}

// GetSessionCookie returns the current session cookie value.
func (s *SessionState) GetSessionCookie() string {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()
	return s.sessionCookie
}

// SetSessionCookie stores a new session cookie value.
func (s *SessionState) SetSessionCookie(cookie string) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	s.sessionCookie = cookie
}

// ClearSession resets both CSRF token and session cookie.
func (s *SessionState) ClearSession() {
	s.SetCSRFToken("")
	s.SetSessionCookie("")
}

// UpdateFromProxyResponse extracts CSRF token and session cookie from a
// ProxyResponse header map (used by JCo transports).
func (s *SessionState) UpdateFromProxyResponse(headers map[string]string) {
	// Extract session cookie from Set-Cookie header.
	for _, key := range []string{"Set-Cookie", "set-cookie"} {
		if cookieHeader, ok := headers[key]; ok {
			if id := ExtractContextID(cookieHeader); id != "" {
				s.SetSessionCookie(id)
			}
		}
	}

	// Extract CSRF token.
	for _, key := range []string{"X-CSRF-Token", "x-csrf-token"} {
		if token, ok := headers[key]; ok && token != "" && token != "Required" {
			s.SetCSRFToken(token)
		}
	}
}
