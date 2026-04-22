// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_debugger_legacy.go contains handlers for legacy REST-based debugging.
// These use REST API which works for Listen/Attach/Step but not for breakpoints.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// DebuggerLegacyToolDefs returns tool definitions for legacy REST-based debugging.
func DebuggerLegacyToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("DebuggerListen",
			mcp.WithDescription("Start a legacy ADT debugger listener. Block until a program hits a breakpoint or timeout."),
			mcp.WithString("user", mcp.Description("Filter by user (default: connection user)")),
			mcp.WithNumber("timeout", mcp.Description("Timeout in seconds (default: 60, max: 240)")),
		), Handler: HandleDebuggerListen, Endpoints: []string{"/sap/bc/adt/debugger"}},

		{Tool: mcp.NewTool("DebuggerAttach",
			mcp.WithDescription("Attach to a caught debuggee session."),
			mcp.WithString("debuggee_id", mcp.Required(), mcp.Description("Debuggee ID from DebuggerListen")),
			mcp.WithString("user", mcp.Description("Filter by user")),
		), Handler: HandleDebuggerAttach, Endpoints: []string{"/sap/bc/adt/debugger"}},

		{Tool: mcp.NewTool("DebuggerDetach",
			mcp.WithDescription("Detach from the current debug session."),
		), Handler: HandleDebuggerDetach, Endpoints: []string{"/sap/bc/adt/debugger"}},

		{Tool: mcp.NewTool("DebuggerStep",
			mcp.WithDescription("Execute a debug step."),
			mcp.WithString("step_type", mcp.Required(), mcp.Description("stepInto, stepOver, stepReturn, stepContinue, stepRunToLine, stepJumpToLine")),
			mcp.WithString("uri", mcp.Description("ADT URI for runToLine or jumpToLine")),
		), Handler: HandleDebuggerStep, Endpoints: []string{"/sap/bc/adt/debugger"}},

		{Tool: mcp.NewTool("DebuggerGetStack",
			mcp.WithDescription("Get the current call stack."),
		), Handler: HandleDebuggerGetStack, ReadOnly: true, Endpoints: []string{"/sap/bc/adt/debugger"}},

		{Tool: mcp.NewTool("DebuggerGetVariables",
			mcp.WithDescription("Read variable values."),
			mcp.WithArray("variable_ids", mcp.Description("List of variable IDs to read (optional, default: @ROOT)"), mcp.WithStringItems()),
		), Handler: HandleDebuggerGetVariables, ReadOnly: true, Endpoints: []string{"/sap/bc/adt/debugger"}},

		{Tool: mcp.NewTool("DebuggerSetVariable",
			mcp.WithDescription("Modify a variable value."),
			mcp.WithString("variable_name", mcp.Required(), mcp.Description("Name of the variable")),
			mcp.WithString("value", mcp.Required(), mcp.Description("New value")),
		), Handler: HandleDebuggerSetVariable, Endpoints: []string{"/sap/bc/adt/debugger"}},
	}
}

// --- Legacy REST-based Debugger Handlers (fallback) ---

func HandleDebuggerListen(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Fallback to empty string if username not available through any means
	user, _ := request.GetArguments()["user"].(string)

	timeout := 60 // default
	if t, ok := request.GetArguments()["timeout"].(float64); ok && t > 0 {
		timeout = int(t)
		if timeout > 240 {
			timeout = 240 // max 240 seconds
		}
	}

	result, err := sys.ADT().DebuggerListen(ctx, &adt.ListenOptions{
		DebuggingMode:  adt.DebuggingModeUser,
		User:           user,
		TimeoutSeconds: timeout,
	})

	if err != nil {
		return types.ErrorResult(fmt.Sprintf("DebuggerListen failed: %v", err)), nil
	}

	if result.TimedOut {
		return mcp.NewToolResultText("Listener timed out - no debuggee hit a breakpoint within the timeout period."), nil
	}

	if result.Conflict != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Listener conflict detected: %s (user: %s)",
			result.Conflict.ConflictText, result.Conflict.IdeUser)), nil
	}

	if result.Debuggee != nil {
		var sb strings.Builder
		sb.WriteString("Debuggee caught!\n\n")
		fmt.Fprintf(&sb, "Debuggee ID: %s\n", result.Debuggee.ID)
		fmt.Fprintf(&sb, "User: %s\n", result.Debuggee.User)
		fmt.Fprintf(&sb, "Program: %s\n", result.Debuggee.Program)
		fmt.Fprintf(&sb, "Include: %s\n", result.Debuggee.Include)
		fmt.Fprintf(&sb, "Line: %d\n", result.Debuggee.Line)
		fmt.Fprintf(&sb, "Kind: %s\n", result.Debuggee.Kind)
		fmt.Fprintf(&sb, "Attachable: %v\n", result.Debuggee.IsAttachable)
		fmt.Fprintf(&sb, "App Server: %s\n", result.Debuggee.AppServer)
		sb.WriteString("\nUse DebuggerAttach with the debuggee_id to attach to this session.")
		return mcp.NewToolResultText(sb.String()), nil
	}

	return mcp.NewToolResultText("Listener returned with no result."), nil
}

func HandleDebuggerAttach(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	debuggeeID, ok := request.GetArguments()["debuggee_id"].(string)
	if !ok || debuggeeID == "" {
		return types.ErrorResult("debuggee_id is required"), nil
	}

	user, _ := request.GetArguments()["user"].(string)

	result, err := sys.ADT().DebuggerAttach(ctx, debuggeeID, user)

	if err != nil {
		if strings.Contains(err.Error(), "invalidDebuggee") {
			return types.ErrorResult("Debuggee expired - the program finished before we could attach. Try again with a fresh run."), nil
		}
		return types.ErrorResult(fmt.Sprintf("DebuggerAttach failed: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString("Successfully attached to debuggee!\n\n")
	fmt.Fprintf(&sb, "Debug Session ID: %s\n", result.DebugSessionID)
	fmt.Fprintf(&sb, "Process ID: %d\n", result.ProcessID)
	fmt.Fprintf(&sb, "Server: %s\n", result.ServerName)
	fmt.Fprintf(&sb, "Stepping Possible: %v\n", result.IsSteppingPossible)
	fmt.Fprintf(&sb, "Termination Possible: %v\n", result.IsTerminationPossible)

	if len(result.ReachedBreakpoints) > 0 {
		sb.WriteString("\nReached Breakpoints:\n")
		for _, bp := range result.ReachedBreakpoints {
			fmt.Fprintf(&sb, "  - ID: %s (kind: %s)\n", bp.ID, bp.Kind)
		}
	}

	if len(result.Actions) > 0 {
		sb.WriteString("\nAvailable Actions:\n")
		for _, action := range result.Actions {
			fmt.Fprintf(&sb, "  - %s: %s\n", action.Name, action.Title)
		}
	}

	sb.WriteString("\nDebugger Tools:")
	sb.WriteString("\n  - DebuggerStep: stepInto, stepOver, stepReturn, stepContinue, stepRunToLine, stepJumpToLine")
	sb.WriteString("\n  - DebuggerGetStack: view the call stack")
	sb.WriteString("\n  - DebuggerGetVariables: read variable values (use '@ROOT' for all top-level)")
	sb.WriteString("\n  - DebuggerSetVariable: modify a variable value")
	sb.WriteString("\n  - DebuggerDetach: end the debug session")
	return mcp.NewToolResultText(sb.String()), nil
}

func HandleDebuggerDetach(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	err := sys.ADT().DebuggerDetach(ctx)

	if err != nil {
		return types.ErrorResult(fmt.Sprintf("DebuggerDetach failed: %v", err)), nil
	}

	return mcp.NewToolResultText("Successfully detached from debug session."), nil
}

func HandleDebuggerStep(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stepTypeStr, ok := request.GetArguments()["step_type"].(string)
	if !ok || stepTypeStr == "" {
		return types.ErrorResult("step_type is required"), nil
	}

	// Map string to step type
	var stepType adt.DebugStepType
	switch stepTypeStr {
	case "stepInto":
		stepType = adt.DebugStepInto
	case "stepOver":
		stepType = adt.DebugStepOver
	case "stepReturn":
		stepType = adt.DebugStepReturn
	case "stepContinue":
		stepType = adt.DebugStepContinue
	case "stepRunToLine":
		stepType = adt.DebugStepRunToLine
	case "stepJumpToLine":
		stepType = adt.DebugStepJumpToLine
	default:
		return types.ErrorResult(fmt.Sprintf("Invalid step_type: %s. Valid values: stepInto, stepOver, stepReturn, stepContinue, stepRunToLine, stepJumpToLine", stepTypeStr)), nil
	}

	uri, _ := request.GetArguments()["uri"].(string)

	result, err := sys.ADT().DebuggerStep(ctx, stepType, uri)

	if err != nil {
		// debuggeeEnded is normal when stepContinue runs the program to completion
		if strings.Contains(err.Error(), "debuggeeEnded") {
			// Reset connection session to prevent "already attached" on next debug run
			sys.ADT().DebuggerResetSession()
			return mcp.NewToolResultText("Program execution completed. The debuggee has ended normally."), nil
		}
		return types.ErrorResult(fmt.Sprintf("DebuggerStep failed: %v", err)), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Step '%s' executed.\n\n", stepTypeStr)
	fmt.Fprintf(&sb, "Session: %s\n", result.DebugSessionID)
	fmt.Fprintf(&sb, "Debuggee Changed: %v\n", result.IsDebuggeeChanged)
	fmt.Fprintf(&sb, "Stepping Possible: %v\n", result.IsSteppingPossible)

	if len(result.ReachedBreakpoints) > 0 {
		sb.WriteString("\nReached Breakpoints:\n")
		for _, bp := range result.ReachedBreakpoints {
			fmt.Fprintf(&sb, "  - ID: %s (kind: %s)\n", bp.ID, bp.Kind)
		}
	}

	sb.WriteString("\nUse DebuggerGetStack to see current position, DebuggerGetVariables to inspect variables.")
	return mcp.NewToolResultText(sb.String()), nil
}

func HandleDebuggerGetStack(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := sys.ADT().DebuggerGetStack(ctx, true)

	if err != nil {
		return types.ErrorResult(fmt.Sprintf("DebuggerGetStack failed: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString("Call Stack:\n\n")
	fmt.Fprintf(&sb, "Server: %s\n", result.ServerName)
	fmt.Fprintf(&sb, "Current Stack Index: %d\n\n", result.DebugCursorStackIndex)

	for i, entry := range result.Stack {
		marker := "  "
		if entry.StackPosition == result.DebugCursorStackIndex {
			marker = "→ "
		}
		fmt.Fprintf(&sb, "%s[%d] %s::%s (line %d)\n",
			marker, entry.StackPosition, entry.ProgramName, entry.EventName, entry.Line)
		fmt.Fprintf(&sb, "      Type: %s, Include: %s\n", entry.EventType, entry.IncludeName)
		if entry.SystemProgram {
			sb.WriteString("      (system program)\n")
		}
		if i < len(result.Stack)-1 {
			sb.WriteString("\n")
		}
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func HandleDebuggerGetVariables(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse variable_ids from request
	var variableIDs []string

	if ids, ok := request.GetArguments()["variable_ids"].([]interface{}); ok {
		for _, id := range ids {
			if s, ok := id.(string); ok {
				variableIDs = append(variableIDs, s)
			}
		}
	}

	// Default to @ROOT if no IDs specified
	if len(variableIDs) == 0 {
		variableIDs = []string{"@ROOT"}
	}

	// If @ROOT is requested, use GetChildVariables for top-level vars
	if len(variableIDs) == 1 && variableIDs[0] == "@ROOT" {
		result, err := sys.ADT().DebuggerGetChildVariables(ctx, []string{"@ROOT", "@DATAAGING"})
		if err != nil {
			return types.ErrorResult(fmt.Sprintf("DebuggerGetVariables failed: %v", err)), nil
		}

		var sb strings.Builder
		sb.WriteString("Variables:\n\n")

		for _, v := range result.Variables {
			fmt.Fprintf(&sb, "%s: %s = %s\n", v.Name, v.DeclaredTypeName, v.Value)
			fmt.Fprintf(&sb, "  MetaType: %s, Kind: %s\n", v.MetaType, v.Kind)
			if v.IsComplexType() {
				fmt.Fprintf(&sb, "  (complex type - use variable ID '%s' to expand)\n", v.ID)
			}
		}

		return mcp.NewToolResultText(sb.String()), nil
	}

	// Get specific variables
	result, err := sys.ADT().DebuggerGetVariables(ctx, variableIDs)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("DebuggerGetVariables failed: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString("Variables:\n\n")

	for _, v := range result {
		fmt.Fprintf(&sb, "%s: %s = %s\n", v.Name, v.DeclaredTypeName, v.Value)
		fmt.Fprintf(&sb, "  ID: %s\n", v.ID)
		fmt.Fprintf(&sb, "  MetaType: %s, Kind: %s\n", v.MetaType, v.Kind)
		if v.HexValue != "" {
			fmt.Fprintf(&sb, "  Hex: %s\n", v.HexValue)
		}
		if v.TableLines > 0 {
			fmt.Fprintf(&sb, "  Table Lines: %d\n", v.TableLines)
		}
		if v.IsComplexType() {
			sb.WriteString("  (complex type - expandable)\n")
		}
		sb.WriteString("\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func HandleDebuggerSetVariable(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	variableName, ok := request.GetArguments()["variable_name"].(string)
	if !ok || variableName == "" {
		return types.ErrorResult("variable_name is required"), nil
	}
	value, ok := request.GetArguments()["value"].(string)
	if !ok {
		return types.ErrorResult("value is required"), nil
	}

	result, err := sys.ADT().DebuggerSetVariableValue(ctx, variableName, value)

	if err != nil {
		return types.ErrorResult(fmt.Sprintf("DebuggerSetVariable failed: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Variable %s set to %s\n\n%s", variableName, value, result)), nil
}
