// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_traces.go contains handlers for ABAP profiler traces (ATRA).
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// TraceToolDefs returns tool definitions for ABAP profiler trace tools.
func TraceToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("ListTraces",
			mcp.WithDescription("List ABAP runtime traces (profiler results) from the SAP system."),
			mcp.WithString("user", mcp.Description("Filter by username")),
			mcp.WithString("process_type", mcp.Description("Filter by process type")),
			mcp.WithString("object_type", mcp.Description("Filter by object type")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of results (default: 100)")),
		), Handler: HandleListTraces, ReadOnly: true, Focused: true, Endpoints: []string{"/sap/bc/adt/runtime/traces"}},

		{Tool: mcp.NewTool("GetTrace",
			mcp.WithDescription("Get trace analysis (hitlist, statements, or database accesses) for a specific trace."),
			mcp.WithString("trace_id", mcp.Required(), mcp.Description("Trace ID from ListTraces result")),
			mcp.WithString("tool_type", mcp.Description("Analysis type: 'hitlist' (default), 'statements', 'dbAccesses'")),
		), Handler: HandleGetTrace, ReadOnly: true, Focused: true, Endpoints: []string{"/sap/bc/adt/runtime/traces"}},
	}
}

// --- ABAP Profiler / Traces Handlers ---

func HandleListTraces(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opts := &adt.TraceQueryOptions{
		MaxResults: 100,
	}

	if user, ok := request.GetArguments()["user"].(string); ok && user != "" {
		opts.User = user
	}
	if procType, ok := request.GetArguments()["process_type"].(string); ok && procType != "" {
		opts.ProcessType = procType
	}
	if objType, ok := request.GetArguments()["object_type"].(string); ok && objType != "" {
		opts.ObjectType = objType
	}
	if max, ok := request.GetArguments()["max_results"].(float64); ok && max > 0 {
		opts.MaxResults = int(max)
	}

	traces, err := sys.ADT().ListTraces(ctx, opts)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to list traces: %v", err)), nil
	}

	result, _ := json.MarshalIndent(traces, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleGetTrace(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	traceID, ok := request.GetArguments()["trace_id"].(string)
	if !ok || traceID == "" {
		return types.ErrorResult("trace_id is required"), nil
	}

	toolType := "hitlist"
	if tt, ok := request.GetArguments()["tool_type"].(string); ok && tt != "" {
		toolType = tt
	}

	analysis, err := sys.ADT().GetTrace(ctx, traceID, toolType)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get trace: %v", err)), nil
	}

	result, _ := json.MarshalIndent(analysis, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}
