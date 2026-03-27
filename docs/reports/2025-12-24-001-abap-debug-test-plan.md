# ABAP Debugging Test Plan

**Date:** 2025-12-24
**Report ID:** 001
**Subject:** End-to-end ABAP debugging verification
**Package:** $ZADT_DEBUG

---

## Objective

Verify that we can:
1. Create and run ABAP code that enters infinite loop
2. Confirm execution via SM66 (active work processes)
3. Debug and escape the loop using:
   - Variable modification (set LV_YES = ABAP_FALSE)
   - Jump to statement (skip the loop)

---

## Phase 1: Create Test Objects

### 1.1 Package
```
$ZADT_DEBUG - Debug Testing Package
```

### 1.2 Program: ZADT_DBG_PROG
```abap
REPORT zadt_dbg_prog.

DATA: lv_yes TYPE abap_bool VALUE abap_true,
      lv_counter TYPE i VALUE 0.

WRITE: / 'Starting infinite loop...'.

WHILE lv_yes = abap_true.
  lv_counter = lv_counter + 1.
  WAIT UP TO 5 SECONDS.
ENDWHILE.

WRITE: / 'Loop escaped! Counter:', lv_counter.
```

### 1.3 Class: ZCL_ADT_DBG_TEST
```abap
CLASS zcl_adt_dbg_test DEFINITION
  PUBLIC
  FINAL
  CREATE PUBLIC.

  PUBLIC SECTION.
    CLASS-METHODS run_infinite_loop.

ENDCLASS.

CLASS zcl_adt_dbg_test IMPLEMENTATION.

  METHOD run_infinite_loop.
    DATA: lv_yes TYPE abap_bool VALUE abap_true,
          lv_counter TYPE i VALUE 0.

    WHILE lv_yes = abap_true.
      lv_counter = lv_counter + 1.
      WAIT UP TO 5 SECONDS.
    ENDWHILE.
  ENDMETHOD.

ENDCLASS.
```

### 1.4 Function Group: ZADT_DBG_FG + Function Module: ZADT_DBG_FM_LOOP
```abap
FUNCTION zadt_dbg_fm_loop.
*"----------------------------------------------------------------------
*"  IMPORTING
*"     VALUE(IV_WAIT_SECONDS) TYPE I DEFAULT 5
*"----------------------------------------------------------------------
  DATA: lv_yes TYPE abap_bool VALUE abap_true,
        lv_counter TYPE i VALUE 0.

  WHILE lv_yes = abap_true.
    lv_counter = lv_counter + 1.
    WAIT UP TO iv_wait_seconds SECONDS.
  ENDWHILE.
ENDFUNCTION.
```

---

## Phase 2: Deploy Test Objects

Using vsp tools:
```bash
# 1. Create package
vsp create-package $ZADT_DEBUG "Debug Testing Package"

# 2. Deploy program
vsp write-source PROG ZADT_DBG_PROG --package $ZADT_DEBUG --source "..."

# 3. Deploy class
vsp write-source CLAS ZCL_ADT_DBG_TEST --package $ZADT_DEBUG --source "..."

# 4. Deploy function group + FM (may need manual creation)
```

---

## Phase 3: Execute Test Objects

### Option A: Run Program via SE38/SA38
```
Program: ZADT_DBG_PROG
```

### Option B: Call Class Method via Unit Test
```abap
" Create test class that calls the method
CLASS ltcl_runner DEFINITION FOR TESTING.
  PRIVATE SECTION.
    METHODS run FOR TESTING.
ENDCLASS.

CLASS ltcl_runner IMPLEMENTATION.
  METHOD run.
    zcl_adt_dbg_test=>run_infinite_loop( ).
  ENDMETHOD.
ENDCLASS.
```

### Option C: Call FM via RFC (WebSocket)
```json
{"domain":"rfc","action":"call","params":{"name":"ZADT_DBG_FM_LOOP","params":{"IV_WAIT_SECONDS":5}}}
```

---

## Phase 4: Verify Execution (SM66)

In SAP GUI:
1. Transaction SM66
2. Look for work process with:
   - Program: ZADT_DBG_PROG or ZCL_ADT_DBG_TEST
   - Status: Running
   - Action: WAIT (due to WAIT UP TO statement)

---

## Phase 5: Debug and Escape

### 5.1 Set External Breakpoint
```bash
# Via MCP tool (if available)
vsp set-breakpoint --program ZADT_DBG_PROG --line 8  # WHILE line

# Or via ADT API
POST /sap/bc/adt/debugger/breakpoints
```

### 5.2 Attach to Running Process
```bash
# Listen for debuggee
vsp debugger-listen --timeout 60

# Attach when caught
vsp debugger-attach --debuggee-id <id>
```

### 5.3 Escape Method 1: Variable Modification
```bash
# Get current variables
vsp debugger-get-variables

# Modify LV_YES to escape loop
vsp debugger-set-variable LV_YES ABAP_FALSE
# (Need to verify if this tool exists)

# Continue execution
vsp debugger-step --type stepContinue
```

### 5.4 Escape Method 2: Jump to Statement
```bash
# Jump past the ENDWHILE
vsp debugger-step --type stepJumpToLine --line 12
```

### 5.5 Detach
```bash
vsp debugger-detach
```

---

## Phase 6: Document Results

### Expected Outcomes

| Test | Method | Expected Result |
|------|--------|-----------------|
| Program execution | SE38 | Visible in SM66 |
| Class execution | Unit Test | Visible in SM66 |
| FM execution | RFC call | Visible in SM66 |
| Breakpoint hit | External BP | Debugger catches |
| Variable modify | Set LV_YES | Loop exits |
| Jump statement | stepJumpToLine | Loop skipped |

### Tools to Verify

| Tool | Status | Notes |
|------|--------|-------|
| DebuggerListen | ? | Long-polling for breakpoint hits |
| DebuggerAttach | ? | Attach to debuggee |
| DebuggerStep | ? | stepInto, stepOver, stepContinue |
| DebuggerGetVariables | ? | Read variable values |
| DebuggerSetVariable | ? | Modify variable (may not exist) |
| SetExternalBreakpoint | ? | Set breakpoint on line |

---

## Open Questions

1. **Can we set variables?** - Need to check if ADT API supports variable modification
2. **Can we jump to statement?** - stepJumpToLine support?
3. **How to trigger execution?** - Best method: Unit test vs RFC vs direct?
4. **Session management** - Does debugger need same session as execution?

---

## Execution Log (2025-12-28)

### Completed Steps

1. [x] **Create package $ZADT_DEBUG** - Created via MCP `CreatePackage`
2. [x] **Deploy ZADT_DBG_PROG** - Created with infinite loop code
3. [x] **Deploy ZCL_ADT_DBG_TEST** - Created with test class for triggering

### Architecture Discovery

**REST API Limitations:**
- `SetExternalBreakpoint` via REST returns 403 CSRF errors
- MCP tools for REST breakpoints were removed due to this issue
- `DebuggerListen` works when user is specified explicitly

**WebSocket Debug Service (ZCL_VSP_DEBUG_SERVICE):**
- Deployed on a4h-105 in package `$ZADT_VSP`
- Full TPDAPI integration for breakpoints:
  - `setBreakpoint` - line, exception, statement breakpoints
  - `getBreakpoints` - list active breakpoints
  - `deleteBreakpoint` - remove breakpoint
  - `listen` - wait for debuggee hits
  - `attach` - attach to debuggee
  - `step` - into/over/return/continue
  - `getStack` - call stack
  - `getVariables` - variable values
  - `detach` - release debuggee

**Gap Filled (2025-12-28):**
- ✅ Created `DebugWebSocketClient` in `pkg/adt/debug_websocket.go`
- ✅ Added MCP tools: `SetBreakpoint`, `GetBreakpoints`, `DeleteBreakpoint`
- ✅ All use WebSocket connection to ZADT_VSP debug domain

### REST API Test Results

```
DebuggerListen (no user):    500 - I_USER is initial
DebuggerListen (with user):  OK  - Timed out (no breakpoint set)
```

### WebSocket API Test Results (2025-12-28)

**All operations successful via ZADT_VSP WebSocket:**

```
Connect:         OK  - Session established
GetStatus:       OK  - {debuggingAvailable: true, breakpointCount: 0}
SetBreakpoint:   OK  - Breakpoint ID returned (ZADT_DBG_PROG line 8)
GetBreakpoints:  OK  - Returns array with breakpoint details
DeleteBreakpoint: OK  - Breakpoint removed
Listen:          OK  - Timed out (expected without execution trigger)
```

**Note:** Setting breakpoints on class pools requires "active, unchanged source" - use the program name format `ZCL_CLASS================CP` for classes.

---

## Next Steps

1. [x] Create package $ZADT_DEBUG
2. [x] Deploy ZADT_DBG_PROG
3. [x] Deploy ZCL_ADT_DBG_TEST
4. [x] **Create DebugWebSocketClient** - `pkg/adt/debug_websocket.go` (600 LOC)
5. [x] Test breakpoint via WebSocket - WORKS!
6. [x] Add MCP tools (SetBreakpoint, GetBreakpoints, DeleteBreakpoint)
7. [ ] Test attach and step (requires simultaneous execution trigger)
8. [ ] Test variable reading/modification
9. [x] Document findings

---

## Implementation Plan: DebugWebSocketClient

Similar to `AMDPWebSocketClient`, create a Go client that:
1. Connects to ZADT_VSP WebSocket endpoint
2. Sends JSON messages to `debug` domain
3. Implements methods:
   - `SetBreakpoint(program, line)`
   - `Listen(timeout)`
   - `Attach(debuggeeID)`
   - `Step(stepType)`
   - `GetVariables()`
   - `Detach()`

**WebSocket Message Format:**
```json
{"domain":"debug","action":"setBreakpoint","id":"1","params":{"program":"ZADT_DBG_PROG","line":8}}
```

