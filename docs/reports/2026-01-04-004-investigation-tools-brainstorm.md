# VSP Investigation Tools Brainstorm

**Date:** 2026-01-04
**Report ID:** 004
**Subject:** Tools for exploring SAP standard, BADIs, BRF+, CDS, SPRO, Fiori/OData
**Follows:** 2026-01-04-002-tool-universalization-strategy.md

---

## Executive Summary

This report brainstorms investigation tools that would help AI agents and developers explore:
1. **BRF+ Business Rules** - Decision tables, expressions, formulas
2. **CDS Dependencies** - Forward and reverse dependency analysis
3. **Code Dependencies** - Where-used, cross-references
4. **Documentation** - SPRO nodes, F1 help, code comments
5. **BADIs/Enhancements** - Enhancement spots, BAdI implementations
6. **Fiori/OData** - Service metadata, annotations, app configurations

---

## 1. BRF+ Investigation Tools

### Current State
- No BRF+ tools currently exist in VSP
- BRF+ uses ADT endpoints under `/sap/bc/adt/brfplus/`

### Proposed Tools

| Tool | Description | ADT Endpoint | Priority |
|------|-------------|--------------|----------|
| **ListBRFApplications** | List all BRF+ applications | `/sap/bc/adt/brfplus/applications` | High |
| **GetBRFApplication** | Get application details (functions, data objects) | `/sap/bc/adt/brfplus/applications/{id}` | High |
| **GetBRFFunction** | Get function with decision table/expression | `/sap/bc/adt/brfplus/functions/{id}` | High |
| **GetBRFDecisionTable** | Extract decision table rules | `/sap/bc/adt/brfplus/decisiontables/{id}` | Medium |
| **GetBRFExpression** | Get expression formula details | `/sap/bc/adt/brfplus/expressions/{id}` | Medium |
| **SimulateBRF** | Execute BRF+ function with test data | Requires runtime call | Low |

### Use Cases
```
Agent: "Analyze the pricing logic in BRF+ application ZSD_PRICING"
→ ListBRFApplications(query="ZSD_PRICING")
→ GetBRFApplication(id="...")
→ GetBRFFunction(id="DETERMINE_PRICE")
→ GetBRFDecisionTable(id="DT_PRICE_CONDITIONS")
→ "The pricing logic uses these decision table rules: [...]"
```

---

## 2. CDS Dependency Analysis

### Current State
- **GetCDSDependencies** - Forward dependencies (what this CDS reads FROM) ✅
- Missing: Reverse dependencies (who reads FROM this CDS)

### Proposed Tools

| Tool | Description | ADT Endpoint | Priority |
|------|-------------|--------------|----------|
| **GetCDSWhereUsed** | Find all CDS views that use this view | `/sap/bc/adt/cds/usagereferences/` | High |
| **GetCDSAnnotations** | Extract CDS annotations | `/sap/bc/adt/ddic/ddl/sources/{name}/annotations` | High |
| **GetCDSAssociations** | List CDS associations and their targets | Parse from DDLS source | Medium |
| **AnalyzeCDSGraph** | Full dependency graph (up and down) | Combine Forward + WhereUsed | Medium |
| **GetCDSMetadata** | Entity metadata (fields, types, keys) | `/sap/bc/adt/ddic/ddl/sources/{name}/metadata` | High |

### Existing Enhancement
```
# Current GetCDSDependencies
GetCDSDependencies(ddls_name="I_SALESORDER")
→ Returns: Base tables (VBAK, VBAP) and base views

# Proposed GetCDSWhereUsed
GetCDSWhereUsed(ddls_name="I_SALESORDER")
→ Returns: All views built ON TOP of this one
  - C_SALESORDER (Consumption view)
  - Z_CUSTOM_SALESORDER (Custom extension)
```

---

## 3. Code Cross-References (Where-Used)

### Current State
- **FindReferences** - Find symbol references (position-based) ✅
- **GetCallGraph** - Call hierarchy ✅
- Missing: Object-level where-used list

### Proposed Tools

| Tool | Description | ADT Endpoint | Priority |
|------|-------------|--------------|----------|
| **GetWhereUsed** | Object-level where-used list | `/sap/bc/adt/repository/informationsystem/usedbylist` | High |
| **GetCrossReferences** | Full CROSS/WBCROSSGT analysis | `/sap/bc/adt/repository/informationsystem/references` | Medium |
| **FindTableUsage** | Where is this table used? | WBCROSSGT query | High |
| **FindFunctionUsage** | Where is this FM called? | WBCROSSGT query | Medium |
| **FindClassUsage** | Where is this class instantiated? | WBCROSSGT query | Medium |

### Design Notes
```
GetWhereUsed(object_type="TABL", object_name="VBAK")
→ Returns:
  - Programs using SELECT FROM VBAK
  - Reports with VBAK in DATA declarations
  - Classes reading/writing VBAK
  - CDS views based on VBAK
```

---

## 4. Documentation Access

### SPRO Customizing Documentation

| Tool | Description | Source | Priority |
|------|-------------|--------|----------|
| **GetSPRONode** | Get SPRO node details and documentation | SPRO API / IMG | Medium |
| **GetSPRODocumentation** | F1 help text for customizing activity | DOKL table | Medium |
| **ListSPROPath** | Navigate SPRO hierarchy | IMG structure | Low |

### Code Documentation

| Tool | Description | Source | Priority |
|------|-------------|--------|----------|
| **GetObjectDocumentation** | ABAP Doc for class/method | ADT documentation endpoint | High |
| **GetFunctionDocumentation** | FM short text and long text | TFDIR/TFTIT | Medium |
| **GetDataElementHelp** | F1 help for data elements | DOKTL/DOKHL | Medium |

### Design Notes
```abap
" ABAP Doc is stored in code comments with special format
"! @parameter iv_input | Input parameter description
"! @return | Return value description
"! @raising cx_error | Exception description
```

ADT Endpoint: `/sap/bc/adt/documentation/{object_uri}`

---

## 5. BADIs and Enhancement Framework

### Enhancement Points Discovery

| Tool | Description | ADT Endpoint | Priority |
|------|-------------|--------------|----------|
| **ListEnhancementSpots** | Find enhancement spots in package | `/sap/bc/adt/enhancements/spots` | High |
| **GetEnhancementSpot** | Get spot details with BADIs | `/sap/bc/adt/enhancements/spots/{name}` | High |
| **ListBADIImplementations** | Find all implementations of a BADI | `/sap/bc/adt/enhancements/badis/{name}/implementations` | High |
| **GetBADIImplementation** | Get implementation code | `/sap/bc/adt/enhancements/implementations/{name}` | High |
| **FindEnhancementOptions** | What can be enhanced in this object? | Scan for enhancement points | Medium |

### Enhancement Analysis

| Tool | Description | Priority |
|------|-------------|----------|
| **AnalyzeEnhancements** | List all enhancements in an object | High |
| **GetImplicitEnhancements** | Find implicit enhancement points | Medium |
| **GetExplicitEnhancements** | Find explicit enhancement points | Medium |

### Use Cases
```
Agent: "Find all BADIs available for sales order processing"
→ SearchObject(query="BADI*SD*ORDER*")
→ ListEnhancementSpots(package="SD")
→ GetEnhancementSpot(name="ES_SD_ORDER_CREATE")
→ "Found 5 BADIs: BADI_SD_ORDER_HEAD, BADI_SD_ORDER_ITEM, ..."
→ ListBADIImplementations(badi="BADI_SD_ORDER_HEAD")
→ "Currently implemented by: ZCL_SD_ORDER_CHECK (active)"
```

---

## 6. Fiori/OData Investigation

### OData Service Metadata

| Tool | Description | Endpoint | Priority |
|------|-------------|----------|----------|
| **ListODataServices** | List published OData services | `/sap/bc/adt/businessservices/odatav4` | High |
| **GetODataMetadata** | Get $metadata for a service | Service URL + `$metadata` | High |
| **GetODataAnnotations** | Get CDS annotations for UI | Annotation file | Medium |
| **GetServiceBinding** | Get SRVB details (RAP) | `/sap/bc/adt/bo/servicebindings/{name}` | High |

### Fiori App Investigation

| Tool | Description | Source | Priority |
|------|-------------|--------|----------|
| **ListFioriApps** | List Fiori apps in launchpad | LPD_CUST / Fiori catalog | Medium |
| **GetFioriAppDetails** | App ID, service bindings, annotations | Launchpad config | Medium |
| **GetFioriTile** | Tile configuration and target mapping | Catalog API | Low |

### Use Cases
```
Agent: "What OData service backs the Fiori app for Purchase Orders?"
→ ListFioriApps(query="*Purchase*Order*")
→ GetFioriAppDetails(app_id="F1234")
→ GetODataMetadata(service="API_PURCHASEORDER_PROCESS_SRV")
→ GetODataAnnotations(service="...")
→ "The app uses service API_PURCHASEORDER_PROCESS_SRV with
    entities: A_PurchaseOrder, A_PurchaseOrderItem..."
```

---

## 7. Implementation Priority Roadmap

### Phase 1: High-Impact Read Tools (1-2 weeks)

| Tool | Effort | Impact |
|------|--------|--------|
| GetWhereUsed | 4h | Very High |
| GetCDSWhereUsed | 3h | High |
| GetCDSAnnotations | 2h | High |
| ListEnhancementSpots | 3h | High |
| GetBADIImplementation | 3h | High |
| GetODataMetadata | 2h | High |

### Phase 2: BRF+ Tools (1 week)

| Tool | Effort | Impact |
|------|--------|--------|
| ListBRFApplications | 3h | Medium |
| GetBRFApplication | 4h | Medium |
| GetBRFFunction | 4h | Medium |
| GetBRFDecisionTable | 4h | Medium |

### Phase 3: Documentation Tools (1 week)

| Tool | Effort | Impact |
|------|--------|--------|
| GetObjectDocumentation | 3h | Medium |
| GetDataElementHelp | 3h | Medium |
| GetSPRODocumentation | 4h | Low |

### Phase 4: Fiori Deep Dive (1 week)

| Tool | Effort | Impact |
|------|--------|--------|
| ListODataServices | 2h | High |
| GetServiceBinding | 2h | High |
| ListFioriApps | 4h | Medium |
| GetFioriAppDetails | 4h | Medium |

---

## 8. ADT Endpoint Discovery

### Known Endpoints to Investigate

```
# BRF+ (unverified)
/sap/bc/adt/brfplus/applications
/sap/bc/adt/brfplus/functions
/sap/bc/adt/brfplus/decisiontables

# Enhancement Framework
/sap/bc/adt/enhancements/spots
/sap/bc/adt/enhancements/badis
/sap/bc/adt/enhancements/implementations

# Documentation
/sap/bc/adt/documentation/{object_uri}
/sap/bc/adt/vit/docu/abapDoc

# OData/RAP
/sap/bc/adt/businessservices/odatav4
/sap/bc/adt/bo/servicebindings

# Usage Analysis
/sap/bc/adt/repository/informationsystem/usedbylist
/sap/bc/adt/repository/informationsystem/references
```

### Discovery Method
```bash
# Use ADT discovery document
curl $SAP_URL/sap/bc/adt/discovery -u $USER:$PASS

# Or explore via browser
# /sap/bc/adt/?sap-client=001
```

---

## 9. Agent Workflow Examples

### Workflow: "Understand Standard Logic"

```
User: "How does SAP calculate the net price in SD?"

Agent:
1. SearchObject("*PRICING*") → Find pricing-related objects
2. GetWhereUsed("KONV") → See where condition table is used
3. FindReferences(pricing_routine) → Find calculation routines
4. GetCDSDependencies("I_PRICINGCONDITION") → Understand data model
5. GetBADIImplementation("BADI_SD_PRICING") → Find custom exits
6. GetBRFDecisionTable(if BRF+ used) → Extract business rules

Result: Comprehensive explanation with code references
```

### Workflow: "Prepare for Enhancement"

```
User: "I need to add custom validation to Purchase Order creation"

Agent:
1. ListEnhancementSpots(package="MM") → Find PO-related spots
2. GetEnhancementSpot("ES_PO_CREATE") → Get available BADIs
3. ListBADIImplementations("BADI_PO_VALIDATION") → Check existing
4. GetBADIImplementation("ZCL_PO_CUSTOM_CHECK") → Review current code
5. GetODataMetadata("API_PURCHASEORDER") → If Fiori app involved

Result: Clear picture of extension options
```

### Workflow: "Analyze Fiori App"

```
User: "Explain the 'Manage Sales Orders' Fiori app"

Agent:
1. ListFioriApps("Manage Sales Orders") → Get app ID
2. GetFioriAppDetails(app_id) → Get tile, service, BSP
3. GetODataMetadata(service_name) → Entity model
4. GetCDSAnnotations(base_cds) → UI annotations
5. UI5GetApp(bsp_name) → Frontend code
6. GetSource(CLAS, handler_class) → Backend logic

Result: Full technical documentation of the app
```

---

## 10. Quick Wins for Today

Based on existing ADT knowledge and patterns:

1. **GetWhereUsed** - Similar pattern to FindReferences, just different endpoint
2. **GetCDSWhereUsed** - Extend existing CDS tooling
3. **GetODataMetadata** - Simple HTTP GET + parse

These three tools would immediately enable powerful investigation workflows.

---

## Appendix: Existing Related Tools

| Tool | What It Does | Gap |
|------|--------------|-----|
| FindReferences | Symbol-level references | Needs object-level |
| GetCDSDependencies | Forward deps only | Needs reverse |
| GetCallGraph | Function call hierarchy | Not for data/tables |
| UI5GetApp | Frontend code | No backend OData link |
| SearchObject | Find by name | No semantic search |

---

*Report generated for investigation tool planning*
