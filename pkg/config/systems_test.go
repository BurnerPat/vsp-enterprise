package config

import (
	"testing"
)

func TestIsToolEnabled(t *testing.T) {
	tests := []struct {
		name     string
		tools    map[string]bool
		toolName string
		want     bool
	}{
		{
			name:     "nil tools map - enabled by default",
			tools:    nil,
			toolName: "GetSource",
			want:     true,
		},
		{
			name:     "empty tools map - enabled by default",
			tools:    map[string]bool{},
			toolName: "GetSource",
			want:     true,
		},
		{
			name:     "tool not in map - enabled by default",
			tools:    map[string]bool{"OtherTool": true},
			toolName: "GetSource",
			want:     true,
		},
		{
			name:     "tool explicitly enabled",
			tools:    map[string]bool{"GetSource": true},
			toolName: "GetSource",
			want:     true,
		},
		{
			name:     "tool explicitly disabled",
			tools:    map[string]bool{"GetSource": false},
			toolName: "GetSource",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SystemsConfig{Tools: tt.tools}
			got := cfg.IsToolEnabled(tt.toolName)
			if got != tt.want {
				t.Errorf("IsToolEnabled(%q) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}
}

func TestSetToolEnabled(t *testing.T) {
	tests := []struct {
		name      string
		initial   map[string]bool
		toolName  string
		enabled   bool
		wantValue bool
	}{
		{
			name:      "set tool enabled on nil map",
			initial:   nil,
			toolName:  "GetSource",
			enabled:   true,
			wantValue: true,
		},
		{
			name:      "set tool disabled on empty map",
			initial:   map[string]bool{},
			toolName:  "GetSource",
			enabled:   false,
			wantValue: false,
		},
		{
			name:      "update existing tool",
			initial:   map[string]bool{"GetSource": true},
			toolName:  "GetSource",
			enabled:   false,
			wantValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SystemsConfig{Tools: tt.initial}
			cfg.SetToolEnabled(tt.toolName, tt.enabled)

			if cfg.Tools == nil {
				t.Fatal("Tools map should not be nil after SetToolEnabled")
			}
			if got := cfg.Tools[tt.toolName]; got != tt.wantValue {
				t.Errorf("Tools[%q] = %v, want %v", tt.toolName, got, tt.wantValue)
			}
		})
	}
}

func TestGetDisabledTools(t *testing.T) {
	tests := []struct {
		name  string
		tools map[string]bool
		want  int // number of disabled tools
	}{
		{
			name:  "nil tools - no disabled",
			tools: nil,
			want:  0,
		},
		{
			name:  "empty tools - no disabled",
			tools: map[string]bool{},
			want:  0,
		},
		{
			name:  "all enabled - no disabled",
			tools: map[string]bool{"A": true, "B": true},
			want:  0,
		},
		{
			name:  "mixed - some disabled",
			tools: map[string]bool{"A": true, "B": false, "C": false},
			want:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SystemsConfig{Tools: tt.tools}
			got := cfg.GetDisabledTools()
			if len(got) != tt.want {
				t.Errorf("GetDisabledTools() returned %d tools, want %d", len(got), tt.want)
			}
		})
	}
}

func TestDefaultDisabledTools(t *testing.T) {
	defaults := DefaultDisabledTools()

	if len(defaults) == 0 {
		t.Error("DefaultDisabledTools() returned empty list")
	}

	// Check that AMDP debugger tools are in the list
	amdpTools := []string{
		"AMDPDebuggerStart", "AMDPDebuggerResume", "AMDPDebuggerStop",
		"AMDPDebuggerStep", "AMDPGetVariables", "AMDPSetBreakpoint", "AMDPGetBreakpoints",
	}

	for _, tool := range amdpTools {
		found := false
		for _, d := range defaults {
			if d == tool {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected %q in DefaultDisabledTools(), not found", tool)
		}
	}
}

func TestDisabledSystem(t *testing.T) {
	cfg := &SystemsConfig{
		Systems: map[string]SystemConfig{
			"dev":  {URL: "http://dev:50000", User: "DEV"},
			"prod": {URL: "http://prod:50000", User: "PROD", Disabled: true},
			"qa":   {URL: "http://qa:50000", User: "QA"},
		},
	}

	// GetSystem should reject disabled systems
	_, err := cfg.GetSystem("prod")
	if err == nil {
		t.Fatal("GetSystem should return error for disabled system")
	}
	if err.Error() != "system 'prod' is disabled" {
		t.Errorf("unexpected error: %v", err)
	}

	// GetSystem should still return enabled systems
	sys, err := cfg.GetSystem("dev")
	if err != nil {
		t.Fatalf("GetSystem('dev') unexpected error: %v", err)
	}
	if sys.User != "DEV" {
		t.Errorf("expected user DEV, got %s", sys.User)
	}

	// ListSystems should exclude disabled systems
	list := cfg.ListSystems()
	if len(list) != 2 {
		t.Errorf("ListSystems() returned %d systems, want 2", len(list))
	}
	for _, name := range list {
		if name == "prod" {
			t.Error("ListSystems() should not include disabled system 'prod'")
		}
	}
}
