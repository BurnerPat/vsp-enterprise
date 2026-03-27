# VSP Tool Universalization Strategy

**Date:** 2026-01-04
**Report ID:** 002
**Subject:** Tool consolidation, abapGit integration, DDIC handling, and gap analysis
**Follows:** 2026-01-04-001-tool-introspection-analysis.md

---

## Corrections from Previous Report

### CompareCallGraphs Misclassification

**Wrong**: Listed under "Compare/Diff" pattern for source comparison
**Correct**: CompareCallGraphs compares **static call graph** vs **actual execution trace**

This tool is for:
- Identifying untested code paths (in static graph but not executed)
- Finding dynamic calls (executed but not in static analysis)
- Root cause analysis (RCA)

**Separate tool needed**: `CompareSource` / `DiffSource` for actual source code comparison

---

## Tool Group Reorganization

### Proposed Tool Groups

```go
toolGroups := map[string][]string{
    // Existing groups
    "5": {"UI5ListApps", "UI5GetApp", "UI5GetFileContent"},  // UI5 Read-only
    "T": {"RunUnitTests", "RunATCCheck"},                     // Testing
    "C": {"ListTransports", "GetTransport", ...},             // CTS
    "G": {"GitTypes", "GitExport"},                           // Git/abapGit
    "R": {"RunReport", "GetVariants", ...},                   // Reports
    "I": {"InstallZADTVSP", "InstallAbapGit", ...},          // Install

    // NEW: Experimental group
    "X": {
        // ABAP Debugger (requires ZADT_VSP)
        "SetBreakpoint", "GetBreakpoints", "DeleteBreakpoint",
        "DebuggerListen", "DebuggerAttach", "DebuggerDetach",
        "DebuggerStep", "DebuggerGetStack", "DebuggerGetVariables",

        // AMDP Debugger (experimental)
        "AMDPDebuggerStart", "AMDPDebuggerResume", "AMDPDebuggerStop",
        "AMDPDebuggerStep", "AMDPGetVariables", "AMDPSetBreakpoint",
        "AMDPGetBreakpoints",

        // UI5 Write (ADT limitation)
        "UI5UploadFile", "UI5DeleteFile", "UI5CreateApp", "UI5DeleteApp",

        // Report execution (APC limitation)
        "RunReport",
    },

    // NEW: DDIC group
    "D": {
        "GetTable", "CreateTable", "GetDataElement", "GetDomain",
        "GetStructure", "GetTableType",
    },
}
```

**Usage**: `--disabled-groups X` hides all experimental tools

---

## abapGit Integration Deep Dive

### Current State

| Tool | Status | Description |
|------|--------|-------------|
| GitTypes | Working | Returns 158 supported object types |
| GitExport | Working | Export packages/objects to base64 ZIP |

### Missing abapGit Tools

#### Tier 1: Essential (via ZADT_VSP WebSocket)

| Tool | Description | Complexity |
|------|-------------|------------|
| **GitImport** | Deploy objects from abapGit ZIP | Medium |
| **GitStatus** | Show modified objects vs last export | Medium |
| **GitDiff** | Diff single object vs ZIP version | Medium |

#### Tier 2: Advanced (requires abapGit installed)

| Tool | Description | Complexity |
|------|-------------|------------|
| **GitClone** | Clone from GitHub/GitLab URL | High |
| **GitPull** | Update from remote repository | High |
| **GitPush** | Push changes to remote | High |
| **GitStage** | Stage objects for commit | Medium |

### GitImport Design

```
GitImport
  --zip_base64: Base64-encoded abapGit ZIP (from GitExport or external)
  --package: Target package for import
  --transport: Transport request (optional for local)
  --mode: "create" | "update" | "upsert" (default: upsert)
  --dry_run: Preview changes without applying

Workflow:
1. Decode ZIP
2. Parse .abapgit.xml for object list
3. For each object:
   a. Check if exists (SearchObject)
   b. Create or Update via WriteSource
   c. Activate
4. Return summary
```

### GitStatus Design

```
GitStatus
  --package: Package to check
  --zip_base64: Reference ZIP to compare against

Returns:
{
  "modified": ["ZCL_FOO", "ZPROG_BAR"],
  "added": ["ZCL_NEW"],
  "deleted": ["ZCL_OLD"],
  "unchanged": ["ZIF_STABLE"]
}
```

---

## DDIC Object Handling

### Current DDIC Support

| Object | Get | Create | Edit | Activate |
|--------|-----|--------|------|----------|
| Table (TABL) | GetTable | ? | ? | Activate |
| Structure (TABL) | GetStructure | ? | ? | Activate |
| Data Element (DTEL) | GetTypeInfo | ? | ? | Activate |
| Domain (DOMA) | ? | ? | ? | Activate |
| Table Type (TTYP) | ? | ? | ? | Activate |
| View (VIEW) | ? | ? | ? | Activate |

### DDIC Objects Are NOT Source Code

**Key insight**: DDIC objects use XML definitions, not ABAP source.

```xml
<!-- Table definition format -->
<asx:abap xmlns:asx="http://www.sap.com/abapxml">
  <asx:values>
    <DD02V>
      <TABNAME>ZTABLE</TABNAME>
      <TABCLASS>TRANSP</TABCLASS>
      ...
    </DD02V>
    <DD03P>
      <FIELDNAME>MANDT</FIELDNAME>
      <POSITION>0001</POSITION>
      <DATATYPE>CLNT</DATATYPE>
      ...
    </DD03P>
  </asx:values>
</asx:abap>
```

### EditSource for DDIC?

**Question**: Can EditSource work for tables?

**Answer**: Partially. The ADT API exposes DDIC as XML, so:
- `GetSource(TABL, table_name)` → Returns XML definition
- `EditSource` → Could do XML string replacement
- But: Risky! XML structure is complex, easy to break

**Better approach**: Specialized DDIC tools

### Proposed DDIC Tools

#### Table Operations

```
CreateTable
  --name: Table name
  --description: Short description
  --package: Target package
  --delivery_class: A, C, L, G, E, S, W (default: A)
  --fields: JSON array of field definitions
    [
      {"name": "MANDT", "type": "CLNT", "key": true},
      {"name": "ID", "type": "CHAR", "length": 10, "key": true},
      {"name": "DESCRIPTION", "type": "CHAR", "length": 100}
    ]
  --transport: Transport request

AddTableField
  --table: Table name
  --field_name: New field name
  --data_element: Data element to use (or inline type)
  --position: Position in table
  --key: Is key field (default: false)

RemoveTableField
  --table: Table name
  --field_name: Field to remove

ChangeTableField
  --table: Table name
  --field_name: Field to modify
  --new_data_element: New data element
  --new_description: New description
```

#### Data Element Operations

```
CreateDataElement
  --name: Data element name
  --description: Short description
  --package: Target package
  --domain: Reference domain (or inline type)
  --labels: {"short": "ID", "medium": "Document ID", "long": "Document Identifier"}
  --transport: Transport request

GetDataElementTexts
  --name: Data element name
  --language: Language key

SetDataElementTexts
  --name: Data element name
  --language: Language key
  --labels: {"short": "...", "medium": "...", "long": "..."}
```

---

## Text Elements Universalization

### Current: GetTextElements / SetTextElements (Programs only)

### Objects with Text Pools

| Object Type | Has Text Pool | ADT Endpoint |
|-------------|---------------|--------------|
| Programs (PROG) | Yes - Selection texts, Text symbols | ✅ Implemented |
| Function Groups (FUGR) | Yes - Same format | Not implemented |
| Module Pools (PROG) | Yes - Same format | Same as PROG |

### Objects with Other Texts

| Object Type | Text Type | ADT Access |
|-------------|-----------|------------|
| Message Class (MSAG) | Message texts (000-999) | `/sap/bc/adt/messageclass/{id}` |
| Data Elements (DTEL) | Short/Medium/Long/Heading | XML in definition |
| Domains (DOMA) | Fixed value texts | XML in definition |
| Tables (TABL) | Field labels | Via data elements |

### Proposed Universal Text Tool

```
GetTexts
  --object_type: PROG, FUGR, MSAG, DTEL, DOMA
  --object_name: Object name
  --language: Language key (default: EN)

SetTexts
  --object_type: PROG, FUGR, MSAG, DTEL, DOMA
  --object_name: Object name
  --language: Language key
  --texts: JSON object (format depends on object_type)
    PROG/FUGR: {"selection_texts": {...}, "text_symbols": {...}}
    MSAG: {"000": "Message text", "001": "Another message"}
    DTEL: {"short": "...", "medium": "...", "long": "...", "heading": "..."}
    DOMA: {"value1": "Description 1", "value2": "Description 2"}
```

---

## Compare/Diff Tools (Corrected)

### Call Graph Analysis (Existing)

| Tool | Purpose |
|------|---------|
| GetCallGraph | Get static call hierarchy |
| GetCallersOf | Who calls this? (up) |
| GetCalleesOf | What does this call? (down) |
| AnalyzeCallGraph | Statistics (nodes, edges, depth) |
| **CompareCallGraphs** | Static graph vs actual execution trace |
| TraceExecution | Composite: graph + test + trace + compare |

### Source Code Comparison (NEW - Gap)

| Tool | Purpose | Priority |
|------|---------|----------|
| **CompareSource** | Diff two objects (e.g., ZFOO vs ZBAR) | Medium |
| **DiffVersions** | Diff active vs inactive version | High |
| **DiffWithTransport** | Diff local vs transported version | Medium |
| **DiffWithGit** | Diff object vs abapGit ZIP version | Medium |

### CompareSource Design

```
CompareSource
  --object1_type: Object type (PROG, CLAS, etc.)
  --object1_name: First object name
  --object2_type: Object type (default: same as object1)
  --object2_name: Second object name
  --format: "unified" | "side-by-side" | "json"

Returns:
{
  "added_lines": 15,
  "removed_lines": 8,
  "changed_lines": 23,
  "diff": "--- ZFOO\n+++ ZBAR\n@@ -1,5 +1,7 @@\n..."
}
```

---

## Batch Operations Pattern

### Current Single-Object Tools

| Tool | Single Object |
|------|---------------|
| Activate | One object |
| RunUnitTests | One object |
| RunATCCheck | One object |
| SyntaxCheck | One object |
| ExportToFile | One object |

### Proposed Batch Tools

| Tool | Batch Operation | Priority |
|------|-----------------|----------|
| **ActivatePackage** | Activate all inactive in package | High |
| **ActivateAll** | Activate all inactive for user | High |
| **RunUnitTestsPackage** | Run all tests in package | High |
| **RunATCCheckPackage** | Check entire package | Medium |
| **SyntaxCheckPackage** | Syntax check all in package | Low |
| **ExportPackage** | Export all to directory | Medium |

### ActivatePackage Design

```
ActivatePackage
  --package: Package name
  --include_subpackages: Include child packages (default: false)
  --max_objects: Limit (default: 100)

Workflow:
1. GetInactiveObjects
2. Filter by package
3. Sort by dependency (interfaces before classes, etc.)
4. Activate each
5. Return summary

Returns:
{
  "activated": ["ZIF_FOO", "ZCL_BAR", "ZPROG"],
  "failed": [{"name": "ZCL_ERROR", "reason": "Syntax error"}],
  "skipped": []
}
```

---

## Tool Consolidation Summary

### Rename/Deprecate

| Current | Action | Reason |
|---------|--------|--------|
| GetDumps | Rename → ListDumps | Consistency with List* pattern |
| DeployFromFile | Deprecate | Duplicate of ImportFromFile |
| SaveToFile | Deprecate | Duplicate of ExportToFile |
| AMDPDebuggerResume | Rename → AMDPDebuggerGetStatus | Name/description mismatch |

### Unify

| Current Tools | Unified Tool |
|---------------|--------------|
| GetProgram, GetClass, GetInterface, GetFunction, GetInclude | GetSource ✅ (done) |
| WriteProgram, WriteClass | WriteSource ✅ (done) |
| GetTextElements, SetTextElements | GetTexts/SetTexts (expand to FUGR, MSAG) |
| GrepObject, GrepObjects | GrepObjects ✅ (done) |
| GrepPackage, GrepPackages | GrepPackages ✅ (done) |

### Add

| New Tool | Category | Priority |
|----------|----------|----------|
| GetMessages | Text | High |
| GitImport | Git | High |
| ActivatePackage | Batch | High |
| DiffVersions | Compare | High |
| CreateTable | DDIC | Medium |
| AddTableField | DDIC | Medium |
| CompareSource | Compare | Medium |
| RunUnitTestsPackage | Batch | Medium |
| MoveObject | CRUD | Medium |
| CloneObject | CRUD | Low |

---

## Priority Roadmap

### Sprint 1: Quick Wins (1-2 days)

1. **GetMessages** - SE91 message texts (2h)
2. **ActivatePackage** - Batch activation (3h)
3. **Rename GetDumps → ListDumps** (30min)
4. **Add "X" experimental group** (1h)

### Sprint 2: abapGit Completion (3-5 days)

1. **GitImport** - Deploy from ZIP (1d)
2. **GitStatus** - Modified object detection (4h)
3. **GitDiff** - Single object diff vs ZIP (4h)

### Sprint 3: DDIC Support (1 week)

1. **CreateTable** - Table creation with fields (1d)
2. **AddTableField** - Add field to existing table (4h)
3. **CreateDataElement** - Data element creation (4h)
4. **GetTexts/SetTexts universalization** (1d)

### Sprint 4: Compare/Diff (3-5 days)

1. **DiffVersions** - Active vs inactive (4h)
2. **CompareSource** - Two object diff (1d)
3. **DiffWithGit** - Object vs ZIP version (4h)

---

## Appendix: Full Tool Inventory After Changes

### Focused Mode (Target: ~60 tools)

**Core Operations** (12)
- GetSource, WriteSource, EditSource
- SearchObject, GrepObjects, GrepPackages
- SyntaxCheck, Activate, RunUnitTests, RunATCCheck
- LockObject, UnlockObject

**Data/Metadata** (8)
- GetTable, GetTableContents, RunQuery
- GetPackage, GetFunctionGroup
- GetCDSDependencies, GetMessages (NEW)
- GetTexts (unified, NEW)

**Code Intelligence** (4)
- FindDefinition, FindReferences
- PrettyPrint, GetInactiveObjects

**Code Analysis** (7)
- GetCallGraph, GetObjectStructure
- GetCallersOf, GetCalleesOf
- AnalyzeCallGraph, CompareCallGraphs, TraceExecution

**Diagnostics** (6)
- ListDumps (renamed), GetDump
- ListTraces, GetTrace
- GetSQLTraceState, ListSQLTraces

**File Operations** (2)
- ImportFromFile, ExportToFile

**System** (4)
- GetSystemInfo, GetInstalledComponents
- GetConnectionInfo, GetFeatures

**Git** (4)
- GitTypes, GitExport
- GitImport (NEW), GitStatus (NEW)

**Transport** (2)
- ListTransports, GetTransport

**Install** (3)
- InstallZADTVSP, InstallAbapGit, ListDependencies

**Batch** (2, NEW)
- ActivatePackage, ActivateAll

### Expert Mode Additions (+30)

- Legacy Get* tools (6)
- Legacy Write* tools (4)
- CreateObject, DeleteObject, RenameObject
- Transport write tools (3)
- DDIC tools (6, NEW)
- Compare tools (3, NEW)
- Debugger tools (10, in "X" group)
- AMDP tools (7, in "X" group)
- UI5 tools (7)
- ExecuteABAP

---

*Report generated as follow-up to tool introspection analysis*
