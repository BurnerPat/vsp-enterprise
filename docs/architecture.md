# vsp Architecture

> **Note**: This is the enterprise fork of [Vibing Steampunk](https://github.com/oisee/vibing-steampunk).
> This architecture document describes the current state after refactoring for multi-system support and enterprise readiness.
> See [FORK.md](../FORK.md) for details on changes from the original project.

## High-Level Architecture

```mermaid
flowchart TB
    subgraph Clients["MCP Clients"]
        CC[Claude Code]
        CD[Claude Desktop]
        Other[Other MCP Clients]
    end

    subgraph VSP["vsp — Go Binary"]
        direction TB

        subgraph Entry["Entry Points"]
            MCP[MCP Server<br/>JSON-RPC / stdio]
            CLI[CLI Mode<br/>config · systems · jco]
        end

        subgraph Core["internal/mcp/"]
            direction LR
            Hyper[Hyperfocused Mode<br/>1 Universal Tool]
            Focused[Focused Mode<br/>~79 Tools]
            Expert[Expert Mode<br/>~120 Tools]
        end

        subgraph Safety["Safety Layer"]
            RO[Read-Only Mode]
            PF[Package Filter]
            TF[Transport Filter]
            OF[Operation Filter]
            TE[Transportable<br/>Edit Guard]
        end

        subgraph ADTLib["pkg/adt/ — ADT Client Library"]
            direction TB
            subgraph Read["Read"]
                client[client.go<br/>Search · Get*]
                cds[cds.go<br/>CDS Dependencies]
            end
            subgraph Write["Write"]
                crud[crud.go<br/>Lock · Create · Update · Delete]
                workflows[workflows.go<br/>GetSource · WriteSource · Grep*]
            end
            subgraph DevTools["DevTools"]
                devtools[devtools.go<br/>Syntax · Activate · Tests · ATC]
                codeintel[codeintel.go<br/>FindDef · FindRefs · Completion]
            end
            subgraph Debug["Debugger"]
                dbg[debugger.go<br/>Breakpoints · Listen · Attach · Step]
                amdp[amdp_debugger.go<br/>HANA SQLScript Debug]
            end
            subgraph Extras["Extras"]
                ui5[ui5.go<br/>UI5/BSP Apps]
                features[features.go<br/>System Probing]
                git[git.go<br/>abapGit Export]
                reports[reports.go<br/>Report Execution]
            end
        end

        subgraph Transport["Transport Layer"]
            HTTP[http.go<br/>CSRF · Sessions · Auth]
            WS[WebSocket Client<br/>ZADT_VSP APC]
            RFC[RFC Transport<br/>via JCo Sidecar]
            STDIO[STDIO Transport<br/>JCo Sidecar Alt]
        end

        subgraph Packages["Supporting Packages"]
            Config[pkg/config/<br/>Multi-System Profiles]
            CtxComp[pkg/ctxcomp/<br/>Context Compression]
        end

        subgraph Embedded["Embedded Assets"]
            Deps[embedded/deps/<br/>jco-proxy.jar]
        end

        subgraph Sidecar["sidecar/jco-proxy/ — Java"]
            JCo[JCo Proxy<br/>RFC Connectivity]
        end
    end

    subgraph SAP["SAP System"]
        ADT[ADT REST API<br/>/sap/bc/adt/*]
        APC[ZADT_VSP<br/>WebSocket APC]
        HANA[HANA DB<br/>AMDP Debug]
        RFCSAP[RFC Interface]
    end

    CC & CD & Other <-->|JSON-RPC / stdio| MCP
    CLI --> Core
    MCP --> Core
    Core --> Safety
    Safety --> ADTLib
    ADTLib --> Transport
    HTTP <-->|HTTPS| ADT
    WS <-->|WebSocket| APC
    amdp <-->|WebSocket| HANA
    RFC <-->|HTTP/STDIO| JCo
    JCo <-->|RFC| RFCSAP
```

## Request Flow

```mermaid
sequenceDiagram
    participant Client as MCP Client
    participant Server as MCP Server
    participant Safety as Safety Layer
    participant ADT as ADT Client
    participant HTTP as HTTP Transport
    participant SAP as SAP System

    Client->>Server: Tool Call (JSON-RPC)
    Server->>Safety: Check permissions

    alt Blocked
        Safety-->>Server: Denied (read-only / package / operation)
        Server-->>Client: Error result
    else Allowed
        Safety->>ADT: Execute operation
        ADT->>HTTP: HTTP request
        HTTP->>HTTP: Add CSRF token + cookies
        HTTP->>SAP: HTTPS / WebSocket
        SAP-->>HTTP: Response
        HTTP-->>ADT: Parsed response
        ADT-->>Server: Result
        Server-->>Client: Tool result (JSON)
    end
```

## Write Operation Flow (EditSource)

```mermaid
sequenceDiagram
    participant AI as AI Assistant
    participant VSP as vsp
    participant SAP as SAP System

    AI->>VSP: EditSource(url, old_string, new_string)

    VSP->>SAP: GET source
    SAP-->>VSP: Current source code

    VSP->>VSP: Find & replace (uniqueness check)

    VSP->>SAP: POST syntax check
    SAP-->>VSP: OK / Errors

    alt Syntax Errors
        VSP-->>AI: Error (no changes saved)
    else Syntax OK
        VSP->>SAP: POST lock
        VSP->>SAP: PUT source
        VSP->>SAP: POST unlock
        VSP->>SAP: POST activate
        VSP-->>AI: Success
    end
```

## Tool Categories

Focused mode exposes ~79 essential tools. Expert mode adds ~40 more granular/legacy tools (~120 total).
Hyperfocused mode collapses everything into 1 universal SAP tool.
Tool groups can be individually disabled via `--disabled-groups` flags or `.vsp.json` config.

```mermaid
flowchart LR
    subgraph Search["Search (3)"]
        SO[SearchObject]
        GO[GrepObjects]
        GP[GrepPackages]
    end

    subgraph Read["Read (10)"]
        GS[GetSource]
        GT[GetTable]
        GTC[GetTableContents]
        RQ[RunQuery]
        GPk[GetPackage]
        GFG[GetFunctionGroup]
        GCD[GetCDSDependencies]
        GCI[GetClassInfo]
        GMs[GetMessages]
        CS[CompareSource]
    end

    subgraph Write["Write (5)"]
        WS[WriteSource]
        ES[EditSource]
        IF[ImportFromFile]
        EF[ExportToFile]
        MO[MoveObject]
    end

    subgraph Dev["Dev (5)"]
        SC[SyntaxCheck]
        UT[RunUnitTests]
        ATC[RunATCCheck]
        LO[LockObject]
        UO[UnlockObject]
    end

    subgraph Intel["Intelligence (2)"]
        FD[FindDefinition]
        FR[FindReferences]
    end

    subgraph Debug["Debugger (6)"]
        DL[Listen]
        DA[Attach]
        DD[Detach]
        DS[Step]
        DGS[GetStack]
        DGV[GetVariables]
    end

    subgraph AMDP["AMDP Debugger (7)"]
        AL[AMDPListen]
        AA[AMDPAttach]
        AD[AMDPDetach]
        AS[AMDPStep]
        AGS[AMDPGetStack]
        AGV[AMDPGetVariables]
        ABP[AMDPBreakpoints]
    end

    subgraph System["System (5)"]
        SI[GetSystemInfo]
        IC[GetInstalledComponents]
        CG[GetCallGraph]
        OS[GetObjectStructure]
        GF[GetFeatures]
    end

    subgraph Diag["Diagnostics (6)"]
        LD[ListDumps]
        GD[GetDump]
        LT[ListTraces]
        GTr[GetTrace]
        STS[GetSQLTraceState]
        LST[ListSQLTraces]
    end

    subgraph CTS["Transport (5)"]
        CT[CreateTransport]
        RT[ReleaseTransport]
        LTr2[ListTransports]
        GTr2[GetTransport]
        ATT[AddToTransport]
    end

    subgraph Git["Git (2)"]
        GiT[GitTypes]
        GiE[GitExport]
    end

    subgraph Reports["Reports (4)"]
        RR[RunReport]
        GV[GetVariants]
        GTE[GetTextElements]
        STE[SetTextElements]
    end

    subgraph Install["Install (3)"]
        IV[InstallZADTVSP]
        IA[InstallAbapGit]
        LDp[ListDependencies]
    end
```

## Triple Transport: HTTP + WebSocket + RFC

```mermaid
flowchart LR
    subgraph VSP["vsp"]
        HTTP[HTTP Client<br/>pkg/adt/http.go]
        WS[WebSocket Client<br/>pkg/adt/websocket.go]
        RFC[RFC Transport<br/>pkg/adt/rfc_transport.go<br/>pkg/adt/stdio_transport.go]
    end

    subgraph Sidecar["Java Sidecar"]
        JCo[jco-proxy.jar<br/>SAP JCo Library]
    end

    subgraph SAP["SAP System"]
        ADT[ADT REST API<br/>/sap/bc/adt/*]
        APC[ZADT_VSP APC Handler<br/>/sap/bc/apc/ws/zadt_vsp]
        RFCSAP[SAP RFC Interface<br/>SADT_REST_RFC_ENDPOINT]
    end

    HTTP -->|"CRUD · Search · Read<br/>Syntax · Activate · Debug"| ADT
    WS -->|"RFC Calls · Breakpoints<br/>Git Export · Reports<br/>AMDP Debug"| APC
    RFC -->|"HTTP or STDIO"| JCo
    JCo -->|"SAP JCo RFC"| RFCSAP

    subgraph WSServices["WebSocket Domains"]
        direction TB
        WSRFC[rfc — Function Calls]
        BRK[debug — Breakpoints]
        GIT[git — abapGit Export]
        RPT[report — Report Execution]
        HLP[help — ABAP Documentation]
    end

    APC --- WSServices
```

## Package Structure

```
vibing-steampunk/
├── cmd/vsp/                    # CLI entry point (cobra/viper)
│   ├── main.go                 #   Flags, env vars, auth, MCP server startup
│   ├── cli.go                  #   Multi-system profile management
│   ├── config_cmd.go           #   config init/show/mcp-to-vsp/tools subcommands
│   └── jco.go                  #   JCo sidecar setup/status commands
│
├── internal/mcp/               # MCP protocol layer (32 files)
│   ├── server.go               #   MCP server core, client init, multi-system
│   ├── tools_register.go       #   Mode-aware tool registration (~120 tools)
│   ├── tools_focused.go        #   Focused mode whitelist (~79 tools)
│   ├── tools_groups.go         #   Disableable tool groups (U/T/H/D/C/G/R/X)
│   ├── tools_aliases.go        #   Short alias names (gs→GetSource, etc.)
│   ├── multi_system.go         #   Multi-system connection management
│   └── handlers_*.go           #   Per-domain tool handlers (20+ files)
│
├── pkg/adt/                    # ADT client library (core, ~54 files)
│   ├── client.go               #   Read operations + search
│   ├── crud.go                 #   Lock / create / update / delete
│   ├── devtools.go             #   Syntax check, activate, unit tests, ATC
│   ├── codeintel.go            #   Find definition, references, completion
│   ├── workflows.go            #   High-level: GetSource, WriteSource, Grep*
│   ├── debugger.go             #   External ABAP debugger (HTTP + WebSocket)
│   ├── amdp_debugger.go        #   HANA/AMDP SQLScript debugger
│   ├── ui5.go                  #   UI5/Fiori BSP management
│   ├── cds.go                  #   CDS view dependency analysis
│   ├── git.go                  #   abapGit export via WebSocket
│   ├── reports.go              #   Report execution, variants, spool output
│   ├── safety.go               #   Read-only, package/op filtering
│   ├── features.go             #   System capability detection
│   ├── http.go                 #   HTTP transport (CSRF, sessions, auth)
│   ├── transport.go            #   Transport interface abstraction
│   ├── rfc_transport.go        #   RFC proxy transport (via JCo sidecar HTTP)
│   ├── stdio_transport.go      #   RFC proxy transport (via JCo sidecar STDIO)
│   ├── sidecar.go              #   JCo sidecar process lifecycle management
│   ├── jco_discovery.go        #   JCo library detection (platform-specific)
│   ├── browser_auth.go         #   Browser-based SSO (Kerberos/SAML/Keycloak)
│   ├── landscape.go            #   SAP UI Landscape XML parsing (SNC/SSO)
│   ├── recorder.go             #   Execution recording for time-travel debug
│   ├── history.go              #   Tool call history tracking
│   ├── websocket.go            #   WebSocket client (ZADT_VSP APC)
│   ├── websocket_base.go       #   WebSocket base types
│   ├── websocket_types.go      #   WebSocket protocol types
│   ├── websocket_rfc.go        #   RFC-over-WebSocket operations
│   ├── websocket_debug.go      #   Debug-over-WebSocket operations
│   ├── xml.go                  #   ADT XML type definitions
│   └── config.go               #   Client configuration with functional options
│
├── pkg/config/                 # Multi-system configuration
│   └── systems.go              #   .vsp.json profile management, granular tool config
│
├── pkg/ctxcomp/                # Context compression
│   └── *.go                    #   Dependency analysis, contract extraction for AI
│
├── sidecar/jco-proxy/          # Java JCo proxy (Maven project)
│   ├── pom.xml                 #   Maven build (SAP JCo dependency)
│   └── src/main/               #   Java source: RFC↔HTTP/STDIO bridge
│
├── embedded/                   # Assets embedded in Go binary
│   └── deps/                   #   jco-proxy.jar (compiled sidecar)
│
├── abap/                       # ABAP source artifacts
│   └── src/                    #   ZADT_VSP and related ABAP objects
│
└── docs/                       # Documentation
    ├── architecture.md         #   This file
    ├── DSL.md                  #   DSL reference
    └── adr/                    #   Architecture Decision Records
```

## Authentication

```mermaid
flowchart TD
    Start[Request] --> Auth{Auth Method?}

    Auth -->|Basic| Basic[Username + Password<br/>--user / --password]
    Auth -->|Cookie File| CFile[Netscape Format<br/>--cookie-file]
    Auth -->|Cookie String| CStr[Key=Value pairs<br/>--cookie-string]
    Auth -->|Browser SSO| Browser[Chromium Automation<br/>--browser-auth<br/>Kerberos / SAML / Keycloak]
    Auth -->|SNC| SNC[Secure Network Comm<br/>--snc + --sysid<br/>via JCo Sidecar]

    Basic --> CSRF[Fetch CSRF Token]
    CFile --> CSRF
    CStr --> CSRF
    Browser --> CookieSave[Save Cookies to File]
    CookieSave --> CSRF

    CSRF --> Session[Stateful Session<br/>Cookie Jar]
    SNC --> JCo[JCo Sidecar<br/>RFC Transport]
    Session --> SAP[SAP ADT API]
    JCo --> SAP
```

## Safety System

```mermaid
flowchart TD
    Request[Tool Call] --> RO{Read-Only?}

    RO -->|Yes, Write Op| Block1[BLOCKED]
    RO -->|No / Read Op| SQL{Free SQL<br/>Blocked?}

    SQL -->|Yes, RunQuery| Block2[BLOCKED]
    SQL -->|No| Ops{Operation<br/>Allowed?}

    Ops -->|Disallowed| Block3[BLOCKED]
    Ops -->|Allowed| Pkg{Package<br/>Allowed?}

    Pkg -->|Outside whitelist| Block4[BLOCKED]
    Pkg -->|In whitelist| TE{Transportable<br/>Package?}

    TE -->|Yes, not enabled| Block5[BLOCKED]
    TE -->|No / Enabled| TR{Transport<br/>Allowed?}

    TR -->|Outside whitelist| Block6[BLOCKED]
    TR -->|In whitelist| OK[EXECUTE]
```

## Testing

312 unit test functions across 34 test files. No SAP system required — all unit tests use mock HTTP transports.

```
go test ./...                                    # All unit tests
go test -tags=integration ./pkg/adt/             # Integration tests (requires SAP)
```

## Design Decisions

1. **Single Binary**: No runtime dependencies (except optional JCo sidecar for RFC mode)
2. **Functional Options**: Flexible client configuration via `adt.WithXxx()` pattern
3. **Stateful HTTP**: Required for CRUD operations (lock handles, CSRF tokens, session cookies)
4. **Triple Transport**: HTTP for ADT REST, WebSocket for ZADT_VSP APC, RFC via JCo sidecar
5. **Mode-Aware Tools**: Focused (default), Expert (all), Hyperfocused (single universal tool)
6. **Granular Tool Control**: Groups can be disabled (`--disabled-groups`), individual tools via `.vsp.json`
7. **Multi-System**: Connect to multiple SAP systems from one instance via profiles
8. **Safety by Default**: Read-only mode, package/transport whitelists, operation filtering
9. **Java Sidecar**: Embedded `jco-proxy.jar` for RFC connectivity without CGo

## Build Targets

Cross-compilation via Makefile and GoReleaser:

| OS | Architectures |
|----|---------------|
| Linux | amd64, arm64, 386, arm |
| macOS | amd64, arm64 (Apple Silicon) |
| Windows | amd64, arm64, 386 |

```
make build          # Current platform
make build-all      # Common targets (linux-amd64, darwin-arm64, windows-amd64)
make build-all-all  # All 9 targets
```
