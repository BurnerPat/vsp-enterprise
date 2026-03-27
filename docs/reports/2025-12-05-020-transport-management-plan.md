# Transport Management Implementation Plan

**Date:** 2025-12-05
**Report ID:** 020
**Subject:** CTS Transport Management tools with safety controls

---

## Executive Summary

Implement 5 transport management tools using the ADT CTS REST API with comprehensive safety controls for transport restrictions. This enables AI assistants to manage transports programmatically while maintaining enterprise governance.

| Aspect | Details |
|--------|---------|
| **Tools** | 5 (ListTransports, GetTransport, CreateTransport, ReleaseTransport, DeleteTransport) |
| **Safety Controls** | Transport whitelist/patterns, read-only mode, tool group disablement |
| **Estimated LOC** | ~600 (client) + ~300 (handlers) + ~200 (safety) + ~150 (tests) |
| **Tool Group** | `C` (CTS/Transports) |

---

## 1. API Endpoints

### 1.1 Verified Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/sap/bc/adt/cts/transportrequests?user={user}` | GET | List modifiable transports |
| `/sap/bc/adt/cts/transportrequests/{number}` | GET | Get transport details |
| `/sap/bc/adt/cts/transports` | POST | Create transport |
| `/sap/bc/adt/cts/transportrequests/{number}` | DELETE | Delete transport |
| `/sap/bc/adt/cts/transportrequests/{number}/newreleasejobs` | POST | Release transport |
| `/sap/bc/adt/cts/transportchecks` | POST | Check transport requirements |

### 1.2 Response Structure

```xml
<tm:root>
  <tm:request tm:number="A4HK900094" tm:owner="DEVELOPER"
              tm:desc="Description" tm:type="K" tm:status="R">
    <tm:abap_object tm:pgmid="R3TR" tm:type="PROG" tm:name="ZPROGRAM"/>
    <tm:task tm:number="A4HK900095" tm:owner="DEVELOPER">
      <tm:abap_object .../>
    </tm:task>
  </tm:request>
</tm:root>
```

---

## 2. Tool Definitions

### 2.1 ListTransports (Read)

**Purpose:** List transport requests for a user

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `user` | string | No | Username (default: current user, `*` for all) |
| `status` | string | No | Filter: `modifiable`, `released`, `all` |

**Response:** Array of transport summaries

### 2.2 GetTransport (Read)

**Purpose:** Get detailed transport information including objects and tasks

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `transport` | string | Yes | Transport number (e.g., `A4HK900094`) |

**Response:** Full transport details with objects, tasks, logs links

### 2.3 CreateTransport (Write)

**Purpose:** Create new transport request

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `description` | string | Yes | Transport description |
| `package` | string | Yes | Target package (DEVCLASS) |
| `transport_layer` | string | No | Transport layer |
| `type` | string | No | Type: `workbench` (K) or `customizing` (W) |

**Response:** Created transport number

### 2.4 ReleaseTransport (Write)

**Purpose:** Release a transport request

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `transport` | string | Yes | Transport number |
| `ignore_locks` | bool | No | Release with ignored locks |
| `skip_atc` | bool | No | Skip ATC checks |

**Response:** Release status and any warnings

### 2.5 DeleteTransport (Write)

**Purpose:** Delete a transport request

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `transport` | string | Yes | Transport number |

**Response:** Success/failure status

---

## 3. Safety & Exposure Design

### 3.1 Tool Group Integration

Add transport tools to the tool group system:

```go
// Tool group "C" for CTS/Transports
toolGroups := map[string][]string{
    "5": { /* UI5 tools */ },
    "T": { /* Test tools */ },
    "H": { /* HANA/AMDP debugger */ },
    "D": { /* ABAP debugger */ },
    "C": { // NEW: CTS/Transport tools
        "ListTransports", "GetTransport",
        "CreateTransport", "ReleaseTransport", "DeleteTransport",
    },
}
```

**Disable with:** `--disabled-groups C` or `SAP_DISABLED_GROUPS=C`

### 3.2 Transport Restrictions

New safety configuration options:

```go
// pkg/adt/safety.go additions
type SafetyConfig struct {
    // Existing fields...
    ReadOnly           bool
    BlockFreeSQL       bool
    AllowedOps         string
    DisallowedOps      string
    AllowedPackages    []string

    // NEW: Transport restrictions
    TransportReadOnly     bool     // Block all write operations on transports
    AllowedTransports     []string // Whitelist specific transports
    AllowedTransportMasks []string // Patterns like "A4HK*", "DEV*"
}
```

### 3.3 Configuration Options

| Flag | Env Variable | Description |
|------|--------------|-------------|
| `--transport-read-only` | `SAP_TRANSPORT_READ_ONLY` | Only allow ListTransports, GetTransport |
| `--allowed-transports` | `SAP_ALLOWED_TRANSPORTS` | Comma-separated whitelist |

**Examples:**
```bash
# Only allow reading transport information
./vsp --transport-read-only

# Only allow operations on specific transports
./vsp --allowed-transports "A4HK900110,A4HK900111"

# Allow pattern-based transports (masks)
./vsp --allowed-transports "A4HK*,DEV*"

# Combine with tool group disablement
./vsp --disabled-groups CH  # Disable CTS and HANA debugger
```

### 3.4 Focused vs Expert Mode

| Tool | Focused | Expert | Rationale |
|------|---------|--------|-----------|
| ListTransports | ✅ | ✅ | Read-only, safe |
| GetTransport | ✅ | ✅ | Read-only, safe |
| CreateTransport | ❌ | ✅ | Write operation |
| ReleaseTransport | ❌ | ✅ | Irreversible action |
| DeleteTransport | ❌ | ✅ | Destructive action |

### 3.5 Safety Check Flow

```
┌─────────────────────────────────────────────────────────┐
│                  Transport Operation                     │
└─────────────────────────┬───────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│ 1. Check Tool Group Enabled?                            │
│    if "C" in DisabledGroups → BLOCK                     │
└─────────────────────────┬───────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│ 2. Check TransportReadOnly?                             │
│    if true && isWriteOp → BLOCK                         │
└─────────────────────────┬───────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│ 3. Check AllowedTransports?                             │
│    if configured && !matches(transport) → BLOCK         │
└─────────────────────────┬───────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│ 4. Execute Operation                                    │
└─────────────────────────────────────────────────────────┘
```

---

## 4. Implementation Plan

### 4.1 File Structure

```
pkg/adt/
├── transports.go       # NEW: Transport API client methods
├── transports_test.go  # NEW: Unit tests
├── safety.go           # UPDATE: Add transport safety checks
├── config.go           # UPDATE: Add transport config options

internal/mcp/
└── server.go           # UPDATE: Add 5 transport tool handlers

cmd/vsp/
└── main.go             # UPDATE: Add CLI flags
```

### 4.2 Implementation Steps

**Phase 1: Client Methods (~600 LOC)**
1. Add XML types for transport responses
2. Implement `ListTransports(ctx, user string) ([]TransportSummary, error)`
3. Implement `GetTransport(ctx, number string) (*Transport, error)`
4. Implement `CreateTransport(ctx, opts CreateTransportOptions) (string, error)`
5. Implement `ReleaseTransport(ctx, number string, opts ReleaseOptions) error`
6. Implement `DeleteTransport(ctx, number string) error`

**Phase 2: Safety Configuration (~200 LOC)**
1. Add `TransportReadOnly`, `AllowedTransports` to SafetyConfig
2. Add `IsTransportAllowed(number string) bool` method
3. Add CLI flags and env vars to config

**Phase 3: MCP Handlers (~300 LOC)**
1. Register 5 tools in `registerTools()`
2. Add handler cases with safety checks
3. Update focused/expert mode tool lists

**Phase 4: Tests (~150 LOC)**
1. Unit tests for transport parsing
2. Unit tests for safety checks
3. Integration tests with real SAP system

---

## 5. Data Types

### 5.1 Go Types

```go
// TransportSummary for list operations
type TransportSummary struct {
    Number      string `json:"number"`
    Owner       string `json:"owner"`
    Description string `json:"description"`
    Type        string `json:"type"`        // K=Workbench, W=Customizing
    Status      string `json:"status"`      // D=Modifiable, R=Released
    StatusText  string `json:"statusText"`
    Target      string `json:"target"`
    ChangedAt   string `json:"changedAt"`
}

// Transport for detailed view
type Transport struct {
    TransportSummary
    Tasks   []TransportTask   `json:"tasks,omitempty"`
    Objects []TransportObject `json:"objects,omitempty"`
}

// TransportTask represents a task within a transport
type TransportTask struct {
    Number      string            `json:"number"`
    Owner       string            `json:"owner"`
    Description string            `json:"description"`
    Type        string            `json:"type"`
    Status      string            `json:"status"`
    Objects     []TransportObject `json:"objects,omitempty"`
}

// TransportObject represents an object in a transport
type TransportObject struct {
    PgmID    string `json:"pgmid"`    // R3TR, LIMU, CORR
    Type     string `json:"type"`     // PROG, CLAS, DEVC, etc.
    Name     string `json:"name"`
    WBType   string `json:"wbtype"`   // PROG/P, CLAS/OC, etc.
    Info     string `json:"info"`     // "Program", "Class", etc.
    Position int    `json:"position"`
}

// CreateTransportOptions for creating transports
type CreateTransportOptions struct {
    Description    string
    Package        string
    TransportLayer string
    Type           string // "workbench" or "customizing"
}

// ReleaseOptions for releasing transports
type ReleaseOptions struct {
    IgnoreLocks bool
    SkipATC     bool
}
```

---

## 6. Tool Descriptions for MCP

### ListTransports
```
List transport requests. Returns modifiable transports for a user.

Parameters:
- user (string, optional): Username to list transports for. Default: current user. Use "*" for all users.

Returns: Array of transport summaries with number, owner, description, status.
```

### GetTransport
```
Get detailed transport information including objects and tasks.

Parameters:
- transport (string, required): Transport request number (e.g., "A4HK900094")

Returns: Full transport details including all objects, nested tasks, and metadata.
```

### CreateTransport
```
Create a new transport request.

Parameters:
- description (string, required): Transport description
- package (string, required): Target package (DEVCLASS)
- transport_layer (string, optional): Transport layer
- type (string, optional): "workbench" (default) or "customizing"

Returns: Created transport number.
```

### ReleaseTransport
```
Release a transport request. This action is IRREVERSIBLE.

Parameters:
- transport (string, required): Transport request number
- ignore_locks (bool, optional): Release even with locked objects
- skip_atc (bool, optional): Skip ATC quality checks

Returns: Release status and any warnings.
```

### DeleteTransport
```
Delete a transport request. Only modifiable transports can be deleted.

Parameters:
- transport (string, required): Transport request number

Returns: Success confirmation.
```

---

## 7. Testing Strategy

### 7.1 Unit Tests
- XML parsing for all response types
- Safety config validation
- Transport mask matching (wildcards)

### 7.2 Integration Tests
```go
func TestIntegration_ListTransports(t *testing.T)
func TestIntegration_GetTransport(t *testing.T)
func TestIntegration_CreateAndDeleteTransport(t *testing.T)
// Note: ReleaseTransport is not tested automatically (irreversible)
```

---

## 8. Success Criteria

- [ ] All 5 tools implemented and working
- [ ] Safety controls enforced (read-only, whitelist)
- [ ] Tool group "C" can be disabled
- [ ] Focused mode includes only read tools
- [ ] Unit tests passing
- [ ] Integration tests passing
- [ ] Documentation updated

---

## Appendix: Transport Status Codes

| Code | Status | Description |
|------|--------|-------------|
| D | Modifiable | Can be edited |
| L | Modifiable, protected | Protected from release |
| O | Release started | Release in progress |
| R | Released | Successfully released |
| N | Released (import not started) | Awaiting import |

## Appendix: Transport Types

| Code | Type | Description |
|------|------|-------------|
| K | Workbench | Development/correction |
| W | Customizing | Customizing request |
| T | Transport of copies | Copy transport |
| S | Development task | Task under K |
| R | Repair task | Repair task under K |
| X | Unclassified task | Unclassified |
