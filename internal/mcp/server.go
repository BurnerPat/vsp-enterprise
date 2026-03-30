// Package mcp provides the MCP server implementation for ABAP ADT tools.
// server.go contains the Server singleton responsible for bootstrapping the MCP server,
// managing system connections via the Router, and handling setup/shutdown lifecycle.
package mcp

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// Server is the singleton MCP server instance.
// It bootstraps the mcp-go server, manages individual System connections
// via the Router, and handles setup/shutdown lifecycle.
type Server struct {
	mcpServer   *server.MCPServer
	router      *Router
	multiSystem bool    // true when --multi-system is active
	config      *Config // global configuration
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
	sys, err := newSystemInstance(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}

	mcpSrv := newMCPServer()
	router := NewRouter(mcpSrv)

	router.AddSystem("default", sys)
	router.RegisterTools(cfg.Mode, cfg.DisabledGroups, cfg.ToolsConfig)

	return &Server{
		mcpServer: mcpSrv,
		router:    router,
		config:    cfg,
	}
}

// NewMultiSystemServer creates an MCP server that routes requests to multiple SAP systems.
// Each system gets its own goroutine-safe System instance with independent connections.
func NewMultiSystemServer(globalCfg *Config) (*Server, error) {
	if len(globalCfg.MultiSystems) == 0 {
		return nil, fmt.Errorf("multi-system mode requires at least one system in configuration")
	}

	// Auto-extract embedded proxy JAR once for all systems (shared JCo infrastructure)
	ensureProxyJAR(globalCfg)

	mcpSrv := newMCPServer()
	router := NewRouter(mcpSrv)

	// Create per-system System instances, each with independent connections
	for sysID, sysCfg := range globalCfg.MultiSystems {
		perSystemCfg := systemConfigForMCP(sysID, sysCfg, globalCfg)

		sys, err := newSystemInstance(perSystemCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create system for %q: %w", sysID, err)
		}

		router.AddSystem(sysID, sys)

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

	return &Server{
		mcpServer:   mcpSrv,
		multiSystem: true,
		router:      router,
		config:      globalCfg,
	}, nil
}

// Shutdown gracefully stops the server and cleans up all system resources.
func (s *Server) Shutdown() {
	if s.router != nil {
		for _, sys := range s.router.systems {
			if system, ok := sys.(*System); ok {
				system.Shutdown()
			}
		}
	}
}

// ServeStdio starts the MCP server on stdin/stdout.
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}
