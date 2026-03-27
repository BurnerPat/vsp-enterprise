// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tools_defs.go contains tool definition methods that return []toolDef slices.
// These are consumed by allToolDefs() in tools_register.go.
//
// Formatting: blank lines separate individual toolDef entries for readability.
package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// readToolDefs returns tool definitions for object read tools.
func (s *Server) readToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("GetProgram",
			mcp.WithDescription("Retrieve ABAP program source code"),
			mcp.WithString("program_name", mcp.Required(), mcp.Description("Name of the ABAP program")),
		), handler: s.handleGetProgram},

		{tool: mcp.NewTool("GetClass",
			mcp.WithDescription("Retrieve ABAP class source code"),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
		), handler: s.handleGetClass},

		{tool: mcp.NewTool("GetInterface",
			mcp.WithDescription("Retrieve ABAP interface source code"),
			mcp.WithString("interface_name", mcp.Required(), mcp.Description("Name of the ABAP interface")),
		), handler: s.handleGetInterface},

		{tool: mcp.NewTool("GetFunction",
			mcp.WithDescription("Retrieve ABAP Function Module source code"),
			mcp.WithString("function_name", mcp.Required(), mcp.Description("Name of the function module")),
			mcp.WithString("function_group", mcp.Required(), mcp.Description("Name of the function group")),
		), handler: s.handleGetFunction},

		{tool: mcp.NewTool("GetFunctionGroup",
			mcp.WithDescription("Retrieve ABAP Function Group source code"),
			mcp.WithString("function_group", mcp.Required(), mcp.Description("Name of the function group")),
		), handler: s.handleGetFunctionGroup},

		{tool: mcp.NewTool("GetInclude",
			mcp.WithDescription("Retrieve ABAP Include Source Code"),
			mcp.WithString("include_name", mcp.Required(), mcp.Description("Name of the ABAP Include")),
		), handler: s.handleGetInclude},

		{tool: mcp.NewTool("GetTable",
			mcp.WithDescription("Retrieve ABAP table structure"),
			mcp.WithString("table_name", mcp.Required(), mcp.Description("Name of the ABAP table")),
		), handler: s.handleGetTable},

		{tool: mcp.NewTool("GetTableContents",
			mcp.WithDescription("Retrieve contents of an ABAP table. For simple queries use table_name + max_rows. For filtered queries use sql_query parameter with ABAP SQL syntax (use ASCENDING/DESCENDING, not ASC/DESC)."),
			mcp.WithString("table_name", mcp.Required(), mcp.Description("Name of the ABAP table")),
			mcp.WithNumber("max_rows", mcp.Description("Maximum number of rows to retrieve (default 100). Use this instead of SQL LIMIT clause")),
			mcp.WithString("sql_query", mcp.Description("Optional ABAP SQL SELECT statement. Uses ABAP syntax: ASCENDING/DESCENDING work, ASC/DESC fail. Example: SELECT * FROM T000 WHERE MANDT = '001' ORDER BY MANDT DESCENDING")),
		), handler: s.handleGetTableContents},

		{tool: mcp.NewTool("RunQuery",
			mcp.WithDescription("Execute a freestyle SQL query against the SAP database. IMPORTANT: Uses ABAP SQL syntax, NOT standard SQL. Use ASCENDING/DESCENDING instead of ASC/DESC. Use max_rows parameter instead of LIMIT. GROUP BY and WHERE work normally."),
			mcp.WithString("sql_query", mcp.Required(), mcp.Description("ABAP SQL query. Example: SELECT carrid, COUNT(*) as cnt FROM sflight GROUP BY carrid ORDER BY cnt DESCENDING. Note: ASC/DESC keywords fail - use ASCENDING/DESCENDING")),
			mcp.WithNumber("max_rows", mcp.Description("Maximum number of rows to retrieve (default 100). Use this instead of SQL LIMIT clause")),
		), handler: s.handleRunQuery},

		{tool: mcp.NewTool("GetCDSDependencies",
			mcp.WithDescription("Retrieve CDS view FORWARD dependencies (tables/views this CDS reads FROM). Returns tree of base objects. Does NOT return reverse dependencies (where-used). Use with GetSource(DDLS) to read CDS source code."),
			mcp.WithString("ddls_name", mcp.Required(), mcp.Description("CDS DDL source name (e.g., 'ZRAY_00_I_DOC_NODE_00'). Use SearchObject to find CDS views first.")),
			mcp.WithString("dependency_level", mcp.Description("Level of dependency resolution: 'unit' (direct only) or 'hierarchy' (recursive). Default: 'hierarchy'")),
			mcp.WithBoolean("with_associations", mcp.Description("Include modeled associations in dependency tree. Default: false")),
			mcp.WithString("context_package", mcp.Description("Filter dependencies to specific package context")),
		), handler: s.handleGetCDSDependencies},

		{tool: mcp.NewTool("GetStructure",
			mcp.WithDescription("Retrieve ABAP Structure"),
			mcp.WithString("structure_name", mcp.Required(), mcp.Description("Name of the ABAP Structure")),
		), handler: s.handleGetStructure},

		{tool: mcp.NewTool("GetPackage",
			mcp.WithDescription("Retrieve ABAP package details"),
			mcp.WithString("package_name", mcp.Required(), mcp.Description("Name of the ABAP package")),
		), handler: s.handleGetPackage},

		{tool: mcp.NewTool("GetMessages",
			mcp.WithDescription("Get all messages from an ABAP message class (SE91). Returns message number, text for all messages in the class. Use SearchObject to find message classes first."),
			mcp.WithString("message_class", mcp.Required(), mcp.Description("Name of the message class (e.g., 'ZRAY_00', 'SY')")),
		), handler: s.handleGetMessages},

		{tool: mcp.NewTool("GetTransaction",
			mcp.WithDescription("Retrieve ABAP transaction details"),
			mcp.WithString("transaction_name", mcp.Required(), mcp.Description("Name of the ABAP transaction")),
		), handler: s.handleGetTransaction},

		{tool: mcp.NewTool("GetTypeInfo",
			mcp.WithDescription("Retrieve ABAP type information"),
			mcp.WithString("type_name", mcp.Required(), mcp.Description("Name of the ABAP type")),
		), handler: s.handleGetTypeInfo},
	}
}

// systemToolDefs returns tool definitions for system information tools.
// All system tools are marked alwaysOn â€” they cannot be disabled via mode, groups, or config.
func (s *Server) systemToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("GetSystemInfo",
			mcp.WithDescription("Get SAP system information (system ID, release, kernel, database)"),
		), handler: s.handleGetSystemInfo, alwaysOn: true},

		{tool: mcp.NewTool("GetInstalledComponents",
			mcp.WithDescription("List installed software components with version information"),
		), handler: s.handleGetInstalledComponents, alwaysOn: true},

		{tool: mcp.NewTool("GetConnectionInfo",
			mcp.WithDescription("Get current MCP connection info: user, URL, client. Useful for debugging and understanding current session context."),
		), handler: s.handleGetConnectionInfo, alwaysOn: true},

		{tool: mcp.NewTool("GetFeatures",
			mcp.WithDescription("Probe SAP system for available features. Returns status of optional capabilities like abapGit, RAP/OData, AMDP debugging, UI5/BSP, and CTS transports. Use this to understand what features are available before attempting to use them."),
		), handler: s.handleGetFeatures, alwaysOn: true},

		{tool: mcp.NewTool("GetAbapHelp",
			mcp.WithDescription("Get ABAP keyword documentation. Returns URL to SAP Help Portal and search query. If ZADT_VSP is installed, also returns real documentation from SAP system."),
			mcp.WithString("keyword", mcp.Required(), mcp.Description("ABAP keyword (e.g., SELECT, LOOP, DATA, METHOD, READ TABLE)")),
		), handler: s.handleGetAbapHelp, alwaysOn: true},
	}
}

// analysisToolDefs returns tool definitions for code analysis tools.
func (s *Server) analysisToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("GetCallGraph",
			mcp.WithDescription("Get call hierarchy for methods/functions. Shows callers or callees of an ABAP object."),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST/source/main#start=10,1)")),
			mcp.WithString("direction", mcp.Description("Direction: 'callers' (who calls this) or 'callees' (what this calls). Default: callers")),
			mcp.WithNumber("max_depth", mcp.Description("Maximum depth of call hierarchy (default: 3)")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of results (default: 100)")),
		), handler: s.handleGetCallGraph},

		{tool: mcp.NewTool("GetObjectStructure",
			mcp.WithDescription("Get object explorer tree structure. Returns hierarchical view of object components."),
			mcp.WithString("object_name", mcp.Required(), mcp.Description("Object name (e.g., ZCL_TEST, ZPROGRAM)")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of results (default: 100)")),
		), handler: s.handleGetObjectStructure},

		{tool: mcp.NewTool("GetCallersOf",
			mcp.WithDescription("Find all callers of an ABAP object (up traversal). Shows who calls this method/function. Simplified wrapper around GetCallGraph."),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST/source/main#start=10,1)")),
			mcp.WithNumber("max_depth", mcp.Description("Maximum depth of caller hierarchy (default: 5)")),
		), handler: s.handleGetCallersOf},

		{tool: mcp.NewTool("GetCalleesOf",
			mcp.WithDescription("Find all callees of an ABAP object (down traversal). Shows what this method/function calls. Simplified wrapper around GetCallGraph."),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST/source/main#start=10,1)")),
			mcp.WithNumber("max_depth", mcp.Description("Maximum depth of callee hierarchy (default: 5)")),
		), handler: s.handleGetCalleesOf},

		{tool: mcp.NewTool("AnalyzeCallGraph",
			mcp.WithDescription("Analyze call graph for an object. Returns statistics: total nodes, edges, max depth, nodes by type. Use for understanding code complexity and dependencies."),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the object to analyze")),
			mcp.WithString("direction", mcp.Description("Direction: 'callers' or 'callees' (default: callees)")),
			mcp.WithNumber("max_depth", mcp.Description("Maximum depth to analyze (default: 5)")),
		), handler: s.handleAnalyzeCallGraph},

		{tool: mcp.NewTool("CompareCallGraphs",
			mcp.WithDescription("Compare static call graph with actual execution trace. Identifies: common paths, untested paths (static only), and dynamic calls (actual only). Use for test coverage analysis and RCA."),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the root object")),
			mcp.WithString("trace_data", mcp.Required(), mcp.Description("JSON array of trace edges from execution (format: [{caller_name, callee_name}, ...])")),
		), handler: s.handleCompareCallGraphs},

		{tool: mcp.NewTool("TraceExecution",
			mcp.WithDescription("COMPOSITE RCA TOOL: Performs traced execution analysis. 1) Builds static call graph from object, 2) Optionally runs unit tests, 3) Collects trace data, 4) Extracts actual call edges, 5) Compares static vs actual for root cause analysis."),
			mcp.WithString("object_uri", mcp.Required(), mcp.Description("ADT URI of the starting object for static call graph")),
			mcp.WithNumber("max_depth", mcp.Description("Maximum depth for call graph traversal (default: 5)")),
			mcp.WithBoolean("run_tests", mcp.Description("Run unit tests before collecting trace (default: false)")),
			mcp.WithString("test_object_uri", mcp.Description("Object URI for tests to run (defaults to object_uri)")),
			mcp.WithString("trace_user", mcp.Description("Filter traces by user (defaults to current user)")),
		), handler: s.handleTraceExecution},
	}
}

// diagnosticsToolDefs returns tool definitions for runtime error, profiler, and SQL trace tools.
func (s *Server) diagnosticsToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("ListDumps",
			mcp.WithDescription("List runtime errors (short dumps) from the SAP system. Filter by user, exception type, program, date range."),
			mcp.WithString("user", mcp.Description("Filter by username")),
			mcp.WithString("exception_type", mcp.Description("Filter by exception type (e.g., CX_SY_ZERODIVIDE)")),
			mcp.WithString("program", mcp.Description("Filter by program name")),
			mcp.WithString("package", mcp.Description("Filter by package")),
			mcp.WithString("date_from", mcp.Description("Start date (YYYYMMDD format)")),
			mcp.WithString("date_to", mcp.Description("End date (YYYYMMDD format)")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of results (default: 100)")),
		), handler: s.handleListDumps},

		{tool: mcp.NewTool("GetDump",
			mcp.WithDescription("Get full details of a specific runtime error (short dump) including stack trace."),
			mcp.WithString("dump_id", mcp.Required(), mcp.Description("Dump ID from ListDumps result")),
		), handler: s.handleGetDump},

		{tool: mcp.NewTool("ListTraces",
			mcp.WithDescription("List ABAP runtime traces (profiler results) from the SAP system."),
			mcp.WithString("user", mcp.Description("Filter by username")),
			mcp.WithString("process_type", mcp.Description("Filter by process type")),
			mcp.WithString("object_type", mcp.Description("Filter by object type")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of results (default: 100)")),
		), handler: s.handleListTraces},

		{tool: mcp.NewTool("GetTrace",
			mcp.WithDescription("Get trace analysis (hitlist, statements, or database accesses) for a specific trace."),
			mcp.WithString("trace_id", mcp.Required(), mcp.Description("Trace ID from ListTraces result")),
			mcp.WithString("tool_type", mcp.Description("Analysis type: 'hitlist' (default), 'statements', 'dbAccesses'")),
		), handler: s.handleGetTrace},

		{tool: mcp.NewTool("GetSQLTraceState",
			mcp.WithDescription("Check if SQL trace (ST05) is currently active."),
		), handler: s.handleGetSQLTraceState},

		{tool: mcp.NewTool("ListSQLTraces",
			mcp.WithDescription("List SQL trace files from ST05."),
			mcp.WithString("user", mcp.Description("Filter by username")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of results (default: 100)")),
		), handler: s.handleListSQLTraces},
	}
}

// debuggerToolDefs returns tool definitions for ABAP debugger tools.
func (s *Server) debuggerToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("SetBreakpoint",
			mcp.WithDescription("Set a breakpoint in ABAP code. Supports three types: 'line' (specific location), 'statement' (ABAP keyword), 'exception' (exception class). For class methods, use 'method' parameter for include-relative line numbers. Uses WebSocket connection to ZADT_VSP."),
			mcp.WithString("kind", mcp.Description("Breakpoint type: 'line' (default), 'statement', or 'exception'")),
			mcp.WithString("program", mcp.Description("Program name for line breakpoints (e.g., 'ZADT_DBG_PROG' or 'ZCL_MY_CLASS')")),
			mcp.WithString("method", mcp.Description("Method name for class breakpoints. When specified, line number is relative to method start (line 1 = first line of METHOD implementation). Enables accurate breakpoints in class methods.")),
			mcp.WithNumber("line", mcp.Description("Line number for line breakpoints. Without 'method': pool-absolute line. With 'method': relative to method start.")),
			mcp.WithString("statement", mcp.Description("ABAP statement for statement breakpoints (e.g., 'CALL FUNCTION', 'SELECT', 'LOOP', 'CALL METHOD')")),
			mcp.WithString("exception", mcp.Description("Exception class for exception breakpoints (e.g., 'CX_SY_ZERODIVIDE', 'CX_SY_OPEN_SQL_DB')")),
		), handler: s.handleSetBreakpoint},

		{tool: mcp.NewTool("GetBreakpoints",
			mcp.WithDescription("Get all breakpoints registered in the current debug session. Uses WebSocket connection to ZADT_VSP."),
		), handler: s.handleGetBreakpoints},

		{tool: mcp.NewTool("DeleteBreakpoint",
			mcp.WithDescription("Delete a breakpoint by ID. Uses WebSocket connection to ZADT_VSP."),
			mcp.WithString("breakpoint_id", mcp.Required(), mcp.Description("ID of the breakpoint to delete")),
		), handler: s.handleDeleteBreakpoint},

		{tool: mcp.NewTool("CallRFC",
			mcp.WithDescription("Call a function module via WebSocket (ZADT_VSP). Useful for triggering ABAP code execution to hit breakpoints. Parameters are passed as key-value pairs."),
			mcp.WithString("function", mcp.Required(), mcp.Description("Function module name (e.g., 'RFC_PING', 'BAPI_USER_GET_DETAIL')")),
			mcp.WithString("params", mcp.Description("JSON object with function parameters (e.g., '{\"IV_PARAM\":\"value\"}')")),
		), handler: s.handleCallRFC},

		{tool: mcp.NewTool("MoveObject",
			mcp.WithDescription("Move an ABAP object to a different package. Uses ZADT_VSP WebSocket to call TR_TADIR_INTERFACE. Requires ZADT_VSP deployed."),
			mcp.WithString("object_type", mcp.Required(), mcp.Description("Object type: CLAS, PROG, INTF, FUGR, TABL, etc.")),
			mcp.WithString("object_name", mcp.Required(), mcp.Description("Name of the object to move (e.g., 'ZCL_TEST')")),
			mcp.WithString("new_package", mcp.Required(), mcp.Description("Target package (e.g., '$ZRAY', 'ZPACKAGE')")),
		), handler: s.handleMoveObject},

		{tool: mcp.NewTool("DebuggerListen",
			mcp.WithDescription("Start a debug listener that waits for a debuggee to hit a breakpoint. This is a BLOCKING call that uses long-polling. Returns when a debuggee is caught, timeout occurs, or a conflict is detected."),
			mcp.WithString("user", mcp.Description("User to listen for (defaults to current user)")),
			mcp.WithNumber("timeout", mcp.Description("Timeout in seconds (default: 60, max: 240)")),
		), handler: s.handleDebuggerListen},

		{tool: mcp.NewTool("DebuggerAttach",
			mcp.WithDescription("Attach to a debuggee that has hit a breakpoint. Use the debuggee_id from DebuggerListen result."),
			mcp.WithString("debuggee_id", mcp.Required(), mcp.Description("ID of the debuggee (from DebuggerListen result)")),
			mcp.WithString("user", mcp.Description("User for debugging (defaults to current user)")),
		), handler: s.handleDebuggerAttach},

		{tool: mcp.NewTool("DebuggerDetach",
			mcp.WithDescription("Detach from the current debug session and release the debuggee."),
		), handler: s.handleDebuggerDetach},

		{tool: mcp.NewTool("DebuggerStep",
			mcp.WithDescription("Perform a step operation in the debugger."),
			mcp.WithString("step_type", mcp.Required(), mcp.Description("Step type: 'stepInto', 'stepOver', 'stepReturn', 'stepContinue', 'stepRunToLine', 'stepJumpToLine'")),
			mcp.WithString("uri", mcp.Description("Target URI for stepRunToLine/stepJumpToLine (e.g., '/sap/bc/adt/programs/programs/ZTEST/source/main#start=42')")),
		), handler: s.handleDebuggerStep},

		{tool: mcp.NewTool("DebuggerGetStack",
			mcp.WithDescription("Get the current call stack during a debug session."),
		), handler: s.handleDebuggerGetStack},

		{tool: mcp.NewTool("DebuggerGetVariables",
			mcp.WithDescription("Get variable values during a debug session. Use '@ROOT' to get top-level variables, or specific variable IDs to get their values."),
			mcp.WithArray("variable_ids",
				mcp.Description("Variable IDs to retrieve (e.g., ['@ROOT'] for top-level, or specific IDs like ['LV_COUNT', 'LS_DATA'])"),
				mcp.Items(map[string]interface{}{"type": "string"}),
			),
		), handler: s.handleDebuggerGetVariables},

		{tool: mcp.NewTool("DebuggerSetVariable",
			mcp.WithDescription("Set a variable value during a debug session. Requires an active debug session (after DebuggerAttach)."),
			mcp.WithString("variable_name", mcp.Required(), mcp.Description("Variable name (e.g., 'GV_SPEED', 'LV_COUNT', 'LS_DATA-FIELD')")),
			mcp.WithString("value", mcp.Required(), mcp.Description("New value as string (e.g., '70', 'Hello', 'X')")),
		), handler: s.handleDebuggerSetVariable},
	}
}

// searchToolDefs returns tool definitions for object search tools.
func (s *Server) searchToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("SearchObject",
			mcp.WithDescription("Search for ABAP objects using quick search"),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query string (use * wildcard for partial match)")),
			mcp.WithNumber("maxResults", mcp.Description("Maximum number of results to return (default 100)")),
		), handler: s.handleSearchObject},
	}
}

// devToolDefs returns tool definitions for development tools.
func (s *Server) devToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("SyntaxCheck",
			mcp.WithDescription("Check ABAP source code for syntax errors"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("content", mcp.Required(), mcp.Description("ABAP source code to check")),
		), handler: s.handleSyntaxCheck},

		{tool: mcp.NewTool("Activate",
			mcp.WithDescription("Activate an ABAP object"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("object_name", mcp.Required(), mcp.Description("Technical name of the object (e.g., ZTEST)")),
		), handler: s.handleActivate},

		{tool: mcp.NewTool("ActivatePackage",
			mcp.WithDescription("Activate all inactive objects. Objects are sorted by dependency order (interfaces before classes). If no package specified, activates ALL inactive objects for current user."),
			mcp.WithString("package", mcp.Description("Package name to filter (optional, empty = all packages)")),
			mcp.WithNumber("max_objects", mcp.Description("Maximum number of objects to activate (default: 100)")),
		), handler: s.handleActivatePackage},

		{tool: mcp.NewTool("RunUnitTests",
			mcp.WithDescription("Run ABAP Unit tests for an object"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST)")),
			mcp.WithBoolean("include_dangerous", mcp.Description("Include dangerous risk level tests (default: false)")),
			mcp.WithBoolean("include_long", mcp.Description("Include long duration tests (default: false)")),
		), handler: s.handleRunUnitTests},

		{tool: mcp.NewTool("RunATCCheck",
			mcp.WithDescription("Run ATC (ABAP Test Cockpit) code quality check on an object. Returns findings with priority, check title, message, and location. Priority: 1=Error, 2=Warning, 3=Info."),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST)")),
			mcp.WithString("variant", mcp.Description("Check variant name (empty = use system default)")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of findings to return (default: 100)")),
		), handler: s.handleRunATCCheck},

		{tool: mcp.NewTool("GetATCCustomizing",
			mcp.WithDescription("Get ATC system configuration including default check variant and exemption reasons"),
		), handler: s.handleGetATCCustomizing},

		{tool: mcp.NewTool("PrettyPrint",
			mcp.WithDescription("Format ABAP source code using the pretty printer"),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP source code to format")),
		), handler: s.handlePrettyPrint},

		{tool: mcp.NewTool("GetPrettyPrinterSettings",
			mcp.WithDescription("Get the current pretty printer (code formatter) settings"),
		), handler: s.handleGetPrettyPrinterSettings},

		{tool: mcp.NewTool("SetPrettyPrinterSettings",
			mcp.WithDescription("Update the pretty printer (code formatter) settings"),
			mcp.WithBoolean("indentation", mcp.Required(), mcp.Description("Enable automatic indentation")),
			mcp.WithString("style", mcp.Required(), mcp.Description("Keyword style: toLower, toUpper, keywordUpper, keywordLower, keywordAuto, none")),
		), handler: s.handleSetPrettyPrinterSettings},

		{tool: mcp.NewTool("GetInactiveObjects",
			mcp.WithDescription("Get all inactive objects for the current user - objects that have been modified but not yet activated"),
		), handler: s.handleGetInactiveObjects},

		{tool: mcp.NewTool("ExecuteABAP",
			mcp.WithDescription("Execute arbitrary ABAP code via unit test wrapper. Creates temp program, injects code into test method, runs via RunUnitTests, extracts results from assertion messages, cleans up. Use lv_result variable to return output. WARNING: Powerful tool - use responsibly."),
			mcp.WithString("code", mcp.Required(), mcp.Description("ABAP code to execute. Set lv_result variable to return output via assertion message.")),
			mcp.WithString("risk_level", mcp.Description("Risk level: harmless (default, no DB writes), dangerous (can write to DB), critical (full access)")),
			mcp.WithString("return_variable", mcp.Description("Name of the variable to return (default: lv_result)")),
			mcp.WithBoolean("keep_program", mcp.Description("Don't delete temp program after execution (for debugging)")),
			mcp.WithString("program_prefix", mcp.Description("Prefix for temp program name (default: ZTEMP_EXEC_)")),
		), handler: s.handleExecuteABAP},
	}
}

// crudToolDefs returns tool definitions for CRUD operations.
func (s *Server) crudToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("LockObject",
			mcp.WithDescription("Acquire an edit lock on an ABAP object"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("access_mode", mcp.Description("Access mode: MODIFY (default) or READ")),
		), handler: s.handleLockObject},

		{tool: mcp.NewTool("UnlockObject",
			mcp.WithDescription("Release an edit lock on an ABAP object"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("lock_handle", mcp.Required(), mcp.Description("Lock handle from LockObject")),
		), handler: s.handleUnlockObject},

		{tool: mcp.NewTool("UpdateSource",
			mcp.WithDescription("Write source code to an ABAP object (requires lock)"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP source code to write")),
			mcp.WithString("lock_handle", mcp.Required(), mcp.Description("Lock handle from LockObject")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleUpdateSource},

		{tool: mcp.NewTool("CreateObject",
			mcp.WithDescription("Create a new ABAP object. Supports: PROG/P (program), CLAS/OC (class), INTF/OI (interface), PROG/I (include), FUGR/F (function group), FUGR/FF (function module), DEVC/K (package), DDLS/DF (CDS view), BDEF/BDO (behavior definition), SRVD/SRV (service definition), SRVB/SVB (service binding)"),
			mcp.WithString("object_type", mcp.Required(), mcp.Description("Object type: PROG/P, CLAS/OC, INTF/OI, PROG/I, FUGR/F, FUGR/FF, DEVC/K, DDLS/DF, BDEF/BDO, SRVD/SRV, SRVB/SVB")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Object name (e.g., ZTEST_PROGRAM)")),
			mcp.WithString("description", mcp.Required(), mcp.Description("Object description")),
			mcp.WithString("package_name", mcp.Required(), mcp.Description("Package name (e.g., $TMP for local, ZPACKAGE for transportable)")),
			mcp.WithString("transport", mcp.Description("Transport request number (required for non-local packages)")),
			mcp.WithString("parent_name", mcp.Description("Parent name (required for function modules - the function group name)")),
			mcp.WithString("service_definition", mcp.Description("For SRVB: the service definition name to bind")),
			mcp.WithString("binding_version", mcp.Description("For SRVB: OData version 'V2' or 'V4' (default: V2)")),
			mcp.WithString("binding_category", mcp.Description("For SRVB: '0' for Web API, '1' for UI (default: 0)")),
		), handler: s.handleCreateObject},

		{tool: mcp.NewTool("CreatePackage",
			mcp.WithDescription("Create a new ABAP package. Local packages ($*) work by default. Transportable packages require --enable-transports flag and transport parameter."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Package name (e.g., $ZTEST for local, ZPRODUCTION for transportable)")),
			mcp.WithString("description", mcp.Required(), mcp.Description("Package description")),
			mcp.WithString("parent", mcp.Description("Parent package name (optional, e.g., $TMP, ZPROD). If not specified, creates a root-level package.")),
			mcp.WithString("transport", mcp.Description("Transport request number (required for transportable packages, e.g., 'A4HK900114')")),
			mcp.WithString("software_component", mcp.Description("Software component name (required for transportable packages, e.g., 'HOME', 'ZLOCAL'). Use GetInstalledComponents to list available components.")),
		), handler: s.handleCreatePackage},

		{tool: mcp.NewTool("CreateTable",
			mcp.WithDescription("Create a DDIC transparent table from a simple JSON definition. Handles full workflow: create â†’ set source â†’ activate. Supports common ABAP types: CHAR, NUMC, INT4, DEC, STRING, TIMESTAMPL, UUID, etc."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Table name (uppercase, max 30 chars, must start with Z/Y)")),
			mcp.WithString("description", mcp.Required(), mcp.Description("Short description of the table")),
			mcp.WithString("package", mcp.Description("Target package (default: $TMP)")),
			mcp.WithString("fields", mcp.Required(), mcp.Description("JSON array of fields: [{\"name\":\"ID\",\"type\":\"CHAR32\",\"key\":true},{\"name\":\"VALUE\",\"type\":\"STRING\"}]. Types: CHAR/CHARnn, NUMC/NUMCnn, INT4, DEC, STRING, TIMESTAMPL, UUID, DATS, TIMS, or data element name.")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for $TMP)")),
			mcp.WithString("delivery_class", mcp.Description("Delivery class: A=Application (default), C=Customizing, L=Temporary")),
		), handler: s.handleCreateTable},

		{tool: mcp.NewTool("CompareSource",
			mcp.WithDescription("Compare source code of two objects and return unified diff. Supports all object types from GetSource."),
			mcp.WithString("type1", mcp.Required(), mcp.Description("Object type of first object: PROG, CLAS, INTF, FUNC, FUGR, INCL, DDLS, BDEF, SRVD")),
			mcp.WithString("name1", mcp.Required(), mcp.Description("Name of first object")),
			mcp.WithString("type2", mcp.Required(), mcp.Description("Object type of second object (can be same or different)")),
			mcp.WithString("name2", mcp.Required(), mcp.Description("Name of second object")),
			mcp.WithString("include1", mcp.Description("Class include type for first object if CLAS: definitions, implementations, macros, testclasses")),
			mcp.WithString("include2", mcp.Description("Class include type for second object if CLAS")),
			mcp.WithString("parent1", mcp.Description("Function group for first object if FUNC")),
			mcp.WithString("parent2", mcp.Description("Function group for second object if FUNC")),
		), handler: s.handleCompareSource},

		{tool: mcp.NewTool("CloneObject",
			mcp.WithDescription("Copy an ABAP object to a new name. Replaces object name in source. Supports PROG, CLAS, INTF."),
			mcp.WithString("object_type", mcp.Required(), mcp.Description("Object type: PROG, CLAS, INTF")),
			mcp.WithString("source_name", mcp.Required(), mcp.Description("Name of object to copy")),
			mcp.WithString("target_name", mcp.Required(), mcp.Description("Name for the new object")),
			mcp.WithString("package", mcp.Required(), mcp.Description("Target package (e.g., $TMP)")),
		), handler: s.handleCloneObject},

		{tool: mcp.NewTool("GetClassInfo",
			mcp.WithDescription("Get class metadata without full source: methods, attributes, interfaces, superclass, abstract/final status."),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
		), handler: s.handleGetClassInfo},

		{tool: mcp.NewTool("DeleteObject",
			mcp.WithDescription("Delete an ABAP object (requires lock)"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("lock_handle", mcp.Required(), mcp.Description("Lock handle from LockObject")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleDeleteObject},

		{tool: mcp.NewTool("GetUserTransports",
			mcp.WithDescription("Get all transport requests for a user (requires --enable-transports flag). Returns both workbench and customizing requests grouped by target system."),
			mcp.WithString("user_name", mcp.Required(), mcp.Description("SAP user name (will be converted to uppercase)")),
		), handler: s.handleGetUserTransports},

		{tool: mcp.NewTool("GetTransportInfo",
			mcp.WithDescription("Get transport information for an ABAP object (requires --enable-transports flag). Returns available transports and lock status."),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("dev_class", mcp.Required(), mcp.Description("Development class/package of the object")),
		), handler: s.handleGetTransportInfo},
	}
}

// classIncludeToolDefs returns tool definitions for class include operations.
func (s *Server) classIncludeToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("GetClassInclude",
			mcp.WithDescription("Retrieve source code of a class include (definitions, implementations, macros, testclasses)"),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
			mcp.WithString("include_type", mcp.Required(), mcp.Description("Include type: main, definitions, implementations, macros, testclasses")),
		), handler: s.handleGetClassInclude},

		{tool: mcp.NewTool("CreateTestInclude",
			mcp.WithDescription("Create the test classes include for a class (required before writing test code)"),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
			mcp.WithString("lock_handle", mcp.Required(), mcp.Description("Lock handle from LockObject (lock the parent class first)")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleCreateTestInclude},

		{tool: mcp.NewTool("UpdateClassInclude",
			mcp.WithDescription("Update source code of a class include (requires lock on parent class)"),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
			mcp.WithString("include_type", mcp.Required(), mcp.Description("Include type: main, definitions, implementations, macros, testclasses")),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP source code to write")),
			mcp.WithString("lock_handle", mcp.Required(), mcp.Description("Lock handle from LockObject (lock the parent class first)")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleUpdateClassInclude},

		{tool: mcp.NewTool("PublishServiceBinding",
			mcp.WithDescription("Publish a service binding to make it available as OData service"),
			mcp.WithString("service_name", mcp.Required(), mcp.Description("Service binding name (e.g., ZTRAVEL_SB)")),
			mcp.WithString("service_version", mcp.Description("Service version (default: 0001)")),
		), handler: s.handlePublishServiceBinding},

		{tool: mcp.NewTool("UnpublishServiceBinding",
			mcp.WithDescription("Unpublish a service binding"),
			mcp.WithString("service_name", mcp.Required(), mcp.Description("Service binding name (e.g., ZTRAVEL_SB)")),
			mcp.WithString("service_version", mcp.Description("Service version (default: 0001)")),
		), handler: s.handleUnpublishServiceBinding},
	}
}

// workflowToolDefs returns tool definitions for workflow tools.
func (s *Server) workflowToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("WriteProgram",
			mcp.WithDescription("Update an existing program with syntax check and activation (Lock -> SyntaxCheck -> Update -> Unlock -> Activate)"),
			mcp.WithString("program_name", mcp.Required(), mcp.Description("Name of the ABAP program")),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP source code")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleWriteProgram},

		{tool: mcp.NewTool("WriteClass",
			mcp.WithDescription("Update an existing class with syntax check and activation (Lock -> SyntaxCheck -> Update -> Unlock -> Activate)"),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP class source code (definition and implementation)")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleWriteClass},

		{tool: mcp.NewTool("CreateAndActivateProgram",
			mcp.WithDescription("Create a new program with source code and activate it (Create -> Lock -> Update -> Unlock -> Activate)"),
			mcp.WithString("program_name", mcp.Required(), mcp.Description("Name of the ABAP program")),
			mcp.WithString("description", mcp.Required(), mcp.Description("Program description")),
			mcp.WithString("package_name", mcp.Required(), mcp.Description("Package name (e.g., $TMP for local)")),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP source code")),
			mcp.WithString("transport", mcp.Description("Transport request number (required for non-local packages)")),
		), handler: s.handleCreateAndActivateProgram},

		{tool: mcp.NewTool("CreateClassWithTests",
			mcp.WithDescription("Create a new class with unit tests and run them (Create -> Lock -> Update -> CreateTestInclude -> UpdateTest -> Unlock -> Activate -> RunTests)"),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
			mcp.WithString("description", mcp.Required(), mcp.Description("Class description")),
			mcp.WithString("package_name", mcp.Required(), mcp.Description("Package name (e.g., $TMP for local)")),
			mcp.WithString("class_source", mcp.Required(), mcp.Description("ABAP class source code (definition and implementation)")),
			mcp.WithString("test_source", mcp.Required(), mcp.Description("ABAP unit test source code")),
			mcp.WithString("transport", mcp.Description("Transport request number (required for non-local packages)")),
		), handler: s.handleCreateClassWithTests},
	}
}

// fileToolDefs returns tool definitions for file-based deployment tools.
func (s *Server) fileToolDefs() []toolDef {
	defs := []toolDef{
		{tool: mcp.NewTool("DeployFromFile",
			mcp.WithDescription("âœ… RECOMMENDED - Smart deploy from file: auto-detects if object exists and creates/updates accordingly. Solves token limit problem for large generated files (ML models, 3948+ lines). Example: DeployFromFile(file_path=\"/path/to/zcl_ml_iris.clas.abap\", package_name=\"$ZAML_IRIS\") deploys any size file. Workflow: Parse â†’ Check existence â†’ Create or Update â†’ Lock â†’ SyntaxCheck â†’ Write â†’ Unlock â†’ Activate. Supports .clas.abap, .prog.abap, .intf.abap, .fugr.abap, .func.abap. Use this for all file-based deployments."),
			mcp.WithString("file_path", mcp.Required(), mcp.Description("Absolute path to ABAP source file")),
			mcp.WithString("package_name", mcp.Required(), mcp.Description("Package name (required for new objects, e.g., $ZAML_IRIS)")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleDeployFromFile},

		{tool: mcp.NewTool("SaveToFile",
			mcp.WithDescription("Save ABAP object source to local file (SAP â†’ File). Enables BIDIRECTIONAL SYNC WORKFLOW: (1) SaveToFile downloads object from SAP, (2) edit locally with vim/VS Code/AI assistants, (3) DeployFromFile uploads changes back to SAP. Example: SaveToFile(objType=\"CLAS/OC\", objectName=\"ZCL_ML_IRIS\", outputPath=\"./src/\") creates ./src/zcl_ml_iris.clas.abap. Then edit locally and use DeployFromFile to sync back. Recommended for iterative development. Auto-determines file extension."),
			mcp.WithString("objType", mcp.Required(), mcp.Description("Object type: CLAS/OC (class), PROG/P (program), INTF/OI (interface), FUGR/F (function group), FUGR/FF (function module)")),
			mcp.WithString("objectName", mcp.Required(), mcp.Description("Object name (e.g., ZCL_ML_IRIS, ZAML_IRIS_DEMO)")),
			mcp.WithString("outputPath", mcp.Description("Output file path or directory. If directory, filename is auto-generated with correct extension. If omitted, saves to current directory.")),
		), handler: s.handleSaveToFile},

		{tool: mcp.NewTool("RenameObject",
			mcp.WithDescription("Rename ABAP object by creating copy with new name and deleting old one. Useful for fixing naming conventions. Workflow: GetSource â†’ Replace names â†’ CreateNew â†’ ActivateNew â†’ DeleteOld"),
			mcp.WithString("objType", mcp.Required(), mcp.Description("Object type: CLAS/OC (class), PROG/P (program), INTF/OI (interface), FUGR/F (function group)")),
			mcp.WithString("oldName", mcp.Required(), mcp.Description("Current object name")),
			mcp.WithString("newName", mcp.Required(), mcp.Description("New object name")),
			mcp.WithString("packageName", mcp.Required(), mcp.Description("Package name for new object (e.g., $ZAML_IRIS)")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleRenameObject},
	}
	// Append ImportFromFile/ExportToFile from handlers_source.go
	defs = append(defs, s.fileSourceToolDefs()...)
	return defs
}

// editToolDefs returns tool definitions for surgical edit tools.
func (s *Server) editToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("EditSource",
			mcp.WithDescription("Surgical string replacement on ABAP source code. Matches the Edit tool pattern for local files. Workflow: GetSource â†’ FindReplace â†’ SyntaxCheck â†’ Lock â†’ Update â†’ Unlock â†’ Activate. Example: EditSource(object_url=\"/sap/bc/adt/programs/programs/ZTEST\", old_string=\"METHOD foo.\\n  ENDMETHOD.\", new_string=\"METHOD foo.\\n  rv_result = 42.\\n  ENDMETHOD.\", replace_all=false, syntax_check=true). Requires unique match if replace_all=false. Use this for incremental edits between syntax checks - no need to download/upload full source!"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of object (e.g., /sap/bc/adt/programs/programs/ZTEST, /sap/bc/adt/oo/classes/zcl_test)")),
			mcp.WithString("old_string", mcp.Required(), mcp.Description("Exact string to find and replace. Must be unique in source if replace_all=false. Include enough context (surrounding lines) to ensure uniqueness.")),
			mcp.WithString("new_string", mcp.Required(), mcp.Description("Replacement string. Can be multiline (use \\n). Length can differ from old_string.")),
			mcp.WithBoolean("replace_all", mcp.Description("If true, replace all occurrences. If false (default), require unique match. Default: false")),
			mcp.WithBoolean("syntax_check", mcp.Description("If true (default), validate syntax before saving. If syntax errors found, changes are NOT saved. Default: true")),
			mcp.WithBoolean("case_insensitive", mcp.Description("If true, ignore case when matching old_string. Useful for renaming variables regardless of case. Default: false")),
			mcp.WithString("method", mcp.Description("For CLAS only: constrain search/replace to this method only. Prevents accidental edits in other methods. (optional)")),
			mcp.WithString("transport", mcp.Description("Transport request number (required for objects not in $TMP package)")),
		), handler: s.handleEditSource},
	}
}

// grepToolDefs returns tool definitions for grep/search tools.
func (s *Server) grepToolDefs() []toolDef {
	defs := []toolDef{
		{tool: mcp.NewTool("GrepObject",
			mcp.WithDescription("Search for regex pattern in a single ABAP object's source code. Returns matches with line numbers and optional context. Use for finding TODO comments, string literals, patterns, or code snippets before editing."),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("pattern", mcp.Required(), mcp.Description("Regular expression pattern (Go regexp syntax). Examples: 'TODO', 'lv_\\\\w+', 'SELECT.*FROM'")),
			mcp.WithBoolean("case_insensitive", mcp.Description("If true, perform case-insensitive matching. Default: false")),
			mcp.WithNumber("context_lines", mcp.Description("Number of lines to show before/after each match (like grep -C). Default: 0")),
		), handler: s.handleGrepObject},

		{tool: mcp.NewTool("GrepPackage",
			mcp.WithDescription("Search for regex pattern across all source objects in an ABAP package. Returns matches grouped by object. Use for package-wide analysis, finding patterns across multiple programs/classes."),
			mcp.WithString("package_name", mcp.Required(), mcp.Description("Package name (e.g., $TMP, ZPACKAGE)")),
			mcp.WithString("pattern", mcp.Required(), mcp.Description("Regular expression pattern (Go regexp syntax). Examples: 'TODO', 'lv_\\\\w+', 'SELECT.*FROM'")),
			mcp.WithBoolean("case_insensitive", mcp.Description("If true, perform case-insensitive matching. Default: false")),
			mcp.WithString("object_types", mcp.Description("Comma-separated object types to search (e.g., 'PROG/P,CLAS/OC'). Empty = search all source objects. Valid: PROG/P, CLAS/OC, INTF/OI, FUGR/F, FUGR/FF, PROG/I")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of matching objects to return. 0 = unlimited. Default: 100")),
		), handler: s.handleGrepPackage},
	}
	// Append GrepObjects/GrepPackages from handlers_source.go
	defs = append(defs, s.grepSourceToolDefs()...)
	return defs
}

// codeIntelToolDefs returns tool definitions for code intelligence tools.
func (s *Server) codeIntelToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("FindDefinition",
			mcp.WithDescription("Navigate to the definition of a symbol at a given position in source code"),
			mcp.WithString("source_url", mcp.Required(), mcp.Description("ADT URL of the source file (e.g., /sap/bc/adt/programs/programs/ZTEST/source/main)")),
			mcp.WithString("source", mcp.Required(), mcp.Description("Full source code of the file")),
			mcp.WithNumber("line", mcp.Required(), mcp.Description("Line number (1-based)")),
			mcp.WithNumber("start_column", mcp.Required(), mcp.Description("Start column of the symbol (1-based)")),
			mcp.WithNumber("end_column", mcp.Required(), mcp.Description("End column of the symbol (1-based)")),
			mcp.WithBoolean("implementation", mcp.Description("Navigate to implementation instead of definition (default: false)")),
			mcp.WithString("main_program", mcp.Description("Main program for includes (optional)")),
		), handler: s.handleFindDefinition},

		{tool: mcp.NewTool("FindReferences",
			mcp.WithDescription("Find all references to an ABAP object or symbol"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST)")),
			mcp.WithNumber("line", mcp.Description("Line number for position-based reference search (1-based, optional)")),
			mcp.WithNumber("column", mcp.Description("Column number for position-based reference search (1-based, optional)")),
		), handler: s.handleFindReferences},

		{tool: mcp.NewTool("GetContext",
			mcp.WithDescription("Analyze ABAP source dependencies and return compressed public API contracts (prologue). Produces a compact summary of all referenced classes, interfaces, and function modules â€” showing only their public signatures. Use this to understand the surrounding codebase context before editing."),
			mcp.WithString("object_type", mcp.Required(), mcp.Description("ABAP object type: PROG, CLAS, INTF, FUNC, FUGR")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Object name (e.g., ZCL_ORDER_PROCESSOR)")),
			mcp.WithString("source", mcp.Description("Source code to analyze (if omitted, fetched from SAP)")),
			mcp.WithNumber("max_deps", mcp.Description("Maximum dependencies to resolve (default: 20)")),
		), handler: s.handleGetContext},

		{tool: mcp.NewTool("CodeCompletion",
			mcp.WithDescription("Get code completion suggestions at a position in source code"),
			mcp.WithString("source_url", mcp.Required(), mcp.Description("ADT URL of the source file (e.g., /sap/bc/adt/programs/programs/ZTEST/source/main)")),
			mcp.WithString("source", mcp.Required(), mcp.Description("Full source code of the file")),
			mcp.WithNumber("line", mcp.Required(), mcp.Description("Line number (1-based)")),
			mcp.WithNumber("column", mcp.Required(), mcp.Description("Column number (1-based)")),
		), handler: s.handleCodeCompletion},

		{tool: mcp.NewTool("GetTypeHierarchy",
			mcp.WithDescription("Get the type hierarchy (supertypes or subtypes) for a class/interface"),
			mcp.WithString("source_url", mcp.Required(), mcp.Description("ADT URL of the source file")),
			mcp.WithString("source", mcp.Required(), mcp.Description("Full source code of the file")),
			mcp.WithNumber("line", mcp.Required(), mcp.Description("Line number (1-based)")),
			mcp.WithNumber("column", mcp.Required(), mcp.Description("Column number (1-based)")),
			mcp.WithBoolean("super_types", mcp.Description("Get supertypes instead of subtypes (default: false = subtypes)")),
		), handler: s.handleGetTypeHierarchy},

		{tool: mcp.NewTool("GetClassComponents",
			mcp.WithDescription("Get the structure of a class - lists all methods, attributes, events, and other components with their visibility and properties"),
			mcp.WithString("class_url", mcp.Required(), mcp.Description("ADT URL of the class (e.g., /sap/bc/adt/oo/classes/ZCL_TEST)")),
		), handler: s.handleGetClassComponents},
	}
}

// ui5ToolDefs returns tool definitions for UI5/Fiori BSP management tools.
func (s *Server) ui5ToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("UI5ListApps",
			mcp.WithDescription("List UI5/Fiori BSP applications. Use query parameter for filtering with wildcards (*)."),
			mcp.WithString("query", mcp.Description("Search query (supports * wildcard, e.g., 'Z*' for custom apps)")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of results (default: 100)")),
		), handler: s.handleUI5ListApps},

		{tool: mcp.NewTool("UI5GetApp",
			mcp.WithDescription("Get details of a UI5/Fiori BSP application including file structure."),
			mcp.WithString("app_name", mcp.Required(), mcp.Description("Name of the UI5 application")),
		), handler: s.handleUI5GetApp},

		{tool: mcp.NewTool("UI5GetFileContent",
			mcp.WithDescription("Get content of a specific file within a UI5/Fiori BSP application."),
			mcp.WithString("app_name", mcp.Required(), mcp.Description("Name of the UI5 application")),
			mcp.WithString("file_path", mcp.Required(), mcp.Description("Path to the file within the app (e.g., '/webapp/manifest.json')")),
		), handler: s.handleUI5GetFileContent},

		{tool: mcp.NewTool("UI5UploadFile",
			mcp.WithDescription("Upload a file to a UI5/Fiori BSP application."),
			mcp.WithString("app_name", mcp.Required(), mcp.Description("Name of the UI5 application")),
			mcp.WithString("file_path", mcp.Required(), mcp.Description("Path for the file within the app (e.g., '/webapp/Component.js')")),
			mcp.WithString("content", mcp.Required(), mcp.Description("File content to upload")),
			mcp.WithString("content_type", mcp.Description("Content type (e.g., 'application/javascript', 'application/json')")),
		), handler: s.handleUI5UploadFile},

		{tool: mcp.NewTool("UI5DeleteFile",
			mcp.WithDescription("Delete a file from a UI5/Fiori BSP application."),
			mcp.WithString("app_name", mcp.Required(), mcp.Description("Name of the UI5 application")),
			mcp.WithString("file_path", mcp.Required(), mcp.Description("Path to the file to delete (e.g., '/webapp/test.js')")),
		), handler: s.handleUI5DeleteFile},

		{tool: mcp.NewTool("UI5CreateApp",
			mcp.WithDescription("Create a new UI5/Fiori BSP application."),
			mcp.WithString("app_name", mcp.Required(), mcp.Description("Name for the new UI5 application")),
			mcp.WithString("description", mcp.Description("Description of the application")),
			mcp.WithString("package", mcp.Required(), mcp.Description("Package name (e.g., '$TMP' for local, 'ZFIORI' for transportable)")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleUI5CreateApp},

		{tool: mcp.NewTool("UI5DeleteApp",
			mcp.WithDescription("Delete a UI5/Fiori BSP application."),
			mcp.WithString("app_name", mcp.Required(), mcp.Description("Name of the UI5 application to delete")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleUI5DeleteApp},
	}
}

// amdpToolDefs returns tool definitions for AMDP/HANA debugger tools.
func (s *Server) amdpToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("AMDPDebuggerStart",
			mcp.WithDescription("Start an AMDP (HANA SQLScript) debug session with persistent goroutine. Creates a background goroutine that maintains the HTTP session cookies. Use AMDPDebuggerStep/AMDPGetVariables to interact, AMDPDebuggerStop to terminate."),
			mcp.WithString("user", mcp.Description("User to debug (defaults to current user)")),
		), handler: s.handleAMDPDebuggerStart},

		{tool: mcp.NewTool("AMDPDebuggerResume",
			mcp.WithDescription("Get current AMDP debug session status. In goroutine model, this returns the current state without blocking. The session manager goroutine handles events internally."),
		), handler: s.handleAMDPDebuggerResume},

		{tool: mcp.NewTool("AMDPDebuggerStop",
			mcp.WithDescription("Stop the AMDP debug session and terminate the background goroutine. Cleans up the HTTP session on SAP server."),
		), handler: s.handleAMDPDebuggerStop},

		{tool: mcp.NewTool("AMDPDebuggerStep",
			mcp.WithDescription("Perform a step operation in the AMDP debugger. Communicates via channel to the session manager goroutine."),
			mcp.WithString("step_type", mcp.Required(), mcp.Description("Step type: 'stepInto', 'stepOver', 'stepReturn', 'stepContinue'")),
		), handler: s.handleAMDPDebuggerStep},

		{tool: mcp.NewTool("AMDPGetVariables",
			mcp.WithDescription("Get variable values during AMDP debugging. Communicates via channel to the session manager goroutine. Returns scalar, table, and array types."),
		), handler: s.handleAMDPGetVariables},

		{tool: mcp.NewTool("AMDPSetBreakpoint",
			mcp.WithDescription("Set a breakpoint in AMDP (SQLScript) code. Requires an active AMDP debug session. Specify the procedure name and line number."),
			mcp.WithString("proc_name", mcp.Required(), mcp.Description("AMDP procedure name (e.g., 'ZCL_TEST=>METHOD_NAME')")),
			mcp.WithNumber("line", mcp.Required(), mcp.Description("Line number in the SQLScript code")),
		), handler: s.handleAMDPSetBreakpoint},

		{tool: mcp.NewTool("AMDPGetBreakpoints",
			mcp.WithDescription("Get all breakpoints registered in the current AMDP debug session. Useful for verifying breakpoints are set correctly."),
		), handler: s.handleAMDPGetBreakpoints},
	}
}

// transportToolDefs returns tool definitions for CTS/Transport management tools.
func (s *Server) transportToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("ListTransports",
			mcp.WithDescription("List transport requests. Returns modifiable transports for a user. Requires --enable-transports OR --allow-transportable-edits flag."),
			mcp.WithString("user", mcp.Description("Username to list transports for (default: current user, '*' for all users)")),
		), handler: s.handleListTransports},

		{tool: mcp.NewTool("GetTransport",
			mcp.WithDescription("Get detailed transport information including objects and tasks. Requires --enable-transports OR --allow-transportable-edits flag."),
			mcp.WithString("transport", mcp.Required(), mcp.Description("Transport request number (e.g., 'A4HK900094')")),
		), handler: s.handleGetTransport},

		{tool: mcp.NewTool("CreateTransport",
			mcp.WithDescription("Create a new transport request. Requires --enable-transports flag and not --transport-read-only."),
			mcp.WithString("description", mcp.Required(), mcp.Description("Transport description")),
			mcp.WithString("package", mcp.Required(), mcp.Description("Target package (DEVCLASS)")),
			mcp.WithString("transport_layer", mcp.Description("Transport layer (optional)")),
			mcp.WithString("type", mcp.Description("Type: 'workbench' (default) or 'customizing'")),
		), handler: s.handleCreateTransport},

		{tool: mcp.NewTool("ReleaseTransport",
			mcp.WithDescription("Release a transport request. This action is IRREVERSIBLE. Requires --enable-transports flag and not --transport-read-only."),
			mcp.WithString("transport", mcp.Required(), mcp.Description("Transport request number")),
			mcp.WithBoolean("ignore_locks", mcp.Description("Release even with locked objects (default: false)")),
			mcp.WithBoolean("skip_atc", mcp.Description("Skip ATC quality checks (default: false)")),
		), handler: s.handleReleaseTransport},

		{tool: mcp.NewTool("DeleteTransport",
			mcp.WithDescription("Delete a transport request. Only modifiable transports can be deleted. Requires --enable-transports flag and not --transport-read-only."),
			mcp.WithString("transport", mcp.Required(), mcp.Description("Transport request number")),
		), handler: s.handleDeleteTransport},
	}
}

// gitToolDefs returns tool definitions for Git/abapGit integration tools.
func (s *Server) gitToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("GitTypes",
			mcp.WithDescription("Get list of supported abapGit object types. Returns 158 object types that can be exported/imported via abapGit. Requires abapGit to be installed on SAP system."),
		), handler: s.handleGitTypes},

		{tool: mcp.NewTool("GitExport",
			mcp.WithDescription("Export ABAP objects as abapGit-compatible ZIP. Supports 158 object types. Saves ZIP file to output_dir (default: current directory). Use packages OR objects parameter."),
			mcp.WithString("packages", mcp.Description("Comma-separated package names to export (e.g., '$ZRAY,$TMP'). Supports wildcards.")),
			mcp.WithString("objects", mcp.Description("JSON array of objects: [{\"type\":\"CLAS\",\"name\":\"ZCL_TEST\"}]")),
			mcp.WithBoolean("include_subpackages", mcp.Description("Include subpackages when exporting by package (default: true)")),
			mcp.WithString("output_dir", mcp.Description("Output directory for ZIP file (default: current directory)")),
		), handler: s.handleGitExport},
	}
}

// reportToolDefs returns tool definitions for report execution tools.
func (s *Server) reportToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("RunReport",
			mcp.WithDescription("Execute an ABAP selection-screen report with parameters or variant. Runs as background job and returns spool output. Requires ZADT_VSP WebSocket handler deployed."),
			mcp.WithString("report", mcp.Required(), mcp.Description("Report program name (e.g., 'RFITEMGL', 'ZREPORT_TEST')")),
			mcp.WithString("variant", mcp.Description("Variant name to use for selection screen (optional)")),
			mcp.WithString("params", mcp.Description("JSON object with selection screen parameters (e.g., '{\"P_BUKRS\":\"1000\",\"S_KUNNR\":{\"SIGN\":\"I\",\"OPTION\":\"EQ\",\"LOW\":\"0000001000\"}}'). Keys are parameter names.")),
		), handler: s.handleRunReport},

		{tool: mcp.NewTool("RunReportAsync",
			mcp.WithDescription("Start report execution in background. Returns task_id immediately. Use GetAsyncResult to poll for completion. Useful for long-running reports that would timeout."),
			mcp.WithString("report", mcp.Required(), mcp.Description("Report program name")),
			mcp.WithString("variant", mcp.Description("Variant name (optional)")),
			mcp.WithString("params", mcp.Description("JSON object with selection screen parameters")),
		), handler: s.handleRunReportAsync},

		{tool: mcp.NewTool("GetAsyncResult",
			mcp.WithDescription("Get result of an async task by ID. Returns status (running/completed/error) and result when done."),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID from RunReportAsync")),
			mcp.WithBoolean("wait", mcp.Description("If true, block until task completes (max 60s). Default: false (poll)")),
		), handler: s.handleGetAsyncResult},

		{tool: mcp.NewTool("GetVariants",
			mcp.WithDescription("Get list of available variants for a report program. Returns variant names and whether they are protected."),
			mcp.WithString("report", mcp.Required(), mcp.Description("Report program name")),
		), handler: s.handleGetVariants},

		{tool: mcp.NewTool("GetTextElements",
			mcp.WithDescription("Get program text elements (selection texts and text symbols). Selection texts describe parameters (P_BUKRS='Company Code'), text symbols are TEXT-001 etc."),
			mcp.WithString("program", mcp.Required(), mcp.Description("Program name")),
			mcp.WithString("language", mcp.Description("Language key (e.g., 'E' for English, 'D' for German). Default: system language.")),
		), handler: s.handleGetTextElements},

		{tool: mcp.NewTool("SetTextElements",
			mcp.WithDescription("Set program text elements (selection texts, text symbols, and heading texts). Use for adding descriptions to selection screen parameters, text symbols, and list/column headings."),
			mcp.WithString("program", mcp.Required(), mcp.Description("Program name")),
			mcp.WithString("language", mcp.Description("Language key (e.g., 'E' for English, 'D' for German). Default: system language.")),
			mcp.WithString("selection_texts", mcp.Description("JSON object of selection texts (e.g., '{\"P_BUKRS\":\"Company Code\",\"S_KUNNR\":\"Customer Range\"}')")),
			mcp.WithString("text_symbols", mcp.Description("JSON object of text symbols (e.g., '{\"001\":\"Header Text\",\"002\":\"Footer\"}')")),
			mcp.WithString("heading_texts", mcp.Description("JSON object of heading texts for list/column headings (e.g., '{\"001\":\"Report Title\",\"002\":\"Column Header\"}')")),
		), handler: s.handleSetTextElements},
	}
}
