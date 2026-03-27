# TAS-Style Debugging, Scripting, and Time Travel for ABAP

**Date:** 2025-12-21
**Report ID:** 001
**Subject:** Brainstorming: Tool-Assisted Debugging, Scripting Languages, and Wild Ideas
**Status:** Research / Vision Document

---

## Executive Summary

This document explores the intersection of:
1. **TAS (Tool-Assisted Speedrun/Superplay)** concepts applied to debugging
2. **Scripting languages** for debugging automation
3. **Time-travel debugging** capabilities
4. **AI-augmented exploration** of code execution

The goal: Transform debugging from a reactive, manual process into a **scriptable, replayable, AI-guided exploration** of program behavior.

---

## Part 1: TAS Concepts Applied to Debugging

### What Makes TAS Powerful?

In retro gaming, TAS enables:

| TAS Feature | Description |
|-------------|-------------|
| **Frame Advance** | Execute one frame at a time, inspect state |
| **Save States** | Snapshot entire emulator state, restore instantly |
| **Rewind** | Go backwards in time |
| **Input Recording** | Record all inputs, replay deterministically |
| **Slowdown** | Execute at any speed for analysis |
| **Memory Watch** | Monitor specific addresses in real-time |
| **Lua Scripting** | Automate input sequences, create bots |

### Translation to ABAP Debugging

| TAS Concept | ABAP Equivalent | Current Status | Potential |
|-------------|-----------------|----------------|-----------|
| Frame Advance | Step Over/Into | ✅ Works | Enhance with scripting |
| Save States | Checkpoint Variables | ❌ Not available | **HIGH PRIORITY** |
| Rewind | Time-travel debug | ❌ Not available | **GAME CHANGER** |
| Input Recording | Trace Recording | ✅ Partial (ATRA) | Extend to full replay |
| Memory Watch | Watchpoints | ⚠️ Limited | Scriptable watches |
| Lua Scripting | ??? | ❌ Not available | **THIS DOCUMENT** |

### The "Save State" Problem in SAP

In emulators, saving state is trivial - dump RAM to disk.

In SAP, "state" includes:
- Local variables (stack frames)
- Internal tables (heap)
- Database state (persistent)
- Lock entries (transactional)
- Session context (user, auth, language)
- Shared memory (buffers, caches)

**Challenge**: SAP doesn't expose a "snapshot all state" API.

**Possible Approaches**:
1. **Variable Logging**: At each breakpoint, serialize all visible variables to JSON
2. **Database Journaling**: Use DB transaction logs for data-level replay
3. **Shadow Tables**: Copy touched tables to shadow versions during debug
4. **Execution Recording**: Record all inputs/outputs, replay in simulation

---

## Part 2: Scripting Language Options

### Requirements for Debug Scripting

1. **Embeddable**: Must run inside Go binary (vsp)
2. **Fast startup**: Sub-millisecond for tight debug loops
3. **Safe**: Sandboxed, no arbitrary file/network access
4. **Expressive**: Handle complex logic, loops, conditionals
5. **Debuggable**: Good error messages, stack traces
6. **Familiar**: Developers should recognize syntax

### Candidate Languages

#### Option A: Lua

**Pros:**
- Industry standard for game scripting (WoW, Roblox, Redis)
- Tiny runtime (~200KB)
- Extremely fast (LuaJIT even faster)
- Simple C API, excellent Go bindings (gopher-lua, golua)
- Coroutines for async patterns
- Battle-tested in TAS tools (BizHawk, FCEUX)

**Cons:**
- 1-indexed arrays (confuses some devs)
- No native async/await
- Limited standard library

**Go Integration:**
```go
import "github.com/yuin/gopher-lua"

L := lua.NewState()
defer L.Close()

// Expose vsp functions to Lua
L.SetGlobal("getSource", L.NewFunction(func(L *lua.LState) int {
    objType := L.ToString(1)
    name := L.ToString(2)
    source, _ := client.GetSource(ctx, objType, name)
    L.Push(lua.LString(source))
    return 1
}))

// Run user script
L.DoString(`
    source = getSource("CLAS", "ZCL_MY_CLASS")
    print("Lines: " .. #source:split("\n"))
`)
```

**Example Debug Script:**
```lua
-- TAS-style debug script for ABAP
local bp = setBreakpoint("ZCL_PRICING", "CALCULATE", 42)

while true do
    local event = waitForBreakpoint(30)  -- 30 second timeout
    if not event then break end

    -- Log all variables
    local vars = getVariables("local")
    saveState("checkpoint_" .. event.hit_count, vars)

    -- Conditional logic
    if vars.LV_AMOUNT > 1000000 then
        print("SUSPICIOUS: Large amount detected!")
        -- Fork into investigation mode
        setWatchpoint("LV_AMOUNT", "write")
    end

    stepOver()
end
```

#### Option B: JavaScript (via Goja)

**Pros:**
- Most familiar language globally
- Modern syntax (async/await, arrow functions)
- Goja is pure Go, no CGO needed
- JSON native (perfect for SAP data)
- TypeScript possible via transpilation

**Cons:**
- Larger runtime than Lua
- Goja doesn't support all ES6+ features
- No true parallelism (single-threaded)

**Go Integration:**
```go
import "github.com/dop251/goja"

vm := goja.New()

// Expose vsp functions
vm.Set("getSource", func(objType, name string) string {
    source, _ := client.GetSource(ctx, objType, name)
    return source
})

// Run user script
vm.RunString(`
    const source = getSource("CLAS", "ZCL_MY_CLASS");
    console.log("Lines:", source.split("\n").length);
`)
```

**Example Debug Script:**
```javascript
// TAS-style debug script for ABAP
async function investigateDump(dumpId) {
    const dump = await getDump(dumpId);
    const callStack = dump.stack;

    // Build call graph from crash point
    const graph = await getCallersOf(callStack[0].uri, 5);

    // Find all paths that lead here
    const paths = flattenCallGraph(graph);

    // For each path, set breakpoints and record
    for (const path of paths) {
        const recording = await recordExecution({
            entryPoint: path.root,
            breakpoints: path.nodes.map(n => n.uri),
            captureVariables: true
        });

        // Compare recordings to find divergence
        recordings.push(recording);
    }

    // AI analysis
    const analysis = await analyzeRecordings(recordings);
    return analysis.rootCause;
}
```

#### Option C: Starlark (Python-like)

**Pros:**
- Python-like syntax (familiar to data scientists)
- Designed for configuration/scripting (Bazel, Buck)
- Deterministic execution (hermetic)
- Pure Go implementation
- Safe by default (no I/O, no imports)

**Cons:**
- No async support
- Limited ecosystem
- Less known than Lua/JS

#### Option D: Custom DSL

**Pros:**
- Tailored exactly to debugging needs
- Can enforce safety constraints
- Domain-specific optimizations

**Cons:**
- Learning curve for users
- Maintenance burden
- No ecosystem

**Example Custom DSL:**
```
# Debug workflow DSL
WORKFLOW investigate_dump(dump_id):
    dump = GET_DUMP(dump_id)

    FOR frame IN dump.stack:
        SET_BREAKPOINT(frame.program, frame.line)

    RECORD_EXECUTION:
        TRIGGER: RUN_UNIT_TESTS(dump.program)
        CAPTURE: ALL_VARIABLES
        MAX_STEPS: 1000

    COMPARE_TRACES:
        STATIC: GET_CALL_GRAPH(dump.program)
        ACTUAL: RECORDED_TRACE

    REPORT: divergence_points
```

### Recommendation: Lua + JS Hybrid

**Phase 1**: Lua for core scripting (lightweight, TAS-familiar)
**Phase 2**: JS/Goja for complex workflows (async, JSON-native)
**Phase 3**: Custom DSL for high-level workflows (user-friendly)

---

## Part 3: Time-Travel Debugging

### The Dream

Imagine debugging like this:

```
> vsp debug --time-travel ZCL_MY_CLASS

[Recording execution...]
Hit breakpoint at line 42.

(vsp-debug) print LV_AMOUNT
LV_AMOUNT = 1000

(vsp-debug) step 10           # Step forward 10 times
(vsp-debug) print LV_AMOUNT
LV_AMOUNT = -500              # Wait, how did it go negative?

(vsp-debug) rewind 5          # Go back 5 steps
(vsp-debug) print LV_AMOUNT
LV_AMOUNT = 500               # Still positive here

(vsp-debug) step              # Step once
(vsp-debug) print LV_AMOUNT
LV_AMOUNT = -500              # THIS is the bug!

(vsp-debug) where             # Show current location
→ Line 47: LV_AMOUNT = LV_AMOUNT - LV_DISCOUNT.
                              # Overflow! LV_DISCOUNT was 1000!
```

### Implementation Approaches

#### Approach A: Full State Recording

Record **everything** at each step:
- All local variables
- All global variables accessed
- All DB reads
- All function calls

**Storage**: Could be huge (GBs for complex programs)
**Replay**: Perfect fidelity
**Implementation**: Modify SAP kernel (impossible without SAP)

#### Approach B: Checkpoint + Delta Recording

Take full snapshots at intervals, record deltas between:
- Checkpoint every N steps
- Record only changed variables between checkpoints
- Rewind = restore checkpoint + replay deltas

**Storage**: Manageable
**Replay**: Fast for nearby states, slower for distant
**Implementation**: Possible via ADT debugger API

#### Approach C: Reverse Execution (like rr/UndoDB)

Don't record state - record **inputs**:
- Record all external inputs (DB reads, RFC calls, user input)
- To rewind: re-execute from start with recorded inputs
- Deterministic execution required

**Storage**: Minimal (only inputs)
**Replay**: Slower (must re-execute)
**Implementation**: Requires execution control we don't have

#### Approach D: Symbolic/Hybrid Execution

Combine concrete execution with symbolic analysis:
- Execute concretely, record path constraints
- At branch points, fork symbolic exploration
- "What if this condition was false?"

**Storage**: Moderate (constraints + concrete values)
**Replay**: Can explore counterfactuals
**Implementation**: Complex, requires SAP bytecode understanding

### Practical Implementation for vsp

**Phase 1: Variable History**
```go
type DebugHistory struct {
    Steps []DebugStep
}

type DebugStep struct {
    StepNumber int
    Location   string // program:line
    Variables  map[string]interface{}
    Timestamp  time.Time
}

// At each step, record state
func (d *Debugger) RecordStep() {
    vars := d.GetAllVariables()
    d.history.Steps = append(d.history.Steps, DebugStep{
        StepNumber: len(d.history.Steps),
        Location:   d.CurrentLocation(),
        Variables:  vars,
    })
}

// Pseudo-rewind: Show historical state
func (d *Debugger) ShowStateAt(stepNum int) {
    step := d.history.Steps[stepNum]
    fmt.Printf("State at step %d (%s):\n", stepNum, step.Location)
    for name, val := range step.Variables {
        fmt.Printf("  %s = %v\n", name, val)
    }
}
```

**Phase 2: Diff-Based Navigation**
```lua
-- Lua script to find when variable changed
function findChange(varName, fromStep, toStep)
    local prev = getStateAt(fromStep)[varName]
    for step = fromStep + 1, toStep do
        local curr = getStateAt(step)[varName]
        if curr ~= prev then
            return step, prev, curr
        end
        prev = curr
    end
    return nil
end

-- Usage: "When did LV_AMOUNT go negative?"
local step, before, after = findChange("LV_AMOUNT", 0, currentStep())
print(string.format("Changed at step %d: %s -> %s", step, before, after))
```

**Phase 3: Branch Exploration**
```javascript
// Fork execution at a branch point
async function exploreBranch(branchStep, condition) {
    // Save current state
    const checkpoint = await saveCheckpoint();

    // Restore to branch point
    await restoreCheckpoint(branchStep);

    // Modify condition
    await setVariable(condition.variable, condition.alternateValue);

    // Continue execution
    const alternateTrace = await continueAndRecord();

    // Restore original
    await restoreCheckpoint(checkpoint);

    return alternateTrace;
}
```

---

## Part 4: Wild Ideas

### 1. Parallel Universe Debugging

Fork execution into multiple "universes", each trying a different fix:

```lua
-- Spawn 3 parallel investigations
universes = {
    fork({ LV_DISCOUNT = 0 }),        -- What if discount was 0?
    fork({ LV_TAX_RATE = 0.19 }),     -- What if tax was different?
    fork({ skip_line = 47 }),          -- What if we skip line 47?
}

-- Run all in parallel
results = parallel_execute(universes)

-- Find which universe doesn't crash
for i, result in ipairs(results) do
    if not result.crashed then
        print("Universe " .. i .. " survived!")
        print("Fix: " .. universes[i].description)
    end
end
```

### 2. AI-Guided Fuzzing

Let Claude explore the state space:

```javascript
// AI decides what inputs to try
async function aiFuzz(program, maxIterations) {
    const crashes = [];
    const coverage = new Set();

    for (let i = 0; i < maxIterations; i++) {
        // AI generates test input based on coverage gaps
        const input = await claude.generateInput({
            program,
            coverage: Array.from(coverage),
            previousCrashes: crashes
        });

        // Execute with input
        const result = await executeWithInput(program, input);

        // Update coverage
        result.coveredLines.forEach(l => coverage.add(l));

        // Record crashes
        if (result.crashed) {
            crashes.push({ input, dump: result.dump });
        }

        // AI learns from results
        await claude.feedback({ input, result });
    }

    return { coverage, crashes };
}
```

### 3. Temporal Queries (SQL for Time)

Query execution history like a database:

```sql
-- Find all moments when LV_AMOUNT was negative
SELECT step, location, LV_AMOUNT
FROM execution_history
WHERE LV_AMOUNT < 0
ORDER BY step;

-- Find the last assignment before crash
SELECT step, location, statement
FROM execution_history
WHERE step < (SELECT step FROM crashes WHERE id = 'DUMP123')
  AND statement LIKE 'LV_AMOUNT =%'
ORDER BY step DESC
LIMIT 1;

-- Find all paths that reached line 42
SELECT DISTINCT call_stack
FROM execution_history
WHERE location = 'ZCL_PRICING:42';
```

### 4. Collaborative Multi-Agent Debugging

Multiple AI agents working together:

```yaml
agents:
  - name: "Investigator"
    role: "Explore the crash site, gather evidence"
    tools: [GetDump, GetSource, GetCallGraph]

  - name: "Historian"
    role: "Track variable changes over time"
    tools: [RecordExecution, TemporalQuery]

  - name: "Theorist"
    role: "Propose hypotheses for the bug"
    tools: [SymbolicExecution, CounterfactualAnalysis]

  - name: "Fixer"
    role: "Write and test patches"
    tools: [EditSource, RunUnitTests, SyntaxCheck]

workflow:
  1. Investigator gathers initial evidence
  2. Historian traces variable history
  3. Theorist proposes 3 hypotheses
  4. Fixer tests each hypothesis with a patch
  5. Team votes on best fix
```

### 5. Genetic Algorithm Bug Fixing

Evolve patches through natural selection:

```python
# Pseudo-code for genetic bug fixing

population = generate_random_patches(100)

for generation in range(MAX_GENERATIONS):
    # Evaluate fitness (tests passed, no crashes)
    fitness = [evaluate(patch) for patch in population]

    # Select best performers
    survivors = select_top(population, fitness, keep=20)

    # Crossover: combine successful patches
    children = crossover(survivors, count=60)

    # Mutation: random small changes
    mutants = mutate(survivors, count=20)

    # New generation
    population = survivors + children + mutants

    # Check for solution
    if max(fitness) == PERFECT_SCORE:
        return population[fitness.index(max(fitness))]
```

### 6. Dream: ABAP in a Bottle

Run ABAP in a sandboxed simulation:

```go
// Hypothetical: Full ABAP VM in Go
vm := abapvm.New()
vm.LoadProgram("ZCL_MY_CLASS")
vm.SetBreakpoint(42)

// Execute with full control
for !vm.Finished() {
    vm.Step()

    // Inspect anything
    stack := vm.CallStack()
    heap := vm.Heap()

    // Modify anything
    vm.SetVariable("LV_X", 42)

    // Fork execution
    vm2 := vm.Fork()
    vm2.SetVariable("LV_X", 0)

    // Compare outcomes
    vm.RunToEnd()
    vm2.RunToEnd()
}
```

### 7. Debugging as a Game

Gamify the debugging experience:

```
╔══════════════════════════════════════════════════════════════╗
║  BUG HUNTER 3000                          Score: 4,250 pts   ║
╠══════════════════════════════════════════════════════════════╣
║                                                              ║
║  LEVEL 7: The Phantom Null Reference                         ║
║                                                              ║
║  Mission: Find why LV_PARTNER becomes NULL                   ║
║                                                              ║
║  Clues Found: 3/5                                            ║
║  [✓] Stack trace collected                                   ║
║  [✓] Call graph mapped                                       ║
║  [✓] Variable history recorded                               ║
║  [ ] Root cause identified                                   ║
║  [ ] Fix verified                                            ║
║                                                              ║
║  Power-ups Available:                                        ║
║  [A] AI Hint (costs 100 pts)                                 ║
║  [R] Rewind 10 steps                                         ║
║  [F] Fork universe                                           ║
║                                                              ║
║  Leaderboard:                                                ║
║  1. alice@corp     - 12,450 pts                              ║
║  2. bob@corp       - 11,200 pts                              ║
║  3. claude@ai      - 99,999 pts (BANNED: Too Good)           ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
```

### 8. Self-Healing Code

AI monitors production, fixes bugs automatically:

```yaml
# Self-healing workflow
trigger: SHORT_DUMP_DETECTED

steps:
  - name: Capture
    action: GetDump

  - name: Analyze
    action: TraceExecution
    params:
      static_graph: true
      compare_actual: true

  - name: Hypothesize
    action: AI_RootCause

  - name: Patch
    action: AI_GenerateFix
    constraints:
      - no_functional_change
      - minimal_diff
      - must_pass_tests

  - name: Test
    action: RunUnitTests
    rollback_on_failure: true

  - name: Deploy
    action: CreateTransport
    requires_approval: true
    notify: [dev_team, change_manager]
```

---

## Part 5: Implementation Roadmap

### Phase 1: Foundation (Q1 2026)
- [ ] Integrate Lua scripting (gopher-lua)
- [ ] Expose all MCP tools to Lua
- [ ] Basic variable history recording
- [ ] "Show state at step N" command

### Phase 2: TAS Features (Q2 2026)
- [ ] Checkpoint/restore via variable serialization
- [ ] Execution recording (inputs + outputs)
- [ ] Replay recorded sessions
- [ ] Watchpoint scripting

### Phase 3: Time Travel Lite (Q3 2026)
- [ ] Pseudo-rewind (state inspection only)
- [ ] Delta-based history compression
- [ ] Temporal queries (basic)
- [ ] "When did X change?" command

### Phase 4: AI Integration (Q4 2026)
- [ ] AI-guided breakpoint suggestions
- [ ] Anomaly detection in traces
- [ ] Automated hypothesis generation
- [ ] Multi-agent debugging prototype

### Phase 5: Advanced (2027+)
- [ ] Full time-travel (if SAP provides kernel access)
- [ ] Symbolic execution for ABAP
- [ ] Parallel universe debugging
- [ ] Self-healing infrastructure

---

## Conclusion

The vision: **Debugging should feel like playing a TAS**

- **Scriptable**: Automate repetitive debug tasks
- **Replayable**: Record once, analyze forever
- **Reversible**: Go back in time to find bugs
- **Intelligent**: AI suggests where to look
- **Collaborative**: Multiple agents working together

The technology exists. The APIs (mostly) exist. What's needed is the integration layer - and that's exactly what vsp is building.

**Next Steps:**
1. Prototype Lua integration
2. Build variable history recording
3. Create first "debug script" examples
4. Document scripting API
5. Gather community feedback

---

## References

- [BizHawk TAS Emulator](https://github.com/TASEmulators/BizHawk) - Gold standard for TAS tools
- [rr (Record & Replay)](https://rr-project.org/) - Time-travel debugging for Linux
- [gopher-lua](https://github.com/yuin/gopher-lua) - Lua VM in pure Go
- [Goja](https://github.com/dop251/goja) - JavaScript interpreter in Go
- [ABAP Debugger Scripting](https://help.sap.com/docs/ABAP_PLATFORM/c238d694b825421f940829321ffa326a/4ec365ce6e391014adc9fffe4e204223.html) - SAP's native scripting (limited)

---

*"The best debugger is one that lets you ask 'what if?' and get an answer."*
