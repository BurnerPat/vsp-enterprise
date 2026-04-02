// Package endpoints contains handlers for each ADT REST endpoint group.
package endpoints

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// patternSource holds a compiled path pattern and its source template.
// Placeholders use {name} in the path and ${name} in the template body.
type patternSource struct {
	regex    *regexp.Regexp
	names    []string // capture group names, in order
	template string
}

// pathPatternToRegex converts a path pattern like
//
//	/sap/bc/adt/programs/programs/{name}/source/main
//
// into a compiled anchored regexp with named capture groups and returns
// the list of placeholder names in the order they appear.
func pathPatternToRegex(pattern string) (*regexp.Regexp, []string, error) {
	placeholderRe := regexp.MustCompile(`\{(\w+)\}`)

	var names []string
	var regexParts []string
	last := 0

	for _, loc := range placeholderRe.FindAllStringSubmatchIndex(pattern, -1) {
		regexParts = append(regexParts, regexp.QuoteMeta(pattern[last:loc[0]]))
		name := pattern[loc[2]:loc[3]]
		names = append(names, name)
		regexParts = append(regexParts, `(?P<`+name+`>[^/]+)`)
		last = loc[1]
	}
	regexParts = append(regexParts, regexp.QuoteMeta(pattern[last:]))

	re, err := regexp.Compile("^" + strings.Join(regexParts, "") + "$")
	if err != nil {
		return nil, nil, fmt.Errorf("compiling pattern %q: %w", pattern, err)
	}
	return re, names, nil
}

// State holds shared server state accessible by all endpoint handlers.
type State struct {
	SysID    string
	Client   string
	User     string
	Password string

	mu             sync.RWMutex
	Sources        map[string]string              // ADT path -> source content (exact)
	patternSources []patternSource                // ADT path patterns -> source templates
	Locks          map[string]string              // object URL -> lock handle
	DataPreview    map[string][]map[string]string // SQL query -> result rows
}

// NewState creates an initialised State for the given system credentials.
func NewState(sysID, client, user, password string) *State {
	return &State{
		SysID:       sysID,
		Client:      client,
		User:        user,
		Password:    password,
		Sources:     make(map[string]string),
		Locks:       make(map[string]string),
		DataPreview: make(map[string][]map[string]string),
	}
}

// GetSource returns the stored source for path, or (_, false) if not set.
// Exact matches are checked first; then compiled path patterns are tried in
// declaration order. For pattern matches, ${name} placeholders in the template
// are replaced with the corresponding path segment (uppercased).
func (s *State) GetSource(path string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 1. Exact match
	if v, ok := s.Sources[path]; ok {
		return v, true
	}

	// 2. Pattern match
	for _, ps := range s.patternSources {
		match := ps.regex.FindStringSubmatch(path)
		if match == nil {
			continue
		}
		content := ps.template
		for i, name := range ps.names {
			val := strings.ToUpper(match[i+1])
			content = strings.ReplaceAll(content, "${"+name+"}", val)
		}
		return content, true
	}

	return "", false
}

// AddPatternSource compiles pathPattern (e.g. "/sap/bc/adt/programs/programs/{name}/source/main")
// and registers it with the given source template. Returns an error if the pattern is invalid.
func (s *State) AddPatternSource(pathPattern, template string) error {
	re, names, err := pathPatternToRegex(pathPattern)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.patternSources = append(s.patternSources, patternSource{
		regex:    re,
		names:    names,
		template: template,
	})
	return nil
}

// SetSource stores or replaces the source for path.
func (s *State) SetSource(path, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Sources[path] = content
}

// AcquireLock records a lock handle for objectURL, replacing any prior lock.
func (s *State) AcquireLock(objectURL, handle string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Locks[objectURL] = handle
}

// ReleaseLock removes the lock entry for objectURL.
func (s *State) ReleaseLock(objectURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Locks, objectURL)
}

// GetLock returns the lock handle for objectURL, or (_, false) if not locked.
func (s *State) GetLock(objectURL string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.Locks[objectURL]
	return v, ok
}

// QueryDataPreview looks up a pre-configured result for an SQL query.
func (s *State) QueryDataPreview(sql string) ([]map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, ok := s.DataPreview[sql]
	return rows, ok
}

// SetDataPreview stores a result set for an exact SQL query string.
func (s *State) SetDataPreview(query string, rows []map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DataPreview[query] = rows
}
