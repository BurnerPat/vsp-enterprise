package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// UnifiedToolDefs returns tool definitions for the unified GetSource/WriteSource tools.
func UnifiedToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{
			ReadOnly: true, Focused: true,
			Tool: mcp.NewTool("GetSource",
				mcp.WithDescription("Unified tool for reading ABAP source code across different object types. Replaces GetProgram, GetClass, GetInterface, GetFunction, GetInclude, GetFunctionGroup, GetClassInclude."),
				mcp.WithString("object_type",
					mcp.Required(),
					mcp.Description("Object type: program, class, interface, function_module, function_group, include, cds_view, view, behavior_definition, service_definition, service_binding, message_class (also accepts: PROG, CLAS, INTF, FUNC, FUGR, INCL, DDLS, VIEW, BDEF, SRVD, SRVB, MSAG)"),
				),
				mcp.WithString("name",
					mcp.Required(),
					mcp.Description("Object name (e.g., program name, class name, function module name)"),
				),
				mcp.WithString("parent",
					mcp.Description("Function group name (required only for FUNC type)"),
				),
				mcp.WithString("include",
					mcp.Description("Class include type for CLAS: definitions, implementations, macros, testclasses (optional)"),
				),
				mcp.WithString("method",
					mcp.Description("Method name for CLAS only: returns only the METHOD...ENDMETHOD block for the specified method (optional)"),
				),
				mcp.WithBoolean("include_context",
					mcp.Description("Append compressed dependency context showing public API contracts of referenced classes/interfaces/FMs (default: true). Set to false to get raw source only."),
				),
				mcp.WithNumber("max_deps",
					mcp.Description("Maximum dependencies to resolve when include_context=true (default: 20)"),
				),
			),
			Handler:   HandleGetSource,
			Endpoints: []string{"/sap/bc/adt/programs/programs"},
			Routes: []types.UniversalRoute{
				{Action: "read", MapArgs: func(ot, on string, p map[string]any) map[string]any {
					p["object_type"] = ot
					p["name"] = on
					return p
				}},
			},
		},

		{
			Focused: true,
			Tool: mcp.NewTool("WriteSource",
				mcp.WithDescription("Unified tool for writing ABAP source code with automatic create/update detection. Supports PROG, CLAS, INTF, and RAP types (DDLS, BDEF, SRVD)."),
				mcp.WithString("object_type",
					mcp.Required(),
					mcp.Description("Object type: program, class, interface, cds_view, behavior_definition, service_definition (also accepts: PROG, CLAS, INTF, DDLS, BDEF, SRVD, SRVB)"),
				),
				mcp.WithString("name",
					mcp.Required(),
					mcp.Description("Object name"),
				),
				mcp.WithString("source",
					mcp.Required(),
					mcp.Description("ABAP source code"),
				),
				mcp.WithString("mode",
					mcp.Description("Operation mode: upsert (default, auto-detect), create (new only), update (existing only)"),
				),
				mcp.WithString("description",
					mcp.Description("Object description (required for create mode)"),
				),
				mcp.WithString("package",
					mcp.Description("Package name (required for create mode)"),
				),
				mcp.WithString("test_source",
					mcp.Description("Test source code for CLAS (auto-creates test include and runs tests)"),
				),
				mcp.WithString("connection",
					mcp.Description("Transport request number"),
				),
				mcp.WithString("method",
					mcp.Description("For CLAS only: update only this method (source must be METHOD...ENDMETHOD block). Method must already exist in the class."),
				),
			),
			Handler:   HandleWriteSource,
			Endpoints: []string{"/sap/bc/adt/programs/programs"},
			Routes: []types.UniversalRoute{
				{Action: "edit", MapArgs: func(ot, on string, p map[string]any) map[string]any {
					p["object_type"] = ot
					p["name"] = on
					return p
				}},
				{Action: "create", MapArgs: func(ot, on string, p map[string]any) map[string]any {
					p["object_type"] = ot
					p["name"] = on
					return p
				}},
			},
		},
	}
}

// HandleGetSource handles the unified GetSource tool call
func HandleGetSource(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.GetArguments()["object_type"].(string)
	if !ok || objectType == "" {
		return types.ErrorResult("object_type is required"), nil
	}

	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return types.ErrorResult("name is required"), nil
	}

	parent, _ := request.GetArguments()["parent"].(string)
	include, _ := request.GetArguments()["include"].(string)
	method, _ := request.GetArguments()["method"].(string)

	ref, err := resolveRef(objectType, name, parent, include, method)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Invalid object reference: %v", err)), nil
	}

	source, err := sys.ADT().GetSource(ctx, ref)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetSource failed: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

// HandleWriteSource handles the unified WriteSource tool call
func HandleWriteSource(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.GetArguments()["object_type"].(string)
	if !ok || objectType == "" {
		return types.ErrorResult("object_type is required"), nil
	}

	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return types.ErrorResult("name is required"), nil
	}

	source, ok := request.GetArguments()["source"].(string)
	if !ok || source == "" {
		return types.ErrorResult("source is required"), nil
	}

	mode, _ := request.GetArguments()["mode"].(string)
	description, _ := request.GetArguments()["description"].(string)
	packageName, _ := request.GetArguments()["package"].(string)
	testSource, _ := request.GetArguments()["test_source"].(string)
	transport, _ := request.GetArguments()["connection"].(string)
	method, _ := request.GetArguments()["method"].(string)

	ref, err := resolveRef(objectType, name, "", "", method)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Invalid object reference: %v", err)), nil
	}

	opts := &adt.WriteSourceOptions{
		Description: description,
		Package:     packageName,
		TestSource:  testSource,
		Transport:   transport,
	}

	if mode != "" {
		opts.Mode = adt.WriteSourceMode(mode)
	}

	result, err := sys.ADT().WriteSource(ctx, ref, source, opts)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("WriteSource failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
