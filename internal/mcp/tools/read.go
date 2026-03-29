package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// ReadToolDefs returns tool definitions for object read tools.
func ReadToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("GetProgram",
			mcp.WithDescription("Retrieve ABAP program source code"),
			mcp.WithString("program_name", mcp.Required(), mcp.Description("Name of the ABAP program")),
		), Handler: HandleGetProgram, ReadOnly: true, Routes: []types.UniversalRoute{
			{Action: "read", TargetType: "PROG", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				return map[string]any{"program_name": on}
			}},
		}},

		{Tool: mcp.NewTool("GetClass",
			mcp.WithDescription("Retrieve ABAP class source code"),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
		), Handler: HandleGetClass, ReadOnly: true, Routes: []types.UniversalRoute{
			{Action: "read", TargetType: "CLAS", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				return map[string]any{"class_name": on}
			}},
		}},

		{Tool: mcp.NewTool("GetInterface",
			mcp.WithDescription("Retrieve ABAP interface source code"),
			mcp.WithString("interface_name", mcp.Required(), mcp.Description("Name of the ABAP interface")),
		), Handler: HandleGetInterface, ReadOnly: true, Routes: []types.UniversalRoute{
			{Action: "read", TargetType: "INTF", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				return map[string]any{"interface_name": on}
			}},
		}},

		{Tool: mcp.NewTool("GetFunction",
			mcp.WithDescription("Retrieve ABAP Function Module source code"),
			mcp.WithString("function_name", mcp.Required(), mcp.Description("Name of the function module")),
			mcp.WithString("function_group", mcp.Required(), mcp.Description("Name of the function group")),
		), Handler: HandleGetFunction, ReadOnly: true, Routes: []types.UniversalRoute{
			{Action: "read", TargetType: "FUNC", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				return map[string]any{"function_name": on, "function_group": p["function_group"]}
			}},
		}},

		{Tool: mcp.NewTool("GetFunctionGroup",
			mcp.WithDescription("Retrieve ABAP Function Group source code"),
			mcp.WithString("function_group", mcp.Required(), mcp.Description("Name of the function group")),
		), Handler: HandleGetFunctionGroup, ReadOnly: true, Focused: true, Routes: []types.UniversalRoute{
			{Action: "read", TargetType: "FUGR", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				return map[string]any{"function_group": on}
			}},
		}},

		{Tool: mcp.NewTool("GetInclude",
			mcp.WithDescription("Retrieve ABAP Include Source Code"),
			mcp.WithString("include_name", mcp.Required(), mcp.Description("Name of the ABAP Include")),
		), Handler: HandleGetInclude, ReadOnly: true, Routes: []types.UniversalRoute{
			{Action: "read", TargetType: "INCL", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				return map[string]any{"include_name": on}
			}},
		}},

		{Tool: mcp.NewTool("GetTable",
			mcp.WithDescription("Retrieve ABAP table structure"),
			mcp.WithString("table_name", mcp.Required(), mcp.Description("Name of the ABAP table")),
		), Handler: HandleGetTable, ReadOnly: true, Focused: true, Routes: []types.UniversalRoute{
			{Action: "read", TargetType: "TABL", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				return map[string]any{"table_name": on}
			}},
		}},

		{Tool: mcp.NewTool("GetTableContents",
			mcp.WithDescription("Retrieve contents of an ABAP table. For simple queries use table_name + max_rows. For filtered queries use sql_query parameter with ABAP SQL syntax (use ASCENDING/DESCENDING, not ASC/DESC)."),
			mcp.WithString("table_name", mcp.Required(), mcp.Description("Name of the ABAP table")),
			mcp.WithNumber("max_rows", mcp.Description("Maximum number of rows to retrieve (default 100). Use this instead of SQL LIMIT clause")),
			mcp.WithString("sql_query", mcp.Description("Optional ABAP SQL SELECT statement. Uses ABAP syntax: ASCENDING/DESCENDING work, ASC/DESC fail. Example: SELECT * FROM T000 WHERE MANDT = '001' ORDER BY MANDT DESCENDING")),
		), Handler: HandleGetTableContents, ReadOnly: true, Focused: true, Routes: []types.UniversalRoute{
			{Action: "query", TargetType: "TABL", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				p["table_name"] = on
				return p
			}},
		}},

		{Tool: mcp.NewTool("RunQuery",
			mcp.WithDescription("Execute a freestyle SQL query against the SAP database. IMPORTANT: Uses ABAP SQL syntax, NOT standard SQL. Use ASCENDING/DESCENDING instead of ASC/DESC. Use max_rows parameter instead of LIMIT. GROUP BY and WHERE work normally."),
			mcp.WithString("sql_query", mcp.Required(), mcp.Description("ABAP SQL query. Example: SELECT carrid, COUNT(*) as cnt FROM sflight GROUP BY carrid ORDER BY cnt DESCENDING. Note: ASC/DESC keywords fail - use ASCENDING/DESCENDING")),
			mcp.WithNumber("max_rows", mcp.Description("Maximum number of rows to retrieve (default 100). Use this instead of SQL LIMIT clause")),
		), Handler: HandleRunQuery, ReadOnly: true, Focused: true, Routes: []types.UniversalRoute{
			{Action: "query", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				return p
			}},
		}},

		{Tool: mcp.NewTool("GetCDSDependencies",
			mcp.WithDescription("Retrieve CDS view FORWARD dependencies (tables/views this CDS reads FROM). Returns tree of base objects. Does NOT return reverse dependencies (where-used). Use with GetSource(DDLS) to read CDS source code."),
			mcp.WithString("ddls_name", mcp.Required(), mcp.Description("CDS DDL source name (e.g., 'ZRAY_00_I_DOC_NODE_00'). Use SearchObject to find CDS views first.")),
			mcp.WithString("dependency_level", mcp.Description("Level of dependency resolution: 'unit' (direct only) or 'hierarchy' (recursive). Default: 'hierarchy'")),
			mcp.WithBoolean("with_associations", mcp.Description("Include modeled associations in dependency tree. Default: false")),
			mcp.WithString("context_package", mcp.Description("Filter dependencies to specific package context")),
		), Handler: HandleGetCDSDependencies, ReadOnly: true, Focused: true, Routes: []types.UniversalRoute{
			{Action: "analyze", TargetType: "DDLS", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				p["ddls_name"] = on
				return p
			}},
		}},

		{Tool: mcp.NewTool("GetStructure",
			mcp.WithDescription("Retrieve ABAP Structure"),
			mcp.WithString("structure_name", mcp.Required(), mcp.Description("Name of the ABAP Structure")),
		), Handler: HandleGetStructure, ReadOnly: true, Routes: []types.UniversalRoute{
			{Action: "read", TargetType: "STRU", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				return map[string]any{"structure_name": on}
			}},
		}},

		{Tool: mcp.NewTool("GetPackage",
			mcp.WithDescription("Retrieve ABAP package details"),
			mcp.WithString("package_name", mcp.Required(), mcp.Description("Name of the ABAP package")),
		), Handler: HandleGetPackage, ReadOnly: true, Focused: true, Routes: []types.UniversalRoute{
			{Action: "read", TargetType: "DEVC", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				return map[string]any{"package_name": on}
			}},
		}},

		{Tool: mcp.NewTool("GetMessages",
			mcp.WithDescription("Get all messages from an ABAP message class (SE91). Returns message number, text for all messages in the class. Use SearchObject to find message classes first."),
			mcp.WithString("message_class", mcp.Required(), mcp.Description("Name of the message class (e.g., 'ZRAY_00', 'SY')")),
		), Handler: HandleGetMessages, ReadOnly: true, Focused: true, Routes: []types.UniversalRoute{
			{Action: "read", TargetType: "MSAG", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				return map[string]any{"message_class": on}
			}},
		}},

		{Tool: mcp.NewTool("GetTransaction",
			mcp.WithDescription("Retrieve ABAP transaction details"),
			mcp.WithString("transaction_name", mcp.Required(), mcp.Description("Name of the ABAP transaction")),
		), Handler: HandleGetTransaction, ReadOnly: true, Routes: []types.UniversalRoute{
			{Action: "read", TargetType: "TRAN", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				return map[string]any{"transaction_name": on}
			}},
		}},

		{Tool: mcp.NewTool("GetTypeInfo",
			mcp.WithDescription("Retrieve ABAP type information"),
			mcp.WithString("type_name", mcp.Required(), mcp.Description("Name of the ABAP type")),
		), Handler: HandleGetTypeInfo, ReadOnly: true, Routes: []types.UniversalRoute{
			{Action: "read", TargetType: "TYPE", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				return map[string]any{"type_name": on}
			}},
		}},
	}
}

// --- Read Handlers ---

func HandleGetProgram(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	programName, ok := request.GetArguments()["program_name"].(string)
	if !ok || programName == "" {
		return types.ErrorResult("program_name is required"), nil
	}

	source, err := sys.ADT().GetProgram(ctx, programName)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get program: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func HandleGetClass(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.GetArguments()["class_name"].(string)
	if !ok || className == "" {
		return types.ErrorResult("class_name is required"), nil
	}

	source, err := sys.ADT().GetClassSource(ctx, className)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get class: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func HandleGetInterface(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	interfaceName, ok := request.GetArguments()["interface_name"].(string)
	if !ok || interfaceName == "" {
		return types.ErrorResult("interface_name is required"), nil
	}

	source, err := sys.ADT().GetInterface(ctx, interfaceName)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get interface: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func HandleGetFunction(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	functionName, ok := request.GetArguments()["function_name"].(string)
	if !ok || functionName == "" {
		return types.ErrorResult("function_name is required"), nil
	}

	functionGroup, ok := request.GetArguments()["function_group"].(string)
	if !ok || functionGroup == "" {
		return types.ErrorResult("function_group is required"), nil
	}

	source, err := sys.ADT().GetFunction(ctx, functionName, functionGroup)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get function: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func HandleGetFunctionGroup(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	groupName, ok := request.GetArguments()["function_group"].(string)
	if !ok || groupName == "" {
		return types.ErrorResult("function_group is required"), nil
	}

	fg, err := sys.ADT().GetFunctionGroup(ctx, groupName)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get function group: %v", err)), nil
	}

	result, _ := json.MarshalIndent(fg, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleGetInclude(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	includeName, ok := request.GetArguments()["include_name"].(string)
	if !ok || includeName == "" {
		return types.ErrorResult("include_name is required"), nil
	}

	source, err := sys.ADT().GetInclude(ctx, includeName)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get include: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func HandleGetTable(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tableName, ok := request.GetArguments()["table_name"].(string)
	if !ok || tableName == "" {
		return types.ErrorResult("table_name is required"), nil
	}

	source, err := sys.ADT().GetTable(ctx, tableName)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get table: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func HandleGetTableContents(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tableName, ok := request.GetArguments()["table_name"].(string)
	if !ok || tableName == "" {
		return types.ErrorResult("table_name is required"), nil
	}

	maxRows := 100
	if mr, ok := request.GetArguments()["max_rows"].(float64); ok && mr > 0 {
		maxRows = int(mr)
	}

	sqlQuery := ""
	if sq, ok := request.GetArguments()["sql_query"].(string); ok {
		sqlQuery = sq
	}

	contents, err := sys.ADT().GetTableContents(ctx, tableName, maxRows, sqlQuery)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get table contents: %v", err)), nil
	}

	result, _ := json.MarshalIndent(contents, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleRunQuery(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sqlQuery, ok := request.GetArguments()["sql_query"].(string)
	if !ok || sqlQuery == "" {
		return types.ErrorResult("sql_query is required"), nil
	}

	maxRows := 100
	if mr, ok := request.GetArguments()["max_rows"].(float64); ok && mr > 0 {
		maxRows = int(mr)
	}

	contents, err := sys.ADT().RunQuery(ctx, sqlQuery, maxRows)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to run query: %v", err)), nil
	}

	result, _ := json.MarshalIndent(contents, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleGetCDSDependencies(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ddlsName, ok := request.GetArguments()["ddls_name"].(string)
	if !ok || ddlsName == "" {
		return types.ErrorResult("ddls_name is required"), nil
	}

	opts := adt.CDSDependencyOptions{
		DependencyLevel:  "hierarchy",
		WithAssociations: false,
	}

	if level, ok := request.GetArguments()["dependency_level"].(string); ok && level != "" {
		opts.DependencyLevel = level
	}

	if assoc, ok := request.GetArguments()["with_associations"].(bool); ok {
		opts.WithAssociations = assoc
	}

	if pkg, ok := request.GetArguments()["context_package"].(string); ok && pkg != "" {
		opts.ContextPackage = pkg
	}

	dependencyTree, err := sys.ADT().GetCDSDependencies(ctx, ddlsName, opts)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get CDS dependencies: %v", err)), nil
	}

	// Add metadata summary
	summary := map[string]interface{}{
		"ddls_name":       ddlsName,
		"dependency_tree": dependencyTree,
		"statistics": map[string]interface{}{
			"total_dependencies":    len(dependencyTree.FlattenDependencies()) - 1, // -1 to exclude root
			"dependency_depth":      dependencyTree.GetDependencyDepth(),
			"by_type":               dependencyTree.CountDependenciesByType(),
			"table_dependencies":    len(dependencyTree.GetTableDependencies()),
			"inactive_dependencies": len(dependencyTree.GetInactiveDependencies()),
			"cycles":                dependencyTree.FindCycles(),
		},
	}

	jsonResult, _ := json.MarshalIndent(summary, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func HandleGetStructure(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	structName, ok := request.GetArguments()["structure_name"].(string)
	if !ok || structName == "" {
		return types.ErrorResult("structure_name is required"), nil
	}

	source, err := sys.ADT().GetStructure(ctx, structName)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get structure: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func HandleGetPackage(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	packageName, ok := request.GetArguments()["package_name"].(string)
	if !ok || packageName == "" {
		return types.ErrorResult("package_name is required"), nil
	}

	pkg, err := sys.ADT().GetPackage(ctx, packageName)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get package: %v", err)), nil
	}

	result, _ := json.MarshalIndent(pkg, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleGetMessages(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	msgClass, ok := request.GetArguments()["message_class"].(string)
	if !ok || msgClass == "" {
		return types.ErrorResult("message_class is required"), nil
	}

	mc, err := sys.ADT().GetMessageClass(ctx, msgClass)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get message class: %v", err)), nil
	}

	result, _ := json.MarshalIndent(mc, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleGetTransaction(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tcode, ok := request.GetArguments()["transaction_name"].(string)
	if !ok || tcode == "" {
		return types.ErrorResult("transaction_name is required"), nil
	}

	tran, err := sys.ADT().GetTransaction(ctx, tcode)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get transaction: %v", err)), nil
	}

	result, _ := json.MarshalIndent(tran, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func HandleGetTypeInfo(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	typeName, ok := request.GetArguments()["type_name"].(string)
	if !ok || typeName == "" {
		return types.ErrorResult("type_name is required"), nil
	}

	typeInfo, err := sys.ADT().GetTypeInfo(ctx, typeName)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to get type info: %v", err)), nil
	}

	result, _ := json.MarshalIndent(typeInfo, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}
