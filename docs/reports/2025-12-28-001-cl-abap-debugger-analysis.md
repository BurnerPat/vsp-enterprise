# CL_ABAP_DEBUGGER Analysis Report

**Date:** 2025-12-28
**Report ID:** 001
**Subject:** Programmatic Debugger Control via CL_ABAP_DEBUGGER
**Related:** ABAP Debugger Scripting, External Breakpoints, ADT Integration

---

## Executive Summary

`CL_ABAP_DEBUGGER` is SAP's standard class for **external debugger control** - setting up debugging from OUTSIDE the debugger session. This is distinct from TPDA scripting classes (`CL_TPDA_*`) which operate INSIDE an active debug session.

**Key Use Cases:**
- Set HTTP/external breakpoints programmatically
- Activate batch job debugging
- Check debugger activation conflicts
- Query debugger options for current session

---

## Class Overview

```abap
CLASS cl_abap_debugger DEFINITION
  PUBLIC
  ABSTRACT
  FINAL
  CREATE PUBLIC.
```

- **Abstract**: Cannot be instantiated directly - all methods are static
- **Final**: Cannot be subclassed
- **Global Friend**: `CL_BGRFC_SUPPORTABILITY` (for background RFC debugging)

---

## Breakpoint Kind Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `bp_kind_prog` | 1 | Program line breakpoint |
| `bp_kind_subr` | 4 | Subroutine/FORM breakpoint |
| `bp_kind_func` | 6 | Function module breakpoint |
| `bp_kind_meth` | 8 | Method breakpoint |
| `bp_kind_stat` | 9 | Statement breakpoint |
| `bp_kind_syex` | 10 | System exception breakpoint |
| `bp_kind_exce` | 12 | Exception class breakpoint |
| `bp_kind_stln` | 15 | Source line breakpoint |
| `bp_kind_tmpl` | 16 | Template breakpoint |

---

## Core Methods

### 1. HTTP/External Breakpoint Management

#### `save_http_breakpoints`
Saves breakpoints to database tables (`ABDBG_BPS`, `ABDBG_INFO`) for external debugging.

```abap
CLASS-METHODS save_http_breakpoints
  IMPORTING
    client                TYPE symandt DEFAULT sy-mandt
    username              TYPE syuname DEFAULT sy-uname
    main_program          TYPE syrepid OPTIONAL
    breakpoints           TYPE breakpoints OPTIONAL
    flag_system_debugging TYPE abap_bool DEFAULT abap_undefined
    flag_exception_object TYPE abap_bool DEFAULT abap_undefined
  EXCEPTIONS
    too_many_breakpoints    " Max 30 breakpoints
    generate
    bp_position_not_found
    error
    wrong_parameters.
```

**Example:**
```abap
DATA: lt_breakpoints TYPE breakpoints,
      ls_breakpoint  TYPE breakpoint.

ls_breakpoint-program = 'ZTEST_PROGRAM'.
ls_breakpoint-line    = 15.
APPEND ls_breakpoint TO lt_breakpoints.

cl_abap_debugger=>save_http_breakpoints(
  EXPORTING
    username     = sy-uname
    main_program = 'ZTEST_PROGRAM'
    breakpoints  = lt_breakpoints ).
```

#### `read_http_breakpoints`
Reads saved HTTP breakpoints from database.

```abap
CLASS-METHODS read_http_breakpoints
  IMPORTING
    main_program           TYPE syrepid DEFAULT '*'
    client                 TYPE symandt DEFAULT sy-mandt
    username               TYPE syuname DEFAULT sy-uname
  EXPORTING
    breakpoints_complete   TYPE breakpoints_complete
    breakpoints            TYPE breakpoints
    flag_system_debugging  TYPE abap_bool
    flag_exception_object  TYPE abap_bool
    number_all_breakpoints TYPE i.
```

#### `delete_http_breakpoints`
Deletes all HTTP breakpoints for a user.

```abap
CLASS-METHODS delete_http_breakpoints
  IMPORTING
    client   TYPE symandt DEFAULT sy-mandt
    username TYPE syuname DEFAULT sy-uname.
```

#### `save_http_breakpoint` / `delete_http_breakpoint`
Single breakpoint operations with detailed control.

```abap
CLASS-METHODS save_http_breakpoint
  IMPORTING
    client      TYPE symandt
    username    TYPE syuname
    bp_kind     TYPE bp_kind           " Use constants above
    ref_bp_info TYPE REF TO data.      " Type depends on bp_kind
```

---

### 2. Session Breakpoint Management

#### `save_breakpoints` / `read_breakpoints`
Session-level breakpoints (within current ABAP session).

```abap
CLASS-METHODS save_breakpoints
  IMPORTING
    flag_other_session        TYPE flag DEFAULT space
    main_program              TYPE syrepid
    breakpoints               TYPE breakpoints
    flag_system_debugging     TYPE abap_bool DEFAULT abap_undefined
    flag_exception_object     TYPE abap_bool DEFAULT abap_undefined
    flag_activate_immediately TYPE abap_bool DEFAULT abap_false.
```

**Note:** `flag_activate_immediately = abap_true` loads breakpoints into running context immediately.

---

### 3. Batch Debugging

#### `check_activation_for_btcdbg`
Pre-check before activating batch debugging. Verifies:
- User has `S_DEVELOP` authorization for debugging
- SAPGui is available
- Valid debugger activation exists

```abap
CLASS-METHODS check_activation_for_btcdbg
  IMPORTING
    is_jobstep_uname TYPE syst_uname
  RAISING
    cx_abdbg_btcact_internal
    cx_abdbg_btcact_illegal_act.
```

#### `activate_batch_debugging`
Activates debugging for a batch job step.

```abap
CLASS-METHODS activate_batch_debugging
  IMPORTING
    is_jobstep_uname TYPE syst_uname
  RAISING
    cx_abdbg_btcact_illegal_act.
```

---

### 4. Debugger Options & Status

#### `get_dbg_options`
Returns debugger options for current session.

```abap
CLASS-METHODS get_dbg_options
  RETURNING
    VALUE(options) TYPE options_t.
```

**Options Structure:**
```abap
TYPES: BEGIN OF OPTIONS_T,
         DebuggeeRunning        TYPE ABAP_BOOL,  " Is this session being debugged?
         SystemDebugging        TYPE ABAP_BOOL,  " System debugging enabled
         SyncWithSavedBPs       TYPE ABAP_BOOL,  " Sync with session breakpoints
         StopOnImodeEnd         TYPE ABAP_BOOL,  " Stop on internal session end
         ExcCreateObject        TYPE ABAP_BOOL,  " Create exception object always
         UpdDebugging           TYPE ABAP_BOOL,  " Update debugging
         aRFCDebugging          TYPE ABAP_BOOL,  " Block sending aRFCs
         EsfDebugging           TYPE ABAP_BOOL,  " ESF debugging
         EmodeCallstack         TYPE ABAP_BOOL,  " Cross-session call stack
         ACFlush                TYPE ABAP_BOOL,  " Automation controller flush
         SoAac                  TYPE ABAP_BOOL,  " Shared Objects: area constructor
         ItabCheckReadBinary    TYPE ABAP_BOOL,  " Check itab order
         LayerControl           TYPE ABAP_BOOL,  " Layer control active
         DynpDebugging          TYPE ABAP_BOOL,  " Dynpro debugging
       END OF OPTIONS_T.
```

**Example - Check if being debugged:**
```abap
DATA(ls_options) = cl_abap_debugger=>get_dbg_options( ).
IF ls_options-debuggeerunning = abap_true.
  " We are being debugged!
ENDIF.
```

---

### 5. Terminal ID Management

#### `get_breakpoint_tid`
Gets terminal ID for breakpoint activation. Tries:
1. Read from Windows registry (`HKCU\Software\sap\ABAP Debugging\TerminalID`)
2. Create new UUID and store in registry
3. Fallback: Create session-only terminal ID

```abap
CLASS-METHODS get_breakpoint_tid
  RETURNING
    VALUE(terminal_id) TYPE sysuuid_c32.
```

#### `set_request_tid`
Sets terminal ID from frontend registry to current request.

---

### 6. Conflict Detection

#### `check_activation_for_sapgui`
Pre-check for SAPGui debugger activation conflicts.

**Mode 1:** User breakpoint activation
- Supply `is_rq_uname` + `is_c_uname`
- Checks for existing activation for user

**Mode 2:** Terminal ID activation
- Supply `is_terminal_id` + `is_c_uname`
- Checks for existing activation for terminal

```abap
CLASS-METHODS check_activation_for_sapgui
  IMPORTING
    is_rq_uname    TYPE syst_uname
    is_c_uname     TYPE syst_uname
    is_terminal_id TYPE sysuuid_c32
  RAISING
    cx_abdbg_sapguiact_conflict.
```

---

## Database Tables

### ABDBG_BPS - Breakpoint Storage
| Field | Type | Description |
|-------|------|-------------|
| CLIENT | MANDT | Client |
| USERNAME | SYUNAME | User name |
| BP_INDEX | INT2 | Breakpoint index |
| BP_KIND | INT2 | Breakpoint kind (see constants) |
| BP_PROGRAM | PROGNAME | Main program |
| BP_CLASS | CHAR30 | Line number (for prog BPs) |
| BP_TEXT | CHAR40 | Include name / method / etc. |
| BP_NUMBER | INT4 | Additional identifier |
| TERM_ID | UUID | Terminal ID |
| IDE_ID | UUID | IDE ID |

### ABDBG_INFO - Breakpoint Header
| Field | Type | Description |
|-------|------|-------------|
| CLIENT | MANDT | Client |
| USERNAME | SYUNAME | User name |
| DATE_VALID | DATS | Valid until date |
| SYSTEM_DBG | ABAP_BOOL | System debugging flag |
| EXCPOBJECT | ABAP_BOOL | Exception object flag |

---

## Comparison: CL_ABAP_DEBUGGER vs CL_TPDA_*

| Aspect | CL_ABAP_DEBUGGER | CL_TPDA_* Classes |
|--------|------------------|-------------------|
| **Context** | OUTSIDE debugger | INSIDE debugger |
| **Purpose** | Setup debugging | Control debugging |
| **Breakpoints** | Set/save to DB | Navigate, not set |
| **Variables** | Cannot access | Full read/write access |
| **Execution** | Cannot step | Step into/over/return |
| **Use Case** | Prepare for debugging | Automate debugging actions |

---

## Practical Applications

### 1. CI/CD Integration
Set breakpoints programmatically before running tests:
```abap
" Set breakpoints on critical code paths
zcl_adt_dbg_controller=>set_breakpoints(
  iv_program = 'ZPRODUCTION_CODE'
  it_lines   = VALUE #( ( 100 ) ( 200 ) ( 300 ) ) ).

" Run tests - if breakpoint hit, debugging activates
SUBMIT ztest_runner AND RETURN.

" Cleanup
zcl_adt_dbg_controller=>delete_breakpoints( ).
```

### 2. Remote Debugging Setup
```abap
" Check for conflicts first
TRY.
    cl_abap_debugger=>check_activation_for_sapgui(
      is_rq_uname    = 'TESTUSER'
      is_c_uname     = sy-uname
      is_terminal_id = '' ).
  CATCH cx_abdbg_sapguiact_conflict INTO DATA(lx_conflict).
    " Another session already has activation
    MESSAGE lx_conflict->get_text( ) TYPE 'E'.
ENDTRY.

" Set breakpoints for remote user
cl_abap_debugger=>save_http_breakpoints(
  username     = 'TESTUSER'
  main_program = 'ZTARGET_PROGRAM'
  breakpoints  = lt_breakpoints ).
```

### 3. Batch Job Debugging
```abap
" Before starting batch job
TRY.
    cl_abap_debugger=>check_activation_for_btcdbg( 'BATCH_USER' ).
    cl_abap_debugger=>activate_batch_debugging( 'BATCH_USER' ).
  CATCH cx_abdbg_btcact_illegal_act INTO DATA(lx_act).
    " No valid activation - need to set breakpoints first
ENDTRY.
```

### 4. Debugger-Aware Code
```abap
METHOD do_sensitive_operation.
  " Skip in debug mode to prevent data corruption
  IF cl_abap_debugger=>get_dbg_options( )-debuggeerunning = abap_true.
    RAISE EXCEPTION TYPE zcx_debug_mode_blocked.
  ENDIF.

  " Proceed with operation...
ENDMETHOD.
```

---

## Limitations

1. **Max 30 breakpoints** per user (HTTP breakpoints)
2. **No execution control** - cannot step, only set breakpoints
3. **No variable access** - use TPDA classes for that
4. **GUI dependency** for terminal ID - fallback to session-only ID
5. **Breakpoints expire** - DATE_VALID in ABDBG_INFO, cleaned by `delete_old_sessions`

---

## Related Classes

| Class | Purpose |
|-------|---------|
| `CL_TPDA_SCRIPT_CLASS_SUPER` | Base for debugger scripts (inside debugger) |
| `CL_TPDA_SCRIPT_DATA_DESCR` | Variable access in debugger |
| `CL_TPDA_SCRIPT_DEBUGGER_CTRL` | Execution control in debugger |
| `CX_ABDBG_SAPGUIACT_CONFLICT` | Activation conflict exception |
| `CX_ABDBG_BTCACT_ILLEGAL_ACT` | Batch activation exception |

---

## Example Implementation

See created objects:
- **ZADT_DBG_CONTROLLER** - Report demonstrating all CL_ABAP_DEBUGGER methods
- **ZADT_DBG_TARGET** - Target program with breakpoint-friendly lines
- **ZCL_ADT_DBG_CONTROLLER** - Wrapper class with unit tests

---

## Conclusion

`CL_ABAP_DEBUGGER` is useful for:
- **Automated testing** - set breakpoints before test runs
- **Remote debugging** - prepare breakpoints for another user
- **Batch debugging** - activate debugging for background jobs
- **Debugger-aware code** - detect if code is being debugged

For in-debugger automation (stepping, variable inspection), use TPDA scripting classes instead.
