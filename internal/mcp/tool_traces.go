// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_traces.go contains handlers for ABAP profiler traces (ATRA).
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// traceToolDefs returns tool definitions for ABAP profiler trace tools.
func (s *Server) traceToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("ListTraces",
			mcp.WithDescription("List ABAP runtime traces (profiler results) from the SAP system."),
			mcp.WithString("user", mcp.Description("Filter by username")),
			mcp.WithString("process_type", mcp.Description("Filter by process type")),
			mcp.WithString("object_type", mcp.Description("Filter by object type")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of results (default: 100)")),
		), handler: s.handleListTraces, readOnly: true, focused: true,
			routes: []universalRoute{{action: "analyze", paramsType: "list_traces"}}},

		{tool: mcp.NewTool("GetTrace",
			mcp.WithDescription("Get trace analysis (hitlist, statements, or database accesses) for a specific trace."),
			mcp.WithString("trace_id", mcp.Required(), mcp.Description("Trace ID from ListTraces result")),
			mcp.WithString("tool_type", mcp.Description("Analysis type: 'hitlist' (default), 'statements', 'dbAccesses'")),
		), handler: s.handleGetTrace, readOnly: true, focused: true,
			routes: []universalRoute{{action: "analyze", paramsType: "get_trace"}}},
	}
}

// --- ABAP Profiler / Traces Handlers ---

func (s *Server) handleListTraces(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opts := &adt.TraceQueryOptions{
		MaxResults: 100,
	}

	if user, ok := request.Params.Arguments["user"].(string); ok && user != "" {
		opts.User = user
	}
	if procType, ok := request.Params.Arguments["process_type"].(string); ok && procType != "" {
		opts.ProcessType = procType
	}
	if objType, ok := request.Params.Arguments["object_type"].(string); ok && objType != "" {
		opts.ObjectType = objType
	}
	if max, ok := request.Params.Arguments["max_results"].(float64); ok && max > 0 {
		opts.MaxResults = int(max)
	}

	traces, err := s.adtClient.ListTraces(ctx, opts)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to list traces: %v", err)), nil
	}

	result, _ := json.MarshalIndent(traces, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func (s *Server) handleGetTrace(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	traceID, ok := request.Params.Arguments["trace_id"].(string)
	if !ok || traceID == "" {
		return newToolResultError("trace_id is required"), nil
	}

	toolType := "hitlist"
	if tt, ok := request.Params.Arguments["tool_type"].(string); ok && tt != "" {
		toolType = tt
	}

	analysis, err := s.adtClient.GetTrace(ctx, traceID, toolType)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get trace: %v", err)), nil
	}

	result, _ := json.MarshalIndent(analysis, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}
