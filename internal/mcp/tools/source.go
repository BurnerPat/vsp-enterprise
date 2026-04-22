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
					mcp.Description("Object type: PROG (program), CLAS (class), INTF (interface), FUNC (function module), FUGR (function group), INCL (include), DDLS (CDS DDL source), VIEW (DDIC view), BDEF (behavior definition), SRVD (service definition), SRVB (service binding), MSAG (message class)"),
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
					mcp.Description("Object type: PROG (program), CLAS (class), INTF (interface), DDLS (CDS view), BDEF (behavior definition), SRVD (service definition)"),
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

// GrepSourceToolDefs returns tool definitions for grep tools.
func GrepSourceToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{
			ReadOnly: true, Focused: true,
			Tool: mcp.NewTool("GrepObjects",
				mcp.WithDescription("Unified tool for searching regex patterns in single or multiple ABAP objects. Replaces GrepObject."),
				mcp.WithArray("object_urls",
					mcp.Required(),
					mcp.Description("Array of ADT object URLs to search (e.g., [\"/sap/bc/adt/programs/programs/ZTEST\"])"),
					mcp.Items(map[string]interface{}{"type": "string"}),
				),
				mcp.WithString("pattern",
					mcp.Required(),
					mcp.Description("Regular expression pattern (Go regexp syntax)"),
				),
				mcp.WithBoolean("case_insensitive",
					mcp.Description("If true, perform case-insensitive matching (default: false)"),
				),
				mcp.WithNumber("context_lines",
					mcp.Description("Number of context lines before/after each match (default: 0)"),
				),
			),
			Handler: HandleGrepObjects,
		},
		{
			ReadOnly: true, Focused: true,
			Tool: mcp.NewTool("GrepPackages",
				mcp.WithDescription("Unified tool for searching regex patterns across single or multiple packages with optional recursive subpackage search. Replaces GrepPackage."),
				mcp.WithArray("packages",
					mcp.Required(),
					mcp.Description("Array of package names to search (e.g., [\"$TMP\"], [\"Z\"] for namespace search)"),
					mcp.Items(map[string]interface{}{"type": "string"}),
				),
				mcp.WithBoolean("include_subpackages",
					mcp.Description("If true, recursively search all subpackages (default: false). Enables namespace-wide searches."),
				),
				mcp.WithString("pattern",
					mcp.Required(),
					mcp.Description("Regular expression pattern (Go regexp syntax)"),
				),
				mcp.WithBoolean("case_insensitive",
					mcp.Description("If true, perform case-insensitive matching (default: false)"),
				),
				mcp.WithArray("object_types",
					mcp.Description("Filter by object types (e.g., [\"CLAS/OC\", \"PROG/P\"]). Empty = search all types."),
					mcp.Items(map[string]interface{}{"type": "string"}),
				),
				mcp.WithNumber("max_results",
					mcp.Description("Maximum number of matching objects to return (0 = unlimited, default: 0)"),
				),
			),
			Handler: HandleGrepPackages,
		},
	}
}

// FileSourceToolDefs returns tool definitions for file import/export tools.
func FileSourceToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{
			Focused: true,
			Tool: mcp.NewTool("ImportFromFile",
				mcp.WithDescription("Import ABAP object from local file into SAP system. Auto-detects object type from file extension, creates or updates, activates. Supports: programs, classes (with includes), interfaces, function groups/modules, CDS views (DDLS), behavior definitions (BDEF), service definitions (SRVD). For class includes (.clas.testclasses.abap, .clas.locals_def.abap, etc.), the parent class must exist."),
				mcp.WithString("file_path",
					mcp.Required(),
					mcp.Description("Absolute path to ABAP source file. Supported extensions: .prog.abap, .clas.abap, .clas.testclasses.abap, .clas.locals_def.abap, .clas.locals_imp.abap, .intf.abap, .fugr.abap, .func.abap, .ddls.asddls, .bdef.asbdef, .srvd.srvdsrv"),
				),
				mcp.WithString("package_name",
					mcp.Description("Target package name (required for new objects, not needed for class includes)"),
				),
				mcp.WithString("connection",
					mcp.Description("Transport request number"),
				),
			),
			Handler: HandleImportFromFile,
		},
		{
			ReadOnly: true, Focused: true,
			Tool: mcp.NewTool("ExportToFile",
				mcp.WithDescription("Export ABAP object from SAP system to local file. Saves source code with appropriate file extension. Supports: programs, classes (with includes), interfaces, function groups/modules, CDS views (DDLS), behavior definitions (BDEF), service definitions (SRVD). For classes, use 'include' parameter to export specific includes (testclasses, definitions, implementations, macros)."),
				mcp.WithString("object_type",
					mcp.Required(),
					mcp.Description("Object type: PROG, CLAS, INTF, FUGR, FUNC, DDLS, BDEF, SRVD"),
				),
				mcp.WithString("object_name",
					mcp.Required(),
					mcp.Description("Object name"),
				),
				mcp.WithString("output_dir",
					mcp.Required(),
					mcp.Description("Output directory path (must exist)"),
				),
				mcp.WithString("include",
					mcp.Description("For CLAS only: include type to export. Values: main (default), testclasses, definitions, implementations, macros. Creates abapGit-compatible files (.clas.testclasses.abap, .clas.locals_def.abap, etc.)"),
				),
				mcp.WithString("parent",
					mcp.Description("Function group name (required for FUNC type)"),
				),
			),
			Handler: HandleExportToFile,
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

	opts := &adt.GetSourceOptions{
		Parent:  parent,
		Include: include,
		Method:  method,
	}

	source, err := sys.ADT().GetSource(ctx, objectType, name, opts)
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

	opts := &adt.WriteSourceOptions{
		Description: description,
		Package:     packageName,
		TestSource:  testSource,
		Transport:   transport,
		Method:      method,
	}

	if mode != "" {
		opts.Mode = adt.WriteSourceMode(mode)
	}

	result, err := sys.ADT().WriteSource(ctx, objectType, name, source, opts)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("WriteSource failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// HandleGrepObjects handles the unified GrepObjects tool call
func HandleGrepObjects(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("Grep results (placeholder)"), nil
}

// HandleGrepPackages handles the unified GrepPackages tool call
func HandleGrepPackages(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("GrepPackages (placeholder)"), nil
}

// HandleImportFromFile handles the ImportFromFile tool call
func HandleImportFromFile(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("ImportFromFile (placeholder)"), nil
}

// HandleExportToFile handles the ExportToFile tool call
func HandleExportToFile(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("ExportToFile (placeholder)"), nil
}
