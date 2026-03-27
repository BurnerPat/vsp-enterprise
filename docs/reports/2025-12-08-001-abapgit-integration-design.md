# abapGit Integration Design for vsp

**Date:** 2025-12-08
**Report ID:** 001
**Subject:** Optional abapGit tools for ADT-MCP (ZIP-based deployment)
**Status:** Design Draft

---

## Executive Summary

This document outlines the design for integrating abapGit's serialization/deserialization API into vsp as **optional tools**. The core idea is to create a thin REST service in SAP that accepts abapGit-compatible ZIP files and uses abapGit's native deserializer to import objects—giving us the power to create *any* object type that abapGit supports (150+ types).

## Problem Statement

Current vsp tools support a limited set of object types via ADT:
- Programs (PROG)
- Classes (CLAS) with includes
- Interfaces (INTF)
- Function Groups/Modules (FUGR, FUNC)
- CDS Views (DDLS)
- Behavior Definitions (BDEF)
- Service Definitions/Bindings (SRVD, SRVB)

However, many object types are **not exposed via ADT REST API**:
- Data Elements (DTEL)
- Domains (DOMA)
- Table Types (TTYP)
- Structures (TABL/INTTAB)
- Database Tables (TABL)
- Lock Objects (ENQU)
- Number Ranges (NROB)
- Authorization Objects (SUSO)
- Message Classes (MSAG) - partial
- Enhancement Implementations (ENHO)
- And 140+ more types...

## Solution: Leverage abapGit's API

abapGit already has mature serializers for 150+ object types. By creating a REST endpoint that accepts abapGit ZIP format and calls the native deserializer, we get:

1. **Full object support** - Everything abapGit supports
2. **Battle-tested code** - abapGit handles edge cases
3. **Dependency ordering** - DDIC before ABAP, proper activation order
4. **Conflict detection** - Built-in checks for existing objects
5. **Translation support** - LXE integration for i18n

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                          vsp (Go)                               │
├─────────────────────────────────────────────────────────────────┤
│  GitDeploy          │ Upload ZIP, get deployment status         │
│  GitValidate        │ Dry-run validation (no actual import)     │
│  GitSupportedTypes  │ List supported object types               │
│  GitSerialize       │ Export package to ZIP (future)            │
└────────────┬────────────────────────────────────────────────────┘
             │ HTTP POST (multipart or BASE64 JSON)
             ▼
┌─────────────────────────────────────────────────────────────────┐
│           SAP ICF Service: /sap/bc/adt/zgit/                    │
├─────────────────────────────────────────────────────────────────┤
│  ZCL_ADT_GIT_DEPLOY (ICF Handler)                               │
│  ├─ POST /deploy     → Accept ZIP, return log                   │
│  ├─ POST /validate   → Dry-run checks                           │
│  └─ GET  /types      → List supported types                     │
└────────────┬────────────────────────────────────────────────────┘
             │ Uses
             ▼
┌─────────────────────────────────────────────────────────────────┐
│           abapGit Native Classes                                │
├─────────────────────────────────────────────────────────────────┤
│  ZCL_ABAPGIT_OBJECTS=>deserialize()                             │
│  ZCL_ABAPGIT_OBJECTS=>serialize()                               │
│  ZCL_ABAPGIT_OBJECTS=>supported_list()                          │
│  ZCL_ABAPGIT_OBJECTS=>deserialize_checks()                      │
└─────────────────────────────────────────────────────────────────┘
```

## API Design

### Tool Summary

| Tool | Mode | Description |
|------|------|-------------|
| `GitDeploy` | Focused | Deploy ZIP to SAP (create any object type) |
| `GitValidate` | Focused | Dry-run validation without import |
| `GitSupportedTypes` | Focused | List 156+ supported object types |

### 1. GitDeploy Tool

**Purpose:** Deploy abapGit-compatible ZIP to SAP system

**Input:**
```json
{
  "zip_data": "BASE64_ENCODED_ZIP_CONTENT",
  "package": "$ZTEST",
  "transport": "A4HK900123",
  "overwrite": "ask|yes|no",
  "activate": true,
  "dry_run": false
}
```

Or multipart with binary ZIP file.

**Output:**
```json
{
  "success": true,
  "objects_processed": 15,
  "objects_created": 10,
  "objects_updated": 5,
  "objects_skipped": 0,
  "activation_errors": 0,
  "log": [
    {"level": "info", "message": ">>> Deserializing 15 objects"},
    {"level": "success", "object": "CLAS/ZCL_TEST", "message": "Object imported"},
    {"level": "warning", "object": "DTEL/ZTEST", "message": "Already exists, overwritten"},
    {"level": "error", "object": "TABL/ZFAIL", "message": "Syntax error: field X not found"}
  ],
  "conflicts": [
    {"object": "PROG/ZEXIST", "action": "overwrite", "reason": "Object exists in different package"}
  ]
}
```

### 2. GitValidate Tool

**Purpose:** Validate ZIP without importing (dry-run)

**Input:**
```json
{
  "zip_data": "BASE64_ENCODED_ZIP_CONTENT",
  "package": "$ZTEST"
}
```

**Output:**
```json
{
  "valid": true,
  "objects_found": 15,
  "supported": 14,
  "unsupported": [
    {"type": "XINX", "name": "ZTEST_IDX", "reason": "Extension index not supported"}
  ],
  "conflicts": [
    {"type": "CLAS", "name": "ZCL_EXIST", "reason": "Object exists in package $OTHER"}
  ],
  "warnings": [
    "Package $ZTEST does not exist, will be created"
  ]
}
```

### 3. GitSupportedTypes Tool

**Purpose:** List all object types supported by abapGit

**Output:**
```json
{
  "types": ["CLAS", "DTEL", "DOMA", "ENHO", "FUGR", "FUNC", "INTF", "MSAG", "PROG", "TABL", ...],
  "count": 156
}
```

## ABAP Implementation

### ICF Handler Class

```abap
CLASS zcl_adt_git_deploy DEFINITION
  PUBLIC
  CREATE PUBLIC.

  PUBLIC SECTION.
    INTERFACES if_http_extension.

  PRIVATE SECTION.
    TYPES:
      BEGIN OF ty_deploy_request,
        zip_data  TYPE string,      " BASE64 encoded
        package   TYPE devclass,
        transport TYPE trkorr,
        overwrite TYPE string,      " ask|yes|no
        activate  TYPE abap_bool,
        dry_run   TYPE abap_bool,
      END OF ty_deploy_request.

    TYPES:
      BEGIN OF ty_log_entry,
        level   TYPE string,
        object  TYPE string,
        message TYPE string,
      END OF ty_log_entry,
      ty_log_entries TYPE STANDARD TABLE OF ty_log_entry WITH DEFAULT KEY.

    METHODS:
      handle_deploy
        IMPORTING
          io_request  TYPE REF TO if_http_request
          io_response TYPE REF TO if_http_response
        RAISING
          cx_static_check,

      handle_validate
        IMPORTING
          io_request  TYPE REF TO if_http_request
          io_response TYPE REF TO if_http_response
        RAISING
          cx_static_check,

      handle_types
        IMPORTING
          io_response TYPE REF TO if_http_response,

      extract_files_from_zip
        IMPORTING
          iv_zip_xstring TYPE xstring
        RETURNING
          VALUE(rt_files) TYPE zif_abapgit_git_definitions=>ty_files_tt
        RAISING
          zcx_abapgit_exception,

      create_virtual_repo
        IMPORTING
          it_files   TYPE zif_abapgit_git_definitions=>ty_files_tt
          iv_package TYPE devclass
        RETURNING
          VALUE(ri_repo) TYPE REF TO zif_abapgit_repo
        RAISING
          zcx_abapgit_exception.
ENDCLASS.
```

### Virtual Repository (Key Component)

The trickiest part is creating a "virtual repository" that implements `zif_abapgit_repo` but works purely in-memory from the uploaded ZIP:

```abap
CLASS zcl_virtual_repo DEFINITION
  CREATE PUBLIC.

  PUBLIC SECTION.
    INTERFACES zif_abapgit_repo.

    METHODS constructor
      IMPORTING
        it_files   TYPE zif_abapgit_git_definitions=>ty_files_tt
        iv_package TYPE devclass
        is_dot     TYPE zif_abapgit_dot_abapgit=>ty_dot_abapgit OPTIONAL.

  PRIVATE SECTION.
    DATA: mt_files   TYPE zif_abapgit_git_definitions=>ty_files_tt,
          mv_package TYPE devclass,
          mo_dot     TYPE REF TO zcl_abapgit_dot_abapgit.
ENDCLASS.

CLASS zcl_virtual_repo IMPLEMENTATION.

  METHOD constructor.
    mt_files = it_files.
    mv_package = iv_package.

    " Create default .abapgit.xml if not in ZIP
    IF is_dot IS NOT INITIAL.
      mo_dot = zcl_abapgit_dot_abapgit=>build( is_dot ).
    ELSE.
      mo_dot = zcl_abapgit_dot_abapgit=>build_default( ).
    ENDIF.
  ENDMETHOD.

  METHOD zif_abapgit_repo~get_files_remote.
    rt_files = mt_files.
  ENDMETHOD.

  METHOD zif_abapgit_repo~get_package.
    rv_package = mv_package.
  ENDMETHOD.

  METHOD zif_abapgit_repo~get_dot_abapgit.
    ri_dot = mo_dot.
  ENDMETHOD.

  " ... implement other interface methods ...

ENDCLASS.
```

## ZIP Format

abapGit uses a standard structure:

```
myrepo.zip
├── .abapgit.xml              # Optional: repo settings
├── src/
│   ├── zcl_myclass.clas.abap
│   ├── zcl_myclass.clas.xml
│   ├── zmy_program.prog.abap
│   ├── zmy_program.prog.xml
│   ├── zmy_dtel.dtel.xml
│   ├── zmy_table.tabl.xml
│   └── ...
```

File naming convention:
- `{name}.{type}.abap` - Source code
- `{name}.{type}.xml` - Metadata/properties
- `{name}.{type}.{subtype}.abap` - Class includes (testclasses, locals_def, etc.)

## Implementation Phases

### Phase 1: Read-Only Exploration
- [ ] `GitSupportedTypes` - List supported types
- [ ] `GitValidate` - Validate ZIP structure and check conflicts

### Phase 2: Deployment
- [ ] `GitDeploy` - Full deployment with options
- [ ] Support for BASE64 JSON input
- [ ] Support for binary stream input

### Phase 3: Export
- [ ] `GitSerialize` - Export package to ZIP (using existing serializer test code)

### Phase 4: Advanced
- [ ] Incremental sync (compare with existing)
- [ ] Conflict resolution dialogs
- [ ] Background job support

## Deployment Options

### Option A: SICF Service (Recommended)
Create ICF service at `/sap/bc/adt/zgit/`:
- Pros: Fits ADT pattern, reuses ADT auth
- Cons: Requires manual SICF setup

### Option B: OData Service
Create OData service `ZABAPGIT_DEPLOY_SRV`:
- Pros: Standard SAP Gateway approach
- Cons: More complex, OData overhead

### Option C: RFC Function Module
Create RFC `Z_ABAPGIT_DEPLOY`:
- Pros: Simpler, no HTTP handling
- Cons: Requires RFC destination in vsp

**Recommendation:** Option A (SICF) as it aligns with ADT patterns.

## vsp Tool Configuration

Tools will be in a new **git** tool group (`G`), all in **Focused mode** (core capabilities):

```go
// internal/mcp/server.go

// Git tools (optional, requires abapGit + service)
{Name: "GitDeploy", Fn: s.handleGitDeploy, Group: "G", Mode: ModeFocused},
{Name: "GitValidate", Fn: s.handleGitValidate, Group: "G", Mode: ModeFocused},
{Name: "GitSupportedTypes", Fn: s.handleGitSupportedTypes, Group: "G", Mode: ModeFocused},
```

Disable with: `--disabled-groups=G` if abapGit service not available.

## Error Handling

```json
{
  "error": "DEPLOYMENT_FAILED",
  "message": "3 objects failed to import",
  "details": [
    {"object": "TABL/ZTABLE", "error": "Field ZFIELD not found in domain"},
    {"object": "CLAS/ZCL_X", "error": "Syntax error at line 42"},
    {"object": "PROG/ZTEST", "error": "Object locked by user ADMIN"}
  ],
  "partial_success": true,
  "objects_imported": 12,
  "objects_failed": 3
}
```

## Security Considerations

1. **Package restrictions** - Use existing `SAP_ALLOWED_PACKAGES` safety
2. **Transport enforcement** - Respect system transport requirements
3. **Object overwrites** - Require explicit confirmation
4. **Authorization** - Use standard S_DEVELOP checks

## Dependencies

**Required in SAP:**
- abapGit installed (standalone or developer version)
- ICF service configured
- Appropriate authorizations

**Optional:**
- Transport request for non-local packages

## Open Questions

1. **Service detection** - How to detect if abapGit service is available?
2. **Version compatibility** - Minimum abapGit version required?
3. **Partial imports** - Continue on error or rollback?
4. **Large ZIPs** - Chunked upload for big packages?

## Next Steps

1. Create minimal ICF service in SAP (`ZCL_ADT_GIT_DEPLOY`)
2. Implement virtual repository class
3. Test with simple ZIP (1-2 objects)
4. Add vsp tool handlers
5. Expand to full feature set

## Related Documents

- `ZGIT_DEV_TEST_SERIALIZER_V2` - Existing serialization test program
- `ZCL_ABAPGIT_OBJECTS` - Core serializer/deserializer
- `ZIF_ABAPGIT_REPO` - Repository interface to implement
