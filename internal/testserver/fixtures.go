//go:build testserver

package testserver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RouteFile mirrors the top-level YAML fixture format.
// A single file may contain any combination of routes, sources, locks, and datapreview.
type RouteFile struct {
	Routes      []Route                        `yaml:"routes"`
	Sources     map[string]string              `yaml:"sources"`
	Locks       map[string]string              `yaml:"locks"`
	DataPreview map[string][]map[string]string `yaml:"datapreview"`
}

// loadGlobs expands each glob pattern via filepath.Glob, loads and merges all
// matching files in order, and returns the combined route list.
// At least one file must match across all patterns; the server fails fast otherwise.
func loadGlobs(patterns []string, state *State) ([]Route, error) {
	var routes []Route
	loaded := 0

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}
		for _, path := range matches {
			rs, err := loadFile(path, state)
			if err != nil {
				return nil, fmt.Errorf("loading %q: %w", path, err)
			}
			routes = append(routes, rs...)
			loaded++
		}
	}

	if loaded == 0 {
		return nil, fmt.Errorf("no fixture files matched any of the patterns: %s",
			strings.Join(patterns, ", "))
	}

	return routes, nil
}

// loadFile reads a single YAML fixture file, seeds state from the sources/locks/datapreview
// sections, compiles routes, and returns them.
func loadFile(path string, state *State) ([]Route, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading: %w", err)
	}

	var f RouteFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	// Seed state — sources (pattern keys use {name} syntax), locks, datapreview.
	for p, src := range f.Sources {
		if strings.Contains(p, "{") {
			if err := state.AddPatternSource(p, src); err != nil {
				return nil, fmt.Errorf("compiling source pattern %q: %w", p, err)
			}
		} else {
			state.SetSource(p, src)
		}
	}
	for objectURL, handle := range f.Locks {
		state.AcquireLock(objectURL, handle)
	}
	for query, rows := range f.DataPreview {
		state.SetDataPreview(query, rows)
	}

	// Compile routes.
	routes := make([]Route, len(f.Routes))
	for i, r := range f.Routes {
		routes[i] = r
		if err := compileRoute(&routes[i]); err != nil {
			return nil, fmt.Errorf("route %d (%s %s): %w", i+1, r.Method, r.Path, err)
		}
	}

	return routes, nil
}
