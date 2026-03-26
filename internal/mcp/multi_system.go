// Package mcp provides the MCP server implementation for ABAP ADT tools.
// multi_system.go implements multi-system support for routing tool requests
// to different SAP system connections based on a system_id parameter.
package mcp

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// systemInstance holds all per-system state: ADT client, sidecars, feature probing, etc.
// Each instance is a fully independent Server that maintains its own SAP connection.
type systemInstance struct {
	id     string  // system ID (lowercase)
	server *Server // full Server instance with its own adtClient, sidecar, etc.
}

// multiSystemRouter manages routing of tool requests to per-system Server instances.
type multiSystemRouter struct {
	systems   map[string]*systemInstance                   // system_id (lowercase) -> instance
	systemIDs []string                                     // sorted list of original system IDs (for display)
	handlers  map[string]map[string]server.ToolHandlerFunc // toolName -> systemID -> handler
	mu        sync.RWMutex                                 // protects handlers map
}

// newMultiSystemRouter creates a new router for the given systems.
func newMultiSystemRouter(systemIDs []string) *multiSystemRouter {
	sorted := make([]string, len(systemIDs))
	copy(sorted, systemIDs)
	sort.Strings(sorted)

	return &multiSystemRouter{
		systems:   make(map[string]*systemInstance),
		systemIDs: sorted,
		handlers:  make(map[string]map[string]server.ToolHandlerFunc),
	}
}

// addSystem registers a per-system Server instance.
func (r *multiSystemRouter) addSystem(id string, srv *Server) {
	r.systems[strings.ToLower(id)] = &systemInstance{
		id:     strings.ToLower(id),
		server: srv,
	}
}

// registerHandler stores a handler for a specific tool on a specific system.
func (r *multiSystemRouter) registerHandler(toolName, systemID string, handler server.ToolHandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.handlers[toolName] == nil {
		r.handlers[toolName] = make(map[string]server.ToolHandlerFunc)
	}
	r.handlers[toolName][strings.ToLower(systemID)] = handler
}

// routeHandler returns a handler that extracts system_id and dispatches to the right per-system handler.
func (r *multiSystemRouter) routeHandler(toolName string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		systemID := getStringParam(request.Params.Arguments, "system_id")
		if systemID == "" {
			return newToolResultError(fmt.Sprintf(
				"system_id is required. Available systems: %s",
				strings.Join(r.systemIDs, ", "),
			)), nil
		}

		r.mu.RLock()
		systemHandlers, ok := r.handlers[toolName]
		r.mu.RUnlock()
		if !ok {
			return newToolResultError(fmt.Sprintf("Tool %s not registered", toolName)), nil
		}

		handler, ok := systemHandlers[strings.ToLower(systemID)]
		if !ok {
			return newToolResultError(fmt.Sprintf(
				"Unknown system_id: %q. Available systems: %s",
				systemID, strings.Join(r.systemIDs, ", "),
			)), nil
		}

		return handler(ctx, request)
	}
}

// addSystemIDToTool injects the system_id parameter into a tool's schema.
func addSystemIDToTool(tool mcp.Tool, systemIDs []string) mcp.Tool {
	if tool.InputSchema.Properties == nil {
		tool.InputSchema.Properties = make(map[string]interface{})
	}
	tool.InputSchema.Properties["system_id"] = map[string]interface{}{
		"type":        "string",
		"description": fmt.Sprintf("Target SAP system ID. Available: %s", strings.Join(systemIDs, ", ")),
		"enum":        systemIDs,
	}
	tool.InputSchema.Required = append(tool.InputSchema.Required, "system_id")
	return tool
}

// systemConfigForMCP converts a config.SystemConfig to an mcp.Config for creating per-system servers.
func systemConfigForMCP(sysID string, sysCfg *SystemConfigResolved, globalCfg *Config) *Config {
	c := &Config{
		// Per-system connection settings
		BaseURL:            sysCfg.BaseURL,
		Username:           sysCfg.Username,
		Password:           sysCfg.Password,
		Client:             sysCfg.Client,
		Language:           sysCfg.Language,
		InsecureSkipVerify: sysCfg.InsecureSkipVerify,
		Cookies:            sysCfg.Cookies,

		// Per-system safety settings
		ReadOnly:        sysCfg.ReadOnly,
		AllowedPackages: sysCfg.AllowedPackages,

		// Per-system RFC settings
		ConnectionMode:   sysCfg.ConnectionMode,
		AsHost:           sysCfg.AsHost,
		SysNr:            sysCfg.SysNr,
		MsHost:           sysCfg.MsHost,
		MsServ:           sysCfg.MsServ,
		R3Name:           sysCfg.R3Name,
		Group:            sysCfg.Group,
		JcoProxyJar:      sysCfg.JcoProxyJar,
		JcoLibsDir:       globalCfg.JcoLibsDir,
		JavaPath:         sysCfg.JavaPath,
		RfcProxyPort:     sysCfg.RfcProxyPort,
		RfcMaxConcurrent: sysCfg.RfcMaxConcurrent,
		SidecarTransport: sysCfg.SidecarTransport,

		// Per-system SNC settings
		SNC:           sysCfg.SNC,
		SysID:         sysCfg.SysID,
		LandscapeFile: sysCfg.LandscapeFile,
		JcoProperties: sysCfg.JcoProperties,

		// Global settings (shared across all systems)
		Verbose:        globalCfg.Verbose || sysCfg.Verbose,
		Mode:           globalCfg.Mode,
		DisabledGroups: globalCfg.DisabledGroups,
		ToolsConfig:    globalCfg.ToolsConfig,

		// Global safety settings (can be overridden per-system)
		BlockFreeSQL:            globalCfg.BlockFreeSQL,
		AllowedOps:              globalCfg.AllowedOps,
		DisallowedOps:           globalCfg.DisallowedOps,
		EnableTransports:        globalCfg.EnableTransports,
		TransportReadOnly:       globalCfg.TransportReadOnly,
		AllowedTransports:       globalCfg.AllowedTransports,
		AllowTransportableEdits: globalCfg.AllowTransportableEdits,

		// Global feature settings
		FeatureHANA:      globalCfg.FeatureHANA,
		FeatureAbapGit:   globalCfg.FeatureAbapGit,
		FeatureRAP:       globalCfg.FeatureRAP,
		FeatureAMDP:      globalCfg.FeatureAMDP,
		FeatureUI5:       globalCfg.FeatureUI5,
		FeatureTransport: globalCfg.FeatureTransport,

		// Global debugger settings
		TerminalID: globalCfg.TerminalID,
	}

	// Per-system safety overrides
	if sysCfg.ReadOnly {
		c.ReadOnly = true
	}
	if len(sysCfg.AllowedPackages) > 0 {
		c.AllowedPackages = sysCfg.AllowedPackages
	}

	// Sidecar transport: per-system overrides global, with global as fallback
	if c.SidecarTransport == "" {
		c.SidecarTransport = globalCfg.SidecarTransport
	}

	// JCo infrastructure: per-system overrides global, with global as fallback
	if c.JcoProxyJar == "" {
		c.JcoProxyJar = globalCfg.JcoProxyJar
	}
	if c.JavaPath == "" {
		c.JavaPath = globalCfg.JavaPath
	}

	return c
}

// SystemConfigResolved holds a fully resolved system configuration ready for use.
type SystemConfigResolved struct {
	BaseURL            string
	Username           string
	Password           string
	Client             string
	Language           string
	InsecureSkipVerify bool
	Cookies            map[string]string

	ReadOnly        bool
	AllowedPackages []string

	ConnectionMode   string
	AsHost           string
	SysNr            string
	MsHost           string
	MsServ           string
	R3Name           string
	Group            string
	JcoProxyJar      string
	JavaPath         string
	RfcProxyPort     int
	RfcMaxConcurrent int
	SidecarTransport string

	// SNC/SSO configuration
	SNC           bool
	SysID         string
	LandscapeFile string
	JcoProperties map[string]string // Resolved JCo properties (populated during SNC resolution)

	// Per-system verbose
	Verbose bool
}
