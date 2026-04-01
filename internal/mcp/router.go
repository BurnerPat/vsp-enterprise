package mcp

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/oisee/vibing-steampunk/internal/config"
	"github.com/oisee/vibing-steampunk/internal/log"
	"github.com/oisee/vibing-steampunk/internal/mcp/tools"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

type SystemInRouter struct {
	System       types.System
	ID           string
	EnabledTools []*types.ToolDef
}

// Router handles tool call routing and system management.
type Router struct {
	mcpServer *server.MCPServer
	systems   map[string]*SystemInRouter
	systemIDs []string
	allTools  []types.ToolDef
}

func NewRouter(mcpServer *server.MCPServer) *Router {
	return &Router{
		mcpServer: mcpServer,
		systems:   make(map[string]*SystemInRouter),
	}
}

func (r *Router) AddSystem(id string, sys types.System) {
	r.systems[strings.ToLower(id)] = &SystemInRouter{
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

func (r *Router) HandleToolCall(ctx context.Context, td *types.ToolDef, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	if slices.IndexFunc(sys.EnabledTools, func(t *types.ToolDef) bool { return t.Tool.Name == td.Tool.Name }) == -1 {
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

func (r *Router) resolvePermissionConfig(cfg *config.GlobalConfig) ([]*types.ToolDef, error) {
	r.allTools = allToolDefs()

	allToolDefs := make([]*types.ToolDef, len(r.allTools))

	for i, td := range r.allTools {
		allToolDefs[i] = &td
	}

	// Identify which tools are enabled globally (in the config root permissions)
	enabledTools, err := resolvePermissions(&cfg.Permissions, allToolDefs)

	// Calculate effective enabled tools for all SystemClasses
	var toolsPerSystemClass = make(map[string][]*types.ToolDef)

	if cfg.SystemClasses != nil {
		var allSystemClassTools = make([]string, 0)

		for name, systemClass := range cfg.SystemClasses {
			systemClassTools, err := resolvePermissions(&systemClass.Permissions, enabledTools)

			if err != nil {
				return nil, err
			}

			toolsPerSystemClass[name] = systemClassTools

			for _, tool := range systemClassTools {
				allSystemClassTools = append(allSystemClassTools, tool.Tool.Name)
			}
		}
	}

	// Calculate effective enabled tools for each system
	var allSystemTools = make([]string, 0)

	for name, system := range cfg.Systems {
		var effectiveTools []*types.ToolDef

		if system.SystemClass != "" {
			effectiveTools = append([]*types.ToolDef(nil), toolsPerSystemClass[system.SystemClass]...)

			if effectiveTools == nil {
				return nil, fmt.Errorf("system class %q not found", system.SystemClass)
			}
		} else {
			effectiveTools = append([]*types.ToolDef(nil), enabledTools...)
		}

		systemTools, err := resolvePermissions(&system.Permissions, effectiveTools)

		if err != nil {
			return nil, err
		}

		sys := r.systems[strings.ToLower(name)]
		sys.EnabledTools = systemTools

		for _, tool := range systemTools {
			allSystemTools = append(allSystemTools, tool.Tool.Name)
		}
	}

	enabledTools = slices.DeleteFunc(append([]*types.ToolDef(nil), enabledTools...), func(tool *types.ToolDef) bool {
		return !slices.Contains(allSystemTools, tool.Tool.Name)
	})

	r.logEffectivePermissions(allToolDefs, enabledTools)

	return enabledTools, err
}

func resolvePermissions(permissions *config.PermissionConfig, availableTools []*types.ToolDef) ([]*types.ToolDef, error) {
	if !permissions.DenyToolsByDefault && len(permissions.Tools) == 0 {
		return availableTools, nil
	}

	toolsCopy := append([]*types.ToolDef(nil), availableTools...)

	return slices.DeleteFunc(toolsCopy, func(tool *types.ToolDef) bool {
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

	for í, part := range parts {
		idx := strings.Index(str, part)

		if idx == -1 || (í == 0 && idx != 0 && !strings.HasPrefix(pattern, "*")) {
			return false
		}

		str = str[idx+len(part):]
	}

	return strings.HasSuffix(pattern, "*") || str == ""
}

func (r *Router) logEffectivePermissions(allTools []*types.ToolDef, enabledTools []*types.ToolDef) {
	if !config.GetInstance().Verbose {
		return
	}

	if len(config.GetInstance().Permissions.Tools) > 0 {
		log.LogInfo("Globally enabled tools (%d/%d)", len(enabledTools), len(allTools))

		for _, tool := range enabledTools {
			log.LogInfo("  - %s", tool.Tool.Name)
		}
	} else {
		log.LogInfo("All tools globally enabled (%d)", len(enabledTools))
	}

	log.LogInfo("Effective permissions per system:")

	for name, sys := range r.systems {
		if len(sys.EnabledTools) == len(enabledTools) {
			log.LogInfo("  - System %q: All tools enabled", name)
			continue
		}

		log.LogInfo("  - System %q: %d/%d tools enabled", name, len(sys.EnabledTools), len(enabledTools))

		for _, tool := range sys.EnabledTools {
			log.LogInfo("      - %s", tool.Tool.Name)
		}
	}
}

//goland:noinspection DuplicatedCode
func allToolDefs() []types.ToolDef {
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
