// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_git.go contains handlers for Git/abapGit operations via ZADT_VSP.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// GitToolDefs returns tool definitions for Git/abapGit integration tools.
func GitToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("GitTypes",
			mcp.WithDescription("Get list of supported abapGit object types. Returns 158 object types that can be exported/imported via abapGit. Requires abapGit to be installed on SAP system."),
		), Handler: HandleGitTypes, ReadOnly: true, Focused: true, Groups: []string{"G"}},
		{Tool: mcp.NewTool("GitExport",
			mcp.WithDescription("Export ABAP objects as abapGit-compatible ZIP. Supports 158 object types. Saves ZIP file to output_dir (default: current directory). Use packages OR objects parameter."),
			mcp.WithString("packages", mcp.Description("Comma-separated package names to export (e.g., '$ZRAY,$TMP'). Supports wildcards.")),
			mcp.WithString("objects", mcp.Description("JSON array of objects: [{\"type\":\"CLAS\",\"name\":\"ZCL_TEST\"}]")),
			mcp.WithBoolean("include_subpackages", mcp.Description("Include subpackages when exporting by package (default: true)")),
			mcp.WithString("output_dir", mcp.Description("Output directory for ZIP file (default: current directory)")),
		), Handler: HandleGitExport, ReadOnly: true, Focused: true, Groups: []string{"G"}},
	}
}

// --- Git/abapGit Handlers ---

func HandleGitTypes(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sys.IsRfcMode() {
		return types.ErrorResult("GitTypes is not available in RFC mode"), nil
	}
	if errResult := sys.EnsureWSConnected(ctx, "GitTypes"); errResult != nil {
		return errResult, nil
	}

	// Placeholder for GitTypes implementation in the new architecture
	return types.ErrorResult("GitTypes is not yet implemented in the new architecture"), nil
}

func HandleGitExport(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sys.IsRfcMode() {
		return types.ErrorResult("GitExport is not available in RFC mode"), nil
	}
	if errResult := sys.EnsureWSConnected(ctx, "GitExport"); errResult != nil {
		return errResult, nil
	}

	params := adt.GitExportParams{}

	// Parse packages
	if pkgStr, ok := request.GetArguments()["packages"].(string); ok && pkgStr != "" {
		params.Packages = strings.Split(pkgStr, ",")
		for i, p := range params.Packages {
			params.Packages[i] = strings.TrimSpace(p)
		}
	}

	// Parse objects
	if objsStr, ok := request.GetArguments()["objects"].(string); ok && objsStr != "" {
		var objs []adt.GitObjectRef
		if err := json.Unmarshal([]byte(objsStr), &objs); err != nil {
			return types.ErrorResult(fmt.Sprintf("Invalid objects JSON: %v", err)), nil
		}
		params.Objects = objs
	}

	// Include subpackages
	if inclSub, ok := request.GetArguments()["include_subpackages"].(bool); ok {
		params.IncludeSubpackages = inclSub
	} else {
		params.IncludeSubpackages = true // default
	}

	if len(params.Packages) == 0 && len(params.Objects) == 0 {
		return types.ErrorResult("Either packages or objects parameter is required"), nil
	}

	// Placeholder for GitExport implementation in the new architecture
	return types.ErrorResult("GitExport is not yet implemented in the new architecture"), nil
}
