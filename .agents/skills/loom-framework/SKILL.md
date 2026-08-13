---
name: loom-framework
description: Develop and maintain the Loom framework repository itself. Use for changes to DSL implementation, expr, codegen, HTTP/gRPC/JSON-RPC transports, OpenAPI generation, framework runtime packages, fixtures, internal architecture, or contributor workflows. Do not use for ordinary application-level Loom adoption.
---

# Maintain the Loom Framework

Use this skill for work inside the Loom framework repository: implementing or
reviewing DSL behavior, expression evaluation, generators, transport runtimes,
OpenAPI output, framework tests, fixtures, and contributor tooling.

Do not use this skill for ordinary consumer work in `design/`, `gen/`, service
implementations, or application bootstrap. Use the `loom` skill for those
tasks. This skill also owns the stronger delivery workflow for new framework
capabilities; there is no separate capability-maintenance skill.

## Start Here

1. Read the repository `AGENTS.md` completely.
2. Identify the owning layer before editing:
   - `dsl/` for public authoring functions
   - `expr/` for evaluated design semantics and validation
   - `codegen/` for shared generated Go models
   - `http/`, `grpc/`, or `jsonrpc/` for transport-specific behavior
   - `http/codegen/openapi/internal/ir` for OpenAPI analysis and contract
     decisions
   - `http/codegen/openapi/v3` for versioned OpenAPI rendering
3. Inspect nearby direct tests and meaningful fixtures.
4. Read the relevant public guide and the consumer `loom` skill only when the
   task changes how users design or operate a Loom service.

Fix the root framework boundary. Do not patch generated `gen/` output or add an
application-facing workaround when the framework owns the behavior.

## Adding or Changing Framework Capabilities

Apply this section when adding public DSL surface, generator or transport
behavior, OpenAPI or JSON Schema features, auth/session semantics, request or
response decoding, or another framework behavior intended to remove
application glue.

Before implementing, identify:

1. the repeated application workaround or concrete risk
2. the generic framework behavior that should replace it
3. a real consuming application or contract that proves the need
4. the framework layer that owns the behavior
5. what policy must remain application-owned

If those boundaries are unclear, do not add framework surface yet. Prefer the
narrowest behavior that removes the actual workaround, and do not special-case
one application's protocol flow when a generic transport or contract behavior
is the real fix.

A capability is complete only when:

- implementation lives at the correct ownership layer
- direct, generated-output, regression, and broader transport tests cover the
  relevant behavior
- coverage includes normal, edge, invalid, ambiguous, and still-rejected cases
  where applicable
- public docs and the consumer `loom` skill describe changed application usage
- internal architecture or workflow changes are recorded in this skill
- any live roadmap item reflects the delivered state
- the change is committed and pushed unless the user asks otherwise

For decoding capabilities, cover successful decoding, invalid input,
validation after decoding, and removal of the old custom workaround. For
contract output, pair structural assertions with rendered output and parser or
consumer validation.

## Architecture Boundaries

- Design DSL is the source of truth; evaluated semantics belong in `expr`.
- Shared analysis belongs in a shared IR rather than being independently
  rediscovered by transport renderers.
- Keep helpers package-private or under an `internal` package when only one
  codegen area needs them.
- Use NameScope helpers (`GoTypeRef`, `GoFullTypeRef`, `GoTypeName`) for emitted
  Go type references. Never construct type syntax by string concatenation.
- Let Loom determine pointer/value semantics except at explicit transport
  validation boundaries.
- Collapse pass-through wrappers that add no behavior.
- Use `codegen.Section`, `codegen.JenniferSection`, and `codegen.RawSection` for
  Loom-owned Go generation. Use neutral `.tmpl` assets only for non-Go output
  that genuinely benefits from templates.
- Never edit checked-in generated fixtures manually. Regenerate them through
  the same Loom path consumers use.

## OpenAPI Version Architecture

Loom emits OpenAPI 3.2.0 by default and supports a 3.1.1 compatibility target.
There is one DSL parser, one shared semantic IR, and one renderer.

- Gate emitted constructs by target version; do not fork parsers, IRs,
  generators, or canonical output paths.
- Render from the shared document model once. Apply 3.2-only transformations in
  `applyOpenAPI32`; for the 3.1 target, remove only incompatible members in
  `filterOpenAPI31` while preserving shared structures.
- Keep isolated version choices explicit and local. Do not introduce a generic
  version router until multiple real render shapes require that abstraction.
- Keep shared JSON Schema features such as `contentSchema` when they are valid
  in both targets. Compatibility filtering must remove only version-specific
  members.
- Start OpenAPI contract changes in `http/codegen/openapi/internal/ir`; keep the
  `v3` package focused on rendering IR-owned decisions.
- OpenAPI JSON uses Go 1.27 `encoding/json/v2` with deterministic ordering.
  Preserve compact output by default and configured prefix/indent behavior.
- Treat stable schema names, canonical `operationId`, reusable component
  identity, and extension output as public framework contracts.

The public feature inventory and metadata usage belong in
`docs/dsl-reference.md` and the consumer `loom` skill. Internal constructor,
filter, and serialization rules belong here.

## Transport Invariants

- HTTP and JSON-RPC SSE handlers defer committing the stream until the first
  frame, except for the raw JSON-RPC `events/stream` GET listener, which opens
  eagerly so clients can observe readiness.
- Protocol errors must use event types compatible with the relevant client
  contract.
- Keep WebSocket lifecycle behavior in the shared runtime wrapper; generated
  endpoints should not grow independent read/write/close loops.
- Preserve the generated public `ServeHTTP` middleware and policy chain when
  adding JSON-RPC dispatch branches.
- Generated transport observability must remain dependency-free and must never
  emit bodies, params, tool arguments, credentials, or result payloads.
- Keep HTTP, gRPC, and JSON-RPC semantics distinct unless a deliberately shared
  transport core owns the behavior.

## Verification Strategy

For framework and codegen bugs, add the failing direct test first. Pair the test
layers that apply:

1. direct seam tests for `expr`, shared IR, generator sections, or runtime logic
2. structural assertions for emitted contract/code shape
3. rendered or golden coverage
4. compile-after-generation coverage
5. relevant integration and adversarial transport cases

OpenAPI changes require:

- direct structural assertions
- meaningful rendered specimens under `http/codegen/testdata`
- `libopenapi` parsing
- Redocly linting
- downstream consumer smoke generation where applicable

SSE changes require coverage for:

- happy paths
- pre-stream endpoint failures
- protocol-level error event compatibility
- compile-after-generation
- branch-specific connection timing

Use repository gates rather than duplicating their logic:

```bash
go fmt ./...
make lint
make test
make openapi-contract          # OpenAPI work
make generated-code-quality    # generated Go/output work
make integration-test          # transport behavior
./check.sh --full              # full repository verification
```

## Local and Remote Source Modes

HTTP and JSON-RPC temp-module tests share a worktree-local source selector:

```bash
make loom-local
make loom-remote
make loom-status
```

- Use local mode, or set `LOOM_DIR=/absolute/path`, while testing unpushed
  framework changes.
- Use remote mode for pushed-commit reproducibility.
- The source-mode marker is untracked worktree state; do not commit it.
- A pre-push check cannot fetch the commit being pushed for the first time. Use
  local mode for that push, then restore remote mode afterward.
- External temp modules must pin a pushed GitHub commit in reproducible CI.

Temp-copy regeneration tests for checked-in fixtures are intentionally
local-only and should rewrite the copied fixture's `replace` directive to the
current repository root before invoking `loom gen`.

## Fixture Contracts

Treat these as regression surfaces, not demos:

- `http/integration_tests/fixtures/ticktock`
- `jsonrpc/integration_tests/fixtures/ticktock`

The checked-in JSON-RPC ticktock fixture covers POST-initiated SSE. The raw
`events/stream` GET listener is covered by
`jsonrpc/integration_tests/tests/sse_get_listener_test.go`, which regenerates a
temporary listener variant. Extend the test matching the branch being changed.

## Documentation Ownership

- Public user behavior belongs under `docs/`.
- Update the consumer `loom` skill only when users must design, generate, wire,
  or reason about their applications differently.
- Framework architecture, contributor workflow, source-mode behavior, and
  internal invariants belong in this skill or `AGENTS.md`, not the consumer
  skill.
- Active framework plans belong under `roadmap/`; remove obsolete temporary
  plans when their durable guidance has moved to docs or skills.
- Update `references/repo-map.md` when framework ownership or navigation
  changes.

## Completion

Before handoff:

1. format touched Go files with the selected Go 1.27 toolchain
2. run proportionate direct and broad gates
3. regenerate every intentionally affected checked-in fixture
4. inspect `git diff --check` and `git status --short`
5. preserve unrelated and worktree-local files
6. commit and push unless the user explicitly asks to stop short

When the change affects public usage, report both the implementation location
and the updated user-facing guide. For internal-only work, do not burden the
consumer skill with contributor detail.

## References

- `references/repo-map.md`
- repository `AGENTS.md`
- `roadmap/`
- `docs/`
