# Fork Information: vsp-enterprise

This repository is a fork of the original **[Vibing Steampunk](https://github.com/oisee/vibing-steampunk)** project, created and maintained by **[Alice Vinogradova](https://github.com/oisee)**.

## What is This Fork?

This fork focuses on creating an **enterprise-ready MCP server** optimized for:

- **Multiple SAP systems** - First-class multi-system configuration and management
- **Enterprise authentication** - Browser-based SSO, cookie-based auth, enterprise proxies
- **Maintainability** - Simplified codebase, reduced complexity, easier to extend
- **Production readiness** - Removed experimental features, focused on stability and core functionality

## Credits & Original Repository

**Original Project**: [github.com/oisee/vibing-steampunk](https://github.com/oisee/vibing-steampunk)  
**Original Creator**: Alice Vinogradova ([@oisee](https://github.com/oisee))  
**License**: MIT (see [LICENSE](LICENSE))

This fork maintains the MIT license and gives full credit to Alice Vinogradova and all original contributors to the Vibing Steampunk project.

## Major Changes From Original

### Features Removed (Intentional Simplification)

The following features were removed to focus on core enterprise MCP server functionality:

| Feature | Reason for Removal |
|---------|-------------------|
| **Lua Scripting Engine** | Replaced with simpler Go-based API; reduced maintenance burden |
| **CLI DevOps Surface** (`export`, `search`, `source` commands) | CLI focused on config/setup; SDK users can access core library directly |
| **Workflow Implementation** | Moved to task-specific handlers; removed generic workflow engine |
| **LSP (Language Server Protocol)** | Complex to maintain; ABAP LSP provided via separate tool chain |
| **Server-side ABAP Component** | Removed legacy server-side ABAP modules; clients use ADT REST API only |
| **Legacy Cache Implementations** | Replaced with modern context compression via dependency analysis |
| **Example Code** | Outdated; users should start with [Reviewer Guide](docs/reviewer-guide.md) |

### Architecture Improvements

| Improvement | Impact |
|-------------|--------|
| **Unified Configuration** | Merged `ResolvedConfig` and `SystemResolvedConfig` into `GlobalConfig` and `SystemConfig` |
| **Explicit System Lifecycle** | New `System.Connect()`, `System.Start()`, `System.Shutdown()` methods for clear state management |
| **Multi-System Support** | First-class support for multiple SAP systems in a single server instance |
| **Simplified Tool Registration** | Colocated tool definitions with handlers; metadata-driven registration |
| **Enhanced Feature Probing** | Improved safety network for detecting system capabilities (abapGit, RAP, AMDP, UI5) |
| **Browser-based SSO** | Added support for Kerberos, SAML, Keycloak via browser integration |

### What's Kept (Core Strengths)

✅ **Hyperfocused Mode** — 1 universal `SAP()` tool with ~99.5% token reduction  
✅ **Context Compression** — Built-in ABAP parser; 7–30x compression on dependencies  
✅ **Method-Level Surgery** — Read/edit individual methods without full-class round-trips  
✅ **AI Debugger** — Breakpoints, step execution, variable inspection  
✅ **ABAP Introspection** — Find definition, find references, code completion, call graphs  
✅ **Code Analysis** — Syntax checking, activation, unit test execution  
✅ **RAP/OData Support** — CDS view creation, service definitions, OData binding  
✅ **Multi-System Configuration** — Profiles for multiple SAP systems  

## Migration Guide

### If You're Using the Original Project

This fork maintains API compatibility with the original where possible, but some command-line interfaces and configuration methods have changed:

**Configuration**: The original project supported both single-system and multi-system modes. This fork is **multi-system-first**, but single-system configuration is still fully supported:

```bash
# Single system (still works)
./vsp --url http://host:50000 --user admin --password secret

# Multi-system with profiles (new)
./vsp config add-system dev --url http://dev:50000 --user admin --password secret
./vsp config add-system prod --url http://prod:50000 --user admin --password secret
./vsp --systems dev,prod
```

**CLI Commands**: The `vsp source`, `vsp export`, `vsp search` commands have been removed. Instead:
- Use `SAP(action="read", target=...)` for source reading
- Use `SAP(action="write", target=...)` for source writing
- Use `SAP(action="search", target=...)` for searching

### Building and Running

```bash
# Build
go build -o vsp ./cmd/vsp

# Run MCP server
./vsp

# Run CLI for configuration
./vsp config add-system production --url http://host:50000 --user user --password pass
./vsp config show

# Run with specific system
export SAP_SYSTEMS=production
./vsp
```

## Codebase Structure (Post-Fork)

```
vsp-enterprise/
├── cmd/vsp/                    # CLI entry point
│   ├── main.go                # Main server entry
│   ├── cli.go                 # CLI command routing
│   ├── config_cmd.go          # Config management commands
│   └── jco.go                 # JCo setup wizard
├── internal/mcp/
│   ├── server.go              # MCP server + tool registration
│   ├── system.go              # System lifecycle management
│   ├── router.go              # Multi-system request routing
│   └── tools/                 # Tool implementations
├── internal/config/           # Configuration (GlobalConfig, SystemConfig)
│   ├── config.go
│   ├── merge.go
│   └── validation.go
├── pkg/adt/                   # ADT client library (unchanged from original)
│   ├── client.go
│   ├── crud.go
│   ├── workflows.go
│   ├── debugger.go
│   └── ...
└── go.mod
```

## Development

This fork welcomes contributions! Areas for improvement:

1. **Enterprise Authentication**: Extend browser-based SSO with additional auth providers
2. **Performance**: Optimize context compression for very large classes
3. **Safety**: Expand the feature probing and safety network
4. **Testing**: Add integration tests for multi-system scenarios
5. **Documentation**: Improve examples and deployment guides

See [CLAUDE.md](docs/CLAUDE.md) and [ARCHITECTURE.md](docs/architecture.md) for development guidelines.

## License

MIT License — See [LICENSE](LICENSE) for details.

**Copyright (c) 2025-2026 Alice Vinogradova and contributors**

This fork maintains the original MIT license and honors all contributions from the Vibing Steampunk project.

---

**Questions or Issues?**

- Original project: [github.com/oisee/vibing-steampunk](https://github.com/oisee/vibing-steampunk)
- This fork: [github.com/BurnerPat/vsp-enterprise](https://github.com/BurnerPat/vsp-enterprise)

