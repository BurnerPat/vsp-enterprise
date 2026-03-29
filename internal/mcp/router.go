package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/oisee/vibing-steampunk/internal/mcp/tools"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

// Router handles tool call routing and system management.
type Router struct {
	mcpServer *server.MCPServer
	systems   map[string]types.System
	systemIDs []string
}

func NewRouter(mcpServer *server.MCPServer) *Router {
	return &Router{
		mcpServer: mcpServer,
		systems:   make(map[string]types.System),
	}
}

func (r *Router) AddSystem(id string, sys types.System) {
	r.systems[strings.ToLower(id)] = sys
	r.systemIDs = append(r.systemIDs, id)
}

// RegisterTools registers all available tools from all packages.
func (r *Router) RegisterTools(mode string, disabledGroups string, toolsConfig map[string]bool) {
	allDefs := r.allToolDefs()

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

	// 2. Permission check (Future expansion)
	// if !r.checkPermission(sys, td) { ... }

	// 3. Invoke handler
	return td.Handler(ctx, sys, request)
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
