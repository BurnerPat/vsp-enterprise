// Package mcp provides the MCP server implementation for ABAP ADT tools.
package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	deps "github.com/oisee/vibing-steampunk/embedded/deps"
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

// Server wraps the MCP server with ADT client.
type Server struct {
	mcpServer     *server.MCPServer
	adtClient     *adt.Client
	amdpWSClient  *adt.AMDPWebSocketClient  // WebSocket-based AMDP client (ZADT_VSP)
	debugWSClient *adt.DebugWebSocketClient // WebSocket-based debug client (ZADT_VSP)
	config        *Config                   // Server configuration for session manager creation
	featureProber *adt.FeatureProber        // Feature detection system (safety network)
	featureConfig adt.FeatureConfig         // Feature configuration
	sidecar       *adt.SidecarManager       // JCo sidecar (RFC mode only)

	// Multi-system support
	multiSystem bool                              // true when --multi-system is active
	router      *multiSystemRouter                // routes requests to per-system servers (multi-system only)
	handlerMap  map[string]server.ToolHandlerFunc // local handler registry (per-system builder mode)
	toolMap     map[string]mcp.Tool               // tool definitions (per-system builder mode)

	// Async task management
	asyncTasks   map[string]*AsyncTask
	asyncTasksMu sync.RWMutex
	asyncTaskID  int64
}

// Config holds MCP server configuration.
type Config struct {
	// SAP connection settings
	BaseURL            string
	Username           string
	Password           string
	Client             string
	Language           string
	InsecureSkipVerify bool

	// Cookie authentication (alternative to basic auth)
	Cookies map[string]string

	// Verbose output
	Verbose bool

	// Mode: focused or expert (default: focused)
	Mode string

	// DisabledGroups disables groups of tools using short codes:
	// 5/U = UI5/BSP tools, T = Test tools, H = HANA/AMDP debugger, D = ABAP Debugger
	// Example: "TH" disables Tests and HANA debugger tools
	DisabledGroups string

	// Safety configuration
	ReadOnly                bool
	BlockFreeSQL            bool
	AllowedOps              string
	DisallowedOps           string
	AllowedPackages         []string
	EnableTransports        bool     // Explicitly enable transport management (default: disabled)
	TransportReadOnly       bool     // Only allow read operations on transports (list, get)
	AllowedTransports       []string // Whitelist specific transports (supports wildcards like "A4HK*")
	AllowTransportableEdits bool     // Allow editing objects that require transport requests

	// Feature configuration (safety network)
	// Values: "auto" (default, probe system), "on" (force enabled), "off" (force disabled)
	FeatureHANA      string // HANA database detection (required for some AMDP features)
	FeatureAbapGit   string // abapGit integration
	FeatureRAP       string // RAP/OData development (DDLS, BDEF, SRVD, SRVB)
	FeatureAMDP      string // AMDP/HANA debugger
	FeatureUI5       string // UI5/Fiori BSP management
	FeatureTransport string // CTS transport management (distinct from EnableTransports safety)

	// Debugger configuration
	TerminalID string // SAP GUI terminal ID for cross-tool breakpoint sharing

	// Session keep-alive interval (0 = disabled)
	// Sends periodic pings to prevent session timeout during idle periods.
	// Useful for cookie/browser-auth where sessions expire server-side.
	KeepAliveInterval time.Duration

	// RFC connection settings (alternative to HTTP)
	ConnectionMode   string
	AsHost           string
	SysNr            string
	MsHost           string
	MsServ           string
	R3Name           string
	Group            string
	JcoProxyJar      string
	JcoLibsDir       string
	JavaPath         string
	RfcProxyPort     int
	RfcMaxConcurrent int

	// SNC/SSO configuration (via SAP UI Landscape)
	SNC           bool              // Enable SNC single sign-on
	SysID         string            // SAP System ID from landscape (3 chars)
	LandscapeFile string            // Explicit path to SAP UI Landscape XML
	JcoProperties map[string]string // Resolved JCo properties (populated during config resolution)

	// Sidecar transport mode: "http" (default) or "stdio"
	SidecarTransport string

	// Multi-system mode
	MultiSystem  bool                             // Enable multi-system routing
	MultiSystems map[string]*SystemConfigResolved // system_id -> resolved config (populated by cmd)

	// Granular tool visibility (from .vsp.json)
	// Key: tool name, Value: true=enabled, false=disabled
	// Takes highest priority over mode and disabled groups
	ToolsConfig map[string]bool
}

// addTool registers a tool on the server. In normal mode, it delegates to the MCP server.
// In per-system builder mode (handlerMap != nil), it stores the handler and tool def locally.
func (s *Server) addTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	if s.handlerMap != nil {
		// Per-system builder mode: store handler and tool definition locally
		s.handlerMap[tool.Name] = handler
		s.toolMap[tool.Name] = tool
		return
	}
	// Normal single-system mode
	s.mcpServer.AddTool(tool, handler)
}

// NewServer creates a new MCP server for ABAP ADT tools.
func NewServer(cfg *Config) *Server {
	// Create ADT client
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

	// Configure safety settings
	safety := adt.UnrestrictedSafetyConfig() // Default: unrestricted for backwards compatibility
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
	opts = append(opts, adt.WithSafety(safety))

	// Create ADT client â€” HTTP or RFC mode
	var adtClient *adt.Client
	var sidecar *adt.SidecarManager

	if strings.EqualFold(cfg.ConnectionMode, "rfc") {
		// RFC mode: start JCo sidecar and use RfcTransport

		// Auto-extract embedded proxy JAR if configured path doesn't exist
		if cfg.JcoProxyJar == "" || !fileExists(cfg.JcoProxyJar) {
			if data := deps.GetEmbeddedProxyJar(); data != nil {
				extractDir := cfg.JcoLibsDir
				if extractDir == "" {
					extractDir = "./jco-libs"
				}
				proxyPath := filepath.Join(extractDir, "jco-proxy.jar")
				if err := os.MkdirAll(extractDir, 0755); err == nil {
					if err := os.WriteFile(proxyPath, data, 0644); err == nil {
						cfg.JcoProxyJar = proxyPath
						if cfg.Verbose {
							fmt.Fprintf(os.Stderr, "[VERBOSE] Auto-extracted embedded proxy JAR to %s\n", proxyPath)
						}
					}
				}
			}
		}

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
		sidecar = adt.NewSidecarManager(sidecarCfg)

		ctx := context.Background()
		if err := sidecar.Start(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to start JCo sidecar: %v\n", err)
			os.Exit(1)
		}

		maxConcurrent := cfg.RfcMaxConcurrent
		if maxConcurrent <= 0 {
			maxConcurrent = 5
		}

		if sidecar.IsSTDIO() {
			stdioTransport := adt.NewStdioRfcTransport(sidecar, adtCfg, maxConcurrent)
			adtClient = adt.NewClientWithTransport(adtCfg, stdioTransport)
		} else {
			rfcTransport := adt.NewRfcTransport(sidecar.URL(), adtCfg, maxConcurrent)
			adtClient = adt.NewClientWithTransport(adtCfg, rfcTransport)
		}
	} else {
		// HTTP mode (default)
		adtClient = adt.NewClient(cfg.BaseURL, cfg.Username, cfg.Password, opts...)
	}

	// Set terminal ID for debugger operations
	// Priority: 1) Custom ID (SAP GUI), 2) User-based ID
	if cfg.TerminalID != "" {
		adt.SetTerminalID(cfg.TerminalID)
	}
	adt.SetTerminalIDUser(cfg.Username)

	// Configure feature detection (safety network)
	featureConfig := adt.FeatureConfig{
		HANA:      parseFeatureMode(cfg.FeatureHANA),
		AbapGit:   parseFeatureMode(cfg.FeatureAbapGit),
		RAP:       parseFeatureMode(cfg.FeatureRAP),
		AMDP:      parseFeatureMode(cfg.FeatureAMDP),
		UI5:       parseFeatureMode(cfg.FeatureUI5),
		Transport: parseFeatureMode(cfg.FeatureTransport),
	}

	// Create feature prober
	featureProber := adt.NewFeatureProber(adtClient, featureConfig, cfg.Verbose)

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"mcp-abap-adt-go",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	s := &Server{
		mcpServer:     mcpServer,
		adtClient:     adtClient,
		config:        cfg,
		featureProber: featureProber,
		featureConfig: featureConfig,
		sidecar:       sidecar,
		asyncTasks:    make(map[string]*AsyncTask),
	}

	// Register tools based on mode, disabled groups, and granular tool config
	s.registerTools(cfg.Mode, cfg.DisabledGroups, cfg.ToolsConfig)

	// Start session keep-alive if configured
	if cfg.KeepAliveInterval > 0 {
		adtClient.StartKeepAlive(cfg.KeepAliveInterval, cfg.Verbose)
	}

	return s
}

// Shutdown gracefully stops the server and cleans up resources.
func (s *Server) Shutdown() {
	if s.sidecar != nil {
		s.sidecar.Stop()
	}
	// In multi-system mode, shut down all per-system servers
	if s.router != nil {
		for _, inst := range s.router.systems {
			inst.server.Shutdown()
		}
	}
}

// NewMultiSystemServer creates an MCP server that routes requests to multiple SAP systems.
// Each system gets its own goroutine-safe Server instance with independent connections.
func NewMultiSystemServer(globalCfg *Config) (*Server, error) {
	if len(globalCfg.MultiSystems) == 0 {
		return nil, fmt.Errorf("multi-system mode requires at least one system in configuration")
	}

	// Auto-extract embedded proxy JAR once for all systems (shared JCo infrastructure)
	if globalCfg.JcoProxyJar == "" || !fileExists(globalCfg.JcoProxyJar) {
		if data := deps.GetEmbeddedProxyJar(); data != nil {
			extractDir := globalCfg.JcoLibsDir
			if extractDir == "" {
				extractDir = "./jco-libs"
			}
			proxyPath := filepath.Join(extractDir, "jco-proxy.jar")
			if err := os.MkdirAll(extractDir, 0755); err == nil {
				if err := os.WriteFile(proxyPath, data, 0644); err == nil {
					globalCfg.JcoProxyJar = proxyPath
					if globalCfg.Verbose {
						fmt.Fprintf(os.Stderr, "[VERBOSE] Auto-extracted embedded proxy JAR to %s\n", proxyPath)
					}
				}
			}
		}
	}

	// Collect system IDs
	systemIDs := make([]string, 0, len(globalCfg.MultiSystems))
	for id := range globalCfg.MultiSystems {
		systemIDs = append(systemIDs, id)
	}

	router := newMultiSystemRouter(systemIDs)

	// Create the main MCP server
	mcpServer := server.NewMCPServer(
		"mcp-abap-adt-go",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	mainServer := &Server{
		mcpServer:   mcpServer,
		multiSystem: true,
		router:      router,
		asyncTasks:  make(map[string]*AsyncTask),
		config:      globalCfg,
	}

	// Create per-system Server instances, each in its own goroutine context
	for sysID, sysCfg := range globalCfg.MultiSystems {
		perSystemCfg := systemConfigForMCP(sysID, sysCfg, globalCfg)

		// Create a per-system server that registers handlers locally (not on MCP server)
		perSystemServer := newPerSystemServer(perSystemCfg)
		if perSystemServer == nil {
			return nil, fmt.Errorf("failed to create server for system %q", sysID)
		}

		router.addSystem(sysID, perSystemServer)

		// Register tools on the per-system server (populates handlerMap)
		perSystemServer.registerTools(globalCfg.Mode, globalCfg.DisabledGroups, globalCfg.ToolsConfig)

		// Copy handlers from per-system server to router
		for toolName, handler := range perSystemServer.handlerMap {
			router.registerHandler(toolName, sysID, handler)
		}

		if globalCfg.Verbose {
			connInfo := perSystemCfg.BaseURL
			if strings.EqualFold(perSystemCfg.ConnectionMode, "rfc") {
				connInfo = fmt.Sprintf("RFC(%s)", perSystemCfg.AsHost)
				if perSystemCfg.MsHost != "" {
					connInfo = fmt.Sprintf("RFC-LB(%s)", perSystemCfg.MsHost)
				}
			}
			fmt.Fprintf(os.Stderr, "[VERBOSE] Multi-system: initialized %q → %s (user: %s, %d tools)\n",
				sysID, connInfo, perSystemCfg.Username, len(perSystemServer.handlerMap))
		}
	}

	// Register tools on main MCP server with system_id routing.
	// Use the first system's handler map as the canonical tool set.
	// All systems have the same tools registered (same mode/disabledGroups).
	mainServer.registerMultiSystemTools(router, globalCfg.Mode, globalCfg.DisabledGroups, globalCfg.ToolsConfig)

	return mainServer, nil
}

// newPerSystemServer creates a Server instance for a single system.
// It registers handlers in handlerMap instead of on the MCP server.
func newPerSystemServer(cfg *Config) *Server {
	// Create ADT client options
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

	// Safety settings
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
	opts = append(opts, adt.WithSafety(safety))

	// Create ADT client — HTTP or RFC mode
	var adtClient *adt.Client
	var sidecar *adt.SidecarManager

	if strings.EqualFold(cfg.ConnectionMode, "rfc") {
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
		sidecar = adt.NewSidecarManager(sidecarCfg)

		ctx := context.Background()
		if err := sidecar.Start(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to start JCo sidecar for system: %v\n", err)
			return nil
		}

		maxConcurrent := cfg.RfcMaxConcurrent
		if maxConcurrent <= 0 {
			maxConcurrent = 5
		}

		if sidecar.IsSTDIO() {
			stdioTransport := adt.NewStdioRfcTransport(sidecar, adtCfg, maxConcurrent)
			adtClient = adt.NewClientWithTransport(adtCfg, stdioTransport)
		} else {
			rfcTransport := adt.NewRfcTransport(sidecar.URL(), adtCfg, maxConcurrent)
			adtClient = adt.NewClientWithTransport(adtCfg, rfcTransport)
		}
	} else {
		// HTTP mode
		adtClient = adt.NewClient(cfg.BaseURL, cfg.Username, cfg.Password, opts...)
	}

	// Terminal ID
	if cfg.TerminalID != "" {
		adt.SetTerminalID(cfg.TerminalID)
	}
	adt.SetTerminalIDUser(cfg.Username)

	// Feature detection
	featureConfig := adt.FeatureConfig{
		HANA:      parseFeatureMode(cfg.FeatureHANA),
		AbapGit:   parseFeatureMode(cfg.FeatureAbapGit),
		RAP:       parseFeatureMode(cfg.FeatureRAP),
		AMDP:      parseFeatureMode(cfg.FeatureAMDP),
		UI5:       parseFeatureMode(cfg.FeatureUI5),
		Transport: parseFeatureMode(cfg.FeatureTransport),
	}
	featureProber := adt.NewFeatureProber(adtClient, featureConfig, cfg.Verbose)

	return &Server{
		adtClient:     adtClient,
		config:        cfg,
		featureProber: featureProber,
		featureConfig: featureConfig,
		sidecar:       sidecar,
		asyncTasks:    make(map[string]*AsyncTask),
		handlerMap:    make(map[string]server.ToolHandlerFunc), // builder mode
		toolMap:       make(map[string]mcp.Tool),               // builder mode
	}
}

// registerMultiSystemTools registers all tools on the main MCP server with system_id routing.
// It uses the first system's tool definitions and wires each to the router.
func (s *Server) registerMultiSystemTools(router *multiSystemRouter, mode string, disabledGroups string, toolsConfig map[string]bool) {
	// Pick any system to get the canonical tool definitions
	var canonicalSystem *Server
	for _, inst := range router.systems {
		canonicalSystem = inst.server
		break
	}
	if canonicalSystem == nil {
		return
	}

	// For each tool, add it to the main MCP server with system_id parameter and routing
	for toolName, tool := range canonicalSystem.toolMap {
		toolWithSystemID := addSystemIDToTool(tool, router.systemIDs)
		s.mcpServer.AddTool(toolWithSystemID, router.routeHandler(toolName))
	}
}

// isRfcMode returns true if the server is running in RFC mode.
func (s *Server) isRfcMode() bool {
	return s.sidecar != nil
}

// fileExists returns true if the path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// parseFeatureMode converts string to FeatureMode
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

// ServeStdio starts the MCP server on stdin/stdout.
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}

// registerTools is now in tools_register.go (decomposed)

// newToolResultError creates an error result for tool execution failures.
func newToolResultError(message string) *mcp.CallToolResult {
	result := mcp.NewToolResultText(message)
	result.IsError = true
	return result
}

// ensureWSConnected ensures the WebSocket client is connected, creating it if needed.
// Returns error result if connection fails, nil on success.
func (s *Server) ensureWSConnected(ctx context.Context, toolName string) *mcp.CallToolResult {
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
func (s *Server) requireActiveAMDPSession() *mcp.CallToolResult {
	if s.amdpWSClient == nil || !s.amdpWSClient.IsActive() {
		return newToolResultError("No active AMDP session. Use AMDPDebuggerStart first.")
	}
	return nil
}

// Tool handlers are in separate files:
// - handlers_read.go: GetProgram, GetClass, GetTable, etc.
// - handlers_system.go: GetSystemInfo, GetFeatures, etc.
// - handlers_analysis.go: GetCallGraph, TraceExecution, etc.
// - handlers_codeintel.go: FindDefinition, FindReferences, CodeCompletion, etc.
// - handlers_devtools.go: SyntaxCheck, Activate, ATC, etc.
// - handlers_crud.go: Lock, Create, Update, Delete, etc.
// - handlers_debugger.go: SetBreakpoint, DebuggerListen, etc.
// - handlers_amdp.go: AMDPDebugger* handlers
// - handlers_ui5.go: UI5ListApps, UI5GetApp, etc.
// - handlers_git.go: GitTypes, GitExport
// - handlers_report.go: RunReport, GetVariants, etc.
// - handlers_install.go: (removed)
// - handlers_transport.go: ListTransports, GetTransport, etc.
//
// Tool registration is in:
// - tools_register.go: registerTools() and all register*Tools() methods
// - tools_groups.go: toolGroups() - group definitions for --disabled-groups
// - tools_focused.go: focusedToolSet() - focused mode whitelist
// - tools_aliases.go: registerToolAliases() - short alias names
