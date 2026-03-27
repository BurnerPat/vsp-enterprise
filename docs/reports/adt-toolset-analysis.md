# ADT Toolset Analysis Report

**Date:** 2025-12-01
**Subject:** Analysis of MCP ABAP ADT Server vs abap-adt-api Library Capabilities

---

## Executive Summary

This report compares what's **exposed in the MCP server** (`mcp-abap-adt` by orchestraight.co/mario-andreschak) vs what's **available in the underlying ADT API** (both the raw SAP ADT REST endpoints and the `abap-adt-api` library by Marcello Urbani).

**Key Finding:** The MCP server implements **custom direct HTTP calls** to SAP ADT endpoints - it does **NOT** use the `abap-adt-api` library. This means there's significant functionality available via ADT that is not exposed in the current MCP server.

---

## Part 1: What's Currently Exposed in MCP Server

The MCP server exposes **13 tools** via direct axios HTTP calls to ADT REST endpoints:

| Tool | ADT Endpoint | Operation |
|------|--------------|-----------|
| `GetProgram` | `/sap/bc/adt/programs/programs/{name}/source/main` | GET source code |
| `GetClass` | `/sap/bc/adt/oo/classes/{name}/source/main` | GET source code |
| `GetInterface` | `/sap/bc/adt/oo/interfaces/{name}/source/main` | GET source code |
| `GetInclude` | `/sap/bc/adt/programs/includes/{name}/source/main` | GET source code |
| `GetFunction` | `/sap/bc/adt/functions/groups/{group}/fmodules/{name}/source/main` | GET source code |
| `GetFunctionGroup` | `/sap/bc/adt/functions/groups/{name}/source/main` | GET source code |
| `GetTable` | `/sap/bc/adt/ddic/tables/{name}/source/main` | GET structure definition |
| `GetStructure` | `/sap/bc/adt/ddic/structures/{name}/source/main` | GET structure definition |
| `GetTypeInfo` | `/sap/bc/adt/ddic/domains/{name}` or `/dataelements/{name}` | GET type info |
| `GetPackage` | `/sap/bc/adt/repository/nodestructure` | POST with params |
| `GetTransaction` | `/sap/bc/adt/repository/informationsystem/objectproperties/values` | GET metadata |
| `SearchObject` | `/sap/bc/adt/repository/informationsystem/search?operation=quickSearch` | GET search results |
| `GetTableContents` | `/z_mcp_abap_adt/z_tablecontent/{name}` | **Custom service required!** |

### Notes on Current Implementation
- All tools are **read-only** (GET/POST for retrieval only)
- No authentication to `abap-adt-api` library - uses custom axios implementation
- CSRF token handling implemented for POST requests
- `GetTableContents` requires a **custom ABAP service** implementation

---

## Part 2: What's Available via abap-adt-api Library (Marcello Urbani)

The `abap-adt-api` library provides a comprehensive TypeScript/JavaScript client for SAP ADT with **100+ methods** across these categories:

### A. Source Code Operations (Partially Exposed)
| Method | Description | In MCP? |
|--------|-------------|---------|
| `getObjectSource()` | Retrieve source code | Yes (partial) |
| `setObjectSource()` | Update source with lock/transport | **No** |
| `syntaxCheck()` | Validate ABAP/CDS syntax | **No** |
| `prettyPrinter()` | Format source code | **No** |
| `prettyPrinterSetting()` | Get formatting preferences | **No** |

### B. Unit Testing (NOT Exposed)
| Method | Description | In MCP? |
|--------|-------------|---------|
| `unitTestRun()` | Execute unit tests | **No** |
| `unitTestEvaluation()` | Evaluate test results | **No** |
| `unitTestOccurrenceMarkers()` | Map test coverage points | **No** |
| `runClass()` | Execute ABAP class directly | **No** |
| `createTestInclude()` | Create unit test class include | **No** |

### C. Debugging (NOT Exposed)
| Method | Description | In MCP? |
|--------|-------------|---------|
| `debuggerListeners()` | Check for listening debuggers | **No** |
| `debuggerListen()` | Wait for debug session | **No** |
| `debuggerSetBreakpoints()` | Create breakpoints | **No** |
| `debuggerDeleteBreakpoints()` | Remove breakpoints | **No** |
| `debuggerAttach()` | Connect to running debuggee | **No** |
| `debuggerStackTrace()` | Retrieve call stack | **No** |
| `debuggerVariables()` | Inspect variable values | **No** |
| `debuggerChildVariables()` | Explore nested variables | **No** |
| `debuggerStep()` | Execute step commands | **No** |
| `debuggerGoToStack()` | Navigate stack frames | **No** |
| `debuggerSetVariableValue()` | Modify variable during debug | **No** |
| `debuggerSaveSettings()` | Persist debug options | **No** |
| `debuggerDeleteListener()` | Stop debug listener | **No** |

### D. Code Quality & ATC Checks (NOT Exposed)
| Method | Description | In MCP? |
|--------|-------------|---------|
| `atcCustomizing()` | Get ATC settings | **No** |
| `atcCheckVariant()` | Validate check variants | **No** |
| `createAtcRun()` | Initiate code check run | **No** |
| `atcWorklists()` | Fetch check results | **No** |
| `fixProposals()` | Suggest code fixes | **No** |
| `fixEdits()` | Apply fix recommendations | **No** |
| `atcExemptProposal()` | Exempt findings | **No** |
| `atcRequestExemption()` | Request exemption approval | **No** |

### E. Code Intelligence (NOT Exposed)
| Method | Description | In MCP? |
|--------|-------------|---------|
| `codeCompletion()` | Completion suggestions | **No** |
| `codeCompletionFull()` | Extended completion | **No** |
| `codeCompletionElement()` | Details for completion items | **No** |
| `findDefinition()` | Locate symbol definitions | **No** |
| `usageReferences()` | Find all symbol usages | **No** |
| `usageReferenceSnippets()` | Usage context snippets | **No** |

### F. Refactoring (NOT Exposed)
| Method | Description | In MCP? |
|--------|-------------|---------|
| `renameEvaluate()` | Propose rename refactoring | **No** |
| `renamePreview()` | Preview rename changes | **No** |
| `renameExecute()` | Apply rename refactoring | **No** |
| `extractMethodEvaluate()` | Propose method extraction | **No** |
| `extractMethodPreview()` | Preview extraction changes | **No** |
| `extractMethodExecute()` | Apply method extraction | **No** |

### G. Transport Management (NOT Exposed)
| Method | Description | In MCP? |
|--------|-------------|---------|
| `transportInfo()` | Get transport details | **No** |
| `createTransport()` | Create transport request | **No** |
| `userTransports()` | Get user's transports | **No** |
| `transportRelease()` | Finalize transport | **No** |
| `transportDelete()` | Remove transport | **No** |
| `transportSetOwner()` | Change ownership | **No** |
| `transportAddUser()` | Grant transport access | **No** |
| `transportReference()` | Link objects to transports | **No** |

### H. Object Lifecycle (NOT Exposed)
| Method | Description | In MCP? |
|--------|-------------|---------|
| `createObject()` | Create new ABAP objects | **No** |
| `deleteObject()` | Remove objects | **No** |
| `lock()` | Acquire edit lock | **No** |
| `unLock()` | Release lock | **No** |
| `activate()` | Activate inactive objects | **No** |
| `inactiveObjects()` | List inactive objects | **No** |
| `validateNewObject()` | Check creation rules | **No** |

### I. Git Integration (NOT Exposed)
| Method | Description | In MCP? |
|--------|-------------|---------|
| `gitRepos()` | List repositories | **No** |
| `gitExternalRepoInfo()` | Query remote repo | **No** |
| `gitCreateRepo()` | Create repo link | **No** |
| `gitPullRepo()` | Pull changes | **No** |
| `stageRepo()` | Stage changes | **No** |
| `pushRepo()` | Push changes | **No** |
| `switchRepoBranch()` | Change branch | **No** |

### J. Tracing & Diagnostics (NOT Exposed)
| Method | Description | In MCP? |
|--------|-------------|---------|
| `tracesList()` | List execution traces | **No** |
| `tracesHitList()` | Show trace data points | **No** |
| `tracesDbAccess()` | Database access traces | **No** |
| `tracesStatements()` | SQL statements | **No** |
| `dumps()` | Retrieve system dumps | **No** |

### K. Class Analysis (NOT Exposed)
| Method | Description | In MCP? |
|--------|-------------|---------|
| `classComponents()` | List class members | **No** |
| `classIncludes()` | Extract include URIs | **No** |
| `mainPrograms()` | Get main program info | **No** |
| `typeHierarchy()` | Show inheritance | **No** |

### L. Service Bindings (NOT Exposed)
| Method | Description | In MCP? |
|--------|-------------|---------|
| `publishServiceBinding()` | Make service public | **No** |
| `unPublishServiceBinding()` | Remove binding | **No** |
| `bindingDetails()` | Get binding info | **No** |

### M. Data Access (Partially Exposed)
| Method | Description | In MCP? |
|--------|-------------|---------|
| `ddicElement()` | Get DDIC definitions | Yes (partial) |
| `tableContents()` | Read table data | Custom impl |
| `runQuery()` | Execute SQL queries | **No** |
| `ddicRepositoryAccess()` | Repository data | **No** |

### N. Documentation (NOT Exposed)
| Method | Description | In MCP? |
|--------|-------------|---------|
| `abapDocumentation()` | ABAP documentation | **No** |
| `packageSearchHelp()` | Package lookup | **No** |

---

## Part 3: What's Available via Raw SAP ADT REST API

The SAP ADT REST API is documented in SAP's official documentation and provides additional endpoints beyond what `abap-adt-api` wraps:

### Core Endpoints (many not exposed)
```
/sap/bc/adt/
├── programs/programs/           # Programs (exposed)
├── programs/includes/           # Includes (exposed)
├── oo/classes/                  # Classes (exposed)
├── oo/interfaces/               # Interfaces (exposed)
├── functions/groups/            # Function Groups (exposed)
├── ddic/tables/                 # Tables (exposed)
├── ddic/structures/             # Structures (exposed)
├── ddic/domains/                # Domains (partial)
├── ddic/dataelements/           # Data Elements (partial)
├── ddic/views/                  # Views (NOT exposed)
├── ddic/lockobjects/            # Lock Objects (NOT exposed)
├── ddic/searchhelps/            # Search Helps (NOT exposed)
├── repository/
│   ├── nodestructure            # Package contents (exposed)
│   ├── informationsystem/       # Search (exposed)
│   └── transportrequests/       # Transports (NOT exposed)
├── cts/                         # Change Transport System (NOT exposed)
├── atc/                         # ATC checks (NOT exposed)
├── unittest/                    # Unit tests (NOT exposed)
├── debugger/                    # Debugger (NOT exposed)
├── traces/                      # Traces (NOT exposed)
├── discovery                    # Capability discovery (NOT exposed)
└── git/                         # abapGit integration (NOT exposed)
```

---

## Part 4: Recommendations for Extension

### High-Value Additions

#### 1. Unit Testing (Priority: HIGH)
```typescript
// Potential new tools:
- RunUnitTests(class_name, method_name?)
- GetTestResults(test_run_id)
- GetTestCoverage(class_name)
```
**ADT Endpoints:**
- `POST /sap/bc/adt/unittest/runner` - Run tests
- `GET /sap/bc/adt/unittest/results/{id}` - Get results

#### 2. ATC/Code Quality Checks (Priority: HIGH)
```typescript
// Potential new tools:
- RunAtcCheck(object_name, check_variant?)
- GetAtcResults(worklist_id)
- GetFixProposals(finding_id)
```

#### 3. Code Intelligence (Priority: MEDIUM)
```typescript
// Potential new tools:
- FindDefinition(uri, line, column)
- FindReferences(object_name)
- GetCodeCompletion(uri, line, column)
```

#### 4. Transport Management (Priority: MEDIUM)
```typescript
// Potential new tools:
- GetUserTransports(username?)
- GetTransportInfo(transport_id)
- GetTransportContents(transport_id)
```

#### 5. Syntax Check (Priority: MEDIUM)
```typescript
// Potential new tools:
- CheckSyntax(object_uri)
```

#### 6. Object Metadata (Priority: LOW)
```typescript
// Potential new tools:
- GetWhereUsed(object_name)
- GetClassComponents(class_name)
- GetTypeHierarchy(class_name)
```

---

## Part 5: Architecture Recommendation

### Option A: Continue with Direct HTTP Calls
- **Pro:** No external dependency, full control
- **Con:** Must implement each ADT feature manually

### Option B: Integrate abap-adt-api Library
- **Pro:** 100+ methods already implemented, battle-tested
- **Con:** External dependency, may need to adapt to MCP patterns

### Recommendation
Consider **hybrid approach**:
1. Keep current implementation for basic read operations
2. Add `abap-adt-api` as dependency for complex operations (unit tests, debugging, refactoring)
3. Create wrapper handlers that translate MCP calls to `abap-adt-api` methods

---

## Appendix: Coverage Statistics

| Category | Available in ADT | Exposed in MCP | Coverage |
|----------|------------------|----------------|----------|
| Source Code Read | 15+ object types | 8 | ~53% |
| Source Code Write | Full CRUD | 0 | 0% |
| Unit Testing | Full support | 0 | 0% |
| Debugging | Full support | 0 | 0% |
| Code Quality (ATC) | Full support | 0 | 0% |
| Code Intelligence | Full support | 0 | 0% |
| Refactoring | Full support | 0 | 0% |
| Transport Mgmt | Full support | 0 | 0% |
| Git Integration | Full support | 0 | 0% |
| Tracing | Full support | 0 | 0% |
| Search | Quick search | 1 | ~20% |

**Overall Estimated Coverage: ~10-15% of available ADT functionality**

---

## References

- **MCP Server Source:** https://github.com/mario-andreschak/mcp-abap-adt
- **abap-adt-api Library:** https://github.com/marcellourbani/abap-adt-api
- **SAP ADT REST API Documentation:** SAP Help Portal (requires SAP account)
- **ABAP Remote FS (uses abap-adt-api):** https://github.com/marcellourbani/vscode_abap_remote_fs
