# Test Case Extraction & Isolated Replay for Rapid Patch Testing

**Date:** 2025-12-21
**Report ID:** 002
**Subject:** Variable History, Test Extraction, Mocked Playgrounds
**Status:** Design Document

---

## Executive Summary

**The Problem:** Testing patches is slow because:
1. Full E2E tests take minutes/hours
2. Setting up test data is painful
3. External dependencies (DB, RFC, HTTP) are unpredictable
4. You can't easily reproduce the exact failure conditions

**The Solution:** Capture real execution → Extract test cases → Replay in isolation

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  PRODUCTION     │     │  EXTRACTION      │     │  PLAYGROUND     │
│  EXECUTION      │ ──► │  ENGINE          │ ──► │  (Isolated)     │
│                 │     │                  │     │                 │
│  • Variables    │     │  • Inputs        │     │  • Mock DB      │
│  • DB reads     │     │  • Outputs       │     │  • Mock RFC     │
│  • RFC calls    │     │  • Dependencies  │     │  • Fast replay  │
│  • State changes│     │  • Assertions    │     │  • Patch & test │
└─────────────────┘     └──────────────────┘     └─────────────────┘
```

**Benefit:** Patch iteration goes from **minutes → seconds**

---

## Part 1: Variable History Recording

### What to Capture

At each debug step (or configurable intervals):

```go
type ExecutionFrame struct {
    StepNumber    int                    `json:"step"`
    Timestamp     time.Time              `json:"timestamp"`
    Location      CodeLocation           `json:"location"`

    // Variable state
    LocalVars     map[string]Variable    `json:"locals"`
    GlobalVars    map[string]Variable    `json:"globals,omitempty"`

    // External interactions (for mocking)
    DBReads       []DBOperation          `json:"db_reads,omitempty"`
    DBWrites      []DBOperation          `json:"db_writes,omitempty"`
    RFCCalls      []RFCCall              `json:"rfc_calls,omitempty"`
    HTTPCalls     []HTTPCall             `json:"http_calls,omitempty"`

    // Control flow
    CallStack     []StackFrame           `json:"call_stack"`
    BranchTaken   *BranchInfo            `json:"branch,omitempty"`
}

type Variable struct {
    Name     string      `json:"name"`
    Type     string      `json:"type"`
    Value    interface{} `json:"value"`
    Changed  bool        `json:"changed,omitempty"`  // Since last frame
}

type DBOperation struct {
    Table     string                 `json:"table"`
    Operation string                 `json:"op"`  // SELECT, INSERT, UPDATE, DELETE
    Where     map[string]interface{} `json:"where,omitempty"`
    Data      []map[string]interface{} `json:"data"`
    RowCount  int                    `json:"rows"`
}

type RFCCall struct {
    Function  string                 `json:"function"`
    Importing map[string]interface{} `json:"importing"`
    Exporting map[string]interface{} `json:"exporting"`
    Tables    map[string]interface{} `json:"tables,omitempty"`
    Duration  time.Duration          `json:"duration"`
}
```

### Recording Strategy

**Option A: Full Recording (Development/Debug)**
```go
// Record everything at every step
func (r *Recorder) OnStep(frame ExecutionFrame) {
    r.frames = append(r.frames, frame)
}
```

**Option B: Delta Recording (Production-Safe)**
```go
// Only record changes
func (r *Recorder) OnStep(frame ExecutionFrame) {
    delta := r.computeDelta(r.lastFrame, frame)
    if delta.HasChanges() {
        r.deltas = append(r.deltas, delta)
    }
    r.lastFrame = frame
}
```

**Option C: Triggered Recording (On-Demand)**
```go
// Record only around interesting events
func (r *Recorder) OnStep(frame ExecutionFrame) {
    if r.isInteresting(frame) {
        // Record N frames before and after
        r.captureWindow(frame, windowSize: 20)
    }
}

func (r *Recorder) isInteresting(frame ExecutionFrame) bool {
    return frame.HasException() ||
           frame.VariableMatchesWatch() ||
           frame.ReachedBreakpoint()
}
```

---

## Part 2: Test Case Extraction

### The Goal

Transform a recorded execution into a **standalone, reproducible test case**:

```
RECORDED EXECUTION                    EXTRACTED TEST CASE
─────────────────────────────────────────────────────────────────
Method: ZCL_PRICING->CALCULATE        TEST_CALCULATE_SCENARIO_42

Inputs:                               " Given
  IV_PRODUCT = 'WIDGET-01'              IV_PRODUCT = 'WIDGET-01'
  IV_QUANTITY = 100                     IV_QUANTITY = 100
  IV_CUSTOMER = 'CUST-123'              IV_CUSTOMER = 'CUST-123'

DB Read: MARA WHERE MATNR = ...       " Mock: Inject recorded data
  → Returns { MATNR, MTART, ... }       setup_mock_mara( recorded_data )

DB Read: KNA1 WHERE KUNNR = ...       " Mock: Inject recorded data
  → Returns { KUNNR, LAND1, ... }       setup_mock_kna1( recorded_data )

RFC: BAPI_PRICING_GET                 " Mock: Return recorded result
  → Returns { PRICE = 42.50 }           setup_mock_rfc( recorded_result )

Output:                               " Then
  RV_TOTAL = 4250.00                    cl_abap_unit_assert=>assert_equals(
                                          exp = 4250.00
                                          act = result-total )
```

### Extraction Algorithm

```go
type TestCaseExtractor struct {
    recording    *ExecutionRecording
    entryPoint   CodeLocation
    exitPoint    CodeLocation
}

func (e *TestCaseExtractor) Extract() *TestCase {
    tc := &TestCase{
        Name:        e.generateName(),
        Description: e.generateDescription(),
    }

    // 1. Find method entry frame
    entryFrame := e.findFrame(e.entryPoint)

    // 2. Extract input parameters
    tc.Inputs = e.extractInputs(entryFrame)

    // 3. Find all external dependencies
    tc.DBMocks = e.extractDBDependencies(entryFrame, e.exitPoint)
    tc.RFCMocks = e.extractRFCDependencies(entryFrame, e.exitPoint)
    tc.HTTPMocks = e.extractHTTPDependencies(entryFrame, e.exitPoint)

    // 4. Find method exit frame
    exitFrame := e.findFrame(e.exitPoint)

    // 5. Extract outputs (return value, changed parameters)
    tc.ExpectedOutputs = e.extractOutputs(exitFrame)

    // 6. Extract any exceptions
    tc.ExpectedException = e.extractException(entryFrame, exitFrame)

    return tc
}

func (e *TestCaseExtractor) extractDBDependencies(start, end CodeLocation) []DBMock {
    var mocks []DBMock

    for _, frame := range e.framesBetween(start, end) {
        for _, dbRead := range frame.DBReads {
            mocks = append(mocks, DBMock{
                Table:     dbRead.Table,
                Where:     dbRead.Where,
                Returns:   dbRead.Data,
            })
        }
    }

    // Deduplicate (same query might be called multiple times)
    return deduplicateMocks(mocks)
}
```

### Generated ABAP Test Class

```abap
*"* Generated test case from execution recording
*"* Source: Recording ID abc123, 2025-12-21 14:32:15
*"* Original execution: ZCL_PRICING->CALCULATE
CLASS lcl_test_calculate_scenario_42 DEFINITION FOR TESTING
  DURATION SHORT
  RISK LEVEL HARMLESS.

  PRIVATE SECTION.
    DATA: mo_cut TYPE REF TO zcl_pricing.  " Class Under Test
    DATA: mo_db_mock TYPE REF TO zcl_test_db_mock.
    DATA: mo_rfc_mock TYPE REF TO zcl_test_rfc_mock.

    METHODS: setup.
    METHODS: teardown.
    METHODS: test_calculate FOR TESTING.
ENDCLASS.

CLASS lcl_test_calculate_scenario_42 IMPLEMENTATION.

  METHOD setup.
    " Create mocks
    mo_db_mock = NEW zcl_test_db_mock( ).
    mo_rfc_mock = NEW zcl_test_rfc_mock( ).

    " Setup DB mock: MARA read
    mo_db_mock->expect_select(
      table = 'MARA'
      where = VALUE #( ( field = 'MATNR' value = 'WIDGET-01' ) )
      returns = VALUE #( ( matnr = 'WIDGET-01' mtart = 'FERT' matkl = 'ELECTRONICS' ) )
    ).

    " Setup DB mock: KNA1 read
    mo_db_mock->expect_select(
      table = 'KNA1'
      where = VALUE #( ( field = 'KUNNR' value = 'CUST-123' ) )
      returns = VALUE #( ( kunnr = 'CUST-123' land1 = 'US' kdgrp = 'RETAIL' ) )
    ).

    " Setup RFC mock: BAPI_PRICING_GET
    mo_rfc_mock->expect_call(
      function = 'BAPI_PRICING_GET'
      importing = VALUE #( ( name = 'IV_MATNR' value = 'WIDGET-01' )
                           ( name = 'IV_KUNNR' value = 'CUST-123' ) )
      exporting = VALUE #( ( name = 'EV_PRICE' value = '42.50' )
                           ( name = 'EV_CURRENCY' value = 'USD' ) )
    ).

    " Create class under test with injected mocks
    mo_cut = NEW zcl_pricing(
      io_db_accessor = mo_db_mock
      io_rfc_caller = mo_rfc_mock
    ).
  ENDMETHOD.

  METHOD test_calculate.
    " Given: Inputs from recorded execution
    DATA(lv_product) = 'WIDGET-01'.
    DATA(lv_quantity) = 100.
    DATA(lv_customer) = 'CUST-123'.

    " When: Execute the method
    DATA(lv_result) = mo_cut->calculate(
      iv_product  = lv_product
      iv_quantity = lv_quantity
      iv_customer = lv_customer
    ).

    " Then: Assert expected outputs
    cl_abap_unit_assert=>assert_equals(
      exp = '4250.00'
      act = lv_result
      msg = 'Total should match recorded value'
    ).

    " Verify all mocks were called
    mo_db_mock->verify_all_expectations( ).
    mo_rfc_mock->verify_all_expectations( ).
  ENDMETHOD.

  METHOD teardown.
    CLEAR: mo_cut, mo_db_mock, mo_rfc_mock.
  ENDMETHOD.

ENDCLASS.
```

---

## Part 3: Isolated Playground

### The Concept

Create a **fast, isolated environment** where:
1. No real DB access (mocked with recorded data)
2. No real RFC calls (mocked with recorded responses)
3. No real HTTP calls (mocked)
4. Execution is **deterministic** and **fast**

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        PLAYGROUND                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │ Code Unit    │    │ Mock Layer   │    │ Assertion    │       │
│  │ (Original or │◄──►│              │◄──►│ Engine       │       │
│  │  Patched)    │    │ • DB Mock    │    │              │       │
│  │              │    │ • RFC Mock   │    │ • Expected   │       │
│  │              │    │ • HTTP Mock  │    │   outputs    │       │
│  │              │    │ • Time Mock  │    │ • Invariants │       │
│  └──────────────┘    └──────────────┘    └──────────────┘       │
│         │                   │                   │                │
│         └───────────────────┴───────────────────┘                │
│                             │                                    │
│                    ┌────────▼────────┐                          │
│                    │   Test Runner   │                          │
│                    │                 │                          │
│                    │ • Fast execution│                          │
│                    │ • No side effects│                         │
│                    │ • Deterministic │                          │
│                    └─────────────────┘                          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Mock Framework Design

```abap
*----------------------------------------------------------------------*
* ZCL_PLAYGROUND_MOCK - Universal Mock Framework
*----------------------------------------------------------------------*
CLASS zcl_playground_mock DEFINITION.
  PUBLIC SECTION.
    " DB Mocking
    METHODS mock_select
      IMPORTING
        iv_table   TYPE tabname
        it_where   TYPE zcl_mock_conditions
        it_returns TYPE ANY TABLE.

    METHODS mock_select_single
      IMPORTING
        iv_table   TYPE tabname
        it_where   TYPE zcl_mock_conditions
        is_returns TYPE any.

    " RFC Mocking
    METHODS mock_rfc
      IMPORTING
        iv_function  TYPE rs38l_fnam
        it_importing TYPE zcl_mock_params OPTIONAL
        it_exporting TYPE zcl_mock_params OPTIONAL
        it_tables    TYPE zcl_mock_tables OPTIONAL.

    " HTTP Mocking
    METHODS mock_http
      IMPORTING
        iv_url      TYPE string
        iv_method   TYPE string DEFAULT 'GET'
        iv_response TYPE string
        iv_status   TYPE i DEFAULT 200.

    " Verification
    METHODS verify
      RETURNING VALUE(rv_success) TYPE abap_bool.

    METHODS get_unexpected_calls
      RETURNING VALUE(rt_calls) TYPE zcl_mock_calls.

  PRIVATE SECTION.
    DATA: mt_db_mocks   TYPE HASHED TABLE OF zcl_db_mock WITH UNIQUE KEY table_name conditions,
          mt_rfc_mocks  TYPE HASHED TABLE OF zcl_rfc_mock WITH UNIQUE KEY function_name,
          mt_http_mocks TYPE HASHED TABLE OF zcl_http_mock WITH UNIQUE KEY url method,
          mt_actual_calls TYPE STANDARD TABLE OF zcl_mock_call.
ENDCLASS.
```

### Playground Execution Flow

```go
// Go-side playground controller
type Playground struct {
    recording    *ExecutionRecording
    testCase     *TestCase
    codeUnit     *CodeUnit      // Original or patched
    mockServer   *MockServer
}

func (p *Playground) Run() *PlaygroundResult {
    // 1. Setup mocks from extracted test case
    p.mockServer.LoadMocks(p.testCase.DBMocks)
    p.mockServer.LoadMocks(p.testCase.RFCMocks)
    p.mockServer.LoadMocks(p.testCase.HTTPMocks)

    // 2. Inject mock endpoints into SAP session
    p.injectMockEndpoints()

    // 3. Execute code unit with inputs
    result, err := p.execute(p.testCase.Inputs)

    // 4. Compare outputs
    comparison := p.compareOutputs(result, p.testCase.ExpectedOutputs)

    // 5. Verify mock expectations
    mockVerification := p.mockServer.Verify()

    return &PlaygroundResult{
        Success:          comparison.Matches && mockVerification.AllExpected,
        ActualOutputs:    result,
        ExpectedOutputs:  p.testCase.ExpectedOutputs,
        Differences:      comparison.Differences,
        UnexpectedCalls:  mockVerification.UnexpectedCalls,
        MissingCalls:     mockVerification.MissingCalls,
        ExecutionTime:    result.Duration,
    }
}
```

---

## Part 4: Rapid Patch Testing Workflow

### The Developer Experience

```
┌─────────────────────────────────────────────────────────────────┐
│  PATCH TESTING WORKFLOW                                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. CAPTURE                                                      │
│     ┌──────────────────────────────────────┐                    │
│     │ $ vsp record ZCL_PRICING->CALCULATE  │                    │
│     │ Recording... (hit Ctrl+C to stop)    │                    │
│     │ ✓ Captured 47 frames                 │                    │
│     │ ✓ Found 3 DB reads, 1 RFC call       │                    │
│     │ ✓ Saved to recording_20251221.json   │                    │
│     └──────────────────────────────────────┘                    │
│                                                                  │
│  2. EXTRACT                                                      │
│     ┌──────────────────────────────────────┐                    │
│     │ $ vsp extract-test recording.json    │                    │
│     │ Analyzing recording...               │                    │
│     │ ✓ Extracted inputs: 3 parameters     │                    │
│     │ ✓ Extracted mocks: 3 DB, 1 RFC       │                    │
│     │ ✓ Extracted outputs: 1 return value  │                    │
│     │ ✓ Generated: test_calculate_42.abap  │                    │
│     └──────────────────────────────────────┘                    │
│                                                                  │
│  3. ITERATE (Fast Loop!)                                         │
│     ┌──────────────────────────────────────┐                    │
│     │ $ vsp playground test_calculate_42   │                    │
│     │                                      │                    │
│     │ Playground ready. Type 'help'.       │                    │
│     │                                      │                    │
│     │ > run                                │                    │
│     │ ✗ FAILED: Expected 4250, got 4200   │  ◄── Original bug  │
│     │                                      │                    │
│     │ > patch line:47 "LV_TOTAL = ..."    │  ◄── Apply fix     │
│     │ ✓ Patch applied                      │                    │
│     │                                      │                    │
│     │ > run                                │  ◄── Instant test! │
│     │ ✓ PASSED: All assertions match       │                    │
│     │ Execution time: 0.3s                 │                    │
│     │                                      │                    │
│     │ > commit-patch                       │  ◄── Save fix      │
│     │ ✓ Patch saved to fix_calculate.diff  │                    │
│     └──────────────────────────────────────┘                    │
│                                                                  │
│  4. VERIFY (E2E - Only When Confident)                          │
│     ┌──────────────────────────────────────┐                    │
│     │ $ vsp apply-patch fix_calculate.diff │                    │
│     │ $ vsp run-unit-tests ZCL_PRICING     │                    │
│     │ Running 15 tests...                  │                    │
│     │ ✓ All tests passed                   │                    │
│     └──────────────────────────────────────┘                    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Time Comparison

| Step | Traditional | With Playground |
|------|-------------|-----------------|
| Setup test data | 10-30 min | 0 (extracted) |
| Run single test | 30-60 sec | 0.3-1 sec |
| Patch iteration (10 tries) | 5-10 min | 3-10 sec |
| Total for fix | 20-60 min | 2-5 min |

**10-20x faster patch iteration!**

---

## Part 5: Implementation Plan

### Phase 1: Variable History Recording

**New MCP Tool: `RecordExecution`**

```go
type RecordExecutionParams struct {
    ObjectURI       string   `json:"object_uri"`
    Method          string   `json:"method,omitempty"`
    Breakpoints     []string `json:"breakpoints,omitempty"`
    CaptureDB       bool     `json:"capture_db"`
    CaptureRFC      bool     `json:"capture_rfc"`
    MaxFrames       int      `json:"max_frames"`
}

type RecordExecutionResult struct {
    RecordingID     string           `json:"recording_id"`
    FrameCount      int              `json:"frame_count"`
    DBOperations    int              `json:"db_operations"`
    RFCCalls        int              `json:"rfc_calls"`
    Duration        time.Duration    `json:"duration"`
    StoragePath     string           `json:"storage_path"`
}

func (c *Client) RecordExecution(ctx context.Context, params RecordExecutionParams) (*RecordExecutionResult, error) {
    recorder := NewExecutionRecorder(params)

    // Set up debugger with recording hooks
    session, err := c.StartDebugSession(ctx, params.ObjectURI)
    if err != nil {
        return nil, err
    }
    defer session.End()

    // Record until completion or max frames
    for recorder.FrameCount() < params.MaxFrames {
        frame, err := session.Step()
        if err != nil || frame.IsEnd() {
            break
        }

        recorder.RecordFrame(frame)

        // Capture DB operations if enabled
        if params.CaptureDB {
            recorder.CaptureDBOperations(frame)
        }

        // Capture RFC calls if enabled
        if params.CaptureRFC {
            recorder.CaptureRFCCalls(frame)
        }
    }

    // Save recording
    path, err := recorder.Save()
    if err != nil {
        return nil, err
    }

    return &RecordExecutionResult{
        RecordingID:  recorder.ID(),
        FrameCount:   recorder.FrameCount(),
        DBOperations: recorder.DBOperationCount(),
        RFCCalls:     recorder.RFCCallCount(),
        Duration:     recorder.Duration(),
        StoragePath:  path,
    }, nil
}
```

### Phase 2: Test Case Extraction

**New MCP Tool: `ExtractTestCase`**

```go
type ExtractTestCaseParams struct {
    RecordingID  string `json:"recording_id"`
    EntryPoint   string `json:"entry_point"`   // e.g., "ZCL_PRICING->CALCULATE"
    ExitPoint    string `json:"exit_point,omitempty"`
    IncludeMocks bool   `json:"include_mocks"`
}

type ExtractedTestCase struct {
    Name           string                 `json:"name"`
    Description    string                 `json:"description"`
    Inputs         map[string]interface{} `json:"inputs"`
    ExpectedOutput map[string]interface{} `json:"expected_output"`
    DBMocks        []DBMock               `json:"db_mocks"`
    RFCMocks       []RFCMock              `json:"rfc_mocks"`
    ABAPCode       string                 `json:"abap_code"`  // Generated test class
}
```

### Phase 3: Playground Execution

**New MCP Tool: `RunInPlayground`**

```go
type PlaygroundParams struct {
    TestCaseID    string            `json:"test_case_id"`
    PatchedCode   string            `json:"patched_code,omitempty"`
    OverrideInputs map[string]interface{} `json:"override_inputs,omitempty"`
}

type PlaygroundResult struct {
    Success        bool                   `json:"success"`
    ActualOutput   map[string]interface{} `json:"actual_output"`
    ExpectedOutput map[string]interface{} `json:"expected_output"`
    Differences    []Difference           `json:"differences,omitempty"`
    ExecutionTime  time.Duration          `json:"execution_time"`
    Coverage       float64                `json:"coverage_percent"`
}
```

### Phase 4: Integration with AI

**Claude-Powered Patch Loop:**

```javascript
async function aiPatchLoop(recording, maxAttempts = 5) {
    // Extract test case from recording
    const testCase = await extractTestCase(recording);

    // Get original code
    const originalCode = await getSource(testCase.objectType, testCase.objectName);

    for (let attempt = 0; attempt < maxAttempts; attempt++) {
        // Run in playground
        const result = await runInPlayground({
            testCaseId: testCase.id,
            patchedCode: attempt === 0 ? originalCode : patchedCode
        });

        if (result.success) {
            console.log(`✓ Fix found on attempt ${attempt + 1}`);
            return { success: true, patch: patchedCode };
        }

        // AI analyzes failure and proposes patch
        const analysis = await claude.analyze({
            code: patchedCode || originalCode,
            testCase: testCase,
            failure: result.differences,
            previousAttempts: attempts
        });

        patchedCode = await claude.generatePatch({
            originalCode: originalCode,
            analysis: analysis,
            constraints: ['minimal_change', 'preserve_behavior']
        });

        console.log(`Attempt ${attempt + 1}: ${analysis.hypothesis}`);
    }

    return { success: false, attempts: maxAttempts };
}
```

---

## Part 6: Mock Generation Strategies

### Strategy A: Record-and-Replay (VCR-style)

Record all external calls, replay exact responses:

```go
type VCRMock struct {
    recordings map[string][]Recording  // key = call signature
}

func (v *VCRMock) Intercept(call ExternalCall) Response {
    key := call.Signature()
    recordings := v.recordings[key]

    if len(recordings) == 0 {
        return ErrorResponse("No recording for: " + key)
    }

    // Return next recorded response (FIFO)
    recording := recordings[0]
    v.recordings[key] = recordings[1:]

    return recording.Response
}
```

### Strategy B: Smart Mocking (Pattern-based)

Generalize from recordings to handle variations:

```go
type SmartMock struct {
    patterns []MockPattern
}

type MockPattern struct {
    Matcher   func(call ExternalCall) bool
    Generator func(call ExternalCall) Response
}

// Example: "Any SELECT from MARA returns the recorded row"
pattern := MockPattern{
    Matcher: func(call ExternalCall) bool {
        return call.Type == "DB" &&
               call.Table == "MARA" &&
               call.Operation == "SELECT"
    },
    Generator: func(call ExternalCall) Response {
        // Return recorded data, filtered by WHERE clause
        return filterRecordedData(call.Where)
    },
}
```

### Strategy C: AI-Generated Mocks

Let Claude generate realistic mock data:

```javascript
async function generateMock(call, context) {
    const prompt = `
        Generate realistic mock data for this SAP database call:
        Table: ${call.table}
        Operation: ${call.operation}
        Where clause: ${JSON.stringify(call.where)}

        Context from recording:
        - Program: ${context.program}
        - Method: ${context.method}
        - Related data: ${JSON.stringify(context.relatedMocks)}

        Generate data that would be typical for this scenario.
    `;

    return await claude.generate(prompt, { format: 'json' });
}
```

---

## Part 7: Advanced Features

### Feature: Mutation Testing in Playground

Test the quality of tests by mutating the code:

```go
func MutationTest(testCase *TestCase, code string) *MutationReport {
    mutations := generateMutations(code)

    var killed, survived int

    for _, mutation := range mutations {
        result := runInPlayground(testCase, mutation.Code)

        if result.Success {
            // Mutation survived = test is weak
            survived++
        } else {
            // Mutation killed = test caught it
            killed++
        }
    }

    return &MutationReport{
        TotalMutations: len(mutations),
        Killed:         killed,
        Survived:       survived,
        Score:          float64(killed) / float64(len(mutations)),
    }
}
```

### Feature: Property-Based Testing

Generate many inputs, verify properties hold:

```javascript
async function propertyTest(testCase, properties, iterations = 100) {
    for (let i = 0; i < iterations; i++) {
        // Generate random inputs based on types
        const inputs = generateInputs(testCase.inputTypes);

        // Run in playground
        const result = await runInPlayground({
            testCaseId: testCase.id,
            overrideInputs: inputs
        });

        // Check properties hold
        for (const property of properties) {
            if (!property.check(inputs, result)) {
                return {
                    success: false,
                    failingInput: inputs,
                    property: property.name
                };
            }
        }
    }

    return { success: true, iterations };
}

// Example properties:
const properties = [
    {
        name: "Total is non-negative",
        check: (inputs, result) => result.total >= 0
    },
    {
        name: "Total proportional to quantity",
        check: (inputs, result) => {
            const ratio = result.total / inputs.quantity;
            return ratio > 0 && ratio < 1000000;  // Sanity check
        }
    }
];
```

### Feature: Differential Testing

Compare patched vs original behavior across many inputs:

```go
func DifferentialTest(original, patched string, testCase *TestCase, variants int) *DiffReport {
    var differences []Difference

    for i := 0; i < variants; i++ {
        inputs := generateVariant(testCase.Inputs)

        origResult := runInPlayground(testCase, original, inputs)
        patchResult := runInPlayground(testCase, patched, inputs)

        if !reflect.DeepEqual(origResult.Output, patchResult.Output) {
            differences = append(differences, Difference{
                Inputs:         inputs,
                OriginalOutput: origResult.Output,
                PatchedOutput:  patchResult.Output,
            })
        }
    }

    return &DiffReport{
        Variants:    variants,
        Differences: differences,
        Compatible:  len(differences) == 0,
    }
}
```

---

## Part 8: Storage and Retrieval

### Recording Storage Format

```
recordings/
├── 2025-12-21/
│   ├── rec_abc123.json       # Full recording
│   ├── rec_abc123_meta.json  # Metadata only
│   └── rec_abc123_mocks.json # Extracted mocks
├── test_cases/
│   ├── tc_xyz789.json        # Extracted test case
│   └── tc_xyz789.abap        # Generated ABAP code
└── index.json                # Searchable index
```

### Recording Index

```json
{
  "recordings": [
    {
      "id": "rec_abc123",
      "timestamp": "2025-12-21T14:32:15Z",
      "object": "ZCL_PRICING",
      "method": "CALCULATE",
      "frames": 47,
      "db_ops": 3,
      "rfc_calls": 1,
      "tags": ["pricing", "production-bug", "TICKET-1234"],
      "size_bytes": 125000
    }
  ]
}
```

### Search and Retrieval

```go
// Find recordings for a specific object
recordings := store.Search(Query{
    Object: "ZCL_PRICING",
    Method: "CALCULATE",
    DateRange: DateRange{
        From: time.Now().AddDate(0, 0, -7),  // Last week
        To:   time.Now(),
    },
})

// Load specific recording
recording, err := store.Load("rec_abc123")
```

---

## Conclusion

### The Vision

```
TRADITIONAL DEBUGGING          →    PLAYGROUND DEBUGGING
─────────────────────────────────────────────────────────────
Slow E2E tests (minutes)       →    Fast isolated tests (ms)
Manual test data setup         →    Automatic from recording
Guess → Test → Wait → Repeat   →    Guess → Test → Instant
Hard to reproduce              →    Perfectly reproducible
No mocking                     →    Full mock framework
Single attempt                 →    Rapid iteration
```

### Implementation Priority

| Phase | Feature | Effort | Impact |
|-------|---------|--------|--------|
| 1 | Variable history recording | Medium | High |
| 2 | Test case extraction | Medium | Very High |
| 3 | Basic playground (no mocks) | Low | Medium |
| 4 | DB mocking | Medium | Very High |
| 5 | RFC mocking | Medium | High |
| 6 | AI patch loop | Low | Very High |
| 7 | Mutation testing | Low | Medium |
| 8 | Property testing | Medium | Medium |

### Next Steps

1. **Prototype variable recording** via WebSocket debug session
2. **Design mock injection** mechanism for SAP
3. **Build test case extractor** from recording format
4. **Create playground runner** with mock support
5. **Integrate with Claude** for AI patch loop

---

*"Record once, test forever, fix in seconds."*
