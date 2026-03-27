# Observations & Changes Since v2.12.5

**Date Range:** 2025-12-09 to 2025-12-22
**Starting Version:** v2.12.5 (EditSource Line Ending Fix)
**Current Version:** v2.15.0 (Phase 5 TAS-Style Debugging)

---

## Executive Summary

Since v2.12.5, the project has undergone a **major evolution** from basic ADT tooling to an **AI-powered debugging and RCA platform**. Key achievements:

- **6 releases** (v2.12.5 → v2.15.0)
- **18 research reports** documenting deep investigations
- **Complete WebSocket debugging infrastructure** (ZADT_VSP)
- **Lua scripting integration** for debug automation
- **Phase 5 implementation** including Force Replay and Variable History
- **Shift from HTTP-based debugging** to WebSocket-based approach

---

## Version Timeline

### v2.12.5 (Dec 9) - EditSource Line Ending Fix
- **CRLF→LF Normalization**: EditSource now handles SAP's `\r\n` vs AI's `\n`
- Fixed "old_string not found" errors from line ending mismatches

### v2.12.6 (Dec 10) - Class Includes Support
- **EditSource Class Includes**: Support for `.testclasses`, `.locals_def`, `.locals_imp`, `.macros`
- Better handling of class include separators

### v2.13.0 (Dec 14) - Call Graph & RCA Tools
- **GetCallersOf/GetCalleesOf**: Navigate call graphs bidirectionally
- **TraceExecution**: Composite RCA tool combining static graph + trace
- **CompareCallGraphs**: Find untested paths and dynamic-only calls
- **AnalyzeCallGraph**: Statistics on graph structure

### v2.14.0 (Dec 18) - Lua Scripting Integration
- **`vsp lua` Command**: Interactive REPL and script execution
- **40+ Lua Bindings**: All MCP tools accessible from Lua
- **Debug Session Management**: Full debug API in Lua
- **Checkpoint System**: Foundation for Force Replay
- **Example Scripts**: search-and-grep, debug-session, call-graph-analysis

### v2.15.0 (Dec 21) - Phase 5 TAS-Style Debugging
- **Variable History Recording**: Track all variable changes during execution
- **Extended Breakpoint Types**: Statement, exception, watchpoint scripting
- **Force Replay / State Injection**: THE KILLER FEATURE - inject saved state into live sessions
- **Phase 5 Complete**: TAS-style debugging operational

---

## Research & Investigation Reports

### Debugging Infrastructure (Dec 10-14)

| Report | Subject | Key Findings |
|--------|---------|--------------|
| [2025-12-10-001](reports/2025-12-10-001-editsource-class-includes.md) | EditSource Class Includes | Include type detection, separator handling |
| [2025-12-11-001](reports/2025-12-11-001-package-reassignment-odata-execute.md) | Package Reassignment & OData | Alternative package creation approaches |
| [2025-12-11-002](reports/2025-12-11-002-adt-abap-debugger-deep-dive.md) | ADT ABAP Debugger Deep Dive | TPDAPI structure, breakpoint APIs |
| [2025-12-11-003](reports/2025-12-11-003-debugger-breakpoint-fix-session.md) | Debugger Breakpoint Fix | XML format corrections, test verification |
| [2025-12-14-001](reports/2025-12-14-001-eclipse-adt-debugger-traffic-analysis.md) | Eclipse ADT Traffic Analysis | HTTP traffic capture, session management |
| [2025-12-14-002](reports/2025-12-14-002-external-debugger-breakpoint-storage-investigation.md) | External Breakpoint Storage | TRDIREPO table, persistence mechanism |

### WebSocket & RCA Infrastructure (Dec 18-19)

| Report | Subject | Key Findings |
|--------|---------|--------------|
| [2025-12-18-001](reports/2025-12-18-001-ai-assisted-rca-anst-integration.md) | AI-Assisted RCA & ANST | ANST tables, test result extraction |
| [2025-12-18-002](reports/2025-12-18-002-websocket-rfc-handler.md) | WebSocket RFC Handler | ZADT_VSP design, APC integration |
| [2025-12-19-001](reports/2025-12-19-001-websocket-debugging-deep-dive.md) | WebSocket Debugging Deep Dive | TPDAPI integration, session management |

### Phase 5 TAS-Style Debugging (Dec 21)

| Report | Subject | Key Findings |
|--------|---------|--------------|
| [2025-12-21-001](reports/2025-12-21-001-tas-scripting-time-travel-vision.md) | TAS Scripting & Time Travel | Complete vision for TAS-style ABAP debugging |
| [2025-12-21-002](reports/2025-12-21-002-test-extraction-isolated-replay.md) | Test Extraction & Isolated Replay | Playground architecture, mock framework |
| [2025-12-21-003](reports/2025-12-21-003-force-replay-state-injection.md) | Force Replay - State Injection | Variable injection API, session control |
| [2025-12-21-004](reports/2025-12-21-004-test-extraction-implications.md) | Test Extraction Implications | Paradigm shift: archaeology → observation |
| [2025-12-21-005](reports/2025-12-21-005-phase5-testing-methodology.md) | Phase 5 Testing Methodology | Verification approach for new features |
| [2025-12-21-006](reports/2025-12-21-006-phase5-data-extraction-examples.md) | Phase 5 Data Extraction Examples | Real-world extraction patterns |
| [2025-12-21-007](reports/2025-12-21-007-phase5-live-experiment.md) | Phase 5 Live Experiment | Production testing results |

### AMDP Debugging (Dec 22)

| Report | Subject | Key Findings |
|--------|---------|--------------|
| [2025-12-22-001](reports/2025-12-22-001-amdp-debugging-investigation.md) | AMDP Debugging Investigation | WebSocket session issue, breakpoint triggering |

---

## Technical Deep Dives

### 1. HTTP Debugger → WebSocket Transition

**The Problem**: HTTP-based external debugging had fundamental limitations:
- Breakpoints persisted but trigger reliability was inconsistent
- Session management across HTTP requests was problematic
- Long-polling listener blocked other operations

**The Solution**: ZADT_VSP WebSocket Handler
- **APC (Application Push Channel)** based architecture
- Unified handler for RFC, Debug, and AMDP domains
- Persistent WebSocket connection maintains session context
- TPDAPI integration for reliable breakpoint triggering

**Files Created**:
- `embedded/abap/zcl_vsp_apc_handler.clas.abap`
- `embedded/abap/zcl_vsp_rfc_service.clas.abap`
- `embedded/abap/zcl_vsp_debug_service.clas.abap`
- `embedded/abap/zif_vsp_service.intf.abap`

### 2. Lua Scripting Integration

**Architecture**:
```
vsp lua → gopher-lua engine → Go bindings → ADT client → SAP
```

**Capabilities**:
- Interactive REPL for ad-hoc debugging
- Script execution for automated workflows
- Full access to all MCP tools
- Checkpoint system for state capture/replay

**Example - Debug Session**:
```lua
local bpId = setBreakpoint("ZTEST_PROG", 42)
local event = listen(60)
if event then
    attach(event.id)
    print(json.encode(getStack()))
    stepOver()
    detach()
end
```

### 3. Force Replay / State Injection

**The Killer Feature**: Inject captured variable state into live debug sessions

**Use Cases**:
- Reproduce bugs without complex setup
- Test edge cases with specific data
- Share reproducible bug reports
- AI-driven exploratory debugging

**API**:
```lua
-- Capture
local checkpoint = saveCheckpoint("bug_repro", getVariables())

-- Replay (in another session)
local state = getCheckpoint("bug_repro")
injectCheckpoint(state)  -- Variables restored!
```

### 4. AMDP Debugging Investigation

**Status**: Experimental (breakpoints don't trigger)

**What Works**:
- AMDP debug session starts (CL_AMDP_DBG_MAIN->start)
- Control interface obtained (CL_AMDP_DBG_CONTROL->create)
- Breakpoints "set" without error (sync_breakpoints)
- AMDP execution works

**What Doesn't Work**:
- Breakpoints don't stop execution
- AMDP runs to completion regardless
- resume() blocks forever (no pending events)

**Root Cause Hypotheses**:
1. Breakpoint position format may be wrong (ABAP position vs native SQLScript)
2. HANA debugger connection may not be established
3. Session isolation - AMDP may run in different HANA session

---

## Code Changes Summary

### New Files
| File | Purpose |
|------|---------|
| `cmd/vsp/workflow.go` | Lua scripting and workflow commands |
| `embedded/abap/zcl_vsp_apc_handler.clas.abap` | WebSocket APC handler |
| `embedded/abap/zcl_vsp_debug_service.clas.abap` | Debug domain handler |
| `embedded/abap/zcl_vsp_amdp_service.clas.abap` | AMDP debug service |
| `pkg/adt/amdp_websocket.go` | Go WebSocket client for AMDP |
| `VISION.md` | Strategic vision document |
| `ROADMAP.md` | Implementation timeline |

### Modified Files
| File | Changes |
|------|---------|
| `pkg/adt/client.go` | Call graph methods, checkpoint system |
| `pkg/adt/crud.go` | Class include support |
| `pkg/adt/debugger.go` | Extended breakpoint types, TPDAPI |
| `pkg/adt/amdp_debugger.go` | AMDP session management |
| `internal/mcp/server.go` | 10+ new tool handlers |

---

## Statistics

| Metric | v2.12.5 | v2.15.0 | Change |
|--------|---------|---------|--------|
| MCP Tools | ~77 | 94+ | +17 |
| Unit Tests | 244 | 270+ | +26 |
| Research Reports | ~30 | 48+ | +18 |
| Lua Bindings | 0 | 40+ | +40 |
| WebSocket Domains | 0 | 3 | +3 |

---

## Key Decisions & Architectural Choices

### 1. WebSocket over HTTP for Debugging
**Rationale**: Session persistence, bidirectional communication, no polling needed

### 2. Lua for Scripting (not Python/JS)
**Rationale**:
- Lightweight, embeddable
- No external runtime dependency
- Familiar syntax for automation
- gopher-lua provides solid Go integration

### 3. ZADT_VSP as Optional Component
**Rationale**:
- Core vsp works without it (pure ADT)
- Enhanced features require SAP-side deployment
- Clear separation of concerns

### 4. Phase 5 Before Phase 4 Completion
**Rationale**:
- HTTP debugging limitations blocked Phase 4
- WebSocket foundation enables Phase 5-8
- Strategic pivot to higher-value features

---

## Lessons Learned

### 1. SAP Session Management is Critical
HTTP is stateless; debugging requires state. WebSocket solves this but requires SAP-side components.

### 2. ADT APIs Have Undocumented Behaviors
Eclipse traffic analysis revealed format requirements not in documentation (e.g., XML structure for breakpoints).

### 3. AMDP Debugging is Complex
Bridges ABAP and HANA debuggers. Even with correct API calls, session isolation prevents breakpoint hits.

### 4. Force Replay is Transformative
Ability to inject state opens new debugging paradigms - not just fixing bugs, but exploring code behavior.

---

## Next Steps

1. **AMDP Debugging**: Investigate breakpoint position format (native_position vs abap_position)
2. **Phase 6 Planning**: Test case extraction from debug sessions
3. **Documentation**: Comprehensive Lua scripting guide
4. **Community**: Share WebSocket handler as optional package

---

## Related Documentation

- [VISION.md](../VISION.md) - Strategic vision
- [ROADMAP.md](../ROADMAP.md) - Implementation timeline
- [CLAUDE.md](../CLAUDE.md) - AI development guidelines
- [README_TOOLS.md](../README_TOOLS.md) - Complete tool reference
