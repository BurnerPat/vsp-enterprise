# RAP OData Service Implementation - Lessons Learned

**Date:** 2025-12-08
**Report ID:** 003
**Subject:** Unmanaged RAP OData Service for abapGit Integration
**Status:** Working

---

## Overview

Successfully created a RAP OData V4 service (`ZADT_GIT_DEPLOY`) that can be deployed and managed entirely via ADT/MCP. This enables future abapGit integration without requiring SAP GUI or manual SEGW projects.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    OData V4 Service                         │
│              ZADT_GIT_DEPLOY_O4 (SRVB)                     │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                  Service Definition                         │
│                   ZADT_GIT_DEPLOY (SRVD)                   │
│              expose ZADT_R_GIT_SERVICE as GitService        │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                 Behavior Definition                         │
│               ZADT_R_GIT_SERVICE (BDEF)                    │
│     unmanaged implementation in class ZCL_ZADT_GIT_SERVICE │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                  CDS Root Entity                            │
│               ZADT_R_GIT_SERVICE (DDLS)                    │
│            Singleton pattern (ServiceId='SERVICE')          │
└─────────────────────────────────────────────────────────────┘
```

## Key Lessons Learned

### 1. Handler Class Naming Convention

For unmanaged RAP, the local handler class name must be:
```
lhc_<alias>
```
Where `<alias>` is the entity alias from the BDEF, matching the **exact case**.

**Example:**
```abap
" BDEF:
define behavior for ZADT_R_GIT_SERVICE alias GitService

" Handler class (in locals_imp):
CLASS lhc_GitService DEFINITION INHERITING FROM cl_abap_behavior_handler.
```

**Wrong patterns that don't work:**
- `lhc_gitservice` (lowercase) - may work on some systems
- `lhc_zadt_r_git_service` (entity name instead of alias)
- `lhc_GITSERVICE` (uppercase)

### 2. Instance Features Require BDEF Declaration

If you implement `get_instance_features` FOR INSTANCE FEATURES, you MUST declare `( features: instance )` on each action in the BDEF:

```abap
" BDEF - Required for instance features:
action ( features: instance ) deploy parameter ... result ...;
action ( features: instance ) getSupportedTypes result ...;
```

**Error without this:**
```
The property "features:instance" is not defined for "ZADT_R_GIT_SERVICE"
```

### 3. %control Not Available in All Scenarios

For unmanaged RAP without draft, the `%control` component is not available in the result structure:

```abap
" WRONG - causes activation error:
APPEND VALUE #( %tky = <key>-%tky
                ServiceId = <key>-ServiceId
                %control = VALUE #( ... ) )  " Not available!
  TO result.

" CORRECT:
APPEND VALUE #( %tky = <key>-%tky
                ServiceId = <key>-ServiceId
                LastChanged = lv_ts )
  TO result.
```

### 4. Saver Class Naming

The saver class follows a different pattern - it uses the **entity name** (not alias):
```abap
CLASS lsc_zadt_r_git_service DEFINITION INHERITING FROM cl_abap_behavior_saver.
```

### 5. Authorization for Actions

For instance actions, you must return authorization for each action in `get_instance_authorizations`:
```abap
METHOD get_instance_authorizations.
  LOOP AT keys ASSIGNING FIELD-SYMBOL(<key>).
    APPEND VALUE #( %tky = <key>-%tky
                    %action-deploy = if_abap_behv=>auth-allowed
                    %action-getSupportedTypes = if_abap_behv=>auth-allowed )
      TO result.
  ENDLOOP.
ENDMETHOD.
```

### 6. Feature Control for Actions

Return feature control to enable/disable actions dynamically:
```abap
METHOD get_instance_features.
  LOOP AT keys ASSIGNING FIELD-SYMBOL(<key>).
    APPEND VALUE #( %tky = <key>-%tky
                    %action-deploy = if_abap_behv=>fc-o-enabled
                    %action-getSupportedTypes = if_abap_behv=>fc-o-enabled )
      TO result.
  ENDLOOP.
ENDMETHOD.
```

### 7. Static vs Instance Actions

We initially tried **static actions** (no instance binding) but encountered issues with unmanaged RAP. **Instance actions** on a singleton entity work better:

```abap
" BDEF - Instance actions (recommended for unmanaged):
action ( features: instance ) getSupportedTypes result [1] ZADT_A_GIT_TYPES_RESULT;

" NOT recommended - static actions have handler discovery issues:
static action getSupportedTypes result [1] ZADT_A_GIT_TYPES_RESULT;
```

### 8. CDS Singleton Pattern

For service-like entities with no real data, use a singleton pattern:
```sql
define root view entity ZADT_R_GIT_SERVICE
  as select from I_Language
{
  key cast( 'SERVICE' as abap.char(10) ) as ServiceId,
      @Semantics.systemDateTime.lastChangedAt: true
      tstmp_current_utctimestamp() as LastChanged
}
where Language = $session.system_language
```

### 9. OData V4 Action URL Pattern

Instance-bound actions use this URL pattern:
```
POST /EntitySet('Key')/Namespace.ActionName
```

Example:
```
POST /GitService('SERVICE')/com.sap.gateway.srvd.zadt_git_deploy.v0001.getSupportedTypes
```

### 10. CSRF Token Required for POST

All modifying operations require X-CSRF-Token:
```bash
# 1. Fetch token
curl -H "X-CSRF-Token: Fetch" .../service/ -D -

# 2. Use token in POST
curl -X POST -H "X-CSRF-Token: <token>" .../action
```

## Error Diagnosis

### CX_SADL_DUMP_APPL_MODEL_ERROR

This is a wrapper exception. Always check the **exception chain** in ST22 for the real cause:
- `CX_RAP_HANDLER_NOT_IMPLEMENTED` - Handler class not found
- Check "Involved Entities" to see which entity is affected

### Handler Not Implemented

If you see "Handler not implemented; Method: INSTANCE_AUTHORIZATION":
1. Check handler class name matches alias exactly
2. Ensure class is activated after BDEF changes
3. Re-activate BDEF to force metadata refresh

## Working Objects

| Object | Type | Description |
|--------|------|-------------|
| ZADT_R_GIT_SERVICE | DDLS | Root CDS entity (singleton) |
| ZADT_R_GIT_SERVICE | BDEF | Unmanaged behavior definition |
| ZCL_ZADT_GIT_SERVICE | CLAS | Behavior pool with lhc_GitService handler |
| ZADT_A_GIT_DEPLOY_PARAM | DDLS | Action input parameter (abstract) |
| ZADT_A_GIT_TYPES_RESULT | DDLS | Action result (abstract) |
| ZADT_GIT_DEPLOY | SRVD | Service definition |
| ZADT_GIT_DEPLOY_O4 | SRVB | V4 service binding |

## API Endpoints

```
Base URL: /sap/opu/odata4/sap/zadt_git_deploy_o4/srvd/sap/zadt_git_deploy/0001/

GET  /GitService                    - List entities (singleton)
GET  /GitService('SERVICE')         - Get entity
POST /GitService('SERVICE')/...getSupportedTypes  - Get supported types
POST /GitService('SERVICE')/...deploy             - Deploy repository
```

## Next Steps

1. Implement `deploy` action to accept zipped repository
2. Add `exportPackages` action to export packages as git-repo format
3. Integrate with existing abapGit classes if available
