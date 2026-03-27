# SAP ADT ABAP Internals Documentation

**Date:** 2025-12-01
**Purpose:** Comprehensive technical documentation of ADT's internal ABAP architecture based on source code analysis
**Source:** Direct source code retrieval via MCP ABAP ADT tools

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Request Flow Deep Dive](#2-request-flow-deep-dive)
3. [Core Framework Classes](#3-core-framework-classes)
4. [Content Handling & Serialization](#4-content-handling--serialization)
5. [Discovery Mechanism](#5-discovery-mechanism)
6. [Data Preview Implementation](#6-data-preview-implementation)
7. [Unit Test Execution](#7-unit-test-execution)
8. [How to Extend ADT](#8-how-to-extend-adt)
9. [Key Types & Interfaces Reference](#9-key-types--interfaces-reference)

---

## 1. Architecture Overview

### 1.1 High-Level Request Flow

```
Eclipse ADT Client
       │
       ▼
HTTP/RFC Request
       │
       ▼
┌─────────────────────────────────────────────────────────────────┐
│  SADT_REST_RFC_ENDPOINT (Function Module)                       │
│  ├── CSRF Token Validation                                      │
│  ├── Authorization Check (S_ADT_RES)                            │
│  ├── Route to Application via BAdI SADT_REST_RFC_APPLICATION    │
│  └── Delegate to CL_REST_RFC_UTILITIES=>CALL_REST_RFC_HANDLER   │
└─────────────────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────────┐
│  CL_ADT_RES_APP_BASE (Application Router)                       │
│  ├── URL Pattern Matching                                       │
│  ├── Resource Class Lookup                                      │
│  └── Instantiate Handler Class                                  │
└─────────────────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────────┐
│  CL_ADT_REST_RESOURCE Subclass (Feature Handler)                │
│  ├── GET / POST / PUT / DELETE methods                          │
│  ├── Content Handler for XML serialization                      │
│  └── Business Logic Execution                                   │
└─────────────────────────────────────────────────────────────────┘
       │
       ▼
HTTP Response to Eclipse
```

### 1.2 Package Structure

| Package | Purpose | Key Contents |
|---------|---------|--------------|
| `SADT_REST` | Core REST Framework | Dispatcher, base classes, discovery |
| `SADT_CORE` | Core Utilities | BAdI registry, helpers |
| `SDP_ADT` | Data Preview | SQL query execution |
| `SABP_UNIT_LEGACY_ADT` | Unit Testing | Test runner, results |
| `SEU_ADT` | Workbench | Object editing, activation |
| `SCTS_ADT` | Transport System | CTS integration |
| `SEO_ADT` | Object-Oriented | Class/Interface handling |

---

## 2. Request Flow Deep Dive

### 2.1 Entry Point: SADT_REST_RFC_ENDPOINT

This function module is the **central dispatcher** for ALL ADT REST requests from Eclipse.

**Signature:**
```abap
FUNCTION sadt_rest_rfc_endpoint
  IMPORTING
    VALUE(request)  TYPE sadt_rest_request
  EXPORTING
    VALUE(response) TYPE sadt_rest_response.
```

**Key Processing Steps:**

1. **Server Instance Routing** - Handles multi-instance SAP configurations:
```abap
cl_adt_res_app_access=>get_server_instance( request  = request
                                            instance = server_instance ).
IF server_instance IS NOT INITIAL.
  " Route to different application server if needed
  cl_adt_rest_target_server_rfc=>call_rest_rfc_handler(
    request  = request
    server   = server_instance
    response = response ).
  RETURN.
ENDIF.
```

2. **Get Application Resource** - Resolves URL to handler class:
```abap
cl_adt_res_app_access=>get_application_resource(
  request     = request
  application = application
  resource    = resource ).
```

3. **CSRF Token Handling**:
```abap
IF cl_adt_rest_csrf_manager=>is_required( request = request ).
  cl_adt_rest_csrf_manager=>handle_csrf(
    request  = request
    response = response ).
ENDIF.
```

4. **Authorization Check** - Uses authorization object `S_ADT_RES`:
```abap
AUTHORITY-CHECK OBJECT 'S_ADT_RES'
  ID 'ADT_APP' FIELD application
  ID 'ACTVT' FIELD activity.
```

5. **Delegate to REST Handler**:
```abap
cl_rest_rfc_utilities=>call_rest_rfc_handler(
  request     = request
  application = application
  resource    = resource
  response    = response ).
```

### 2.2 Application Router: CL_ADT_RES_APP_BASE

The router base class provides URL pattern matching and resource instantiation.

**Key Method - GET_RESOURCE_CLASS:**
```abap
METHOD get_resource_class.
  " Returns the ABAP class name for a given URI pattern
  " Subclasses override this to provide mapping

  " Example patterns:
  " /sap/bc/adt/programs/programs/{name}  -> CL_ADT_RES_PROGRAM
  " /sap/bc/adt/oo/classes/{name}         -> CL_ADT_RES_CLASS
  " /sap/bc/adt/datapreview/ddic          -> CL_ADT_DATAPREVIEW_RES
ENDMETHOD.
```

**URL Registration via BAdI:**
Applications register their URL patterns through BAdI `SADT_REST_RFC_APPLICATION` with interface `IF_ADT_REST_RFC_APPLICATION`.

---

## 3. Core Framework Classes

### 3.1 CL_ADT_REST_RESOURCE (Abstract Base Class)

**Every ADT resource handler extends this class.**

**Definition:**
```abap
CLASS cl_adt_rest_resource DEFINITION PUBLIC ABSTRACT
  CREATE PUBLIC.

  PUBLIC SECTION.
    METHODS get
      IMPORTING
        request  TYPE REF TO if_adt_rest_request
        response TYPE REF TO if_adt_rest_response
      RAISING
        cx_adt_rest.

    METHODS post
      IMPORTING
        request  TYPE REF TO if_adt_rest_request
        response TYPE REF TO if_adt_rest_response
      RAISING
        cx_adt_rest.

    METHODS put
      IMPORTING
        request  TYPE REF TO if_adt_rest_request
        response TYPE REF TO if_adt_rest_response
      RAISING
        cx_adt_rest.

    METHODS delete
      IMPORTING
        request  TYPE REF TO if_adt_rest_request
        response TYPE REF TO if_adt_rest_response
      RAISING
        cx_adt_rest.
ENDCLASS.
```

**Default Behavior:**
All HTTP methods raise `cx_adt_res_meth_not_supported` by default:
```abap
METHOD get.
  RAISE EXCEPTION TYPE cx_adt_res_meth_not_supported
    EXPORTING
      textid = cx_adt_res_meth_not_supported=>get.
ENDMETHOD.
```

**Subclasses override only the methods they support.**

### 3.2 IF_ADT_REST_REQUEST (Request Interface)

**Provides access to incoming request data:**

```abap
INTERFACE if_adt_rest_request PUBLIC.

  METHODS get_uri
    RETURNING VALUE(result) TYPE string.

  METHODS get_uri_query_parameter
    IMPORTING
      name          TYPE string
      mandatory     TYPE boole_d DEFAULT abap_false
    RETURNING
      VALUE(result) TYPE string
    RAISING
      cx_adt_rest.

  METHODS get_uri_query_parameter_values
    IMPORTING
      name          TYPE string
    RETURNING
      VALUE(result) TYPE string_table.

  METHODS get_uri_attribute
    IMPORTING
      name          TYPE string
    RETURNING
      VALUE(result) TYPE string.

  METHODS get_body_data
    IMPORTING
      content_handler TYPE REF TO if_adt_rest_content_handler
    EXPORTING
      data            TYPE data
    RAISING
      cx_adt_rest.

  METHODS get_inner_rest_request
    RETURNING
      VALUE(result) TYPE REF TO if_rest_request.

  METHODS has_body
    RETURNING
      VALUE(result) TYPE boole_d.

ENDINTERFACE.
```

**Key Methods:**
- `get_uri_query_parameter()` - Extract URL parameters like `?maxRows=100`
- `get_uri_attribute()` - Extract path segments like `/classes/{name}`
- `get_body_data()` - Deserialize request body using content handler
- `get_inner_rest_request()` - Access raw REST request for advanced use

### 3.3 IF_ADT_REST_RESPONSE (Response Interface)

**Provides methods to build the response:**

```abap
INTERFACE if_adt_rest_response PUBLIC.

  METHODS set_body_data
    IMPORTING
      content_handler TYPE REF TO if_adt_rest_content_handler
      data            TYPE data
    RAISING
      cx_adt_rest.

  METHODS set_status
    IMPORTING
      status TYPE i.

  METHODS set_header_field
    IMPORTING
      name  TYPE string
      value TYPE string.

  METHODS get_inner_rest_response
    RETURNING
      VALUE(result) TYPE REF TO if_rest_response.

ENDINTERFACE.
```

**Common Status Codes:**
```abap
" Success
response->set_status( 200 ).  " OK
response->set_status( 201 ).  " Created
response->set_status( 204 ).  " No Content

" Errors
response->set_status( 400 ).  " Bad Request
response->set_status( 404 ).  " Not Found
response->set_status( 409 ).  " Conflict (e.g., lock failed)
```

---

## 4. Content Handling & Serialization

### 4.1 CL_ADT_REST_ST_HANDLER (Simple Transformation Handler)

**The primary mechanism for XML serialization/deserialization using ABAP Simple Transformations.**

**Creation:**
```abap
DATA(content_handler) = cl_adt_rest_st_handler=>create_instance(
  st_name      = 'SADT_AUNIT_RUN_CONFIG'    " Transformation name
  root_name    = 'AUNIT'                     " Root element name
  content_type = if_rest_media_type=>gc_appl_xml
).
```

**Deserialization (XML → ABAP):**
```abap
METHOD if_adt_rest_content_handler~deserialize.
  DATA rtab TYPE abap_trans_resbind_tab.
  DATA body_data TYPE xstring.

  body_data = request_entity->get_binary_data( ).

  APPEND INITIAL LINE TO rtab ASSIGNING FIELD-SYMBOL(<rtab>).
  <rtab>-name = me->root_name.
  GET REFERENCE OF data INTO <rtab>-value.

  CALL TRANSFORMATION (me->st_name)
    SOURCE XML body_data
    RESULT     (rtab)
    OPTIONS    value_handling = 'reject_illegal_characters'.
ENDMETHOD.
```

**Serialization (ABAP → XML):**
```abap
METHOD if_adt_rest_content_handler~serialize.
  DATA stab TYPE abap_trans_srcbind_tab.
  DATA body_data TYPE xstring.

  APPEND INITIAL LINE TO stab ASSIGNING FIELD-SYMBOL(<stab>).
  <stab>-name = me->root_name.
  GET REFERENCE OF data INTO <stab>-value.

  CALL TRANSFORMATION (me->st_name)
    SOURCE    (stab)
    RESULT XML body_data.

  response_entity->set_binary_data( body_data ).
  response_entity->set_content_type(
    iv_media_type = 'application/xml'
    it_parameter  = VALUE #( ( name = 'charset' value = 'utf-8' ) ) ).
ENDMETHOD.
```

### 4.2 IF_ADT_REST_CONTENT_HANDLER Interface

```abap
INTERFACE if_adt_rest_content_handler PUBLIC.

  METHODS deserialize
    IMPORTING
      request_entity TYPE REF TO if_rest_entity
    EXPORTING
      data           TYPE data
    RAISING
      cx_adt_rest.

  METHODS serialize
    IMPORTING
      data            TYPE data
      response_entity TYPE REF TO if_rest_entity
    RAISING
      cx_adt_rest.

  METHODS get_supported_content_type
    RETURNING
      VALUE(result) TYPE string.

ENDINTERFACE.
```

### 4.3 Key Transformations

| Transformation | Purpose | Used By |
|----------------|---------|---------|
| `SADT_ATOM` | ATOM feed format | Discovery, lists |
| `SADT_EXCEPTION` | Exception to XML | Error responses |
| `ST_DATA_PREVIEW` | Data preview results | Table queries |
| `SADT_AUNIT_*` | Unit test structures | Test runner |
| `SADT_ABAP_SOURCE_OBJECT` | Source code | Object retrieval |

---

## 5. Discovery Mechanism

### 5.1 How Discovery Works

Eclipse calls `/sap/bc/adt/discovery` to learn what capabilities are available on the system.

**Handler: CL_ADT_RES_DISCOVERY**

```abap
METHOD get.
  DATA service_document TYPE if_app_types=>service_s.

  " Get relative URI path
  uri_path = request->get_inner_rest_request( )->get_header_field(
    if_http_header_fields_sap=>path ).

  " Collect registered workspaces from all providers
  DATA(access) = lcl_discovery_access_facade=>get_discovery_access( ).
  access->get_workspaces(
    uri        = uri_path
    workspaces = REF #( service_document-workspaces ) ).

  " Return as ATOM Service Document
  response->set_body_data(
    data            = service_document
    content_handler = cl_adt_rest_cnt_hdl_factory=>get_instance(
      )->get_handler_for_atom_pub( ) ).
ENDMETHOD.
```

### 5.2 Registering Capabilities (BAdI)

**Interface: IF_ADT_DISCOVERY_PROVIDER**
```abap
INTERFACE if_adt_discovery_provider PUBLIC.
  INTERFACES if_badi_interface.

  METHODS register_workspaces
    IMPORTING
      registry TYPE REF TO if_adt_discovery_registry.
ENDINTERFACE.
```

**Registering a Workspace:**
```abap
METHOD if_adt_discovery_provider~register_workspaces.
  DATA(workspace) = registry->register_workspace( 'my_feature' ).

  workspace->register_collection(
    name        = 'My Collection'
    title       = 'My Feature'
    uri         = '/sap/bc/adt/myfeature'
    media_types = VALUE #( ( 'application/xml' ) ) ).
ENDMETHOD.
```

### 5.3 IF_ADT_DISCOVERY_REGISTRY

```abap
INTERFACE if_adt_discovery_registry PUBLIC.
  METHODS register_workspace
    IMPORTING
      name          TYPE csequence
    RETURNING
      VALUE(result) TYPE REF TO if_adt_discovery_workspace.
ENDINTERFACE.
```

---

## 6. Data Preview Implementation

### 6.1 Overview

The Data Preview feature allows executing SQL queries on database tables and views via ADT.

**Endpoint:** `POST /sap/bc/adt/datapreview/ddic`

**Handler:** `CL_ADT_DATAPREVIEW_RES` in package `SDP_ADT`

### 6.2 Request Format

```http
POST /sap/bc/adt/datapreview/ddic?ddicEntityName=TADIR&rowNumber=100
Content-Type: text/plain

SELECT * FROM TADIR WHERE PGMID = 'R3TR' AND OBJECT = 'CLAS'
```

**Query Parameters:**
- `ddicEntityName` - Table or view name
- `rowNumber` - Maximum rows to return (default 100)

**Body:**
- Optional SQL query string for WHERE clause and other SQL options

### 6.3 Handler Implementation

**CL_ADT_DATAPREVIEW_RES - POST Method:**
```abap
METHOD post.
  DATA lv_ddic_entity TYPE string.
  DATA lv_row_count TYPE i.
  DATA lv_query_string TYPE string.

  " Get query parameters
  lv_ddic_entity = request->get_uri_query_parameter( 'ddicEntityName' ).
  lv_row_count = request->get_uri_query_parameter( 'rowNumber' ).

  IF lv_row_count <= 0.
    lv_row_count = if_adt_datapreview_res_co=>co_default_row_count. " 100
  ENDIF.

  " Get SQL query from body
  lv_query_string = request->get_inner_rest_request(
    )->get_entity( )->get_string_data( ).

  " Create Open SQL handler
  DATA(lo_query_handler) = cl_adt_dp_open_sql_handler=>get_instance(
    iv_query_string = lv_query_string ).

  " Execute query with authorization check
  lo_query_handler->get_query_result(
    EXPORTING
      iv_row_count   = lv_row_count
      iv_entity_name = lv_ddic_entity
    IMPORTING
      er_result      = lr_query_result ).

  " Serialize response
  response->set_body_data(
    data            = lr_query_result
    content_handler = ... ).
ENDMETHOD.
```

### 6.4 SQL Execution: CL_ADT_DP_OPEN_SQL_HANDLER

**Key Methods:**

```abap
CLASS cl_adt_dp_open_sql_handler.

  METHODS get_instance
    IMPORTING
      iv_query_string TYPE string
    RETURNING
      VALUE(result)   TYPE REF TO cl_adt_dp_open_sql_handler.

  METHODS get_query_result
    IMPORTING
      iv_row_count   TYPE i DEFAULT 100
      iv_entity_name TYPE string
    EXPORTING
      er_result      TYPE REF TO data
    RAISING
      cx_adt_datapreview.

ENDCLASS.
```

**Execution Flow:**

1. **Parse SQL** - Extract SELECT, WHERE, ORDER BY clauses
2. **Authorization Check** - Verify user has access to the table
3. **Build Dynamic SELECT** - Construct `SELECT` with row limit
4. **Execute** - Run the query
5. **Format Result** - Convert to XML-serializable structure

**Security Features:**
- Table-level authorization via `SE16` authority checks
- SQL injection prevention through parameterization
- Row limit enforcement
- Audit logging

---

## 7. Unit Test Execution

### 7.1 Overview

Eclipse ADT uses the ABAP Unit framework via ADT REST endpoints to execute and retrieve test results.

**Endpoint:** `POST /sap/bc/adt/abapunit/testruns`

**Handler:** `CL_AUNIT_ADT_RES_TEST_RUNS` in package `SABP_UNIT_LEGACY_ADT`

### 7.2 Request Format

```http
POST /sap/bc/adt/abapunit/testruns
Content-Type: application/vnd.sap.adt.abapunit.testruns.config.v4+xml

<?xml version="1.0" encoding="UTF-8"?>
<aunit:runConfiguration xmlns:aunit="http://www.sap.com/adt/aunit">
  <options>
    <uriType value="semantic"/>
    <testDeterminationStrategy appendAssignedTestsPreview="false"
                               assignedTests="false"
                               sameProgram="true"/>
    <testRiskCoverage harmless="true" dangerous="true" critical="true"/>
    <testDurationCoverage long="true" medium="true" short="true"/>
  </options>
  <adtcore:objectSets xmlns:adtcore="http://www.sap.com/adt/core">
    <objectSet kind="inclusive">
      <adtcore:objectReferences>
        <adtcore:objectReference adtcore:uri="/sap/bc/adt/oo/classes/zcl_my_test"/>
      </adtcore:objectReferences>
    </objectSet>
  </adtcore:objectSets>
</aunit:runConfiguration>
```

### 7.3 Request Types (IF_AUNIT_ADT_TYPES)

```abap
INTERFACE if_aunit_adt_types PUBLIC.

  " Request configuration
  TYPES: BEGIN OF ty_run_config_request,
           context      TYPE ty_context,
           config_options TYPE ty_run_config_options,
           object_sets  TYPE ty_object_sets,
         END OF ty_run_config_request.

  " Configuration options
  TYPES: BEGIN OF ty_run_config_options,
           coverage_active    TYPE boole_d,
           test_risk_levels   TYPE ty_risk_levels,
           test_durations     TYPE ty_durations,
           with_navigation_uri TYPE boole_d,
         END OF ty_run_config_options.

  " Risk level filtering
  TYPES: BEGIN OF ty_risk_levels,
           harmless  TYPE boole_d,
           dangerous TYPE boole_d,
           critical  TYPE boole_d,
         END OF ty_risk_levels.

  " Duration filtering
  TYPES: BEGIN OF ty_durations,
           short  TYPE boole_d,
           medium TYPE boole_d,
           long   TYPE boole_d,
         END OF ty_durations.

ENDINTERFACE.
```

### 7.4 Response Structure

```abap
TYPES: BEGIN OF ty_run_config_response,
         external    TYPE ty_external_info,
         alerts      TYPE ty_alerts,
         programs    TYPE ty_programs,
       END OF ty_run_config_response.

" Programs contain test classes
TYPES: BEGIN OF ty_program,
         uri          TYPE string,
         name         TYPE string,
         type         TYPE string,
         alerts       TYPE ty_alerts,
         test_classes TYPE ty_test_classes,
       END OF ty_program.

" Test classes contain test methods
TYPES: BEGIN OF ty_test_class,
         uri          TYPE string,
         name         TYPE string,
         alerts       TYPE ty_alerts,
         test_methods TYPE ty_test_methods,
       END OF ty_test_class.

" Test method with result status
TYPES: BEGIN OF ty_test_method,
         uri          TYPE string,
         name         TYPE string,
         execution_time TYPE p,
         alerts       TYPE ty_alerts,
       END OF ty_test_method.

" Alert structure (failures, errors, warnings)
TYPES: BEGIN OF ty_alert,
         kind     TYPE string,  " 'failure', 'error', 'warning'
         severity TYPE string,
         title    TYPE string,
         details  TYPE ty_alert_details,
         stack    TYPE ty_stack_entries,
       END OF ty_alert.
```

### 7.5 Handler Implementation

**CL_AUNIT_ADT_RES_TEST_RUNS - POST Method (Simplified):**

```abap
METHOD post.
  DATA ls_request TYPE if_aunit_adt_types=>ty_run_config_request.
  DATA ls_response TYPE if_aunit_adt_types=>ty_run_config_response.

  " 1. Parse request with content type negotiation
  DATA(content_handler) = get_content_handler_for_version( request ).
  request->get_body_data(
    EXPORTING content_handler = content_handler
    IMPORTING data = ls_request ).

  " 2. Extract test objects from request
  DATA(lt_test_objects) = extract_test_objects( ls_request-object_sets ).

  " 3. Execute tests
  DATA(lo_listener) = NEW cl_aunit_adt_listener( ).

  LOOP AT lt_test_objects INTO DATA(ls_object).
    " Run ABAP Unit for each object
    cl_aunit_test_runner=>run_tests(
      object     = ls_object
      listener   = lo_listener
      options    = ls_request-config_options ).
  ENDLOOP.

  " 4. Collect results from listener
  ls_response = lo_listener->get_results( ).

  " 5. Return response
  response->set_body_data(
    data            = ls_response
    content_handler = content_handler ).
ENDMETHOD.
```

### 7.6 Content-Type Versioning

ADT supports multiple API versions via content-type headers:

```
application/vnd.sap.adt.abapunit.testruns.config.v1+xml
application/vnd.sap.adt.abapunit.testruns.config.v2+xml
application/vnd.sap.adt.abapunit.testruns.config.v3+xml
application/vnd.sap.adt.abapunit.testruns.config.v4+xml
```

The handler negotiates the appropriate transformation based on the `Accept` header.

---

## 8. How to Extend ADT

### 8.1 Creating a New ADT Endpoint

**Step 1: Create Resource Class**

```abap
CLASS zcl_my_adt_resource DEFINITION
  PUBLIC FINAL
  CREATE PUBLIC
  INHERITING FROM cl_adt_rest_resource.

  PUBLIC SECTION.
    METHODS get REDEFINITION.
    METHODS post REDEFINITION.

ENDCLASS.

CLASS zcl_my_adt_resource IMPLEMENTATION.

  METHOD get.
    DATA ls_data TYPE my_data_type.

    " Get URI parameter
    DATA(lv_id) = request->get_uri_attribute( 'id' ).

    " Fetch data
    SELECT SINGLE * FROM my_table INTO ls_data WHERE id = lv_id.

    " Serialize response
    response->set_body_data(
      data            = ls_data
      content_handler = cl_adt_rest_st_handler=>create_instance(
        st_name = 'ZMY_DATA_TRANSFORM' ) ).
  ENDMETHOD.

  METHOD post.
    DATA ls_input TYPE my_input_type.
    DATA ls_result TYPE my_result_type.

    " Deserialize request body
    request->get_body_data(
      EXPORTING
        content_handler = cl_adt_rest_st_handler=>create_instance(
          st_name = 'ZMY_INPUT_TRANSFORM' )
      IMPORTING
        data = ls_input ).

    " Process
    ls_result = process_input( ls_input ).

    " Return result
    response->set_status( 201 ).
    response->set_body_data(
      data            = ls_result
      content_handler = cl_adt_rest_st_handler=>create_instance(
        st_name = 'ZMY_RESULT_TRANSFORM' ) ).
  ENDMETHOD.

ENDCLASS.
```

**Step 2: Create Simple Transformation**

```xml
<?sap.transform simple?>
<tt:transform xmlns:tt="http://www.sap.com/transformation-templates">
  <tt:root name="ROOT"/>
  <tt:template>
    <my_data>
      <id tt:value-ref="ROOT.ID"/>
      <name tt:value-ref="ROOT.NAME"/>
    </my_data>
  </tt:template>
</tt:transform>
```

**Step 3: Register URL Pattern (BAdI)**

```abap
CLASS zcl_my_adt_app DEFINITION
  PUBLIC FINAL
  CREATE PUBLIC.

  PUBLIC SECTION.
    INTERFACES if_adt_rest_rfc_application.

ENDCLASS.

CLASS zcl_my_adt_app IMPLEMENTATION.

  METHOD if_adt_rest_rfc_application~get_static_uri_path.
    result = '/sap/bc/adt/zmyfeature'.
  ENDMETHOD.

  METHOD if_adt_rest_rfc_application~get_resource_class.
    CASE uri_path.
      WHEN '/sap/bc/adt/zmyfeature/{id}'.
        result = 'ZCL_MY_ADT_RESOURCE'.
    ENDCASE.
  ENDMETHOD.

ENDCLASS.
```

**Step 4: Create SICF Service Node**

1. Transaction `SICF`
2. Navigate to `/sap/bc/adt/`
3. Create new service node `zmyfeature`
4. Set handler class or use default ADT handler

### 8.2 Implementing Discovery Provider

```abap
CLASS zcl_my_discovery_provider DEFINITION
  PUBLIC FINAL
  CREATE PUBLIC.

  PUBLIC SECTION.
    INTERFACES if_adt_discovery_provider.

ENDCLASS.

CLASS zcl_my_discovery_provider IMPLEMENTATION.

  METHOD if_adt_discovery_provider~register_workspaces.
    DATA(workspace) = registry->register_workspace( 'zmyfeature' ).

    workspace->register_collection(
      name        = 'myFeature'
      title       = 'My Custom ADT Feature'
      uri         = '/sap/bc/adt/zmyfeature'
      media_types = VALUE #(
        ( 'application/xml' )
        ( 'application/vnd.sap.my.feature+xml' ) ) ).
  ENDMETHOD.

ENDCLASS.
```

---

## 9. Key Types & Interfaces Reference

### 9.1 Exception Classes

| Exception | Purpose |
|-----------|---------|
| `CX_ADT_REST` | Base exception for ADT REST |
| `CX_ADT_RES_METH_NOT_SUPPORTED` | HTTP method not implemented |
| `CX_ADT_REST_DATA_INVALID` | Invalid request/response data |
| `CX_ADT_RES_CHARS_NOT_ACCEPTED` | Invalid characters in content |
| `CX_ADT_RES_NOT_FOUND` | Resource not found (404) |
| `CX_ADT_RES_CONFLICT` | Lock conflict (409) |
| `CX_ADT_DATAPREVIEW` | Data preview specific errors |

### 9.2 Constants Interfaces

| Interface | Purpose |
|-----------|---------|
| `IF_ADT_DATAPREVIEW_RES_CO` | Data preview constants |
| `IF_REST_MEDIA_TYPE` | Media type constants |
| `IF_HTTP_HEADER_FIELDS_SAP` | HTTP header names |

### 9.3 Core Types

```abap
" ADT REST Request Structure (RFC)
TYPES: BEGIN OF sadt_rest_request,
         method        TYPE string,
         uri           TYPE string,
         headers       TYPE tihttpnvp,
         body          TYPE xstring,
         query_params  TYPE tihttpnvp,
       END OF sadt_rest_request.

" ADT REST Response Structure (RFC)
TYPES: BEGIN OF sadt_rest_response,
         status        TYPE i,
         reason        TYPE string,
         headers       TYPE tihttpnvp,
         body          TYPE xstring,
       END OF sadt_rest_response.
```

---

## Appendix A: URL Pattern Quick Reference

| Pattern | Handler | Package | Purpose |
|---------|---------|---------|---------|
| `/sap/bc/adt/discovery` | `CL_ADT_RES_DISCOVERY` | SADT_REST | Capability discovery |
| `/sap/bc/adt/programs/programs/{name}/source/main` | `CL_ADT_RES_PROGRAM*` | SEU_ADT | Program source |
| `/sap/bc/adt/oo/classes/{name}/source/main` | `CL_ADT_RES_CLASS*` | SEO_ADT | Class source |
| `/sap/bc/adt/oo/interfaces/{name}/source/main` | `CL_ADT_RES_INTF*` | SEO_ADT | Interface source |
| `/sap/bc/adt/functions/groups/{grp}/fmodules/{name}` | `CL_ADT_RES_FM*` | - | Function module |
| `/sap/bc/adt/ddic/tables/{name}` | `CL_ADT_RES_TABLE*` | - | Table structure |
| `/sap/bc/adt/datapreview/ddic` | `CL_ADT_DATAPREVIEW_RES` | SDP_ADT | SQL query |
| `/sap/bc/adt/abapunit/testruns` | `CL_AUNIT_ADT_RES_TEST_RUNS` | SABP_UNIT_LEGACY_ADT | Unit tests |
| `/sap/bc/adt/cts/transportrequests` | `CL_ADT_CTS_*` | SCTS_ADT | Transports |
| `/sap/bc/adt/atc/runs` | `CL_ADT_ATC_*` | SATC_ADT | Code checks |
| `/sap/bc/adt/debugger/*` | `CL_ADT_DEBUG_*` | - | Debugger |

---

## Appendix B: Debugging Tips

### Setting Breakpoints

1. **External Breakpoint on SADT_REST_RFC_ENDPOINT:**
   - Set for your user
   - Trigger action in Eclipse
   - Step through to see routing

2. **Breakpoint on Resource GET/POST:**
   - Identify handler class from URL
   - Set breakpoint on specific method
   - Inspect request/response objects

### Useful Transactions

| TCode | Purpose |
|-------|---------|
| SICF | ICF service configuration |
| SMICM | ICM trace and logs |
| ST01 | System trace |
| SE24 | Class browser (inspect ADT classes) |
| SE37 | Function module browser |
| SE80 | Object navigator |

### Trace Flags

```abap
" Enable verbose logging in development
SET EXTENDED CHECK ON.

" Trace ADT requests
CALL FUNCTION 'SADT_REST_TRACE_START'.
" ... perform operation ...
CALL FUNCTION 'SADT_REST_TRACE_STOP'.
```

---

## Appendix C: Common Issues & Solutions

### Issue: 404 on Custom Endpoint
**Cause:** SICF service not activated or BAdI not registered
**Solution:**
1. Check SICF service node is active
2. Verify BAdI implementation is active
3. Check authorization object S_ADT_RES

### Issue: CSRF Token Error
**Cause:** Token not refreshed or missing
**Solution:**
1. First call `GET` with `X-CSRF-Token: Fetch` header
2. Use returned token in subsequent `POST/PUT/DELETE`

### Issue: Content Type Mismatch
**Cause:** Wrong content type or missing Accept header
**Solution:**
1. Check supported content types in discovery
2. Set correct `Content-Type` and `Accept` headers
3. Use proper version suffix (v1, v2, v3, v4)

---

## Appendix D: Mapping ABAP Internals to abap-adt-api Library

This section bridges the gap between ABAP ADT internals and the `abap-adt-api` library by Marcello Urbani, showing which ABAP classes/endpoints each library method calls.

### D.1 Library Architecture Overview

The `abap-adt-api` library provides a TypeScript/JavaScript wrapper around ADT REST endpoints. It handles:
- CSRF token management
- Session cookies
- XML serialization/deserialization
- Content-type negotiation

**Key Insight:** The library does NOT use RFC or BAPIs - it's pure HTTP/REST against the same endpoints Eclipse uses.

### D.2 Method-to-ABAP Mapping

| Library Method | ADT Endpoint | ABAP Handler Class | Package |
|----------------|--------------|-------------------|---------|
| `getObjectSource()` | `GET /sap/bc/adt/{type}/{name}/source/main` | `CL_ADT_RES_*` | SEU_ADT, SEO_ADT |
| `setObjectSource()` | `PUT /sap/bc/adt/{type}/{name}/source/main` | `CL_ADT_RES_*` | SEU_ADT, SEO_ADT |
| `lock()` | `POST /sap/bc/adt/{type}/{name}?_action=LOCK` | `CL_ADT_LOCK_HANDLER` | SADT_CORE |
| `unLock()` | `POST /sap/bc/adt/{type}/{name}?_action=UNLOCK` | `CL_ADT_LOCK_HANDLER` | SADT_CORE |
| `activate()` | `POST /sap/bc/adt/activation` | `CL_ADT_WB_ACTIVATION` | SEU_ADT |
| `syntaxCheck()` | `POST /sap/bc/adt/checkruns` | `CL_ADT_CHECK_RUN_RES` | SEU_ADT |
| `unitTestRun()` | `POST /sap/bc/adt/abapunit/testruns` | `CL_AUNIT_ADT_RES_TEST_RUNS` | SABP_UNIT_LEGACY_ADT |
| `tableContents()` | `POST /sap/bc/adt/datapreview/ddic` | `CL_ADT_DATAPREVIEW_RES` | SDP_ADT |
| `createAtcRun()` | `POST /sap/bc/adt/atc/runs` | `CL_ADT_ATC_RES_RUNS` | SATC_ADT |
| `atcWorklists()` | `GET /sap/bc/adt/atc/worklists` | `CL_ADT_ATC_RES_WORKLISTS` | SATC_ADT |
| `codeCompletion()` | `POST /sap/bc/adt/codecompletion` | `CL_ADT_COCO_RES` | S_CODE_COMPLETION_ADT |
| `findDefinition()` | `GET /sap/bc/adt/navigation/targets` | `CL_ADT_NAV_TARGET_RES` | SADT_REST |
| `usageReferences()` | `GET /sap/bc/adt/uses` | `CL_ADT_WHERE_USED_RES` | SEU_ADT |
| `debuggerListen()` | `POST /sap/bc/adt/debugger/listeners` | `CL_ADT_DEBUG_LISTENER_RES` | SADT_DEBUG |
| `debuggerSetBreakpoints()` | `POST /sap/bc/adt/debugger/breakpoints` | `CL_ADT_DEBUG_BREAKPOINTS_RES` | SADT_DEBUG |
| `transportInfo()` | `GET /sap/bc/adt/cts/transportrequests/{id}` | `CL_ADT_CTS_RES_TR` | SCTS_ADT |
| `userTransports()` | `GET /sap/bc/adt/cts/transportrequests` | `CL_ADT_CTS_RES_TR_LIST` | SCTS_ADT |
| `gitRepos()` | `GET /sap/bc/adt/abapgit/repos` | `CL_ABAPGIT_ADT_RES_REPOS` | SABP_ABAPGIT_ADT |
| `prettyPrinter()` | `POST /sap/bc/adt/abapsource/prettyprinter` | `CL_ADT_PP_RES` | SADT_TOOLS_CORE |

### D.3 What's Covered in abap-adt-api But Missing From MCP

**High-Value Missing Features:**

| Feature | Library Method | ABAP Internals Documented? | Implementation Complexity |
|---------|---------------|---------------------------|--------------------------|
| Unit Testing | `unitTestRun()` | **Yes** (Section 7) | Medium - XML parsing |
| SQL Queries | `tableContents()` | **Yes** (Section 6) | Low - already documented |
| ATC Checks | `createAtcRun()` | Partial | Medium |
| Syntax Check | `syntaxCheck()` | No | Low |
| Code Navigation | `findDefinition()` | No | Low |
| Where-Used | `usageReferences()` | No | Low |
| Transport Info | `transportInfo()` | No | Low |
| Activation | `activate()` | No | Medium |

### D.4 What's NOT in abap-adt-api (Useful Missing Pieces)

These ADT capabilities exist in ABAP but are **not implemented** in the `abap-adt-api` library:

#### D.4.1 CDS-Specific Operations

| Capability | ADT Endpoint | ABAP Handler | Notes |
|------------|--------------|--------------|-------|
| CDS Annotation Completion | `/sap/bc/adt/ddic/cds/annotationvaluehelp` | `CL_ADT_CDS_ANNO_VH_RES` | Annotation suggestions |
| CDS Impact Analysis | `/sap/bc/adt/ddic/cds/impactanalysis` | `CL_ADT_CDS_IMPACT_RES` | Find affected objects |
| CDS Element Info | `/sap/bc/adt/ddic/cds/elementinfo` | `CL_ADT_CDS_ELEM_INFO_RES` | Element metadata |

#### D.4.2 RAP (RESTful ABAP Programming) Operations

| Capability | ADT Endpoint | Notes |
|------------|--------------|-------|
| Behavior Definition Preview | `/sap/bc/adt/bo/behaviors/{name}` | BDEF source |
| Projection View | `/sap/bc/adt/ddic/cds/projectionview` | Projection metadata |
| Service Binding Test | `/sap/bc/adt/businessservices/binding/{name}/test` | Test OData service |

#### D.4.3 Performance Analysis

| Capability | ADT Endpoint | ABAP Handler | Notes |
|------------|--------------|--------------|-------|
| SQL Explain Plan | `/sap/bc/adt/datapreview/sqlexplainplan` | `CL_ADT_DP_EXPLAIN_PLAN_RES` | Query optimization |
| Runtime Analysis | `/sap/bc/adt/runtime/traces` | `CL_ADT_RUNTIME_TRACE_RES` | SE30-like analysis |
| Memory Analysis | `/sap/bc/adt/runtime/memoryanalysis` | - | Memory inspector |

#### D.4.4 Advanced Debugging

| Capability | ADT Endpoint | Notes |
|------------|--------------|-------|
| Conditional Breakpoints | `/sap/bc/adt/debugger/breakpoints` (with condition XML) | Not fully exposed |
| Watchpoints | `/sap/bc/adt/debugger/watchpoints` | Variable change monitoring |
| Exception Breakpoints | `/sap/bc/adt/debugger/exceptionbreakpoints` | Break on exception |
| Debug Script Support | `/sap/bc/adt/debugger/scripts` | ABAP debugger scripts |

#### D.4.5 Feed/Activity Streams

| Capability | ADT Endpoint | Notes |
|------------|--------------|-------|
| Object Change Feed | `/sap/bc/adt/feed/objects` | Recent changes |
| User Activity Feed | `/sap/bc/adt/feed/users/{user}` | Developer activity |
| Transport Feed | `/sap/bc/adt/feed/transports` | Transport changes |

### D.5 Discovery Endpoint Analysis

The `/sap/bc/adt/discovery` endpoint returns the complete feature catalog. Capabilities NOT documented elsewhere:

```xml
<!-- Sample discovery entries often overlooked -->
<collection href="/sap/bc/adt/vit/wb/object_type">
  <title>Workbench Object Types</title>
</collection>

<collection href="/sap/bc/adt/ddic/srvd/srvd">
  <title>Service Definitions</title>
</collection>

<collection href="/sap/bc/adt/ddic/srvb/srvb">
  <title>Service Bindings</title>
</collection>

<collection href="/sap/bc/adt/classifications">
  <title>API Classifications</title>
</collection>
```

---

## Appendix E: Implementation Recipes for MCP Server

Ready-to-implement patterns for extending the MCP server with high-value features.

### E.1 Add Native Table Query (Replace Z-Service)

**Current Problem:** MCP uses custom `/z_mcp_abap_adt/z_tablecontent` endpoint.

**Solution:** Use native `/sap/bc/adt/datapreview/ddic`:

```typescript
// New handler: handleQueryTable.ts
export async function handleQueryTable(args: {
  table_name: string;
  where_clause?: string;
  max_rows?: number;
}) {
  const url = `${await getBaseUrl()}/sap/bc/adt/datapreview/ddic`;

  const params = new URLSearchParams({
    ddicEntityName: args.table_name,
    rowNumber: String(args.max_rows || 100)
  });

  const sqlQuery = args.where_clause
    ? `SELECT * FROM ${args.table_name} WHERE ${args.where_clause}`
    : `SELECT * FROM ${args.table_name}`;

  const response = await makeAdtRequest(
    `${url}?${params}`,
    'POST',
    30000,
    sqlQuery,
    { 'Content-Type': 'text/plain' }
  );

  return return_response(response);
}
```

### E.2 Add Unit Test Runner

```typescript
// handleRunUnitTests.ts
export async function handleRunUnitTests(args: {
  object_uri: string;  // e.g., "/sap/bc/adt/oo/classes/zcl_my_test"
  risk_level?: 'harmless' | 'dangerous' | 'critical';
}) {
  const url = `${await getBaseUrl()}/sap/bc/adt/abapunit/testruns`;

  const requestBody = `<?xml version="1.0" encoding="UTF-8"?>
<aunit:runConfiguration xmlns:aunit="http://www.sap.com/adt/aunit">
  <options>
    <uriType value="semantic"/>
    <testRiskCoverage harmless="true" dangerous="true" critical="true"/>
    <testDurationCoverage short="true" medium="true" long="true"/>
  </options>
  <adtcore:objectSets xmlns:adtcore="http://www.sap.com/adt/core">
    <objectSet kind="inclusive">
      <adtcore:objectReferences>
        <adtcore:objectReference adtcore:uri="${args.object_uri}"/>
      </adtcore:objectReferences>
    </objectSet>
  </adtcore:objectSets>
</aunit:runConfiguration>`;

  const response = await makeAdtRequest(
    url,
    'POST',
    60000,  // Tests may take longer
    requestBody,
    {
      'Content-Type': 'application/vnd.sap.adt.abapunit.testruns.config.v4+xml',
      'Accept': 'application/vnd.sap.adt.abapunit.testruns.result.v1+xml'
    }
  );

  return return_response(response);
}
```

### E.3 Add Syntax Check

```typescript
// handleSyntaxCheck.ts
export async function handleSyntaxCheck(args: {
  object_uri: string;
}) {
  const url = `${await getBaseUrl()}/sap/bc/adt/checkruns`;

  const requestBody = `<?xml version="1.0" encoding="UTF-8"?>
<chkrun:checkRunRoot xmlns:chkrun="http://www.sap.com/adt/checkrun"
                     xmlns:adtcore="http://www.sap.com/adt/core">
  <adtcore:objectSets>
    <adtcore:objectSet kind="inclusive">
      <adtcore:objectReferences>
        <adtcore:objectReference adtcore:uri="${args.object_uri}"/>
      </adtcore:objectReferences>
    </adtcore:objectSet>
  </adtcore:objectSets>
</chkrun:checkRunRoot>`;

  const response = await makeAdtRequest(
    url,
    'POST',
    30000,
    requestBody,
    { 'Content-Type': 'application/vnd.sap.adt.checkobjects+xml' }
  );

  return return_response(response);
}
```

### E.4 Add Where-Used (Find References)

```typescript
// handleFindReferences.ts
export async function handleFindReferences(args: {
  object_uri: string;
}) {
  const url = `${await getBaseUrl()}/sap/bc/adt/uses`;

  const params = new URLSearchParams({
    uri: args.object_uri
  });

  const response = await makeAdtRequest(
    `${url}?${params}`,
    'GET',
    30000
  );

  return return_response(response);
}
```

### E.5 Add ATC Check

```typescript
// handleAtcCheck.ts
export async function handleAtcCheck(args: {
  object_uri: string;
  check_variant?: string;
}) {
  const url = `${await getBaseUrl()}/sap/bc/adt/atc/runs`;

  const requestBody = `<?xml version="1.0" encoding="UTF-8"?>
<atc:run xmlns:atc="http://www.sap.com/adt/atc"
         xmlns:adtcore="http://www.sap.com/adt/core"
         maximumVerdicts="100">
  <objectSets>
    <objectSet kind="inclusive">
      <adtcore:objectReferences>
        <adtcore:objectReference adtcore:uri="${args.object_uri}"/>
      </adtcore:objectReferences>
    </objectSet>
  </objectSets>
</atc:run>`;

  const response = await makeAdtRequest(
    url,
    'POST',
    60000,
    requestBody,
    { 'Content-Type': 'application/vnd.sap.atc.run.parameters.v1+xml' }
  );

  // Response contains worklist ID for retrieving results
  return return_response(response);
}
```

---

## Appendix F: Package Deep Dive Reference

Quick reference for exploring specific ADT functionality areas.

### F.1 For Unit Testing Investigation

```
Package: SABP_UNIT_LEGACY_ADT
Key Classes:
  - CL_AUNIT_ADT_RES_TEST_RUNS    (POST handler)
  - CL_AUNIT_ADT_RES_EVALUATION   (Result evaluation)
  - CL_AUNIT_ADT_LISTENER         (Test result collector)

Key Interfaces:
  - IF_AUNIT_ADT_TYPES            (All type definitions)
  - IF_AUNIT_LISTENER             (Callback interface)

Key Transformations:
  - SADT_AUNIT_RUN_CONFIG         (Request XML)
  - SADT_AUNIT_RUN_RESULT         (Response XML)
```

### F.2 For ATC/Code Quality Investigation

```
Package: SATC_ADT
Key Classes:
  - CL_ADT_ATC_RES_RUNS           (Create check run)
  - CL_ADT_ATC_RES_WORKLISTS      (Get results)
  - CL_ADT_ATC_RES_CUSTOMIZING    (Settings)

Key Tables:
  - SATC_AC_RESULTH               (Check run header)
  - SATC_AC_RESULTI               (Check run items)
```

### F.3 For Debugging Investigation

```
Package: SADT_DEBUG (or search SADT*DEBUG*)
Key Classes:
  - CL_ADT_DEBUG_LISTENER_RES     (Listener management)
  - CL_ADT_DEBUG_BREAKPOINTS_RES  (Breakpoint CRUD)
  - CL_ADT_DEBUG_STACK_RES        (Stack trace)
  - CL_ADT_DEBUG_VARIABLES_RES    (Variable inspection)

Key FM:
  - SADT_DEBUG_*                  (Debug helper FMs)
```

### F.4 For Transport Investigation

```
Package: SCTS_ADT
Key Classes:
  - CL_ADT_CTS_RES_TR             (Transport details)
  - CL_ADT_CTS_RES_TR_LIST        (Transport list)
  - CL_ADT_CTS_MANAGEMENT         (Transport operations)

Key Tables:
  - E070                          (Transport header)
  - E071                          (Transport objects)
  - E07T                          (Transport texts)
```

---

## Appendix G: Content-Type Version Cheat Sheet

ADT uses versioned content types for API evolution. Always check discovery for supported versions.

| Feature | Content-Type Pattern | Common Versions |
|---------|---------------------|-----------------|
| Unit Tests | `application/vnd.sap.adt.abapunit.testruns.config.v{N}+xml` | v1, v2, v3, v4 |
| ATC Runs | `application/vnd.sap.atc.run.parameters.v{N}+xml` | v1 |
| Check Runs | `application/vnd.sap.adt.checkobjects+xml` | (unversioned) |
| Activation | `application/vnd.sap.adt.activation+xml` | (unversioned) |
| Object Source | `text/plain` | N/A |
| Data Preview | `text/plain` (request), `application/xml` (response) | N/A |

**Best Practice:** Always send both `Content-Type` AND `Accept` headers to ensure proper request/response handling.

---

*Document generated from live SAP system source code analysis via MCP ABAP ADT tools*
*Last Updated: 2025-12-01*
*Coverage: Core ADT Framework, Data Preview, Unit Testing, with implementation recipes*
