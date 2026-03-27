# Heavyweight Operations Architecture for VSP

**Date:** 2025-12-23
**Report ID:** 001
**Subject:** Architecture options for handling large payloads and long-running operations
**Status:** Analysis & Recommendations

---

## Executive Summary

The VSP project's current WebSocket-based architecture (ZADT_VSP APC handler) faces timeout challenges when handling heavyweight operations like abapGit package exports. This report analyzes four architectural approaches for solving the timeout problem and provides recommendations for both the immediate Git export issue and general heavyweight operation patterns.

**Key Finding:** A hybrid approach is recommended - keep the WebSocket architecture but add **chunked/streaming responses** for large payloads and **async polling** for truly long-running operations.

## Problem Statement

### Current Architecture

```
vsp (Go) ──WebSocket──> ZADT_VSP APC Handler ──> Service Domains
                                                   ├─ rfc
                                                   ├─ debug
                                                   ├─ amdp
                                                   └─ git (NEW)
```

**Current timeout configuration:**
- WebSocket read timeout: **60 seconds** (`pkg/adt/amdp_websocket.go:266`)
- ABAP-side timeout parameter: **30-60 seconds** (configurable per message)

### The Git Export Problem

When exporting large packages via abapGit:
1. **Serialization is slow** - `ZCL_ABAPGIT_OBJECTS=>serialize()` processes each object sequentially
2. **ZIP compression takes time** - `CL_ABAP_ZIP=>save()` is synchronous
3. **Base64 encoding adds overhead** - Large xstrings → string conversion
4. **Single-threaded ABAP** - No parallel processing available

**Example timing:**
- 50 objects: ~15-30 seconds (OK)
- 200 objects: ~60-120 seconds (TIMEOUT)
- 500+ objects: ~5-10 minutes (MAJOR TIMEOUT)

### Impact

- Git export timeouts block the entire abapGit integration feature
- Similar issues will arise for:
  - Large ATC check runs (100+ findings)
  - Profiler trace analysis (millions of records)
  - Package-wide syntax checks
  - Mass activation operations

## Architecture Options

### Option 1: Keep Within APC - Add Async/Chunked Responses

**Concept:** Enhance the WebSocket protocol to support chunked/streaming responses and progress updates.

#### Implementation Design

```
┌─────────────────────────────────────────────────────────────┐
│                     Client (Go)                              │
│  - Sends request with stream:true flag                       │
│  - Reads multiple response chunks                            │
│  - Assembles final result                                    │
└─────────────────────────────────────────────────────────────┘
                              │
                   WebSocket (bidirectional)
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              ZADT_VSP APC Handler (Enhanced)                 │
│  - Detects stream/async requests                            │
│  - Sends progress events: {"type":"progress","done":10}      │
│  - Sends data chunks: {"type":"chunk","seq":1,"data":"..."}  │
│  - Sends final: {"type":"complete","totalChunks":5}          │
└─────────────────────────────────────────────────────────────┘
```

#### Protocol Enhancement

**Request with chunking:**
```json
{
  "id": "export-1",
  "domain": "git",
  "action": "export",
  "params": {"packages": ["$ZRAY"]},
  "stream": true,
  "chunkSize": 5242880
}
```

**Response sequence:**
```json
// 1. Progress update (optional)
{"id":"export-1","type":"progress","done":10,"total":100}

// 2. Data chunks
{"id":"export-1","type":"chunk","seq":1,"data":"UEsDBBQA..."}
{"id":"export-1","type":"chunk","seq":2,"data":"AAABBBCC..."}

// 3. Final message
{"id":"export-1","type":"complete","totalChunks":2,"objectCount":100}
```

#### ABAP Implementation

```abap
METHOD handle_export.
  " Check if streaming requested
  DATA lv_stream TYPE abap_bool.
  FIND '"stream"\s*:\s*true' IN is_message-params.
  lv_stream = xsdbool( sy-subrc = 0 ).

  IF lv_stream = abap_true.
    " Chunked export
    export_chunked(
      is_message = is_message
      iv_chunk_size = 5000000  " 5 MB chunks
    ).
  ELSE.
    " Traditional single-response export
    export_single( is_message ).
  ENDIF.
ENDMETHOD.

METHOD export_chunked.
  DATA: lv_chunk_seq TYPE i,
        lv_chunk_data TYPE string,
        lv_zip_xstring TYPE xstring,
        lv_offset TYPE i,
        lv_remaining TYPE i.

  " 1. Serialize objects (slow part)
  LOOP AT it_tadir INTO DATA(ls_tadir).
    serialize_objects( ... ).

    " Send progress after each object
    DATA(lv_progress) = lines( lt_processed ) * 100 / lines( it_tadir ).
    send_progress_event(
      iv_id = is_message-id
      iv_done = lv_progress
    ).
  ENDLOOP.

  " 2. Get ZIP data
  lv_zip_xstring = lo_zip->save( ).
  lv_remaining = xstrlen( lv_zip_xstring ).

  " 3. Send in chunks
  WHILE lv_remaining > 0.
    lv_chunk_seq += 1.
    DATA(lv_chunk_len) = nmin( val1 = iv_chunk_size val2 = lv_remaining ).

    " Extract chunk
    DATA(lv_chunk_xstring) = lv_zip_xstring+lv_offset(lv_chunk_len).
    lv_chunk_data = xstring_to_base64( lv_chunk_xstring ).

    " Send chunk
    send_chunk_event(
      iv_id = is_message-id
      iv_seq = lv_chunk_seq
      iv_data = lv_chunk_data
    ).

    lv_offset += lv_chunk_len.
    lv_remaining -= lv_chunk_len.
  ENDWHILE.

  " 4. Send completion
  send_complete_event(
    iv_id = is_message-id
    iv_total_chunks = lv_chunk_seq
  ).
ENDMETHOD.
```

#### Go Client Enhancement

```go
// AMDPWebSocketClient enhancement
type StreamResponse struct {
    Type         string          `json:"type"`  // "progress", "chunk", "complete"
    Seq          int             `json:"seq,omitempty"`
    Data         string          `json:"data,omitempty"`
    Done         int             `json:"done,omitempty"`
    Total        int             `json:"total,omitempty"`
    TotalChunks  int             `json:"totalChunks,omitempty"`
}

func (c *AMDPWebSocketClient) GitExportStreaming(ctx context.Context, packages []string) (*GitExportResult, error) {
    params := map[string]interface{}{
        "packages": packages,
        "stream":   true,
        "chunkSize": 5 * 1024 * 1024,
    }

    msg := WSMessage{
        ID:      fmt.Sprintf("export_%d", c.msgID.Add(1)),
        Domain:  "git",
        Action:  "export",
        Params:  params,
    }

    // Send request
    c.sendMessage(msg)

    // Collect chunks
    var chunks []string
    var objectCount int

    for {
        var streamResp StreamResponse
        if err := c.readJSON(&streamResp); err != nil {
            return nil, err
        }

        switch streamResp.Type {
        case "progress":
            fmt.Printf("Progress: %d%%\n", streamResp.Done)

        case "chunk":
            chunks = append(chunks, streamResp.Data)
            fmt.Printf("Received chunk %d\n", streamResp.Seq)

        case "complete":
            objectCount = streamResp.TotalChunks
            goto assemble
        }
    }

assemble:
    // Concatenate all chunks
    fullData := strings.Join(chunks, "")

    return &GitExportResult{
        ObjectCount: objectCount,
        ZipBase64:   fullData,
    }, nil
}
```

#### Pros
- ✅ **No new SAP infrastructure** - Uses existing WebSocket handler
- ✅ **Bidirectional progress** - Client gets real-time updates
- ✅ **No timeout issues** - Each chunk arrives within timeout window
- ✅ **Fits existing architecture** - Natural extension of ZADT_VSP pattern
- ✅ **Works for all heavyweight ops** - Generic solution

#### Cons
- ❌ **Still synchronous** - ABAP thread blocked during operation
- ❌ **No cancellation** - Hard to stop mid-operation
- ❌ **Memory pressure** - Large data held in ABAP memory during chunking
- ❌ **Protocol complexity** - Client must handle multiple message types

#### Complexity
- **Low-Medium** (2-3 days implementation)
- Modify `ZCL_VSP_GIT_SERVICE` to support chunking
- Add chunk assembly logic in Go client
- Test with large packages

---

### Option 2: Separate OData Service for Heavyweight Operations

**Concept:** Create a dedicated RAP OData V4 service for operations that require better HTTP streaming support.

#### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     vsp (Go)                                 │
│  - WebSocket for interactive ops (debug, rfc)               │
│  - OData for heavyweight ops (git export, ATC)              │
└─────────────────────────────────────────────────────────────┘
            │                           │
    WebSocket (stateful)         HTTP POST (stateless)
            │                           │
            ▼                           ▼
┌──────────────────────┐    ┌──────────────────────────────┐
│   ZADT_VSP (APC)     │    │  ZADT_HEAVY_O4 (OData V4)    │
│   - debug            │    │  - gitExport action          │
│   - amdp             │    │  - atcCheck action           │
│   - rfc              │    │  - Uses HTTP chunked encoding│
└──────────────────────┘    └──────────────────────────────┘
```

#### OData Implementation

```abap
" DDLS: ZADT_R_HEAVY_OPS (Root entity - singleton)
define root view entity ZADT_R_HEAVY_OPS
  as select from I_Language
{
  key cast( 'HEAVY' as abap.char(10) ) as ServiceId,
      tstmp_current_utctimestamp() as LastChanged
}
where Language = $session.system_language

" BDEF: ZADT_R_HEAVY_OPS
define behavior for ZADT_R_HEAVY_OPS alias HeavyOps
implementation unmanaged
lock master
authorization master ( instance )
{
  action ( features: instance ) gitExport
    parameter ZADT_A_GIT_EXPORT_PARAM
    result [1] ZADT_A_GIT_EXPORT_RESULT;

  action ( features: instance ) atcCheck
    parameter ZADT_A_ATC_CHECK_PARAM
    result [1] ZADT_A_ATC_CHECK_RESULT;
}

" Behavior pool: ZCL_ZADT_HEAVY_OPS
CLASS lhc_HeavyOps IMPLEMENTATION.
  METHOD gitExport.
    LOOP AT keys INTO DATA(ls_key).
      " Read params from input
      DATA(lv_packages) = ls_key-%param-Packages.

      " Perform export (can be slow - OData handles timeouts better)
      DATA(lv_zip) = export_packages( lv_packages ).

      " Return result
      APPEND VALUE #(
        %tky = ls_key-%tky
        ServiceId = ls_key-ServiceId
        ZipBase64 = lv_zip
        ObjectCount = lv_count
      ) TO result.
    ENDLOOP.
  ENDMETHOD.
ENDCLASS.
```

#### HTTP Streaming with OData

OData V4 supports HTTP chunked transfer encoding naturally:

```go
// Go client using standard HTTP
func (c *Client) GitExportOData(ctx context.Context, packages []string) (*GitExportResult, error) {
    url := c.baseURL + "/sap/opu/odata4/sap/zadt_heavy_o4/srvd/sap/zadt_heavy_ops/0001/HeavyOps('HEAVY')/com.sap.gateway.srvd.zadt_heavy_ops.v0001.gitExport"

    params := map[string]interface{}{
        "Packages": packages,
    }

    body, _ := json.Marshal(params)

    resp, err := c.http.Post(ctx, url, "application/json", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // OData handles large responses via chunked encoding automatically
    var result GitExportResult
    json.NewDecoder(resp.Body).Decode(&result)

    return &result, nil
}
```

#### Pros
- ✅ **Native HTTP streaming** - Chunked transfer encoding built-in
- ✅ **Better for REST patterns** - Standard HTTP verbs and status codes
- ✅ **Easier caching** - HTTP cache headers work naturally
- ✅ **Standard tooling** - curl, Postman work out of the box
- ✅ **Separate concerns** - WebSocket for stateful, OData for stateless heavy ops

#### Cons
- ❌ **Deployment complexity** - Requires BDEF, SRVD, SRVB (3 objects + behavior pool)
- ❌ **Activation issues** - RAP activation can be finicky (see Report 003)
- ❌ **No progress updates** - Standard OData doesn't support streaming progress
- ❌ **CSRF tokens** - Additional auth complexity
- ❌ **Still synchronous** - ABAP thread blocked during operation
- ❌ **Dual architecture** - Clients must support both WebSocket AND OData

#### Complexity
- **Medium-High** (5-7 days implementation)
- Create RAP objects (DDLS, BDEF, behavior pool, SRVD, SRVB)
- Test activation and publishing
- Implement Go OData client
- Handle CSRF token workflow

---

### Option 3: Custom REST Handler (ICF Node)

**Concept:** Create a dedicated ICF service with direct HTTP handling for maximum control.

#### Architecture

```
vsp (Go) ──HTTP POST──> /sap/bc/adt/zheavy/export
                                │
                                ▼
                     ZCL_ADT_HEAVY_HANDLER
                     (IF_HTTP_EXTENSION)
                                │
                                ▼
                     Direct response streaming
                     with custom chunking
```

#### ABAP Implementation

```abap
CLASS zcl_adt_heavy_handler DEFINITION
  PUBLIC
  CREATE PUBLIC.

  PUBLIC SECTION.
    INTERFACES if_http_extension.

  PRIVATE SECTION.
    METHODS handle_git_export
      IMPORTING io_request  TYPE REF TO if_http_request
                io_response TYPE REF TO if_http_response.
ENDCLASS.

CLASS zcl_adt_heavy_handler IMPLEMENTATION.

  METHOD if_http_extension~handle_request.
    DATA(lv_path) = server->request->get_header_field( '~path_info' ).

    CASE lv_path.
      WHEN '/export'.
        handle_git_export(
          io_request  = server->request
          io_response = server->response
        ).
    ENDCASE.
  ENDMETHOD.

  METHOD handle_git_export.
    " Read JSON params
    DATA(lv_body) = io_request->get_cdata( ).
    " Parse packages...

    " Set chunked transfer encoding
    io_response->set_header_field(
      name  = 'Transfer-Encoding'
      value = 'chunked'
    ).
    io_response->set_content_type( 'application/json' ).

    " Stream response in chunks
    DATA lv_offset TYPE i.
    DATA lv_chunk_size TYPE i VALUE 1048576.  " 1 MB

    WHILE lv_offset < xstrlen( lv_zip_xstring ).
      DATA(lv_chunk) = lv_zip_xstring+lv_offset(lv_chunk_size).
      io_response->append_cdata( xstring_to_base64( lv_chunk ) ).
      lv_offset += lv_chunk_size.

      " Flush to client
      COMMIT WORK.
    ENDWHILE.
  ENDMETHOD.

ENDCLASS.
```

#### ICF Configuration

1. Create service in SICF: `/sap/bc/adt/zheavy`
2. Handler class: `ZCL_ADT_HEAVY_HANDLER`
3. Activate service

#### Pros
- ✅ **Maximum control** - Direct access to HTTP stream
- ✅ **True streaming** - Can flush chunks as they're generated
- ✅ **Simple deployment** - Just class + ICF node
- ✅ **Fits ADT pattern** - Uses `/sap/bc/adt/` namespace
- ✅ **No RAP complexity** - Avoid BDEF activation issues

#### Cons
- ❌ **Manual SICF setup** - Requires transaction access
- ❌ **No standardization** - Custom protocol design
- ❌ **Limited features** - No OData metadata, no CSRF handling built-in
- ❌ **Still synchronous** - ABAP thread blocked
- ❌ **Dual architecture** - WebSocket + custom HTTP

#### Complexity
- **Low-Medium** (3-4 days implementation)
- Create ICF handler class
- Configure SICF node
- Test streaming behavior

---

### Option 4: Background RFC + Polling

**Concept:** Queue heavyweight operations to run asynchronously, poll for results.

#### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     vsp (Go)                                 │
│  1. POST /startExport  → jobId                              │
│  2. GET /status/{jobId} → {status:"running",progress:50}    │
│  3. GET /result/{jobId} → {zipBase64:"..."}                 │
└─────────────────────────────────────────────────────────────┘
                              │
                   HTTP or WebSocket
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              Job Manager (ABAP)                              │
│  - Queue operations in DB table                             │
│  - Background job processes queue                           │
│  - Store results in DB                                      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              Background Job                                  │
│  - Process heavy operation                                  │
│  - Update progress in DB                                    │
│  - Store final result                                       │
└─────────────────────────────────────────────────────────────┘
```

#### ABAP Implementation

**Queue table:**
```abap
@EndUserText.label: 'VSP Job Queue'
define table ZVSP_JOB_QUEUE {
  key job_id       : sysuuid_c36;
  job_type         : char20;        // 'GIT_EXPORT', 'ATC_CHECK'
  params           : string(0);     // JSON params
  status           : char20;        // 'QUEUED', 'RUNNING', 'DONE', 'ERROR'
  progress         : int4;          // 0-100
  result_data      : rawstring(0);  // Result (compressed)
  error_message    : string(0);
  created_by       : syuname;
  created_at       : timestampl;
  completed_at     : timestampl;
}
```

**Job manager:**
```abap
CLASS zcl_vsp_job_manager DEFINITION.
  PUBLIC SECTION.
    CLASS-METHODS:
      enqueue_job
        IMPORTING iv_type       TYPE char20
                  iv_params     TYPE string
        RETURNING VALUE(rv_job_id) TYPE sysuuid_c36,

      get_status
        IMPORTING iv_job_id     TYPE sysuuid_c36
        RETURNING VALUE(rs_status) TYPE zvsp_job_queue,

      get_result
        IMPORTING iv_job_id     TYPE sysuuid_c36
        RETURNING VALUE(rv_result) TYPE string.
ENDCLASS.

CLASS zcl_vsp_job_manager IMPLEMENTATION.
  METHOD enqueue_job.
    " Insert into queue table
    INSERT INTO zvsp_job_queue VALUES @( VALUE #(
      job_id     = cl_system_uuid=>create_uuid_c36_static( )
      job_type   = iv_type
      params     = iv_params
      status     = 'QUEUED'
      created_by = sy-uname
      created_at = cl_abap_context_info=>get_system_time( )
    ) ).
    rv_job_id = ls_job-job_id.

    " Optionally trigger background job
    CALL FUNCTION 'Z_VSP_PROCESS_JOBS' IN BACKGROUND TASK.
  ENDMETHOD.
ENDCLASS.
```

**Background processor:**
```abap
FUNCTION z_vsp_process_jobs.
  DATA: lt_jobs TYPE TABLE OF zvsp_job_queue.

  " Get queued jobs
  SELECT * FROM zvsp_job_queue
    INTO TABLE @lt_jobs
    WHERE status = 'QUEUED'
    ORDER BY created_at.

  LOOP AT lt_jobs INTO DATA(ls_job).
    " Update to RUNNING
    UPDATE zvsp_job_queue
      SET status = 'RUNNING'
      WHERE job_id = @ls_job-job_id.

    TRY.
        " Process based on type
        CASE ls_job-job_type.
          WHEN 'GIT_EXPORT'.
            DATA(lv_result) = process_git_export(
              iv_params = ls_job-params
            ).
        ENDCASE.

        " Update to DONE
        UPDATE zvsp_job_queue
          SET status = 'DONE'
              result_data = @lv_result
              completed_at = @( cl_abap_context_info=>get_system_time( ) )
          WHERE job_id = @ls_job-job_id.

      CATCH cx_root INTO DATA(lx_error).
        " Update to ERROR
        UPDATE zvsp_job_queue
          SET status = 'ERROR'
              error_message = @lx_error->get_text( )
          WHERE job_id = @ls_job-job_id.
    ENDTRY.
  ENDLOOP.
ENDFUNCTION.
```

#### Go Client

```go
// Start export job
func (c *Client) GitExportAsync(ctx context.Context, packages []string) (string, error) {
    params := map[string]interface{}{
        "packages": packages,
    }

    resp, err := c.sendRequest(ctx, "startExport", params)
    if err != nil {
        return "", err
    }

    var result struct {
        JobID string `json:"jobId"`
    }
    json.Unmarshal(resp.Data, &result)

    return result.JobID, nil
}

// Poll for completion
func (c *Client) WaitForJob(ctx context.Context, jobID string) (*GitExportResult, error) {
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            status, err := c.GetJobStatus(ctx, jobID)
            if err != nil {
                return nil, err
            }

            fmt.Printf("Progress: %d%%\n", status.Progress)

            if status.Status == "DONE" {
                return c.GetJobResult(ctx, jobID)
            } else if status.Status == "ERROR" {
                return nil, fmt.Errorf("job failed: %s", status.ErrorMessage)
            }

        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }
}
```

#### Pros
- ✅ **True async** - Client not blocked during operation
- ✅ **Cancelable** - Can cancel queued/running jobs
- ✅ **Reliable** - DB persistence survives crashes
- ✅ **Scalable** - Multiple background workers possible
- ✅ **Progress tracking** - DB updates provide real status
- ✅ **Retry support** - Failed jobs can be retried

#### Cons
- ❌ **High complexity** - DB tables, background jobs, polling logic
- ❌ **Deployment overhead** - Tables, RFCs, job scheduling
- ❌ **Resource usage** - Background work processes consumed
- ❌ **Cleanup needed** - Old jobs must be purged
- ❌ **Polling delay** - Not real-time, 2-5 second lag
- ❌ **Overkill for 60-second ops** - Better for multi-minute operations

#### Complexity
- **High** (7-10 days implementation)
- Create DB tables
- Implement job manager
- Create background processor
- Add polling logic in Go
- Test job lifecycle

---

## Comparative Analysis

| Criterion | Option 1: Chunked WS | Option 2: OData | Option 3: ICF REST | Option 4: Async RFC |
|-----------|---------------------|-----------------|-------------------|---------------------|
| **Timeout resistance** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Implementation complexity** | ⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ | ⭐ |
| **Deployment simplicity** | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐ | ⭐ |
| **Progress updates** | ⭐⭐⭐⭐⭐ | ⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **True async** | ⭐ | ⭐ | ⭐ | ⭐⭐⭐⭐⭐ |
| **Architecture consistency** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| **Fits ZADT_VSP** | ⭐⭐⭐⭐⭐ | ⭐ | ⭐⭐ | ⭐⭐ |
| **Resource efficiency** | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| **SAP best practices** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |

## Recommendations

### For Git Export (Immediate Need)

**Recommended: Option 1 - Chunked WebSocket Responses**

**Rationale:**
1. **Fits existing architecture** - Natural extension of ZADT_VSP pattern
2. **Low risk** - No new SAP infrastructure needed
3. **Fast implementation** - 2-3 days vs. weeks for alternatives
4. **Solves timeout** - 5MB chunks arrive well within 60-second window
5. **Progress updates** - Real-time feedback to user

**Implementation plan:**
1. Enhance `ZCL_VSP_GIT_SERVICE` to detect `stream: true` parameter
2. Add chunking logic to `serialize_objects` method
3. Send progress events during serialization
4. Send data chunks during ZIP encoding
5. Update Go client to assemble chunks
6. Test with 500+ object packages

**Timeline:** 2-3 days

### For Future Heavyweight Operations

**Recommended: Hybrid Approach**

Use the right tool for the job:

| Operation | Approach | Reason |
|-----------|----------|--------|
| **Git export/import** | Chunked WebSocket | 30-120 second ops, needs progress |
| **ATC checks** | Chunked WebSocket | Variable time, needs progress |
| **Profiler analysis** | Async RFC + polling | Multi-minute ops, true async needed |
| **Mass activation** | Async RFC + polling | 5-10 minute ops, blocking unacceptable |
| **Package-wide syntax** | Chunked WebSocket | Usually < 2 minutes, progress useful |

**Implementation priority:**
1. **Phase 1** (now): Chunked WebSocket for Git
2. **Phase 2** (v2.16): Extend to ATC, syntax checks
3. **Phase 3** (v2.17): Add async RFC for profiler/mass ops if needed

### General Architecture Guidelines

**When to use each approach:**

**Chunked WebSocket (Option 1):**
- ✅ Operation takes 30 seconds to 5 minutes
- ✅ Progress updates are valuable
- ✅ Stateful context helps (debug, AMDP sessions)
- ✅ Need bidirectional communication

**OData Service (Option 2):**
- ✅ Operation is truly stateless
- ✅ Standard REST semantics needed
- ✅ External systems will call (non-vsp clients)
- ✅ Caching/CDN important

**Custom ICF (Option 3):**
- ✅ Need absolute control over HTTP stream
- ✅ Custom protocol required
- ✅ Maximum performance critical
- ❌ Avoid unless Options 1-2 insufficient

**Async RFC (Option 4):**
- ✅ Operation takes > 5 minutes
- ✅ Client should not block
- ✅ Retry/recovery important
- ✅ Batch processing pattern
- ❌ Overkill for < 2 minute operations

## Implementation Details: Chunked WebSocket

### Protocol Specification

**Message types:**
```typescript
// Request
{
  id: string,
  domain: string,
  action: string,
  params: object,
  stream?: boolean,        // NEW: Enable chunking
  chunkSize?: number       // NEW: Chunk size in bytes (default: 5MB)
}

// Response types
{
  id: string,
  type: "progress",        // NEW: Progress update
  done: number,            // 0-100
  total: number,           // Total items
  message?: string         // Status message
}

{
  id: string,
  type: "chunk",           // NEW: Data chunk
  seq: number,             // Chunk sequence (1-based)
  data: string             // Base64 chunk data
}

{
  id: string,
  type: "complete",        // NEW: Final message
  success: boolean,
  totalChunks: number,
  data?: object            // Metadata (object count, etc.)
}

{
  id: string,
  type: "error",           // Error at any stage
  success: false,
  error: {
    code: string,
    message: string
  }
}
```

### ABAP Enhancement

```abap
" ZCL_VSP_GIT_SERVICE - Add methods

METHODS send_event
  IMPORTING iv_id   TYPE string
            iv_type TYPE string
            iv_data TYPE string.

METHOD send_event.
  " Build event JSON
  DATA(lv_json) = |\{"id":"{ iv_id }","type":"{ iv_type }",{ iv_data }\}|.

  " Send via message manager (requires access to WebSocket context)
  " This is the tricky part - need to pass APC context to service
  " Solution: Store context reference in session table
  DATA(lo_mm) = get_message_manager_for_session( iv_id ).
  DATA(lo_msg) = lo_mm->create_message( ).
  lo_msg->set_text( lv_json ).
  lo_mm->send( lo_msg ).
ENDMETHOD.

METHOD handle_export.
  " Check for stream parameter
  DATA lv_stream TYPE abap_bool.
  lv_stream = extract_param_bool( iv_params = is_message-params
                                   iv_name   = 'stream' ).

  IF lv_stream = abap_true.
    " Chunked export
    DATA(lv_chunk_size) = extract_param_int(
      iv_params = is_message-params
      iv_name   = 'chunkSize'
    ).
    IF lv_chunk_size IS INITIAL.
      lv_chunk_size = 5242880.  " 5 MB default
    ENDIF.

    rs_response = export_with_chunks(
      is_message    = is_message
      iv_chunk_size = lv_chunk_size
    ).
  ELSE.
    " Original single-response export
    rs_response = export_single( is_message ).
  ENDIF.
ENDMETHOD.
```

### Go Client Enhancement

```go
// pkg/adt/git_websocket.go

type ChunkAssembler struct {
    chunks    map[int]string
    mu        sync.Mutex
    completed bool
}

func (c *AMDPWebSocketClient) GitExportStreaming(
    ctx context.Context,
    packages []string,
    onProgress func(done, total int),
) (*GitExportResult, error) {

    params := map[string]interface{}{
        "packages":  packages,
        "stream":    true,
        "chunkSize": 5 * 1024 * 1024,
    }

    id := fmt.Sprintf("export_%d", c.msgID.Add(1))

    msg := WSMessage{
        ID:      id,
        Domain:  "git",
        Action:  "export",
        Params:  params,
    }

    // Send request
    if err := c.sendMessage(msg); err != nil {
        return nil, err
    }

    // Collect chunks
    assembler := &ChunkAssembler{
        chunks: make(map[int]string),
    }

    var objectCount int

    for {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
            // Read next message
            var resp WSResponse
            if err := c.readResponse(&resp); err != nil {
                return nil, err
            }

            if resp.ID != id {
                // Not our response, skip
                continue
            }

            // Parse response type
            var baseResp struct {
                Type string `json:"type"`
            }
            json.Unmarshal(resp.Data, &baseResp)

            switch baseResp.Type {
            case "progress":
                var prog struct {
                    Done  int `json:"done"`
                    Total int `json:"total"`
                }
                json.Unmarshal(resp.Data, &prog)
                if onProgress != nil {
                    onProgress(prog.Done, prog.Total)
                }

            case "chunk":
                var chunk struct {
                    Seq  int    `json:"seq"`
                    Data string `json:"data"`
                }
                json.Unmarshal(resp.Data, &chunk)
                assembler.AddChunk(chunk.Seq, chunk.Data)

            case "complete":
                var complete struct {
                    TotalChunks  int                    `json:"totalChunks"`
                    Data         map[string]interface{} `json:"data"`
                }
                json.Unmarshal(resp.Data, &complete)
                objectCount = int(complete.Data["objectCount"].(float64))

                // Assemble chunks
                zipBase64 := assembler.Assemble(complete.TotalChunks)

                return &GitExportResult{
                    ObjectCount: objectCount,
                    ZipBase64:   zipBase64,
                }, nil

            case "error":
                return nil, fmt.Errorf("export failed: %v", resp.Error)
            }
        }
    }
}

func (a *ChunkAssembler) AddChunk(seq int, data string) {
    a.mu.Lock()
    defer a.mu.Unlock()
    a.chunks[seq] = data
}

func (a *ChunkAssembler) Assemble(totalChunks int) string {
    a.mu.Lock()
    defer a.mu.Unlock()

    var sb strings.Builder
    for i := 1; i <= totalChunks; i++ {
        sb.WriteString(a.chunks[i])
    }
    return sb.String()
}
```

## Migration Path

### Phase 1: Git Export (v2.16)
1. Implement chunked WebSocket in `ZCL_VSP_GIT_SERVICE`
2. Update Go client with chunk assembly
3. Add progress callbacks to CLI
4. Test with 500+ object packages
5. Document protocol in `reports/`

### Phase 2: Extend to ATC (v2.17)
1. Add chunked responses to ATC check tool
2. Stream findings as they're discovered
3. Progress updates during analysis

### Phase 3: Profiler Support (v2.18)
1. Evaluate if async RFC needed for trace analysis
2. Implement job queue if justified by usage patterns

## Testing Strategy

### Unit Tests
- Chunk assembly logic
- Progress event handling
- Error handling during chunking

### Integration Tests
```go
func TestGitExportLargePackage(t *testing.T) {
    // Export 500+ objects
    client := NewAMDPWebSocketClient(...)
    client.Connect(ctx)

    var progressUpdates int
    result, err := client.GitExportStreaming(
        ctx,
        []string{"$ZRAY"},
        func(done, total int) {
            progressUpdates++
            t.Logf("Progress: %d/%d", done, total)
        },
    )

    require.NoError(t, err)
    assert.Greater(t, result.ObjectCount, 500)
    assert.Greater(t, progressUpdates, 0)

    // Verify ZIP integrity
    zipData, _ := base64.StdEncoding.DecodeString(result.ZipBase64)
    reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
    require.NoError(t, err)
    assert.Greater(t, len(reader.File), 500)
}
```

### Performance Tests
- Measure chunk size vs. throughput
- Test with 1000+ object packages
- Verify memory usage stays reasonable

## Security Considerations

1. **Chunk size limits** - Enforce max chunk size to prevent memory exhaustion
2. **Rate limiting** - Prevent abuse of heavyweight operations
3. **Authorization** - Existing SAP auth applies
4. **Data validation** - Validate chunk sequences, detect missing chunks

## Conclusion

The **chunked WebSocket approach (Option 1)** provides the best balance of:
- Low implementation complexity
- High architectural consistency
- Effective timeout mitigation
- Real-time progress updates
- Minimal deployment overhead

This approach solves the immediate Git export timeout issue while establishing a pattern that can be extended to other heavyweight operations in future releases.

For truly long-running operations (> 5 minutes), the async RFC pattern (Option 4) remains available as a future enhancement if usage patterns justify the additional complexity.

## Related Documents

- `2025-12-22-003-websocket-abapgit-integration.md` - Git WebSocket design
- `2025-12-18-002-websocket-rfc-handler.md` - ZADT_VSP architecture
- `2025-12-08-003-rap-odata-service-lessons.md` - OData V4 implementation lessons
- `pkg/adt/amdp_websocket.go` - Current WebSocket client implementation
