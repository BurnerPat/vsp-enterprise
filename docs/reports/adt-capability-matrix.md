# ADT Capability Matrix

**Date:** 2025-12-01
**Purpose:** Comprehensive comparison of ADT capabilities across implementation layers

---

## Legend

| Symbol | Meaning |
|--------|---------|
| Y | Fully supported/implemented |
| P | Partially supported |
| N | Not supported/implemented |
| ? | Unknown/needs verification |
| - | Not applicable |

**Effort Scale (0-10):**
- 0-2: Trivial (copy existing pattern)
- 3-4: Easy (simple endpoint, clear mapping)
- 5-6: Medium (some complexity, parsing needed)
- 7-8: Hard (complex logic, multiple calls)
- 9-10: Very Hard (architectural changes needed)

---

## 1. Source Code Read Operations

| Capability | ADT Native | abap-adt-api | mcp-abap-adt | Purpose | Usefulness | Effort |
|------------|------------|--------------|--------------|---------|------------|--------|
| Get Program Source | Y | Y | **Y** | Read ABAP report code | High | - |
| Get Class Source | Y | Y | **Y** | Read OO class code | High | - |
| Get Interface Source | Y | Y | **Y** | Read interface definitions | High | - |
| Get Include Source | Y | Y | **Y** | Read include programs | High | - |
| Get Function Module Source | Y | Y | **Y** | Read FM implementation | High | - |
| Get Function Group Source | Y | Y | **Y** | Read FUGR container | Medium | - |
| Get Table Definition | Y | Y | **Y** | Read DDIC table structure | High | - |
| Get Structure Definition | Y | Y | **Y** | Read DDIC structure | High | - |
| Get Domain | Y | Y | **P** | Read domain definition | Medium | - |
| Get Data Element | Y | Y | **P** | Read data element | Medium | - |
| Get View Definition | Y | Y | N | Read DDIC view | Medium | 2 |
| Get Search Help | Y | Y | N | Read search help | Low | 2 |
| Get Lock Object | Y | Y | N | Read enqueue object | Low | 2 |
| Get Type Group | Y | Y | N | Read type-pools | Low | 2 |
| Get Message Class | Y | Y | N | Read message definitions | Medium | 3 |
| Get CDS View Source | Y | Y | N | Read CDS definitions | High | 3 |
| Get BDEF (Behavior Def) | Y | Y | N | Read RAP behavior | High | 3 |
| Get Metadata Extension | Y | Y | N | Read CDS annotations | Medium | 3 |
| Get Access Control | Y | Y | N | Read DCL definitions | Medium | 3 |
| Get Service Definition | Y | Y | N | Read service def | Medium | 3 |
| Get Service Binding | Y | Y | N | Read service binding | Medium | 3 |

---

## 2. Source Code Write Operations

| Capability | ADT Native | abap-adt-api | mcp-abap-adt | Purpose | Usefulness | Effort |
|------------|------------|--------------|--------------|---------|------------|--------|
| Update Source Code | Y | Y | N | Modify ABAP code | Critical | 5 |
| Lock Object | Y | Y | N | Acquire edit lock | Critical | 3 |
| Unlock Object | Y | Y | N | Release edit lock | Critical | 2 |
| Activate Object | Y | Y | N | Activate changes | Critical | 4 |
| Create Object | Y | Y | N | Create new dev objects | High | 6 |
| Delete Object | Y | Y | N | Remove objects | Medium | 4 |
| Get Inactive Objects | Y | Y | N | List pending activations | Medium | 3 |
| Validate New Object | Y | Y | N | Check creation rules | Medium | 3 |

---

## 3. Data Preview & Query

| Capability | ADT Native | abap-adt-api | mcp-abap-adt | Purpose | Usefulness | Effort |
|------------|------------|--------------|--------------|---------|------------|--------|
| Table Contents (basic) | Y | Y | **P*** | Read table data | High | - |
| Table Contents (filtered) | Y | Y | N | Query with WHERE | Critical | 4 |
| Run SQL Query | Y | Y | N | Execute free-form SQL | Critical | 4 |
| CDS View Preview | Y | Y | N | Query CDS views | High | 4 |

*\*Requires custom Z-service, not native ADT*

**Note:** Native ADT endpoint `/sap/bc/adt/datapreview/ddic` supports full SQL queries with WHERE clauses. The `abap-adt-api` library implements this via `tableContents(entity, rowNumber, decode, sqlQuery)`.

---

## 4. Unit Testing

| Capability | ADT Native | abap-adt-api | mcp-abap-adt | Purpose | Usefulness | Effort |
|------------|------------|--------------|--------------|---------|------------|--------|
| Run Unit Tests | Y | Y | N | Execute ABAP Unit | Critical | 5 |
| Get Test Results | Y | Y | N | Parse test outcomes | Critical | 4 |
| Test Evaluation | Y | Y | N | Re-run specific tests | High | 4 |
| Coverage Markers | Y | Y | N | Map tested code | High | 5 |
| Test Risk Levels | Y | Y | N | Filter by risk | Medium | 2 |
| Test Duration Filter | Y | Y | N | Filter by duration | Medium | 2 |
| Create Test Include | Y | Y | N | Generate test class | Medium | 5 |

**ADT Endpoints:**
- `POST /sap/bc/adt/abapunit/testruns` - Run tests
- `POST /sap/bc/adt/abapunit/testruns/evaluation` - Evaluate results

---

## 5. Debugging

| Capability | ADT Native | abap-adt-api | mcp-abap-adt | Purpose | Usefulness | Effort |
|------------|------------|--------------|--------------|---------|------------|--------|
| List Debug Listeners | Y | Y | N | Check active sessions | Medium | 3 |
| Create Debug Listener | Y | Y | N | Wait for debug event | High | 5 |
| Delete Listener | Y | Y | N | Stop listening | Low | 2 |
| Set Breakpoints | Y | Y | N | Create breakpoints | Critical | 5 |
| Conditional Breakpoints | Y | Y | N | Break on condition | High | 6 |
| Delete Breakpoints | Y | Y | N | Remove breakpoints | Medium | 2 |
| Attach to Debuggee | Y | Y | N | Connect to process | Critical | 6 |
| Get Stack Trace | Y | Y | N | View call stack | Critical | 4 |
| Inspect Variables | Y | Y | N | Read variable values | Critical | 5 |
| Child Variables | Y | Y | N | Explore structures | High | 4 |
| Step Into | Y | Y | N | Step execution | Critical | 4 |
| Step Over | Y | Y | N | Skip calls | Critical | 4 |
| Step Return | Y | Y | N | Exit method | High | 4 |
| Continue | Y | Y | N | Resume execution | Critical | 3 |
| Run to Line | Y | Y | N | Execute to cursor | Medium | 4 |
| Set Variable Value | Y | Y | N | Modify in debug | Medium | 5 |
| Save Debug Settings | Y | Y | N | Persist config | Low | 3 |

---

## 6. Code Quality (ATC)

| Capability | ADT Native | abap-adt-api | mcp-abap-adt | Purpose | Usefulness | Effort |
|------------|------------|--------------|--------------|---------|------------|--------|
| Get ATC Customizing | Y | Y | N | Read ATC settings | Medium | 3 |
| Get Check Variant | Y | Y | N | Read check config | Medium | 3 |
| Create ATC Run | Y | Y | N | Start code analysis | Critical | 5 |
| Get ATC Worklist | Y | Y | N | Read findings | Critical | 4 |
| Get Fix Proposals | Y | Y | N | Suggested fixes | High | 4 |
| Apply Fix | Y | Y | N | Auto-fix issues | High | 5 |
| Request Exemption | Y | Y | N | Waive finding | Medium | 4 |
| Exempt Proposal | Y | Y | N | Get exemption info | Low | 3 |
| Get ATC Users | Y | Y | N | List processors | Low | 2 |
| Change Contact | Y | Y | N | Reassign finding | Low | 3 |

---

## 7. Code Intelligence

| Capability | ADT Native | abap-adt-api | mcp-abap-adt | Purpose | Usefulness | Effort |
|------------|------------|--------------|--------------|---------|------------|--------|
| Code Completion | Y | Y | N | Autocomplete | High | 5 |
| Completion Details | Y | Y | N | Element info | High | 4 |
| Full Completion Text | Y | Y | N | Insert snippets | Medium | 3 |
| Find Definition | Y | Y | N | Go to definition | Critical | 4 |
| Find References | Y | Y | N | Where-used list | Critical | 4 |
| Reference Snippets | Y | Y | N | Usage context | High | 4 |
| Type Hierarchy | Y | Y | N | Inheritance tree | High | 4 |
| Fragment Mappings | Y | Y | N | Resolve locations | Medium | 4 |

---

## 8. Refactoring

| Capability | ADT Native | abap-adt-api | mcp-abap-adt | Purpose | Usefulness | Effort |
|------------|------------|--------------|--------------|---------|------------|--------|
| Rename Evaluate | Y | Y | N | Check rename | High | 5 |
| Rename Preview | Y | Y | N | Preview changes | High | 4 |
| Rename Execute | Y | Y | N | Apply rename | High | 5 |
| Extract Method Evaluate | Y | Y | N | Check extraction | High | 6 |
| Extract Method Preview | Y | Y | N | Preview extraction | High | 5 |
| Extract Method Execute | Y | Y | N | Apply extraction | High | 6 |
| Quick Fix Proposals | Y | Y | N | Get fixes | High | 4 |
| Apply Quick Fix | Y | Y | N | Auto-fix | High | 5 |

---

## 9. Syntax & Formatting

| Capability | ADT Native | abap-adt-api | mcp-abap-adt | Purpose | Usefulness | Effort |
|------------|------------|--------------|--------------|---------|------------|--------|
| Syntax Check | Y | Y | N | Validate code | Critical | 4 |
| Syntax Check Types | Y | Y | N | Available checkers | Low | 2 |
| Pretty Printer | Y | Y | N | Format code | High | 4 |
| Pretty Printer Settings | Y | Y | N | Get format config | Medium | 3 |
| Set PP Settings | Y | Y | N | Configure format | Low | 3 |

---

## 10. Transport Management

| Capability | ADT Native | abap-adt-api | mcp-abap-adt | Purpose | Usefulness | Effort |
|------------|------------|--------------|--------------|---------|------------|--------|
| Transport Info | Y | Y | N | Get transport details | High | 4 |
| Create Transport | Y | Y | N | New TR request | High | 5 |
| User Transports | Y | Y | N | List my transports | High | 4 |
| Release Transport | Y | Y | N | Finalize TR | High | 5 |
| Delete Transport | Y | Y | N | Remove TR | Medium | 3 |
| Set Owner | Y | Y | N | Change TR owner | Medium | 3 |
| Add User | Y | Y | N | Add task user | Medium | 3 |
| Transport Reference | Y | Y | N | Link object to TR | High | 4 |
| System Users | Y | Y | N | List users | Low | 2 |
| Transport Configs | Y | Y | N | Search configs | Low | 3 |
| Transports by Config | Y | Y | N | Filter transports | Medium | 3 |

---

## 11. Object Navigation & Search

| Capability | ADT Native | abap-adt-api | mcp-abap-adt | Purpose | Usefulness | Effort |
|------------|------------|--------------|--------------|---------|------------|--------|
| Quick Search | Y | Y | **Y** | Find objects | High | - |
| Node Contents | Y | Y | **Y** | Package tree | High | - |
| Object Structure | Y | Y | N | Object metadata | High | 3 |
| Find Object Path | Y | Y | N | Locate in tree | Medium | 3 |
| Load Types | Y | Y | N | Object type defs | Low | 2 |
| Transaction Details | Y | Y | **Y** | T-code info | Medium | - |
| Main Programs | Y | Y | N | Get main programs | Medium | 3 |
| Class Components | Y | Y | N | List class members | High | 4 |
| Class Includes | Y | Y | N | Class include URIs | Medium | 3 |

---

## 12. Git Integration (abapGit)

| Capability | ADT Native | abap-adt-api | mcp-abap-adt | Purpose | Usefulness | Effort |
|------------|------------|--------------|--------------|---------|------------|--------|
| List Repositories | Y | Y | N | Show linked repos | High | 4 |
| External Repo Info | Y | Y | N | Query remote | Medium | 4 |
| Create Repo Link | Y | Y | N | Link new repo | High | 5 |
| Pull Repository | Y | Y | N | Download changes | Critical | 6 |
| Stage Changes | Y | Y | N | Prepare commit | High | 5 |
| Push Repository | Y | Y | N | Upload changes | Critical | 6 |
| Check Repository | Y | Y | N | Validate state | Medium | 4 |
| Switch Branch | Y | Y | N | Change branch | High | 5 |
| Unlink Repository | Y | Y | N | Remove link | Low | 3 |

---

## 13. Tracing & Diagnostics

| Capability | ADT Native | abap-adt-api | mcp-abap-adt | Purpose | Usefulness | Effort |
|------------|------------|--------------|--------------|---------|------------|--------|
| List Traces | Y | Y | N | Show traces | Medium | 3 |
| Trace Requests | Y | Y | N | List trace reqs | Medium | 3 |
| Trace Hit List | Y | Y | N | Data points | High | 4 |
| DB Access Traces | Y | Y | N | SQL analysis | High | 4 |
| Trace Statements | Y | Y | N | SQL statements | High | 4 |
| Set Trace Params | Y | Y | N | Configure trace | Medium | 4 |
| Create Trace Config | Y | Y | N | New trace setup | Medium | 5 |
| Delete Trace Config | Y | Y | N | Remove config | Low | 2 |
| Delete Traces | Y | Y | N | Clear data | Low | 2 |
| System Dumps | Y | Y | N | Short dumps | High | 4 |

---

## 14. Service Bindings & OData

| Capability | ADT Native | abap-adt-api | mcp-abap-adt | Purpose | Usefulness | Effort |
|------------|------------|--------------|--------------|---------|------------|--------|
| Publish Binding | Y | Y | N | Activate service | High | 5 |
| Unpublish Binding | Y | Y | N | Deactivate service | Medium | 4 |
| Binding Details | Y | Y | N | Service info | High | 4 |
| Annotation Defs | Y | Y | N | List annotations | Medium | 4 |

---

## 15. Session & Authentication

| Capability | ADT Native | abap-adt-api | mcp-abap-adt | Purpose | Usefulness | Effort |
|------------|------------|--------------|--------------|---------|------------|--------|
| Login | Y | Y | **Y** | Authenticate | Critical | - |
| Logout | Y | Y | N | End session | Low | 2 |
| CSRF Token | Y | Y | **Y** | Security token | Critical | - |
| Reentrance Ticket | Y | Y | N | SSO ticket | Low | 4 |
| Stateful Session | Y | Y | N | Session mode | Medium | 4 |
| ADT Discovery | Y | Y | N | Feature catalog | Medium | 4 |

---

## 16. Documentation

| Capability | ADT Native | abap-adt-api | mcp-abap-adt | Purpose | Usefulness | Effort |
|------------|------------|--------------|--------------|---------|------------|--------|
| ABAP Documentation | Y | Y | N | Help text | High | 4 |
| Package Search Help | Y | Y | N | Package lookup | Medium | 3 |
| Activity Feeds | Y | Y | N | Change history | Medium | 4 |

---

## Summary Statistics

| Category | Total Capabilities | In mcp-abap-adt | Coverage |
|----------|-------------------|-----------------|----------|
| Source Read | 20 | 10 | 50% |
| Source Write | 8 | 0 | 0% |
| Data Query | 4 | 1* | 25% |
| Unit Testing | 7 | 0 | 0% |
| Debugging | 17 | 0 | 0% |
| Code Quality | 10 | 0 | 0% |
| Code Intelligence | 8 | 0 | 0% |
| Refactoring | 8 | 0 | 0% |
| Syntax/Format | 5 | 0 | 0% |
| Transports | 11 | 0 | 0% |
| Navigation | 11 | 3 | 27% |
| Git | 9 | 0 | 0% |
| Tracing | 10 | 0 | 0% |
| Service Bindings | 4 | 0 | 0% |
| Session/Auth | 6 | 2 | 33% |
| Documentation | 3 | 0 | 0% |
| **TOTAL** | **141** | **16** | **11%** |

---

## Priority Recommendations

### Tier 1: High Impact, Medium Effort (Implement First)
1. **Table Contents with SQL Filter** - Effort: 4, Usefulness: Critical
2. **Run Unit Tests** - Effort: 5, Usefulness: Critical
3. **Syntax Check** - Effort: 4, Usefulness: Critical
4. **Find Definition** - Effort: 4, Usefulness: Critical
5. **Find References** - Effort: 4, Usefulness: Critical
6. **Create ATC Run** - Effort: 5, Usefulness: Critical

### Tier 2: High Impact, Higher Effort
7. **Update Source Code** - Effort: 5, Usefulness: Critical
8. **Lock/Unlock Objects** - Effort: 3, Usefulness: Critical
9. **Activate Object** - Effort: 4, Usefulness: Critical
10. **Get ATC Worklist** - Effort: 4, Usefulness: Critical

### Tier 3: Nice to Have
11. **Pretty Printer** - Effort: 4, Usefulness: High
12. **Transport Info/Create** - Effort: 4-5, Usefulness: High
13. **Code Completion** - Effort: 5, Usefulness: High
14. **Class Components** - Effort: 4, Usefulness: High

---

## Key ADT Endpoints Reference

```
/sap/bc/adt/datapreview/ddic          # SQL query with WHERE (POST)
/sap/bc/adt/abapunit/testruns         # Unit test execution (POST)
/sap/bc/adt/checkruns                 # Syntax check (POST)
/sap/bc/adt/atc/runs                  # ATC check runs (POST)
/sap/bc/adt/atc/worklists             # ATC results (GET)
/sap/bc/adt/navigation/targets        # Find definition (GET)
/sap/bc/adt/uses                      # Find references (GET)
/sap/bc/adt/oo/classes/{name}/source/main  # Update class (PUT)
/sap/bc/adt/debugger/breakpoints      # Breakpoints (POST/DELETE)
/sap/bc/adt/cts/transportrequests     # Transport mgmt (GET/POST)
```
