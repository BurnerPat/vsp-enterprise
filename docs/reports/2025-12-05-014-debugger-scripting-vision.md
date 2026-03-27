# External Debugger Scripting Vision

**Date:** 2025-12-05
**Report ID:** 014
**Subject:** Watchpoints API + AI-Powered Debugger Scripting Architecture
**Status:** Research & Brainstorming

---

## Executive Summary

SAP ADT provides a comprehensive REST API for debugging that includes **watchpoints** (data breakpoints). Combined with external breakpoints, variable inspection, and stepping controls, this enables a powerful vision: **AI-powered debugger scripting** where an AI assistant can autonomously debug ABAP programs, inspect state, and diagnose issues.

---

## Part 1: Watchpoint API Discovery

### Endpoint: `/sap/bc/adt/debugger/watchpoints`

| Method | Path | Parameters | Description |
|--------|------|------------|-------------|
| POST | `/watchpoints` | `variableName`, `condition` | Create watchpoint |
| GET | `/watchpoints` | - | List all watchpoints |
| GET | `/watchpoints/{id}` | - | Get specific watchpoint |
| PUT | `/watchpoints/{id}` | `condition`, `active` | Modify watchpoint |
| DELETE | `/watchpoints/{id}` | - | Delete watchpoint |

### Watchpoint Kinds

```abap
ce_tpdapi_wp_kind=>system_global      " System-wide global variables
ce_tpdapi_wp_kind=>program_global     " Program-global variables
ce_tpdapi_wp_kind=>local              " Local variables
ce_tpdapi_wp_kind=>instance_attribute " Object instance attributes
```

### Watchpoint Response Structure

```json
{
  "id": 1,
  "varname": "LV_COUNTER",
  "active": true,
  "procedure": "CALCULATE_TOTAL",
  "program": "ZTEST_PROGRAM",
  "condition": "LV_COUNTER > 100",
  "kind": "local",
  "currentvalue": "150",
  "oldvalue": "99"
}
```

### Key Limitation

**Watchpoints require an attached debug session.** Unlike external breakpoints which persist, watchpoints only exist during active debugging. This means:
- Must first attach to a debuggee process
- Watchpoints are created within that session
- They're lost when session ends

---

## Part 2: Complete Debug API Surface

### Session Lifecycle

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Set External   │     │    Listen for   │     │    Attach to    │
│   Breakpoints   │────▶│    Debuggee     │────▶│    Process      │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                              │                        │
                              │ (long poll)            │
                              ▼                        ▼
                        ┌─────────────────┐     ┌─────────────────┐
                        │  Debuggee Hits  │     │  Debug Session  │
                        │   Breakpoint    │────▶│    Active       │
                        └─────────────────┘     └─────────────────┘
```

### Available Operations (from abap-adt-api + SAP classes)

| Category | Operation | Endpoint |
|----------|-----------|----------|
| **Breakpoints** | Set external | POST `/debugger/breakpoints` |
| | Get external | GET `/debugger/breakpoints` |
| | Delete | DELETE `/debugger/breakpoints/{id}` |
| | Validate condition | POST `/debugger/breakpoints/conditions` |
| **Watchpoints** | Create | POST `/debugger/watchpoints?variableName=...` |
| | List all | GET `/debugger/watchpoints` |
| | Modify | PUT `/debugger/watchpoints/{id}` |
| | Delete | DELETE `/debugger/watchpoints/{id}` |
| **Listeners** | Check active | GET `/debugger/listeners` |
| | Start listening | POST `/debugger/listeners` (100h timeout) |
| | Stop listening | DELETE `/debugger/listeners` |
| **Session** | Attach | POST `/debugger?method=attach` |
| | Get settings | GET `/debugger?method=getSettings` |
| | Save settings | POST `/debugger?method=setDebuggerSettings` |
| **Execution** | Step into | POST `/debugger?method=stepInto` |
| | Step over | POST `/debugger?method=stepOver` |
| | Step return | POST `/debugger?method=stepReturn` |
| | Continue | POST `/debugger?method=stepContinue` |
| | Run to line | POST `/debugger?method=stepRunToLine&uri=...` |
| | Jump to line | POST `/debugger?method=stepJumpToLine&uri=...` |
| | Terminate | POST `/debugger?method=terminateDebuggee` |
| **Inspection** | Get stack | GET `/debugger/stack` |
| | Get variables | POST `/debugger?method=getVariables` |
| | Get children | POST `/debugger?method=getChildVariables` |
| | Set variable | POST `/debugger?method=setVariableValue` |

---

## Part 3: AI-Powered Debugger Scripting Vision

### Concept: Autonomous Debugging Agent

Imagine an AI assistant that can:
1. Set strategic breakpoints based on code analysis
2. Wait for program execution to hit breakpoints
3. Automatically inspect variables and state
4. Make decisions about stepping or continuing
5. Collect diagnostic data across multiple runs
6. Generate root cause analysis reports

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    AI Debugging Agent                            │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │   Planner   │  │  Inspector  │  │  Analyzer   │              │
│  │             │  │             │  │             │              │
│  │ - Analyze   │  │ - Variables │  │ - Patterns  │              │
│  │   code      │  │ - Stack     │  │ - Anomalies │              │
│  │ - Place BPs │  │ - Memory    │  │ - Root cause│              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│         │                │                │                      │
│         ▼                ▼                ▼                      │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                   Debug Script Engine                        ││
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐        ││
│  │  │Breakpnt │  │ Watch   │  │  Step   │  │ Inspect │        ││
│  │  │ Manager │  │ Manager │  │ Control │  │ Engine  │        ││
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘        ││
│  └─────────────────────────────────────────────────────────────┘│
│                              │                                   │
└──────────────────────────────┼───────────────────────────────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │   SAP ADT REST API  │
                    │  /debugger/*        │
                    └─────────────────────┘
```

### Debug Script DSL

```yaml
# debug-script.yaml - Automated debugging scenario
name: Investigate Zero Division
description: Find why lv_total becomes zero

triggers:
  - exception: CX_SY_ZERODIVIDE
  - breakpoint:
      program: ZCL_PRICING
      method: CALCULATE_RATIO
      line: 42

on_break:
  # Capture current state
  - capture:
      variables: [lv_total, lv_count, lt_items]
      stack: true

  # Conditional logic
  - if:
      condition: "lv_total == 0"
      then:
        - log: "Found zero total!"
        - capture:
            variables: [lt_items->*]  # Expand table
        - step_back: 5  # Review last 5 statements
      else:
        - continue

  # Set watchpoint for next time
  - watchpoint:
      variable: lv_total
      condition: "lv_total = 0"

output:
  format: markdown
  include:
    - variable_history
    - call_stack
    - root_cause_hypothesis
```

### Fluent Go API

```go
// Fluent debugging API
script := dsl.Debug(client).
    OnException("CX_SY_ZERODIVIDE").
    OnBreakpoint("/sap/bc/adt/oo/classes/zcl_pricing/source/main", 42).
    WhenHit(func(ctx *DebugContext) Action {
        total := ctx.Variable("lv_total")
        if total.AsInt() == 0 {
            ctx.Log("Zero total detected!")
            ctx.CaptureAll()
            return StepInto
        }
        return Continue
    }).
    CollectHistory(100).  // Keep last 100 states
    Build()

// Execute with timeout
result, err := script.Run(ctx, 5*time.Minute)

// Generate report
report := result.GenerateRCA()
```

### Use Cases

#### 1. Automated Regression Debugging

```yaml
name: Regression Hunter
description: Find what changed between working and broken versions

setup:
  - set_breakpoints:
      pattern: "ZCL_ORDER_*=>CALCULATE_*"

  - watchpoints:
      - gs_order-total
      - gs_order-discount

run:
  iterations: 2
  scenarios:
    - name: working
      input: { order_id: "OLD_001" }
    - name: broken
      input: { order_id: "NEW_001" }

compare:
  - variable_values_at_breakpoints
  - execution_paths
  - timing_differences
```

#### 2. Performance Profiling via Debugging

```yaml
name: Hot Path Finder
description: Identify slow code paths

setup:
  - breakpoints:
      entry: ZCL_REPORT=>GENERATE
      exit: ZCL_REPORT=>GENERATE

on_entry:
  - start_timer: generate_time
  - capture: [iv_params]

on_exit:
  - stop_timer: generate_time
  - capture: [rv_result]
  - if:
      condition: "generate_time > 1000ms"
      then:
        - flag: slow_execution
        - capture_full_state
```

#### 3. Data Flow Analysis

```yaml
name: Track Data Origin
description: Find where corrupted data originated

target_variable: gs_document-amount
trace_mode: backward  # Track assignments backward

on_assignment:
  - capture:
      source_expression: true
      stack: true

  - if:
      condition: "new_value < 0"
      then:
        - flag: corruption_point
        - stop

output:
  - assignment_chain
  - source_locations
```

#### 4. Interactive AI Debugging Session

```
User: "Debug why order 12345 has wrong total"

AI Agent:
1. Searches code for order processing logic
2. Sets breakpoints at key calculation points
3. Triggers order 12345 processing
4. Waits for breakpoint hit
5. Inspects variables: finds discount applied twice
6. Reports: "Double discount bug in ZCL_ORDER=>APPLY_PROMOTIONS line 87"
```

---

## Part 4: Implementation Roadmap

### Phase 1: Foundation (Current)
- [x] External breakpoints (Set, Delete, Get)
- [x] Breakpoint kinds (line, exception, statement, message)
- [x] Validate breakpoint conditions
- [x] Debug listener (long-poll for debuggee) ✅ **IMPLEMENTED**
  - `DebuggerListen()` - long-poll POST with configurable timeout
  - `DebuggerCheckListener()` - check for active listeners
  - `DebuggerStopListener()` - stop listener
  - `Debuggee` struct for debuggee response parsing
- [ ] Session attach/detach

### Phase 2: Core Debug Operations
- [ ] Watchpoints (Create, List, Modify, Delete)
- [ ] Step operations (into, over, return, continue)
- [ ] Variable inspection (read)
- [ ] Variable modification (write)
- [ ] Stack inspection

### Phase 3: Debug Scripting Engine
- [ ] Script DSL definition
- [ ] YAML parser for debug scripts
- [ ] Fluent Go API
- [ ] Condition evaluator
- [ ] State capture/history

### Phase 4: AI Integration
- [ ] Breakpoint placement heuristics
- [ ] Anomaly detection
- [ ] Root cause hypothesis generation
- [ ] Natural language debug interface

---

## Part 5: WebSocket vs HTTP Analysis

**Question:** Does SAP ADT use WebSocket for debugging?

**Answer:** **No.** The debugger uses **long-polling HTTP**, not WebSocket.

### SAP APC (ABAP Push Channel) Infrastructure

SAP has WebSocket support via APC:
- `CL_SSI_WEBSOCKET` - Basic WebSocket handling
- `CL_WEBSOCKET_UTILITY` - Utility class
- `CL_APC_WS_PROGRESS_INDICATOR` - Progress updates

### ADT WebSocket Usage

ADT uses APC **only for progress indicators** (`CL_ADT_REST_APC_UTIL`):
```abap
" Headers used for progress channel
HEADER_NAME_CONNECTION_ID = 'sap-adt-connection-id'
HEADER_NAME_REQUEST_ID = 'sap-adt-request-id'
```

### Debugger Protocol Summary

| Feature | Protocol | Notes |
|---------|----------|-------|
| **Debug Listener** | Long-polling HTTP | 240s default timeout |
| **Progress Indicators** | WebSocket (APC) | Real-time progress |
| **All Debug Operations** | REST HTTP | Step, inspect, breakpoints |

**No TPDA (debugger) classes use APC/WebSocket.**

---

## Part 6: Technical Challenges

### 1. Long-Polling Listener

The debug listener uses configurable timeout (default 240 seconds):
```go
// vsp implementation
opts.TimeoutSeconds = 240 // SAP default
httpTimeout := time.Duration(opts.TimeoutSeconds+30) * time.Second
```

This requires:
- Persistent HTTP connection handling
- Graceful timeout/reconnect
- Background goroutine management

### 2. Session State Management

Debug sessions are stateful:
- Terminal ID identifies the IDE instance
- Session must stay attached
- Operations must be sequenced correctly

### 3. Concurrency

Multiple debug scripts might run simultaneously:
- Need session isolation
- Breakpoint namespacing
- Resource cleanup on failure

### 4. Security Considerations

- Debug access = full system access
- Variable modification can corrupt data
- Should require explicit authorization
- Audit logging essential

---

## Part 7: Comparison with Existing Tools

| Feature | SAP GUI Debugger | ADT (Eclipse) | vsp (Vision) |
|---------|------------------|---------------|--------------|
| Interactive debugging | Yes | Yes | No (scripted) |
| External breakpoints | Yes | Yes | Yes |
| Watchpoints | Yes | Yes | Planned |
| Scripted debugging | No | Limited | Full |
| AI-assisted | No | No | Yes |
| CI/CD integration | No | No | Yes |
| Cross-session analysis | No | No | Yes |
| Automated RCA | No | No | Yes |

---

## Part 8: Sample Implementation Sketch

### Watchpoint Types for vsp

```go
// pkg/adt/watchpoint.go

type WatchpointKind string

const (
    WatchpointKindSystemGlobal    WatchpointKind = "system_global"
    WatchpointKindProgramGlobal   WatchpointKind = "program_global"
    WatchpointKindLocal           WatchpointKind = "local"
    WatchpointKindInstanceAttr    WatchpointKind = "instance_attribute"
)

type Watchpoint struct {
    ID           int            `json:"id"`
    VariableName string         `json:"varname"`
    Active       bool           `json:"active"`
    Condition    string         `json:"condition,omitempty"`
    Kind         WatchpointKind `json:"kind"`
    Program      string         `json:"program"`
    Procedure    string         `json:"procedure"`
    CurrentValue string         `json:"currentvalue"`
    OldValue     string         `json:"oldvalue"`
}

// CreateWatchpoint creates a watchpoint in the current debug session
func (c *Client) CreateWatchpoint(ctx context.Context, variableName, condition string) (*Watchpoint, error) {
    query := url.Values{}
    query.Set("variableName", variableName)
    if condition != "" {
        query.Set("condition", condition)
    }

    resp, err := c.transport.Request(ctx, "/sap/bc/adt/debugger/watchpoints", &RequestOptions{
        Method: http.MethodPost,
        Accept: "application/xml",
        Query:  query,
    })
    // ... parse response
}
```

### Debug Script Runner

```go
// pkg/dsl/debug_script.go

type DebugScript struct {
    Name        string
    Triggers    []Trigger
    OnBreak     []Action
    MaxDuration time.Duration
}

type DebugRunner struct {
    client    *adt.Client
    script    *DebugScript
    history   []DebugState
    listeners []func(DebugEvent)
}

func (r *DebugRunner) Run(ctx context.Context) (*DebugResult, error) {
    // 1. Set up breakpoints/watchpoints from triggers
    for _, trigger := range r.script.Triggers {
        if err := r.setupTrigger(ctx, trigger); err != nil {
            return nil, err
        }
    }

    // 2. Start listener (long-poll)
    debuggee, err := r.waitForDebuggee(ctx)
    if err != nil {
        return nil, err
    }

    // 3. Attach to debuggee
    session, err := r.client.DebuggerAttach(ctx, debuggee.ID)
    if err != nil {
        return nil, err
    }
    defer r.client.DebuggerDetach(ctx)

    // 4. Execute script actions
    for {
        state := r.captureState(ctx)
        r.history = append(r.history, state)

        action := r.evaluateActions(ctx, state)
        switch action {
        case ActionContinue:
            r.client.DebuggerStep(ctx, "stepContinue")
        case ActionStepInto:
            r.client.DebuggerStep(ctx, "stepInto")
        case ActionStop:
            return r.generateResult(), nil
        }
    }
}
```

---

## Conclusion

The SAP ADT debug API is remarkably complete, providing all the primitives needed for sophisticated automated debugging:

1. **Watchpoints ARE supported** via `/debugger/watchpoints` (session-bound)
2. **External breakpoints** persist across sessions
3. **Full stepping control** enables programmatic execution
4. **Variable read/write** allows state inspection and modification

The vision of **AI-powered debugger scripting** is technically feasible. An AI assistant could:
- Analyze code to place strategic breakpoints
- Wait for execution to hit them
- Inspect state and make decisions
- Collect data across runs
- Generate root cause analysis

This would transform debugging from an interactive manual process to an **automated diagnostic capability** that integrates with CI/CD pipelines and AI-assisted development workflows.

---

## References

- `CL_TPDA_ADT_RES_WATCHPOINTS` - Watchpoint REST handler
- `CL_TPDA_ADT_RES_BREAKPOINTS` - Breakpoint REST handler
- `CL_TPDA_ADT_RES_DEBUGGER` - Main debugger REST handler
- `IF_TPDAPI_WP_SERVICES` - Watchpoint service interface
- `abap-adt-api/debugger.ts` - TypeScript reference implementation
- Simple Transformation: `TPDA_ADT_DEBUGGER_WP` - Watchpoint XML format
