// Package config provides system configuration management for vsp CLI.
package config

// ---------------------------------------------------------------------------
// Composable configuration sub-structs.
// Each groups related fields and carries JSON tags. When embedded, Go's
// encoding/json flattens them so the serialized JSON stays identical to the
// previous monolithic SystemConfig layout.
// ---------------------------------------------------------------------------

// ConnectionConfig holds core SAP connection settings.
type ConnectionConfig struct {
	URL      string `json:"url"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"` // Not recommended, use env var
	Client   string `json:"client,omitempty"`
	Language string `json:"language,omitempty"`
	Insecure bool   `json:"insecure,omitempty"`

	// Cookie authentication (alternative to user/password)
	CookieFile   string `json:"cookie_file,omitempty"`   // Path to Netscape-format cookie file
	CookieString string `json:"cookie_string,omitempty"` // Inline cookie string
}

// BrowserAuthConfig holds browser-based SSO authentication settings.
type BrowserAuthConfig struct {
	BrowserAuth        bool   `json:"browser_auth,omitempty"`         // Enable SSO login via browser
	BrowserAuthTimeout string `json:"browser_auth_timeout,omitempty"` // Timeout for browser login (e.g. "120s")
	BrowserExec        string `json:"browser_exec,omitempty"`         // Path to Chromium browser (auto-detect if empty)
	CookieSave         string `json:"cookie_save,omitempty"`          // Save browser cookies to file for reuse
}

// RfcConfig holds RFC connection settings (alternative to URL-based HTTP).
type RfcConfig struct {
	ConnectionMode string `json:"connection_mode,omitempty"` // "http" (default) or "rfc"
	AsHost         string `json:"ashost,omitempty"`
	SysNr          string `json:"sysnr,omitempty"`
	MsHost         string `json:"mshost,omitempty"`
	MsServ         string `json:"msserv,omitempty"`
	R3Name         string `json:"r3name,omitempty"`
	Group          string `json:"group,omitempty"`
}

// SncConfig holds SNC/SSO configuration (via SAP UI Landscape).
type SncConfig struct {
	SNC           bool   `json:"snc,omitempty"`            // Enable SNC single sign-on
	SysID         string `json:"sysid,omitempty"`          // SAP System ID from landscape (3 chars)
	LandscapeFile string `json:"landscape_file,omitempty"` // Explicit path to SAP UI Landscape XML
}

// SystemConfig represents a SAP system configuration.
// It composes the sub-structs above; Go's encoding/json flattens embedded
// structs so the serialized JSON stays identical to the previous layout.
type SystemConfig struct {
	Disabled bool `json:"disabled,omitempty"` // Skip this system when loading
	ConnectionConfig
	BrowserAuthConfig
	RfcConfig
	SncConfig

	SystemClass string           `json:"system_class,omitempty"`
	Permissions PermissionConfig `json:"permissions,omitempty"`

	// Per-system output settings
	Verbose bool `json:"verbose,omitempty"` // Enable verbose logging for this system

	// JCo connection properties resolved from SAP UI Landscape (SNC mode).
	// Stored here so they can be passed to the JCo sidecar and, in future,
	// also be specified directly in the config file.
	JcoProperties map[string]string `json:"jco_properties,omitempty"`
}

type SystemClassConfig struct {
	Permissions PermissionConfig `json:"permissions,omitempty"`
}

// SetToolEnabled sets the enabled state for a tool.
func (c *GlobalConfigJSON) SetToolEnabled(toolName string, enabled bool) {
	if c.Tools == nil {
		c.Tools = make(map[string]bool)
	}
	c.Tools[toolName] = enabled
}

// DefaultDisabledTools returns the list of tools that should be disabled by default.
// These are experimental or non-working tools.
func DefaultDisabledTools() []string {
	return []string{
		// AMDP/HANA Debugger - session management issues
		"AMDPDebuggerStart", "AMDPDebuggerResume", "AMDPDebuggerStop",
		"AMDPDebuggerStep", "AMDPGetVariables", "AMDPSetBreakpoint", "AMDPGetBreakpoints",
		// ABAP Debugger - requires ZADT_VSP WebSocket, HTTP unreliable
		"DebuggerListen", "DebuggerAttach", "DebuggerDetach",
		"DebuggerStep", "DebuggerGetStack", "DebuggerGetVariables",
		// Breakpoints - requires ZADT_VSP WebSocket
		"SetBreakpoint", "GetBreakpoints", "DeleteBreakpoint",
		// UI5 write operations - need alternate API
		"UI5CreateApp", "UI5DeleteApp", "UI5DeleteFile", "UI5UploadFile",
	}
}
