---
name: loom-mcp
description: Build and maintain Loom-MCP services in Go. Use this skill when a user mentions Loom-MCP, Loom-MCP DSL, generated `gen/` transport code, OpenAPI/proto generation, service implementation after DSL changes, or refactoring a project with a `design` package.
---
# Loom-MCP

Use this skill for Loom-MCP work only.

## Non-Negotiables

- Treat `design/*.go` as the source of truth.
- Regenerate after every design change with the generator command provided by the checked-out Loom-MCP repo.
- Never hand-edit generated `gen/` files.
- Implement business logic in non-generated files.
- Use Go import paths for Loom-MCP generator commands, not filesystem paths.
- Commit generated code; do not rely on CI to regenerate it.

## Runtime Gotchas

- SSE server streams do not expose a generated `Open()` hook. Loom writes SSE headers on the first `Send`, so idle subscriptions that must complete the HTTP handshake before the first business event need a non-generated transport/runtime flush path or an explicit bootstrap event in the contract.
- Never repair SSE or cookie behavior by editing generated files. Keep the fix in `design/*.go` or non-generated transport/runtime code.
- For responses that need multiple `Set-Cookie` headers, prefer idiomatic framework cookies in the DSL. If a flow still depends on raw cookie header strings, write them from non-generated transport code against the live `http.ResponseWriter` rather than patching generated encoders.

## Default Workflow

1. Detect the Loom service surface: `go.mod`, `design/`, DSL imports, or `gen/` folders.
2. Edit the DSL in `design/`.
3. Run the checked-in Loom-MCP generator for `<module>/design`.
4. Run the repo's example or scaffold command only when new starter files are explicitly wanted.
5. Implement logic outside `gen/`.
6. Verify with `go mod tidy` and project tests.

## Command Reminders

- Prefer the Loom-MCP repo's current generator entrypoints over stale upstream `goa` command examples.
- Use import paths like `<module>/design`, not filesystem paths like `./design`.

## References

- Framework/source map: `references/repo-map.md`
- Prefer the small fragments under `references/user-guides/<topic>/` first.
- Use the top-level full transcripts under `references/user-guides/*.md` only when a fragment is insufficient.
- For framework/runtime internals, inspect the Loom source tree described in `references/repo-map.md`.

## Fragment Routing

### Quickstart

- `references/user-guides/quickstart/installation.md`: install the CLI and verify `PATH`
- `references/user-guides/quickstart/first-service.md`: create the project, define your agent DSL, and generate Loom-MCP code
- `references/user-guides/quickstart/run-and-test.md`: run a stub planner/executor and understand the plan/execute loop
- `references/user-guides/quickstart/streaming.md`: inspect execution events in real time with stream sinks
- `references/user-guides/quickstart/validation.md`: define schema constraints and use automatic retry hints
- `references/user-guides/quickstart/real-llm.md`: connect OpenAI or Claude via model clients
- `references/user-guides/quickstart/composition.md`: compose agents via agent-as-tool patterns

### DSL

- `references/user-guides/dsl/data-modeling.md`: primitives, `Type`, arrays/maps, validation, formats, examples
- `references/user-guides/dsl/services-and-methods.md`: `API`, `Service`, `Method`, payload/result, streaming basics
- `references/user-guides/dsl/http-mapping.md`: `HTTP`, paths, params, body/header mapping, files
- `references/user-guides/dsl/grpc-and-security.md`: `GRPC`, metadata/message mapping, security schemes
- `references/user-guides/dsl-reference.md`: complete Loom-MCP DSL reference (agents, toolsets, policies, MCP)

### Runtime

- `references/user-guides/runtime.md`: agent lifecycle, run tree, streaming, policies, model clients, and engine behavior
- `references/user-guides/toolsets.md`: toolset types, validation, retry hints, bounded results, server data, injected fields, and executors
- `references/user-guides/composition.md`: agent-as-tool composition, passthrough, run trees, child streaming, and run-tree-aware UIs
- `references/user-guides/mcp-integration.md`: MCP toolset declaration, caller wiring, and tool-result error flow
- `references/user-guides/memory.md`: transcript ledger model, message parts, sessions/runs, and memory vs runlog stores
- `references/user-guides/internal-tool-registry.md`: clustered tool registry design, provider integration, health checks, and gRPC discovery/invocation
- `references/user-guides/testing.md`: testing agents/planners/tools and common runtime troubleshooting patterns
- `references/user-guides/production.md`: production deployment patterns for Loom-MCP (Temporal, rate limiting, stream profiles, reminders)

### Code Generation

- `references/user-guides/codegen/commands-and-workflow.md`: generator and scaffold workflow
- `references/user-guides/codegen/generated-layout.md`: what lands in `gen/`, `cmd/`, HTTP, and gRPC packages
- `references/user-guides/codegen/customization.md`: metadata knobs for type/package/proto/OpenAPI generation

### HTTP

- `references/user-guides/http/routing.md`: routing, prefixes, params, wildcards, parent services
- `references/user-guides/http/streaming.md`: WebSocket and SSE behavior and examples
- `references/user-guides/http/content-negotiation-cors-static.md`: encoders, CORS, and static files

### gRPC

- `references/user-guides/grpc/service-design.md`: service structure, field numbering, metadata, headers/trailers
- `references/user-guides/grpc/streaming-and-errors.md`: streaming modes and status-code mapping
- `references/user-guides/grpc/implementation.md`: server/client setup and protobuf generation

### Errors

- `references/user-guides/errors/definitions-and-types.md`: API/service/method errors, `ErrorResult`, custom types
- `references/user-guides/errors/transport-and-formatting.md`: HTTP/gRPC mappings, formatter patterns, tests

### Interceptors

- `references/user-guides/interceptors/loom-interceptors.md`: generated wrapper model, accessor contracts, ordering
- `references/user-guides/interceptors/http-and-grpc-middleware.md`: HTTP middleware and gRPC interceptor patterns

### Production
- `references/user-guides/production/index.md`: production topic index (model rate limiting, prompt overrides, Temporal, streaming UI, reminders)
- `references/user-guides/production/model-rate-limiting.md`: adaptive AIMD model rate limiting and runtime integration
- `references/user-guides/production/prompt-overrides.md`: prompt override scope resolution and Mongo-backed storage
- `references/user-guides/production/temporal-setup.md`: Temporal installation, runtime wiring, activity options, and failure handling
- `references/user-guides/production/streaming-ui.md`: stream event types, sink contracts, profiles, and Pulse/WebSocket/SSE wiring
- `references/user-guides/production/system-reminders.md`: system reminder model, priorities, and planner/tool reminders

### Full-Transcript-to-Fragment Index

Use this when you need to jump directly to a bounded chunk of Loom-MCP content without browsing the full transcript tree.

- `references/user-guides/quickstart.md`
- `references/user-guides/dsl-reference.md`
- `references/user-guides/runtime.md`
- `references/user-guides/composition.md`
- `references/user-guides/toolsets.md`
- `references/user-guides/mcp-integration.md`
- `references/user-guides/memory.md`
- `references/user-guides/internal-tool-registry.md`
- `references/user-guides/testing.md`
- `references/user-guides/production.md`
- `references/user-guides/production/index.md`
- `references/user-guides/production/model-rate-limiting.md`
- `references/user-guides/production/prompt-overrides.md`
- `references/user-guides/production/temporal-setup.md`
- `references/user-guides/production/streaming-ui.md`
- `references/user-guides/production/system-reminders.md`
- `references/user-guides/code-generation.md`
- `references/user-guides/http-guide.md`
- `references/user-guides/grpc-guide.md`
- `references/user-guides/error-handling.md`
- `references/user-guides/interceptors.md`

## Selection Rules

- Start with the fragment that matches the user's immediate task.
- Load additional fragments only if the first one is insufficient.
- Prefer `references/repo-map.md` and the Loom source tree for framework internals or runtime behavior.

## Full Transcripts

Full transcripts are grouped by document boundary:

#### Core design and architecture
- `references/user-guides/quickstart.md`
- `references/user-guides/dsl-reference.md`
- `references/user-guides/runtime.md`
- `references/user-guides/composition.md`
- `references/user-guides/toolsets.md`

#### Service surface and transport
- `references/user-guides/code-generation.md`
- `references/user-guides/http-guide.md`
- `references/user-guides/grpc-guide.md`
- `references/user-guides/error-handling.md`
- `references/user-guides/interceptors.md`

#### Execution and operations
- `references/user-guides/mcp-integration.md`
- `references/user-guides/memory.md`
- `references/user-guides/internal-tool-registry.md`
- `references/user-guides/testing.md`
- `references/user-guides/production.md`
