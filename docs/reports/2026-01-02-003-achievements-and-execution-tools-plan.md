# VSP Achievements & Execution Tools Plan

**Date:** 2026-01-02
**Report ID:** 003
**Subject:** Current achievements summary and execution tools roadmap

---

## Current Achievements (v2.18+)

### Tool Count
| Mode | Tools |
|------|-------|
| Focused | 55 |
| Expert | 103 |

### Recently Added (This Session)

| Tool | Domain | Description |
|------|--------|-------------|
| **RunReport** | report | Execute ABAP reports with params/variants, capture ALV |
| **GetVariants** | report | List report variants |
| **GetTextElements** | report | Get program text elements |
| **SetTextElements** | report | Set program text elements |

### Existing Execution Tools

| Tool | Type | Via | Notes |
|------|------|-----|-------|
| **CallRFC** | Function Module | WebSocket (ZADT_VSP) | Pass params as JSON, returns result |
| **ExecuteABAP** | Arbitrary Code | ADT Unit Test | Wraps code in test class, executes |
| **RunReport** | Report | WebSocket (ZADT_VSP) | Selection screen params, ALV capture |
| **RunUnitTests** | Unit Tests | ADT REST | Execute ABAP Unit tests |
| **RunATCCheck** | Quality Check | ADT REST | ATC code analysis |

---

## Proposed New Execution Tools

### Priority 1: RunStaticMethod

Execute static class methods without instantiation.

```
Tool: RunStaticMethod
Parameters:
  - class: ZCL_MY_CLASS
  - method: MY_STATIC_METHOD
  - params: {"IV_PARAM1": "value1", "IV_PARAM2": 123}
  - return_type: "TABLE" | "STRUCTURE" | "SCALAR" (auto-detect)

Returns:
  - Exporting parameters as JSON
  - Return value if any
  - Runtime statistics
```

**Implementation approach:**
- Add action `runStaticMethod` to ZCL_VSP_REPORT_SERVICE (or new ZCL_VSP_EXEC_SERVICE)
- Use dynamic CALL METHOD with parameter table
- Introspect method signature via RTTI for parameter types

**ABAP skeleton:**
```abap
DATA(lo_class) = cl_abap_typedescr=>describe_by_name( iv_class ).
DATA(lo_method) = CAST cl_abap_classdescr( lo_class )->get_method( iv_method ).

" Build parameter table dynamically
CALL METHOD (iv_class)=>(iv_method)
  PARAMETER-TABLE lt_params
  EXCEPTION-TABLE lt_exceptions.
```

### Priority 2: RunInstanceMethod

Execute instance methods with configurable instantiation.

```
Tool: RunInstanceMethod
Parameters:
  - class: ZCL_MY_CLASS
  - method: MY_METHOD
  - constructor_params: {"IV_CONFIG": "value"} (optional)
  - method_params: {"IV_INPUT": "data"}
  - factory_method: "GET_INSTANCE" (optional, alternative to constructor)

Returns:
  - Method results as JSON
  - Instance state if requested
```

**Complexity:** Higher - needs to handle:
- Constructor parameters
- Factory methods (singleton patterns)
- Instance lifecycle (create → call → destroy)
- Stateful vs stateless execution

### Priority 3: RunFM (Enhanced CallRFC)

Current `CallRFC` is basic. Enhanced version would add:

```
Tool: RunFM (or enhance CallRFC)
Enhancements:
  - TABLES parameters (not just IMPORTING/EXPORTING)
  - CHANGING parameters
  - Structured parameter introspection
  - FM signature discovery (like GetSource but for FM interface)
```

---

## Implementation Comparison

| Tool | Complexity | Risk | Value |
|------|------------|------|-------|
| RunStaticMethod | Medium | Low | High - most utility classes use static methods |
| RunInstanceMethod | High | Medium | Medium - requires instance management |
| Enhanced RunFM | Medium | Low | High - FMs are backbone of SAP |

---

## Architecture Decision: JSON Parsing

### Current: PCRE Regex
- **Pro:** No dependencies, consistent with other VSP services
- **Con:** Fragile for complex nested JSON

### Alternative: /UI2/CL_JSON
- **Pro:** Robust, handles all JSON edge cases
- **Con:** Availability varies by system, different method names

### Recommendation
Keep regex for simple key-value extraction. For complex nested structures (like method parameters), consider:
1. Hybrid approach: regex for outer structure, JSON class for nested objects
2. Or accept JSON as serialized string parameters (current approach)

---

## Files to Modify

| File | Changes |
|------|---------|
| `embedded/abap/zcl_vsp_exec_service.clas.abap` | NEW - execution service |
| `embedded/abap/zcl_vsp_apc_handler.clas.abap` | Register exec service |
| `pkg/adt/exec.go` | NEW - Go client methods |
| `internal/mcp/server.go` | Tool registrations + handlers |

---

## Tool Groups Update

```go
"E": { // Execution tools (via ZADT_VSP WebSocket)
    "CallRFC", "RunReport", "RunStaticMethod", "RunInstanceMethod",
},
"R": { // Report tools (subset of E, or merge?)
    "RunReport", "GetVariants", "GetTextElements", "SetTextElements",
},
```

---

## Next Steps

1. **Immediate:** Design `RunStaticMethod` API and ABAP implementation
2. **Short-term:** Implement and test with simple static methods
3. **Medium-term:** Add parameter introspection (get method signature)
4. **Future:** `RunInstanceMethod` with factory support

---

## Session Summary

### Completed Today
- [x] ZCL_VSP_REPORT_SERVICE - 4 actions (runReport, getVariants, getTextElements, setTextElements)
- [x] Go client methods in `pkg/adt/reports.go`
- [x] MCP tool handlers (RunReport, GetVariants, GetTextElements, SetTextElements)
- [x] Tool group "R" for report tools
- [x] Exported ABAP to embedded/abap/
- [x] End-to-end testing (WebSocket works, basic operations functional)

### Known Limitations
- RunReport times out on reports requiring mandatory selection screen input (expected)
- Need variant or params for reports with selection screens
- ALV capture only works for reports using CL_SALV_* or REUSE_ALV_*

### Version
- ZADT_VSP: 2.3.0 (domains: rfc, debug, amdp, git, report)
