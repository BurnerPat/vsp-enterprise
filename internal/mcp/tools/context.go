// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_context.go contains the GetContext handler for dependency context compression.
package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/ctxcomp"
)

// ContextToolDefs returns tool definitions for context compression tools.
func ContextToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{
			Tool: mcp.NewTool("GetContext",
				mcp.WithDescription("Fetch source and all direct dependencies of an ABAP object, providing compressed context for editing. Recursively resolves and fetches sources of used classes, interfaces, types, etc."),
				mcp.WithString("object_type", mcp.Required(), mcp.Description("ADT object type (e.g. CLAS/OC, PROG/P)")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Object name (e.g. ZCL_MY_CLASS)")),
				mcp.WithString("source", mcp.Description("Optional: provide source code directly instead of fetching it from SAP")),
				mcp.WithNumber("max_deps", mcp.Description("Maximum number of dependencies to resolve. Default: 20")),
			),
			Handler: HandleGetContext,
		},
	}
}

// adtSourceAdapter adapts adt.Client to the ctxcomp.ADTSourceFetcher interface.
type adtSourceAdapter struct {
	sys types.System
}

func (a *adtSourceAdapter) GetSource(ctx context.Context, objectType, name string, opts interface{}) (string, error) {
	return a.sys.ADT().GetSource(ctx, objectType, name, nil)
}

func HandleGetContext(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.GetArguments()["object_type"].(string)
	if !ok || objectType == "" {
		return types.ErrorResult("object_type is required"), nil
	}

	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return types.ErrorResult("name is required"), nil
	}

	source, _ := request.GetArguments()["source"].(string)
	maxDeps := 20
	if md, ok := request.GetArguments()["max_deps"].(float64); ok && md > 0 {
		maxDeps = int(md)
	}

	// Fetch source from SAP if not provided
	if source == "" {
		var err error
		source, err = sys.ADT().GetSource(ctx, objectType, name, nil)
		if err != nil {
			return types.ErrorResult(fmt.Sprintf("GetContext: failed to fetch source for %s %s: %v", objectType, name, err)), nil
		}
	}

	provider := ctxcomp.NewMultiSourceProvider("", &adtSourceAdapter{sys: sys})
	compressor := ctxcomp.NewCompressor(provider, maxDeps)
	result, err := compressor.Compress(ctx, source, name, objectType)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetContext failed: %v", err)), nil
	}

	if result.Prologue == "" {
		return mcp.NewToolResultText(fmt.Sprintf("No resolvable dependencies found for %s %s", objectType, name)), nil
	}

	// Append stats
	output := fmt.Sprintf("%s\n* Stats: %d deps found, %d resolved, %d failed, %d lines",
		result.Prologue, result.Stats.DepsFound, result.Stats.DepsResolved, result.Stats.DepsFailed, result.Stats.TotalLines)

	return mcp.NewToolResultText(output), nil
}
