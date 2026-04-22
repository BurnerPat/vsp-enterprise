package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

// UniversalToolDefs returns the definition for the single-tool SAP "universal" mode.
func UniversalToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{
			AlwaysOn: true,
			Tool: mcp.NewTool("SAP",
				mcp.WithDescription(`Universal SAP tool. Use SAP(action="help") for full documentation.

Actions: read, edit, create, delete, search, query, grep, test, analyze, debug, system, help
Target: "TYPE NAME" (e.g. "CLAS ZCL_TEST", "PROG ZREPORT")
Params: action-specific parameters as JSON object`),
				mcp.WithString("action",
					mcp.Required(),
					mcp.Description("Action to perform: read, edit, create, delete, search, query, grep, test, analyze, debug, system, help"),
				),
				mcp.WithString("target",
					mcp.Description("Target object as 'TYPE NAME' (e.g. 'CLAS ZCL_TEST', 'PROG ZREPORT'). Some actions don't need a target."),
				),
				mcp.WithObject("params",
					mcp.Description("Action-specific parameters as a JSON object"),
				),
			),
			Handler: HandleUniversalTool,
		},
	}
}

// HandleUniversalTool dispatches universal SAP(action, target, params) calls.
// It iterates all tool definitions and matches against their declared routes.
func HandleUniversalTool(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action, _ := request.GetArguments()["action"].(string)
	if action == "" {
		return types.ErrorResult("action is required. Use SAP(action=\"help\") for documentation."), nil
	}
	action = strings.ToLower(action)

	target, _ := request.GetArguments()["target"].(string)

	// Extract params as map
	params := getObject(request.GetArguments(), "params")
	if params == nil {
		params = make(map[string]any)
	}

	// Help action
	if action == "help" {
		return HandleHelp(target), nil
	}

	// Parse target into type and name
	objectType, objectName := parseTarget(target)
	pType := getStringParam(params, "type")

	// Match against all tool definitions
	// NOTE: This requires access to all tool definitions.
	// In the new architecture, the Router has them, but here we only have the handler.
	// We need a way to get all definitions.
	// For now, we'll assume there's a GlobalAllToolDefs function or we inject it.
	// Actually, let's use a simpler approach: the router itself should probably handle the "universal" routing if it wants to be truly universal.
	// But the user wanted it here.
	// Let's assume we can call the tool def providers from here too, or we use a registry.

	allDefs := getAllToolDefs()

	for _, td := range allDefs {
		for _, r := range td.Routes {
			if r.Action != action {
				continue
			}
			// Match on TargetType (from SAP target string)
			if r.TargetType != "" && r.TargetType != objectType {
				continue
			}
			// Match on ParamsType (from params.type)
			if r.ParamsType != "" && r.ParamsType != pType {
				continue
			}
			// Route matched — build args and call handler
			args := params
			if r.MapArgs != nil {
				args = r.MapArgs(objectType, objectName, params)
			}

			// Construct a new request with mapped args
			newReq := request
			newReq.Params.Arguments = args

			return td.Handler(ctx, sys, newReq)
		}
	}

	// Nothing matched
	return types.ErrorResult(fmt.Sprintf("Unhandled action: %s (Target: %s %s)", action, objectType, objectName)), nil
}

func getAllToolDefs() []types.ToolDef {
	var defs []types.ToolDef
	defs = append(defs, SystemToolDefs()...)
	defs = append(defs, ReadToolDefs()...)
	defs = append(defs, UnifiedToolDefs()...)
	defs = append(defs, AnalysisToolDefs()...)
	defs = append(defs, ContextToolDefs()...)
	defs = append(defs, ATCToolDefs()...)
	defs = append(defs, ClassIncludeToolDefs()...)
	defs = append(defs, CodeIntelToolDefs()...)
	defs = append(defs, CRUDToolDefs()...)
	defs = append(defs, DevToolDefs()...)
	defs = append(defs, DumpToolDefs()...)
	defs = append(defs, FileToolDefs()...)
	defs = append(defs, GrepToolDefs()...)
	defs = append(defs, ServiceBindingToolDefs()...)
	defs = append(defs, SQLTraceToolDefs()...)
	defs = append(defs, TraceToolDefs()...)
	defs = append(defs, WorkflowToolDefs()...)
	defs = append(defs, DebuggerLegacyToolDefs()...)
	return defs
}

// parseTarget splits "TYPE NAME" into objectType and objectName, uppercasing both.
func parseTarget(target string) (objectType, objectName string) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", ""
	}
	parts := strings.SplitN(target, " ", 2)
	objectType = strings.ToUpper(strings.TrimSpace(parts[0]))
	if len(parts) > 1 {
		objectName = strings.ToUpper(strings.TrimSpace(parts[1]))
	}
	return
}

// getObject extracts a nested object (map[string]any) from args.
func getObject(args map[string]any, key string) map[string]any {
	if v, ok := args[key]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return nil
}

// getStringParam extracts a string value from a map.
func getStringParam(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func HandleHelp(target string) *mcp.CallToolResult {
	// Simple help for now
	return mcp.NewToolResultText("SAP Universal Tool Help\n\nActions: read, edit, create, delete, query, analyze, system\nTargets: CLAS, PROG, INTF, TABL, DEVC, etc.\n\nExample: SAP(action='read', target='CLAS ZCL_TEST')")
}
