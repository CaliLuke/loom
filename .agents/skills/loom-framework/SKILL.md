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
   - `vet/` for evaluated-design and consuming-module adoption diagnostics
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
- Put design-semantic vet rules over the evaluated `expr` graph. Use Go source
  analysis only for module adoption facts that the graph cannot contain. A
  source analyzer must report when an active target-module package lacks
  complete syntax or type information, without repeating usable-package or
  dependency diagnostics. Route conflict analysis must use exact typed
  method/path constants from active source files and skip dynamic expressions
  rather than infer speculative runtime overlap.
- Classify heuristic vet rules as warnings and provide a scoped suppression.
- Keep `untyped-semantic-attribute` conservative: never warn on `Any` alone;
  allow name-only inference only for timestamp-style names such as `*_at`,
  require explicit UUID evidence for IDs, and ignore ambiguous descriptions
  that describe multiple possible shapes.
- Keep `service-not-mounted` opt-in through API metadata. Scan typed `Mount`
  calls only in the configured consumer packages, count calls in conditional
  branches, support multiple packages, and let service metadata suppress an
  intentional omission. Do not infer runtime reachability or enable the rule
  for library modules that omit the metadata.
- Fingerprint the evaluated design and output-affecting generation context in
  `gen/loom.json`. The fingerprint must ignore runtime-only evaluator state,
  all `loom:vet:*` metadata, timestamps, and filesystem paths, and must
  serialize maps deterministically. `loom vet` reports design skew separately
  from Loom version skew.
- Keep helpers package-private or under an `internal` package when only one
  codegen area needs them.
- Use NameScope helpers (`GoTypeRef`, `GoFullTypeRef`, `GoTypeName`) for emitted
  Go type references. Never construct type syntax by string concatenation.
- Let Loom determine pointer/value semantics except at explicit transport
  validation boundaries.
- Collapse pass-through wrappers that add no behavior.
- Keep generated transport code declarative. Move stable protocol execution to
  handwritten runtime packages, and pass typed handlers or callbacks from
  generated code without broad reflection.
- Use `codegen.Section`, `codegen.JenniferSection`, and `codegen.RawSection` for
  Loom-owned Go generation. Use neutral `.tmpl` assets only for non-Go output
  that genuinely benefits from templates.
- Never edit checked-in generated fixtures manually. Regenerate them through
  the same Loom path consumers use.
- Generated example service methods fail closed with a transport-neutral Loom
  fault. They never create placeholder success bodies, files, or stream events.
  The design remains the source of declared transport responses.
- HTTP transport generation defaults to all artifacts. API metadata
  `Meta("http:generate", "server")` omits client, client-type, client-path, and
  aggregate CLI files while retaining service and server packages. Stale client
  directories are removed only in the safe generator post-write phase.

## OpenAPI Version Architecture

Loom emits OpenAPI 3.2.0 by default and supports a 3.1.1 compatibility target.
There is one DSL parser, one shared semantic IR, and one renderer.

- Gate emitted constructs by target version; do not fork parsers, IRs,
  generators, or canonical output paths.
- Render from the shared document model once. Apply 3.2-only transformations in
  `applyOpenAPI32`; for the 3.1 target, remove only incompatible members in
  `filterOpenAPI31` while preserving shared structures.
- Route version-dependent construction and document passes through
  `versionRouter`. Constructors declare bounded or open-ended `versionRange`
  values; when multiple constructors match, `construct` selects the one with
  the newest lower bound, while `mustConstruct` turns a missing required shape
  into an error. `runPasses` applies every matching pass in registration order,
  and the router carries constructor and pass diagnostics to the generated
  files. Keep non-versioned behavior outside the router.
- Keep shared JSON Schema features such as `contentSchema` when they are valid
  in both targets. Compatibility filtering must remove only version-specific
  members.
- Start OpenAPI contract changes in `http/codegen/openapi/internal/ir`; keep the
  `v3` package focused on rendering IR-owned decisions.
- Keep `http/codegen/openapi` limited to the active schema/document model and
  shared renderer helpers. Schema analysis state is per-render state owned by
  `internal/ir`; do not reintroduce the legacy JSON Hyper-Schema generator,
  package-global definitions, or compatibility aliases for those APIs.
- OpenAPI JSON uses Go 1.27 `encoding/json/v2` with deterministic ordering.
  Preserve two-space indentation, the final newline added by `codegen.File`,
  and configured prefix/indent behavior.
- All Loom-owned runtime, generator, test, fixture, and example JSON uses Go
  1.27 `encoding/json/v2` and `encoding/json/jsontext`. Generated code must use
  the same packages; do not emit or retain the legacy JSON package.
- OpenAPI output defaults to both canonical files. API metadata may select JSON
  or YAML alone; `codegen.File.RemovePaths` must remove only the stale sibling
  in the generator's deterministic post-write phase.
- Seed synthesized examples from stable schema and occurrence identity. Do not
  consume a shared traversal sequence that lets unrelated design changes
  perturb output. API-level example omission must retain explicitly authored
  examples in both JSON and YAML without mutating the evaluated design.
- Treat stable schema names, canonical `operationId`, reusable component
  identity, and extension output as public framework contracts.
- An `Extensions map[string]any` field is tagged `json:"-" yaml:"-"`, so the type
  that declares it must also implement `MarshalJSON` and `MarshalYAML` through
  `openapi.MarshalJSON` and `openapi.MarshalYAML` with a `_Type` alias. A type
  without that pair accepts extensions and drops them at serialization.
- A scoped extension metadata key such as `openapi:<scope>:extension:<x-name>`
  needs a reader at the scope that emits it. Adding the writer alone leaves the
  metadata dead. Cover every scope in the semantic import round trip rather than
  in renderer unit tests alone.

The public feature inventory and metadata usage belong in
`docs/dsl-reference.md` and the consumer `loom` skill. Internal constructor,
filter, and serialization rules belong here.

## OpenAPI Importer Invariants

- Keep a field-level grammar ledger for each OpenAPI object the importer reads.
  Classify every parser field as preserved, conditional, lossy, rejected, or
  parser-only, and name the test or diagnostic that proves the classification.
- Compare that parser ledger with the fixed fields in the official OpenAPI 3.0,
  3.1, and 3.2 schemas. Add a raw-source guard when the standard defines a field
  that the parser does not expose.
- Guard version-specific fields even when the parser exposes their union for
  every version. Reject a 3.2-only field in a 3.0 or 3.1 document before render.
- Treat the OpenAPI 3.1 and 3.2 Schema Object as an open vocabulary at the
  parser boundary. Preserve JSON-compatible `x-*` extensions. Reject unknown
  non-`x-*` keywords unless Loom explicitly supports and tests them.
- Keep the canonical JSON and YAML fixtures in parity. Add every strict,
  cross-boundary construct to both fixtures.
- Test a newly supported schema construct in every legal component and
  operation location. Test shared response and naming policies as matrices,
  not only through the issue specimen that exposed the gap.
- A strict import must render and evaluate. Forward generation must complete,
  and the generated Go code must compile. The generated OpenAPI contract must
  preserve the accepted source semantics.
- Keep an exporter-symmetry fixture for constructs that Loom emits and imports.
  Import the fixture, compile the service, regenerate OpenAPI, and compare the
  semantic contracts. Add each new inverse mapping to this fixture. If an
  exporter shape cannot round-trip, record and test the explicit rejection.
- Add each accepted response shape to an import-to-generation contract test.
  Schema-less errors use a concrete non-default error type and an explicit HTTP
  `Body(Empty)` mapping. This prevents synthesized default error headers. Never
  render `Error(name, Empty)`.
- Plan imported request bodies explicitly as raw, direct typed payloads, or
  selected-body envelopes. Schema-valued dynamic form maps use direct map
  payloads when body-only, selected `Body(...)` attributes when transport
  fields coexist, and raw contracts for multi-media or optional multipart
  bodies.
- Generated HTTP request code currently reserves the local name `body`. Until
  function-local names have an independent allocator, fail an imported
  request-body operation closed when a parameter or security field Goifies to
  `body`; never emit code known not to compile.
- Retain supported source values in the normalized import model before
  rendering. A diagnostic-only path cannot recover discarded values later.
- Preserve `byte` and `binary` string formats as separate OpenAPI contracts.
  Both formats use Loom `Bytes`. The rendered design must retain `byte` in
  metadata because `Bytes` defaults to `binary`.
- Scope diagnostics by their JSON Pointer and owning layer, not by diagnostic
  code alone. The same code can identify a root omission or an operation
  blocker.
- Split safe document omissions before per-operation filtering. Root metadata
  must not make every operation fail during `--skip-unrenderable`.
- Apply component-closure filtering independently from document and operation
  filtering. Referenced component diagnostics must stay with their consumers.
- Copy a shared normalized schema before adding request or response metadata.
  Do not mutate a component schema while rendering one media type.
- Classify a schema as unconstrained only when it has no type and no retained
  assertions or applicators. Map that state to `Any`; do not infer `Any` from a
  missing type alone, because typeless constraints must continue to fail
  closed.
- Normalize `anyOf: [{}, {type: "null"}]` to unconstrained `Any`. The empty
  schema already accepts null, so the explicit null branch adds no contract
  semantics and must not force a nullable wrapper.
- Normalize `oneOf: [$ref, {type: "null"}]` to a nullable named alias only
  when the local referenced schema explicitly excludes null. Regeneration may
  use nullable `anyOf`; reject already-nullable targets because the two
  applicators are not equivalent in that case.
- Normalize JSON request and success-response `oneOf` object branches to an
  explicit untagged Loom union when every branch is a flat object whose fields
  are primitives, concrete named objects, or arrays of either. Promote inline
  branches to deterministic named components, retain the generated sum-type
  API, and select a decoded branch only after exact JSON-shape and generated
  value validation succeeds. Generate every nested named-type validator used
  by a branch. Reject zero or multiple matches, inline nested object fields,
  and string-encoded transport locations.
- Normalize a scalar JSON Schema `const` without a sibling `enum` to a
  one-member Loom enum. Keep structured constants and combined `const` plus
  `enum` schemas rejected.
- Preserve canonical named aliases as OpenAPI components, including aliases of
  `Any`. Preserve the exact source component key even when multiple keys map to
  the same Go identifier. Canonical component identity takes precedence over
  ordinary primitive alias inlining.
- Preserve schema titles as first-class attribute annotations through import,
  attribute duplication, nullable wrapping, shared IR analysis, and rendering.
- Represent unconstrained payload and object fields with explicit presence so
  required JSON `null` is not mistaken for absence. An unconstrained error
  schema needs a concrete error envelope; only the envelope's root user type
  may receive generated error methods.
- Make every response-rendering branch honor `hasSchemaBlock`. Primitive and
  referenced-type shortcuts must not discard examples, defaults, or metadata.
- Preserve numeric-bound and compatible scalar-default siblings on a
  single-reference `allOf` as occurrence constraints over the canonical
  component. The importer emits `openapi:allOf:reference` metadata so
  regeneration retains the reference in `allOf`; other contract siblings
  remain unsupported until Loom can preserve them without flattening.
- Keep method-level error identities collision-safe across operations. Two
  responses with the same HTTP status must not share a generated error type
  when their schemas belong to different operation contracts.
- For a newly supported import construct, cover analysis, rendered source, DSL
  evaluation, and regenerated contract shape. Keep unsupported variants as
  source-specific diagnostics.
- Keep OpenAPI evolution policy outside Loom's module graph. Consumers own
  oasdiff versions, severity overrides, and ignores. Loom documentation can
  provide a pinned external workflow.

## Transport Invariants

- Response-contract manifests classify unary HTTP, supported multipart, SSE,
  and plain HTTP WebSocket success cases explicitly. Multipart cases expose flat primitive and bytes
  parts from prepared transport data. Multipart SSE combinations remain an
  explicit limitation. Applications own codecs and fixtures.
  SSE cases own handshake, message, field-mapping, projection event type, and
  terminal-completion metadata. Declared pre-stream errors stay unary HTTP cases
  with the same stable case IDs.
- WebSocket cases own the `101` handshake headers, stream direction, inbound
  and outbound message types, and terminal behavior. Declared pre-upgrade
  errors stay unary HTTP cases with the same stable case IDs.
- gRPC response-contract cases own status codes, protobuf success messages,
  typed status details, and required header and trailer metadata. Unary and
  server-streaming cases are supported. A successful server stream terminates
  with clean EOF. Client-streaming and bidirectional completion contracts stay
  explicit generation limitations.
- JSON-RPC response-contract cases own success result types, declared error
  codes, typed error-data names, and ID-less notification suppression. Unary
  and server-SSE cases are supported. Server-SSE requests with an ID retain a
  final response. ID-less streams suppress it. Other streaming completion
  shapes stay explicit generation limitations.
- HTTP and JSON-RPC SSE handlers defer committing the stream until the first
  frame, except for the raw JSON-RPC `events/stream` GET listener, which opens
  eagerly so clients can observe readiness.
- Generated HTTP decoders check key membership for optional string query
  parameters. They use nil for an omitted key. They use a nonnil pointer for
  an empty or nonempty value. Generated clients emit the key for every nonnil
  pointer.
- Ordinary unary HTTP handlers delegate request context, observation, decode,
  invocation, response encode, and failure routing to the typed runtime helper.
  A response encoder failure that occurs before commit is encoded through the
  generated error contract; a post-commit failure goes only to the application
  failure handler.
  Other generated HTTP handlers use `HandlerLifecycle` for context,
  observation, and commit-aware failure routing. File and raw-body branches
  also delegate writes and cleanup. Generated code retains typed result and
  stream adapters.
- Generated wildcard static-file routes delegate target resolution to
  `http.NewStaticFileServer`. Directory targets map the captured request suffix
  below the target, while file targets serve the same file for every matching
  route. Generators must not infer target kind from a path extension.
- Protocol errors must use event types compatible with the relevant client
  contract.
- Keep WebSocket lifecycle behavior in the shared runtime wrapper; generated
  endpoints should not grow independent read/write/close loops.
- Keep JSON-RPC envelope validation, batch framing, notification suppression,
  HTTP/SSE negotiation, WebSocket setup, and final stream-response decisions
  in `jsonrpc`. Generated JSON-RPC code supplies typed dispatch, error, and
  stream adapters only.
- Preserve the generated public `ServeHTTP` middleware and policy chain when
  adding JSON-RPC dispatch branches.
- Keep gRPC request context, metadata application, status conversion,
  observation, and clean stream completion in `grpc.ServeUnary`,
  `grpc.ServeStream`, and `grpc.EncodeServerError`. Generated gRPC code supplies
  protocol buffer adapters and typed `grpc.ErrorMapper` callbacks. Do not copy
  the status lifecycle into generated service methods.
- Generated transport observability must remain dependency-free and must never
  emit bodies, params, tool arguments, credentials, or result payloads.
- Keep HTTP, gRPC, and JSON-RPC semantics distinct unless a deliberately shared
  transport core owns the behavior.
- Keep deployment-owned HTTP policies composable at generated method fields.
  Request body policy must cover built-in JSON, text, form, and multipart
  decoding while exposing the same bounded reader to raw-body services.
  Response-cookie policy may alter only deployment-owned `Domain`, `Secure`,
  and `Expires`; designed cookie attributes remain immutable. Strict response
  negotiation and derived HEAD remain opt-in.
- Canonicalize implicit service collections after validation and before
  generation. Preserve explicit `Server.Services` order; do not let Go package
  initialization order churn aggregate clients, CLIs, or transport contracts.
- Keep requiredness and nullability orthogonal. `expr` owns semantic
  nullability; shared service models use `loom.Nullable[T]` for null-admitting
  object fields; JSON decoding boundaries alone use `loom.Optional[T]` for
  optional non-null fields. Reject nullable shapes at transports that cannot
  preserve absent, null, and concrete states instead of weakening the service
  contract.
- JSON decoding types use `loom.Nullable[T]` for non-null array elements and
  map values so validation can reject explicit `null` with stable collection
  paths before conversion to service types.

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
- exact-byte comparison across process-isolated generations when output
  determinism is part of the contract; same-process or semantic comparisons do
  not exercise independent map seeds
- deterministic nested serialization from every custom marshaler that can
  contribute map-valued schema, example, extension, or reference content
- JSON v2 lint coverage that rejects direct `json.Marshal` calls inside
  `MarshalJSON` methods unless they pass `json.Deterministic(true)`
- `libopenapi` parsing
- Redocly linting
- downstream consumer smoke generation where applicable

Serialization migrations must preserve error identity as well as successful
decoding. At JSON request boundaries, test empty, JSON-whitespace-only,
malformed, truncated, and valid bodies through both the handwritten runtime and
generated required and optional body handling. At the generated transport seam,
assert HTTP status, problem code and detail, service invocation, and decoded
payload state.

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
make coverage-ratchet         # protected consumer-aware boundaries
make openapi-contract          # OpenAPI work
make generated-code-quality    # generated Go/output work
make integration-test          # transport behavior
./check.sh --full              # full repository verification
```

The checked-in floors in `coverage/baselines.json` protect evaluated-design
semantics, shared service codegen, HTTP/gRPC/JSON-RPC codegen, and the shared
OpenAPI pipeline. `make coverage-baseline` rewrites them from the current
consumer-aware profile. Treat any reduction as an explicit reviewed contract
change and document its reason; aggregate coverage remains an inspection
artifact, not the ratchet.

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
