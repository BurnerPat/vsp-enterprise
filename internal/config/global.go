package config

import (
	"strings"
	"time"

	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// ---------------------------------------------------------------------------
// Singleton access to the global configuration.
// Set once at startup before any system is created; read-only afterwards.
// ---------------------------------------------------------------------------

var instance *GlobalConfig

// SetInstance stores the global configuration singleton.
// Must be called once at startup before creating systems.
func SetInstance(cfg *GlobalConfig) { instance = cfg }

// GetInstance returns the global configuration singleton.
// It panics if called before SetInstance — callers may assume the result is never nil.
func GetInstance() *GlobalConfig {
	if instance == nil {
		panic("config.GetInstance() called before SetInstance — global configuration not initialized")
	}

	return instance
}

// ---------------------------------------------------------------------------
// Top-level (global) configuration — the singleton.
// ---------------------------------------------------------------------------

// PermissionConfig Configuration for tool permissions
type PermissionConfig struct {
	DenyToolsByDefault bool            `json:"deny_tools_by_default"`
	Tools              map[string]bool `json:"tools"`
	ReadOnly           bool            `json:"read_only,omitempty"`
	AllowedPackages    []string        `json:"allowed_packages,omitempty"`
}

// GlobalConfigJSON is the part of the global configuration that can be (un)marshalled
type GlobalConfigJSON struct {
	Systems       map[string]SystemConfig `json:"systems"`
	DefaultSystem string                  `json:"default,omitempty"`

	// Tools configuration - granular tool visibility control
	// Key: tool name, Value: true=enabled, false=disabled
	// Tools not listed are enabled by default
	Tools map[string]bool `json:"tools,omitempty"`

	Permissions   PermissionConfig             `json:"permissions,omitempty"`
	SystemClasses map[string]SystemClassConfig `json:"system_classes,omitempty"`
}

// GlobalConfig is the top-level runtime configuration.
// It holds global settings and all resolved system configurations.
// In single-system mode the CLI stores the system under key "default".
// Access it from anywhere via config.GetInstance().
type GlobalConfig struct {
	GlobalConfigJSON

	// Global server settings
	Verbose        bool            // Global verbose flag
	Mode           string          // Tool mode: focused, expert, hyperfocused
	DisabledGroups string          // Disabled tool groups (e.g. "TH")
	ToolsConfig    map[string]bool // Per-tool visibility from .vsp.json

	// Global JCo/sidecar settings
	JcoLibsDir       string
	RfcProxyPort     int
	RfcMaxConcurrent int
	SidecarTransport string
	JcoProxyJar      string
	JavaPath         string

	// Global safety settings
	ReadOnly                bool
	AllowedPackages         []string
	BlockFreeSQL            bool
	AllowedOps              string
	DisallowedOps           string
	EnableTransports        bool
	TransportReadOnly       bool
	AllowedTransports       []string
	AllowTransportableEdits bool

	// Feature configuration (safety network)
	FeatureHANA      string
	FeatureAbapGit   string
	FeatureRAP       string
	FeatureAMDP      string
	FeatureUI5       string
	FeatureTransport string

	// Debugger configuration
	TerminalID string

	// Session keep-alive interval (0 = disabled)
	KeepAliveInterval time.Duration
}

// DefaultSystemID is the key used for the single-system entry in the Systems map.
const DefaultSystemID = "default"

// ---------------------------------------------------------------------------
// Methods on SystemConfig
// These require access to global settings and therefore live here, next to
// the GlobalConfig definition.
// ---------------------------------------------------------------------------

// IsVerbose returns true if either the per-system or global verbose flag is set.
func (c *SystemConfig) IsVerbose() bool {
	return c.Verbose || GetInstance().Verbose
}

// BuildADTOptions constructs the common adt.Option slice.
// Note: cookies are a runtime concern owned by the System instance, not this
// config struct. The System adds them separately when creating the ADT client.
func (c *SystemConfig) BuildADTOptions() []adt.Option {
	opts := []adt.Option{
		adt.WithClient(c.Client),
		adt.WithLanguage(c.Language),
	}
	if c.Insecure {
		opts = append(opts, adt.WithInsecureSkipVerify())
	}
	if c.IsVerbose() {
		opts = append(opts, adt.WithVerbose())
	}
	opts = append(opts, adt.WithSafety(c.BuildSafetyConfig()))
	return opts
}

// BuildSafetyConfig constructs an adt.SafetyConfig by merging per-system
// SafetySettings with global safety flags from the singleton.
func (c *SystemConfig) BuildSafetyConfig() adt.SafetyConfig {
	g := GetInstance()
	safety := adt.UnrestrictedSafetyConfig()

	// ReadOnly / AllowedPackages: per-system OR global
	if c.Permissions.ReadOnly || g.ReadOnly {
		safety.ReadOnly = true
	}

	if len(c.Permissions.AllowedPackages) > 0 {
		safety.AllowedPackages = c.Permissions.AllowedPackages
	} else if len(g.AllowedPackages) > 0 {
		safety.AllowedPackages = g.AllowedPackages
	}

	// Remaining safety flags are global-only
	if g.BlockFreeSQL {
		safety.BlockFreeSQL = true
	}
	if g.AllowedOps != "" {
		safety.AllowedOps = g.AllowedOps
	}
	if g.DisallowedOps != "" {
		safety.DisallowedOps = g.DisallowedOps
	}
	if g.EnableTransports {
		safety.EnableTransports = true
	}
	if g.TransportReadOnly {
		safety.TransportReadOnly = true
	}
	if len(g.AllowedTransports) > 0 {
		safety.AllowedTransports = g.AllowedTransports
	}
	if g.AllowTransportableEdits {
		safety.AllowTransportableEdits = true
	}
	return safety
}

// BuildFeatureConfig constructs an adt.FeatureConfig from global settings.
func (c *SystemConfig) BuildFeatureConfig() adt.FeatureConfig {
	g := GetInstance()
	return adt.FeatureConfig{
		HANA:      parseFeatureMode(g.FeatureHANA),
		AbapGit:   parseFeatureMode(g.FeatureAbapGit),
		RAP:       parseFeatureMode(g.FeatureRAP),
		AMDP:      parseFeatureMode(g.FeatureAMDP),
		UI5:       parseFeatureMode(g.FeatureUI5),
		Transport: parseFeatureMode(g.FeatureTransport),
	}
}

// BuildSidecarConfig constructs an adt.SidecarConfig, combining per-system
// connection fields with global JCo settings from the singleton.
func (c *SystemConfig) BuildSidecarConfig() *adt.SidecarConfig {
	g := GetInstance()

	return &adt.SidecarConfig{
		// Global JCo settings
		JcoProxyJar:   g.JcoProxyJar,
		JavaPath:      g.JavaPath,
		JcoLibsDir:    g.JcoLibsDir,
		Port:          g.RfcProxyPort,
		MaxConcurrent: g.RfcMaxConcurrent,
		Transport:     g.SidecarTransport,

		// Per-system connection fields
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
