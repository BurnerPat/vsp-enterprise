// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_codeintel.go contains handlers for code intelligence operations
// (find definition, references, completion, pretty print, type hierarchy, etc.).
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// CodeIntelToolDefs returns tool definitions for code intelligence tools.
func CodeIntelToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("FindDefinition",
			mcp.WithDescription("Navigate to the definition of a symbol at a given position in source code"),
			mcp.WithString("source_url", mcp.Required(), mcp.Description("ADT URL of the source file (e.g., /sap/bc/adt/programs/programs/ZTEST/source/main)")),
			mcp.WithString("source", mcp.Required(), mcp.Description("Full source code of the file")),
			mcp.WithNumber("line", mcp.Required(), mcp.Description("Line number (1-based)")),
			mcp.WithNumber("start_column", mcp.Required(), mcp.Description("Start column of the symbol (1-based)")),
			mcp.WithNumber("end_column", mcp.Required(), mcp.Description("End column of the symbol (1-based)")),
			mcp.WithBoolean("implementation", mcp.Description("Navigate to implementation instead of definition (default: false)")),
			mcp.WithString("main_program", mcp.Description("Main program for includes (optional)")),
		), Handler: HandleFindDefinition, ReadOnly: true, Focused: true},

		{Tool: mcp.NewTool("FindReferences",
			mcp.WithDescription("Find all references to an ABAP object or symbol"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST)")),
			mcp.WithNumber("line", mcp.Description("Line number for position-based reference search (1-based, optional)")),
			mcp.WithNumber("column", mcp.Description("Column number for position-based reference search (1-based, optional)")),
		), Handler: HandleFindReferences, ReadOnly: true, Focused: true},

		{Tool: mcp.NewTool("CodeCompletion",
			mcp.WithDescription("Get code completion suggestions at a position in source code"),
			mcp.WithString("source_url", mcp.Required(), mcp.Description("ADT URL of the source file (e.g., /sap/bc/adt/programs/programs/ZTEST/source/main)")),
			mcp.WithString("source", mcp.Required(), mcp.Description("Full source code of the file")),
			mcp.WithNumber("line", mcp.Required(), mcp.Description("Line number (1-based)")),
			mcp.WithNumber("column", mcp.Required(), mcp.Description("Column number (1-based)")),
		), Handler: HandleCodeCompletion, ReadOnly: true},

		{Tool: mcp.NewTool("GetTypeHierarchy",
			mcp.WithDescription("Get the type hierarchy (supertypes or subtypes) for a class/interface"),
			mcp.WithString("source_url", mcp.Required(), mcp.Description("ADT URL of the source file")),
			mcp.WithString("source", mcp.Required(), mcp.Description("Full source code of the file")),
			mcp.WithNumber("line", mcp.Required(), mcp.Description("Line number (1-based)")),
			mcp.WithNumber("column", mcp.Required(), mcp.Description("Column number (1-based)")),
			mcp.WithBoolean("super_types", mcp.Description("Get supertypes instead of subtypes (default: false = subtypes)")),
		), Handler: HandleGetTypeHierarchy, ReadOnly: true},

		{Tool: mcp.NewTool("GetClassComponents",
			mcp.WithDescription("Get the structure of a class - lists all methods, attributes, events, and other components with their visibility and properties"),
			mcp.WithString("class_url", mcp.Required(), mcp.Description("ADT URL of the class (e.g., /sap/bc/adt/oo/classes/ZCL_TEST)")),
		), Handler: HandleGetClassComponents, ReadOnly: true},
	}
}

// --- Code Intelligence Handlers ---

func HandleFindDefinition(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceURL, ok := request.GetArguments()["source_url"].(string)
	if !ok || sourceURL == "" {
		return types.ErrorResult("source_url is required"), nil
	}

	source, ok := request.GetArguments()["source"].(string)
	if !ok || source == "" {
		return types.ErrorResult("source is required"), nil
	}

	lineF, ok := request.GetArguments()["line"].(float64)
	if !ok {
		return types.ErrorResult("line is required"), nil
	}
	line := int(lineF)

	startColF, ok := request.GetArguments()["start_column"].(float64)
	if !ok {
		return types.ErrorResult("start_column is required"), nil
	}
	startCol := int(startColF)

	endColF, ok := request.GetArguments()["end_column"].(float64)
	if !ok {
		return types.ErrorResult("end_column is required"), nil
	}
	endCol := int(endColF)

	implementation := false
	if impl, ok := request.GetArguments()["implementation"].(bool); ok {
		implementation = impl
	}

	mainProgram := ""
	if mp, ok := request.GetArguments()["main_program"].(string); ok {
		mainProgram = mp
	}

	result, err := sys.ADT().FindDefinition(ctx, sourceURL, source, line, startCol, endCol, implementation, mainProgram)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("FindDefinition failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleFindReferences(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.GetArguments()["object_url"].(string)
	if !ok || objectURL == "" {
		return types.ErrorResult("object_url is required"), nil
	}

	line := 0
	column := 0
	if lineF, ok := request.GetArguments()["line"].(float64); ok {
		line = int(lineF)
	}
	if colF, ok := request.GetArguments()["column"].(float64); ok {
		column = int(colF)
	}

	results, err := sys.ADT().FindReferences(ctx, objectURL, line, column)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("FindReferences failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleCodeCompletion(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceURL, ok := request.GetArguments()["source_url"].(string)
	if !ok || sourceURL == "" {
		return types.ErrorResult("source_url is required"), nil
	}

	source, ok := request.GetArguments()["source"].(string)
	if !ok || source == "" {
		return types.ErrorResult("source is required"), nil
	}

	lineF, ok := request.GetArguments()["line"].(float64)
	if !ok {
		return types.ErrorResult("line is required"), nil
	}
	line := int(lineF)

	colF, ok := request.GetArguments()["column"].(float64)
	if !ok {
		return types.ErrorResult("column is required"), nil
	}
	column := int(colF)

	proposals, err := sys.ADT().CodeCompletion(ctx, sourceURL, source, line, column)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("CodeCompletion failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(proposals, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleGetTypeHierarchy(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceURL, ok := request.GetArguments()["source_url"].(string)
	if !ok || sourceURL == "" {
		return types.ErrorResult("source_url is required"), nil
	}

	source, ok := request.GetArguments()["source"].(string)
	if !ok || source == "" {
		return types.ErrorResult("source is required"), nil
	}

	lineF, ok := request.GetArguments()["line"].(float64)
	if !ok {
		return types.ErrorResult("line is required"), nil
	}
	line := int(lineF)

	colF, ok := request.GetArguments()["column"].(float64)
	if !ok {
		return types.ErrorResult("column is required"), nil
	}
	column := int(colF)

	superTypes := false
	if st, ok := request.GetArguments()["super_types"].(bool); ok {
		superTypes = st
	}

	hierarchy, err := sys.ADT().GetTypeHierarchy(ctx, sourceURL, source, line, column, superTypes)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetTypeHierarchy failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(hierarchy, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleGetClassComponents(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	classURL, ok := request.GetArguments()["class_url"].(string)
	if !ok || classURL == "" {
		return types.ErrorResult("class_url is required"), nil
	}

	components, err := sys.ADT().GetClassComponents(ctx, classURL)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetClassComponents failed: %v", err)), nil
	}

	// Format output with summary
	output := formatClassComponents(components)
	return mcp.NewToolResultText(output), nil
}

// formatClassComponents creates a readable summary of class components
func formatClassComponents(comp *adt.ClassComponent) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Class: %s (%s)\n\n", comp.Name, comp.Type)

	// Group components by type
	methods := []adt.ClassComponent{}
	attributes := []adt.ClassComponent{}
	events := []adt.ClassComponent{}
	types := []adt.ClassComponent{}
	others := []adt.ClassComponent{}

	for _, c := range comp.Components {
		switch {
		case strings.Contains(c.Type, "METH"):
			methods = append(methods, c)
		case strings.Contains(c.Type, "DATA") || strings.Contains(c.Type, "ATTR"):
			attributes = append(attributes, c)
		case strings.Contains(c.Type, "EVNT") || strings.Contains(c.Type, "EVENT"):
			events = append(events, c)
		case strings.Contains(c.Type, "TYPE"):
			types = append(types, c)
		default:
			others = append(others, c)
		}
	}

	if len(methods) > 0 {
		fmt.Fprintf(&sb, "## Methods (%d)\n", len(methods))
		for _, m := range methods {
			flags := componentFlags(m)
			fmt.Fprintf(&sb, "  - %s [%s]%s\n", m.Name, m.Visibility, flags)
			if m.Description != "" {
				fmt.Fprintf(&sb, "    %s\n", m.Description)
			}
		}
		sb.WriteString("\n")
	}

	if len(attributes) > 0 {
		fmt.Fprintf(&sb, "## Attributes (%d)\n", len(attributes))
		for _, a := range attributes {
			flags := componentFlags(a)
			fmt.Fprintf(&sb, "  - %s [%s]%s\n", a.Name, a.Visibility, flags)
		}
		sb.WriteString("\n")
	}

	if len(events) > 0 {
		fmt.Fprintf(&sb, "## Events (%d)\n", len(events))
		for _, e := range events {
			fmt.Fprintf(&sb, "  - %s [%s]\n", e.Name, e.Visibility)
		}
		sb.WriteString("\n")
	}

	if len(types) > 0 {
		fmt.Fprintf(&sb, "## Types (%d)\n", len(types))
		for _, t := range types {
			fmt.Fprintf(&sb, "  - %s [%s]\n", t.Name, t.Visibility)
		}
		sb.WriteString("\n")
	}

	if len(others) > 0 {
		fmt.Fprintf(&sb, "## Other Components (%d)\n", len(others))
		for _, o := range others {
			fmt.Fprintf(&sb, "  - %s (%s) [%s]\n", o.Name, o.Type, o.Visibility)
		}
	}

	return sb.String()
}

func componentFlags(c adt.ClassComponent) string {
	var flags []string
	if c.IsStatic {
		flags = append(flags, "static")
	}
	if c.IsFinal {
		flags = append(flags, "final")
	}
	if c.IsAbstract {
		flags = append(flags, "abstract")
	}
	if c.ReadOnly {
		flags = append(flags, "read-only")
	}
	if c.Constant {
		flags = append(flags, "constant")
	}
	if len(flags) > 0 {
		return " " + strings.Join(flags, ", ")
	}
	return ""
}
