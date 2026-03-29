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
	multiSystem bool                             // true when --multi-system is active
	router      *Router                          // routes requests to per-system servers (multi-system only)
	handlerMap  map[string]types.ToolHandlerFunc // local handler registry (per-system builder mode)
	toolMap     map[string]mcp.Tool              // tool definitions (per-system builder mode)

	// Async task management
	asyncTasks   map[string]*AsyncTask
	asyncTasksMu sync.RWMutex
	asyncTaskID  int64
}

// ADT implements types.System
func (s *Server) ADT() *adt.Client {
	return s.adtClient
}

// Config implements types.System
func (s *Server) Config() any {
	return s.config
}

// IsRfcMode implements types.System
func (s *Server) IsRfcMode() bool {
	return s.sidecar != nil
}

// Sidecar implements types.System
func (s *Server) Sidecar() *adt.SidecarManager {
	return s.sidecar
}

// RequireActiveAMDPSession implements types.System
func (s *Server) RequireActiveAMDPSession() *mcp.CallToolResult {
	if s.amdpWSClient == nil || !s.amdpWSClient.IsActive() {
		return types.ErrorResult("No active AMDP debug session. Start one using StartAMDPDebugSession tool.")
	}
	return nil
}

// EnsureWSConnected implements types.System
func (s *Server) EnsureWSConnected(ctx context.Context, toolName string) *mcp.CallToolResult {
	return s.ensureWSConnected(ctx, toolName)
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
func (s *Server) addTool(tool mcp.Tool, handler types.ToolHandlerFunc) {
	if s.handlerMap != nil {
		// Per-system builder mode: store handler and tool definition locally
		s.handlerMap[tool.Name] = handler
		s.toolMap[tool.Name] = tool
		return
	}
	// Normal single-system mode
	s.mcpServer.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handler(ctx, s, request)
	})
}

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

// newServerInstance creates a Server with an ADT client, feature prober, and optional sidecar.
// It does NOT create an mcpServer or register tools — that is the caller's responsibility.
func newServerInstance(cfg *Config) (*Server, error) {
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

	s := &Server{
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

	return s, nil
}

// newMCPServer creates the underlying mcp-go MCPServer instance.
func newMCPServer() *server.MCPServer {
	return server.NewMCPServer(
		"mcp-abap-adt-go",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)
}

// NewServer creates a new MCP server for ABAP ADT tools (single-system mode).
func NewServer(cfg *Config) *Server {
	s, err := newServerInstance(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}

	mcpSrv := newMCPServer()
	router := NewRouter(mcpSrv)
	s.mcpServer = mcpSrv
	s.router = router

	router.AddSystem("default", s)
	router.RegisterTools(cfg.Mode, cfg.DisabledGroups, cfg.ToolsConfig)

	return s
}

// Shutdown gracefully stops the server and cleans up resources.
func (s *Server) Shutdown() {
	if s.sidecar != nil {
		s.sidecar.Stop()
	}
	// In multi-system mode, shut down all per-system servers
	if s.router != nil {
		for _, sys := range s.router.systems {
			if srv, ok := sys.(*Server); ok {
				srv.Shutdown()
			}
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
	ensureProxyJAR(globalCfg)

	mcpSrv := newMCPServer()
	router := NewRouter(mcpSrv)

	mainServer := &Server{
		mcpServer:   mcpSrv,
		multiSystem: true,
		router:      router,
		asyncTasks:  make(map[string]*AsyncTask),
		config:      globalCfg,
	}

	// Create per-system Server instances, each in its own goroutine context
	for sysID, sysCfg := range globalCfg.MultiSystems {
		perSystemCfg := systemConfigForMCP(sysID, sysCfg, globalCfg)

		perSystemServer, err := newPerSystemServer(perSystemCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create server for system %q: %w", sysID, err)
		}

		router.AddSystem(sysID, perSystemServer)
		perSystemServer.router = router // Allow per-system server to access router if needed

		if globalCfg.Verbose {
			connInfo := perSystemCfg.BaseURL
			if strings.EqualFold(perSystemCfg.ConnectionMode, "rfc") {
				connInfo = fmt.Sprintf("RFC(%s)", perSystemCfg.AsHost)
				if perSystemCfg.MsHost != "" {
					connInfo = fmt.Sprintf("RFC-LB(%s)", perSystemCfg.MsHost)
				}
			}
			fmt.Fprintf(os.Stderr, "[VERBOSE] Multi-system: initialized %q → %s (user: %s)\n",
				sysID, connInfo, perSystemCfg.Username)
		}
	}

	// Register tools on the router (which registers them on the main MCP server)
	router.RegisterTools(globalCfg.Mode, globalCfg.DisabledGroups, globalCfg.ToolsConfig)

	return mainServer, nil
}

// newPerSystemServer creates a Server instance for use in multi-system mode.
// It uses newServerInstance for the heavy lifting, then enables builder mode
// (handlerMap/toolMap) so tools register locally instead of on an MCP server.
func newPerSystemServer(cfg *Config) (*Server, error) {
	s, err := newServerInstance(cfg)
	if err != nil {
		return nil, err
	}
	s.handlerMap = make(map[string]types.ToolHandlerFunc)
	s.toolMap = make(map[string]mcp.Tool)
	return s, nil
}

// registerMultiSystemTools registers all tools on the main MCP server with system_id routing.
// It uses the first system's tool definitions and wires each to the router.
func (s *Server) registerMultiSystemTools(router *Router, mode string, disabledGroups string, toolsConfig map[string]bool) {
	// Handled by NewMultiSystemServer and Router now
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

// Tool definitions and handlers are colocated in tool_*.go files:
// - tool_read.go: readToolDefs + GetProgram, GetClass, GetTable, etc.
// - tool_system.go: systemToolDefs + GetSystemInfo, GetFeatures, etc.
// - tool_analysis.go: analysisToolDefs + GetCallGraph, TraceExecution, etc.
// - tool_codeintel.go: codeIntelToolDefs + FindDefinition, FindReferences, etc.
// - tool_devtools.go: devToolDefs + SyntaxCheck, Activate, ATC, etc.
// - tool_crud.go: crudToolDefs + Lock, Create, Update, Delete, etc.
// - tool_debugger.go: debuggerToolDefs + SetBreakpoint, DebuggerListen, etc.
// - tool_amdp.go: amdpToolDefs + AMDPDebugger* handlers
// - tool_ui5.go: ui5ToolDefs + UI5ListApps, UI5GetApp, etc.
// - tool_git.go: gitToolDefs + GitTypes, GitExport
// - tool_report.go: reportToolDefs + RunReport, GetVariants, etc.
// - tool_transport.go: transportToolDefs + ListTransports, etc.
// - tool_source.go: unifiedToolDefs + GetSource, WriteSource, grep/file defs
// - tool_fileio.go: fileToolDefs + editToolDefs + DeployFromFile, EditSource, etc.
// - tool_dumps.go: dumpToolDefs + ListDumps, GetDump
// - tool_traces.go: traceToolDefs + ListTraces, GetTrace
// - tool_sqltrace.go: sqlTraceToolDefs + GetSQLTraceState, ListSQLTraces
// - tool_grep.go: grepToolDefs + GrepObject, GrepPackage
// - tool_classinclude.go: classIncludeToolDefs + GetClassInclude, etc.
// - tool_workflow.go: workflowToolDefs + WriteProgram, CreateClassWithTests, etc.
// - tool_universal.go: SAP() universal tool dispatcher
// - tool_help.go: help text generation
//
// Tool registration infrastructure:
// - tools_register.go: toolDef type, registerTools(), allToolDefs(), buildShouldRegister()
// - tools_aliases.go: registerToolAliases() - short alias names
