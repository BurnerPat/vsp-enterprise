package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// AnalysisToolDefs returns tool definitions for code analysis tools.
func AnalysisToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("GetCallGraph",
			mcp.WithDescription("Generate static call graph for an ABAP object"),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the object")),
			mcp.WithNumber("max_depth", mcp.Description("Maximum depth of call graph (default: 3)")),
		), Handler: HandleGetCallGraph, ReadOnly: true, Focused: true, Endpoints: []string{"/sap/bc/adt/cai/callgraph"},
			Routes: []types.UniversalRoute{{Action: "analyze", TargetType: "CALL_GRAPH"}}},

		{Tool: mcp.NewTool("GetObjectStructure",
			mcp.WithDescription("Get structural overview of an ABAP object (classes, interfaces, FMs)"),
			mcp.WithString("object_name", mcp.Required(), mcp.Description("Name of the object")),
		), Handler: HandleGetObjectStructure, ReadOnly: true, Focused: true, Endpoints: []string{"/sap/bc/adt/oo/classes"},
			Routes: []types.UniversalRoute{{Action: "analyze", TargetType: "STRUCTURE"}}},

		{Tool: mcp.NewTool("GetCallersOf",
			mcp.WithDescription("Find all callers of a given ABAP object"),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the object")),
			mcp.WithNumber("max_depth", mcp.Description("Maximum depth (default: 3)")),
		), Handler: HandleGetCallersOf, ReadOnly: true, Endpoints: []string{"/sap/bc/adt/cai/callgraph"}},

		{Tool: mcp.NewTool("GetCalleesOf",
			mcp.WithDescription("Find all objects called by a given ABAP object"),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the object")),
			mcp.WithNumber("max_depth", mcp.Description("Maximum depth (default: 3)")),
		), Handler: HandleGetCalleesOf, ReadOnly: true, Endpoints: []string{"/sap/bc/adt/cai/callgraph"}},

		{Tool: mcp.NewTool("AnalyzeCallGraph",
			mcp.WithDescription("Compute metrics and statistics for a call graph"),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the object")),
		), Handler: HandleAnalyzeCallGraph, ReadOnly: true, Endpoints: []string{"/sap/bc/adt/cai/callgraph"}},

		{Tool: mcp.NewTool("CompareCallGraphs",
			mcp.WithDescription("Compare static vs dynamic (trace-based) call graphs"),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the object")),
			mcp.WithString("trace_id", mcp.Required(), mcp.Description("Trace ID for dynamic call graph")),
		), Handler: HandleCompareCallGraphs, ReadOnly: true, Endpoints: []string{"/sap/bc/adt/cai/callgraph", "/sap/bc/adt/runtime/traces"}},

		{Tool: mcp.NewTool("TraceExecution",
			mcp.WithDescription("Execute ABAP code with coverage/performance tracing."),
			mcp.WithString("object_type", mcp.Required(), mcp.Description("PROG, CLAS, FUNC")),
			mcp.WithString("object_name", mcp.Required(), mcp.Description("Name of the object")),
			mcp.WithString("method", mcp.Description("Method name for CLAS")),
		), Handler: HandleTraceExecution, Focused: true},
	}
}

// --- Analysis Handlers ---

func HandleGetCallGraph(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uri, _ := request.GetArguments()["object_uri"].(string)
	depth := 3
	if d, ok := request.GetArguments()["max_depth"].(float64); ok {
		depth = int(d)
	}

	opts := &adt.CallGraphOptions{MaxDepth: depth}
	graph, err := sys.ADT().GetCallGraph(ctx, uri, opts)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get call graph: %v", err)), nil
	}

	result, _ := json.MarshalIndent(graph, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleGetObjectStructure(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, _ := request.GetArguments()["object_name"].(string)
	structure, err := sys.ADT().GetObjectStructureCAI(ctx, name, 100)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get structure: %v", err)), nil
	}

	result, _ := json.MarshalIndent(structure, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleGetCallersOf(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uri, _ := request.GetArguments()["object_uri"].(string)
	depth := 3
	if d, ok := request.GetArguments()["max_depth"].(float64); ok {
		depth = int(d)
	}

	graph, err := sys.ADT().GetCallersOf(ctx, uri, depth)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get callers: %v", err)), nil
	}

	result, _ := json.MarshalIndent(graph, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleGetCalleesOf(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uri, _ := request.GetArguments()["object_uri"].(string)
	depth := 3
	if d, ok := request.GetArguments()["max_depth"].(float64); ok {
		depth = int(d)
	}

	graph, err := sys.ADT().GetCalleesOf(ctx, uri, depth)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get callees: %v", err)), nil
	}

	result, _ := json.MarshalIndent(graph, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleAnalyzeCallGraph(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uri, _ := request.GetArguments()["object_uri"].(string)
	graph, err := sys.ADT().GetCallGraph(ctx, uri, nil)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get call graph: %v", err)), nil
	}

	stats := adt.AnalyzeCallGraph(graph)
	result, _ := json.MarshalIndent(stats, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleCompareCallGraphs(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("CompareCallGraphs (placeholder)"), nil
}

func HandleTraceExecution(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("TraceExecution (placeholder)"), nil
}
