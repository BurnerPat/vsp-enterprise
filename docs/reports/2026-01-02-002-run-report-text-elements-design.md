# RunReport & SetTextElements Tools Design

**Date:** 2026-01-02
**Report ID:** 002
**Subject:** API Design for Report Execution and Text Elements Management
**Status:** Design Phase

---

## Overview

Two new tools for ABAP report execution and text element management:

1. **RunReport** - Execute selection-screen reports with parameters/variants, capture ALV output
2. **SetTextElements** - Manage program text elements (selection texts, text symbols)

Both tools will be available in:
- Go MCP server (focused + expert modes)
- ABAP WebSocket service (ZADT_VSP)

---

## Tool 1: RunReport

### Purpose

Execute ABAP reports (selection-screen programs) programmatically with:
- Individual parameter values
- Named variants
- ALV output capture (when available)

### Use Cases

1. **AI-driven testing**: Run reports to verify functionality
2. **Data extraction**: Execute reports and capture ALV results as JSON
3. **Batch processing**: Run reports with different parameter sets
4. **Debugging support**: Execute specific scenarios to trigger breakpoints

### API Design

#### MCP Tool: RunReport

```json
{
  "name": "RunReport",
  "description": "Execute an ABAP report with selection screen parameters or variant. Optionally captures ALV output.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "report": {
        "type": "string",
        "description": "Report program name (e.g., 'ZTEST_REPORT', 'RFBILA00')"
      },
      "variant": {
        "type": "string",
        "description": "Named variant to use (optional, mutually exclusive with params)"
      },
      "params": {
        "type": "string",
        "description": "JSON object with parameter values: {\"P_BUKRS\":\"1000\",\"S_GJAHR\":{\"low\":\"2024\",\"high\":\"2025\"}}"
      },
      "capture_alv": {
        "type": "boolean",
        "description": "Capture ALV output if report produces one (default: true)"
      },
      "max_rows": {
        "type": "number",
        "description": "Maximum rows to return from ALV capture (default: 1000)"
      },
      "background": {
        "type": "boolean",
        "description": "Run in background (returns job info instead of waiting)"
      }
    },
    "required": ["report"]
  }
}
```

#### Parameter Types

| Parameter Type | JSON Format | Example |
|---------------|-------------|---------|
| Single value (P_*) | `"value"` | `"P_BUKRS": "1000"` |
| Select-option range (S_*) | `{"low": "x", "high": "y", "sign": "I", "option": "BT"}` | `"S_GJAHR": {"low": "2024", "high": "2025"}` |
| Select-option list | `[{"low": "x"}, {"low": "y"}]` | `"S_BUKRS": [{"low": "1000"}, {"low": "2000"}]` |
| Checkbox | `true` or `false` | `"P_TEST": true` |
| Radio button | `"X"` or `""` | `"R_DETAIL": "X"` |

#### Response Format

```json
{
  "status": "success",
  "report": "ZTEST_REPORT",
  "variant": "DEFAULT",
  "runtime_ms": 1234,
  "messages": [
    {"type": "S", "message": "Report completed successfully"},
    {"type": "I", "message": "42 records processed"}
  ],
  "alv_captured": true,
  "alv_data": {
    "columns": [
      {"name": "BUKRS", "description": "Company Code", "type": "CHAR4"},
      {"name": "GJAHR", "description": "Fiscal Year", "type": "NUMC4"},
      {"name": "DMBTR", "description": "Amount", "type": "CURR13"}
    ],
    "rows": [
      {"BUKRS": "1000", "GJAHR": "2024", "DMBTR": "12345.67"},
      {"BUKRS": "1000", "GJAHR": "2025", "DMBTR": "23456.78"}
    ],
    "total_rows": 42,
    "truncated": false
  }
}
```

#### Error Response

```json
{
  "status": "error",
  "report": "ZTEST_REPORT",
  "error_type": "SELECTION_SCREEN",
  "message": "Mandatory parameter P_BUKRS not supplied",
  "messages": [
    {"type": "E", "message": "Enter a company code"}
  ]
}
```

### ABAP Implementation

#### WebSocket Action: runReport

```json
{
  "action": "runReport",
  "report": "ZTEST_REPORT",
  "variant": "DEFAULT",
  "params": {"P_BUKRS": "1000"},
  "capture_alv": true,
  "max_rows": 1000
}
```

#### Core ABAP Logic

```abap
METHOD run_report.
  DATA: lt_rsparams TYPE TABLE OF rsparams,
        lt_messages TYPE TABLE OF bal_s_msg,
        lr_data     TYPE REF TO data.

  " Option 1: Use variant
  IF iv_variant IS NOT INITIAL.
    SUBMIT (iv_report)
      USING SELECTION-SET iv_variant
      AND RETURN.

  " Option 2: Use parameters
  ELSEIF it_params IS NOT INITIAL.
    " Convert params to RSPARAMS format
    LOOP AT it_params INTO DATA(ls_param).
      APPEND VALUE rsparams(
        selname = ls_param-name
        kind    = ls_param-kind      " P=parameter, S=select-option
        sign    = ls_param-sign      " I=include
        option  = ls_param-option    " EQ, BT, etc.
        low     = ls_param-low
        high    = ls_param-high
      ) TO lt_rsparams.
    ENDLOOP.

    SUBMIT (iv_report)
      WITH SELECTION-TABLE lt_rsparams
      AND RETURN.

  " Option 3: No params (use defaults)
  ELSE.
    SUBMIT (iv_report) AND RETURN.
  ENDIF.

  " Capture ALV if requested
  IF iv_capture_alv = abap_true.
    TRY.
        " Set ALV capture mode BEFORE submit
        cl_salv_bs_runtime_info=>set(
          display  = abap_false
          metadata = abap_true
          data     = abap_true ).

        " Re-run with capture
        SUBMIT (iv_report)
          USING SELECTION-SET iv_variant
          AND RETURN.

        " Get captured data
        cl_salv_bs_runtime_info=>get_data_ref(
          IMPORTING r_data = lr_data ).

        " Convert to JSON...

      CATCH cx_salv_bs_sc_runtime_info.
        " Report doesn't produce ALV
      CLEANUP.
        cl_salv_bs_runtime_info=>clear_all( ).
    ENDTRY.
  ENDIF.
ENDMETHOD.
```

#### ALV Capture Details

The `CL_SALV_BS_RUNTIME_INFO` class intercepts ALV output:

```abap
" Enable capture (must be called BEFORE SUBMIT)
cl_salv_bs_runtime_info=>set(
  display  = abap_false    " Don't display ALV
  metadata = abap_true     " Capture field catalog
  data     = abap_true ).  " Capture data

" After SUBMIT, retrieve data
DATA: lr_data     TYPE REF TO data,
      lt_fieldcat TYPE salv_bs_t_runtime_fieldcat.

" Get field catalog (column metadata)
cl_salv_bs_runtime_info=>get_metadata(
  IMPORTING t_fieldcat = lt_fieldcat ).

" Get actual data
cl_salv_bs_runtime_info=>get_data_ref(
  IMPORTING r_data = lr_data ).

" Always clear after use
cl_salv_bs_runtime_info=>clear_all( ).
```

---

## Tool 2: SetTextElements

### Purpose

Manage program text elements:
- **Selection texts**: Descriptions for parameters/select-options (P_BUKRS -> "Company Code")
- **Text symbols**: TEXT-001, TEXT-002 etc. used in WRITE statements

### Use Cases

1. **Automated translation**: Set texts in multiple languages
2. **Documentation sync**: Keep parameter descriptions consistent
3. **Code generation**: Create complete programs with proper texts
4. **Maintenance**: Bulk update text elements

### API Design

#### MCP Tool: SetTextElements

```json
{
  "name": "SetTextElements",
  "description": "Set program text elements (selection texts and text symbols)",
  "inputSchema": {
    "type": "object",
    "properties": {
      "program": {
        "type": "string",
        "description": "Program name (e.g., 'ZTEST_REPORT')"
      },
      "language": {
        "type": "string",
        "description": "Language key (default: EN)"
      },
      "selection_texts": {
        "type": "string",
        "description": "JSON object mapping parameter names to descriptions: {\"P_BUKRS\":\"Company Code\",\"S_GJAHR\":\"Fiscal Year\"}"
      },
      "text_symbols": {
        "type": "string",
        "description": "JSON object mapping symbol IDs to text: {\"001\":\"Processing started\",\"002\":\"Records processed:\"}"
      }
    },
    "required": ["program"]
  }
}
```

#### Response Format

```json
{
  "status": "success",
  "program": "ZTEST_REPORT",
  "language": "EN",
  "selection_texts_set": 5,
  "text_symbols_set": 3,
  "details": [
    {"type": "selection_text", "name": "P_BUKRS", "text": "Company Code", "status": "created"},
    {"type": "selection_text", "name": "P_GJAHR", "text": "Fiscal Year", "status": "updated"},
    {"type": "text_symbol", "id": "001", "text": "Processing started", "status": "created"}
  ]
}
```

#### MCP Tool: GetTextElements

```json
{
  "name": "GetTextElements",
  "description": "Get program text elements (selection texts and text symbols)",
  "inputSchema": {
    "type": "object",
    "properties": {
      "program": {
        "type": "string",
        "description": "Program name"
      },
      "language": {
        "type": "string",
        "description": "Language key (default: EN)"
      }
    },
    "required": ["program"]
  }
}
```

#### GetTextElements Response

```json
{
  "program": "ZTEST_REPORT",
  "language": "EN",
  "selection_texts": {
    "P_BUKRS": "Company Code",
    "P_GJAHR": "Fiscal Year",
    "S_MATNR": "Material Number"
  },
  "text_symbols": {
    "001": "Processing started",
    "002": "Records processed:",
    "003": "Completed successfully"
  }
}
```

### ABAP Implementation

#### WebSocket Action: setTextElements

```json
{
  "action": "setTextElements",
  "program": "ZTEST_REPORT",
  "language": "EN",
  "selection_texts": {"P_BUKRS": "Company Code"},
  "text_symbols": {"001": "Processing started"}
}
```

#### Core ABAP Logic

```abap
METHOD set_text_elements.
  DATA: lt_textpool TYPE TABLE OF textpool.

  " Read existing text pool
  READ TEXTPOOL iv_program INTO lt_textpool LANGUAGE iv_language.

  " Update/add selection texts (ID = 'S')
  LOOP AT it_sel_texts INTO DATA(ls_sel).
    READ TABLE lt_textpool ASSIGNING FIELD-SYMBOL(<fs>)
      WITH KEY id = 'S' key = ls_sel-name.
    IF sy-subrc = 0.
      <fs>-entry = ls_sel-text.
    ELSE.
      APPEND VALUE textpool(
        id    = 'S'              " Selection text
        key   = ls_sel-name      " Parameter name (8 chars)
        entry = ls_sel-text      " Description
      ) TO lt_textpool.
    ENDIF.
  ENDLOOP.

  " Update/add text symbols (ID = 'I')
  LOOP AT it_symbols INTO DATA(ls_sym).
    READ TABLE lt_textpool ASSIGNING <fs>
      WITH KEY id = 'I' key = ls_sym-id.
    IF sy-subrc = 0.
      <fs>-entry = ls_sym-text.
    ELSE.
      APPEND VALUE textpool(
        id    = 'I'              " Text symbol
        key   = ls_sym-id        " 3-char ID (001, 002, etc.)
        entry = ls_sym-text      " Text content
      ) TO lt_textpool.
    ENDIF.
  ENDLOOP.

  " Write updated text pool
  INSERT TEXTPOOL iv_program FROM lt_textpool LANGUAGE iv_language.
  IF sy-subrc <> 0.
    RAISE EXCEPTION TYPE cx_failed.
  ENDIF.
ENDMETHOD.
```

#### Alternative: Using RS_TEXTPOOL_INSERT

```abap
" For more control, use function module
CALL FUNCTION 'RS_TEXTPOOL_INSERT'
  EXPORTING
    program        = iv_program
    language       = iv_language
  TABLES
    textpool_table = lt_textpool
  EXCEPTIONS
    insert_error   = 1
    OTHERS         = 2.
```

#### Text Pool ID Types

| ID | Type | Key Format | Example |
|----|------|------------|---------|
| R | Title | (empty) | Program title |
| S | Selection text | Parameter name (8 chars) | `P_BUKRS` |
| I | Text symbol | 3-digit ID | `001`, `002` |
| H | List header | (varies) | Column headers |

---

## Go Implementation Plan

### File Structure

```
pkg/adt/
├── reports.go          # NEW: RunReport, GetTextElements, SetTextElements
└── reports_test.go     # NEW: Unit tests

internal/mcp/
└── server.go           # Add tool handlers
```

### ADT Client Methods

```go
// pkg/adt/reports.go

// RunReportParams contains parameters for report execution
type RunReportParams struct {
    Report     string            `json:"report"`
    Variant    string            `json:"variant,omitempty"`
    Params     map[string]any    `json:"params,omitempty"`
    CaptureALV bool              `json:"capture_alv"`
    MaxRows    int               `json:"max_rows,omitempty"`
}

// RunReportResult contains report execution results
type RunReportResult struct {
    Status     string          `json:"status"`
    Report     string          `json:"report"`
    RuntimeMs  int64           `json:"runtime_ms"`
    Messages   []ReportMessage `json:"messages"`
    ALVCaptured bool           `json:"alv_captured"`
    ALVData    *ALVData        `json:"alv_data,omitempty"`
}

// ALVData contains captured ALV output
type ALVData struct {
    Columns   []ALVColumn        `json:"columns"`
    Rows      []map[string]any   `json:"rows"`
    TotalRows int                `json:"total_rows"`
    Truncated bool               `json:"truncated"`
}

// RunReport executes an ABAP report via ZADT_VSP WebSocket
func (c *Client) RunReport(ctx context.Context, params RunReportParams) (*RunReportResult, error) {
    // Uses WebSocket action: runReport
}

// TextElements contains program text elements
type TextElements struct {
    Program        string            `json:"program"`
    Language       string            `json:"language"`
    SelectionTexts map[string]string `json:"selection_texts"`
    TextSymbols    map[string]string `json:"text_symbols"`
}

// GetTextElements retrieves program text elements
func (c *Client) GetTextElements(ctx context.Context, program, language string) (*TextElements, error) {
    // Uses WebSocket action: getTextElements
}

// SetTextElements updates program text elements
func (c *Client) SetTextElements(ctx context.Context, elements TextElements) error {
    // Uses WebSocket action: setTextElements
}
```

### MCP Server Integration

```go
// internal/mcp/server.go

// In registerTools():
s.registerTool("RunReport", "Execute ABAP report with parameters/variant, capture ALV output", ...)
s.registerTool("GetTextElements", "Get program text elements", ...)
s.registerTool("SetTextElements", "Set program text elements", ...)

// Tool handlers
case "RunReport":
    report, _ := getString(args, "report")
    variant, _ := getString(args, "variant")
    paramsJSON, _ := getString(args, "params")
    captureALV, _ := getBool(args, "capture_alv")
    maxRows, _ := getInt(args, "max_rows")

    var params map[string]any
    if paramsJSON != "" {
        json.Unmarshal([]byte(paramsJSON), &params)
    }

    result, err := s.adtClient.RunReport(ctx, adt.RunReportParams{
        Report:     report,
        Variant:    variant,
        Params:     params,
        CaptureALV: captureALV,
        MaxRows:    maxRows,
    })
    // ...
```

---

## ZADT_VSP Integration

### New WebSocket Actions

Add to `ZCL_VSP_RFC_SERVICE` (or create new `ZCL_VSP_REPORT_SERVICE`):

| Action | Description |
|--------|-------------|
| `runReport` | Execute report with params/variant |
| `getTextElements` | Read program text pool |
| `setTextElements` | Update program text pool |
| `getVariants` | List available variants for report |

### Message Formats

#### runReport Request
```json
{
  "action": "runReport",
  "report": "ZTEST_REPORT",
  "variant": "",
  "params": {"P_BUKRS": "1000", "S_GJAHR": {"low": "2024"}},
  "capture_alv": true,
  "max_rows": 1000
}
```

#### runReport Response
```json
{
  "status": "success",
  "report": "ZTEST_REPORT",
  "runtime_ms": 456,
  "messages": [...],
  "alv_captured": true,
  "alv_data": {...}
}
```

---

## Tool Visibility

| Tool | Focused Mode | Expert Mode | WebSocket |
|------|-------------|-------------|-----------|
| RunReport | Yes | Yes | Yes |
| GetTextElements | Yes | Yes | Yes |
| SetTextElements | Yes | Yes | Yes |
| GetVariants | No | Yes | Yes |

---

## Implementation Order

1. **Phase 1: ABAP Service** (ZCL_VSP_REPORT_SERVICE)
   - [ ] Create new class in $ZADT_VSP
   - [ ] Implement `runReport` action with basic params
   - [ ] Implement ALV capture logic
   - [ ] Implement `getTextElements` action
   - [ ] Implement `setTextElements` action

2. **Phase 2: Go Client** (pkg/adt/reports.go)
   - [ ] Define types (RunReportParams, TextElements, etc.)
   - [ ] Implement WebSocket client methods
   - [ ] Add unit tests with mocks

3. **Phase 3: MCP Tools** (internal/mcp/server.go)
   - [ ] Register tools
   - [ ] Implement handlers
   - [ ] Integration tests

4. **Phase 4: Documentation**
   - [ ] Update README.md
   - [ ] Update CLAUDE.md tool count
   - [ ] Export updated ABAP to embedded/abap/

---

## Security Considerations

1. **Report Authorization**: SUBMIT respects SAP authorization checks
2. **ALV Data**: May contain sensitive business data - consider max_rows limit
3. **Text Elements**: Write access requires ABAP development authorization
4. **Variant Access**: Users can only use variants they have access to

---

## Limitations

1. **Interactive Reports**: Only works with non-interactive reports (SUBMIT AND RETURN)
2. **ALV Types**: Only captures CL_SALV_TABLE-based ALV, not classic REUSE_ALV_*
3. **Selection Screen**: Complex dynpro screens may not work (only standard SEL-OPT)
4. **Background**: Background execution requires job scheduling authorization

---

## References

- `CL_SALV_BS_RUNTIME_INFO` - ALV data capture
- `RS_TEXTPOOL_INSERT` - Text pool manipulation
- `RSPARAMS` - Selection screen parameter structure
- `TEXTPOOL` - Text pool table structure
