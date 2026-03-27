# ABAP ADT Discovery Guide

**Date:** 2025-12-01
**Purpose:** How to explore ADT internals in ABAP to understand and extend MCP capabilities

---

## Overview

This guide documents which ABAP packages, classes, function modules, and tables to explore when researching ADT capabilities. All objects listed were discovered via the connected SAP system.

---

## Part 1: Key ADT Packages

### Core ADT Framework

| Package | Description | Key Contents |
|---------|-------------|--------------|
| `SADT` | ADT Main Package (structure) | Parent of all ADT packages |
| `SADT_REST` | ADT REST Framework | Core REST infrastructure, SICF handlers |
| `SADT_CORE` | ADT Core | Central components |
| `SADT_CORE_UTILITIES` | ADT Core Utilities | BAdI registry, helper functions |
| `SADT_TOOLS_CORE` | ADT Tools Core | Source transformations, configurations |
| `SADT_UTILITIES` | ADT Utilities | Logging, CTS management, ATOM utilities |
| `SADT_COMPATIBILITY` | Compatibility Graph | Version negotiation between Eclipse/ABAP |

### Feature-Specific Packages

| Package | Description | Use Case |
|---------|-------------|----------|
| `SDP_ADT` | Data Preview | Table/CDS data queries |
| `SABP_UNIT_LEGACY_ADT` | ABAP Unit ADT | Unit test execution via ADT |
| `SEU_ADT` | Workbench ADT | Object editing, activation |
| `SCTS_ADT` | CTS ADT | Transport management |
| `SDDIC_ADT_COMMON` | DDIC ADT Common | CDS, where-used |
| `S_CODE_COMPLETION_ADT` | Code Completion | Autocomplete functionality |
| `SEO_ADT` | OO ADT | Class/interface handling |
| `SMESSAGE_ADT` | Message ADT | Message class handling |
| `SWB_ADT_BLUE` | Workbench "Blue" | Generic WB object handling |

---

## Part 2: Central Entry Points

### The Main Dispatcher

**Function Module:** `SADT_REST_RFC_ENDPOINT`
- **Package:** `SADT_REST`
- **Purpose:** Central entry point for ALL ADT REST requests from Eclipse
- **How to use:** Set breakpoint here to trace any ADT operation

```abap
" To trace ADT requests:
" 1. Set external breakpoint on SADT_REST_RFC_ENDPOINT
" 2. Perform action in Eclipse
" 3. Step through to see how requests are routed
```

### Request Routing BAdI

**Enhancement Spot:** `SADT_REST_RFC_APPLICATION`
- **Purpose:** Routes incoming URLs to handler classes
- **Implementation classes** register their URL patterns here

---

## Part 3: Key Classes by Functionality

### 3.1 REST Framework Classes

| Class | Description | Explore For |
|-------|-------------|-------------|
| `CL_ADT_REST_RESOURCE` | Base class for all ADT resources | How to implement REST endpoints |
| `CL_ADT_REST_ST_HANDLER` | Simple Transformation handler | XML serialization patterns |
| `CL_ADT_REST_REQUEST` | Request wrapper | How parameters are parsed |
| `CL_ADT_REST_RESPONSE` | Response wrapper | How responses are built |
| `CL_ADT_RES_APP_BASE` | Default REST application | URL routing logic |
| `CL_ADT_DISC_RES_APP_BASE` | Discovery application | Feature discovery |
| `CL_ADT_REST_URI_BUILDER` | URI building | How ADT URIs are constructed |

### 3.2 Data Preview Classes (Table Queries)

| Class | Description | Key Methods |
|-------|-------------|-------------|
| `CL_ADT_DATAPREVIEW_RES` | Data preview resource | GET/POST for SQL queries |
| `CL_ADT_DATAPREVIEW_RES_APP` | Data preview router | URL registration |
| `CL_ADT_DATAPREVIEW_UTIL` | Data preview utilities | Query execution logic |
| `CL_ADT_DP_OPEN_SQL_HANDLER` | Open SQL handler | WHERE clause processing |
| `CL_ADT_DP_CDS_RES` | CDS data preview | CDS view queries |

**Key Interface:** `IF_ADT_DATAPREVIEW_RES_CO` - Constants for data preview

### 3.3 Unit Testing Classes

| Class | Description | Key Methods |
|-------|-------------|-------------|
| `CL_AUNIT_ADT_RES_APP` | Unit test router | URL patterns for /abapunit/* |
| `CL_AUNIT_ADT_RES_TEST_RUNS` | Test run resource | POST to execute tests |
| `CL_AUNIT_ADT_RES_TEST_EVALUATN` | Test evaluation resource | GET results |
| `CL_AUNIT_ADT_RES_TEST_METADATA` | Test metadata | Test class discovery |
| `CL_AUNIT_ASSERT` | Assertion class | How assertions work |

**Package:** `SABP_UNIT_LEGACY_ADT` - Contains all ADT unit test resources

### 3.4 Transport Management Classes

| Class | Description | Key Methods |
|-------|-------------|-------------|
| `CL_ADT_CTS_MANAGEMENT` | CTS management | Transport request handling |
| `CL_ADT_CTS_REGISTRY` | CTS registry | Request tracking |
| `CL_ADT_CTS_TM_VIT_URI_MAPPER` | Transport URI mapper | URI to transport mapping |

### 3.5 Code Intelligence Classes

| Class | Description | Key Methods |
|-------|-------------|-------------|
| `CL_ADT_CDS_WHERE_USED_UTIL` | Where-used utility | Find references |
| `CL_ADT_ABSTR_GRAMMAR_COMP_PROV` | Grammar component provider | Code completion |

**Package:** `S_CODE_COMPLETION_ADT` - Code completion structures

---

## Part 4: Key Tables

### ADT Configuration Tables

| Table | Description | Contents |
|-------|-------------|----------|
| `SADT_SRVC_GRP` | ADT Service Group Registration | Registered ADT services |
| `SADT_SRVC_GRP_T` | Service Group Texts | Descriptions |
| `SADT_SRVC_GRP_U` | Registered URI Paths | URL patterns per service |
| `SADT_BG_RUNS` | Background Runs | Async operation tracking |
| `SADT_SRL_DATA` | Data Mappings | External/internal type mappings |
| `SADT_BADI_ACCESS` | BAdI Access Control | Protected mode settings |
| `SADT_ACT_STATUS` | Activation Status | WB object activation state |
| `SADT_OAUTH2SCOPE` | OAuth2 Scopes | Authorization scopes |

### Compatibility Tables

| Table | Description | Contents |
|-------|-------------|----------|
| `SADT_CT_NODE` | Compatibility Nodes | Feature nodes |
| `SADT_CT_NODET` | Node Texts | Node descriptions |
| `SADT_CT_EDGE` | Compatibility Edges | Feature dependencies |
| `SADT_CT_EDGET` | Edge Texts | Dependency descriptions |

---

## Part 5: Key Function Modules

| Function Module | Package | Purpose |
|-----------------|---------|---------|
| `SADT_REST_RFC_ENDPOINT` | SADT_REST | Main ADT dispatcher |
| `SADT_BG_RUN_RFC` | SADT_REST | Background run trigger |
| `SADT_CREATE_APPL_REST_RESOURCE` | SADT_REST | Create ADT resource |
| `SADT_CORE_INVALIDATE_INST_RFC` | SADT_CORE_UTILITIES | Invalidate BAdI cache |
| `SADT_CORE_INVALIDATE_REG_RFC` | SADT_CORE_UTILITIES | Invalidate registry |

---

## Part 6: Key Transformations (XML Serialization)

| Transformation | Package | Purpose |
|----------------|---------|---------|
| `SADT_ATOM` | SADT_REST | ATOM feed serialization |
| `SADT_EXCEPTION` | SADT_REST | Exception to XML |
| `SADT_BASIC_OBJECT` | SEU_ADT | Main object transformation |
| `SADT_ABAP_SOURCE_OBJECT` | SADT_TOOLS_CORE | Source code transformation |
| `SADT_COCO_PROPOSALS` | S_CODE_COMPLETION_ADT | Code completion proposals |
| `SADT_CHECK_RUN_OBJECTS` | SEU_ADT | Check run results |
| `ST_DATA_PREVIEW` | SDP_ADT | Data preview results |

---

## Part 7: Key Interfaces

### Core Interfaces

| Interface | Description | Implement For |
|-----------|-------------|---------------|
| `IF_ADT_REST_RFC_APPLICATION` | REST RFC Application | Custom ADT endpoints |
| `IF_ADT_REST_CONTENT_HANDLER` | Content Handler | Custom serialization |
| `IF_ADT_REST_REQUEST` | Request access | Reading request data |
| `IF_ADT_REST_RESPONSE` | Response access | Writing responses |
| `IF_ADT_DISCOVERY_PROVIDER` | Discovery provider | Registering capabilities |
| `IF_ADT_DISCOVERY_COLLECTION` | Discovery collection | Feature collections |

### Feature-Specific Interfaces

| Interface | Description |
|-----------|-------------|
| `IF_ADT_DATAPREVIEW_RES_CO` | Data preview constants |
| `IF_ADT_DP_AUTH_CHECK` | Data preview auth check |
| `IF_ADT_CTS_MANAGEMENT` | Transport management |
| `IF_ADT_CHECK_REPORTER` | Check result reporting |
| `IF_ADT_ATOM_UTILITY` | ATOM utility methods |

---

## Part 8: Discovery Commands

Use these search patterns with `SearchObject` to explore:

### Find All ADT Resource Classes
```
CL_ADT_RES_*
```

### Find All ADT Interfaces
```
IF_ADT_*
```

### Find Unit Test ADT Components
```
CL_AUNIT_ADT*
```

### Find Data Preview Components
```
*DATAPREVIEW*
```

### Find Transport/CTS Components
```
*CTS_ADT*
```

### Find Debugger Components
```
*DEBUG*ADT* or CL_ADT*DEBUG*
```

### Find ATC Components
```
*ATC* in package SATC*
```

---

## Part 9: Practical Exploration Workflow

### Step 1: Find the Relevant Package
```abap
" Use SearchObject to find objects by pattern
" Example: Find all unit test related ADT objects
SearchObject: CL_AUNIT_ADT*
```

### Step 2: Get Package Contents
```abap
" Use GetPackage to see all objects in a package
GetPackage: SABP_UNIT_LEGACY_ADT
```

### Step 3: Read Resource Class
```abap
" Use GetClass to read the implementation
GetClass: CL_AUNIT_ADT_RES_TEST_RUNS
```

### Step 4: Find URL Pattern
```abap
" Look for BAdI enhancement implementation
" Search for: ENHO/XH objects in the package
" These register URL patterns
```

### Step 5: Trace the Flow
```abap
" Set breakpoint on SADT_REST_RFC_ENDPOINT
" Trigger action from Eclipse
" Step through to understand the flow
```

---

## Part 10: URL Pattern to Handler Mapping

Based on discovered objects, here are the key mappings:

| URL Pattern | Handler Class | Package |
|-------------|---------------|---------|
| `/sap/bc/adt/discovery` | `CL_ADT_RES_DISCOVERY` | SADT_REST |
| `/sap/bc/adt/datapreview/*` | `CL_ADT_DATAPREVIEW_RES` | SDP_ADT |
| `/sap/bc/adt/abapunit/*` | `CL_AUNIT_ADT_RES_TEST_RUNS` | SABP_UNIT_LEGACY_ADT |
| `/sap/bc/adt/programs/*` | `CL_ADT_RES_PROGRAM*` | SEU_ADT |
| `/sap/bc/adt/oo/classes/*` | `CL_ADT_RES_CLASS*` | SEO_ADT |
| `/sap/bc/adt/cts/*` | `CL_ADT_CTS_*` | SCTS_ADT |
| `/sap/bc/adt/repository/*` | `CL_ADT_RES_REPOSITORY*` | SEU_ADT |

---

## Part 11: BAdI Enhancement Spots

Key BAdIs for extending ADT:

| Enhancement Spot | Purpose | Implementation Interface |
|------------------|---------|--------------------------|
| `SADT_REST_RFC_APPLICATION` | Register REST applications | `IF_ADT_REST_RFC_APPLICATION` |
| `SADT_DISCOVERY_PROVIDER` | Register discovery entries | `IF_ADT_DISCOVERY_PROVIDER` |
| `SADT_REST_AUTHORIZATION` | Custom authorization | `IF_ADT_REST_AUTHORIZATION` |
| `SADT_REST_RES_ACCESSIBILITY` | Resource visibility | `IF_ADT_REST_RES_ACCESSIBILITY` |
| `SADT_REST_RFC_TRACING` | Request tracing | `IF_ADT_RFC_TRACING` |

---

## Part 12: Quick Reference - MCP Extension Targets

### For Unit Testing
- **Package:** `SABP_UNIT_LEGACY_ADT`
- **Key Class:** `CL_AUNIT_ADT_RES_TEST_RUNS`
- **Endpoint:** `POST /sap/bc/adt/abapunit/testruns`
- **Transformation:** Look for `SADT_AUNIT*` or `ST_AUNIT*`

### For Data Query with SQL
- **Package:** `SDP_ADT`
- **Key Class:** `CL_ADT_DATAPREVIEW_RES`
- **Endpoint:** `POST /sap/bc/adt/datapreview/ddic`
- **Interface:** `IF_ADT_DATAPREVIEW_RES_CO`

### For Code Completion
- **Package:** `S_CODE_COMPLETION_ADT`
- **Structures:** `SADT_COCO_PROPOSAL`, `SADT_COCO_PROPOSALS`
- **Transformation:** `SADT_COCO_PROPOSALS`

### For Transport Management
- **Package:** `SCTS_ADT`
- **Key Class:** `CL_ADT_CTS_MANAGEMENT`
- **Interface:** `IF_ADT_CTS_MANAGEMENT`

### For Where-Used/References
- **Package:** `SDDIC_ADT_COMMON`
- **Key Class:** `CL_ADT_CDS_WHERE_USED_UTIL`
- **Interface:** `IF_ADT_CDS_WHERE_USED_UTIL`

---

## Summary: Recommended Exploration Order

1. **Start with `SADT_REST`** - Understand the framework
2. **Explore `SDP_ADT`** - Data preview (table queries)
3. **Examine `SABP_UNIT_LEGACY_ADT`** - Unit testing
4. **Study `SEU_ADT`** - Workbench operations
5. **Review `SCTS_ADT`** - Transport management
6. **Investigate `S_CODE_COMPLETION_ADT`** - Code intelligence

Each package contains:
- Resource classes (`CL_*_RES_*`) - Handle HTTP methods
- Transformations (`SADT_*`, `ST_*`) - XML serialization
- Interfaces (`IF_*`) - Type definitions and constants
- BAdI implementations - URL registration

Use `GetClass` to read implementations and understand the patterns!
