# Loom-MCP References Map

Use the Loom and Loom-MCP framework sources as the authoritative references when the bundled guide fragments are insufficient.

## Preferred Source Locations

- First choice: sibling `loom` and `loom-mcp` checkouts in the workspace, if present
- Otherwise: the checked-out `github.com/CaliLuke/loom` module in the local Go module cache
- Otherwise: the checked-out `github.com/CaliLuke/loom-mcp` module in the local Go module cache
- Otherwise: framework source available to the agent in the current environment

## What To Inspect

- `README.md`: install + quick-start commands (`loom gen`, `loom example`)
- `dsl/`: DSL surface (`Service`, `Method`, `HTTP`, `GRPC`, `JSONRPC`, security)
- `codegen/`: generator internals and conventions
- `http/`, `grpc/`, `jsonrpc/`: transport packages and patterns
- `middleware/`: reusable middleware components
- `expr/`: design expression model
- In Loom-MCP, inspect `dsl/`, `codegen/`, `runtime/agent/`, and `runtime/mcp/` for agent contracts, generated MCP adapters, orchestration, and session behavior.

## Skill-Bundled User Guides

- `SKILL.md`: primary routing index for bundled Loom docs
- `references/user-guides/<topic>/...`: task-sized fragments for quick lookup
- `references/user-guides/*.md`: full transcripts kept as fallbacks when a fragment is insufficient

## Suggested Lookup Flow

1. If the task needs end-user doc steps, open the matching fragment listed in `SKILL.md`.
2. Check the available Loom source tree for transport DSL and generation behavior.
3. Check the available Loom-MCP source tree for agent DSL, MCP codegen, and runtime behavior.
4. Confirm transport behavior in Loom's `http/`, `grpc/`, or `jsonrpc/` packages.
5. Apply changes in user code by editing DSL first, then regenerating with `loom gen`.

## Useful Search Commands

```bash
# search DSL and transport implementations
rg -n "Service\(|Method\(|HTTP\(|GRPC\(|JSONRPC\(" <loom-source>/dsl <loom-source>/expr

# search generation/runtime behavior
rg -n "Generate|Mount|Decode|Encode|OpenAPI" <loom-source>/codegen <loom-source>/http <loom-source>/grpc <loom-source>/jsonrpc

# search Loom-MCP adapter and runtime behavior
rg -n "MCPAdapterOptions|SDKServerOptions|ToolsetRegistration|RegisterAgent|SessionPrincipal" <loom-mcp-source>/codegen <loom-mcp-source>/runtime
```
