// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_grep.go contains handlers for grep/search operations on ABAP objects.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// grepToolDefs returns tool definitions for grep/search tools.
func (s *Server) grepToolDefs() []toolDef {
	defs := []toolDef{
		{tool: mcp.NewTool("GrepObject",
			mcp.WithDescription("Search for regex pattern in a single ABAP object's source code. Returns matches with line numbers and optional context. Use for finding TODO comments, string literals, patterns, or code snippets before editing."),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("pattern", mcp.Required(), mcp.Description("Regular expression pattern (Go regexp syntax). Examples: 'TODO', 'lv_\\\\w+', 'SELECT.*FROM'")),
			mcp.WithBoolean("case_insensitive", mcp.Description("If true, perform case-insensitive matching. Default: false")),
			mcp.WithNumber("context_lines", mcp.Description("Number of lines to show before/after each match (like grep -C). Default: 0")),
		), handler: s.handleGrepObject, readOnly: true},

		{tool: mcp.NewTool("GrepPackage",
			mcp.WithDescription("Search for regex pattern across all source objects in an ABAP package. Returns matches grouped by object. Use for package-wide analysis, finding patterns across multiple programs/classes."),
			mcp.WithString("package_name", mcp.Required(), mcp.Description("Package name (e.g., $TMP, ZPACKAGE)")),
			mcp.WithString("pattern", mcp.Required(), mcp.Description("Regular expression pattern (Go regexp syntax). Examples: 'TODO', 'lv_\\\\w+', 'SELECT.*FROM'")),
			mcp.WithBoolean("case_insensitive", mcp.Description("If true, perform case-insensitive matching. Default: false")),
			mcp.WithString("object_types", mcp.Description("Comma-separated object types to search (e.g., 'PROG/P,CLAS/OC'). Empty = search all source objects. Valid: PROG/P, CLAS/OC, INTF/OI, FUGR/F, FUGR/FF, PROG/I")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of matching objects to return. 0 = unlimited. Default: 100")),
		), handler: s.handleGrepPackage, readOnly: true},
	}
	// Append GrepObjects/GrepPackages from tool_source.go
	defs = append(defs, s.grepSourceToolDefs()...)
	return defs
}

// routeGrepAction routes "grep" action.
func (s *Server) routeGrepAction(ctx context.Context, action, objectType, objectName string, params map[string]any) (*mcp.CallToolResult, bool, error) {
	if action != "grep" {
		return nil, false, nil
	}

	// GrepObjects (multiple objects)
	if _, ok := params["object_urls"]; ok {
		return s.callHandler(ctx, s.handleGrepObjects, params)
	}

	// GrepPackages (multiple packages)
	if _, ok := params["packages"]; ok {
		return s.callHandler(ctx, s.handleGrepPackages, params)
	}

	// GrepPackage (single package)
	if pkgName := getStringParam(params, "package_name"); pkgName != "" {
		return s.callHandler(ctx, s.handleGrepPackage, params)
	}

	// GrepObject (single object)
	if objectURL := getStringParam(params, "object_url"); objectURL != "" {
		return s.callHandler(ctx, s.handleGrepObject, params)
	}

	return nil, false, nil
}

// --- Grep/Search Handlers ---

func (s *Server) handleGrepObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	pattern, ok := request.Params.Arguments["pattern"].(string)
	if !ok || pattern == "" {
		return newToolResultError("pattern is required"), nil
	}

	caseInsensitive := false
	if ci, ok := request.Params.Arguments["case_insensitive"].(bool); ok {
		caseInsensitive = ci
	}

	contextLines := 0
	if cl, ok := request.Params.Arguments["context_lines"].(float64); ok {
		contextLines = int(cl)
	}

	result, err := s.adtClient.GrepObject(ctx, objectURL, pattern, caseInsensitive, contextLines)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GrepObject failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleGrepPackage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	packageName, ok := request.Params.Arguments["package_name"].(string)
	if !ok || packageName == "" {
		return newToolResultError("package_name is required"), nil
	}

	pattern, ok := request.Params.Arguments["pattern"].(string)
	if !ok || pattern == "" {
		return newToolResultError("pattern is required"), nil
	}

	caseInsensitive := false
	if ci, ok := request.Params.Arguments["case_insensitive"].(bool); ok {
		caseInsensitive = ci
	}

	// Parse object_types (comma-separated string to slice)
	var objectTypes []string
	if ot, ok := request.Params.Arguments["object_types"].(string); ok && ot != "" {
		objectTypes = strings.Split(ot, ",")
		// Trim whitespace from each type
		for i := range objectTypes {
			objectTypes[i] = strings.TrimSpace(objectTypes[i])
		}
	}

	maxResults := 100 // default
	if mr, ok := request.Params.Arguments["max_results"].(float64); ok {
		maxResults = int(mr)
	}

	result, err := s.adtClient.GrepPackage(ctx, packageName, pattern, caseInsensitive, objectTypes, maxResults)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GrepPackage failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
