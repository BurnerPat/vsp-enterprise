# VSP Tool Introspection & Analysis Report

**Date:** 2026-01-04
**Report ID:** 001
**Subject:** Comprehensive analysis of all MCP tools - functionality, naming, descriptions, gaps

---

## Executive Summary

VSP exposes **99 tools** in expert mode (70 in focused mode). This report analyzes each tool for:
1. Functionality correctness
2. Naming consistency
3. Description quality for AI agents
4. Usefulness ranking
5. Gaps and missing tools

---

## Tool Inventory by Category

### 1. Source Code Read Tools (12 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| **GetSource** | Focused | Working | Excellent | Unified tool - replaces type-specific tools |
| GetProgram | Expert | Working | Good | Legacy - superseded by GetSource |
| GetClass | Expert | Working | Good | Legacy - superseded by GetSource |
| GetInterface | Expert | Working | Good | Legacy - superseded by GetSource |
| GetFunction | Expert | Working | Good | Requires function_group param |
| GetFunctionGroup | Focused | Working | Good | Returns JSON with module list |
| GetInclude | Expert | Working | Good | Legacy - superseded by GetSource |
| GetClassInclude | Expert | Working | Good | For test/local includes |
| GetTable | Focused | Working | Good | Returns structure, not contents |
| GetStructure | Expert | Working | Good | DDIC structure definition |
| GetTransaction | Expert | Working | Good | Transaction code details |
| GetTypeInfo | Expert | Working | Good | DDIC type information |

**Naming Issues:**
- `GetTable` vs `GetTableContents` - clear distinction
- `GetFunctionGroup` - returns JSON while others return source (inconsistent)

**Recommendations:**
- Deprecate legacy Get* tools in favor of unified GetSource
- Add `format` parameter to GetFunctionGroup (source vs JSON)

---

### 2. Source Code Write Tools (8 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| **WriteSource** | Focused | Working | Excellent | Unified - auto create/update |
| **EditSource** | Focused | Working | Excellent | Surgical replacement |
| WriteProgram | Expert | Working | Good | Legacy workflow |
| WriteClass | Expert | Working | Good | Legacy workflow |
| UpdateSource | Expert | Working | Good | Low-level, requires lock |
| UpdateClassInclude | Expert | Working | Good | For test/local includes |
| CreateTestInclude | Expert | Working | Good | Creates test include |
| CreateAndActivateProgram | Expert | Working | Good | Complex workflow |
| CreateClassWithTests | Expert | Working | Good | Complex workflow |

**Naming Issues:**
- WriteSource vs WriteProgram/WriteClass - naming overlap, confusion
- UpdateSource sounds like it should be the unified tool

**Recommendations:**
- Rename WriteProgram → UpdateProgram or deprecate
- Rename WriteClass → UpdateClass or deprecate
- Consider removing low-level UpdateSource from focused mode

---

### 3. Object CRUD Tools (7 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| CreateObject | Expert | Working | Excellent | Supports 11 object types |
| CreatePackage | Focused | Working | Good | Simplified for local packages |
| DeleteObject | Expert | Working | Good | Requires lock handle |
| LockObject | Focused | Working | Good | Returns lock handle |
| UnlockObject | Focused | Working | Good | Requires lock handle |
| RenameObject | Expert | Working | Good | Copy-delete workflow |
| GetInactiveObjects | Focused | Working | Good | Lists pending activations |

**Naming Issues:** None significant

**Recommendations:**
- Add `MoveObject` tool (change package without rename)
- Consider `GetLockedObjects` tool

---

### 4. Search & Grep Tools (6 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| **SearchObject** | Focused | Working | Good | Quick search with wildcards |
| **GrepObjects** | Focused | Working | Excellent | Multi-object regex search |
| **GrepPackages** | Focused | Working | Excellent | Multi-package + recursive |
| GrepObject | Expert | Working | Good | Single object search |
| GrepPackage | Expert | Working | Good | Single package search |

**Naming Issues:**
- GrepObjects vs GrepObject - plural is unified version (good pattern)
- SearchObject - should be pluralized for consistency?

**Recommendations:**
- Consider `SearchObjects` as plural form
- Add `SearchPackage` for object listing (vs grep for content)

---

### 5. Code Intelligence Tools (7 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| **FindDefinition** | Focused | Working | Good | Go-to-definition |
| **FindReferences** | Focused | Working | Good | Where-used |
| CodeCompletion | Expert | Working | Fair | Requires source context |
| GetTypeHierarchy | Expert | Working | Good | Super/sub types |
| GetClassComponents | Expert | Working | Good | Class structure |
| PrettyPrint | Focused | Working | Good | Code formatting |
| GetPrettyPrinterSettings | Expert | Working | Good | Formatter config |
| SetPrettyPrinterSettings | Expert | Working | Good | Formatter config |

**Naming Issues:**
- FindDefinition vs FindReferences - consistent pattern
- CodeCompletion - should be GetCodeCompletion for consistency

**Description Issues:**
- CodeCompletion description is vague about what's returned

**Recommendations:**
- Rename `CodeCompletion` → `GetCodeCompletion`
- Improve CodeCompletion description with example output

---

### 6. Development Tools (5 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| **SyntaxCheck** | Focused | Working | Good | Returns errors/warnings |
| **Activate** | Focused | Working | Good | Activation |
| **RunUnitTests** | Focused | Working | Good | ABAP Unit |
| **RunATCCheck** | Focused | Working | Excellent | Code quality |
| GetATCCustomizing | Expert | Working | Good | ATC configuration |

**Naming Issues:** None

**Recommendations:**
- Add `RunATCCheckPackage` for package-wide ATC
- Add `GetUnitTestCoverage` for coverage metrics

---

### 7. Code Analysis Infrastructure (7 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| **GetCallGraph** | Focused | Working | Good | Call hierarchy |
| **GetObjectStructure** | Focused | Working | Good | Object explorer |
| **GetCallersOf** | Focused | Working | Good | Simplified up-traversal |
| **GetCalleesOf** | Focused | Working | Good | Simplified down-traversal |
| **AnalyzeCallGraph** | Focused | Working | Good | Statistics |
| **CompareCallGraphs** | Focused | Working | Good | Static vs actual |
| **TraceExecution** | Focused | Working | Excellent | Composite RCA |

**Naming Issues:**
- GetCallersOf/GetCalleesOf - great simplification pattern

**Recommendations:**
- Add `GetImpactAnalysis` - what's affected by change
- Add `GetDeadCode` - unreachable code detection

---

### 8. Runtime Diagnostics (6 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| **GetDumps** | Focused | Working | Good | List short dumps |
| **GetDump** | Focused | Working | Good | Dump details |
| **ListTraces** | Focused | Working | Good | ATRA traces |
| **GetTrace** | Focused | Working | Good | Trace analysis |
| **GetSQLTraceState** | Focused | Working | Good | ST05 state |
| **ListSQLTraces** | Focused | Working | Good | ST05 files |

**Naming Issues:**
- GetDumps (plural) vs GetDump (singular) - inconsistent with List pattern
- Should be ListDumps/GetDump for consistency

**Recommendations:**
- Rename `GetDumps` → `ListDumps` for consistency
- Add `StartSQLTrace`/`StopSQLTrace` for trace control
- Add `GetSQLTrace` for trace content (symmetric with GetTrace)

---

### 9. ABAP Debugger Tools (10 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| **SetBreakpoint** | Focused | Partial | Good | WebSocket required |
| **GetBreakpoints** | Focused | Partial | Good | WebSocket required |
| **DeleteBreakpoint** | Focused | Partial | Good | WebSocket required |
| **CallRFC** | Focused | Working | Good | Trigger execution |
| **DebuggerListen** | Focused | Working | Excellent | Blocking wait |
| **DebuggerAttach** | Focused | Working | Good | Attach to debuggee |
| **DebuggerDetach** | Focused | Working | Good | Release debuggee |
| **DebuggerStep** | Focused | Working | Good | Step operations |
| **DebuggerGetStack** | Focused | Working | Good | Call stack |
| **DebuggerGetVariables** | Focused | Working | Good | Variable values |

**Naming Issues:**
- SetBreakpoint vs DebuggerAttach - prefix inconsistency
- Should be consistent: all Debugger* or all Debug* or all Bp*

**Recommendations:**
- Standardize naming: DebuggerSetBreakpoint, DebuggerGetBreakpoints, etc.
- Add `DebuggerEvaluate` for expression evaluation
- Add `DebuggerSetWatchpoint` for data change detection

---

### 10. AMDP (HANA) Debugger (7 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| AMDPDebuggerStart | Focused | Experimental | Good | Session start |
| AMDPDebuggerResume | Focused | Experimental | Fair | Status check |
| AMDPDebuggerStop | Focused | Experimental | Good | Session end |
| AMDPDebuggerStep | Focused | Experimental | Good | Step operations |
| AMDPGetVariables | Focused | Experimental | Good | SQLScript vars |
| AMDPSetBreakpoint | Focused | Experimental | Good | SQLScript BP |
| AMDPGetBreakpoints | Focused | Experimental | Good | List BPs |

**Naming Issues:**
- AMDP prefix vs Debugger prefix - inconsistent with ABAP debugger
- AMDPGetVariables vs AMDPDebuggerGetVariables - missing Debugger

**Description Issues:**
- AMDPDebuggerResume - description says "status" but name says "resume"

**Recommendations:**
- Rename all to AMDPDebugger* pattern for consistency
- Fix AMDPDebuggerResume → AMDPDebuggerGetStatus
- Add AMDPDebuggerGetStack for SQLScript call stack

---

### 11. Transport Management (7 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| **ListTransports** | Focused | Working | Good | User's transports |
| **GetTransport** | Focused | Working | Good | Transport details |
| CreateTransport | Expert | Working | Good | Create request |
| ReleaseTransport | Expert | Working | Excellent | Irreversible warning |
| DeleteTransport | Expert | Working | Good | Delete request |
| GetUserTransports | Expert | Working | Good | By username |
| GetTransportInfo | Expert | Working | Good | Object transport info |

**Naming Issues:**
- ListTransports vs GetUserTransports - overlapping functionality

**Recommendations:**
- Merge GetUserTransports into ListTransports with user param
- Add `AddToTransport` for adding objects
- Add `GetTransportLogs` for import/export logs

---

### 12. UI5/Fiori BSP Tools (7 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| **UI5ListApps** | Focused | Working | Good | List applications |
| **UI5GetApp** | Focused | Working | Good | App details |
| **UI5GetFileContent** | Focused | Working | Good | File content |
| UI5UploadFile | Expert | Not Working | Good | ADT API limitation |
| UI5DeleteFile | Expert | Not Working | Good | ADT API limitation |
| UI5CreateApp | Expert | Not Working | Good | ADT API limitation |
| UI5DeleteApp | Expert | Not Working | Good | ADT API limitation |

**Naming Issues:**
- Consistent UI5* prefix

**Recommendations:**
- Mark write operations as "Requires alternate API" in description
- Investigate /UI5/CL_REPOSITORY_LOAD for write operations
- Remove non-working tools from focused mode

---

### 13. Git/abapGit Integration (2 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| **GitTypes** | Focused | Working | Good | 158 object types |
| **GitExport** | Focused | Working | Excellent | Export to ZIP |

**Naming Issues:**
- Git prefix vs abapGit - should clarify these use abapGit

**Recommendations:**
- Add `GitImport` for deploying from ZIP/URL
- Add `GitDiff` for comparing versions
- Add `GitStatus` for modified objects

---

### 14. Report Execution Tools (4 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| **RunReport** | Focused | Partial | Excellent | APC limitation |
| **GetVariants** | Focused | Working | Good | List variants |
| **GetTextElements** | Focused | Working | Good | Selection texts |
| **SetTextElements** | Focused | Working | Good | Update texts |

**Naming Issues:** None

**Recommendations:**
- Fix RunReport APC issue (XBP BAPI approach)
- Add `CreateVariant` for variant management
- Add `GetMessages` for message class texts (SE91)

---

### 15. File-Based Operations (4 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| **ImportFromFile** | Focused | Working | Excellent | File → SAP |
| **ExportToFile** | Focused | Working | Excellent | SAP → File |
| DeployFromFile | Expert | Working | Excellent | Legacy alias |
| SaveToFile | Expert | Working | Excellent | Legacy alias |

**Naming Issues:**
- ImportFromFile/ExportToFile vs DeployFromFile/SaveToFile - duplicates

**Recommendations:**
- Deprecate DeployFromFile/SaveToFile in favor of Import/Export
- Add `SyncFromDirectory` for batch operations

---

### 16. System Information (4 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| **GetSystemInfo** | Focused | Working | Good | System ID, release |
| **GetInstalledComponents** | Focused | Working | Good | Software components |
| **GetConnectionInfo** | Always | Working | Good | Current session |
| **GetFeatures** | Always | Working | Excellent | Feature detection |

**Naming Issues:** None - consistent Get* pattern

**Recommendations:**
- Add `GetLicense` for license info
- Add `GetUsers` for user list (admin)

---

### 17. Install/Setup Tools (4 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| **InstallZADTVSP** | Focused | Working | Excellent | Deploy handler |
| **InstallAbapGit** | Focused | Working | Excellent | Deploy abapGit |
| **ListDependencies** | Focused | Working | Good | Available deps |
| InstallDummyTest | Focused | Working | Good | Test workflow |

**Naming Issues:** None

**Recommendations:**
- Add `Uninstall*` counterparts
- Add `UpdateZADTVSP` for upgrades

---

### 18. Expert-Only Tools (3 tools)

| Tool | Mode | Status | Description Quality | Notes |
|------|------|--------|---------------------|-------|
| ExecuteABAP | Expert | Working | Excellent | Code execution |
| GetCDSDependencies | Focused | Working | Excellent | CDS deps |
| GetPackage | Focused | Working | Good | Package details |

**Naming Issues:** None

---

## Description Quality Assessment

### Excellent Descriptions (Rating: 5/5)
These descriptions clearly explain what the tool does, provide examples, and guide AI agents:

1. **WriteSource** - Explains create/update auto-detection, object types
2. **EditSource** - Workflow, example with line replacement
3. **DeployFromFile** - Solves token limit problem, file types
4. **TraceExecution** - Step-by-step workflow explanation
5. **ReleaseTransport** - IRREVERSIBLE warning
6. **GetFeatures** - Lists all feature categories

### Good Descriptions (Rating: 4/5)
These work but could use examples:
- Most Get* tools
- Most List* tools

### Fair Descriptions (Rating: 3/5)
Need improvement:
- **CodeCompletion** - No example of output format
- **AMDPDebuggerResume** - Name/description mismatch

### Poor Descriptions (Rating: 2/5)
None identified

---

## Naming Consistency Analysis

### Patterns Used

| Pattern | Examples | Consistency |
|---------|----------|-------------|
| Get{Object} | GetSource, GetProgram, GetTable | Good |
| List{Objects} | ListTransports, ListTraces | Good |
| {Action}{Object} | CreateObject, DeleteObject | Good |
| {Domain}{Action} | DebuggerAttach, UI5GetApp | Mixed |
| {Object}{Action} | SearchObject, SyntaxCheck | Inconsistent |

### Inconsistencies Found

1. **GetDumps vs ListDumps** - Should be List for collections
2. **SetBreakpoint vs DebuggerSetBreakpoint** - Missing prefix
3. **AMDPGetVariables vs AMDPDebuggerGetVariables** - Missing Debugger
4. **SearchObject vs SearchObjects** - Singular for search tool
5. **GrepObject vs GrepObjects** - Plural is unified (good)

### Recommended Naming Convention

```
{Domain}{Action}{Target}

Domains: Debugger, AMDP, UI5, Git, Transport
Actions: Get, List, Create, Update, Delete, Set, Run, Start, Stop
Targets: singular for single object, plural for collections
```

---

## Missing Tools / Gaps Analysis

### High Priority Gaps

| Tool | Description | Effort | Impact |
|------|-------------|--------|--------|
| **GetMessages** | Read message class texts (SE91) | 2h | High |
| **GitImport** | Deploy from abapGit ZIP | 1d | High |
| **MoveObject** | Change object package | 4h | Medium |
| **AddToTransport** | Add object to transport | 4h | High |
| **StartSQLTrace** | Start ST05 trace | 2h | Medium |
| **StopSQLTrace** | Stop ST05 trace | 2h | Medium |

### Medium Priority Gaps

| Tool | Description | Effort | Impact |
|------|-------------|--------|--------|
| **GetUnitTestCoverage** | Coverage metrics | 4h | Medium |
| **RunATCCheckPackage** | Package-wide ATC | 4h | Medium |
| **DebuggerEvaluate** | Expression evaluation | 4h | Medium |
| **GetDeadCode** | Unreachable code | 1d | Medium |
| **GetImpactAnalysis** | Change impact | 1d | Medium |
| **CreateVariant** | Report variant | 4h | Low |
| **GetTransportLogs** | Import/export logs | 4h | Low |

### Low Priority Gaps

| Tool | Description | Effort | Impact |
|------|-------------|--------|--------|
| **GetLockedObjects** | Currently locked objects | 2h | Low |
| **SearchPackage** | List package objects | 2h | Low |
| **GitDiff** | Compare versions | 1d | Low |
| **GitStatus** | Modified objects | 4h | Low |
| **Uninstall*** | Remove installed deps | 4h | Low |

---

## Tool Usefulness Rankings

### Tier 1: Essential (Daily Use)
1. **GetSource** - Foundation for all code work
2. **WriteSource** - Code modification
3. **EditSource** - Incremental changes
4. **SearchObject** - Finding objects
5. **GrepPackages** - Finding code patterns
6. **SyntaxCheck** - Validation
7. **Activate** - Deployment
8. **RunUnitTests** - Quality assurance

### Tier 2: Frequent (Weekly Use)
9. **GetTableContents** - Data inspection
10. **RunQuery** - Ad-hoc queries
11. **FindReferences** - Impact analysis
12. **GetCallGraph** - Code understanding
13. **GetDumps** - Debugging
14. **RunATCCheck** - Code quality
15. **ExportToFile** - Backup/version control
16. **ImportFromFile** - Deployment

### Tier 3: Occasional (As Needed)
17. **CreatePackage** - New development
18. **CreateObject** - New objects
19. **ListTransports** - Transport management
20. **GetFeatures** - System capability check
21. **GitExport** - Version control
22. **InstallZADTVSP** - Initial setup

### Tier 4: Specialized (Expert Use)
23. **DebuggerListen** - Interactive debugging
24. **ExecuteABAP** - Dynamic execution
25. **TraceExecution** - Performance analysis
26. **AMDPDebugger*** - HANA debugging
27. **UI5*** - Fiori development

---

## Recommendations Summary

### Immediate Actions (Quick Wins)

1. **Rename GetDumps → ListDumps** for consistency
2. **Fix AMDPDebuggerResume → AMDPDebuggerGetStatus**
3. **Improve CodeCompletion description** with example
4. **Deprecate duplicate tools** (DeployFromFile/SaveToFile)

### Short-Term Actions (This Sprint)

1. **Add GetMessages** - Complete report authoring story
2. **Fix RunReport** - XBP BAPI implementation
3. **Standardize debugger naming** - All Debugger* prefix
4. **Mark UI5 write tools as not-working** in descriptions

### Medium-Term Actions (This Month)

1. **Add MoveObject** tool
2. **Add StartSQLTrace/StopSQLTrace**
3. **Add GetUnitTestCoverage**
4. **Add GitImport** for deployment

### Long-Term Actions (This Quarter)

1. **Add full GitImport** workflow
2. **Add DebuggerEvaluate** expression evaluation
3. **Add ImpactAnalysis** tooling
4. **Investigate UI5 write operations** via alternate API

---

## Appendix: Complete Tool Count

| Category | Focused | Expert | Total |
|----------|---------|--------|-------|
| Source Read | 6 | 6 | 12 |
| Source Write | 2 | 7 | 9 |
| Object CRUD | 5 | 2 | 7 |
| Search/Grep | 4 | 2 | 6 |
| Code Intelligence | 4 | 4 | 8 |
| Development | 4 | 1 | 5 |
| Code Analysis | 7 | 0 | 7 |
| Runtime Diagnostics | 6 | 0 | 6 |
| ABAP Debugger | 10 | 0 | 10 |
| AMDP Debugger | 7 | 0 | 7 |
| Transport | 2 | 5 | 7 |
| UI5/BSP | 3 | 4 | 7 |
| Git/abapGit | 2 | 0 | 2 |
| Report Execution | 4 | 0 | 4 |
| File Operations | 2 | 2 | 4 |
| System Info | 4 | 0 | 4 |
| Install/Setup | 4 | 0 | 4 |
| Expert-Only | 0 | 1 | 1 |
| **Total** | **70** | **29** | **99** |

---

*Report generated by VSP Tool Introspection Analysis*
