package mcp

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/oisee/vibing-steampunk/internal/config"
	"github.com/oisee/vibing-steampunk/internal/mcp/tools"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

type SystemInRouter struct {
	System       types.System
	ID           string
	EnabledTools []types.ToolDef
}

// Router handles tool call routing and system management.
type Router struct {
	mcpServer *server.MCPServer
	systems   map[string]SystemInRouter
	systemIDs []string
}

func NewRouter(mcpServer *server.MCPServer) *Router {
	return &Router{
		mcpServer: mcpServer,
		systems:   make(map[string]SystemInRouter),
	}
}

func (r *Router) AddSystem(id string, sys types.System) {
	r.systems[strings.ToLower(id)] = SystemInRouter{
		System: sys,
		ID:     id,
	}

	r.systemIDs = append(r.systemIDs, id)
}

// RegisterTools registers all available tools from all packages.
func (r *Router) RegisterTools(cfg *config.GlobalConfig) {
	allDefs, err := r.resolvePermissionConfig(cfg)

	if err != nil {
		panic(err)
	}

	// In the new architecture, we register each tool once with a central handler
	for _, td := range allDefs {
		tool := td.Tool
		// If multiple systems, inject system_id
		if len(r.systemIDs) > 1 {
			tool = r.addSystemIDToTool(tool)
		}

		// Capture td for the closure
		toolDef := td
		r.mcpServer.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return r.HandleToolCall(ctx, toolDef, request)
		})
	}
}

func (r *Router) HandleToolCall(ctx context.Context, td types.ToolDef, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 1. Determine target system
	systemID, _ := request.GetArguments()["system_id"].(string)

	if systemID == "" && len(r.systemIDs) == 1 {
		systemID = r.systemIDs[0]
	}

	if systemID == "" {
		return types.ErrorResult(fmt.Sprintf("system_id is required. Available: %s", strings.Join(r.systemIDs, ", "))), nil
	}

	sys, ok := r.systems[strings.ToLower(systemID)]
	if !ok {
		return types.ErrorResult(fmt.Sprintf("Unknown system: %s", systemID)), nil
	}

	// 2. Permission check
	if slices.IndexFunc(sys.EnabledTools, func(t types.ToolDef) bool { return t.Tool.Name == td.Tool.Name }) == -1 {
		return types.ErrorResult(fmt.Sprintf("Permission denied for tool %s on system %s", td.Tool.Name, systemID)), nil
	}

	// 3. Invoke handler
	return td.Handler(ctx, sys.System, request)
}

func (r *Router) addSystemIDToTool(tool mcp.Tool) mcp.Tool {
	if tool.InputSchema.Properties == nil {
		tool.InputSchema.Properties = make(map[string]interface{})
	}

	// Create a new map for properties to avoid modifying original
	newProps := make(map[string]interface{})

	for k, v := range tool.InputSchema.Properties {
		newProps[k] = v
	}

	newProps["system_id"] = map[string]interface{}{
		"type":        "string",
		"description": fmt.Sprintf("Target SAP system ID. Available: %s", strings.Join(r.systemIDs, ", ")),
		"enum":        r.systemIDs,
	}

	tool.InputSchema.Properties = newProps
	tool.InputSchema.Required = append(tool.InputSchema.Required, "system_id")

	return tool
}

func (r *Router) resolvePermissionConfig(cfg *config.GlobalConfig) ([]types.ToolDef, error) {
	allToolDefs := r.allToolDefs()

	// Identify which tools are enabled globally (in the config root permissions)
	enabledTools, err := resolvePermissions(&cfg.Permissions, &allToolDefs)

	// Calculate effective enabled tools for all SystemClasses
	var toolsPerSystemClass = make(map[string][]types.ToolDef)

	if cfg.SystemClasses != nil {
		var allSystemClassTools = make([]string, 0)

		for name, systemClass := range cfg.SystemClasses {
			systemClassTools, err := resolvePermissions(&systemClass.Permissions, &enabledTools)

			if err != nil {
				return nil, err
			}

			toolsPerSystemClass[name] = systemClassTools

			for _, tool := range systemClassTools {
				allSystemClassTools = append(allSystemClassTools, tool.Tool.Name)
			}
		}

		enabledTools = slices.DeleteFunc(enabledTools, func(tool types.ToolDef) bool {
			return !slices.Contains(allSystemClassTools, tool.Tool.Name)
		})
	}

	// Calculate effective enabled tools for each system
	for name, system := range cfg.Systems {
		var allSystemTools = make([]string, 0)

		var effectiveTools []types.ToolDef

		if system.SystemClass != "" {
			effectiveTools = toolsPerSystemClass[system.SystemClass]

			if effectiveTools == nil {
				return nil, fmt.Errorf("system class %q not found", system.SystemClass)
			}
		} else {
			effectiveTools = enabledTools
		}

		systemTools, err := resolvePermissions(&system.Permissions, &enabledTools)

		if err != nil {
			return nil, err
		}

		sys := r.systems[name]
		sys.EnabledTools = systemTools

		for _, tool := range systemTools {
			allSystemTools = append(allSystemTools, tool.Tool.Name)
		}

		enabledTools = slices.DeleteFunc(enabledTools, func(tool types.ToolDef) bool {
			return !slices.Contains(allSystemTools, tool.Tool.Name)
		})
	}

	return enabledTools, err
}

func resolvePermissions(permissions *config.PermissionConfig, availableTools *[]types.ToolDef) ([]types.ToolDef, error) {
	return slices.DeleteFunc(*availableTools, func(tool types.ToolDef) bool {
		for pattern, enabled := range permissions.Tools {
			if simpleGlobMatch(pattern, tool.Tool.Name) {
				return !enabled
			}
		}

		return permissions.DenyToolsByDefault
	}), nil
}

func simpleGlobMatch(pattern string, str string) bool {
	// Simple glob matching using '*' as wildcard character
	if pattern == "*" {
		return true
	}

	parts := strings.Split(pattern, "*")

	for _, part := range parts {
		idx := strings.Index(str, part)

		if idx == -1 {
			return false
		}

		str = str[idx+len(part):]
	}

	return strings.HasSuffix(pattern, "*") || str == ""
}

//goland:noinspection DuplicatedCode
func (r *Router) allToolDefs() []types.ToolDef {
	var defs []types.ToolDef
	defs = append(defs, tools.SystemToolDefs()...)
	defs = append(defs, tools.ReadToolDefs()...)
	defs = append(defs, tools.UnifiedToolDefs()...)
	defs = append(defs, tools.GrepSourceToolDefs()...)
	defs = append(defs, tools.FileSourceToolDefs()...)
	defs = append(defs, tools.AnalysisToolDefs()...)
	defs = append(defs, tools.TransportToolDefs()...)
	defs = append(defs, tools.ContextToolDefs()...)
	defs = append(defs, tools.ATCToolDefs()...)
	defs = append(defs, tools.ClassIncludeToolDefs()...)
	defs = append(defs, tools.CodeIntelToolDefs()...)
	defs = append(defs, tools.CRUDToolDefs()...)
	defs = append(defs, tools.DevToolDefs()...)
	defs = append(defs, tools.DumpToolDefs()...)
	defs = append(defs, tools.FileToolDefs()...)
	defs = append(defs, tools.GitToolDefs()...)
	defs = append(defs, tools.GrepToolDefs()...)
	defs = append(defs, tools.ReportToolDefs()...)
	defs = append(defs, tools.ServiceBindingToolDefs()...)
	defs = append(defs, tools.SQLTraceToolDefs()...)
	defs = append(defs, tools.TraceToolDefs()...)
	defs = append(defs, tools.WorkflowToolDefs()...)
	defs = append(defs, tools.DebuggerLegacyToolDefs()...)
	return defs
}
