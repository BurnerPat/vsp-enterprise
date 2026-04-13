//go:build testserver

package testserver

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// pathPatternToRegex converts a path pattern like
//
//	/sap/bc/adt/programs/programs/{name}/source/main
//
// into a compiled anchored regexp with named capture groups and returns
// the list of placeholder names in order.
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

// lockHandleFor generates a deterministic lock handle using FNV-1a.
func lockHandleFor(objectURL string) string {
	h := uint32(2166136261)
	for _, c := range objectURL {
		h ^= uint32(c)
		h *= 16777619
	}
	return fmt.Sprintf("TS%016X", h)
}

// patternSource holds a compiled path pattern and its source template.
type patternSource struct {
	regex    *regexp.Regexp
	names    []string
	template string
}

// State holds shared server state accessible by all handlers.
type State struct {
	SysID    string
	Client   string
	User     string
	Password string

	mu             sync.RWMutex
	Sources        map[string]string
	patternSources []patternSource
	Locks          map[string]string
	DataPreview    map[string][]map[string]string
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

// GetSource returns the stored source for path. Exact matches are checked first;
// then compiled path patterns are tried in declaration order.
// ${name} placeholders in pattern templates are substituted with the raw path segment value.
func (s *State) GetSource(path string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if v, ok := s.Sources[path]; ok {
		return v, true
	}

	for _, ps := range s.patternSources {
		m := ps.regex.FindStringSubmatch(path)
		if m == nil {
			continue
		}
		content := ps.template
		for i, name := range ps.names {
			content = strings.ReplaceAll(content, "${"+name+"}", m[i+1])
		}
		return content, true
	}

	return "", false
}

// AddPatternSource compiles pathPattern and registers it with the given source template.
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

// DeleteSource removes the source entry for path.
func (s *State) DeleteSource(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Sources, path)
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

// GetLock returns the lock handle for objectURL, or ("", false) if not locked.
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
