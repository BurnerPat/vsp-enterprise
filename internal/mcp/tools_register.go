// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tools_register.go contains the tool registration helpers and filter logic.
// Individual tool definitions live alongside their handlers in the tools/ package.
package mcp

import (
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

// paramMapper transforms universal-mode params into handler-specific params.
// objectType and objectName come from parseTarget("TYPE NAME").
type paramMapper func(objectType, objectName string, params map[string]any) map[string]any

// universalRoute describes how a tool is accessible via the single SAP(action, target, params) tool.
// A tool can have multiple routes (e.g., accessible via different action/target combinations).
type UniversalRoute struct {
	Action     string      // universal-mode action: "read", "edit", "create", "delete", "search", "query", "grep", "test", "analyze", "debug", "system"
	TargetType string      // match objectType from target (e.g., "PROG", "INFO"). Empty = don't match on targetType.
	ParamsType string      // match params["type"] (e.g., "list_transports"). Empty = don't match on params.type.
	MapArgs    paramMapper // optional: transform params before calling handler. nil = pass through.
}

// ToolDef is a self-contained, declarative tool definition.
// All metadata lives here — no separate focused/group maps needed.
type ToolDef struct {
	Tool     mcp.Tool
	Handler  types.ToolHandlerFunc
	AlwaysOn bool     // if true, registered regardless of mode/groups/config
	ReadOnly bool     // tool only reads data, never modifies the SAP system
	Focused  bool     // included in "focused" mode (default mode)
	Groups   []string // group codes for --disabled-groups (e.g., "D", "H", "X")

	// Universal mode routing (optional).
	// Describes how this tool is reachable via SAP(action, target, params).
	// If empty, the tool is only available as an individual named tool.
	Routes []UniversalRoute
}

// buildShouldRegister creates a filter function.
// Focused set and group membership are derived from ToolDef metadata.
func buildShouldRegister(mode string, disabledGroups string, toolsConfig map[string]bool, defs []ToolDef) func(string) bool {
	// Derive focused set from metadata
	focusedTools := make(map[string]bool)
	for _, td := range defs {
		if td.Focused || td.AlwaysOn {
			focusedTools[td.Tool.Name] = true
		}
	}

	// Derive group membership from metadata
	groupTools := make(map[string]map[string]bool) // group code → tool names
	for _, td := range defs {
		for _, g := range td.Groups {
			if groupTools[g] == nil {
				groupTools[g] = make(map[string]bool)
			}
			groupTools[g][td.Tool.Name] = true
		}
	}
	// "U" alias for "5" (UI5)
	groupTools["U"] = groupTools["5"]

	// Build disabled set from --disabled-groups flag
	disabledTools := make(map[string]bool)
	for _, code := range strings.ToUpper(disabledGroups) {
		if tools, ok := groupTools[string(code)]; ok {
			for tool := range tools {
				disabledTools[tool] = true
			}
		}
	}

	return func(toolName string) bool {
		if toolsConfig != nil {
			if enabled, exists := toolsConfig[toolName]; exists {
				return enabled
			}
		}
		if disabledTools[toolName] {
			return false
		}
		if mode == "expert" {
			return true
		}
		return focusedTools[toolName]
	}
}
