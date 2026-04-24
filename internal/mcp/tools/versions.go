package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// VersionToolDefs returns tool definitions for object version history tools.
func VersionToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{
			Tool: mcp.NewTool("GetObjectVersions",
				mcp.WithDescription("List version history (revisions) of an ABAP object. Returns versions with dates, authors, and transport requests. Use version URIs with GetObjectVersionSource or CompareObjectVersions."),
				mcp.WithString("type", mcp.Required(),
					mcp.Description("Object type: PROG, CLAS, INTF, FUNC, INCL, DDLS, BDEF, SRVD")),
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
				mcp.WithDescription("Compare two versions of an ABAP object with unified diff. Use version URIs from GetObjectVersions. Use 'current' as version2_uri to compare against the active version."),
				mcp.WithString("type", mcp.Required(),
					mcp.Description("Object type: PROG, CLAS, INTF, FUNC, INCL, DDLS, BDEF, SRVD")),
				mcp.WithString("name", mcp.Required(),
					mcp.Description("Object name")),
				mcp.WithString("version1_uri", mcp.Required(),
					mcp.Description("Version URI for first (older) version, from GetObjectVersions")),
				mcp.WithString("version2_uri",
					mcp.Description("Version URI for second (newer) version, from GetObjectVersions. Default: 'current' (compare against active version)")),
				mcp.WithString("include",
					mcp.Description("Class include type for CLAS: main, definitions, implementations, macros, testclasses")),
				mcp.WithString("parent",
					mcp.Description("Function group name (required for FUNC type)")),
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

	opts := &adt.GetSourceOptions{}
	if include, ok := request.GetArguments()["include"].(string); ok && include != "" {
		opts.Include = include
	}
	if parent, ok := request.GetArguments()["parent"].(string); ok && parent != "" {
		opts.Parent = parent
	}

	revisions, err := sys.ADT().GetObjectVersions(ctx, objectType, name, opts)
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
	objectType, _ := request.GetArguments()["type"].(string)
	name, _ := request.GetArguments()["name"].(string)
	version1, _ := request.GetArguments()["version1_uri"].(string)
	version2, _ := request.GetArguments()["version2_uri"].(string)

	if objectType == "" || name == "" || version1 == "" {
		return types.ErrorResult("type, name, and version1_uri are required"), nil
	}
	if version2 == "" {
		version2 = "current"
	}

	opts := &adt.GetSourceOptions{}
	if include, ok := request.GetArguments()["include"].(string); ok && include != "" {
		opts.Include = include
	}
	if parent, ok := request.GetArguments()["parent"].(string); ok && parent != "" {
		opts.Parent = parent
	}

	diff, err := sys.ADT().CompareObjectVersions(ctx, objectType, name, version1, version2, opts)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("CompareObjectVersions failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(diff, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
