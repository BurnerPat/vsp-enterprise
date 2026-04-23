// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_codeintel.go contains handlers for code intelligence operations
// (find definition, references, completion, pretty print, type hierarchy, etc.).
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

// CodeIntelToolDefs returns tool definitions for code intelligence tools.
func CodeIntelToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("FindDefinition",
			mcp.WithDescription("Navigate to the definition of a symbol at a given position in source code"),
			mcp.WithString("source_url", mcp.Required(), mcp.Description("ADT URL of the source file (e.g., /sap/bc/adt/programs/programs/ZTEST/source/main)")),
			mcp.WithString("source", mcp.Required(), mcp.Description("Full source code of the file")),
			mcp.WithNumber("line", mcp.Required(), mcp.Description("Line number (1-based)")),
			mcp.WithNumber("start_column", mcp.Required(), mcp.Description("Start column of the symbol (1-based)")),
			mcp.WithNumber("end_column", mcp.Required(), mcp.Description("End column of the symbol (1-based)")),
			mcp.WithBoolean("implementation", mcp.Description("Navigate to implementation instead of definition (default: false)")),
			mcp.WithString("main_program", mcp.Description("Main program for includes (optional)")),
		), Handler: HandleFindDefinition, ReadOnly: true, Focused: true, Endpoints: []string{"/sap/bc/adt/navigation/target"}},

		{Tool: mcp.NewTool("FindReferences",
			mcp.WithDescription("Find all references to an ABAP object or symbol"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST)")),
			mcp.WithNumber("line", mcp.Description("Line number for position-based reference search (1-based, optional)")),
			mcp.WithNumber("column", mcp.Description("Column number for position-based reference search (1-based, optional)")),
		), Handler: HandleFindReferences, ReadOnly: true, Focused: true, Endpoints: []string{"/sap/bc/adt/repository/informationsystem/usageReferences"}},

		{Tool: mcp.NewTool("CodeCompletion",
			mcp.WithDescription("Get code completion suggestions at a position in source code"),
			mcp.WithString("source_url", mcp.Required(), mcp.Description("ADT URL of the source file (e.g., /sap/bc/adt/programs/programs/ZTEST/source/main)")),
			mcp.WithString("source", mcp.Required(), mcp.Description("Full source code of the file")),
			mcp.WithNumber("line", mcp.Required(), mcp.Description("Line number (1-based)")),
			mcp.WithNumber("column", mcp.Required(), mcp.Description("Column number (1-based)")),
		), Handler: HandleCodeCompletion, ReadOnly: true, Endpoints: []string{"/sap/bc/adt/abapsource/codecompletion"}},

		{Tool: mcp.NewTool("GetTypeHierarchy",
			mcp.WithDescription("Get the type hierarchy (supertypes or subtypes) for a class/interface"),
			mcp.WithString("source_url", mcp.Required(), mcp.Description("ADT URL of the source file")),
			mcp.WithString("source", mcp.Required(), mcp.Description("Full source code of the file")),
			mcp.WithNumber("line", mcp.Required(), mcp.Description("Line number (1-based)")),
			mcp.WithNumber("column", mcp.Required(), mcp.Description("Column number (1-based)")),
			mcp.WithBoolean("super_types", mcp.Description("Get supertypes instead of subtypes (default: false = subtypes)")),
		), Handler: HandleGetTypeHierarchy, ReadOnly: true, Endpoints: []string{"/sap/bc/adt/abapsource/typehierarchy"}},

	}
}

// --- Code Intelligence Handlers ---

func HandleFindDefinition(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceURL, ok := request.GetArguments()["source_url"].(string)
	if !ok || sourceURL == "" {
		return types.ErrorResult("source_url is required"), nil
	}

	source, ok := request.GetArguments()["source"].(string)
	if !ok || source == "" {
		return types.ErrorResult("source is required"), nil
	}

	lineF, ok := request.GetArguments()["line"].(float64)
	if !ok {
		return types.ErrorResult("line is required"), nil
	}
	line := int(lineF)

	startColF, ok := request.GetArguments()["start_column"].(float64)
	if !ok {
		return types.ErrorResult("start_column is required"), nil
	}
	startCol := int(startColF)

	endColF, ok := request.GetArguments()["end_column"].(float64)
	if !ok {
		return types.ErrorResult("end_column is required"), nil
	}
	endCol := int(endColF)

	implementation := false
	if impl, ok := request.GetArguments()["implementation"].(bool); ok {
		implementation = impl
	}

	mainProgram := ""
	if mp, ok := request.GetArguments()["main_program"].(string); ok {
		mainProgram = mp
	}

	result, err := sys.ADT().FindDefinition(ctx, sourceURL, source, line, startCol, endCol, implementation, mainProgram)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("FindDefinition failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleFindReferences(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.GetArguments()["object_url"].(string)
	if !ok || objectURL == "" {
		return types.ErrorResult("object_url is required"), nil
	}

	line := 0
	column := 0
	if lineF, ok := request.GetArguments()["line"].(float64); ok {
		line = int(lineF)
	}
	if colF, ok := request.GetArguments()["column"].(float64); ok {
		column = int(colF)
	}

	results, err := sys.ADT().FindReferences(ctx, objectURL, line, column)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("FindReferences failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleCodeCompletion(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceURL, ok := request.GetArguments()["source_url"].(string)
	if !ok || sourceURL == "" {
		return types.ErrorResult("source_url is required"), nil
	}

	source, ok := request.GetArguments()["source"].(string)
	if !ok || source == "" {
		return types.ErrorResult("source is required"), nil
	}

	lineF, ok := request.GetArguments()["line"].(float64)
	if !ok {
		return types.ErrorResult("line is required"), nil
	}
	line := int(lineF)

	colF, ok := request.GetArguments()["column"].(float64)
	if !ok {
		return types.ErrorResult("column is required"), nil
	}
	column := int(colF)

	proposals, err := sys.ADT().CodeCompletion(ctx, sourceURL, source, line, column)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("CodeCompletion failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(proposals, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleGetTypeHierarchy(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceURL, ok := request.GetArguments()["source_url"].(string)
	if !ok || sourceURL == "" {
		return types.ErrorResult("source_url is required"), nil
	}

	source, ok := request.GetArguments()["source"].(string)
	if !ok || source == "" {
		return types.ErrorResult("source is required"), nil
	}

	lineF, ok := request.GetArguments()["line"].(float64)
	if !ok {
		return types.ErrorResult("line is required"), nil
	}
	line := int(lineF)

	colF, ok := request.GetArguments()["column"].(float64)
	if !ok {
		return types.ErrorResult("column is required"), nil
	}
	column := int(colF)

	superTypes := false
	if st, ok := request.GetArguments()["super_types"].(bool); ok {
		superTypes = st
	}

	hierarchy, err := sys.ADT().GetTypeHierarchy(ctx, sourceURL, source, line, column, superTypes)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetTypeHierarchy failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(hierarchy, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}


