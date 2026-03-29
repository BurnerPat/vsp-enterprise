// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_grep.go contains handlers for grep/search operations on ABAP objects.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

// GrepToolDefs returns tool definitions for grep/search tools.
func GrepToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("GrepObject",
			mcp.WithDescription("Search for regex pattern in a single ABAP object's source code. Returns matches with line numbers and optional context. Use for finding TODO comments, string literals, patterns, or code snippets before editing."),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("pattern", mcp.Required(), mcp.Description("Regular expression pattern (Go regexp syntax). Examples: 'TODO', 'lv_\\\\w+', 'SELECT.*FROM'")),
			mcp.WithBoolean("case_insensitive", mcp.Description("If true, perform case-insensitive matching. Default: false")),
			mcp.WithNumber("context_lines", mcp.Description("Number of lines to show before/after each match (like grep -C). Default: 0")),
		), Handler: HandleGrepObject, ReadOnly: true, Routes: []types.UniversalRoute{
			{Action: "grep", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				// If we have a target like "CLAS ZCL_TEST", we could potentially derive the URL
				// but for now, we just pass through.
				return p
			}},
		}},

		{Tool: mcp.NewTool("GrepPackage",
			mcp.WithDescription("Search for regex pattern across all source objects in an ABAP package. Returns matches grouped by object. Use for package-wide analysis, finding patterns across multiple programs/classes."),
			mcp.WithString("package_name", mcp.Required(), mcp.Description("Package name (e.g., $TMP, ZPACKAGE)")),
			mcp.WithString("pattern", mcp.Required(), mcp.Description("Regular expression pattern (Go regexp syntax). Examples: 'TODO', 'lv_\\\\w+', 'SELECT.*FROM'")),
			mcp.WithBoolean("case_insensitive", mcp.Description("If true, perform case-insensitive matching. Default: false")),
			mcp.WithString("object_types", mcp.Description("Comma-separated object types to search (e.g., 'PROG/P,CLAS/OC'). Empty = search all source objects. Valid: PROG/P, CLAS/OC, INTF/OI, FUGR/F, FUGR/FF, PROG/I")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of matching objects to return. 0 = unlimited. Default: 100")),
		), Handler: HandleGrepPackage, ReadOnly: true, Routes: []types.UniversalRoute{
			{Action: "grep", TargetType: "DEVC", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				p["package_name"] = on
				return p
			}},
		}},
	}
}

// --- Grep/Search Handlers ---

func HandleGrepObject(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.GetArguments()["object_url"].(string)
	if !ok || objectURL == "" {
		return types.ErrorResult("object_url is required"), nil
	}

	pattern, ok := request.GetArguments()["pattern"].(string)
	if !ok || pattern == "" {
		return types.ErrorResult("pattern is required"), nil
	}

	caseInsensitive := false
	if ci, ok := request.GetArguments()["case_insensitive"].(bool); ok {
		caseInsensitive = ci
	}

	contextLines := 0
	if cl, ok := request.GetArguments()["context_lines"].(float64); ok {
		contextLines = int(cl)
	}

	result, err := sys.ADT().GrepObject(ctx, objectURL, pattern, caseInsensitive, contextLines)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GrepObject failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleGrepPackage(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	packageName, ok := request.GetArguments()["package_name"].(string)
	if !ok || packageName == "" {
		return types.ErrorResult("package_name is required"), nil
	}

	pattern, ok := request.GetArguments()["pattern"].(string)
	if !ok || pattern == "" {
		return types.ErrorResult("pattern is required"), nil
	}

	caseInsensitive := false
	if ci, ok := request.GetArguments()["case_insensitive"].(bool); ok {
		caseInsensitive = ci
	}

	// Parse object_types (comma-separated string to slice)
	var objectTypes []string
	if ot, ok := request.GetArguments()["object_types"].(string); ok && ot != "" {
		objectTypes = strings.Split(ot, ",")
		// Trim whitespace from each type
		for i := range objectTypes {
			objectTypes[i] = strings.TrimSpace(objectTypes[i])
		}
	}

	maxResults := 100 // default
	if mr, ok := request.GetArguments()["max_results"].(float64); ok {
		maxResults = int(mr)
	}

	result, err := sys.ADT().GrepPackage(ctx, packageName, pattern, caseInsensitive, objectTypes, maxResults)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GrepPackage failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
