// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_workflow.go contains handlers for high-level workflow operations.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

// WorkflowToolDefs returns tool definitions for workflow tools.
func WorkflowToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("WriteProgram",
			mcp.WithDescription("Update an existing program with syntax check and activation (Lock -> SyntaxCheck -> Update -> Unlock -> Activate)"),
			mcp.WithString("program_name", mcp.Required(), mcp.Description("Name of the ABAP program")),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP source code")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), Handler: HandleWriteProgram},

		{Tool: mcp.NewTool("WriteClass",
			mcp.WithDescription("Update an existing class with syntax check and activation (Lock -> SyntaxCheck -> Update -> Unlock -> Activate)"),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP class source code (definition and implementation)")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), Handler: HandleWriteClass},

		{Tool: mcp.NewTool("CreateAndActivateProgram",
			mcp.WithDescription("Create a new program with source code and activate it (Create -> Lock -> Update -> Unlock -> Activate)"),
			mcp.WithString("program_name", mcp.Required(), mcp.Description("Name of the ABAP program")),
			mcp.WithString("description", mcp.Required(), mcp.Description("Program description")),
			mcp.WithString("package_name", mcp.Required(), mcp.Description("Package name (e.g., $TMP for local)")),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP source code")),
			mcp.WithString("transport", mcp.Description("Transport request number (required for non-local packages)")),
		), Handler: HandleCreateAndActivateProgram},

		{Tool: mcp.NewTool("CreateClassWithTests",
			mcp.WithDescription("Create a new class with unit tests and run them (Create -> Lock -> Update -> CreateTestInclude -> UpdateTest -> Unlock -> Activate -> RunTests)"),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
			mcp.WithString("description", mcp.Required(), mcp.Description("Class description")),
			mcp.WithString("package_name", mcp.Required(), mcp.Description("Package name (e.g., $TMP for local)")),
			mcp.WithString("class_source", mcp.Required(), mcp.Description("ABAP class source code (definition and implementation)")),
			mcp.WithString("test_source", mcp.Required(), mcp.Description("ABAP unit test source code")),
			mcp.WithString("transport", mcp.Description("Transport request number (required for non-local packages)")),
		), Handler: HandleCreateClassWithTests},
	}
}

// --- Workflow Handlers ---

func HandleWriteProgram(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	programName, ok := request.GetArguments()["program_name"].(string)
	if !ok || programName == "" {
		return types.ErrorResult("program_name is required"), nil
	}

	source, ok := request.GetArguments()["source"].(string)
	if !ok || source == "" {
		return types.ErrorResult("source is required"), nil
	}

	transport := ""
	if t, ok := request.GetArguments()["transport"].(string); ok {
		transport = t
	}

	result, err := sys.ADT().WriteProgram(ctx, programName, source, transport)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("WriteProgram failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleWriteClass(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.GetArguments()["class_name"].(string)
	if !ok || className == "" {
		return types.ErrorResult("class_name is required"), nil
	}

	source, ok := request.GetArguments()["source"].(string)
	if !ok || source == "" {
		return types.ErrorResult("source is required"), nil
	}

	transport := ""
	if t, ok := request.GetArguments()["transport"].(string); ok {
		transport = t
	}

	result, err := sys.ADT().WriteClass(ctx, className, source, transport)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("WriteClass failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleCreateAndActivateProgram(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	programName, ok := request.GetArguments()["program_name"].(string)
	if !ok || programName == "" {
		return types.ErrorResult("program_name is required"), nil
	}

	description, ok := request.GetArguments()["description"].(string)
	if !ok || description == "" {
		return types.ErrorResult("description is required"), nil
	}

	packageName, ok := request.GetArguments()["package_name"].(string)
	if !ok || packageName == "" {
		return types.ErrorResult("package_name is required"), nil
	}

	source, ok := request.GetArguments()["source"].(string)
	if !ok || source == "" {
		return types.ErrorResult("source is required"), nil
	}

	transport := ""
	if t, ok := request.GetArguments()["transport"].(string); ok {
		transport = t
	}

	result, err := sys.ADT().CreateAndActivateProgram(ctx, programName, description, packageName, source, transport)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("CreateAndActivateProgram failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleCreateClassWithTests(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.GetArguments()["class_name"].(string)
	if !ok || className == "" {
		return types.ErrorResult("class_name is required"), nil
	}

	description, ok := request.GetArguments()["description"].(string)
	if !ok || description == "" {
		return types.ErrorResult("description is required"), nil
	}

	packageName, ok := request.GetArguments()["package_name"].(string)
	if !ok || packageName == "" {
		return types.ErrorResult("package_name is required"), nil
	}

	classSource, ok := request.GetArguments()["class_source"].(string)
	if !ok || classSource == "" {
		return types.ErrorResult("class_source is required"), nil
	}

	testSource, ok := request.GetArguments()["test_source"].(string)
	if !ok || testSource == "" {
		return types.ErrorResult("test_source is required"), nil
	}

	transport := ""
	if t, ok := request.GetArguments()["transport"].(string); ok {
		transport = t
	}

	result, err := sys.ADT().CreateClassWithTests(ctx, className, description, packageName, classSource, testSource, transport)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("CreateClassWithTests failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
