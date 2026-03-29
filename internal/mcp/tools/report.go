// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_report.go contains handlers for report execution and text elements.
package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

// ReportToolDefs returns tool definitions for report execution tools.
func ReportToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("RunReport",
			mcp.WithDescription("Execute an ABAP selection-screen report with parameters or variant. Runs as background job and returns spool output. Requires ZADT_VSP WebSocket handler deployed."),
			mcp.WithString("report", mcp.Required(), mcp.Description("Report program name (e.g., 'RFITEMGL', 'ZREPORT_TEST')")),
			mcp.WithString("variant", mcp.Description("Variant name to use for selection screen (optional)")),
			mcp.WithString("params", mcp.Description("JSON object with selection screen parameters (e.g., '{\"P_BUKRS\":\"1000\",\"S_KUNNR\":{\"SIGN\":\"I\",\"OPTION\":\"EQ\",\"LOW\":\"0000001000\"}}'). Keys are parameter names.")),
		), Handler: HandleRunReport, Focused: true, Groups: []string{"R", "X"}},

		{Tool: mcp.NewTool("GetVariants",
			mcp.WithDescription("Get list of available variants for a report program. Returns variant names and whether they are protected."),
			mcp.WithString("report", mcp.Required(), mcp.Description("Report program name")),
		), Handler: HandleGetVariants, ReadOnly: true, Focused: true, Groups: []string{"R"}},

		{Tool: mcp.NewTool("GetTextElements",
			mcp.WithDescription("Get program text elements (selection texts and text symbols). Selection texts describe parameters (P_BUKRS='Company Code'), text symbols are TEXT-001 etc."),
			mcp.WithString("program", mcp.Required(), mcp.Description("Program name")),
			mcp.WithString("language", mcp.Description("Language key (e.g., 'E' for English, 'D' for German). Default: system language.")),
		), Handler: HandleGetTextElements, ReadOnly: true, Focused: true, Groups: []string{"R"}},

		{Tool: mcp.NewTool("SetTextElements",
			mcp.WithDescription("Set program text elements (selection texts, text symbols, and heading texts). Use for adding descriptions to selection screen parameters, text symbols, and list/column headings."),
			mcp.WithString("program", mcp.Required(), mcp.Description("Program name")),
			mcp.WithString("language", mcp.Description("Language key (e.g., 'E' for English, 'D' for German). Default: system language.")),
			mcp.WithString("selection_texts", mcp.Description("JSON object of selection texts (e.g., '{\"P_BUKRS\":\"Company Code\",\"S_KUNNR\":\"Customer Range\"}')")),
			mcp.WithString("text_symbols", mcp.Description("JSON object of text symbols (e.g., '{\"001\":\"Header Text\",\"002\":\"Footer\"}')")),
			mcp.WithString("heading_texts", mcp.Description("JSON object of heading texts for list/column headings (e.g., '{\"001\":\"Report Title\",\"002\":\"Column Header\"}')")),
		), Handler: HandleSetTextElements, Focused: true, Groups: []string{"R"}},
	}
}

// --- Report Execution Handlers ---

func HandleRunReport(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sys.IsRfcMode() {
		return types.ErrorResult("RunReport is not available in RFC mode"), nil
	}
	// Ensure WebSocket is connected
	if errResult := sys.EnsureWSConnected(ctx, "RunReport"); errResult != nil {
		return errResult, nil
	}

	// Placeholder for RunReport implementation in the new architecture
	return types.ErrorResult("RunReport is not yet implemented in the new architecture"), nil
}

func HandleGetVariants(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sys.IsRfcMode() {
		return types.ErrorResult("GetVariants is not available in RFC mode"), nil
	}
	if errResult := sys.EnsureWSConnected(ctx, "GetVariants"); errResult != nil {
		return errResult, nil
	}

	report, _ := request.GetArguments()["report"].(string)
	if report == "" {
		return types.ErrorResult("report parameter is required"), nil
	}

	// Placeholder for GetVariants implementation in the new architecture
	return types.ErrorResult("GetVariants is not yet implemented in the new architecture"), nil
}

func HandleGetTextElements(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sys.IsRfcMode() {
		return types.ErrorResult("GetTextElements is not available in RFC mode"), nil
	}
	if errResult := sys.EnsureWSConnected(ctx, "GetTextElements"); errResult != nil {
		return errResult, nil
	}

	program, _ := request.GetArguments()["program"].(string)
	if program == "" {
		return types.ErrorResult("program parameter is required"), nil
	}

	// Placeholder for GetTextElements implementation in the new architecture
	return types.ErrorResult("GetTextElements is not yet implemented in the new architecture"), nil
}

func HandleSetTextElements(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sys.IsRfcMode() {
		return types.ErrorResult("SetTextElements is not available in RFC mode"), nil
	}
	if errResult := sys.EnsureWSConnected(ctx, "SetTextElements"); errResult != nil {
		return errResult, nil
	}

	program, _ := request.GetArguments()["program"].(string)
	if program == "" {
		return types.ErrorResult("program parameter is required"), nil
	}

	// Placeholder for SetTextElements implementation in the new architecture
	return types.ErrorResult("SetTextElements is not yet implemented in the new architecture"), nil
}
