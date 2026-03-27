// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tools_register.go contains the tool registration loop and the toolDef type.
// Individual tool definitions live in their respective handlers_*.go files.
package mcp

import (
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// toolDef is a declarative tool definition pairing a schema with its handler.
type toolDef struct {
	tool     mcp.Tool
	handler  server.ToolHandlerFunc
	alwaysOn bool // if true, registered regardless of mode/groups/config
}

// registerTools registers ADT tools with the MCP server based on mode, disabled groups, and granular config.
func (s *Server) registerTools(mode string, disabledGroups string, toolsConfig map[string]bool) {
	// Hyperfocused mode: single universal SAP tool
	if mode == "hyperfocused" {
		s.registerUniversalTool()
		return
	}

	shouldRegister := buildShouldRegister(mode, disabledGroups, toolsConfig)

	// Collect all tool definitions and register those that pass the filter
	for _, td := range s.allToolDefs() {
		if td.alwaysOn || shouldRegister(td.tool.Name) {
			s.addTool(td.tool, td.handler)
		}
	}

	// Register tool aliases for common operations
	s.registerToolAliases(shouldRegister)
}

// buildShouldRegister creates a filter function based on mode, disabled groups, and granular config.
func buildShouldRegister(mode string, disabledGroups string, toolsConfig map[string]bool) func(string) bool {
	groups := toolGroups()
	focusedTools := focusedToolSet()

	disabledTools := make(map[string]bool)
	for _, code := range strings.ToUpper(disabledGroups) {
		if tools, ok := groups[string(code)]; ok {
			for _, tool := range tools {
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
	defs = append(defs, s.diagnosticsToolDefs()...)
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
