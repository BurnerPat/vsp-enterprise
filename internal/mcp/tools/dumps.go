// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_dumps.go contains handlers for runtime errors (short dumps / RABAX).
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// DumpToolDefs returns tool definitions for runtime error (short dump) tools.
func DumpToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("ListDumps",
			mcp.WithDescription("List runtime errors (short dumps) from the SAP system. Filter by user, exception type, program, date range."),
			mcp.WithString("user", mcp.Description("Filter by username")),
			mcp.WithString("exception_type", mcp.Description("Filter by exception type (e.g., CX_SY_ZERODIVIDE)")),
			mcp.WithString("program", mcp.Description("Filter by program name")),
			mcp.WithString("package", mcp.Description("Filter by package")),
			mcp.WithString("date_from", mcp.Description("Start date (YYYYMMDD format)")),
			mcp.WithString("date_to", mcp.Description("End date (YYYYMMDD format)")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of results (default: 100)")),
		), Handler: HandleListDumps, ReadOnly: true, Focused: true, Endpoints: []string{"/sap/bc/adt/runtime/dumps"}},

		{Tool: mcp.NewTool("GetDump",
			mcp.WithDescription("Get full details of a specific runtime error (short dump) including stack trace."),
			mcp.WithString("dump_id", mcp.Required(), mcp.Description("Dump ID from ListDumps result")),
		), Handler: HandleGetDump, ReadOnly: true, Focused: true, Endpoints: []string{"/sap/bc/adt/runtime/dumps"}},
	}
}

// --- Runtime Errors / Short Dumps Handlers ---

func HandleListDumps(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opts := &adt.DumpQueryOptions{
		MaxResults: 100,
	}

	if user, ok := request.GetArguments()["user"].(string); ok && user != "" {
		opts.User = user
	}
	if excType, ok := request.GetArguments()["exception_type"].(string); ok && excType != "" {
		opts.ExceptionType = excType
	}
	if prog, ok := request.GetArguments()["program"].(string); ok && prog != "" {
		opts.Program = prog
	}
	if pkg, ok := request.GetArguments()["package"].(string); ok && pkg != "" {
		opts.Package = pkg
	}
	if dateFrom, ok := request.GetArguments()["date_from"].(string); ok && dateFrom != "" {
		opts.DateFrom = dateFrom
	}
	if dateTo, ok := request.GetArguments()["date_to"].(string); ok && dateTo != "" {
		opts.DateTo = dateTo
	}
	if max, ok := request.GetArguments()["max_results"].(float64); ok && max > 0 {
		opts.MaxResults = int(max)
	}

	dumps, err := sys.ADT().GetDumps(ctx, opts)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get dumps: %v", err)), nil
	}

	result, _ := json.MarshalIndent(dumps, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleGetDump(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dumpID, ok := request.GetArguments()["dump_id"].(string)
	if !ok || dumpID == "" {
		return types.ErrorResult("dump_id is required"), nil
	}

	dump, err := sys.ADT().GetDump(ctx, dumpID)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get dump: %v", err)), nil
	}

	result, _ := json.MarshalIndent(dump, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}
