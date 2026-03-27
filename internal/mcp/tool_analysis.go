// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_analysis.go contains handlers for code analysis infrastructure (call graphs, tracing).
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// analysisToolDefs returns tool definitions for code analysis tools.
func (s *Server) analysisToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("GetCallGraph",
			mcp.WithDescription("Get call hierarchy for methods/functions. Shows callers or callees of an ABAP object."),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST/source/main#start=10,1)")),
			mcp.WithString("direction", mcp.Description("Direction: 'callers' (who calls this) or 'callees' (what this calls). Default: callers")),
			mcp.WithNumber("max_depth", mcp.Description("Maximum depth of call hierarchy (default: 3)")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of results (default: 100)")),
		), handler: s.handleGetCallGraph, readOnly: true, focused: true,
			routes: []universalRoute{{action: "analyze", paramsType: "call_graph"}}},

		{tool: mcp.NewTool("GetObjectStructure",
			mcp.WithDescription("Get object explorer tree structure. Returns hierarchical view of object components."),
			mcp.WithString("object_name", mcp.Required(), mcp.Description("Object name (e.g., ZCL_TEST, ZPROGRAM)")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of results (default: 100)")),
		), handler: s.handleGetObjectStructure, readOnly: true, focused: true,
			routes: []universalRoute{{action: "analyze", paramsType: "object_structure"}}},

		{tool: mcp.NewTool("GetCallersOf",
			mcp.WithDescription("Find all callers of an ABAP object (up traversal). Shows who calls this method/function. Simplified wrapper around GetCallGraph."),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST/source/main#start=10,1)")),
			mcp.WithNumber("max_depth", mcp.Description("Maximum depth of caller hierarchy (default: 5)")),
		), handler: s.handleGetCallersOf, readOnly: true, focused: true,
			routes: []universalRoute{{action: "analyze", paramsType: "callers"}}},

		{tool: mcp.NewTool("GetCalleesOf",
			mcp.WithDescription("Find all callees of an ABAP object (down traversal). Shows what this method/function calls. Simplified wrapper around GetCallGraph."),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST/source/main#start=10,1)")),
			mcp.WithNumber("max_depth", mcp.Description("Maximum depth of callee hierarchy (default: 5)")),
		), handler: s.handleGetCalleesOf, readOnly: true, focused: true,
			routes: []universalRoute{{action: "analyze", paramsType: "callees"}}},

		{tool: mcp.NewTool("AnalyzeCallGraph",
			mcp.WithDescription("Analyze call graph for an object. Returns statistics: total nodes, edges, max depth, nodes by type. Use for understanding code complexity and dependencies."),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the object to analyze")),
			mcp.WithString("direction", mcp.Description("Direction: 'callers' or 'callees' (default: callees)")),
			mcp.WithNumber("max_depth", mcp.Description("Maximum depth to analyze (default: 5)")),
		), handler: s.handleAnalyzeCallGraph, readOnly: true, focused: true,
			routes: []universalRoute{{action: "analyze", paramsType: "analyze_call_graph"}}},

		{tool: mcp.NewTool("CompareCallGraphs",
			mcp.WithDescription("Compare static call graph with actual execution trace. Identifies: common paths, untested paths (static only), and dynamic calls (actual only). Use for test coverage analysis and RCA."),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the root object")),
			mcp.WithString("trace_data", mcp.Required(), mcp.Description("JSON array of trace edges from execution (format: [{caller_name, callee_name}, ...])")),
		), handler: s.handleCompareCallGraphs, readOnly: true, focused: true,
			routes: []universalRoute{{action: "analyze", paramsType: "compare_call_graphs"}}},

		{tool: mcp.NewTool("TraceExecution",
			mcp.WithDescription("COMPOSITE RCA TOOL: Performs traced execution analysis. 1) Builds static call graph from object, 2) Optionally runs unit tests, 3) Collects trace data, 4) Extracts actual call edges, 5) Compares static vs actual for root cause analysis."),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the starting object for static call graph")),
			mcp.WithNumber("max_depth", mcp.Description("Maximum depth for call graph traversal (default: 5)")),
			mcp.WithBoolean("run_tests", mcp.Description("Run unit tests before collecting trace (default: false)")),
			mcp.WithString("test_object_uri", mcp.Description("Object URI for tests to run (defaults to object_uri)")),
			mcp.WithString("trace_user", mcp.Description("Filter traces by user (defaults to current user)")),
		), handler: s.handleTraceExecution, readOnly: true, focused: true,
			routes: []universalRoute{{action: "analyze", paramsType: "trace_execution"}}},
	}
}

// --- Code Analysis Infrastructure Handlers ---

func (s *Server) handleGetCallGraph(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURI, ok := request.Params.Arguments["object_uri"].(string)
	if !ok || objectURI == "" {
		return newToolResultError("object_uri is required"), nil
	}

	opts := &adt.CallGraphOptions{
		Direction:  "callers",
		MaxDepth:   3,
		MaxResults: 100,
	}

	if dir, ok := request.Params.Arguments["direction"].(string); ok && dir != "" {
		opts.Direction = dir
	}
	if depth, ok := request.Params.Arguments["max_depth"].(float64); ok && depth > 0 {
		opts.MaxDepth = int(depth)
	}
	if max, ok := request.Params.Arguments["max_results"].(float64); ok && max > 0 {
		opts.MaxResults = int(max)
	}

	graph, err := s.adtClient.GetCallGraph(ctx, objectURI, opts)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get call graph: %v", err)), nil
	}

	result, _ := json.MarshalIndent(graph, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func (s *Server) handleGetObjectStructure(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectName, ok := request.Params.Arguments["object_name"].(string)
	if !ok || objectName == "" {
		return newToolResultError("object_name is required"), nil
	}

	maxResults := 100
	if max, ok := request.Params.Arguments["max_results"].(float64); ok && max > 0 {
		maxResults = int(max)
	}

	structure, err := s.adtClient.GetObjectStructureCAI(ctx, objectName, maxResults)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get object structure: %v", err)), nil
	}

	result, _ := json.MarshalIndent(structure, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func (s *Server) handleGetCallersOf(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURI, ok := request.Params.Arguments["object_uri"].(string)
	if !ok || objectURI == "" {
		return newToolResultError("object_uri is required"), nil
	}

	maxDepth := 5
	if depth, ok := request.Params.Arguments["max_depth"].(float64); ok && depth > 0 {
		maxDepth = int(depth)
	}

	graph, err := s.adtClient.GetCallersOf(ctx, objectURI, maxDepth)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get callers: %v", err)), nil
	}

	// Flatten to edges for easier consumption
	edges := adt.FlattenCallGraph(graph)
	stats := adt.AnalyzeCallGraph(graph)

	output := map[string]interface{}{
		"root":  graph,
		"edges": edges,
		"stats": stats,
	}

	result, _ := json.MarshalIndent(output, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func (s *Server) handleGetCalleesOf(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURI, ok := request.Params.Arguments["object_uri"].(string)
	if !ok || objectURI == "" {
		return newToolResultError("object_uri is required"), nil
	}

	maxDepth := 5
	if depth, ok := request.Params.Arguments["max_depth"].(float64); ok && depth > 0 {
		maxDepth = int(depth)
	}

	graph, err := s.adtClient.GetCalleesOf(ctx, objectURI, maxDepth)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get callees: %v", err)), nil
	}

	// Flatten to edges for easier consumption
	edges := adt.FlattenCallGraph(graph)
	stats := adt.AnalyzeCallGraph(graph)

	output := map[string]interface{}{
		"root":  graph,
		"edges": edges,
		"stats": stats,
	}

	result, _ := json.MarshalIndent(output, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func (s *Server) handleAnalyzeCallGraph(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURI, ok := request.Params.Arguments["object_uri"].(string)
	if !ok || objectURI == "" {
		return newToolResultError("object_uri is required"), nil
	}

	direction := "callees"
	if dir, ok := request.Params.Arguments["direction"].(string); ok && dir != "" {
		direction = dir
	}

	maxDepth := 5
	if depth, ok := request.Params.Arguments["max_depth"].(float64); ok && depth > 0 {
		maxDepth = int(depth)
	}

	graph, err := s.adtClient.GetCallGraph(ctx, objectURI, &adt.CallGraphOptions{
		Direction:  direction,
		MaxDepth:   maxDepth,
		MaxResults: 1000,
	})
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get call graph: %v", err)), nil
	}

	stats := adt.AnalyzeCallGraph(graph)
	edges := adt.FlattenCallGraph(graph)

	output := map[string]interface{}{
		"object_uri": objectURI,
		"direction":  direction,
		"stats":      stats,
		"edge_count": len(edges),
		"edges":      edges,
	}

	result, _ := json.MarshalIndent(output, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func (s *Server) handleCompareCallGraphs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURI, ok := request.Params.Arguments["object_uri"].(string)
	if !ok || objectURI == "" {
		return newToolResultError("object_uri is required"), nil
	}

	traceDataStr, ok := request.Params.Arguments["trace_data"].(string)
	if !ok || traceDataStr == "" {
		return newToolResultError("trace_data is required (JSON array of edges)"), nil
	}

	// Parse trace data
	var actualEdges []adt.CallGraphEdge
	if err := json.Unmarshal([]byte(traceDataStr), &actualEdges); err != nil {
		return newToolResultError(fmt.Sprintf("Failed to parse trace_data: %v", err)), nil
	}

	// Get static call graph
	graph, err := s.adtClient.GetCallGraph(ctx, objectURI, &adt.CallGraphOptions{
		Direction:  "callees",
		MaxDepth:   10,
		MaxResults: 1000,
	})
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get static call graph: %v", err)), nil
	}

	staticEdges := adt.FlattenCallGraph(graph)

	// Compare
	comparison := adt.CompareCallGraphs(staticEdges, actualEdges)

	output := map[string]interface{}{
		"object_uri":     objectURI,
		"static_edges":   len(staticEdges),
		"actual_edges":   len(actualEdges),
		"common_edges":   len(comparison.CommonEdges),
		"untested_paths": len(comparison.StaticOnly),
		"dynamic_calls":  len(comparison.ActualOnly),
		"coverage_ratio": comparison.CoverageRatio,
		"common":         comparison.CommonEdges,
		"static_only":    comparison.StaticOnly,
		"actual_only":    comparison.ActualOnly,
	}

	result, _ := json.MarshalIndent(output, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func (s *Server) handleTraceExecution(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURI, ok := request.Params.Arguments["object_uri"].(string)
	if !ok || objectURI == "" {
		return newToolResultError("object_uri is required"), nil
	}

	opts := &adt.TraceExecutionOptions{
		ObjectURI: objectURI,
		MaxDepth:  5,
	}

	if maxDepth, ok := request.Params.Arguments["max_depth"].(float64); ok {
		opts.MaxDepth = int(maxDepth)
	}

	if runTests, ok := request.Params.Arguments["run_tests"].(bool); ok {
		opts.RunTests = runTests
	}

	if testURI, ok := request.Params.Arguments["test_object_uri"].(string); ok && testURI != "" {
		opts.TestObjectURI = testURI
	} else if opts.RunTests {
		opts.TestObjectURI = objectURI // Default to same object
	}

	if traceUser, ok := request.Params.Arguments["trace_user"].(string); ok && traceUser != "" {
		opts.TraceUser = traceUser
	}

	result, err := s.adtClient.TraceExecution(ctx, opts)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Trace execution failed: %v", err)), nil
	}

	// Build comprehensive output
	output := map[string]interface{}{
		"object_uri": objectURI,
	}

	if result.StaticStats != nil {
		output["static_stats"] = result.StaticStats
	}

	if result.Trace != nil {
		output["trace"] = map[string]interface{}{
			"id":          result.Trace.TraceID,
			"total_time":  result.Trace.TotalTime,
			"total_calls": result.Trace.TotalCalls,
			"entries":     len(result.Trace.Entries),
		}
	}

	if len(result.ActualEdges) > 0 {
		output["actual_edges"] = result.ActualEdges
	}

	if result.Comparison != nil {
		output["comparison"] = map[string]interface{}{
			"common_edges":   len(result.Comparison.CommonEdges),
			"untested_paths": len(result.Comparison.StaticOnly),
			"dynamic_calls":  len(result.Comparison.ActualOnly),
			"coverage_ratio": result.Comparison.CoverageRatio,
			"static_only":    result.Comparison.StaticOnly,
			"actual_only":    result.Comparison.ActualOnly,
		}
	}

	if len(result.ExecutedTests) > 0 {
		output["executed_tests"] = result.ExecutedTests
	}

	output["execution_time_us"] = result.ExecutionTime

	jsonResult, _ := json.MarshalIndent(output, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}
