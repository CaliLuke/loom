# Concrete HTTP and JSON-RPC server member namespaces

`ServerMembers.tla` models #700's shared HTTP/JSON-RPC analysis owner. Fixed
Server members are reserved before endpoint and file-handler fields. JSON-RPC
WebSocket fields are copied into another receiver, so their allocation reserves
both receivers. Distinct handler and streaming-endpoint roles share that scope;
a method named `readEndpoint` must not reuse another method's derived field.

`current` reproduces unchecked spelling collisions. `server_only` rejects a
partial repair that ignores the copied WebSocket receiver. `owned` reserves the
whole receiving scope and allocates every role once. Keyword specimens abstract
valid identifier normalization as reservation; the model does not prove Goify or
NameScope, import qualification or Go syntax. Candidate suffix spellings abstract
NameScope choices, including numeric file-field suffixes. Ordinary preferences
stay exact.

Run with Corretto 25 and pinned TLA+ tools 1.7.4 (TLC 2.19), jar SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 \
  -metadir /tmp/loom-server-members-owned -config owned.cfg ServerMembers.tla
```

Use separate external metadirs for all configurations. Final readiness results:

| Configuration | Exit | Generated / distinct | Result |
| --- | --- | --- | --- |
| current | 12 | 7 / 7 | Unchecked member collision |
| server_only | 12 | 15 / 15 | Copied receiver collision remains |
| owned | 0 | 30 / 24 | All six bounded specimens pass |

The earlier model included a gRPC example. Its native server has a distinct
analysis/rendering owner; the independently reproduced residual belongs in a
separate ticket. Removing that specimen does not claim gRPC is repaired.

## Correspondence and limits

`http/codegen.ServicesData.analyze` constructs shared HTTP/JSON-RPC endpoint and
file-server data; `finalizeServerMembers` finalizes recorded names
before returning that data, including FileData import reanalysis. HTTP Server
struct/init/Use/Mount/CORS routes and JSON-RPC Server/stream struct/init/dispatch/
receive operations must read those records. They must retain canonical service,
endpoint aggregate and protobuf names at all service-facing accesses.

Direct tests inspect the actual Server fields and JSON-RPC fields/methods in
both receiving types, including keywords. Generated compile/vet specimens cover
HTTP, JSON-RPC unary, server/bidirectional WebSocket, SSE and mixed HTTP/SSE, with
ordinary controls. Mounted race tests verify handler identity, CORS OPTIONS
routing, Use and
WebSocket endpoint-role selection; compilation alone does not prove correct
wiring. `ServerHandler` and `ServerEndpoint` are the shared role records. HTTP
AST checks compare fixed registrations to emitted declarations; JSON-RPC checks
derive adversarial names from actual receivers. File-only and import-alias
reanalysis controls cover analysis outside the finite model. See the complete
owner/consumer table and bounded
acceptance in [the plan](../../../roadmap/http-server-member-identity.md).

This is a finite namespace exploration, not full Go refinement or protocol
verification. It excludes clients, gRPC, package symbols, stream wrapper type
names and unrelated endpoint helper declarations. Keep those residuals separate.
After independent review, remove task-owned checker states/logs after preserving
concise results; keep model sources, configuration and run instructions.
