//go:build testserver

package testserver

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Route represents a single YAML route entry after compilation.
type Route struct {
	Method   string      `yaml:"method"`
	Path     string      `yaml:"path"`
	Match    MatchConds  `yaml:"match"`
	Action   string      `yaml:"action"`
	Response ResponseDef `yaml:"response"`

	// compiled fields — not from YAML
	pathRegex   *regexp.Regexp
	pathNames   []string
	isExact     bool
	specificity int
}

// MatchConds holds optional matching conditions.
// In match.query, a value of the form {name} captures the actual query-param value
// into vars["name"] instead of doing an exact match. An absent or empty param is a non-match.
type MatchConds struct {
	Query       map[string]string `yaml:"query"`
	ContentType string            `yaml:"content-type"`
	Accept      string            `yaml:"accept"`
	Headers     map[string]string `yaml:"headers"`
}

// ResponseDef describes the HTTP response to send.
type ResponseDef struct {
	Status      int               `yaml:"status"`
	ContentType string            `yaml:"content-type"`
	Headers     map[string]string `yaml:"headers"`
	Body        string            `yaml:"body"`
}

// candidate is an intermediate result during route matching.
type candidate struct {
	route *Route
	vars  map[string]string
}

// compileRoute pre-computes the path regex, capture names, and specificity score.
// Must be called once on each Route after loading from YAML.
func compileRoute(r *Route) error {
	if !strings.Contains(r.Path, "{") {
		r.isExact = true
	} else {
		re, names, err := pathPatternToRegex(r.Path)
		if err != nil {
			return err
		}
		r.pathRegex = re
		r.pathNames = names
	}

	score := 0
	if len(r.Match.Query) > 0 {
		score++
	}
	if r.Match.ContentType != "" {
		score++
	}
	if r.Match.Accept != "" {
		score++
	}
	if len(r.Match.Headers) > 0 {
		score++
	}
	r.specificity = score
	return nil
}

// isCaptureValue reports whether v is a query capture pattern like {name},
// returning the placeholder name without braces.
func isCaptureValue(v string) (string, bool) {
	if len(v) > 2 && v[0] == '{' && v[len(v)-1] == '}' {
		return v[1 : len(v)-1], true
	}
	return "", false
}

// matchConditions checks all Match fields against the request.
// Capture patterns in match.query bind to vars on success.
// Returns false if any condition fails.
func matchConditions(rt *Route, r *http.Request, vars map[string]string) bool {
	q := r.URL.Query()
	for k, v := range rt.Match.Query {
		actual := q.Get(k)
		if capName, isCapture := isCaptureValue(v); isCapture {
			if actual == "" {
				return false // absent or empty param is a non-match
			}
			vars[capName] = actual
		} else {
			if actual != v {
				return false
			}
		}
	}

	if rt.Match.ContentType != "" {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), rt.Match.ContentType) {
			return false
		}
	}

	if rt.Match.Accept != "" {
		if !strings.Contains(r.Header.Get("Accept"), rt.Match.Accept) {
			return false
		}
	}

	for k, v := range rt.Match.Headers {
		if r.Header.Get(k) != v {
			return false
		}
	}

	return true
}

// matchRoute finds the best matching route for the request.
// Exact path matches are preferred over pattern matches; among equals the highest
// specificity score wins; declaration order breaks ties.
func matchRoute(routes []Route, r *http.Request) (*Route, map[string]string) {
	method := r.Method
	path := r.URL.Path

	var exactCandidates []candidate
	var patternCandidates []candidate

	for i := range routes {
		rt := &routes[i]
		if !strings.EqualFold(rt.Method, method) {
			continue
		}

		vars := make(map[string]string)

		if rt.isExact {
			if rt.Path != path {
				continue
			}
		} else {
			m := rt.pathRegex.FindStringSubmatch(path)
			if m == nil {
				continue
			}
			for j, name := range rt.pathNames {
				vars[name] = m[j+1]
			}
		}

		if !matchConditions(rt, r, vars) {
			continue
		}

		if rt.isExact {
			exactCandidates = append(exactCandidates, candidate{rt, vars})
		} else {
			patternCandidates = append(patternCandidates, candidate{rt, vars})
		}
	}

	best := pickBest(exactCandidates)
	if best == nil {
		best = pickBest(patternCandidates)
	}
	if best == nil {
		return nil, nil
	}
	return best.route, best.vars
}

// pickBest returns the candidate with the highest specificity score, or nil if empty.
// The first candidate wins among ties (declaration order).
func pickBest(cs []candidate) *candidate {
	if len(cs) == 0 {
		return nil
	}
	best := &cs[0]
	for i := 1; i < len(cs); i++ {
		if cs[i].route.specificity > best.route.specificity {
			best = &cs[i]
		}
	}
	return best
}

// generateGUID returns a random 32-character lowercase hex string (no dashes).
func generateGUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// substitute replaces ${name} tokens in s using vars, plus two special tokens:
//   - ${csrf-token} → the static CSRF constant
//   - ${guid}       → a per-request random hex ID (same value everywhere in one response)
func substitute(s string, vars map[string]string, csrfToken, guid string) string {
	s = strings.ReplaceAll(s, "${csrf-token}", csrfToken)
	s = strings.ReplaceAll(s, "${guid}", guid)
	for k, v := range vars {
		s = strings.ReplaceAll(s, "${"+k+"}", v)
	}
	return s
}

// serveRoute writes the HTTP response for a matched route.
// guid is generated once per request so that ${guid} is consistent across headers and body.
func serveRoute(w http.ResponseWriter, r *http.Request, rt *Route, vars map[string]string, state *State, csrfToken string) {
	guid := generateGUID()
	sub := func(s string) string {
		return substitute(s, vars, csrfToken, guid)
	}

	if rt.Action != "" {
		dispatchAction(w, r, rt, vars, state, csrfToken, guid)
		return
	}

	status := rt.Response.Status
	if status == 0 {
		status = http.StatusOK
	}
	if rt.Response.ContentType != "" {
		w.Header().Set("Content-Type", rt.Response.ContentType)
	}
	for k, v := range rt.Response.Headers {
		w.Header().Set(k, sub(v))
	}
	w.WriteHeader(status)
	if rt.Response.Body != "" {
		fmt.Fprint(w, sub(rt.Response.Body))
	}
}
