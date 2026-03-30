// Package mcp provides the MCP server implementation for ABAP ADT tools.
// server.go contains the Server singleton responsible for bootstrapping the MCP server,
// managing system connections via the Router, and handling setup/shutdown lifecycle.
package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	"github.com/oisee/vibing-steampunk/embedded/deps"
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

	proxyJarChecked := false

	for sysID, sysCfg := range globalCfg.Systems {
		if strings.EqualFold(sysCfg.ConnectionMode, "rfc") && !proxyJarChecked {
			proxyJarChecked = true

			if err := ensureProxyJARIsAvailable(globalCfg); err != nil {
				return nil, fmt.Errorf("failed to ensure proxy JAR availability for system %q: %w", sysID, err)
			}
		}

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

func ensureProxyJARIsAvailable(cfg *config.ResolvedConfig) error {
	if cfg.JcoProxyJar != "" {
		if !fileExists(cfg.JcoProxyJar) {
			return fmt.Errorf("proxy JAR file %s supplied by user does not exist", cfg.JcoProxyJar)
		}

		return nil
	}

	data := deps.GetEmbeddedProxyJar()
	if data == nil {
		panic("Embedded proxy JAR not found")
	}

	extractDir := cfg.JcoLibsDir
	if extractDir == "" {
		extractDir = "./jco-libs"
	}
	proxyPath := filepath.Join(extractDir, "jco-proxy.jar")

	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory for proxy JAR: %s, %v", proxyPath, err)
	}
	if err := os.WriteFile(proxyPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write proxy JAR to %s: %v", proxyPath, err)
	}

	cfg.JcoProxyJar = proxyPath

	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Auto-extracted embedded proxy JAR to %s\n", proxyPath)
	}

	return nil
}

// fileExists returns true if the path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
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
