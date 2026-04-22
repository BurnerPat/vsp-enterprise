// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_classinclude.go contains handlers for class include operations (testclasses, locals_def, etc.).
package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// ClassIncludeToolDefs returns tool definitions for class include operations.
func ClassIncludeToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{
			Tool: mcp.NewTool("GetClassInclude",
				mcp.WithDescription("Retrieve source code of a class include (definitions, implementations, macros, testclasses)"),
				mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
				mcp.WithString("include_type", mcp.Required(), mcp.Description("Include type: main, definitions, implementations, macros, testclasses")),
			),
			Handler:   HandleGetClassInclude,
			ReadOnly:  true,
			Endpoints: []string{"/sap/bc/adt/oo/classes"},
		},
		{
			Tool: mcp.NewTool("CreateTestInclude",
				mcp.WithDescription("Create the test classes include for a class (required before writing test code)"),
				mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
				mcp.WithString("lock_handle", mcp.Required(), mcp.Description("Lock handle from LockObject (lock the parent class first)")),
				mcp.WithString("connection", mcp.Description("Transport request number (optional for local packages)")),
			),
			Handler:   HandleCreateTestInclude,
			Endpoints: []string{"/sap/bc/adt/oo/classes"},
		},
		{
			Tool: mcp.NewTool("UpdateClassInclude",
				mcp.WithDescription("Update source code of a class include (requires lock on parent class)"),
				mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
				mcp.WithString("include_type", mcp.Required(), mcp.Description("Include type: main, definitions, implementations, macros, testclasses")),
				mcp.WithString("source", mcp.Required(), mcp.Description("ABAP source code to write")),
				mcp.WithString("lock_handle", mcp.Required(), mcp.Description("Lock handle from LockObject (lock the parent class first)")),
				mcp.WithString("connection", mcp.Description("Transport request number (optional for local packages)")),
			),
			Handler:   HandleUpdateClassInclude,
			Endpoints: []string{"/sap/bc/adt/oo/classes"},
		},
	}
}

// --- Class Include Handlers ---

func HandleGetClassInclude(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.GetArguments()["class_name"].(string)
	if !ok || className == "" {
		return types.ErrorResult("class_name is required"), nil
	}

	includeType, ok := request.GetArguments()["include_type"].(string)
	if !ok || includeType == "" {
		return types.ErrorResult("include_type is required"), nil
	}

	source, err := sys.ADT().GetClassInclude(ctx, className, adt.ClassIncludeType(includeType))
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get class include: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func HandleCreateTestInclude(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.GetArguments()["class_name"].(string)
	if !ok || className == "" {
		return types.ErrorResult("class_name is required"), nil
	}

	lockHandle, ok := request.GetArguments()["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return types.ErrorResult("lock_handle is required"), nil
	}

	transport := ""
	if t, ok := request.GetArguments()["connection"].(string); ok {
		transport = t
	}

	err := sys.ADT().CreateTestInclude(ctx, className, lockHandle, transport)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to create test include: %v", err)), nil
	}

	return mcp.NewToolResultText("Test include created successfully"), nil
}

func HandleUpdateClassInclude(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.GetArguments()["class_name"].(string)
	if !ok || className == "" {
		return types.ErrorResult("class_name is required"), nil
	}

	includeType, ok := request.GetArguments()["include_type"].(string)
	if !ok || includeType == "" {
		return types.ErrorResult("include_type is required"), nil
	}

	source, ok := request.GetArguments()["source"].(string)
	if !ok || source == "" {
		return types.ErrorResult("source is required"), nil
	}

	lockHandle, ok := request.GetArguments()["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return types.ErrorResult("lock_handle is required"), nil
	}

	transport := ""
	if t, ok := request.GetArguments()["connection"].(string); ok {
		transport = t
	}

	err := sys.ADT().UpdateClassInclude(ctx, className, adt.ClassIncludeType(includeType), source, lockHandle, transport)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to update class include: %v", err)), nil
	}

	return mcp.NewToolResultText("Class include updated successfully"), nil
}
