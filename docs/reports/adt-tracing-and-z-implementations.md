# ADT Tracing & Custom Implementation Guide

**Date:** 2025-12-01
**Purpose:** How to trace Eclipse ADT requests and implement custom services

---

## Part 1: Z-Implementation Requirements in mcp-abap-adt

### What Requires Custom Implementation?

The current `mcp-abap-adt` server has **ONE** tool requiring a custom Z-service:

| Tool | Required Service | Type | Native Alternative |
|------|------------------|------|-------------------|
| `GetTableContents` | `/z_mcp_abap_adt/z_tablecontent` | Custom SICF/REST | **YES** - see below |

### The Problem

The MCP server's `GetTableContents` calls a non-standard endpoint:
```
GET /z_mcp_abap_adt/z_tablecontent/{table_name}?maxRows={n}
```

This requires ABAP development on the target system, which:
- May not be permitted (production systems, managed environments)
- Requires transport to each system
- Adds maintenance burden

### Native ADT Alternative (NO Z-code needed!)

SAP provides a **built-in data preview endpoint**:

```
POST /sap/bc/adt/datapreview/ddic
Content-Type: application/xml

Parameters:
- ddicEntityName: Table/View name
- rowNumber: Max rows (default 100)

Body: SQL query string (supports WHERE, ORDER BY, etc.)
```

**Example - Query TADIR with filter:**
```http
POST /sap/bc/adt/datapreview/ddic?ddicEntityName=TADIR&rowNumber=50
Content-Type: text/plain

SELECT * FROM TADIR WHERE PGMID = 'R3TR' AND OBJECT = 'CLAS' AND DEVCLASS LIKE 'Z%'
```

The `abap-adt-api` library implements this as:
```typescript
tableContents(
  ddicEntityName: string,   // Table name
  rowNumber?: number,       // Max rows (default 100)
  decode?: boolean,         // Decode type-specific values
  sqlQuery?: string         // WHERE clause / full SQL
)
```

### Recommendation

**Replace the custom Z-service approach with native ADT endpoint** - no ABAP development required, supports filtering, works on any ADT-enabled system.

---

## Part 2: RFC_READ_TABLE as Web Service (Alternative Approach)

If you need table access on older systems without `/sap/bc/adt/datapreview`, you can expose the standard **RFC_READ_TABLE** function module as a web service:

### Setup Steps (No Custom ABAP Code)

1. **Transaction SE80** → Create Enterprise Service
2. Select: Service Provider → Existing ABAP Object → Function Module
3. Choose: `RFC_READ_TABLE`
4. Configure authentication profile
5. Activate in **SOAMANAGER**

### Endpoint Pattern
```
/sap/bc/srt/rfc/sap/{service_name}/001/{service_name}/rfc_read_table
```

### SOAP Call Structure
```xml
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <RFC_READ_TABLE>
      <QUERY_TABLE>TADIR</QUERY_TABLE>
      <DELIMITER>|</DELIMITER>
      <ROWCOUNT>100</ROWCOUNT>
      <OPTIONS>
        <item><TEXT>PGMID = 'R3TR' AND OBJECT = 'CLAS'</TEXT></item>
      </OPTIONS>
      <FIELDS>
        <item><FIELDNAME>OBJ_NAME</FIELDNAME></item>
        <item><FIELDNAME>DEVCLASS</FIELDNAME></item>
      </FIELDS>
    </RFC_READ_TABLE>
  </soapenv:Body>
</soapenv:Envelope>
```

**Source:** [SAP Community - RFC_READ_TABLE via WebService](https://community.sap.com/t5/application-development-blog-posts/how-to-use-rfc-read-table-from-javascript-via-webservice/ba-p/13172358)

---

## Part 3: Tracing Eclipse ADT HTTP Requests

### Method 1: Eclipse ABAP Communication Log (Built-in)

**Best for:** Seeing what Eclipse sends to SAP

1. In Eclipse: **Window → Show View → Other**
2. Search for: **ABAP Communication Log**
3. Enable logging in the view
4. Perform actions in ADT
5. View captured HTTP GET/POST requests with URLs and payloads

This shows requests like:
```
GET /sap/bc/adt/programs/programs/ZTEST/source/main
POST /sap/bc/adt/abapunit/testruns
```

### Method 2: Fiddler Proxy (Detailed Traffic Analysis)

**Best for:** Full request/response inspection, modifying requests

**Setup Eclipse with Fiddler:**

1. Install [Fiddler](https://www.telerik.com/fiddler)
2. In Eclipse: **Run → Run Configurations → Arguments tab**
3. Add VM arguments:
```
-DproxySet=true
-DproxyHost=127.0.0.1
-DproxyPort=8888
```

**For HTTPS (typical for SAP):**
1. Export Fiddler's root certificate
2. Import into Java keystore:
```bash
keytool -import -alias fiddler -file FiddlerRoot.cer -keystore cacerts
```
3. Add to VM args:
```
-Djavax.net.ssl.trustStore=path/to/cacerts
```

**Source:** [Stack Overflow - Configuring Eclipse with Fiddler](https://stackoverflow.com/questions/5302476/configuring-eclipse-to-allow-fiddler-to-intercept-requests)

### Method 3: Charles Proxy (macOS/Cross-platform)

**Setup similar to Fiddler:**
```
-DproxySet=true
-DproxyHost=127.0.0.1
-DproxyPort=8888
-Dhttp.proxyHost=127.0.0.1
-Dhttp.proxyPort=8888
-Dhttps.proxyHost=127.0.0.1
-Dhttps.proxyPort=8888
```

**Source:** [Charles Proxy](https://www.charlesproxy.com/)

### Method 4: Wireshark (Network Level)

**Best for:** Low-level packet analysis

- No Eclipse configuration needed
- Filter: `http.host contains "sap"` or port 443
- **Note:** HTTPS requires SSL key logging for decryption

### Method 5: ABAP-Side Tracing

**Transaction SMICM** - ICM Trace:
1. Open `SMICM`
2. Goto → Trace Level → Set to 3
3. Execute operations
4. Goto → Trace File → Display

**Transaction ST01** - System Trace:
- Trace HTTP requests from ABAP side
- Shows internal processing

---

## Part 4: ADT Internal Architecture (For Reverse Engineering)

### Central Entry Point

**Function Module:** `SADT_REST_RFC_ENDPOINT`

This is the main dispatcher for ALL Eclipse ADT requests. Set a breakpoint here to trace any ADT operation.

### Key Classes to Investigate

| Class | Purpose |
|-------|---------|
| `CL_ADT_REST_RESOURCE` | Base class for ADT resources |
| `CL_ADT_REST_ST_HANDLER` | Content handler (XML serialization) |
| `CL_OO_ADT_RES_APP` | Resource registration (URL → Handler mapping) |
| `CL_ADT_DISCOVERY` | ADT capability discovery |

### Finding Handler Classes

1. **Method: URL Pattern Analysis**
   - Capture URL in proxy/log: `/sap/bc/adt/abapunit/testruns`
   - Search in SICF (transaction `SICF`) for path
   - Check handler class in service definition

2. **Method: Function Module Trace**
   - Set breakpoint on `HTTP_GET_HANDLER_LIST`
   - Returns list of handler classes for URL path

3. **Method: BAdI Investigation**
   - Check implementations of `BADI_ADT_REST_RFC_APPLICATION`
   - These route requests to specific handlers

### ADT URL Patterns → Handler Classes

| URL Pattern | Handler Area | Key Classes |
|-------------|--------------|-------------|
| `/sap/bc/adt/programs/*` | Programs | `CL_ADT_RES_PROGRAM*` |
| `/sap/bc/adt/oo/classes/*` | Classes | `CL_ADT_RES_CLASS*` |
| `/sap/bc/adt/abapunit/*` | Unit Tests | `CL_ADT_RES_UNIT*` |
| `/sap/bc/adt/debugger/*` | Debugger | `CL_ADT_RES_DEBUG*` |
| `/sap/bc/adt/atc/*` | Code Checks | `CL_ADT_RES_ATC*` |
| `/sap/bc/adt/datapreview/*` | Data Preview | `CL_ADT_RES_DATAPREVIEW*` |
| `/sap/bc/adt/cts/*` | Transports | `CL_ADT_RES_CTS*` |

### Debugging ADT Requests in ABAP

1. Set external breakpoint on `SADT_REST_RFC_ENDPOINT`
2. In Eclipse, attach debugger to your user
3. Perform ADT action (e.g., run unit test)
4. Breakpoint triggers - trace through the code

**Key method to trace:**
```abap
CLASS cl_adt_rest_resource
  METHOD get.    " For GET requests
  METHOD post.   " For POST requests
  METHOD put.    " For PUT requests
```

---

## Part 5: Practical Reverse Engineering Workflow

### Goal: Understand how "Run Unit Tests" works

**Step 1: Capture the Request**
```
# In Fiddler/Charles, observe:
POST /sap/bc/adt/abapunit/testruns
Content-Type: application/vnd.sap.adt.abapunit.testruns.config.v3+xml

<aunit:runConfiguration>
  <objectSets>
    <objectSet kind="inclusive">
      <adtcore:objectReference adtcore:uri="/sap/bc/adt/oo/classes/zcl_my_test"/>
    </objectSet>
  </objectSets>
</aunit:runConfiguration>
```

**Step 2: Find the Handler**
```abap
" In SICF, navigate to:
/sap/bc/adt/abapunit/testruns

" Check handler class - likely CL_ADT_RES_UNITTEST_RUNS or similar
```

**Step 3: Trace the Code**
```abap
" Set breakpoint on the handler's POST method
" Step through to understand:
" - How test configuration is parsed
" - How tests are executed
" - How results are formatted
```

**Step 4: Replicate in MCP**
- Construct same XML payload
- POST to same endpoint
- Parse XML response

---

## Part 6: Debugging via ADT vs OData/APC

### ADT Debugging (What Eclipse Uses)

**Endpoint:** `/sap/bc/adt/debugger/*`

**Flow:**
1. Create listener: `POST /sap/bc/adt/debugger/listeners`
2. Set breakpoints: `POST /sap/bc/adt/debugger/breakpoints`
3. Wait for hit: Long-polling on listener
4. Inspect: `GET /sap/bc/adt/debugger/stack`, `/variables`
5. Step: `POST /sap/bc/adt/debugger` with step command

**Pros:** Native integration, full Eclipse debugger feature parity
**Cons:** Complex protocol, many endpoints, session management

### OData Alternative

Some newer ABAP systems expose debugging via OData services.
Check: `/sap/opu/odata/sap/` for debugger-related services.

### ABAP Push Channels (APC/AMC)

**WebSocket-based communication** for real-time debugging events.

**Check:** Transaction `SAPC` for configured push channels.
Eclipse may use APC for:
- Real-time variable updates
- Debugger event notifications
- Console output streaming

**Tracing APC:**
- Transaction `SAMC` - Message channels
- Set trace on APC framework classes

---

## Summary: Recommended Approach

### For Table Data Access
1. **Use native ADT endpoint** `/sap/bc/adt/datapreview/ddic` - no Z-code needed
2. Fallback: RFC_READ_TABLE as web service (no coding, just config)

### For Understanding ADT Operations
1. **Quick:** Eclipse ABAP Communication Log
2. **Detailed:** Fiddler/Charles proxy
3. **Deep dive:** ABAP debugger on `SADT_REST_RFC_ENDPOINT`

### For Implementing New MCP Tools
1. Capture Eclipse request with proxy
2. Find SICF handler class
3. Debug handler to understand logic
4. Replicate HTTP call in MCP server

---

## References

- [SAP Community - How ADT Works](https://community.sap.com/t5/technology-blogs-by-sap/an-example-to-help-you-understand-how-does-adt-work/ba-p/13262858)
- [SAP Community - RFC_READ_TABLE WebService](https://community.sap.com/t5/application-development-blog-posts/how-to-use-rfc-read-table-from-javascript-via-webservice/ba-p/13172358)
- [Stack Overflow - Eclipse with Fiddler](https://stackoverflow.com/questions/5302476/configuring-eclipse-to-allow-fiddler-to-intercept-requests)
- [DSAG Handbook - ADT in Eclipse](https://impulsant-dsag.de/wp-content/uploads/2024/08/2303_DSAG-Leitfaden_Handbook_ADT_Eclipse_EN.pdf)
- [SAP ADT Configuration Guide](https://help.sap.com/doc/2e65ad9a26c84878b1413009f8ac07c3/202210.000/en-US/config_guide_system_backend_abap_development_tools.pdf)
