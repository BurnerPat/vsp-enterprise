# AMDP Debugging Investigation

**Date:** 2025-12-22
**Report ID:** 001
**Subject:** WebSocket-based AMDP Step-by-Step Debugging

---

## Summary

Implemented and tested a combined `executeAndDebug` action for same-session AMDP debugging via WebSocket. While the infrastructure works, **AMDP breakpoints do not trigger execution stops**.

## What Was Implemented

### ABAP Side (ZCL_VSP_AMDP_SERVICE)
- Added `executeAndDebug` action that:
  1. Starts AMDP debug session (or reuses existing)
  2. Sets breakpoint at specified line
  3. Executes AMDP method
  4. Calls `resume()` to get debug events
  5. Returns combined result

### Go Side (pkg/adt/amdp_websocket.go)
- Added `ExecuteAndDebug()` method
- Added `AMDPExecuteDebugResult` type

## Test Results

### Working
- AMDP debug session starts successfully (CL_AMDP_DBG_MAIN->start)
- Control interface obtained (CL_AMDP_DBG_CONTROL->create)
- Breakpoints "set" without error (sync_breakpoints)
- AMDP execution works (method runs and returns data)
- Resume blocks correctly (waiting for events)

### Not Working
- **Breakpoints don't stop execution** - The critical issue
- AMDP method runs to completion
- resume() then blocks forever (no pending ON_BREAK event)
- executeAndDebug times out

## Root Cause Analysis

The IF_AMDP_DBG_CONTROL->sync_breakpoints API is being called correctly, but breakpoints aren't triggering. Possible causes:

1. **Breakpoint position format**:
   - We use `abap_position-program_name = 'ZCL_ADT_00_AMDP_TEST'`
   - May need different format for AMDP (schema.procedure?)

2. **HANA debugger not connected**:
   - AMDP debugging bridges ABAP and HANA debuggers
   - May need additional HANA-side configuration

3. **Session isolation**:
   - AMDP debugging may require specific HANA session configuration
   - The generated procedure may run in different context

4. **Line number mapping**:
   - ABAP source line 40 vs SQLScript procedure line numbers
   - May need native_position instead of abap_position

## Code Locations

- **ABAP Service**: `/home/alice/dev/vibing-steampunk/embedded/abap/zcl_vsp_amdp_service.clas.abap`
- **Go Client**: `/home/alice/dev/vibing-steampunk/pkg/adt/amdp_websocket.go:563-618`
- **Test Class**: `ZCL_ADT_00_AMDP_TEST` (line 40 = `lv_square = :lv_i * :lv_i`)

## Test Scripts

- `/tmp/test_execute_and_debug.go` - Tests combined executeAndDebug
- `/tmp/test_amdp_step_by_step.go` - Step-by-step analysis

## Next Steps

1. **Investigate breakpoint format**: Research IF_AMDP_DBG_MAIN=>ty_dbg_breakpoint_req structure
2. **Check HANA debugger logs**: May provide insight on why breakpoints don't fire
3. **Eclipse ADT comparison**: Capture Eclipse AMDP debug traffic to see correct API usage
4. **Try native_position**: Use SQLScript position instead of ABAP position

## Architecture Note

The fundamental challenge with WebSocket AMDP debugging:
- `resume()` blocks the session waiting for events
- `execute()` must happen in the same session for breakpoints to work
- Our `executeAndDebug` solves this by combining both in one request

However, this only works if breakpoints actually trigger. Without working breakpoints, the method runs to completion and resume() blocks forever.

## Current Status

**AMDP Debugging: Experimental** - Infrastructure in place, breakpoint triggering needs investigation.
