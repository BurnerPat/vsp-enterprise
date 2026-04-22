package tools

import "github.com/oisee/vibing-steampunk/internal/mcp/types"

//goland:noinspection DuplicatedCode
func AllToolDefs() []types.ToolDef {
	var defs []types.ToolDef
	defs = append(defs, SystemToolDefs()...)
	defs = append(defs, ReadToolDefs()...)
	defs = append(defs, UnifiedToolDefs()...)
	defs = append(defs, GrepSourceToolDefs()...)
	defs = append(defs, FileSourceToolDefs()...)
	defs = append(defs, AnalysisToolDefs()...)
	defs = append(defs, TransportToolDefs()...)
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
