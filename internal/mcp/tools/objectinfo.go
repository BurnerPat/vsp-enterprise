package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

// ObjectInfoToolDefs returns tool definitions for ABAP object information tools.
func ObjectInfoToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("GetObjectProperties",
			mcp.WithDescription("Get metadata about an ABAP object: creator, package hierarchy, creation date, language, system, API release state, and description."),
			mcp.WithString("object_type", mcp.Required(),
				mcp.Description("Object type: CLAS, INTF, PROG, FUGR, FUNC, TABL, DDLS, DTEL, DOMA, SRVD, BDEF, SRVB")),
			mcp.WithString("name", mcp.Required(),
				mcp.Description("Object name (e.g., ZCL_MY_CLASS, Z_MY_PROGRAM)")),
		), Handler: HandleGetObjectProperties, ReadOnly: true, Focused: true,
			Endpoints: []string{"/sap/bc/adt/repository/informationsystem/objectproperties"},
			Routes: []types.UniversalRoute{{Action: "analyze", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				p["object_type"] = ot
				p["name"] = on
				return p
			}}}},

		{Tool: mcp.NewTool("GetObjectOutline",
			mcp.WithDescription("Get the structural outline of an ABAP object: methods, attributes, events, types, and other components with their visibility and properties. For classes/interfaces, optionally includes inherited members."),
			mcp.WithString("object_type", mcp.Required(),
				mcp.Description("Object type: CLAS, INTF, PROG, FUGR, FUNC, TABL, DDLS, DTEL, DOMA, SRVD, BDEF, SRVB")),
			mcp.WithString("name", mcp.Required(),
				mcp.Description("Object name (e.g., ZCL_MY_CLASS)")),
			mcp.WithBoolean("include_inherited",
				mcp.Description("Include inherited members in the outline (default: false). Only relevant for CLAS/INTF.")),
		), Handler: HandleGetObjectOutline, ReadOnly: true, Focused: true,
			Endpoints: []string{"/sap/bc/adt/oo/classes"},
			Routes: []types.UniversalRoute{{Action: "analyze", TargetType: "STRUCTURE", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				p["object_type"] = ot
				p["name"] = on
				return p
			}}}},

		{Tool: mcp.NewTool("GetObjectNetwork",
			mcp.WithDescription("Get the dependency network of an ABAP object: all directly used objects (classes, interfaces, tables, data elements, etc.) with their types, descriptions, and packages."),
			mcp.WithString("object_type", mcp.Required(),
				mcp.Description("Object type: CLAS, INTF, PROG, FUGR, FUNC, TABL, DDLS, DTEL, DOMA, SRVD, BDEF, SRVB")),
			mcp.WithString("name", mcp.Required(),
				mcp.Description("Object name (e.g., ZCL_MY_CLASS)")),
		), Handler: HandleGetObjectNetwork, ReadOnly: true, Focused: true,
			Endpoints: []string{"/sap/bc/adt/objectrelations/network"},
			Routes: []types.UniversalRoute{{Action: "analyze", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				p["object_type"] = ot
				p["name"] = on
				return p
			}}}},

		{Tool: mcp.NewTool("GetWhereUsed",
			mcp.WithDescription("Get where-used list: find all objects that reference a given ABAP object or one of its members. Optionally includes code snippets showing exact usage locations."),
			mcp.WithString("object_type", mcp.Required(),
				mcp.Description("Object type: CLAS, INTF, PROG, FUGR, FUNC, TABL, DDLS, DTEL, DOMA, SRVD, BDEF, SRVB")),
			mcp.WithString("name", mcp.Required(),
				mcp.Description("Object name (e.g., ZCL_MY_CLASS)")),
			mcp.WithString("member_uri",
				mcp.Description("ADT URI of a specific member to search for (e.g., from GetObjectOutline href). If omitted, searches for the object itself.")),
			mcp.WithBoolean("include_snippets",
				mcp.Description("Fetch code snippets showing the exact usage locations (default: false). Requires an additional server request.")),
		), Handler: HandleGetWhereUsed, ReadOnly: true, Focused: true,
			Endpoints: []string{"/sap/bc/adt/repository/informationsystem/usageReferences"},
			Routes: []types.UniversalRoute{{Action: "analyze", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				p["object_type"] = ot
				p["name"] = on
				return p
			}}}},
	}
}

// --- Object Info Handlers ---

func HandleGetObjectProperties(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.GetArguments()["object_type"].(string)
	if !ok || objectType == "" {
		return types.ErrorResult("object_type is required"), nil
	}
	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return types.ErrorResult("name is required"), nil
	}

	props, err := sys.ADT().GetObjectProperties(ctx, objectType, name)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetObjectProperties failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(props, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleGetObjectOutline(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.GetArguments()["object_type"].(string)
	if !ok || objectType == "" {
		return types.ErrorResult("object_type is required"), nil
	}
	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return types.ErrorResult("name is required"), nil
	}

	includeInherited := false
	if v, ok := request.GetArguments()["include_inherited"].(bool); ok {
		includeInherited = v
	}

	outline, err := sys.ADT().GetObjectOutline(ctx, objectType, name, includeInherited)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetObjectOutline failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(outline, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleGetObjectNetwork(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.GetArguments()["object_type"].(string)
	if !ok || objectType == "" {
		return types.ErrorResult("object_type is required"), nil
	}
	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return types.ErrorResult("name is required"), nil
	}

	network, err := sys.ADT().GetObjectNetwork(ctx, objectType, name)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetObjectNetwork failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(network, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleGetWhereUsed(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.GetArguments()["object_type"].(string)
	if !ok || objectType == "" {
		return types.ErrorResult("object_type is required"), nil
	}
	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return types.ErrorResult("name is required"), nil
	}

	memberURI, _ := request.GetArguments()["member_uri"].(string)

	includeSnippets := false
	if v, ok := request.GetArguments()["include_snippets"].(bool); ok {
		includeSnippets = v
	}

	result, err := sys.ADT().GetWhereUsed(ctx, objectType, name, memberURI, includeSnippets)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetWhereUsed failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
