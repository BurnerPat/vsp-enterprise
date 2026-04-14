package config

import (
	"encoding/json"
	"testing"
)

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
			cfg := &GlobalConfigJSON{Tools: tt.initial}
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

// TestSystemConfigJSONRoundTrip verifies that the embedded-struct refactoring
// produces the exact same JSON keys as the original flat struct.
func TestSystemConfigJSONRoundTrip(t *testing.T) {
	original := GlobalConfigJSON{
		DefaultSystem: "dev",
		Systems: map[string]SystemConfig{
			"dev": {
				ConnectionConfig: ConnectionConfig{
					URL:      "http://dev:50000",
					Username: "DEV",
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
					Username: "RFC_USER",
					Client:   "100",
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
				Roles: []string{"reader"},
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
	for _, key := range []string{"url", "username", "password", "client", "language", "insecure", "browser_auth", "browser_auth_timeout", "verbose"} {
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
	var roundTripped GlobalConfigJSON
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("Round-trip unmarshal failed: %v", err)
	}
	if roundTripped.DefaultSystem != original.DefaultSystem {
		t.Errorf("Default mismatch: got %q, want %q", roundTripped.DefaultSystem, original.DefaultSystem)
	}
	dev := roundTripped.Systems["dev"]
	if dev.URL != "http://dev:50000" || dev.Username != "DEV" || dev.Client != "001" || !dev.Insecure || !dev.BrowserAuth {
		t.Errorf("dev system round-trip mismatch: %+v", dev)
	}
	rfc := roundTripped.Systems["rfc"]
	if rfc.ConnectionMode != "rfc" || rfc.AsHost != "sap.example.com" || !rfc.SNC || rfc.SysID != "A4H" {
		t.Errorf("rfc system round-trip mismatch: %+v", rfc)
	}
	if len(rfc.Roles) != 1 || rfc.Roles[0] != "reader" {
		t.Errorf("rfc roles round-trip mismatch: got %v, want [reader]", rfc.Roles)
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
