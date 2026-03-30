// Package mcp provides the MCP server implementation for ABAP ADT tools.
// system.go contains the System struct representing a configured SAP system destination.
// It holds the ADT client, WebSocket clients, sidecar, feature prober, and per-system state.
// System implements the types.System interface consumed by all tool handlers.
package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	deps "github.com/oisee/vibing-steampunk/embedded/deps"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// AsyncTask represents a background task status.
type AsyncTask struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`   // "report", "export", etc.
	Status    string      `json:"status"` // "running", "completed", "error"
	StartedAt time.Time   `json:"started_at"`
	EndedAt   *time.Time  `json:"ended_at,omitempty"`
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// System represents a configured destination to an SAP system.
// It holds the connection (ADT client), WebSocket clients, sidecar manager,
// feature prober, and async task state. System implements types.System.
type System struct {
	adtClient     *adt.Client
	amdpWSClient  *adt.AMDPWebSocketClient  // WebSocket-based AMDP client (ZADT_VSP)
	debugWSClient *adt.DebugWebSocketClient // WebSocket-based debug client (ZADT_VSP)
	config        *Config                   // Per-system configuration
	featureProber *adt.FeatureProber        // Feature detection system (safety network)
	featureConfig adt.FeatureConfig         // Feature configuration
	sidecar       *adt.SidecarManager       // JCo sidecar (RFC mode only)

	// Async task management
	asyncTasks   map[string]*AsyncTask
	asyncTasksMu sync.RWMutex
	asyncTaskID  int64
}

// Ensure System implements types.System at compile time.
var _ types.System = (*System)(nil)

// ADT implements types.System.
func (s *System) ADT() *adt.Client {
	return s.adtClient
}

// Config implements types.System.
func (s *System) Config() any {
	return s.config
}

// IsRfcMode implements types.System.
func (s *System) IsRfcMode() bool {
	return s.sidecar != nil
}

// Sidecar implements types.System.
func (s *System) Sidecar() *adt.SidecarManager {
	return s.sidecar
}

// RequireActiveAMDPSession implements types.System.
func (s *System) RequireActiveAMDPSession() *mcp.CallToolResult {
	if s.amdpWSClient == nil || !s.amdpWSClient.IsActive() {
		return types.ErrorResult("No active AMDP debug session. Start one using StartAMDPDebugSession tool.")
	}
	return nil
}

// EnsureWSConnected implements types.System.
func (s *System) EnsureWSConnected(ctx context.Context, toolName string) *mcp.CallToolResult {
	return s.ensureWSConnected(ctx, toolName)
}

// Shutdown gracefully stops system-level resources (sidecar, keep-alive, etc.).
func (s *System) Shutdown() {
	if s.sidecar != nil {
		s.sidecar.Stop()
	}
}

// ensureWSConnected ensures the WebSocket client is connected, creating it if needed.
// Returns error result if connection fails, nil on success.
func (s *System) ensureWSConnected(ctx context.Context, toolName string) *mcp.CallToolResult {
	if s.amdpWSClient == nil || !s.amdpWSClient.IsConnected() {
		s.amdpWSClient = adt.NewAMDPWebSocketClient(
			s.config.BaseURL, s.config.Client, s.config.Username, s.config.Password, s.config.InsecureSkipVerify,
		)
		if err := s.amdpWSClient.Connect(ctx); err != nil {
			s.amdpWSClient = nil
			return newToolResultError(fmt.Sprintf("%s: WebSocket connect failed: %v", toolName, err))
		}
	}
	return nil
}

// requireActiveAMDPSession checks if there's an active AMDP debug session.
// Returns error result if no session, nil if session is active.
func (s *System) requireActiveAMDPSession() *mcp.CallToolResult {
	if s.amdpWSClient == nil || !s.amdpWSClient.IsActive() {
		return newToolResultError("No active AMDP session. Use AMDPDebuggerStart first.")
	}
	return nil
}

// ---------------------------------------------------------------------------
// System construction helpers
// ---------------------------------------------------------------------------

// newSystemInstance creates a System with an ADT client, feature prober, and optional sidecar.
// It does NOT create an MCP server or register tools — that is the Server's responsibility.
func newSystemInstance(cfg *Config) (*System, error) {
	ensureProxyJAR(cfg)

	opts := buildADTOptions(cfg)
	adtClient, sidecar, err := createADTClient(cfg, opts)
	if err != nil {
		return nil, err
	}

	// Set terminal ID for debugger operations
	if cfg.TerminalID != "" {
		adt.SetTerminalID(cfg.TerminalID)
	}
	adt.SetTerminalIDUser(cfg.Username)

	featureConfig := buildFeatureConfig(cfg)

	sys := &System{
		adtClient:     adtClient,
		config:        cfg,
		featureProber: adt.NewFeatureProber(adtClient, featureConfig, cfg.Verbose),
		featureConfig: featureConfig,
		sidecar:       sidecar,
		asyncTasks:    make(map[string]*AsyncTask),
	}

	// Start session keep-alive if configured
	if cfg.KeepAliveInterval > 0 {
		adtClient.StartKeepAlive(cfg.KeepAliveInterval, cfg.Verbose)
	}

	return sys, nil
}

// ---------------------------------------------------------------------------
// ADT client & config builder helpers (package-level, used by newSystemInstance)
// ---------------------------------------------------------------------------

// buildADTOptions constructs the common ADT client options from config.
func buildADTOptions(cfg *Config) []adt.Option {
	opts := []adt.Option{
		adt.WithClient(cfg.Client),
		adt.WithLanguage(cfg.Language),
	}
	if cfg.InsecureSkipVerify {
		opts = append(opts, adt.WithInsecureSkipVerify())
	}
	if len(cfg.Cookies) > 0 {
		opts = append(opts, adt.WithCookies(cfg.Cookies))
	}
	if cfg.Verbose {
		opts = append(opts, adt.WithVerbose())
	}
	opts = append(opts, adt.WithSafety(buildSafetyConfig(cfg)))
	return opts
}

// buildSafetyConfig constructs the safety configuration from config.
func buildSafetyConfig(cfg *Config) adt.SafetyConfig {
	safety := adt.UnrestrictedSafetyConfig()
	if cfg.ReadOnly {
		safety.ReadOnly = true
	}
	if cfg.BlockFreeSQL {
		safety.BlockFreeSQL = true
	}
	if cfg.AllowedOps != "" {
		safety.AllowedOps = cfg.AllowedOps
	}
	if cfg.DisallowedOps != "" {
		safety.DisallowedOps = cfg.DisallowedOps
	}
	if len(cfg.AllowedPackages) > 0 {
		safety.AllowedPackages = cfg.AllowedPackages
	}
	if cfg.EnableTransports {
		safety.EnableTransports = true
	}
	if cfg.TransportReadOnly {
		safety.TransportReadOnly = true
	}
	if len(cfg.AllowedTransports) > 0 {
		safety.AllowedTransports = cfg.AllowedTransports
	}
	if cfg.AllowTransportableEdits {
		safety.AllowTransportableEdits = true
	}
	return safety
}

// buildFeatureConfig constructs the feature detection configuration from config.
func buildFeatureConfig(cfg *Config) adt.FeatureConfig {
	return adt.FeatureConfig{
		HANA:      parseFeatureMode(cfg.FeatureHANA),
		AbapGit:   parseFeatureMode(cfg.FeatureAbapGit),
		RAP:       parseFeatureMode(cfg.FeatureRAP),
		AMDP:      parseFeatureMode(cfg.FeatureAMDP),
		UI5:       parseFeatureMode(cfg.FeatureUI5),
		Transport: parseFeatureMode(cfg.FeatureTransport),
	}
}

// createADTClient creates an ADT client and optional sidecar based on connection mode.
func createADTClient(cfg *Config, opts []adt.Option) (*adt.Client, *adt.SidecarManager, error) {
	if strings.EqualFold(cfg.ConnectionMode, "rfc") {
		return createRFCADTClient(cfg, opts)
	}
	return adt.NewClient(cfg.BaseURL, cfg.Username, cfg.Password, opts...), nil, nil
}

// createRFCADTClient creates an ADT client using RFC mode with a JCo sidecar.
func createRFCADTClient(cfg *Config, opts []adt.Option) (*adt.Client, *adt.SidecarManager, error) {
	adtCfg := adt.NewConfig("", cfg.Username, cfg.Password, opts...)

	sidecarCfg := &adt.SidecarConfig{
		JcoProxyJar:   cfg.JcoProxyJar,
		JcoLibsDir:    cfg.JcoLibsDir,
		JavaPath:      cfg.JavaPath,
		Port:          cfg.RfcProxyPort,
		MaxConcurrent: cfg.RfcMaxConcurrent,
		Transport:     cfg.SidecarTransport,
		AsHost:        cfg.AsHost,
		SysNr:         cfg.SysNr,
		MsHost:        cfg.MsHost,
		MsServ:        cfg.MsServ,
		R3Name:        cfg.R3Name,
		Group:         cfg.Group,
		Client:        cfg.Client,
		Username:      cfg.Username,
		Password:      cfg.Password,
		Language:      cfg.Language,
		JcoProperties: cfg.JcoProperties,
	}
	sidecar := adt.NewSidecarManager(sidecarCfg)

	if err := sidecar.Start(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("failed to start JCo sidecar: %w", err)
	}

	maxConcurrent := cfg.RfcMaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}

	var adtClient *adt.Client
	if sidecar.IsSTDIO() {
		stdioTransport := adt.NewStdioRfcTransport(sidecar, adtCfg, maxConcurrent)
		adtClient = adt.NewClientWithTransport(adtCfg, stdioTransport)
	} else {
		rfcTransport := adt.NewRfcTransport(sidecar.URL(), adtCfg, maxConcurrent)
		adtClient = adt.NewClientWithTransport(adtCfg, rfcTransport)
	}
	return adtClient, sidecar, nil
}

// ensureProxyJAR auto-extracts the embedded proxy JAR if no valid path is configured.
func ensureProxyJAR(cfg *Config) {
	if cfg.JcoProxyJar != "" && fileExists(cfg.JcoProxyJar) {
		return
	}
	data := deps.GetEmbeddedProxyJar()
	if data == nil {
		return
	}
	extractDir := cfg.JcoLibsDir
	if extractDir == "" {
		extractDir = "./jco-libs"
	}
	proxyPath := filepath.Join(extractDir, "jco-proxy.jar")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return
	}
	if err := os.WriteFile(proxyPath, data, 0644); err != nil {
		return
	}
	cfg.JcoProxyJar = proxyPath
	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Auto-extracted embedded proxy JAR to %s\n", proxyPath)
	}
}

// fileExists returns true if the path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// parseFeatureMode converts string to FeatureMode.
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

// newToolResultError creates an error result for tool execution failures.
func newToolResultError(message string) *mcp.CallToolResult {
	result := mcp.NewToolResultText(message)
	result.IsError = true
	return result
}
