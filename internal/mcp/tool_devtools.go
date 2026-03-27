// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_devtools.go contains handlers for development tools (syntax check, activation, unit tests).
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// devToolDefs returns tool definitions for development tools.
func (s *Server) devToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("SyntaxCheck",
			mcp.WithDescription("Check ABAP source code for syntax errors"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("content", mcp.Required(), mcp.Description("ABAP source code to check")),
		), handler: s.handleSyntaxCheck, readOnly: true, focused: true},

		{tool: mcp.NewTool("Activate",
			mcp.WithDescription("Activate an ABAP object"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("object_name", mcp.Required(), mcp.Description("Technical name of the object (e.g., ZTEST)")),
		), handler: s.handleActivate, focused: true},

		{tool: mcp.NewTool("ActivatePackage",
			mcp.WithDescription("Activate all inactive objects. Objects are sorted by dependency order (interfaces before classes). If no package specified, activates ALL inactive objects for current user."),
			mcp.WithString("package", mcp.Description("Package name to filter (optional, empty = all packages)")),
			mcp.WithNumber("max_objects", mcp.Description("Maximum number of objects to activate (default: 100)")),
		), handler: s.handleActivatePackage, focused: true},

		{tool: mcp.NewTool("RunUnitTests",
			mcp.WithDescription("Run ABAP Unit tests for an object"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST)")),
			mcp.WithBoolean("include_dangerous", mcp.Description("Include dangerous risk level tests (default: false)")),
			mcp.WithBoolean("include_long", mcp.Description("Include long duration tests (default: false)")),
		), handler: s.handleRunUnitTests, readOnly: true, focused: true, groups: []string{"T"}},

		{tool: mcp.NewTool("RunATCCheck",
			mcp.WithDescription("Run ATC (ABAP Test Cockpit) code quality check on an object. Returns findings with priority, check title, message, and location. Priority: 1=Error, 2=Warning, 3=Info."),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST)")),
			mcp.WithString("variant", mcp.Description("Check variant name (empty = use system default)")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of findings to return (default: 100)")),
		), handler: s.handleRunATCCheck, readOnly: true, focused: true, groups: []string{"T"}},

		{tool: mcp.NewTool("GetATCCustomizing",
			mcp.WithDescription("Get ATC system configuration including default check variant and exemption reasons"),
		), handler: s.handleGetATCCustomizing, readOnly: true},

		{tool: mcp.NewTool("PrettyPrint",
			mcp.WithDescription("Format ABAP source code using the pretty printer"),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP source code to format")),
		), handler: s.handlePrettyPrint, readOnly: true, focused: true,
			routes: []universalRoute{{action: "analyze", paramsType: "pretty_print"}}},

		{tool: mcp.NewTool("GetPrettyPrinterSettings",
			mcp.WithDescription("Get the current pretty printer (code formatter) settings"),
		), handler: s.handleGetPrettyPrinterSettings, readOnly: true,
			routes: []universalRoute{{action: "analyze", paramsType: "get_pretty_printer_settings"}}},

		{tool: mcp.NewTool("SetPrettyPrinterSettings",
			mcp.WithDescription("Update the pretty printer (code formatter) settings"),
			mcp.WithBoolean("indentation", mcp.Required(), mcp.Description("Enable automatic indentation")),
			mcp.WithString("style", mcp.Required(), mcp.Description("Keyword style: toLower, toUpper, keywordUpper, keywordLower, keywordAuto, none")),
		), handler: s.handleSetPrettyPrinterSettings,
			routes: []universalRoute{{action: "analyze", paramsType: "set_pretty_printer_settings"}}},

		{tool: mcp.NewTool("GetInactiveObjects",
			mcp.WithDescription("Get all inactive objects for the current user - objects that have been modified but not yet activated"),
		), handler: s.handleGetInactiveObjects, readOnly: true, focused: true,
			routes: []universalRoute{{action: "analyze", paramsType: "inactive_objects"}}},

		{tool: mcp.NewTool("ExecuteABAP",
			mcp.WithDescription("Execute arbitrary ABAP code via unit test wrapper. Creates temp program, injects code into test method, runs via RunUnitTests, extracts results from assertion messages, cleans up. Use lv_result variable to return output. WARNING: Powerful tool - use responsibly."),
			mcp.WithString("code", mcp.Required(), mcp.Description("ABAP code to execute. Set lv_result variable to return output via assertion message.")),
			mcp.WithString("risk_level", mcp.Description("Risk level: harmless (default, no DB writes), dangerous (can write to DB), critical (full access)")),
			mcp.WithString("return_variable", mcp.Description("Name of the variable to return (default: lv_result)")),
			mcp.WithBoolean("keep_program", mcp.Description("Don't delete temp program after execution (for debugging)")),
			mcp.WithString("program_prefix", mcp.Description("Prefix for temp program name (default: ZTEMP_EXEC_)")),
		), handler: s.handleExecuteABAP},
	}
}

// routeDevToolsAction routes "test" (unit tests), "analyze" (syntax check), "edit" (activate), and "analyze" (execute_abap).
func (s *Server) routeDevToolsAction(ctx context.Context, action, objectType, objectName string, params map[string]any) (*mcp.CallToolResult, bool, error) {
	if action == "test" {
		analysisType := getStringParam(params, "type")
		if analysisType == "" || analysisType == "unit" {
			// Unit tests
			objectURL := getStringParam(params, "object_url")
			if objectURL == "" {
				return nil, false, nil
			}
			args := map[string]any{"object_url": objectURL}
			if v, ok := getBoolParam(params, "include_dangerous"); ok {
				args["include_dangerous"] = v
			}
			if v, ok := getBoolParam(params, "include_long"); ok {
				args["include_long"] = v
			}
			return s.callHandler(ctx, s.handleRunUnitTests, args)
		}
	}

	if action == "analyze" {
		analysisType := getStringParam(params, "type")
		switch analysisType {
		case "syntax_check":
			return s.callHandler(ctx, s.handleSyntaxCheck, params)
		case "execute_abap":
			return s.callHandler(ctx, s.handleExecuteABAP, params)
		}
	}

	if action == "edit" {
		switch objectType {
		case "ACTIVATE":
			return s.callHandler(ctx, s.handleActivate, params)
		case "ACTIVATE_PACKAGE":
			return s.callHandler(ctx, s.handleActivatePackage, params)
		}
	}

	return nil, false, nil
}

// --- Development Tool Handlers ---

func (s *Server) handleSyntaxCheck(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	content, ok := request.Params.Arguments["content"].(string)
	if !ok || content == "" {
		return newToolResultError("content is required"), nil
	}

	results, err := s.adtClient.SyntaxCheck(ctx, objectURL, content)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Syntax check failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleActivate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	objectName, ok := request.Params.Arguments["object_name"].(string)
	if !ok || objectName == "" {
		return newToolResultError("object_name is required"), nil
	}

	result, err := s.adtClient.Activate(ctx, objectURL, objectName)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Activation failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleActivatePackage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	packageName := ""
	if pkg, ok := request.Params.Arguments["package"].(string); ok {
		packageName = pkg
	}

	maxObjects := 100
	if max, ok := request.Params.Arguments["max_objects"].(float64); ok && max > 0 {
		maxObjects = int(max)
	}

	result, err := s.adtClient.ActivatePackage(ctx, packageName, maxObjects)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Batch activation failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleRunUnitTests(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	// Build flags from optional parameters
	flags := adt.DefaultUnitTestFlags()

	if includeDangerous, ok := request.Params.Arguments["include_dangerous"].(bool); ok && includeDangerous {
		flags.Dangerous = true
	}

	if includeLong, ok := request.Params.Arguments["include_long"].(bool); ok && includeLong {
		flags.Long = true
	}

	result, err := s.adtClient.RunUnitTests(ctx, objectURL, &flags)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Unit test run failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
