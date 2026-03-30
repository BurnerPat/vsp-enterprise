package config

import (
	"encoding/json"
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
			"dev":  {ConnectionConfig: ConnectionConfig{URL: "http://dev:50000", User: "DEV"}},
			"prod": {ConnectionConfig: ConnectionConfig{URL: "http://prod:50000", User: "PROD"}, Disabled: true},
			"qa":   {ConnectionConfig: ConnectionConfig{URL: "http://qa:50000", User: "QA"}},
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

// TestSystemConfigJSONRoundTrip verifies that the embedded-struct refactoring
// produces the exact same JSON keys as the original flat struct.
func TestSystemConfigJSONRoundTrip(t *testing.T) {
	original := SystemsConfig{
		Default: "dev",
		Systems: map[string]SystemConfig{
			"dev": {
				ConnectionConfig: ConnectionConfig{
					URL:      "http://dev:50000",
					User:     "DEV",
					Password: "secret",
					Client:   "001",
					Language: "EN",
					Insecure: true,
				},
				BrowserAuthConfig: BrowserAuthConfig{
					BrowserAuth:        true,
					BrowserAuthTimeout: "60s",
				},
				Verbose: true,
			},
			"rfc": {
				ConnectionConfig: ConnectionConfig{
					User:   "RFC_USER",
					Client: "100",
				},
				RfcConfig: RfcConfig{
					ConnectionMode: "rfc",
					AsHost:         "sap.example.com",
					SysNr:          "00",
				},
				SncConfig: SncConfig{
					SNC:   true,
					SysID: "A4H",
				},
				SafetySettings: SafetySettings{
					ReadOnly:        true,
					AllowedPackages: []string{"Z*"},
				},
			},
		},
		Tools: map[string]bool{"GetSource": true, "DeleteObject": false},
	}

	// Marshal
	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify key JSON keys exist at top level (flat, not nested)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to raw failed: %v", err)
	}
	for _, key := range []string{"systems", "default", "tools"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected top-level key %q in JSON", key)
		}
	}

	// Verify system-level keys are flat (not nested under sub-struct names)
	var rawSystems struct {
		Systems map[string]json.RawMessage `json:"systems"`
	}
	if err := json.Unmarshal(data, &rawSystems); err != nil {
		t.Fatalf("Unmarshal systems failed: %v", err)
	}
	devRaw := rawSystems.Systems["dev"]
	var devMap map[string]json.RawMessage
	if err := json.Unmarshal(devRaw, &devMap); err != nil {
		t.Fatalf("Unmarshal dev system failed: %v", err)
	}
	// These must be flat JSON keys, not nested under ConnectionConfig etc.
	for _, key := range []string{"url", "user", "password", "client", "language", "insecure", "browser_auth", "browser_auth_timeout", "verbose"} {
		if _, ok := devMap[key]; !ok {
			t.Errorf("expected flat key %q in dev system JSON, got keys: %v", key, keysOf(devMap))
		}
	}
	// Must NOT have embedded struct names as keys
	for _, bad := range []string{"ConnectionConfig", "BrowserAuthConfig", "RfcConfig", "SncConfig", "SafetySettings"} {
		if _, ok := devMap[bad]; ok {
			t.Errorf("embedded struct name %q must not appear as JSON key", bad)
		}
	}

	// Round-trip: unmarshal back and compare
	var roundTripped SystemsConfig
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("Round-trip unmarshal failed: %v", err)
	}
	if roundTripped.Default != original.Default {
		t.Errorf("Default mismatch: got %q, want %q", roundTripped.Default, original.Default)
	}
	dev := roundTripped.Systems["dev"]
	if dev.URL != "http://dev:50000" || dev.User != "DEV" || dev.Client != "001" || !dev.Insecure || !dev.BrowserAuth {
		t.Errorf("dev system round-trip mismatch: %+v", dev)
	}
	rfc := roundTripped.Systems["rfc"]
	if rfc.ConnectionMode != "rfc" || rfc.AsHost != "sap.example.com" || !rfc.SNC || rfc.SysID != "A4H" || !rfc.ReadOnly {
		t.Errorf("rfc system round-trip mismatch: %+v", rfc)
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
