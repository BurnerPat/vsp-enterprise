# Non-MCP Features Inventory

**Date:** 2026-03-27  
**Purpose:** Identify all features in vsp that are not part of the MCP server and not required for the MCP server to function.

---

## Summary

vsp has grown from an MCP server into a multi-mode ABAP development platform. The MCP server (`vsp` with no subcommand, reading stdio) is one of **six operation modes**. The majority of code in the repository supports features that are independent of the MCP server.

### What the MCP server needs

The MCP server mode (`runServer` in `main.go`) requires:

- `internal/mcp/` — MCP protocol handling, tool registration, tool dispatch
- `pkg/adt/` — ADT client library (HTTP transport, CRUD, code intel, debugger, etc.)
- `embedded/deps/` — JCo proxy JAR (for RFC connection mode)
- `pkg/config/` — Multi-system configuration (used by `--multi-system` MCP mode)
- `pkg/ctxcomp/` — Context compression (used by `GetSource` and `GetContext` MCP tools)
- `cmd/vsp/main.go` — Root command, flag parsing, `runServer`
- `sidecar/` — JCo proxy (used by RFC connection mode, which MCP server supports)

Everything else listed below is **not required** for the MCP server to function.

---

## Already Removed

- **CLI Mode** (`cmd/vsp/devops.go`, parts of `cmd/vsp/cli.go`) — Removed. All direct terminal commands (`vsp search`, `vsp source`, `vsp export`, `vsp test`, `vsp atc`, `vsp deploy`, `vsp transport`, `vsp install`) have been deleted. The `vsp systems init/list` and `vsp jco` commands were kept.
- **Lua Scripting Engine** (`pkg/scripting/`, `cmd/vsp/lua.go`) — Removed. Lua REPL, script execution, 40+ ADT bindings. Also removed `gopher-lua` dependency.
- **LSP Server** (`internal/lsp/`, `cmd/vsp/lsp.go`) — Removed. Language Server Protocol for editor integration.
- **Cache Infrastructure** (`pkg/cache/`) — Removed. In-memory + SQLite caching for graph analysis. Never wired into MCP. Also removed `go-sqlite3` CGO dependency.
- **Interactive CLI Debugger** (`cmd/vsp/debug.go`) — Removed. Command-line REPL debugger for ABAP.
- **DSL / Workflow Engine** (`pkg/dsl/`, `cmd/vsp/workflow.go`) — Removed. YAML workflow runner, fluent Go API, batch pipelines.
- **Install / Deploy Tools** (`internal/mcp/handlers_install.go`, `handlers_deploy.go`, `embedded/abap/`) — Removed. InstallZADTVSP, InstallAbapGit, InstallDummyTest, ListDependencies, DeployZip tools. Also removed embedded ABAP source files and abapGit ZIPs.

---

## Remaining Non-MCP Features

### 1. Configuration Management CLI

**Files:** `cmd/vsp/config_cmd.go`

Configuration file generation and management commands:

| Command | Description |
|---------|-------------|
| `vsp config init` | Generate example `.env`, `.vsp.json`, `.mcp.json` files |
| `vsp config show` | Display effective configuration |
| `vsp config mcp-to-vsp` | Import systems from `.mcp.json` to `.vsp.json` |
| `vsp config vsp-to-mcp` | Export `.vsp.json` to `.mcp.json` format |
| `vsp config tools enable/disable` | Per-tool visibility control |

These are convenience commands for managing config files. The MCP server reads its config from flags/env/`.env` directly.

---

### 2. Documentation & Articles

**Files not required for MCP server operation:**

| Path | Content |
|------|---------|
| `reports/` | 30+ research reports, design documents, analysis |
| `articles/` | Blog posts / published articles |
| `docs/cli-agents/` | MCP config examples for various AI agents |
| `docs/plans/` | Phase planning documents |
| `docs/DSL.md` | DSL documentation |
| `docs/architecture.md` | Architecture documentation |
| `docs/reviewer-guide.md` | Code review guide |
| `docs/adr/` | Architecture Decision Records |
| `contexts/` | Design context documents |
| `ARCHITECTURE.md` | High-level architecture |
| `VISION.md` | Project vision |
| `ROADMAP.md` | Feature roadmap |
| `MCP_USAGE.md` | MCP usage guide |
| `README_TOOLS.md` | Tool reference |
| `CHANGELOG.md` | Release changelog |
| `cliff.toml` | git-cliff changelog generator config |
| `scripts/` | Utility scripts (Lua sync, etc.) |

---

### 3. Development-Time ABAP Sources

**Files:** `abap/src/`, `scripts/sync-embedded.lua`

Source-of-truth ABAP files and sync scripts. These are development-time artifacts only — the `embedded/abap/` directory and Install tools have been removed.

---

## Feature Dependency Matrix

| Feature | Package(s) | Used by MCP? | Standalone? |
|---------|-----------|:------------:|:-----------:|
| MCP Server | `internal/mcp/` | **Yes** | — |
| ADT Client | `pkg/adt/` | **Yes** | — |
| Embedded JCo JAR | `embedded/deps/` | **Yes** (RFC mode) | — |
| Multi-system Config | `pkg/config/` | **Yes** (multi-system mode) | Also used by CLI |
| ~~CLI Commands~~ | ~~`cmd/vsp/cli.go`, `devops.go`~~ | ~~No~~ | ~~Removed~~ |
| ~~Lua Scripting~~ | ~~`pkg/scripting/`, `cmd/vsp/lua.go`~~ | ~~No~~ | ~~Removed~~ |
| ~~LSP Server~~ | ~~`internal/lsp/`, `cmd/vsp/lsp.go`~~ | ~~No~~ | ~~Removed~~ |
| ~~Cache Infrastructure~~ | ~~`pkg/cache/`~~ | ~~No~~ | ~~Removed~~ |
| ~~CLI Debugger~~ | ~~`cmd/vsp/debug.go`~~ | ~~No~~ | ~~Removed~~ |
| ~~DSL/Workflows~~ | ~~`pkg/dsl/`, `cmd/vsp/workflow.go`~~ | ~~No~~ | ~~Removed~~ |
| ~~Install/Deploy~~ | ~~`handlers_install.go`, `handlers_deploy.go`, `embedded/abap/`~~ | ~~No~~ | ~~Removed~~ |
| Context Compression | `pkg/ctxcomp/` | **Yes** | — |
| Config CLI | `cmd/vsp/config_cmd.go` | No | Yes |
| Java Sidecar | `sidecar/jco-proxy/` | **Yes** (RFC mode) | — |
| JCo Setup Wizard | `cmd/vsp/jco.go` | **Yes** (RFC setup) | — |

---

## Impact Assessment

Removing all remaining non-MCP features would eliminate:

- **1 cmd file:** `config_cmd.go`
- **Supporting directories:** `examples/`, `scripts/`, `abap/src/`

What remains is a focused MCP server binary: `cmd/vsp/main.go` + `cmd/vsp/cli.go` (systems commands) + `cmd/vsp/jco.go` → `internal/mcp/` → `pkg/adt/` → `pkg/ctxcomp/` → `embedded/` with optional `sidecar/` and `pkg/config/`.
