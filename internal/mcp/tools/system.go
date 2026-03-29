package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

// SystemToolDefs returns tool definitions for system information tools.
func SystemToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("GetSystemInfo",
			mcp.WithDescription("Get SAP system information (system ID, release, kernel, database)"),
		), Handler: HandleGetSystemInfo, AlwaysOn: true, ReadOnly: true,
			Routes: []types.UniversalRoute{{Action: "system", TargetType: "INFO"}}},

		{Tool: mcp.NewTool("GetInstalledComponents",
			mcp.WithDescription("List installed software components with version information"),
		), Handler: HandleGetInstalledComponents, AlwaysOn: true, ReadOnly: true,
			Routes: []types.UniversalRoute{{Action: "system", TargetType: "COMPONENTS"}}},

		{Tool: mcp.NewTool("GetConnectionInfo",
			mcp.WithDescription("Get current MCP connection info: user, URL, client. Useful for debugging and understanding current session context."),
		), Handler: HandleGetConnectionInfo, AlwaysOn: true, ReadOnly: true,
			Routes: []types.UniversalRoute{{Action: "system", TargetType: "CONNECTION"}}},

		{Tool: mcp.NewTool("GetFeatures",
			mcp.WithDescription("Probe SAP system for available features. Returns status of optional capabilities like abapGit, RAP/OData, AMDP debugging, UI5/BSP, and CTS transports. Use this to understand what features are available before attempting to use them."),
		), Handler: HandleGetFeatures, AlwaysOn: true, ReadOnly: true,
			Routes: []types.UniversalRoute{{Action: "system", TargetType: "FEATURES"}}},

		{Tool: mcp.NewTool("GetAbapHelp",
			mcp.WithDescription("Get ABAP keyword documentation. Returns URL to SAP Help Portal and search query. If ZADT_VSP is installed, also returns real documentation from SAP system."),
			mcp.WithString("keyword", mcp.Required(), mcp.Description("ABAP keyword (e.g., SELECT, LOOP, DATA, METHOD, READ TABLE)")),
		), Handler: HandleGetAbapHelp, AlwaysOn: true, ReadOnly: true,
			Routes: []types.UniversalRoute{{Action: "analyze", ParamsType: "abap_help"}}},
	}
}

// --- System Information Handlers ---

func HandleGetSystemInfo(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	info, err := sys.ADT().GetSystemInfo(ctx)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get system info: %v", err)), nil
	}

	result, _ := json.MarshalIndent(info, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleGetInstalledComponents(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	components, err := sys.ADT().GetInstalledComponents(ctx)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get installed components: %v", err)), nil
	}

	result, _ := json.MarshalIndent(components, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleGetConnectionInfo(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Return current connection info for introspection
	info := map[string]interface{}{
		"status": "connected",
	}

	result, _ := json.MarshalIndent(info, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleGetFeatures(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("Features info not yet available in new router architecture"), nil
}

func HandleGetAbapHelp(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyword, _ := request.GetArguments()["keyword"].(string)
	if keyword == "" {
		return types.ErrorResult("keyword is required"), nil
	}

	helpResult, err := sys.ADT().GetAbapHelp(ctx, keyword)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetAbapHelp failed: %v", err)), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "ABAP Keyword: %s\n\n", helpResult.Keyword)
	fmt.Fprintf(&sb, "Documentation URL:\n  %s\n\n", helpResult.URL)
	fmt.Fprintf(&sb, "Search Query:\n  %s\n", helpResult.SearchQuery)

	if helpResult.Documentation != "" {
		fmt.Fprintf(&sb, "\n---\nDocumentation from SAP system:\n\n%s", helpResult.Documentation)
	}

	return mcp.NewToolResultText(sb.String()), nil
}
