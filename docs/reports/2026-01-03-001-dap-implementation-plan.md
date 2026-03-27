# DAP Implementation Plan: ABAP Debug Adapter

**Date:** 2026-01-03
**Report ID:** 001
**Subject:** Concrete implementation plan for Debug Adapter Protocol support
**Related Documents:**
- [2026-01-02-004-dap-abap-debugging-vision.md](./2026-01-02-004-dap-abap-debugging-vision.md) - Architecture vision
- [2025-12-05-014-external-debugger-scripting-vision.md](./2025-12-05-014-external-debugger-scripting-vision.md) - Debugger scripting

---

## Inspiration: DZRP Pattern

The DZRP (DeZog Remote Protocol) debugger for Z80 emulators demonstrates the ideal pattern:

```
┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│   VS Code       │  DAP    │   mzrun/dezog   │  DZRP   │   Emulator      │
│   Debug UI      │◄───────►│   (adapter)     │◄───────►│   (target)      │
└─────────────────┘  stdio  └─────────────────┘   TCP   └─────────────────┘
```

Key observations:
1. **Separate adapter binary** - `mzrun` handles DAP↔DZRP translation
2. **Interactive CLI mode** - Standalone debugger without IDE
3. **Remote target** - Connects via TCP to actual runtime
4. **Simple commands** - s=step, o=step-over, c=continue, r=regs, m=mem, q=quit

## Proposed Architecture for ABAP

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              VS Code                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    Native Debug UI                                   │    │
│  │  • Breakpoint gutter icons                                          │    │
│  │  • Call stack panel                                                 │    │
│  │  • Variables panel (locals, globals, watch)                         │    │
│  │  • Debug toolbar (continue, step over, step into, step out)         │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                    │                                         │
│                                    │ DAP (stdio JSON-RPC)                    │
└────────────────────────────────────┼────────────────────────────────────────┘
                                     │
┌────────────────────────────────────┼────────────────────────────────────────┐
│                           vsp-dap binary                                     │
│  ┌─────────────────────────────────┴─────────────────────────────────────┐  │
│  │                     DAP Protocol Handler                               │  │
│  │  • Capabilities negotiation                                           │  │
│  │  • Request/Response handling                                          │  │
│  │  • Event emission (stopped, output, breakpoint)                       │  │
│  └─────────────────────────────────┬─────────────────────────────────────┘  │
│                                    │                                         │
│  ┌─────────────────────────────────┴─────────────────────────────────────┐  │
│  │                     Debug Session Manager                              │  │
│  │  • Breakpoint registry                                                │  │
│  │  • Variable cache                                                     │  │
│  │  • Stack frame mapping                                                │  │
│  │  • Source file mapping (ADT URL ↔ local path)                         │  │
│  └─────────────────────────────────┬─────────────────────────────────────┘  │
│                                    │                                         │
│  ┌─────────────────────────────────┴─────────────────────────────────────┐  │
│  │                     ADT Client (pkg/adt)                               │  │
│  │  • HTTP/REST for ADT APIs                                             │  │
│  │  • WebSocket for ZADT_VSP                                             │  │
│  │  • CSRF token management                                              │  │
│  └─────────────────────────────────┬─────────────────────────────────────┘  │
└────────────────────────────────────┼────────────────────────────────────────┘
                                     │
                     HTTP + WebSocket │
                                     │
┌────────────────────────────────────┼────────────────────────────────────────┐
│                           SAP System                                         │
│  ┌─────────────────────────────────┴─────────────────────────────────────┐  │
│  │         ADT REST APIs                    ZADT_VSP WebSocket            │  │
│  │  /sap/bc/adt/debugger/*                 /sap/bc/apc/vsp                │  │
│  │  • External breakpoints                  • debug domain                │  │
│  │  • Debugger listener                     • Stateful session            │  │
│  │  • Attach/detach                         • RFC calls (trigger)         │  │
│  │  • Step execution                                                      │  │
│  │  • Variable inspection                                                 │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Implementation Phases

### Phase 1: Interactive CLI Debugger (`vsp debug`)

Create a standalone interactive debugger like `mzrun`:

```bash
# Attach mode - wait for any debuggee
$ vsp debug --attach
Waiting for debuggee... (Ctrl+C to cancel)
Caught debuggee: ZTEST_PROGRAM at line 42

=== VSP ABAP Debugger ===
Commands: s=step, n=next, o=out, c=continue, r=stack, v=vars, b=break, q=quit

ZTEST_PROGRAM:42  DATA(lv_result) = calculate( iv_input ).
> v
Variables:
  IV_INPUT = 'TEST' (STRING)
  LV_RESULT = <unassigned>

ZTEST_PROGRAM:42  DATA(lv_result) = calculate( iv_input ).
> s
ZTEST_PROGRAM:43  IF lv_result > 0.
> r
Call Stack:
  #0 ZTEST_PROGRAM:43  main
  #1 ZCL_TEST=>RUN:15
  #2 SAPMSSY1:1
> c
Debuggee released.
$
```

**Implementation:**
- New `cmd/vsp/debug.go` - cobra command
- Uses existing `pkg/adt/debugger.go` functions
- REPL loop with readline support
- Connects via WebSocket for stateful session

**Estimated effort:** 1 day

### Phase 2: DAP Adapter Binary (`vsp-dap` or `vsp --dap`)

Implement full DAP protocol:

```bash
# As separate binary
$ vsp-dap --sap-url http://host:50000 --sap-user admin

# Or as mode flag
$ vsp --dap
```

**DAP Message Flow:**

```
VS Code                    vsp-dap                     SAP
   │                          │                          │
   │──initialize──────────────►│                          │
   │◄─────────capabilities─────│                          │
   │                          │                          │
   │──setBreakpoints──────────►│                          │
   │                          │───SetBreakpoint──────────►│
   │                          │◄──────breakpoint_id───────│
   │◄────breakpoints verified──│                          │
   │                          │                          │
   │──attach──────────────────►│                          │
   │                          │───DebuggerListen─────────►│
   │                          │         (long poll)       │
   │                          │◄──────debuggee_id─────────│
   │                          │───DebuggerAttach─────────►│
   │◄────stopped event─────────│                          │
   │                          │                          │
   │──stackTrace──────────────►│                          │
   │                          │───DebuggerGetStack───────►│
   │                          │◄─────stack frames─────────│
   │◄────stack frames──────────│                          │
   │                          │                          │
   │──variables───────────────►│                          │
   │                          │───DebuggerGetVariables───►│
   │                          │◄──────variables───────────│
   │◄────variables─────────────│                          │
   │                          │                          │
   │──stepOver────────────────►│                          │
   │                          │───DebuggerStep(over)─────►│
   │                          │◄──────new position────────│
   │◄────stopped event─────────│                          │
   │                          │                          │
   │──disconnect──────────────►│                          │
   │                          │───DebuggerDetach─────────►│
   │◄────terminated event──────│                          │
```

**DAP Protocol Implementation:**

| DAP Request | ADT/WebSocket Mapping |
|-------------|----------------------|
| `initialize` | Return capabilities (supportsConfigurationDoneRequest, etc.) |
| `setBreakpoints` | `SetBreakpoint` via WebSocket |
| `configurationDone` | Start listening |
| `attach` | `DebuggerListen` + `DebuggerAttach` |
| `launch` | `RunUnitTests` or `CallRFC` + `DebuggerListen` |
| `continue` | `DebuggerStep(stepContinue)` |
| `next` | `DebuggerStep(stepOver)` |
| `stepIn` | `DebuggerStep(stepInto)` |
| `stepOut` | `DebuggerStep(stepReturn)` |
| `stackTrace` | `DebuggerGetStack` |
| `scopes` | Map stack frame to variable scopes |
| `variables` | `DebuggerGetVariables` |
| `disconnect` | `DebuggerDetach` |

**DAP Events to Emit:**

| Event | When |
|-------|------|
| `initialized` | After `initialize` request |
| `stopped` | Breakpoint hit, step complete |
| `continued` | After `continue` request |
| `output` | Debug console messages |
| `breakpoint` | Breakpoint verified/changed |
| `terminated` | Debug session ended |

**Estimated effort:** 3-4 days

### Phase 3: VS Code Extension

Minimal extension to configure the debug adapter:

```
vscode-abap-debug/
├── package.json          # Extension manifest
├── src/
│   └── extension.ts      # Minimal activation
└── launch-configs/
    └── abap-debug.json   # Example launch.json
```

**package.json (key parts):**

```json
{
  "name": "vscode-abap-debug",
  "displayName": "ABAP Debug",
  "contributes": {
    "debuggers": [{
      "type": "abap",
      "label": "ABAP Debug",
      "program": "./vsp-dap",
      "runtime": "node",
      "configurationAttributes": {
        "launch": {
          "properties": {
            "program": {
              "type": "string",
              "description": "Program or class to debug"
            },
            "testClass": {
              "type": "string",
              "description": "Test class for launch mode"
            }
          }
        },
        "attach": {
          "properties": {
            "user": {
              "type": "string",
              "description": "SAP user to attach to"
            },
            "timeout": {
              "type": "number",
              "default": 60,
              "description": "Timeout waiting for debuggee"
            }
          }
        }
      },
      "initialConfigurations": [{
        "type": "abap",
        "request": "attach",
        "name": "Attach to ABAP"
      }]
    }],
    "configuration": {
      "title": "ABAP Debug",
      "properties": {
        "abapDebug.sapUrl": {
          "type": "string",
          "description": "SAP system URL"
        },
        "abapDebug.sapUser": {
          "type": "string",
          "description": "SAP username"
        }
      }
    }
  }
}
```

**Example launch.json:**

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "type": "abap",
      "request": "attach",
      "name": "Attach to ABAP",
      "user": "${config:abapDebug.sapUser}",
      "timeout": 120
    },
    {
      "type": "abap",
      "request": "launch",
      "name": "Debug Unit Tests",
      "program": "ZCL_MY_CLASS",
      "testClass": "ltcl_test"
    },
    {
      "type": "abap",
      "request": "launch",
      "name": "Run Report",
      "program": "ZTEST_REPORT",
      "variant": "TEST_VAR1"
    }
  ]
}
```

**Estimated effort:** 1-2 days

## File Structure

```
vibing-steampunk/
├── cmd/
│   ├── vsp/
│   │   ├── main.go           # Existing MCP server
│   │   └── debug.go          # NEW: Interactive debugger command
│   └── vsp-dap/
│       └── main.go           # NEW: DAP adapter binary
│
├── pkg/
│   ├── adt/
│   │   ├── debugger.go       # Existing: ADT debugger client
│   │   └── websocket.go      # Existing: WebSocket client
│   └── dap/                  # NEW: DAP protocol package
│       ├── protocol.go       # DAP types and constants
│       ├── handler.go        # Request handlers
│       ├── session.go        # Debug session state
│       └── mapping.go        # ADT ↔ DAP mapping
│
├── vscode-abap-debug/        # NEW: VS Code extension
│   ├── package.json
│   ├── src/extension.ts
│   └── README.md
│
└── Makefile                  # Add build targets for vsp-dap
```

## Go Libraries

**DAP Protocol:**
- `github.com/google/go-dap` - Official Go DAP implementation (used by Delve)

**Example usage:**

```go
import "github.com/google/go-dap"

// Read DAP message from stdin
reader := bufio.NewReader(os.Stdin)
msg, err := dap.ReadProtocolMessage(reader)

// Handle request
switch request := msg.(type) {
case *dap.InitializeRequest:
    response := &dap.InitializeResponse{
        Response: dap.Response{
            RequestSeq: request.Seq,
            Success:    true,
            Command:    "initialize",
        },
        Body: dap.Capabilities{
            SupportsConfigurationDoneRequest: true,
            SupportsFunctionBreakpoints:      true,
            SupportsConditionalBreakpoints:   false, // Future
        },
    }
    dap.WriteProtocolMessage(os.Stdout, response)

case *dap.AttachRequest:
    // Start debug listener
    go func() {
        event, err := adtClient.DebuggerListen(ctx, timeout)
        if err == nil {
            adtClient.DebuggerAttach(ctx, event.DebuggeeID)
            // Send stopped event
            dap.WriteProtocolMessage(os.Stdout, &dap.StoppedEvent{
                Event: dap.Event{Event: "stopped"},
                Body: dap.StoppedEventBody{
                    Reason:   "breakpoint",
                    ThreadId: 1,
                },
            })
        }
    }()
}
```

## Timeline & Priorities

| Phase | Effort | Priority | Delivers |
|-------|--------|----------|----------|
| 1. CLI Debugger | 1 day | High | Standalone debugging, validates architecture |
| 2. DAP Adapter | 3-4 days | High | VS Code integration |
| 3. VS Code Extension | 1-2 days | Medium | Marketplace distribution |
| **Total** | **5-7 days** | | |

## Benefits

1. **VS Code native debugging** - Familiar UI, no learning curve
2. **Standalone CLI** - Debug without IDE, scriptable
3. **Shared codebase** - pkg/adt reused across MCP and DAP
4. **AI + IDE** - Use Claude for analysis, VS Code for interaction

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| ADT listener timeout | Configurable timeout, auto-reconnect |
| Variable inspection limits | Pagination, lazy loading |
| Source file mapping | Convention-based (ADT URL → local path) |
| Multiple threads | Map SAP work processes to DAP threads |

## Next Steps

1. **Quick win:** Implement `vsp debug` CLI command (Phase 1)
2. **Validate:** Test with real SAP debugging scenarios
3. **Iterate:** Build DAP adapter incrementally
4. **Release:** Package VS Code extension for marketplace

---

## Appendix: DAP Capabilities Reference

```go
type Capabilities struct {
    // We support
    SupportsConfigurationDoneRequest bool // Yes - need to finish config before attach
    SupportsFunctionBreakpoints      bool // Yes - via SetBreakpoint
    SupportsStepBack                 bool // No - SAP doesn't support
    SupportsRestartFrame             bool // No
    SupportsGotoTargetsRequest       bool // No
    SupportsStepInTargetsRequest     bool // No
    SupportsCompletionsRequest       bool // Future - code completion in debug console
    SupportsModulesRequest           bool // Future - list loaded includes
    SupportsExceptionOptions         bool // Yes - exception breakpoints
    SupportsValueFormattingOptions   bool // No
    SupportsExceptionInfoRequest     bool // Yes - exception details
    SupportTerminateDebuggee         bool // Yes - detach releases
    SupportsDelayedStackTraceLoading bool // Future
    SupportsLoadedSourcesRequest     bool // Future - list all source files
    SupportsLogPoints                bool // No - SAP doesn't support
    SupportsTerminateThreadsRequest  bool // No
    SupportsSetExpression            bool // No
    SupportsTerminateRequest         bool // Yes
    SupportsDataBreakpoints          bool // Future - watchpoints
    SupportsReadMemoryRequest        bool // No
    SupportsDisassembleRequest       bool // No
    SupportsCancelRequest            bool // Yes
    SupportsBreakpointLocationsReq   bool // Future
    SupportsClipboardContext         bool // No
    SupportsSteppingGranularity      bool // No
    SupportsInstructionBreakpoints   bool // No
    SupportsExceptionFilterOptions   bool // Yes
}
```

## Appendix: Interactive CLI Commands

| Command | Short | Description |
|---------|-------|-------------|
| `step` | `s` | Step into next statement |
| `next` | `n` | Step over (same as step-over) |
| `out` | `o` | Step out of current routine |
| `continue` | `c` | Continue execution |
| `stack` | `r` | Show call stack |
| `vars` | `v` | Show local variables |
| `watch` | `w` | Add watch expression |
| `break` | `b` | Set breakpoint |
| `delete` | `d` | Delete breakpoint |
| `list` | `l` | List source around current line |
| `print` | `p` | Evaluate expression |
| `quit` | `q` | Detach and exit |
| `help` | `h` | Show help |
