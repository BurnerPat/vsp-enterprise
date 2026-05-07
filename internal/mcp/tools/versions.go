package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

// VersionToolDefs returns tool definitions for object version history tools.
func VersionToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{
			Tool: mcp.NewTool("GetObjectVersions",
				mcp.WithDescription("List version history (revisions) of an ABAP object. Returns versions with dates, authors, and transport requests. Use version URIs with GetObjectVersionSource or CompareObjectVersions."),
				mcp.WithString("type", mcp.Required(),
					mcp.Description("Object type: program, class, interface, function_module, include, cds_view, behavior_definition, service_definition, table (also accepts short codes: PROG, CLAS, INTF, FUNC, INCL, DDLS, BDEF, SRVD, TABL)")),
				mcp.WithString("name", mcp.Required(),
					mcp.Description("Object name")),
				mcp.WithString("include",
					mcp.Description("Class include type for CLAS: main, definitions, implementations, macros, testclasses (default: main)")),
				mcp.WithString("parent",
					mcp.Description("Function group name (required for FUNC type)")),
			),
			Handler:  HandleGetObjectVersions,
			ReadOnly: true,
			Focused:  true,
		},
		{
			Tool: mcp.NewTool("GetObjectVersionSource",
				mcp.WithDescription("Get source code of a specific version of an ABAP object. Use version_uri from GetObjectVersions output."),
				mcp.WithString("version_uri", mcp.Required(),
					mcp.Description("Version URI from GetObjectVersions result (the 'uri' field of a revision entry)")),
			),
			Handler:  HandleGetObjectVersionSource,
			ReadOnly: true,
			Focused:  true,
		},
		{
			Tool: mcp.NewTool("CompareObjectVersions",
				mcp.WithDescription("Compare two versions of an ABAP object with unified diff. "+
					"Works like `git diff <base> <target>`: lines prefixed with '-' exist only in base_version_uri, "+
					"lines prefixed with '+' exist only in target_version_uri. "+
					"Output is a git-style unified diff with `--- base/...` and `+++ target/...` headers. "+
					"For chronological diffs, pass the OLDER version as base_version_uri and the NEWER as target_version_uri."),
				mcp.WithString("base_version_uri", mcp.Required(),
					mcp.Description("Version URI for the base (--- side), like the first argument in `git diff <base> <target>`. Use the OLDER version here. From GetObjectVersions.")),
				mcp.WithString("target_version_uri", mcp.Required(),
					mcp.Description("Version URI for the target (+++ side), like the second argument in `git diff <base> <target>`. Use the NEWER version here. From GetObjectVersions.")),
			),
			Handler:  HandleCompareObjectVersions,
			ReadOnly: true,
			Focused:  true,
		},
	}
}

// --- Version History Handlers ---

func HandleGetObjectVersions(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, _ := request.GetArguments()["type"].(string)
	name, _ := request.GetArguments()["name"].(string)

	if objectType == "" || name == "" {
		return types.ErrorResult("type and name are required"), nil
	}

	include, _ := request.GetArguments()["include"].(string)
	parent, _ := request.GetArguments()["parent"].(string)

	ref, err := resolveRef(objectType, name, parent, include, "")
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Invalid object reference: %v", err)), nil
	}

	revisions, err := sys.ADT().GetObjectVersions(ctx, ref)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetObjectVersions failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(revisions, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleGetObjectVersionSource(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	versionURI, ok := request.GetArguments()["version_uri"].(string)
	if !ok || versionURI == "" {
		return types.ErrorResult("version_uri is required (from GetObjectVersions output)"), nil
	}

	source, err := sys.ADT().GetObjectVersionSource(ctx, versionURI)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetObjectVersionSource failed: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func HandleCompareObjectVersions(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	baseURI, _ := request.GetArguments()["base_version_uri"].(string)
	targetURI, _ := request.GetArguments()["target_version_uri"].(string)

	if baseURI == "" || targetURI == "" {
		return types.ErrorResult("base_version_uri and target_version_uri are required"), nil
	}

	diff, err := sys.ADT().CompareObjectVersions(ctx, baseURI, targetURI)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("CompareObjectVersions failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(diff, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
