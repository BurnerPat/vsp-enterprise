# ZADT_VSP-Dependent Tools

**Date:** 2026-03-27  
**Purpose:** Catalog all MCP tools that depend on the ZADT_VSP WebSocket service (custom ABAP code on the SAP backend). With the `InstallZADTVSP` tool removed, these tools no longer have an automated deployment path — users must manually deploy the ABAP code or these tools need rework.

---

## Background

### What is ZADT_VSP?

ZADT_VSP is a custom **APC (ABAP Push Channel) WebSocket handler** deployed to the SAP backend. It provides a stateful, bidirectional JSON-over-WebSocket protocol for operations that ADT's REST API cannot support (debugging sessions, RFC calls, report execution, AMDP debugging).

The ABAP source files live in `src/` (abapGit format) and `abap/src/zadt_vsp/` (older subset).

### WebSocket Connection

- **URL:** `ws[s]://{host}/sap/bc/apc/sap/zadt_vsp?sap-client={client}`
- **Auth:** Basic Authentication via `Authorization` header
- **Handshake:** 30s timeout, expects a `welcome` message within 5s of connect
- **Protocol:** JSON request/response with `id`, `domain`, `action`, `params` fields

### Message Format

```json
// Request
{ "id": "debug_1", "domain": "debug", "action": "setBreakpoint", "params": { ... }, "timeout": 60000 }

// Response
{ "id": "debug_1", "success": true, "data": { ... } }
```

### Server-Side WebSocket Clients

The MCP server maintains two lazy WebSocket client instances (both connect to the same endpoint):

| Field | Type | Purpose |
|-------|------|---------|
| `s.debugWSClient` | `DebugWebSocketClient` | Breakpoints, CallRFC, MoveObject |
| `s.amdpWSClient` | `AMDPWebSocketClient` | AMDP debugger, reports, text elements, Git ops, GetAbapHelp |

---

## Affected Tools (17 required + 1 optional)

### Dual-Path Tools (4) — WebSocket in HTTP mode, standard API in RFC mode

These tools work **without ZADT_VSP when using RFC connection mode**. They only require ZADT_VSP in HTTP mode.

| Tool | Focused? | Group | HTTP Mode | RFC Mode |
|------|:--------:|:-----:|-----------|----------|
| **SetBreakpoint** | Yes | D, X | `debugWSClient.SetLineBreakpoint()` etc. | `adtClient.SetExternalBreakpoint()` via ADT REST |
| **GetBreakpoints** | Yes | D, X | `debugWSClient.GetBreakpoints()` | `adtClient.GetExternalBreakpoints()` |
| **DeleteBreakpoint** | Yes | D, X | `debugWSClient.DeleteBreakpoint()` | `adtClient.DeleteExternalBreakpoint()` |
| **CallRFC** | Yes | D, X | `debugWSClient.CallRFC()` | `sidecar.CallRFC()` via JCo |

**File:** `internal/mcp/handlers_debugger.go`

### Always-Required Tools — No Fallback

#### MoveObject (1 tool)

| Tool | Focused? | Group | Behavior |
|------|:--------:|:-----:|----------|
| **MoveObject** | Yes | — | `debugWSClient.MoveObject()` → calls `TR_TADIR_INTERFACE` on backend |

RFC mode returns "not available". No ADT REST alternative exists.

**File:** `internal/mcp/handlers_crud.go`

#### AMDP Debugger (7 tools)

| Tool | Focused? | Group | WebSocket Action |
|------|:--------:|:-----:|-----------------|
| **AMDPDebuggerStart** | Yes | H, X | `amdp:start` — creates session with cascade mode |
| **AMDPDebuggerResume** | Yes | H, X | `amdp:resume` — returns break events with positions |
| **AMDPDebuggerStop** | Yes | H, X | `amdp:stop` — closes session |
| **AMDPDebuggerStep** | Yes | H, X | `amdp:step` — into/over/return/continue |
| **AMDPGetVariables** | Yes | H, X | `amdp:getVariables` — scalars and tables |
| **AMDPSetBreakpoint** | Yes | H, X | `amdp:setBreakpoint` — program + line |
| **AMDPGetBreakpoints** | Yes | H, X | `amdp:getBreakpoints` |

All return "not available in RFC mode". No ADT REST alternative exists for AMDP debugging.

**File:** `internal/mcp/handlers_amdp.go`

#### Report Tools (5 tools)

| Tool | Focused? | Group | WebSocket Action |
|------|:--------:|:-----:|-----------------|
| **RunReport** | Yes | R, X | Schedules background job → polls status → retrieves spool |
| **RunReportAsync** | Yes | — | Same as RunReport but in background goroutine (5min timeout) |
| **GetVariants** | Yes | R | `rfc:getVariants` |
| **GetTextElements** | Yes | R | `rfc:getTextElements` |
| **SetTextElements** | Yes | R | `rfc:setTextElements` |

All return "not available in RFC mode". `GetAsyncResult` is **not** affected (reads from in-memory task map only).

**File:** `internal/mcp/handlers_report.go`

### Optional Dependency (1 tool)

| Tool | Focused? | Group | Behavior |
|------|:--------:|:-----:|----------|
| **GetAbapHelp** | Yes | — | Tries `amdpWSClient.GetAbapDocumentation()` for full HTML docs; silently falls back to URL-only result if ZADT_VSP is unavailable |

This tool works fine without ZADT_VSP — the WebSocket call is purely an enrichment.

**File:** `internal/mcp/handlers_system.go`

---

## Tool Group Summary

| Group | Code | Tools Affected | Disableable? |
|-------|------|---------------|:------------:|
| ABAP Debugger | `D` | SetBreakpoint, GetBreakpoints, DeleteBreakpoint, CallRFC | Yes |
| HANA/AMDP | `H` | All 7 AMDP tools | Yes |
| Reports | `R` | RunReport, GetVariants, GetTextElements, SetTextElements | Yes |
| Experimental | `X` | Union of D + H + RunReport (17 tools) | Yes |

MoveObject, RunReportAsync, and GetAbapHelp are not in any disableable group.

---

## ABAP Service Components

The ZADT_VSP service consists of these ABAP classes (in `src/`):

| Class | Role |
|-------|------|
| `ZCL_VSP_APC_HANDLER` | APC WebSocket entry point — routes messages to domain services |
| `ZIF_VSP_SERVICE` | Service interface (`handle_message`, `on_disconnect`) |
| `ZCL_VSP_DEBUG_SERVICE` | Domain `debug` — TPDAPI breakpoints, debugger attach, call stack |
| `ZCL_VSP_RFC_SERVICE` | Domain `rfc` — dynamic RFC/BAPI calls, moveToPackage, runReport, variants, text elements |
| `ZCL_VSP_AMDP_SERVICE` | Domain `amdp` — AMDP/HANA SQLScript debugging |
| `ZCL_VSP_GIT_SERVICE` | Domain `git` — abapGit serialization (import/export) |
| `ZCL_VSP_REPORT_SERVICE` | Domain `report` — report execution, ALV capture |
| `ZCL_VSP_UTILS` | Shared helpers (JSON escaping, param extraction) |
| `ZCL_VSP_TADIR_MOVE` | Package move via `TR_TADIR_INTERFACE` |

---

## Options for Each Tool Category

### Dual-path tools (SetBreakpoint, GetBreakpoints, DeleteBreakpoint, CallRFC)
- Already work in RFC mode via standard APIs
- Could add ADT REST fallback for HTTP mode (breakpoints already have `adtClient.SetExternalBreakpoint`)
- Lowest effort — the alternative code paths exist

### MoveObject
- No ADT REST API for `TR_TADIR_INTERFACE`
- Options: remove the tool, make it RFC-only, or document ZADT_VSP as prerequisite

### AMDP Debugger (7 tools)
- No ADT REST alternative — AMDP debugging requires stateful WebSocket
- Options: remove entirely, or keep with documented ZADT_VSP prerequisite

### Report Tools (5 tools)
- Report execution requires background job scheduling + spool retrieval
- Options: remove, implement via ADT REST if endpoints exist, or keep with prerequisite

### GetAbapHelp
- No action needed — already works without ZADT_VSP
