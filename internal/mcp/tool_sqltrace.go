// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_sqltrace.go contains handlers for SQL trace (ST05).
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// sqlTraceToolDefs returns tool definitions for SQL trace (ST05) tools.
func (s *Server) sqlTraceToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("GetSQLTraceState",
			mcp.WithDescription("Check if SQL trace (ST05) is currently active."),
		), handler: s.handleGetSQLTraceState, readOnly: true, focused: true,
			routes: []universalRoute{{action: "analyze", paramsType: "sql_trace_state"}}},

		{tool: mcp.NewTool("ListSQLTraces",
			mcp.WithDescription("List SQL trace files from ST05."),
			mcp.WithString("user", mcp.Description("Filter by username")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of results (default: 100)")),
		), handler: s.handleListSQLTraces, readOnly: true, focused: true,
			routes: []universalRoute{{action: "analyze", paramsType: "list_sql_traces"}}},
	}
}

// --- SQL Trace (ST05) Handlers ---

func (s *Server) handleGetSQLTraceState(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	state, err := s.adtClient.GetSQLTraceState(ctx)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get SQL trace state: %v", err)), nil
	}

	result, _ := json.MarshalIndent(state, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func (s *Server) handleListSQLTraces(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	user := ""
	maxResults := 100

	if u, ok := request.Params.Arguments["user"].(string); ok {
		user = u
	}
	if max, ok := request.Params.Arguments["max_results"].(float64); ok && max > 0 {
		maxResults = int(max)
	}

	traces, err := s.adtClient.ListSQLTraces(ctx, user, maxResults)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to list SQL traces: %v", err)), nil
	}

	result, _ := json.MarshalIndent(traces, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}
