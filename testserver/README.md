# adt-testserver

A standalone HTTP server that emulates the SAP ADT REST API for integration testing.
It requires no SAP system, has no dependencies on the rest of the `vsp-enterprise`
codebase, and returns plausible XML/text responses with object-name placeholders.

## Quick start

```bash
# From the repo root
go run ./testserver

# With explicit credentials and port
go run ./testserver --sys-id DEV --client 100 --user myuser --password mypass --port 9090

# Pre-populate sources and query results from a YAML file
go run ./testserver --fixtures testserver/example.fixtures.yaml
```

The server prints its address and configured identity on startup:

```
ADT test server listening on http://localhost:8080  (sys=TST  client=001  user=developer)
```

## Flags

| Flag         | Default     | Description                              |
|--------------|-------------|------------------------------------------|
| `--sys-id`   | `TST`       | SAP System ID returned in T000/LOGSYS    |
| `--client`   | `001`       | SAP client number                        |
| `--user`     | `developer` | Expected Basic Auth username             |
| `--password` | `secret`    | Expected Basic Auth password             |
| `--port`     | `8080`      | TCP port to listen on                    |
| `--fixtures` | _(none)_    | Optional path to a YAML fixtures file    |

## Authentication & CSRF

Every request must carry valid Basic Auth credentials matching `--user`/`--password`.

Mutating requests (`POST`, `PUT`, `DELETE`) must include the header:

```
X-CSRF-Token: testserver-csrf-token-001
```

To obtain the token (mirroring the real ADT flow):

```bash
curl -s -X HEAD -u developer:secret http://localhost:8080/sap/bc/adt/core/discovery \
     -D - | grep -i x-csrf-token
```

## Endpoints

| Prefix | File | Notes |
|--------|------|-------|
| `/sap/bc/adt/core/discovery`, `/sap/bc/adt/discovery` | `endpoints/core.go` | CSRF token, service document |
| `/sap/bc/adt/programs/` | `endpoints/programs.go` | Programs + includes; GET/PUT source, lock/unlock, create, delete |
| `/sap/bc/adt/oo/` | `endpoints/oo.go` | Classes + interfaces; objectstructure included |
| `/sap/bc/adt/functions/` | `endpoints/functions.go` | Function groups + function modules |
| `/sap/bc/adt/ddic/`, `/sap/bc/adt/bo/behaviordefinitions/` | `endpoints/ddic.go` | Tables, structures, views, CDS DDL, SRVD, BDEF |
| `/sap/bc/adt/checkruns` | `endpoints/checkruns.go` | Always returns clean (zero-error) result |
| `/sap/bc/adt/activation` | `endpoints/activation.go` | Always returns success; inactive objects list is empty |
| `/sap/bc/adt/abapunit/testruns` | `endpoints/abapunit.go` | Returns a passing (no-test-class) unit test result |
| `/sap/bc/adt/atc/` | `endpoints/atc.go` | ATC run, worklist (empty findings), customizing |
| `/sap/bc/adt/repository/` | `endpoints/repository.go` | Object search, package node structure |
| `/sap/bc/adt/datapreview/` | `endpoints/datapreview.go` | DDIC + freestyle SQL; fixture or T000 fallback |
| `/sap/bc/adt/runtime/` | `endpoints/runtime.go` | Short dumps feed + detail, ABAP traces feed + analysis |

### Source CRUD

All source endpoints share the same behaviour:

- **GET** `…/source/main` — returns the fixture-loaded source if present, otherwise a
  hardcoded default snippet derived from the object name.
- **PUT** `…/source/main` — stores the request body in memory; subsequent GETs return
  the updated content.
- **POST** `…?_action=LOCK` — registers a deterministic lock handle and returns it in
  the ADT lock-result XML format.
- **POST** `…?_action=UNLOCK` — releases the lock.
- **POST** _(collection)_ — returns `201 Created` with a `Location` header.
- **DELETE** — releases any lock and returns `204 No Content`.

## Fixtures file

Supply a YAML file to pre-populate sources, locks, and data-preview results:

```yaml
# Exact key — returned verbatim for this specific path
sources:
  /sap/bc/adt/programs/programs/ZTEST/source/main: |
    REPORT ztest.

    START-OF-SELECTION.
      WRITE: / 'Hello from fixture'.

  # Pattern key — {name} matches any single path segment.
  # ${name} in the template body is replaced with the uppercased captured value.
  # Exact keys always take precedence over pattern keys.
  /sap/bc/adt/programs/programs/{name}/source/main: |
    REPORT ${name}.

    START-OF-SELECTION.
      WRITE: / '${name} loaded via pattern'.

  /sap/bc/adt/oo/classes/{name}/source/main: |
    CLASS ${name} DEFINITION PUBLIC FINAL CREATE PUBLIC.
      PUBLIC SECTION.
        METHODS run RETURNING VALUE(rv_result) TYPE string.
    ENDCLASS.
    CLASS ${name} IMPLEMENTATION.
      METHOD run.
        rv_result = '${name} works'.
      ENDMETHOD.
    ENDCLASS.

# Pre-locked objects (objectURL -> lock handle)
locks:
  /sap/bc/adt/programs/programs/ZLOCKED: MYHANDLE001

# Exact SQL query -> result rows (1-to-1 mapping)
datapreview:
  "SELECT * FROM T000 WHERE MANDT = '001'":
    - MANDT: "001"
      MTEXT: "Test Client"
      LOGSYS: "TSTCLNT001"
  "SELECT BNAME, USTYP FROM USR02 WHERE MANDT = '001'":
    - BNAME: "DEVELOPER"
      USTYP: "A"
    - BNAME: "TESTER"
      USTYP: "A"
```

Any query not matched by a fixture key falls back to:
1. A synthetic `T000` row using `--sys-id`/`--client` values, if the SQL contains `T000`.
2. An empty result set otherwise.

## Directory structure

```
testserver/
├── go.mod                      # module adt-testserver (standalone, no main-module imports)
├── go.sum
├── testserver.go               # main: flags, middleware, route registration
├── fixtures.go                 # YAML fixture loader
├── example.fixtures.yaml       # Example fixtures file
└── endpoints/
    ├── state.go                # Shared in-memory state (sources, locks, datapreview)
    ├── helpers.go              # Shared response helpers and lock-handle generator
    ├── core.go                 # /sap/bc/adt/core/discovery, /sap/bc/adt/discovery
    ├── programs.go             # /sap/bc/adt/programs/
    ├── oo.go                   # /sap/bc/adt/oo/ (classes + interfaces)
    ├── functions.go            # /sap/bc/adt/functions/
    ├── ddic.go                 # /sap/bc/adt/ddic/ + /sap/bc/adt/bo/behaviordefinitions/
    ├── checkruns.go            # /sap/bc/adt/checkruns
    ├── activation.go           # /sap/bc/adt/activation
    ├── abapunit.go             # /sap/bc/adt/abapunit/
    ├── atc.go                  # /sap/bc/adt/atc/
    ├── repository.go           # /sap/bc/adt/repository/
    ├── datapreview.go          # /sap/bc/adt/datapreview/
    └── runtime.go              # /sap/bc/adt/runtime/
```

