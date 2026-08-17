# Loom Framework Repository Map

Use this map when maintaining the framework itself. Public usage instructions
belong in the `loom` skill and the canonical guides under `docs/`.

## Ownership

- `dsl/`: public DSL functions and DSL-time validation entry points
- `expr/`: evaluated design model, preparation, finalization, and semantic
  validation
- `codegen/`: shared generator model, service code, sections, plugins, and CLI
  generation
- `http/`: HTTP runtime, middleware, client/server generation, integrations,
  and OpenAPI generation
- `http/codegen/openapi/internal/ir`: shared OpenAPI analysis and contract
  decisions
- `http/codegen/openapi/v3`: OpenAPI 3.2 rendering and 3.1 compatibility
  projection
- `grpc/`: gRPC runtime-owned metadata, status, observation, and stream
  completion lifecycle; protobuf/transport generation; and typed error mapping
- `jsonrpc/`: JSON-RPC runtime-owned envelope, batch, notification, SSE, and
  WebSocket lifecycle; typed transport generation; and integrations
- `observability/`: framework-owned tracing, metrics, logging, and transport
  event contracts
- `internal/`: repository-private support packages and release/source tooling
- `scripts/`, `Makefile`, `check.sh`: canonical verification and contributor
  workflows
- `docs/`: public user documentation
- `roadmap/`: active framework plans

## Lookup Flow

1. Start from the public DSL or runtime behavior being changed.
2. Trace its expression ownership and validation.
3. Inspect the shared generator or transport IR before renderer-specific code.
4. Find direct tests at the ownership seam.
5. Find the meaningful checked-in fixture or integration path.
6. Update public docs only if the consuming workflow changes.

## Useful Searches

```bash
rg -n "Service\(|Method\(|HTTP\(|GRPC\(|JSONRPC\(" dsl expr
rg -n "Generate|Mount|Decode|Encode|OpenAPI" codegen http grpc jsonrpc
rg -n "openapi:|versionedConstructor|versionRouter" http/codegen/openapi dsl expr
rg -n "loom-local|loom-remote|LOOM_DIR" Makefile scripts http jsonrpc
```
