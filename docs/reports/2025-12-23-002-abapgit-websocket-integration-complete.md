# abapGit WebSocket Integration - Implementation Report

**Date:** 2025-12-23
**Report ID:** 002
**Subject:** Git domain implementation via ZADT_VSP WebSocket handler
**Status:** Complete
**Version:** v2.16.0

---

## Executive Summary

VSP now supports abapGit-compatible export of ABAP objects via the ZADT_VSP WebSocket handler. This enables export of **158 object types** using abapGit's native serialization, producing ZIP files that are fully compatible with abapGit repositories.

## Architecture

### Data Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           vsp (Go Client)                                │
│  GitTypes() / GitExport()                                               │
│       │                                                                  │
│       ▼                                                                  │
│  AMDPWebSocketClient.sendGitRequest()                                   │
│       │                                                                  │
│       ▼                                                                  │
│  JSON Message: {"domain":"git","action":"export","params":{...}}        │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │ WebSocket
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    ZADT_VSP APC Handler (ABAP)                          │
│                                                                          │
│  ZCL_VSP_APC_HANDLER                                                    │
│       │ routes to                                                        │
│       ▼                                                                  │
│  ZCL_VSP_GIT_SERVICE (domain: "git")                                    │
│       │                                                                  │
│       ├─► getTypes: ZCL_ABAPGIT_OBJECTS=>supported_list()               │
│       │                                                                  │
│       └─► export:                                                        │
│            1. Get objects from TADIR                                    │
│            2. ZCL_ABAPGIT_OBJECTS=>serialize() for each                 │
│            3. CL_ABAP_ZIP=>add() files                                  │
│            4. SSFC_BASE64_ENCODE (ZIP → base64)                         │
│            5. Return JSON with base64 ZIP                               │
└─────────────────────────────────────────────────────────────────────────┘
```

### Binary Transfer via Base64

Since WebSocket/JSON only supports text, binary ZIP data is encoded:

```
ABAP:  xstring (ZIP) ──► SSFC_BASE64_ENCODE ──► string (base64)
                                │
                         JSON over WebSocket
                                │
Go:    string (base64) ──► base64.StdEncoding.DecodeString ──► []byte (ZIP)
```

**Overhead:** Base64 adds ~33% to payload size (e.g., 23KB ZIP → 31KB base64).

## SAP/ABAP Internal Dependencies

### Required SAP Components

The Git service depends on **abapGit** being installed on the SAP system:

| Component | Purpose | Required |
|-----------|---------|----------|
| `ZCL_ABAPGIT_OBJECTS` | Object serialization/deserialization | ✅ Yes |
| `ZCL_ABAPGIT_FACTORY` | Factory for TADIR access | ✅ Yes |
| `ZIF_ABAPGIT_DEFINITIONS` | Type definitions (ty_tadir, ty_item) | ✅ Yes |
| `ZCL_ABAPGIT_I18N_PARAMS` | Internationalization parameters | ✅ Yes |
| `ZCX_ABAPGIT_EXCEPTION` | Exception class | ✅ Yes |

### abapGit Installation

abapGit must be installed via one of:
1. **abapGit standalone** - https://github.com/abapGit/abapGit
2. **abapGit Developer Edition** - SAP standard delivery (S/4HANA 2020+)

### Feature Detection

VSP detects abapGit availability via `GetFeatures` tool:
```go
// pkg/adt/features.go
func (p *FeatureProber) ProbeAbapGit(ctx context.Context) FeatureStatus {
    // Checks for ZCL_ABAPGIT_OBJECTS class existence
}
```

If abapGit is not installed, Git tools will return an error.

## Implementation Details

### New Files

| File | Lines | Purpose |
|------|-------|---------|
| `pkg/adt/git.go` | 95 | Go WebSocket client for Git operations |
| `embedded/abap/zcl_vsp_git_service.clas.abap` | 397 | ABAP Git service implementation |

### Modified Files

| File | Changes |
|------|---------|
| `embedded/abap/zcl_vsp_apc_handler.clas.abap` | Fixed nested JSON parsing, registered git service |
| `internal/mcp/server.go` | Added GitTypes/GitExport tools (+100 lines) |

### MCP Tools

#### GitTypes

```
Description: Get list of supported abapGit object types
Parameters: none
Returns: List of 158 object types (CLAS, INTF, PROG, DDLS, etc.)
```

#### GitExport

```
Description: Export ABAP objects as abapGit-compatible ZIP
Parameters:
  - packages: Comma-separated package names (e.g., "$ZRAY,$TMP")
  - objects: JSON array of objects [{"type":"CLAS","name":"ZCL_TEST"}]
  - include_subpackages: Include subpackages (default: true)
Returns: Base64-encoded ZIP with abapGit file structure
```

### Tool Group

Git tools can be disabled via `--disabled-groups G`:

```bash
vsp --disabled-groups G  # Disables GitTypes, GitExport
```

## Supported Object Types (158)

The following object types can be exported via GitExport:

```
ACID, AIFC, AMSD, APIS, APLO, AQBG, AQQU, AQSG, AREA, ASFC,
AUTH, AVAR, AVAS, BDEF, BGQC, CDBO, CHAR, CHDO, CHKC, CHKO,
CHKV, CLAS, CMOD, CMPT, CUS0, CUS1, CUS2, DCLS, DDLS, DDLX,
DEVC, DIAL, DOCT, DOCV, DOMA, DRAS, DRTY, DRUL, DSFD, DSFI,
DSYS, DTDC, DTEB, DTEL, ECAT, ECSD, ECSP, ECTC, ECTD, ECVO,
EEEC, ENHC, ENHO, ENHS, ENQU, ENSC, EVTB, FDT0, FORM, FTGL,
FUGR, FUGS, G4BA, G4BS, GSMP, HTTP, IAMU, IARP, IASP, IATU,
IAXU, IDOC, IEXT, INTF, IOBJ, IWMO, IWOM, IWPR, IWSG, IWSV,
IWVB, JOBD, MSAG, NONT, NROB, NSPC, OA2P, ODSO, OTGR, PARA,
PDTS, PERS, PINF, PRAG, PROG, RONT, SAJC, SAJT, SAMC, SAPC,
SCP1, SCVI, SFBF, SFBS, SFPF, SFPI, SFSW, SHI3, SHI5, SHI8,
SHLP, SHMA, SICF, SKTD, SMBC, SMIM, SMTG, SOBJ, SOD1, SOD2,
SOTS, SPLO, SPPF, SPRX, SQSC, SRFC, SRVB, SRVD, SSFO, SSST,
STVI, STYL, SUCU, SUSC, SUSH, SUSO, SXCI, SXSD, TABL, TOBJ,
TRAN, TTYP, TYPE, UCSA, UDMO, UENO, VCLS, VIEW, W3HT, W3MI,
WAPA, WDCA, WDCC, WDYA, WDYN, WEBI, XINX, XSLT
```

## Export File Format

Exported ZIP follows abapGit structure:

```
src/
├── zcl_example.clas.abap      # Class source code
├── zcl_example.clas.xml       # Class metadata (abapGit XML)
├── zif_example.intf.abap      # Interface source
├── zif_example.intf.xml       # Interface metadata
├── zprogram.prog.abap         # Program source
├── zprogram.prog.xml          # Program metadata
└── ...
```

### XML Metadata Format

```xml
<?xml version="1.0" encoding="utf-8"?>
<abapGit version="v1.0.0" serializer="LCL_OBJECT_CLAS" serializer_version="v1.0.0">
 <asx:abap xmlns:asx="http://www.sap.com/abapxml" version="1.0">
  <asx:values>
   <VSEOCLASS>
    <CLSNAME>ZCL_EXAMPLE</CLSNAME>
    <LANGU>E</LANGU>
    <DESCRIPT>Example Class</DESCRIPT>
    <STATE>1</STATE>
    <CLSCCINCL>X</CLSCCINCL>
    <FIXPT>X</FIXPT>
    <UNICODE>X</UNICODE>
   </VSEOCLASS>
  </asx:values>
 </asx:abap>
</abapGit>
```

## Usage Examples

### CLI Usage

```bash
# List supported object types
vsp git-types

# Export a package
vsp git-export --packages "$ZADT_VSP"

# Export multiple packages
vsp git-export --packages "$ZRAY,$TMP" --include-subpackages

# Export specific objects
vsp git-export --objects '[{"type":"CLAS","name":"ZCL_TEST"},{"type":"INTF","name":"ZIF_TEST"}]'
```

### MCP Tool Usage

```json
// GitTypes
{"name": "GitTypes"}

// GitExport by package
{
  "name": "GitExport",
  "arguments": {
    "packages": "$ZADT_VSP",
    "include_subpackages": true
  }
}

// GitExport by objects
{
  "name": "GitExport",
  "arguments": {
    "objects": "[{\"type\":\"CLAS\",\"name\":\"ZCL_TEST\"}]"
  }
}
```

## Test Results

### Export of $ZADT_VSP Package

```
Objects: 6
Files: 12
Source size: 109,637 bytes
ZIP size: 23,502 bytes (21.4% of original)

Files:
  src/zcl_vsp_amdp_service.clas.abap   (30,666 bytes)
  src/zcl_vsp_apc_handler.clas.abap    (7,747 bytes)
  src/zcl_vsp_debug_service.clas.abap  (35,303 bytes)
  src/zcl_vsp_git_service.clas.abap    (13,103 bytes)
  src/zcl_vsp_rfc_service.clas.abap    (19,207 bytes)
  src/zif_vsp_service.intf.abap        (704 bytes)
  + 6 .xml metadata files
```

## Known Limitations

1. **Import not implemented** - `GitImport` requires `ZCL_ABAPGIT_OBJECTS=>deserialize` which needs a virtual repository implementation
2. **Large packages may timeout** - For 200+ objects, consider chunked streaming (see Report 001)
3. **No progress feedback** - Single response, no streaming progress updates yet
4. **Base64 overhead** - 33% payload increase for binary transfer

## Future Enhancements

1. **GitImport** - Implement deserialize for importing from ZIP
2. **Chunked streaming** - For large exports (see architecture report)
3. **Progress updates** - WebSocket events during serialization
4. **Diff support** - Compare local vs SAP versions

## Related Documents

- `reports/2025-12-23-001-heavyweight-operations-architecture.md` - Chunked streaming design
- `reports/2025-12-22-003-websocket-abapgit-integration.md` - Original design document
- `embedded/abap/zcl_vsp_git_service.clas.abap` - ABAP implementation
- `pkg/adt/git.go` - Go client implementation
