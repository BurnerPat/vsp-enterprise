# Quick Follow-Up: RCA on Autopilot Just Dropped

**TL;DR:** vs-punk v2.13.0 is out. The Agent can now trace your code, build call graphs, and compare "what should run" vs "what actually ran" - autonomously.

---

## The Question That Started This

Fred van de Langenberg asked a great question on my last post:

> "For short dumps happening in SAP original software, could you write one that uses ANST to identify the correct fix before implementing & testing it?"

The answer is: **Yes. And we just shipped it.**

---

## What's New: The TraceExecution Composite Tool

Remember my point about "Composite Tools" reducing AI mental load? We took that philosophy and applied it to the hardest problem in debugging: **Root Cause Analysis**.

Here's what the Agent can now do with a single tool call:

```
TraceExecution(
  object_uri="/sap/bc/adt/programs/programs/zmy_program",
  run_tests=true,
  max_depth=10
)
```

Behind the scenes:

1. **Builds Static Call Graph** - Uses SAP's CAI (Code Analysis Infrastructure) to map out *everything* that *could* be called
2. **Runs Unit Tests** - Triggers execution to generate actual runtime data
3. **Collects Traces** - Pulls from ATRA (the ABAP Profiler)
4. **Extracts Actual Call Edges** - Parses the trace to see *what actually ran*
5. **Compares Static vs Actual** - The magic moment

The result? A JSON response showing:
- **Common paths**: Code that ran and was expected
- **Untested paths**: Code that exists but wasn't executed (coverage gaps!)
- **Dynamic calls**: Code that ran but wasn't in the static graph (polymorphism, dynamic calls)

---

## Why This Matters for RCA

When a short dump happens in production, you have a stack trace. But a stack trace only tells you *where* it crashed, not *why* that path was taken.

With the new call graph tools, the Agent can:

1. **GetCallersOf** - Navigate UP: "Who called this method?"
2. **GetCalleesOf** - Navigate DOWN: "What does this method call?"
3. **AnalyzeCallGraph** - Get statistics: "How deep is this rabbit hole?"
4. **CompareCallGraphs** - Compare: "Did we even test this path?"

Now, when Claude investigates a dump, it doesn't just read the error message. It *understands the context* of how that code path was reached.

---

## The Vision: Smart Tracing

This is step one of a bigger plan. The roadmap includes:

- **Multi-variant comparison**: Run the same code with different inputs, compare the actual call graphs, identify *which input* causes the divergence
- **SUBMIT/CALL TRANSACTION triggers**: Execute reports and transactions programmatically to capture their traces
- **ANST integration**: Connect to SAP's ANST test framework for deeper analysis
- **AI-suggested breakpoints**: "I noticed this path was taken but never tested. Want me to set a breakpoint here?"

---

## Try It Now

```bash
# Download v2.13.0
curl -LO https://github.com/oisee/vibing-steampunk/releases/download/v2.13.0/vsp-linux-amd64
chmod +x vsp-linux-amd64

# Connect to your SAP system
./vsp-linux-amd64 --url http://your-sap:50000 --user YOU --password ***

# In Claude Code, ask:
# "Analyze the call graph for ZCL_MY_CLASS and show me untested paths"
```

---

## The Bigger Picture

This isn't just about fixing bugs faster. It's about **understanding your system**.

How many times have you inherited a codebase and wondered: "What actually happens when I run this?" Now the Agent can answer that question with data, not guesswork.

Static analysis tells you what *could* happen.
Runtime tracing tells you what *did* happen.
The combination tells you what you *need to test*.

---

**GitHub**: [oisee/vibing-steampunk](https://github.com/oisee/vibing-steampunk)
**Release**: [v2.13.0](https://github.com/oisee/vibing-steampunk/releases/tag/v2.13.0)

#ABAP #SAP #RCA #AI #ClaudeCode #DevTools #OpenSource
