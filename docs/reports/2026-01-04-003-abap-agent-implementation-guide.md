# ABAP Agent Implementation Guide

**Date:** 2026-01-04
**Report ID:** 003
**Subject:** Recommendations for implementing ZLLM agentic investigator using VSP tools
**Audience:** ABAP developer building E2E agent on SAP side

---

## Executive Summary

This guide explains how VSP tools are implemented and how to leverage them from an ABAP-side LLM agent (ZLLM). Tools fall into two categories:

1. **ADT REST APIs** - Standard SAP endpoints, stateless, reliable
2. **ZADT_VSP WebSocket** - Custom handler for stateful operations

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        ZLLM Agent (ABAP)                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐   │
│  │  LLM Engine  │────▶│  Tool Router │────▶│  ADT Client  │   │
│  └──────────────┘     └──────────────┘     └──────────────┘   │
│                              │                    │             │
│                              │              ┌─────▼─────┐       │
│                              │              │ /sap/bc/  │       │
│                              │              │ adt/*     │       │
│                              │              └───────────┘       │
│                              │                                  │
│                        ┌─────▼─────┐                           │
│                        │ WebSocket │                           │
│                        │  Client   │                           │
│                        └─────┬─────┘                           │
│                              │                                  │
│                        ┌─────▼─────┐                           │
│                        │ /sap/bc/  │                           │
│                        │ apc/sap/  │                           │
│                        │ zadt_vsp  │                           │
│                        └───────────┘                           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Tool Implementation Categories

### Category 1: Pure ADT REST (Recommended)

These tools use standard SAP ADT endpoints. **Most reliable for ABAP-side implementation.**

#### Read Operations (GET requests)

| Tool | ADT Endpoint | ABAP Implementation |
|------|--------------|---------------------|
| GetSource (PROG) | `/sap/bc/adt/programs/programs/{name}/source/main` | `cl_adt_rest_client` |
| GetSource (CLAS) | `/sap/bc/adt/oo/classes/{name}/source/main` | `cl_adt_rest_client` |
| GetSource (INTF) | `/sap/bc/adt/oo/interfaces/{name}/source/main` | `cl_adt_rest_client` |
| GetSource (FUNC) | `/sap/bc/adt/functions/groups/{fg}/fmodules/{fm}/source/main` | `cl_adt_rest_client` |
| GetSource (FUGR) | `/sap/bc/adt/functions/groups/{name}/source/main` | `cl_adt_rest_client` |
| GetSource (DDLS) | `/sap/bc/adt/ddic/ddl/sources/{name}/source/main` | `cl_adt_rest_client` |
| GetTable | `/sap/bc/adt/ddic/tables/{name}` | `cl_adt_rest_client` |
| GetPackage | `/sap/bc/adt/packages/{name}` | `cl_adt_rest_client` |
| SearchObject | `/sap/bc/adt/repository/informationsystem/search?operation=quickSearch&query={q}` | `cl_adt_rest_client` |

**ABAP Example - Reading Source:**

```abap
METHOD get_program_source.
  DATA: lo_client TYPE REF TO if_http_client,
        lv_source TYPE string.

  " Create HTTP client for ADT
  cl_http_client=>create_by_destination(
    EXPORTING destination = 'NONE'  " Same system
    IMPORTING client = lo_client ).

  " Set URI
  lo_client->request->set_uri(
    |/sap/bc/adt/programs/programs/{ iv_program }/source/main| ).

  " Set headers
  lo_client->request->set_header_field(
    name = 'Accept' value = 'text/plain' ).

  " GET request
  lo_client->send( ).
  lo_client->receive( ).

  " Get response
  lv_source = lo_client->response->get_cdata( ).

  lo_client->close( ).
  rv_source = lv_source.
ENDMETHOD.
```

#### Write Operations (POST/PUT requests)

| Tool | Workflow | CSRF Required |
|------|----------|---------------|
| WriteSource | Lock → PUT source → Unlock → Activate | Yes |
| EditSource | GET → Replace → Lock → PUT → Unlock → Activate | Yes |
| CreateObject | POST to parent collection | Yes |
| Activate | POST to activation endpoint | Yes |

**ABAP Example - Write with CSRF:**

```abap
METHOD write_program_source.
  DATA: lo_client    TYPE REF TO if_http_client,
        lv_csrf      TYPE string,
        lv_lock_handle TYPE string.

  cl_http_client=>create_by_destination(
    EXPORTING destination = 'NONE'
    IMPORTING client = lo_client ).

  " Step 1: Fetch CSRF token
  lo_client->request->set_uri( '/sap/bc/adt/discovery' ).
  lo_client->request->set_header_field(
    name = 'X-CSRF-Token' value = 'Fetch' ).
  lo_client->send( ).
  lo_client->receive( ).
  lv_csrf = lo_client->response->get_header_field( 'X-CSRF-Token' ).

  " Step 2: Lock object
  lo_client->request->set_method( 'POST' ).
  lo_client->request->set_uri(
    |/sap/bc/adt/programs/programs/{ iv_program }?_action=LOCK&accessMode=MODIFY| ).
  lo_client->request->set_header_field( name = 'X-CSRF-Token' value = lv_csrf ).
  lo_client->send( ).
  lo_client->receive( ).
  " Parse lock handle from response...
  lv_lock_handle = parse_lock_handle( lo_client->response->get_cdata( ) ).

  " Step 3: PUT source
  lo_client->request->set_method( 'PUT' ).
  lo_client->request->set_uri(
    |/sap/bc/adt/programs/programs/{ iv_program }/source/main?lockHandle={ lv_lock_handle }| ).
  lo_client->request->set_header_field( name = 'Content-Type' value = 'text/plain' ).
  lo_client->request->set_cdata( iv_source ).
  lo_client->send( ).
  lo_client->receive( ).

  " Step 4: Unlock
  lo_client->request->set_method( 'POST' ).
  lo_client->request->set_uri(
    |/sap/bc/adt/programs/programs/{ iv_program }?_action=UNLOCK&lockHandle={ lv_lock_handle }| ).
  lo_client->send( ).
  lo_client->receive( ).

  " Step 5: Activate
  lo_client->request->set_method( 'POST' ).
  lo_client->request->set_uri( '/sap/bc/adt/activation' ).
  lo_client->request->set_cdata( build_activation_xml( iv_program ) ).
  lo_client->send( ).
  lo_client->receive( ).

  lo_client->close( ).
ENDMETHOD.
```

---

### Category 2: ZADT_VSP WebSocket (Custom Handler)

These tools require the custom WebSocket handler deployed to the SAP system.

**Package**: `$ZADT_VSP`
**APC Service**: `ZADT_VSP`
**SICF Path**: `/sap/bc/apc/sap/zadt_vsp`

#### WebSocket Services (Domains)

| Domain | Service Class | Tools |
|--------|---------------|-------|
| `rfc` | `ZCL_VSP_RFC_SERVICE` | CallRFC, moveToPackage |
| `debug` | `ZCL_VSP_DEBUG_SERVICE` | SetBreakpoint, GetBreakpoints, DeleteBreakpoint |
| `amdp` | `ZCL_VSP_AMDP_SERVICE` | AMDPDebugger* tools |
| `git` | `ZCL_VSP_GIT_SERVICE` | GitTypes, GitExport |
| `report` | `ZCL_VSP_REPORT_SERVICE` | RunReport, GetVariants, GetTextElements, SetTextElements |

#### WebSocket Message Format

```json
// Request
{
  "id": "unique-request-id",
  "domain": "rfc",
  "action": "call",
  "params": {
    "function": "RFC_PING"
  },
  "timeout": 30000
}

// Response
{
  "id": "unique-request-id",
  "success": true,
  "data": {"subrc": 0, "exports": {...}, "tables": {...}}
}

// Error Response
{
  "id": "unique-request-id",
  "success": false,
  "error": {"code": "FUNC_NOT_FOUND", "message": "Function XYZ not found"}
}
```

**ABAP Example - WebSocket Client:**

```abap
CLASS zcl_llm_ws_client DEFINITION.
  PUBLIC SECTION.
    METHODS call_rfc
      IMPORTING iv_function TYPE string
                iv_params   TYPE string OPTIONAL
      RETURNING VALUE(rv_result) TYPE string.
ENDCLASS.

CLASS zcl_llm_ws_client IMPLEMENTATION.
  METHOD call_rfc.
    " For same-system calls, use direct ABAP instead of WebSocket
    " WebSocket is for external clients (Go, Python, etc.)

    " Direct approach - call the service class directly:
    DATA: lo_service TYPE REF TO zcl_vsp_rfc_service,
          ls_message TYPE zif_vsp_service=>ty_message,
          ls_response TYPE zif_vsp_service=>ty_response.

    CREATE OBJECT lo_service.

    ls_message-id = cl_system_uuid=>create_uuid_c32_static( ).
    ls_message-domain = 'rfc'.
    ls_message-action = 'call'.
    ls_message-params = |{ "\{" }"function":"{ iv_function }","params":{ iv_params }{ "\}" }|.

    ls_response = lo_service->zif_vsp_service~handle_message(
      iv_session_id = 'ZLLM'
      is_message = ls_message ).

    rv_result = ls_response-data.
  ENDMETHOD.
ENDCLASS.
```

---

### Category 3: Direct ABAP (Best for Same-System)

For an ABAP agent running on the same system, **skip HTTP entirely** and use direct ABAP APIs.

#### Source Code Access

```abap
" Read program source
READ REPORT iv_program INTO lt_source.

" Write program source
INSERT REPORT iv_program FROM lt_source.

" Read class source
DATA(lo_class) = cl_oo_class=>get_instance( iv_class ).
DATA(lv_source) = lo_class->get_source( ).

" Read function module
CALL FUNCTION 'FUNCTION_INCLUDE_INFO'
  EXPORTING funcname = iv_function
  IMPORTING include = lv_include.
READ REPORT lv_include INTO lt_source.
```

#### Object Search

```abap
" Quick search using TADIR
SELECT obj_name, object, devclass
  FROM tadir
  WHERE obj_name LIKE @lv_pattern
    AND pgmid = 'R3TR'
  INTO TABLE @lt_results
  UP TO 100 ROWS.

" Cross-reference search
SELECT * FROM wbcrossgt
  WHERE name LIKE @lv_pattern
  INTO TABLE @lt_refs.
```

#### Syntax Check

```abap
SYNTAX-CHECK FOR lt_source
  PROGRAM iv_program
  MESSAGE lv_message
  LINE lv_line
  WORD lv_word.
IF sy-subrc <> 0.
  " Handle syntax error
ENDIF.
```

#### Activation

```abap
" Using activation API
DATA: lt_objects TYPE TABLE OF dwinactiv.
APPEND VALUE #( object = 'PROG' obj_name = iv_program ) TO lt_objects.

CALL FUNCTION 'RS_WORKING_OBJECTS_ACTIVATE'
  TABLES objects = lt_objects
  EXCEPTIONS OTHERS = 1.
```

---

## Tool Reliability Matrix

### Green: Highly Reliable (Use Confidently)

| Tool | Implementation | Notes |
|------|----------------|-------|
| GetSource | ADT REST | All object types work |
| SearchObject | ADT REST | Fast, supports wildcards |
| GetTable | ADT REST | Returns structure |
| GetTableContents | ADT REST | Uses CDS/SQL |
| RunQuery | ADT REST | ABAP SQL syntax |
| SyntaxCheck | ADT REST | Returns errors/warnings |
| Activate | ADT REST | Standard activation |
| RunUnitTests | ADT REST | ABAP Unit framework |
| GetDumps/GetDump | ADT REST | RABAX access |
| ListTraces/GetTrace | ADT REST | ATRA access |
| FindReferences | ADT REST | Where-used list |
| GetCallGraph | ADT REST | Call hierarchy |

### Yellow: Reliable with Caveats

| Tool | Implementation | Caveat |
|------|----------------|--------|
| WriteSource | ADT REST | Requires CSRF + lock workflow |
| EditSource | ADT REST | String must be unique in source |
| CreateObject | ADT REST | Some types need special handling |
| LockObject/UnlockObject | ADT REST | Must always unlock (use TRY/FINALLY) |
| GitExport | ZADT_VSP WS | Requires abapGit + ZADT_VSP |
| CallRFC | ZADT_VSP WS | Requires ZADT_VSP deployed |
| GetVariants | ZADT_VSP WS | Requires ZADT_VSP |
| GetTextElements | ZADT_VSP WS | Requires ZADT_VSP |

### Red: Experimental / Known Issues

| Tool | Implementation | Issue |
|------|----------------|-------|
| SetBreakpoint | ZADT_VSP WS | Works, but external BP behavior varies |
| DebuggerListen | ADT REST | Blocking, timeout handling needed |
| DebuggerAttach | ADT REST | Session management complex |
| AMDPDebugger* | ADT REST | Session persistence issues |
| RunReport | ZADT_VSP WS | APC blocks SUBMIT statement |
| UI5Upload/Delete/Create | ADT REST | Returns 405 (API limitation) |

---

## Recommended Tool Sequences for Agent

### Sequence 1: Investigate Object

```
1. SearchObject(query="ZFOO*")
   → Find matching objects

2. GetSource(type="CLAS", name="ZCL_FOO")
   → Read source code

3. FindReferences(object_url="/sap/bc/adt/oo/classes/ZCL_FOO")
   → Find where it's used

4. GetCallGraph(object_uri="...", direction="callees")
   → What does it call?
```

### Sequence 2: Modify Object

```
1. GetSource(type="PROG", name="ZTEST")
   → Read current source

2. [Agent modifies source]

3. SyntaxCheck(object_url="...", content=new_source)
   → Validate changes

4. IF syntax OK:
   WriteSource(type="PROG", name="ZTEST", source=new_source)
   → Save and activate
```

### Sequence 3: Debug Investigation

```
1. GetDumps(user="DEVELOPER", max_results=5)
   → Find recent errors

2. GetDump(dump_id="...")
   → Get stack trace and details

3. GetSource(type from dump program)
   → Read the failing code

4. GetCallersOf(object_uri="...")
   → Understand call chain
```

### Sequence 4: Code Quality

```
1. RunATCCheck(object_url="/sap/bc/adt/oo/classes/ZCL_FOO")
   → Get findings

2. FOR EACH finding:
   GetSource → Read code at finding location
   [Agent analyzes and suggests fix]

3. EditSource(old_string=problematic_code, new_string=fixed_code)
   → Apply fix

4. RunATCCheck again → Verify fix
```

---

## Error Handling Patterns

### CSRF Token Expiry

```abap
METHOD call_adt_with_retry.
  DATA: lv_retry TYPE i VALUE 0.

  WHILE lv_retry < 3.
    TRY.
        rv_result = call_adt( iv_url ).
        RETURN.
      CATCH zcx_csrf_invalid.
        " Refresh token and retry
        refresh_csrf_token( ).
        lv_retry = lv_retry + 1.
    ENDTRY.
  ENDWHILE.

  RAISE EXCEPTION TYPE zcx_adt_error.
ENDMETHOD.
```

### Lock Cleanup

```abap
METHOD modify_with_lock.
  DATA: lv_lock_handle TYPE string.

  TRY.
      lv_lock_handle = lock_object( iv_url ).
      update_source( iv_url = iv_url iv_source = iv_source iv_lock = lv_lock_handle ).
    CLEANUP.
      " ALWAYS unlock, even on error
      IF lv_lock_handle IS NOT INITIAL.
        unlock_object( iv_url = iv_url iv_lock = lv_lock_handle ).
      ENDIF.
  ENDTRY.
ENDMETHOD.
```

### WebSocket Timeout

```abap
METHOD call_websocket_with_timeout.
  DATA: lv_start TYPE timestampl,
        lv_now   TYPE timestampl.

  GET TIME STAMP FIELD lv_start.

  send_ws_message( is_message ).

  WHILE response_not_received( ).
    GET TIME STAMP FIELD lv_now.
    IF cl_abap_tstmp=>subtract( tstmp1 = lv_now tstmp2 = lv_start ) > iv_timeout.
      RAISE EXCEPTION TYPE zcx_timeout.
    ENDIF.

    " Small wait to avoid busy loop
    WAIT UP TO '0.1' SECONDS.
  ENDWHILE.

  rv_response = get_response( ).
ENDMETHOD.
```

---

## Prerequisites Checklist

### For ADT REST Tools

- [ ] ICF service `/sap/bc/adt` activated
- [ ] User has ADT development authorization (S_ADT_*)
- [ ] CSRF handling implemented

### For ZADT_VSP WebSocket Tools

- [ ] Package `$ZADT_VSP` exists with:
  - `ZIF_VSP_SERVICE` interface
  - `ZCL_VSP_APC_HANDLER` class
  - `ZCL_VSP_RFC_SERVICE` class
  - `ZCL_VSP_DEBUG_SERVICE` class
  - `ZCL_VSP_AMDP_SERVICE` class
  - `ZCL_VSP_GIT_SERVICE` class
  - `ZCL_VSP_REPORT_SERVICE` class
- [ ] SAPC configuration for `ZADT_VSP`
- [ ] SICF node `/sap/bc/apc/sap/zadt_vsp` activated
- [ ] For GitExport: abapGit must be installed

### For Debugger Tools

- [ ] User has debug authorization (S_DEBUG, S_DEVELOP)
- [ ] External breakpoints enabled for user
- [ ] For AMDP: HANA debugging enabled

---

## Quick Reference: ADT Endpoints

### Source Code

```
Programs:    /sap/bc/adt/programs/programs/{name}/source/main
Classes:     /sap/bc/adt/oo/classes/{name}/source/main
Interfaces:  /sap/bc/adt/oo/interfaces/{name}/source/main
Functions:   /sap/bc/adt/functions/groups/{fg}/fmodules/{fm}/source/main
FuncGroups:  /sap/bc/adt/functions/groups/{name}/source/main
Includes:    /sap/bc/adt/programs/includes/{name}/source/main
CDS:         /sap/bc/adt/ddic/ddl/sources/{name}/source/main
BDEF:        /sap/bc/adt/bo/behaviordefinitions/{name}/source/main
```

### Metadata

```
Tables:      /sap/bc/adt/ddic/tables/{name}
Structures:  /sap/bc/adt/ddic/structures/{name}
Packages:    /sap/bc/adt/packages/{name}
```

### Operations

```
Search:      /sap/bc/adt/repository/informationsystem/search?operation=quickSearch&query={q}
Activate:    /sap/bc/adt/activation
Lock:        {object_url}?_action=LOCK&accessMode=MODIFY
Unlock:      {object_url}?_action=UNLOCK&lockHandle={handle}
Syntax:      /sap/bc/adt/checkruns
Unit Tests:  /sap/bc/adt/abapunit/testruns
ATC:         /sap/bc/adt/atc/runs
References:  /sap/bc/adt/repository/informationsystem/usagereferences
```

### Diagnostics

```
Dumps:       /sap/bc/adt/runtime/dumps
Traces:      /sap/bc/adt/runtime/traces
SQL Traces:  /sap/bc/adt/runtime/traces/sqlTrace
```

---

## Summary Recommendations

1. **For same-system ABAP agent**: Use direct ABAP APIs (READ REPORT, etc.) - fastest and most reliable

2. **For cross-system or external agent**: Use ADT REST APIs with proper CSRF handling

3. **Avoid experimental tools in production agents**: SetBreakpoint, Debugger*, AMDP*, RunReport

4. **Always implement**:
   - CSRF token refresh on 403
   - Lock cleanup in FINALLY/CLEANUP blocks
   - Timeout handling for long operations

5. **Test with**: The VSP MCP server provides a reference implementation in Go - study `pkg/adt/*.go` for patterns

---

*Guide prepared for ZLLM agentic investigator implementation*
