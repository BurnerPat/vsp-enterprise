// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_workflow.go contains handlers for high-level workflow operations.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// workflowToolDefs returns tool definitions for workflow tools.
func (s *Server) workflowToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("WriteProgram",
			mcp.WithDescription("Update an existing program with syntax check and activation (Lock -> SyntaxCheck -> Update -> Unlock -> Activate)"),
			mcp.WithString("program_name", mcp.Required(), mcp.Description("Name of the ABAP program")),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP source code")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleWriteProgram},

		{tool: mcp.NewTool("WriteClass",
			mcp.WithDescription("Update an existing class with syntax check and activation (Lock -> SyntaxCheck -> Update -> Unlock -> Activate)"),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP class source code (definition and implementation)")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleWriteClass},

		{tool: mcp.NewTool("CreateAndActivateProgram",
			mcp.WithDescription("Create a new program with source code and activate it (Create -> Lock -> Update -> Unlock -> Activate)"),
			mcp.WithString("program_name", mcp.Required(), mcp.Description("Name of the ABAP program")),
			mcp.WithString("description", mcp.Required(), mcp.Description("Program description")),
			mcp.WithString("package_name", mcp.Required(), mcp.Description("Package name (e.g., $TMP for local)")),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP source code")),
			mcp.WithString("transport", mcp.Description("Transport request number (required for non-local packages)")),
		), handler: s.handleCreateAndActivateProgram},

		{tool: mcp.NewTool("CreateClassWithTests",
			mcp.WithDescription("Create a new class with unit tests and run them (Create -> Lock -> Update -> CreateTestInclude -> UpdateTest -> Unlock -> Activate -> RunTests)"),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
			mcp.WithString("description", mcp.Required(), mcp.Description("Class description")),
			mcp.WithString("package_name", mcp.Required(), mcp.Description("Package name (e.g., $TMP for local)")),
			mcp.WithString("class_source", mcp.Required(), mcp.Description("ABAP class source code (definition and implementation)")),
			mcp.WithString("test_source", mcp.Required(), mcp.Description("ABAP unit test source code")),
			mcp.WithString("transport", mcp.Description("Transport request number (required for non-local packages)")),
		), handler: s.handleCreateClassWithTests},
	}
}

// routeWorkflowAction routes workflow operations for high-level create/edit.
func (s *Server) routeWorkflowAction(ctx context.Context, action, objectType, objectName string, params map[string]any) (*mcp.CallToolResult, bool, error) {
	if action == "edit" {
		editType := getStringParam(params, "type")
		switch editType {
		case "write_program":
			return s.callHandler(ctx, s.handleWriteProgram, params)
		case "write_class":
			return s.callHandler(ctx, s.handleWriteClass, params)
		}
	}
	if action == "create" {
		switch objectType {
		case "PROGRAM":
			return s.callHandler(ctx, s.handleCreateAndActivateProgram, params)
		case "CLASS_WITH_TESTS":
			return s.callHandler(ctx, s.handleCreateClassWithTests, params)
		}
	}
	return nil, false, nil
}

// --- Workflow Handlers ---

func (s *Server) handleWriteProgram(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	programName, ok := request.Params.Arguments["program_name"].(string)
	if !ok || programName == "" {
		return newToolResultError("program_name is required"), nil
	}

	source, ok := request.Params.Arguments["source"].(string)
	if !ok || source == "" {
		return newToolResultError("source is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	result, err := s.adtClient.WriteProgram(ctx, programName, source, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("WriteProgram failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleWriteClass(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.Params.Arguments["class_name"].(string)
	if !ok || className == "" {
		return newToolResultError("class_name is required"), nil
	}

	source, ok := request.Params.Arguments["source"].(string)
	if !ok || source == "" {
		return newToolResultError("source is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	result, err := s.adtClient.WriteClass(ctx, className, source, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("WriteClass failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleCreateAndActivateProgram(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	programName, ok := request.Params.Arguments["program_name"].(string)
	if !ok || programName == "" {
		return newToolResultError("program_name is required"), nil
	}

	description, ok := request.Params.Arguments["description"].(string)
	if !ok || description == "" {
		return newToolResultError("description is required"), nil
	}

	packageName, ok := request.Params.Arguments["package_name"].(string)
	if !ok || packageName == "" {
		return newToolResultError("package_name is required"), nil
	}

	source, ok := request.Params.Arguments["source"].(string)
	if !ok || source == "" {
		return newToolResultError("source is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	result, err := s.adtClient.CreateAndActivateProgram(ctx, programName, description, packageName, source, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("CreateAndActivateProgram failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleCreateClassWithTests(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.Params.Arguments["class_name"].(string)
	if !ok || className == "" {
		return newToolResultError("class_name is required"), nil
	}

	description, ok := request.Params.Arguments["description"].(string)
	if !ok || description == "" {
		return newToolResultError("description is required"), nil
	}

	packageName, ok := request.Params.Arguments["package_name"].(string)
	if !ok || packageName == "" {
		return newToolResultError("package_name is required"), nil
	}

	classSource, ok := request.Params.Arguments["class_source"].(string)
	if !ok || classSource == "" {
		return newToolResultError("class_source is required"), nil
	}

	testSource, ok := request.Params.Arguments["test_source"].(string)
	if !ok || testSource == "" {
		return newToolResultError("test_source is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	result, err := s.adtClient.CreateClassWithTests(ctx, className, description, packageName, classSource, testSource, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("CreateClassWithTests failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
