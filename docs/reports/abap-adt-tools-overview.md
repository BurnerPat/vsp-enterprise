# ABAP ADT Tools Overview

## Available Tools

| Tool | Description | Use Case |
|------|-------------|----------|
| `GetProgram` | Retrieve ABAP program source code | Z-code & standard programs |
| `GetClass` | Retrieve ABAP class source code | Z-code & standard classes |
| `GetInterface` | Retrieve ABAP interface source code | Z-code & standard interfaces |
| `GetFunction` | Retrieve Function Module source code | Requires function group name |
| `GetFunctionGroup` | Retrieve Function Group source code | Get all FMs in a group |
| `GetInclude` | Retrieve ABAP Include source code | Reusable code includes |
| `GetTable` | Retrieve ABAP table structure/definition | Table metadata (fields, types) |
| `GetTableContents` | Select data from a table | **Data retrieval** (max 100 rows default) |
| `GetStructure` | Retrieve ABAP structure definition | Data type definitions |
| `GetTypeInfo` | Retrieve ABAP type information | Domain, data element info |
| `GetPackage` | Retrieve ABAP package details | Package contents/hierarchy |
| `GetTransaction` | Retrieve transaction details | T-code to program mapping |
| `SearchObject` | Quick search for ABAP objects | Find objects by name pattern (supports `*` wildcard) |

---

## What's Missing

| Missing Capability | Impact |
|--------------------|--------|
| **CDS View source code** | Cannot retrieve DDL source |
| **CDS dependency graph** | No way to trace CDS associations/compositions |
| **Select from CDS views** | `GetTableContents` works on tables only, not CDS |
| **BADI/Enhancement implementations** | Cannot probe custom enhancements |
| **Where-used list** | No reverse dependency lookup |
| **Transport requests** | Cannot see change history |
| **Message classes** | Cannot retrieve message texts |
| **Domain values** | Fixed values not directly accessible |

---

## Cheat Sheet by Use Case

### Z-Code Probing (Custom Development)

```
# Find all Z-programs
SearchObject(query="Z*")

# Find custom classes
SearchObject(query="ZCL_*")
SearchObject(query="YCL_*")

# Get specific program
GetProgram(program_name="ZREPORT_NAME")

# Get custom class
GetClass(class_name="ZCL_MY_CLASS")

# Get custom table structure + data
GetTable(table_name="ZTABLE")
GetTableContents(table_name="ZTABLE", max_rows=100)

# Find custom function modules
SearchObject(query="Z_*")
GetFunction(function_name="Z_MY_FM", function_group="ZFG_GROUP")
```

### Standard SAP Code Analysis

```
# Search standard objects
SearchObject(query="CL_ABAP_*")

# Get standard class
GetClass(class_name="CL_SALV_TABLE")

# Get standard interface
GetInterface(interface_name="IF_SALV_TABLE")

# Check standard table structure
GetTable(table_name="MARA")
GetTable(table_name="VBAK")

# Transaction details
GetTransaction(transaction_name="SE38")
GetTransaction(transaction_name="MM01")
```

### Data Analysis

```
# Get table data (up to 100 rows by default)
GetTableContents(table_name="T001", max_rows=50)

# Check config tables
GetTableContents(table_name="T000")  # Clients
GetTableContents(table_name="T001")  # Company codes
```

### Package/Structure Exploration

```
# Explore package contents
GetPackage(package_name="ZPACKAGE")
GetPackage(package_name="SLIS")  # Standard ALV

# Get data types
GetStructure(structure_name="BAPIRET2")
GetTypeInfo(type_name="MATNR")
```

---

## Search Patterns

| Pattern | Finds |
|---------|-------|
| `Z*` | All Z-objects |
| `ZCL_*` | All Z-classes |
| `*SALES*` | Objects containing "SALES" |
| `CL_ABAP_*` | Standard ABAP classes |
| `BAPI_*` | BAPI function modules |

---

## Limitations Summary

| Can Do | Cannot Do |
|--------|-----------|
| Read ABAP source (programs, classes, FMs) | Read CDS DDL source |
| Search objects by pattern | Get CDS dependencies |
| Get table definitions | Query CDS views |
| Select from transparent tables | Access BADIs/enhancements |
| Get type/structure info | Where-used analysis |
| Explore packages | Transport history |

---

## Quick Reference Card

```
┌─────────────────────────────────────────────────────────────┐
│  ABAP ADT TOOLS - QUICK REFERENCE                          │
├─────────────────────────────────────────────────────────────┤
│  SEARCH:     SearchObject(query="pattern*")                │
│  PROGRAM:    GetProgram(program_name="...")                │
│  CLASS:      GetClass(class_name="...")                    │
│  INTERFACE:  GetInterface(interface_name="...")            │
│  FM:         GetFunction(function_name, function_group)    │
│  TABLE DEF:  GetTable(table_name="...")                    │
│  TABLE DATA: GetTableContents(table_name, max_rows=100)    │
│  STRUCTURE:  GetStructure(structure_name="...")            │
│  TYPE:       GetTypeInfo(type_name="...")                  │
│  PACKAGE:    GetPackage(package_name="...")                │
│  TCODE:      GetTransaction(transaction_name="...")        │
│  INCLUDE:    GetInclude(include_name="...")                │
│  FN GROUP:   GetFunctionGroup(function_group="...")        │
└─────────────────────────────────────────────────────────────┘
```
