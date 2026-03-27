// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tools_register.go contains the tool registration loop and the toolDef type.
// Individual tool definitions live alongside their handlers in tool_*.go files.
package mcp

import (
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// paramMapper transforms universal-mode params into handler-specific params.
// objectType and objectName come from parseTarget("TYPE NAME").
type paramMapper func(objectType, objectName string, params map[string]any) map[string]any

// universalRoute describes how a tool is accessible via the single SAP(action, target, params) tool.
// A tool can have multiple routes (e.g., accessible via different action/target combinations).
type universalRoute struct {
	action     string      // universal-mode action: "read", "edit", "create", "delete", "search", "query", "grep", "test", "analyze", "debug", "system"
	targetType string      // match objectType from target (e.g., "PROG", "INFO"). Empty = don't match on targetType.
	paramsType string      // match params["type"] (e.g., "list_transports"). Empty = don't match on params.type.
	mapArgs    paramMapper // optional: transform params before calling handler. nil = pass through.
}

// toolDef is a self-contained, declarative tool definition.
// All metadata lives here — no separate focused/group maps needed.
type toolDef struct {
	tool     mcp.Tool
	handler  server.ToolHandlerFunc
	alwaysOn bool     // if true, registered regardless of mode/groups/config
	readOnly bool     // tool only reads data, never modifies the SAP system
	focused  bool     // included in "focused" mode (default mode)
	groups   []string // group codes for --disabled-groups (e.g., "D", "H", "X")

	// Universal mode routing (optional).
	// Describes how this tool is reachable via SAP(action, target, params).
	// If empty, the tool is only available as an individual named tool.
	routes []universalRoute
}

// registerTools registers ADT tools with the MCP server based on mode, disabled groups, and granular config.
func (s *Server) registerTools(mode string, disabledGroups string, toolsConfig map[string]bool) {
	// Hyperfocused mode: single universal SAP tool
	if mode == "hyperfocused" {
		s.registerUniversalTool()
		return
	}

	defs := s.allToolDefs()
	shouldRegister := buildShouldRegister(mode, disabledGroups, toolsConfig, defs)

	// Register tools that pass the filter
	for _, td := range defs {
		if td.alwaysOn || shouldRegister(td.tool.Name) {
			s.addTool(td.tool, td.handler)
		}
	}

	// Register tool aliases for common operations
	s.registerToolAliases(shouldRegister)
}

// buildShouldRegister creates a filter function.
// Focused set and group membership are derived from toolDef metadata.
func buildShouldRegister(mode string, disabledGroups string, toolsConfig map[string]bool, defs []toolDef) func(string) bool {
	// Derive focused set from metadata
	focusedTools := make(map[string]bool)
	for _, td := range defs {
		if td.focused || td.alwaysOn {
			focusedTools[td.tool.Name] = true
		}
	}

	// Derive group membership from metadata
	groupTools := make(map[string]map[string]bool) // group code → tool names
	for _, td := range defs {
		for _, g := range td.groups {
			if groupTools[g] == nil {
				groupTools[g] = make(map[string]bool)
			}
			groupTools[g][td.tool.Name] = true
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

// allToolDefs collects tool definitions from all handler groups.
func (s *Server) allToolDefs() []toolDef {
	var defs []toolDef
	defs = append(defs, s.unifiedToolDefs()...)
	defs = append(defs, s.readToolDefs()...)
	defs = append(defs, s.systemToolDefs()...)
	defs = append(defs, s.analysisToolDefs()...)
	defs = append(defs, s.dumpToolDefs()...)
	defs = append(defs, s.traceToolDefs()...)
	defs = append(defs, s.sqlTraceToolDefs()...)
	defs = append(defs, s.debuggerToolDefs()...)
	defs = append(defs, s.searchToolDefs()...)
	defs = append(defs, s.devToolDefs()...)
	defs = append(defs, s.crudToolDefs()...)
	defs = append(defs, s.classIncludeToolDefs()...)
	defs = append(defs, s.workflowToolDefs()...)
	defs = append(defs, s.fileToolDefs()...)
	defs = append(defs, s.editToolDefs()...)
	defs = append(defs, s.grepToolDefs()...)
	defs = append(defs, s.codeIntelToolDefs()...)
	defs = append(defs, s.ui5ToolDefs()...)
	defs = append(defs, s.amdpToolDefs()...)
	defs = append(defs, s.transportToolDefs()...)
	defs = append(defs, s.gitToolDefs()...)
	defs = append(defs, s.reportToolDefs()...)
	return defs
}
