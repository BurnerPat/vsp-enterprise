// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_devtools.go contains handlers for development tools (syntax check, activation, unit tests).
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// DevToolDefs returns tool definitions for development tools.
func DevToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("SyntaxCheck",
			mcp.WithDescription("Check ABAP source code for syntax errors"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("content", mcp.Required(), mcp.Description("ABAP source code to check")),
		), Handler: HandleSyntaxCheck, ReadOnly: true, Focused: true, Endpoints: []string{"/sap/bc/adt/checkruns"}},

		{Tool: mcp.NewTool("Activate",
			mcp.WithDescription("Activate an ABAP object"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("object_name", mcp.Required(), mcp.Description("Technical name of the object (e.g., ZTEST)")),
		), Handler: HandleActivate, Focused: true, Endpoints: []string{"/sap/bc/adt/activation"}},

		{Tool: mcp.NewTool("ActivatePackage",
			mcp.WithDescription("Activate all inactive objects. Objects are sorted by dependency order (interfaces before classes). If no package specified, activates ALL inactive objects for current user."),
			mcp.WithString("package", mcp.Description("Package name to filter (optional, empty = all packages)")),
			mcp.WithNumber("max_objects", mcp.Description("Maximum number of objects to activate (default: 100)")),
		), Handler: HandleActivatePackage, Focused: true, Endpoints: []string{"/sap/bc/adt/activation"}},

		{Tool: mcp.NewTool("RunUnitTests",
			mcp.WithDescription("Run ABAP Unit tests for an object"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST)")),
			mcp.WithBoolean("include_dangerous", mcp.Description("Include dangerous risk level tests (default: false)")),
			mcp.WithBoolean("include_long", mcp.Description("Include long duration tests (default: false)")),
		), Handler: HandleRunUnitTests, ReadOnly: true, Focused: true, Groups: []string{"T"}, Endpoints: []string{"/sap/bc/adt/abapunit/testruns"}},

		{Tool: mcp.NewTool("RunATCCheck",
			mcp.WithDescription("Run ATC (ABAP Test Cockpit) code quality check on an object. Returns findings with priority, check title, message, and location. Priority: 1=Error, 2=Warning, 3=Info."),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST)")),
			mcp.WithString("variant", mcp.Description("Check variant name (empty = use system default)")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of findings to return (default: 100)")),
		), Handler: HandleRunATCCheck, ReadOnly: true, Focused: true, Groups: []string{"T"}, Endpoints: []string{"/sap/bc/adt/atc"}},

		{Tool: mcp.NewTool("PrettyPrint",
			mcp.WithDescription("Format ABAP source code using the pretty printer"),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP source code to format")),
		), Handler: HandlePrettyPrint, ReadOnly: true, Focused: true, Endpoints: []string{"/sap/bc/adt/abapsource/prettyprinter"}},

		{Tool: mcp.NewTool("GetInactiveObjects",
			mcp.WithDescription("Get all inactive objects for the current user - objects that have been modified but not yet activated"),
		), Handler: HandleGetInactiveObjects, ReadOnly: true, Focused: true, Endpoints: []string{"/sap/bc/adt/activation/inactiveobjects"}},

		{Tool: mcp.NewTool("ExecuteABAP",
			mcp.WithDescription("Execute arbitrary ABAP code via unit test wrapper. Creates temp program, injects code into test method, runs via RunUnitTests, extracts results from assertion messages, cleans up. Use lv_result variable to return output. WARNING: Powerful tool - use responsibly."),
			mcp.WithString("code", mcp.Required(), mcp.Description("ABAP code to execute. Set lv_result variable to return output via assertion message.")),
			mcp.WithString("risk_level", mcp.Description("Risk level: harmless (default, no DB writes), dangerous (can write to DB), critical (full access)")),
			mcp.WithString("return_variable", mcp.Description("Name of the variable to return (default: lv_result)")),
			mcp.WithBoolean("keep_program", mcp.Description("Don't delete temp program after execution (for debugging)")),
			mcp.WithString("program_prefix", mcp.Description("Prefix for temp program name (default: ZTEMP_EXEC_)")),
		), Handler: HandleExecuteABAP},
	}
}

// --- Development Tool Handlers ---

func HandleSyntaxCheck(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.GetArguments()["object_url"].(string)
	if !ok || objectURL == "" {
		return types.ErrorResult("object_url is required"), nil
	}

	content, ok := request.GetArguments()["content"].(string)
	if !ok || content == "" {
		return types.ErrorResult("content is required"), nil
	}

	results, err := sys.ADT().SyntaxCheck(ctx, objectURL, content)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Syntax check failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleActivate(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.GetArguments()["object_url"].(string)
	if !ok || objectURL == "" {
		return types.ErrorResult("object_url is required"), nil
	}

	objectName, ok := request.GetArguments()["object_name"].(string)
	if !ok || objectName == "" {
		return types.ErrorResult("object_name is required"), nil
	}

	result, err := sys.ADT().Activate(ctx, objectURL, objectName)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Activation failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleActivatePackage(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	packageName := ""
	if pkg, ok := request.GetArguments()["package"].(string); ok {
		packageName = pkg
	}

	maxObjects := 100
	if max, ok := request.GetArguments()["max_objects"].(float64); ok && max > 0 {
		maxObjects = int(max)
	}

	result, err := sys.ADT().ActivatePackage(ctx, packageName, maxObjects)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Batch activation failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleRunUnitTests(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.GetArguments()["object_url"].(string)
	if !ok || objectURL == "" {
		return types.ErrorResult("object_url is required"), nil
	}

	// Build flags from optional parameters
	flags := adt.DefaultUnitTestFlags()

	if includeDangerous, ok := request.GetArguments()["include_dangerous"].(bool); ok && includeDangerous {
		flags.Dangerous = true
	}

	if includeLong, ok := request.GetArguments()["include_long"].(bool); ok && includeLong {
		flags.Long = true
	}

	result, err := sys.ADT().RunUnitTests(ctx, objectURL, &flags)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Unit test run failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleRunATCCheck(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.GetArguments()["object_url"].(string)
	if !ok || objectURL == "" {
		return types.ErrorResult("object_url is required"), nil
	}

	variant := ""
	if v, ok := request.GetArguments()["variant"].(string); ok {
		variant = v
	}

	maxResults := 100
	if mr, ok := request.GetArguments()["max_results"].(float64); ok && mr > 0 {
		maxResults = int(mr)
	}

	result, err := sys.ADT().RunATCCheck(ctx, objectURL, variant, maxResults)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("ATC check failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleGetInactiveObjects(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objects, err := sys.ADT().GetInactiveObjects(ctx)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetInactiveObjects failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(objects, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandlePrettyPrint(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	source, ok := request.GetArguments()["source"].(string)
	if !ok || source == "" {
		return types.ErrorResult("source is required"), nil
	}

	formatted, err := sys.ADT().PrettyPrint(ctx, source)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("PrettyPrint failed: %v", err)), nil
	}

	return mcp.NewToolResultText(formatted), nil
}

func HandleExecuteABAP(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	code, ok := request.GetArguments()["code"].(string)
	if !ok || code == "" {
		return types.ErrorResult("code is required"), nil
	}

	riskLevel := "harmless"
	if rl, ok := request.GetArguments()["risk_level"].(string); ok && rl != "" {
		riskLevel = rl
	}

	returnVar := "lv_result"
	if rv, ok := request.GetArguments()["return_variable"].(string); ok && rv != "" {
		returnVar = rv
	}

	keepProgram := false
	if kp, ok := request.GetArguments()["keep_program"].(bool); ok {
		keepProgram = kp
	}

	programPrefix := "ZTEMP_EXEC_"
	if pp, ok := request.GetArguments()["program_prefix"].(string); ok && pp != "" {
		programPrefix = pp
	}

	opts := &adt.ExecuteABAPOptions{
		RiskLevel:      riskLevel,
		ReturnVariable: returnVar,
		KeepProgram:    keepProgram,
		ProgramPrefix:  programPrefix,
	}

	result, err := sys.ADT().ExecuteABAP(ctx, code, opts)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("ExecuteABAP failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
