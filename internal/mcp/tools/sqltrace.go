// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_sqltrace.go contains handlers for SQL trace (ST05).
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

// SQLTraceToolDefs returns tool definitions for SQL trace (ST05) tools.
func SQLTraceToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("GetSQLTraceState",
			mcp.WithDescription("Check if SQL trace (ST05) is currently active."),
		), Handler: HandleGetSQLTraceState, ReadOnly: true, Focused: true, Endpoints: []string{"/sap/bc/adt/st05/trace"}},

		{Tool: mcp.NewTool("ListSQLTraces",
			mcp.WithDescription("List SQL trace files from ST05."),
			mcp.WithString("user", mcp.Description("Filter by username")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of results (default: 100)")),
		), Handler: HandleListSQLTraces, ReadOnly: true, Focused: true, Endpoints: []string{"/sap/bc/adt/st05/trace"}},
	}
}

// --- SQL Trace (ST05) Handlers ---

func HandleGetSQLTraceState(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	state, err := sys.ADT().GetSQLTraceState(ctx)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get SQL trace state: %v", err)), nil
	}

	result, _ := json.MarshalIndent(state, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleListSQLTraces(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	user := ""
	maxResults := 100

	if u, ok := request.GetArguments()["user"].(string); ok {
		user = u
	}
	if max, ok := request.GetArguments()["max_results"].(float64); ok && max > 0 {
		maxResults = int(max)
	}

	traces, err := sys.ADT().ListSQLTraces(ctx, user, maxResults)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to list SQL traces: %v", err)), nil
	}

	result, _ := json.MarshalIndent(traces, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}
