# AMC & Async Architecture Analysis

**Date:** 2026-01-06
**Report ID:** 002
**Subject:** Can we implement AMC tools via existing WebSocket interface?

---

## Current Architecture

```
vsp (Go) ──WebSocket──> ZADT_VSP (APC Handler)
                              │
                              ├── rfc domain      → ZCL_VSP_RFC_SERVICE
                              ├── debug domain    → ZCL_VSP_DEBUG_SERVICE
                              ├── amdp domain     → ZCL_VSP_AMDP_SERVICE
                              ├── git domain      → ZCL_VSP_GIT_SERVICE
                              └── report domain   → ZCL_VSP_REPORT_SERVICE
```

**Key insight**: APC (ABAP Push Channel) is **built on top of AMC** (ABAP Messaging Channels).

When we use `cl_apc_wsp_ext_stateful_base`, SAP internally uses AMC for message delivery. Our WebSocket IS an AMC channel!

---

## The Problem

APC handlers cannot:
- Use `WAIT UP TO ... SECONDS` (blocks the work process)
- Use `CALL FUNCTION IN BACKGROUND TASK` with COMMIT (session issue)
- Make long HTTP calls (timeout)

This is why RunReport via RFC works - it stays within the RFC call boundary.

---

## Option 1: Extend Existing WebSocket with Async Pattern

**Already implemented!** Our `RunReportAsync` / `GetAsyncResult` pattern in Go:

```
Client                    vsp (Go)                    SAP (ZADT_VSP)
   │                         │                              │
   │── RunReportAsync ──────>│                              │
   │                         │── spawn goroutine ──────────>│
   │<── task_id ─────────────│                              │
   │                         │      [goroutine has own WS]  │
   │                         │<────────── result ───────────│
   │── GetAsyncResult ──────>│                              │
   │<── result ──────────────│                              │
```

**Pros**: Already working, no SAP-side changes needed
**Cons**: Each async task needs own WebSocket connection

---

## Option 2: AMC-based Async (SAP-side background)

```
Client ──WS──> APC Handler ──AMC──> Background Worker ──HTTP──> LLM
                    │                      │
                    │<────────AMC──────────┘
                    │                  (can use WAIT!)
               sends result
```

### Implementation Steps

1. **Create AMC Application** (one-time setup in SAPC):
   ```
   Application: ZADT_VSP_ASYNC
   Channels:
     /vsp/requests   - APC publishes, Background subscribes
     /vsp/responses  - Background publishes, APC subscribes
   ```

2. **Modify APC Handler** to:
   - Subscribe to `/vsp/responses` on startup
   - Forward results to WebSocket when received

3. **Create Background Worker Class**:
   ```abap
   CLASS zcl_vsp_async_worker DEFINITION.
     INTERFACES if_amc_message_receiver_text.
   ENDCLASS.

   CLASS zcl_vsp_async_worker IMPLEMENTATION.
     METHOD if_amc_message_receiver_text~receive.
       " Parse request
       " Do HTTP call (can use WAIT!)
       " Publish result to /vsp/responses
     ENDMETHOD.
   ENDCLASS.
   ```

4. **Start Background Listener** (job or daemon):
   ```abap
   DATA(lo_consumer) = cl_amc_channel_manager=>create_message_consumer(
     i_application_id = 'ZADT_VSP_ASYNC'
     i_channel_id     = '/vsp/requests'
   ).
   lo_consumer->start_message_delivery( i_receiver = lo_worker ).
   ```

### Complexity Assessment

| Component | Effort | Risk |
|-----------|--------|------|
| AMC App (SAPC) | Low | Config only |
| APC subscription | Medium | New code in handler |
| Background worker | Medium | New class |
| Worker lifecycle | **High** | Needs job/daemon management |
| Error handling | High | Async = harder debugging |

---

## Option 3: Use WebSocket + RFC Background Task

Simpler hybrid approach:

```abap
" In APC handler
METHOD handle_async_request.
  " Generate correlation ID
  DATA(lv_corr_id) = cl_system_uuid=>create_uuid_c32_static( ).

  " Start background RFC
  CALL FUNCTION 'ZADT_ASYNC_WORKER' IN BACKGROUND TASK
    EXPORTING
      iv_corr_id  = lv_corr_id
      iv_request  = iv_request_json.
  COMMIT WORK.

  " Return correlation ID immediately
  rv_response = |"task_id":"{ lv_corr_id }"|.
ENDMETHOD.

" Background RFC publishes result to shared memory/database
" Separate polling endpoint retrieves result
```

**Pros**: Simpler than full AMC, uses existing patterns
**Cons**: Still need polling, shared storage for results

---

## Option 4: No New SAP Code - Pure Go Async

What we have now with `RunReportAsync`:

```go
// Each async task gets its own WebSocket connection
go func() {
    wsClient := adt.NewAMDPWebSocketClient(...)
    wsClient.Connect(ctx)
    result := wsClient.RunReport(...)
    // Store result
}()
```

**Pros**:
- Already working
- No SAP-side changes
- Simpler architecture

**Cons**:
- Multiple WebSocket connections
- Go process must stay alive

---

## Recommendation

### For Current Use Case (LLM Async Calls)

**Option 1 (Go-side async) is sufficient** because:
1. Already implemented and working
2. No SAP deployment needed
3. vsp process manages task lifecycle

### For Future "SAP as Async Worker" Pattern

**Option 2 (AMC) would be needed** for scenarios like:
- SAP calling external LLMs autonomously
- Long-running batch operations initiated from ABAP
- Event-driven ABAP workflows

### AMC Tools Assessment

| Tool | Via WebSocket? | Complexity |
|------|---------------|------------|
| AMCCreateApp | ❌ No - SAPC config | N/A |
| AMCPublish | ✅ Yes - add service method | Low |
| AMCSubscribe | ⚠️ Partial - APC subscribes | Medium |
| AMCListApps | ✅ Yes - query tables | Low |

**AMCPublish** could be useful for:
- Triggering other SAP processes
- Integration with existing AMC-based applications
- Cross-system messaging

```abap
" Could add to zcl_vsp_rfc_service
METHOD handle_amc_publish.
  DATA(lo_producer) = CAST if_amc_message_producer_text(
    cl_amc_channel_manager=>create_message_producer(
      i_application_id = iv_app_id
      i_channel_id     = iv_channel_id
    )
  ).
  lo_producer->send( i_message = iv_message ).
ENDMETHOD.
```

---

## Conclusion

**For async LLM calls**: Current Go-side async (Option 1) is fine.

**For deeper SAP integration**: AMCPublish tool would be useful addition to ZADT_VSP, but:
- AMCCreateApp is config, not runtime
- AMCSubscribe requires persistent background process
- AMCListApps is nice-to-have

### Recommended Immediate Action

None - current architecture handles async adequately.

### Recommended Future Enhancement

Add `AMCPublish` action to `zcl_vsp_rfc_service`:
```json
{"domain": "rfc", "action": "amcPublish", "params": {
  "application": "ZAPP",
  "channel": "/channel",
  "message": "..."
}}
```

This enables vsp to trigger AMC-based workflows in SAP without full AMC tooling.
