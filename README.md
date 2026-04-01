# Vibing Steampunk Enterprise (`vsp`)

## Introduction

`vsp` is an MCP server and CLI for SAP ABAP Development Tools (ADT). It connects MCP-capable assistants (for example Claude Code, Copilot, Codex, and others) to SAP systems so they can read, analyze, and modify ABAP artifacts through controlled tool calls.

This repository is an enterprise-focused fork of the original [Vibing Steampunk](https://github.com/oisee/vibing-steampunk) project by [Alice Vinogradova](https://github.com/oisee). The fork focuses on MCP-first workflows, multi-system operations, and enterprise-ready configuration and safety controls.

### Available Tools

| Tool Group | What it does |
|---|---|
| `System` | System metadata and capability checks |
| `Read` | Read ABAP sources and repository objects |
| `Unified` | Unified object operations via a consolidated interface |
| `GrepSource` | Search patterns inside ABAP source content |
| `FileSource` | File-based source import and export workflows |
| `Analysis` | Call graph and code analysis helpers |
| `Transport` | Transport-related operations and controls |
| `Context` | Context gathering for agent workflows |
| `ATC` | ABAP Test Cockpit checks |
| `ClassInclude` | Class include and method-level handling |
| `CodeIntel` | Definition/reference style code intelligence |
| `CRUD` | Create/update/delete style repository actions |
| `Dev` | Developer operations (syntax checks, activation, tests) |
| `Dump` | Runtime dump diagnostics |
| `File` | File-centric utility operations |
| `Git` | abapGit-related integrations |
| `Grep` | Object-level grep and repository search |
| `Report` | Report execution and report metadata |
| `ServiceBinding` | RAP service binding operations |
| `SQLTrace` | SQL trace access and inspection |
| `Trace` | Trace listing and trace retrieval |
| `Workflow` | Workflow-oriented tool operations |
| `DebuggerLegacy` | Legacy debugger-compatible operations |

## Setup & Requirements

### Requirements

- Go `1.24` (from `go.mod`)
- Network access to SAP ADT endpoint(s)
- Optional for RFC sidecar usage: Java + Maven in `sidecar/jco-proxy`

### Install from Release (recommended)

```bash
curl -LO https://github.com/BurnerPat/vsp-enterprise/releases/latest/download/vsp-linux-amd64
chmod +x vsp-linux-amd64
mv vsp-linux-amd64 vsp
./vsp --version
```

### Build from Source

```bash
git clone https://github.com/BurnerPat/vsp-enterprise.git
cd vsp-enterprise
make build
./build/vsp --version
```

## Usage

### Start MCP server with direct connection flags

```bash
./build/vsp --url https://sap-host:44300 --user DEVELOPER --password secret --client 001
```

### Start MCP server from named profile

```bash
./build/vsp --system dev
```

### Start with multiple configured systems

```bash
./build/vsp --multi-system
```

### Useful CLI utilities

```bash
./build/vsp systems
./build/vsp config init
./build/vsp config show
./build/vsp config tools list
./build/vsp jco status
./build/vsp jco setup --system dev
```

## Configuration

`vsp` uses standard precedence:

1. CLI flags
2. Environment variables
3. `.env`
4. Config defaults

### Config files

- `.env` for default connection values (`SAP_URL`, `SAP_USER`, `SAP_PASSWORD`, `SAP_CLIENT`, ...)
- `.vsp.json` for named systems and permissions

`vsp` searches for `.vsp.json` in this order:

1. `.vsp.json`
2. `.vsp/systems.json`
3. `~/.vsp.json`
4. `~/.vsp/systems.json`

### Minimal `.vsp.json`

```json
{
  "default": "dev",
  "systems": {
    "dev": {
      "url": "https://sap-dev.example.com:44300",
      "user": "DEVELOPER",
      "client": "001"
    },
    "prod": {
      "url": "https://sap-prod.example.com:44300",
      "user": "READONLY",
      "client": "100",
      "read_only": true
    }
  }
}
```

### Hierarchical permissions

Tool permissions can be configured at three levels:

1. Root `permissions`
2. `system_classes.<name>.permissions`
3. `systems.<id>.permissions`

`tools` supports wildcard matching (`*`).

- `deny_tools_by_default: true` means deny unless explicitly allowed.
- `deny_tools_by_default: false` means allow unless explicitly denied.

```json
{
  "default": "dev",
  "permissions": {
    "deny_tools_by_default": true,
    "tools": {
      "Get*": true,
      "GetSystemInfo": true
    }
  },
  "system_classes": {
    "dev_test": {
      "permissions": {
        "deny_tools_by_default": false,
        "tools": {
          "Delete*": false,
          "Write*": true
        }
      }
    }
  },
  "systems": {
    "dev": {
      "url": "https://sap-dev.example.com:44300",
      "user": "DEVELOPER",
      "client": "001",
      "system_class": "dev_test"
    },
    "prod": {
      "url": "https://sap-prod.example.com:44300",
      "user": "READONLY",
      "client": "100",
      "permissions": {
        "deny_tools_by_default": true,
        "tools": {
          "Get*": true,
          "Write*": false,
          "Delete*": false
        }
      }
    }
  }
}
```

For per-system passwords with named systems, use environment variables such as `VSP_DEV_PASSWORD`.

## Contribution

Contributions are welcome.

Preferred contribution flow:

```bash
git checkout -b feature/my-change
make fmt
make test
make build
```

Then open a pull request with:

- clear problem statement
- implementation summary
- test notes

For project background and architecture context, see `docs/architecture.md` and `FORK.md`.

## License

MIT License. See `LICENSE`.

This repository is a fork of the original [Vibing Steampunk](https://github.com/oisee/vibing-steampunk) project created by [Alice Vinogradova](https://github.com/oisee). Credit for the original work and contributors remains with the upstream project.
