// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_classinclude.go contains handlers for class include operations (testclasses, locals_def, etc.).
package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// classIncludeToolDefs returns tool definitions for class include operations.
func (s *Server) classIncludeToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("GetClassInclude",
			mcp.WithDescription("Retrieve source code of a class include (definitions, implementations, macros, testclasses)"),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
			mcp.WithString("include_type", mcp.Required(), mcp.Description("Include type: main, definitions, implementations, macros, testclasses")),
		), handler: s.handleGetClassInclude, readOnly: true},

		{tool: mcp.NewTool("CreateTestInclude",
			mcp.WithDescription("Create the test classes include for a class (required before writing test code)"),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
			mcp.WithString("lock_handle", mcp.Required(), mcp.Description("Lock handle from LockObject (lock the parent class first)")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleCreateTestInclude},

		{tool: mcp.NewTool("UpdateClassInclude",
			mcp.WithDescription("Update source code of a class include (requires lock on parent class)"),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
			mcp.WithString("include_type", mcp.Required(), mcp.Description("Include type: main, definitions, implementations, macros, testclasses")),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP source code to write")),
			mcp.WithString("lock_handle", mcp.Required(), mcp.Description("Lock handle from LockObject (lock the parent class first)")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleUpdateClassInclude},

		{tool: mcp.NewTool("PublishServiceBinding",
			mcp.WithDescription("Publish a service binding to make it available as OData service"),
			mcp.WithString("service_name", mcp.Required(), mcp.Description("Service binding name (e.g., ZTRAVEL_SB)")),
			mcp.WithString("service_version", mcp.Description("Service version (default: 0001)")),
		), handler: s.handlePublishServiceBinding},

		{tool: mcp.NewTool("UnpublishServiceBinding",
			mcp.WithDescription("Unpublish a service binding"),
			mcp.WithString("service_name", mcp.Required(), mcp.Description("Service binding name (e.g., ZTRAVEL_SB)")),
			mcp.WithString("service_version", mcp.Description("Service version (default: 0001)")),
		), handler: s.handleUnpublishServiceBinding},
	}
}

// routeClassIncludeAction routes class include operations.
func (s *Server) routeClassIncludeAction(ctx context.Context, action, objectType, objectName string, params map[string]any) (*mcp.CallToolResult, bool, error) {
	switch {
	case action == "read" && objectType == "CLAS_INCLUDE":
		return s.callHandler(ctx, s.handleGetClassInclude, map[string]any{
			"class_name":   objectName,
			"include_type": getStringParam(params, "include_type"),
		})
	case action == "create" && objectType == "CLAS_TEST_INCLUDE":
		return s.callHandler(ctx, s.handleCreateTestInclude, params)
	case action == "edit" && objectType == "CLAS_INCLUDE":
		return s.callHandler(ctx, s.handleUpdateClassInclude, params)
	}
	return nil, false, nil
}

// --- Class Include Handlers ---

func (s *Server) handleGetClassInclude(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.Params.Arguments["class_name"].(string)
	if !ok || className == "" {
		return newToolResultError("class_name is required"), nil
	}

	includeType, ok := request.Params.Arguments["include_type"].(string)
	if !ok || includeType == "" {
		return newToolResultError("include_type is required"), nil
	}

	source, err := s.adtClient.GetClassInclude(ctx, className, adt.ClassIncludeType(includeType))
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get class include: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func (s *Server) handleCreateTestInclude(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.Params.Arguments["class_name"].(string)
	if !ok || className == "" {
		return newToolResultError("class_name is required"), nil
	}

	lockHandle, ok := request.Params.Arguments["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return newToolResultError("lock_handle is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	err := s.adtClient.CreateTestInclude(ctx, className, lockHandle, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to create test include: %v", err)), nil
	}

	return mcp.NewToolResultText("Test include created successfully"), nil
}

func (s *Server) handleUpdateClassInclude(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.Params.Arguments["class_name"].(string)
	if !ok || className == "" {
		return newToolResultError("class_name is required"), nil
	}

	includeType, ok := request.Params.Arguments["include_type"].(string)
	if !ok || includeType == "" {
		return newToolResultError("include_type is required"), nil
	}

	source, ok := request.Params.Arguments["source"].(string)
	if !ok || source == "" {
		return newToolResultError("source is required"), nil
	}

	lockHandle, ok := request.Params.Arguments["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return newToolResultError("lock_handle is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	err := s.adtClient.UpdateClassInclude(ctx, className, adt.ClassIncludeType(includeType), source, lockHandle, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to update class include: %v", err)), nil
	}

	return mcp.NewToolResultText("Class include updated successfully"), nil
}
