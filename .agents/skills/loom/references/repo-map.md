# Loom References Map

Use Loom framework source as the authoritative reference for DSL and runtime behavior when the bundled guide fragments are insufficient.

## Preferred Source Locations

- First choice: a vendored or sibling `references/loom` clone in the workspace, if present
- Otherwise: the checked-out `github.com/CaliLuke/loom` module in the local Go module cache
- Otherwise: the Loom repository source available to the agent in the current environment

## What To Inspect

- `README.md`: install + quick-start commands (`loom gen`, `loom example`)
- `dsl/`: DSL surface (`Service`, `Method`, `HTTP`, `GRPC`, `JSONRPC`, security)
- `codegen/`: generator internals and conventions
- `http/`, `grpc/`, `jsonrpc/`: transport packages and patterns
- `middleware/`: reusable middleware components
- `expr/`: design expression model

## Canonical User Guides

- `SKILL.md`: primary routing index and Loom-specific contract rules
- `docs/`: maintained user guides for quickstart, DSL, code generation, HTTP,
  gRPC, errors, interceptors, and production
- `jsonrpc/README.md`: maintained JSON-RPC user and architecture documentation

## Suggested Lookup Flow

1. If the task needs end-user instructions, open the matching canonical guide
   listed in `SKILL.md`.
2. Check the available Loom source tree for DSL and generation behavior.
3. Confirm transport behavior in `http/`, `grpc/`, or `jsonrpc/`.
4. Apply changes in user code by editing DSL first, then regenerating with `loom gen`.

## Useful Search Commands

```bash
# search DSL and transport implementations
rg -n "Service\(|Method\(|HTTP\(|GRPC\(|JSONRPC\(" <loom-source>/dsl <loom-source>/expr

# search generation/runtime behavior
rg -n "Generate|Mount|Decode|Encode|OpenAPI" <loom-source>/codegen <loom-source>/http <loom-source>/grpc <loom-source>/jsonrpc
```
