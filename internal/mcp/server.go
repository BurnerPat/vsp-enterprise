// Package mcp provides the MCP server implementation for ABAP ADT tools.
// server.go contains the Server singleton responsible for bootstrapping the MCP server,
// managing system connections via the Router, and handling setup/shutdown lifecycle.
package mcp

import (
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	"github.com/oisee/vibing-steampunk/internal/config"
)

// Server is the singleton MCP server instance.
// It bootstraps the mcp-go server, manages individual System connections
// via the Router, and handles setup/shutdown lifecycle.
type Server struct {
	mcpServer *server.MCPServer
	router    *Router
	config    *GlobalConfig // top-level configuration
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

// NewServer creates a new MCP server from a fully populated GlobalConfig.
// All systems (single or multi) must already be present in globalCfg.Systems.
func NewServer(globalCfg *GlobalConfig) (*Server, error) {
	if len(globalCfg.Systems) == 0 {
		return nil, fmt.Errorf("no systems configured")
	}

	// Store the global config as a singleton so that Build* methods and
	// other code can access global settings without field copying.
	config.SetInstance(globalCfg)

	mcpSrv := newMCPServer()
	router := NewRouter(mcpSrv)

	for sysID, sysCfg := range globalCfg.Systems {
		sys, err := newSystemInstance(sysCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create system for %q: %w", sysID, err)
		}

		router.AddSystem(sysID, sys)

		if globalCfg.Verbose {
			connInfo := sysCfg.URL
			if strings.EqualFold(sysCfg.ConnectionMode, "rfc") {
				connInfo = fmt.Sprintf("RFC(%s)", sysCfg.AsHost)
				if sysCfg.MsHost != "" {
					connInfo = fmt.Sprintf("RFC-LB(%s)", sysCfg.MsHost)
				}
			}
			fmt.Fprintf(os.Stderr, "[VERBOSE] Initialized system %q → %s (user: %s)\n",
				sysID, connInfo, sysCfg.User)
		}
	}

	router.RegisterTools(globalCfg.Mode, globalCfg.DisabledGroups, globalCfg.ToolsConfig)

	return &Server{
		mcpServer: mcpSrv,
		router:    router,
		config:    globalCfg,
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
