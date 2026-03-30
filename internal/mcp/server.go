// Package mcp provides the MCP server implementation for ABAP ADT tools.
// server.go contains the Server singleton responsible for bootstrapping the MCP server,
// managing system connections via the Router, and handling setup/shutdown lifecycle.
package mcp

import (
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/server"
)

// Server is the singleton MCP server instance.
// It bootstraps the mcp-go server, manages individual System connections
// via the Router, and handles setup/shutdown lifecycle.
type Server struct {
	mcpServer   *server.MCPServer
	router      *Router
	multiSystem bool    // true when --multi-system is active
	config      *Config // global configuration (type alias for config.ResolvedConfig)
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
		perSystemCfg := *sysCfg // copy to avoid mutating the original
		perSystemCfg.MergeGlobal(globalCfg)

		sys, err := newSystemInstance(&perSystemCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create system for %q: %w", sysID, err)
		}

		router.AddSystem(sysID, sys)

		if globalCfg.Verbose {
			connInfo := perSystemCfg.URL
			if strings.EqualFold(perSystemCfg.ConnectionMode, "rfc") {
				connInfo = fmt.Sprintf("RFC(%s)", perSystemCfg.AsHost)
				if perSystemCfg.MsHost != "" {
					connInfo = fmt.Sprintf("RFC-LB(%s)", perSystemCfg.MsHost)
				}
			}
			fmt.Fprintf(os.Stderr, "[VERBOSE] Multi-system: initialized %q → %s (user: %s)\n",
				sysID, connInfo, perSystemCfg.User)
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
