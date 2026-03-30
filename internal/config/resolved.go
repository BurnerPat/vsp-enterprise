package config

import (
	"strings"
	"time"

	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// ResolvedConfig is the single central configuration structure used at runtime.
// It replaces the former mcp.Config, mcp.SystemConfigResolved, and systemParams
// types. For per-system JSON config it embeds the same sub-structs as SystemConfig;
// additional runtime-only and global-only fields are declared directly.
type ResolvedConfig struct {
	// Per-system settings (from JSON config or CLI flags).
	// Promoted fields: URL, User, Password, Client, Language, Insecure,
	// CookieFile, CookieString, BrowserAuth, BrowserAuthTimeout, BrowserExec,
	// CookieSave, ConnectionMode, AsHost, SysNr, MsHost, MsServ, R3Name,
	// Group, JcoProxyJar, JavaPath, SNC, SysID, LandscapeFile,
	// ReadOnly, AllowedPackages.
	ConnectionConfig
	BrowserAuthConfig
	RfcConfig
	SncConfig
	SafetySettings

	// Runtime-resolved fields (not from JSON)
	Cookies       map[string]string // Parsed cookies (from cookie file/string/browser)
	JcoProperties map[string]string // Resolved JCo properties (from SNC landscape)
	Verbose       bool              // Enable verbose logging

	// Global JCo/sidecar settings (CLI-only, not per-system JSON)
	JcoLibsDir       string // Path to JCo libraries directory
	RfcProxyPort     int    // Fixed sidecar port (0 = auto-assign)
	RfcMaxConcurrent int    // Max concurrent RFC calls
	SidecarTransport string // Sidecar transport: "http" or "stdio"

	// Global server settings
	Mode           string          // Tool mode: focused, expert, hyperfocused
	DisabledGroups string          // Disabled tool groups (e.g. "TH")
	ToolsConfig    map[string]bool // Per-tool visibility from .vsp.json

	// Global safety settings (applied on top of per-system SafetySettings)
	BlockFreeSQL            bool
	AllowedOps              string
	DisallowedOps           string
	EnableTransports        bool
	TransportReadOnly       bool
	AllowedTransports       []string
	AllowTransportableEdits bool

	// Feature configuration (safety network)
	// Values: "auto" (default, probe system), "on" (force enabled), "off" (force disabled)
	FeatureHANA      string
	FeatureAbapGit   string
	FeatureRAP       string
	FeatureAMDP      string
	FeatureUI5       string
	FeatureTransport string

	// Debugger configuration
	TerminalID string // SAP GUI terminal ID for cross-tool breakpoint sharing

	// Session keep-alive interval (0 = disabled)
	KeepAliveInterval time.Duration

	// Multi-system mode
	MultiSystem  bool                       // Enable multi-system routing
	MultiSystems map[string]*ResolvedConfig // system_id → resolved config
}

// ---------------------------------------------------------------------------
// Converters
// ---------------------------------------------------------------------------

// ToResolved creates a ResolvedConfig from a SystemConfig, copying all
// per-system fields. Global-only fields are left at their zero values and
// should be populated via MergeGlobal or direct assignment.
func (sc *SystemConfig) ToResolved() *ResolvedConfig {
	return &ResolvedConfig{
		ConnectionConfig:  sc.ConnectionConfig,
		BrowserAuthConfig: sc.BrowserAuthConfig,
		RfcConfig:         sc.RfcConfig,
		SncConfig:         sc.SncConfig,
		SafetySettings:    sc.SafetySettings,
		Verbose:           sc.Verbose,
	}
}

// MergeGlobal overlays global settings from the CLI-level config onto this
// per-system ResolvedConfig. It is called once per system when building a
// multi-system server, replacing the former systemConfigForMCP function.
func (c *ResolvedConfig) MergeGlobal(global *ResolvedConfig) {
	// Verbose: OR of per-system and global
	c.Verbose = c.Verbose || global.Verbose

	// Global JCo libs shared across all systems
	c.JcoLibsDir = global.JcoLibsDir

	// Global server settings
	c.Mode = global.Mode
	c.DisabledGroups = global.DisabledGroups
	c.ToolsConfig = global.ToolsConfig

	// Global safety settings
	c.BlockFreeSQL = global.BlockFreeSQL
	c.AllowedOps = global.AllowedOps
	c.DisallowedOps = global.DisallowedOps
	c.EnableTransports = global.EnableTransports
	c.TransportReadOnly = global.TransportReadOnly
	c.AllowedTransports = global.AllowedTransports
	c.AllowTransportableEdits = global.AllowTransportableEdits

	// Global feature settings
	c.FeatureHANA = global.FeatureHANA
	c.FeatureAbapGit = global.FeatureAbapGit
	c.FeatureRAP = global.FeatureRAP
	c.FeatureAMDP = global.FeatureAMDP
	c.FeatureUI5 = global.FeatureUI5
	c.FeatureTransport = global.FeatureTransport

	// Global debugger settings
	c.TerminalID = global.TerminalID

	// Global keep-alive
	c.KeepAliveInterval = global.KeepAliveInterval
}

// ---------------------------------------------------------------------------
// ADT builder methods — replace the former buildADTOptions / buildSafetyConfig /
// buildFeatureConfig / sidecar construction scattered across system.go.
// ---------------------------------------------------------------------------

// BuildADTOptions constructs the common adt.Option slice from this config.
func (c *ResolvedConfig) BuildADTOptions() []adt.Option {
	opts := []adt.Option{
		adt.WithClient(c.Client),
		adt.WithLanguage(c.Language),
	}
	if c.Insecure {
		opts = append(opts, adt.WithInsecureSkipVerify())
	}
	if len(c.Cookies) > 0 {
		opts = append(opts, adt.WithCookies(c.Cookies))
	}
	if c.Verbose {
		opts = append(opts, adt.WithVerbose())
	}
	opts = append(opts, adt.WithSafety(c.BuildSafetyConfig()))
	return opts
}

// BuildSafetyConfig constructs an adt.SafetyConfig from this config.
func (c *ResolvedConfig) BuildSafetyConfig() adt.SafetyConfig {
	safety := adt.UnrestrictedSafetyConfig()
	if c.ReadOnly {
		safety.ReadOnly = true
	}
	if c.BlockFreeSQL {
		safety.BlockFreeSQL = true
	}
	if c.AllowedOps != "" {
		safety.AllowedOps = c.AllowedOps
	}
	if c.DisallowedOps != "" {
		safety.DisallowedOps = c.DisallowedOps
	}
	if len(c.AllowedPackages) > 0 {
		safety.AllowedPackages = c.AllowedPackages
	}
	if c.EnableTransports {
		safety.EnableTransports = true
	}
	if c.TransportReadOnly {
		safety.TransportReadOnly = true
	}
	if len(c.AllowedTransports) > 0 {
		safety.AllowedTransports = c.AllowedTransports
	}
	if c.AllowTransportableEdits {
		safety.AllowTransportableEdits = true
	}
	return safety
}

// BuildFeatureConfig constructs an adt.FeatureConfig from this config.
func (c *ResolvedConfig) BuildFeatureConfig() adt.FeatureConfig {
	return adt.FeatureConfig{
		HANA:      parseFeatureMode(c.FeatureHANA),
		AbapGit:   parseFeatureMode(c.FeatureAbapGit),
		RAP:       parseFeatureMode(c.FeatureRAP),
		AMDP:      parseFeatureMode(c.FeatureAMDP),
		UI5:       parseFeatureMode(c.FeatureUI5),
		Transport: parseFeatureMode(c.FeatureTransport),
	}
}

// BuildSidecarConfig constructs an adt.SidecarConfig for the JCo sidecar.
func (c *ResolvedConfig) BuildSidecarConfig() *adt.SidecarConfig {
	return &adt.SidecarConfig{
		JcoProxyJar:   c.JcoProxyJar,
		JcoLibsDir:    c.JcoLibsDir,
		JavaPath:      c.JavaPath,
		Port:          c.RfcProxyPort,
		MaxConcurrent: c.RfcMaxConcurrent,
		Transport:     c.SidecarTransport,
		AsHost:        c.AsHost,
		SysNr:         c.SysNr,
		MsHost:        c.MsHost,
		MsServ:        c.MsServ,
		R3Name:        c.R3Name,
		Group:         c.Group,
		Client:        c.Client,
		Username:      c.User,
		Password:      c.Password,
		Language:      c.Language,
		JcoProperties: c.JcoProperties,
	}
}

// parseFeatureMode converts a string feature flag to an adt.FeatureMode.
func parseFeatureMode(s string) adt.FeatureMode {
	switch strings.ToLower(s) {
	case "on", "true", "1", "yes", "enabled":
		return adt.FeatureModeOn
	case "off", "false", "0", "no", "disabled":
		return adt.FeatureModeOff
	default:
		return adt.FeatureModeAuto
	}
}
