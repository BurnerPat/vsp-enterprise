// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_debugger.go contains handlers for WebSocket-based debugging (via ZADT_VSP).
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

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
		), handler: s.handleSetBreakpoint, focused: true, groups: []string{"X"},
			routes: []universalRoute{{action: "debug", targetType: "SET_BREAKPOINT"}}},

		{tool: mcp.NewTool("GetBreakpoints",
			mcp.WithDescription("Get all breakpoints registered in the current debug session. Uses WebSocket connection to ZADT_VSP."),
		), handler: s.handleGetBreakpoints, readOnly: true, focused: true, groups: []string{"X"},
			routes: []universalRoute{{action: "debug", targetType: "GET_BREAKPOINTS"}}},

		{tool: mcp.NewTool("DeleteBreakpoint",
			mcp.WithDescription("Delete a breakpoint by ID. Uses WebSocket connection to ZADT_VSP."),
			mcp.WithString("breakpoint_id", mcp.Required(), mcp.Description("ID of the breakpoint to delete")),
		), handler: s.handleDeleteBreakpoint, focused: true, groups: []string{"X"},
			routes: []universalRoute{{action: "debug", targetType: "DELETE_BREAKPOINT"}}},

		{tool: mcp.NewTool("CallRFC",
			mcp.WithDescription("Call a function module via WebSocket (ZADT_VSP). Useful for triggering ABAP code execution to hit breakpoints. Parameters are passed as key-value pairs."),
			mcp.WithString("function", mcp.Required(), mcp.Description("Function module name (e.g., 'RFC_PING', 'BAPI_USER_GET_DETAIL')")),
			mcp.WithString("params", mcp.Description("JSON object with function parameters (e.g., '{\"IV_PARAM\":\"value\"}')")),
		), handler: s.handleCallRFC, focused: true,
			routes: []universalRoute{{action: "debug", targetType: "CALL_RFC"}}},

		{tool: mcp.NewTool("MoveObject",
			mcp.WithDescription("Move an ABAP object to a different package. Uses ZADT_VSP WebSocket to call TR_TADIR_INTERFACE. Requires ZADT_VSP deployed."),
			mcp.WithString("object_type", mcp.Required(), mcp.Description("Object type: CLAS, PROG, INTF, FUGR, TABL, etc.")),
			mcp.WithString("object_name", mcp.Required(), mcp.Description("Name of the object to move (e.g., 'ZCL_TEST')")),
			mcp.WithString("new_package", mcp.Required(), mcp.Description("Target package (e.g., '$ZRAY', 'ZPACKAGE')")),
		), handler: s.handleMoveObject, focused: true,
			routes: []universalRoute{{action: "debug", targetType: "MOVE"}}},

		{tool: mcp.NewTool("DebuggerListen",
			mcp.WithDescription("Start a debug listener that waits for a debuggee to hit a breakpoint. This is a BLOCKING call that uses long-polling. Returns when a debuggee is caught, timeout occurs, or a conflict is detected."),
			mcp.WithString("user", mcp.Description("User to listen for (defaults to current user)")),
			mcp.WithNumber("timeout", mcp.Description("Timeout in seconds (default: 60, max: 240)")),
		), handler: s.handleDebuggerListen, readOnly: true, focused: true, groups: []string{"D", "X"}},

		{tool: mcp.NewTool("DebuggerAttach",
			mcp.WithDescription("Attach to a debuggee that has hit a breakpoint. Use the debuggee_id from DebuggerListen result."),
			mcp.WithString("debuggee_id", mcp.Required(), mcp.Description("ID of the debuggee (from DebuggerListen result)")),
			mcp.WithString("user", mcp.Description("User for debugging (defaults to current user)")),
		), handler: s.handleDebuggerAttach, focused: true, groups: []string{"D", "X"}},

		{tool: mcp.NewTool("DebuggerDetach",
			mcp.WithDescription("Detach from the current debug session and release the debuggee."),
		), handler: s.handleDebuggerDetach, focused: true, groups: []string{"D", "X"}},

		{tool: mcp.NewTool("DebuggerStep",
			mcp.WithDescription("Perform a step operation in the debugger."),
			mcp.WithString("step_type", mcp.Required(), mcp.Description("Step type: 'stepInto', 'stepOver', 'stepReturn', 'stepContinue', 'stepRunToLine', 'stepJumpToLine'")),
			mcp.WithString("uri", mcp.Description("Target URI for stepRunToLine/stepJumpToLine (e.g., '/sap/bc/adt/programs/programs/ZTEST/source/main#start=42')")),
		), handler: s.handleDebuggerStep, focused: true, groups: []string{"D", "X"}},

		{tool: mcp.NewTool("DebuggerGetStack",
			mcp.WithDescription("Get the current call stack during a debug session."),
		), handler: s.handleDebuggerGetStack, readOnly: true, focused: true, groups: []string{"D", "X"}},

		{tool: mcp.NewTool("DebuggerGetVariables",
			mcp.WithDescription("Get variable values during a debug session. Use '@ROOT' to get top-level variables, or specific variable IDs to get their values."),
			mcp.WithArray("variable_ids",
				mcp.Description("Variable IDs to retrieve (e.g., ['@ROOT'] for top-level, or specific IDs like ['LV_COUNT', 'LS_DATA'])"),
				mcp.Items(map[string]interface{}{"type": "string"}),
			),
		), handler: s.handleDebuggerGetVariables, readOnly: true, focused: true, groups: []string{"D", "X"}},

		{tool: mcp.NewTool("DebuggerSetVariable",
			mcp.WithDescription("Set a variable value during a debug session. Requires an active debug session (after DebuggerAttach)."),
			mcp.WithString("variable_name", mcp.Required(), mcp.Description("Variable name (e.g., 'GV_SPEED', 'LV_COUNT', 'LS_DATA-FIELD')")),
			mcp.WithString("value", mcp.Required(), mcp.Description("New value as string (e.g., '70', 'Hello', 'X')")),
		), handler: s.handleDebuggerSetVariable, focused: true, groups: []string{"D", "X"}},
	}
}

// --- Debugger Session Handlers (WebSocket-based via ZADT_VSP) ---
// All breakpoint operations use WebSocket for reliable CSRF-free communication.

// ensureDebugWSClient ensures WebSocket debug client is connected.
func (s *Server) ensureDebugWSClient(ctx context.Context) error {
	if s.debugWSClient != nil && s.debugWSClient.IsConnected() {
		return nil
	}

	// Create new client
	s.debugWSClient = adt.NewDebugWebSocketClient(
		s.config.BaseURL,
		s.config.Client,
		s.config.Username,
		s.config.Password,
		s.config.InsecureSkipVerify,
	)

	return s.debugWSClient.Connect(ctx)
}

func (s *Server) handleSetBreakpoint(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get breakpoint kind (default: "line")
	kind, _ := request.Params.Arguments["kind"].(string)
	if kind == "" {
		kind = "line"
	}

	// RFC mode: use ADT REST API (routed through RfcTransport → sidecar → SADT_REST_RFC_ENDPOINT)
	// No CSRF issues since the HTTP call is internal to SAP.
	if s.isRfcMode() {
		return s.handleSetBreakpointRfc(ctx, request, kind)
	}

	// HTTP mode: use WebSocket client (ZADT_VSP)
	if err := s.ensureDebugWSClient(ctx); err != nil {
		return newToolResultError(fmt.Sprintf("Failed to connect to ZADT_VSP WebSocket: %v. Ensure ZADT_VSP is deployed and SAPC/SICF are configured.", err)), nil
	}

	var bpID string
	var err error
	var msg strings.Builder

	switch kind {
	case "line":
		program, ok := request.Params.Arguments["program"].(string)
		if !ok || program == "" {
			return newToolResultError("program is required for line breakpoints"), nil
		}

		lineFloat, ok := request.Params.Arguments["line"].(float64)
		if !ok || lineFloat <= 0 {
			return newToolResultError("line is required and must be positive for line breakpoints"), nil
		}
		line := int(lineFloat)

		// Optional method parameter for include-relative line numbers
		method, _ := request.Params.Arguments["method"].(string)

		// Auto-convert class names to pool format (ZCL_TEST → ZCL_TEST================CP)
		originalProgram := program
		program = convertToClassPool(program)

		// Use method-aware breakpoint if method is specified
		if method != "" {
			bpID, err = s.debugWSClient.SetMethodBreakpoint(ctx, program, method, line)
			if err != nil {
				return newToolResultError(fmt.Sprintf("SetMethodBreakpoint failed: %v", err)), nil
			}

			msg.WriteString("Method breakpoint set successfully!\n\n")
			fmt.Fprintf(&msg, "Breakpoint ID: %s\n", bpID)
			if program != originalProgram {
				fmt.Fprintf(&msg, "Program: %s (converted from %s)\n", program, originalProgram)
			} else {
				fmt.Fprintf(&msg, "Program: %s\n", program)
			}
			fmt.Fprintf(&msg, "Method: %s\n", method)
			fmt.Fprintf(&msg, "Line: %d (relative to method start)\n", line)
			msg.WriteString("\nℹ️  Line number is relative to the METHOD implementation, not the full class.\n")
		} else {
			bpID, err = s.debugWSClient.SetLineBreakpoint(ctx, program, line)
			if err != nil {
				return newToolResultError(fmt.Sprintf("SetLineBreakpoint failed: %v", err)), nil
			}

			msg.WriteString("Line breakpoint set successfully!\n\n")
			fmt.Fprintf(&msg, "Breakpoint ID: %s\n", bpID)
			if program != originalProgram {
				fmt.Fprintf(&msg, "Program: %s (converted from %s)\n", program, originalProgram)
			} else {
				fmt.Fprintf(&msg, "Program: %s\n", program)
			}
			fmt.Fprintf(&msg, "Line: %d (pool-absolute)\n", line)
		}

	case "statement":
		statement, ok := request.Params.Arguments["statement"].(string)
		if !ok || statement == "" {
			return newToolResultError("statement is required for statement breakpoints (e.g., 'CALL FUNCTION', 'SELECT', 'LOOP')"), nil
		}

		bpID, err = s.debugWSClient.SetStatementBreakpoint(ctx, statement)
		if err != nil {
			return newToolResultError(fmt.Sprintf("SetStatementBreakpoint failed: %v", err)), nil
		}

		msg.WriteString("Statement breakpoint set successfully!\n\n")
		fmt.Fprintf(&msg, "Breakpoint ID: %s\n", bpID)
		fmt.Fprintf(&msg, "Statement: %s\n", statement)
		msg.WriteString("\nThis breakpoint will trigger on ALL occurrences of this statement type.\n")

	case "exception":
		exception, ok := request.Params.Arguments["exception"].(string)
		if !ok || exception == "" {
			return newToolResultError("exception is required for exception breakpoints (e.g., 'CX_SY_ZERODIVIDE')"), nil
		}

		bpID, err = s.debugWSClient.SetExceptionBreakpoint(ctx, exception)
		if err != nil {
			return newToolResultError(fmt.Sprintf("SetExceptionBreakpoint failed: %v", err)), nil
		}

		msg.WriteString("Exception breakpoint set successfully!\n\n")
		fmt.Fprintf(&msg, "Breakpoint ID: %s\n", bpID)
		fmt.Fprintf(&msg, "Exception: %s\n", exception)
		msg.WriteString("\nThis breakpoint will trigger when this exception is raised.\n")

	default:
		return newToolResultError(fmt.Sprintf("Invalid breakpoint kind: %s. Valid kinds: line, statement, exception", kind)), nil
	}

	msg.WriteString("\n⚠️  IMPORTANT: Breakpoints only trigger for code executed in a DIFFERENT SAP session.\n")
	msg.WriteString("Use DebuggerListen in this session, then trigger execution from another session\n")
	msg.WriteString("(e.g., SAP GUI, HTTP request, RunUnitTests from another connection).")

	return mcp.NewToolResultText(msg.String()), nil
}

// handleSetBreakpointRfc handles breakpoints in RFC mode using the ADT REST API.
func (s *Server) handleSetBreakpointRfc(ctx context.Context, request mcp.CallToolRequest, kind string) (*mcp.CallToolResult, error) {
	var bp adt.Breakpoint
	var msg strings.Builder

	switch kind {
	case "line":
		program, ok := request.Params.Arguments["program"].(string)
		if !ok || program == "" {
			return newToolResultError("program is required for line breakpoints"), nil
		}
		lineFloat, ok := request.Params.Arguments["line"].(float64)
		if !ok || lineFloat <= 0 {
			return newToolResultError("line is required and must be positive for line breakpoints"), nil
		}

		originalProgram := program
		program = convertToClassPool(program)

		// Build ADT URI for the program
		uri := fmt.Sprintf("/sap/bc/adt/programs/programs/%s/source/main", strings.ToUpper(program))
		if strings.Contains(program, "=") {
			// Class pool — use class URI
			className := strings.Split(program, "=")[0]
			uri = fmt.Sprintf("/sap/bc/adt/oo/classes/%s/source/main", strings.ToLower(className))
		}

		bp = adt.Breakpoint{
			Kind:    adt.BreakpointKindLine,
			Enabled: true,
			URI:     uri,
			Line:    int(lineFloat),
		}

		if program != originalProgram {
			fmt.Fprintf(&msg, "Program: %s (converted from %s)\n", program, originalProgram)
		} else {
			fmt.Fprintf(&msg, "Program: %s\n", program)
		}
		fmt.Fprintf(&msg, "Line: %d\n", int(lineFloat))

	case "statement":
		statement, ok := request.Params.Arguments["statement"].(string)
		if !ok || statement == "" {
			return newToolResultError("statement is required for statement breakpoints"), nil
		}
		bp = adt.Breakpoint{
			Kind:      adt.BreakpointKindStatement,
			Enabled:   true,
			Statement: statement,
		}
		fmt.Fprintf(&msg, "Statement: %s\n", statement)

	case "exception":
		exception, ok := request.Params.Arguments["exception"].(string)
		if !ok || exception == "" {
			return newToolResultError("exception is required for exception breakpoints"), nil
		}
		bp = adt.Breakpoint{
			Kind:      adt.BreakpointKindException,
			Enabled:   true,
			Exception: exception,
		}
		fmt.Fprintf(&msg, "Exception: %s\n", exception)

	default:
		return newToolResultError(fmt.Sprintf("Invalid breakpoint kind: %s. Valid kinds: line, statement, exception", kind)), nil
	}

	bpReq := &adt.BreakpointRequest{
		User:        s.config.Username,
		Breakpoints: []adt.Breakpoint{bp},
	}
	resp, err := s.adtClient.SetExternalBreakpoint(ctx, bpReq)
	if err != nil {
		return newToolResultError(fmt.Sprintf("SetBreakpoint failed (RFC mode): %v", err)), nil
	}

	// Check if breakpoint was actually set (must have an ID)
	if len(resp.Breakpoints) == 0 || resp.Breakpoints[0].ID == "" {
		return newToolResultError(fmt.Sprintf("Breakpoint was NOT set — line %s is not an executable statement. Try a different line.",
			msg.String())), nil
	}

	var result strings.Builder
	result.WriteString("Breakpoint set successfully (RFC mode, ADT REST API)!\n\n")
	fmt.Fprintf(&result, "Breakpoint ID: %s\n", resp.Breakpoints[0].ID)
	result.WriteString(msg.String())

	return mcp.NewToolResultText(result.String()), nil
}

// convertToClassPool converts class/interface names to pool format for debugging.
// Example: ZCL_TEST → ZCL_TEST================CP (padded to 30 chars + CP suffix)
func convertToClassPool(program string) string {
	program = strings.ToUpper(program)

	// Already in pool format
	if strings.HasSuffix(program, "CP") && strings.Contains(program, "=") {
		return program
	}

	// Check if it looks like a class or interface name
	isClass := strings.HasPrefix(program, "ZCL_") ||
		strings.HasPrefix(program, "YCL_") ||
		strings.HasPrefix(program, "ZIF_") ||
		strings.HasPrefix(program, "YIF_") ||
		strings.HasPrefix(program, "LCL_") ||
		strings.HasPrefix(program, "LIF_") ||
		strings.Contains(program, "/CL_") ||
		strings.Contains(program, "/IF_")

	if !isClass {
		return program
	}

	// Pad to 30 chars with '=' and add 'CP' suffix
	// Total length: 30 + 2 = 32 (standard ABAP class pool naming)
	if len(program) < 30 {
		padding := 30 - len(program)
		program = program + strings.Repeat("=", padding) + "CP"
	}

	return program
}

func (s *Server) handleGetBreakpoints(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// RFC mode: use ADT REST API
	if s.isRfcMode() {
		resp, err := s.adtClient.GetExternalBreakpoints(ctx, s.config.Username)
		if err != nil {
			return newToolResultError(fmt.Sprintf("GetBreakpoints failed (RFC mode): %v", err)), nil
		}
		if len(resp.Breakpoints) == 0 {
			return mcp.NewToolResultText("No breakpoints are currently set."), nil
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "Active Breakpoints (%d):\n\n", len(resp.Breakpoints))
		for i, bp := range resp.Breakpoints {
			fmt.Fprintf(&sb, "%d. ID: %s\n   Kind: %s\n", i+1, bp.ID, bp.Kind)
			if bp.URI != "" {
				fmt.Fprintf(&sb, "   URI: %s\n", bp.URI)
			}
			if bp.Line > 0 {
				fmt.Fprintf(&sb, "   Line: %d\n", bp.Line)
			}
			sb.WriteString("\n")
		}
		return mcp.NewToolResultText(sb.String()), nil
	}

	// HTTP mode: WebSocket
	if err := s.ensureDebugWSClient(ctx); err != nil {
		return newToolResultError(fmt.Sprintf("Failed to connect to ZADT_VSP WebSocket: %v", err)), nil
	}

	breakpoints, err := s.debugWSClient.GetBreakpoints(ctx)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GetBreakpoints failed: %v", err)), nil
	}

	if len(breakpoints) == 0 {
		return mcp.NewToolResultText("No breakpoints are currently set."), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Active Breakpoints (%d):\n\n", len(breakpoints))
	for i, bp := range breakpoints {
		fmt.Fprintf(&sb, "%d. ID: %v\n", i+1, bp["id"])
		if kind, ok := bp["kind"]; ok {
			fmt.Fprintf(&sb, "   Kind: %v\n", kind)
		}
		if uri, ok := bp["uri"]; ok {
			fmt.Fprintf(&sb, "   URI: %v\n", uri)
		}
		if line, ok := bp["line"]; ok {
			fmt.Fprintf(&sb, "   Line: %v\n", line)
		}
		sb.WriteString("\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleDeleteBreakpoint(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	bpID, ok := request.Params.Arguments["breakpoint_id"].(string)
	if !ok || bpID == "" {
		return newToolResultError("breakpoint_id is required"), nil
	}

	// RFC mode: use ADT REST API
	if s.isRfcMode() {
		if err := s.adtClient.DeleteExternalBreakpoint(ctx, bpID, s.config.Username); err != nil {
			return newToolResultError(fmt.Sprintf("DeleteBreakpoint failed (RFC mode): %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Breakpoint %s deleted successfully.", bpID)), nil
	}

	// HTTP mode: WebSocket
	if err := s.ensureDebugWSClient(ctx); err != nil {
		return newToolResultError(fmt.Sprintf("Failed to connect to ZADT_VSP WebSocket: %v", err)), nil
	}

	if err := s.debugWSClient.DeleteBreakpoint(ctx, bpID); err != nil {
		return newToolResultError(fmt.Sprintf("DeleteBreakpoint failed: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Breakpoint %s deleted successfully.", bpID)), nil
}

func (s *Server) handleCallRFC(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	function, ok := request.Params.Arguments["function"].(string)
	if !ok || function == "" {
		return newToolResultError("function is required"), nil
	}

	// Parse params if provided
	var rawParams map[string]interface{}
	if paramsStr, ok := request.Params.Arguments["params"].(string); ok && paramsStr != "" {
		if err := json.Unmarshal([]byte(paramsStr), &rawParams); err != nil {
			return newToolResultError(fmt.Sprintf("Invalid params JSON: %v", err)), nil
		}
	}

	// RFC mode: use sidecar's direct /rfc-call endpoint (JCo)
	if s.isRfcMode() {
		if s.sidecar == nil {
			return newToolResultError("JCo sidecar not available"), nil
		}
		result, err := s.sidecar.CallRFC(ctx, function, rawParams)
		if err != nil {
			return newToolResultError(fmt.Sprintf("CallRFC failed: %v", err)), nil
		}
		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(fmt.Sprintf("RFC call completed (via JCo sidecar).\n\nFunction: %s\n\nResult:\n%s", function, string(resultJSON))), nil
	}

	// HTTP mode: use WebSocket (ZADT_VSP)
	params := make(map[string]string)
	for k, v := range rawParams {
		params[k] = fmt.Sprintf("%v", v)
	}

	if err := s.ensureDebugWSClient(ctx); err != nil {
		return newToolResultError(fmt.Sprintf("Failed to connect to ZADT_VSP WebSocket: %v. Ensure ZADT_VSP is deployed and SAPC/SICF are configured.", err)), nil
	}

	result, err := s.debugWSClient.CallRFC(ctx, function, params)
	if err != nil {
		return newToolResultError(fmt.Sprintf("CallRFC failed: %v", err)), nil
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("RFC call completed.\n\nFunction: %s\nSubrc: %d\n\nResult:\n%s", function, result.Subrc, string(resultJSON))), nil
}
