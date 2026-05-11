# Refactor Checklist

This document is the execution checklist for the planned behavior-preserving
refactor. It is written for an implementing agent: follow the passes in order,
do not merge passes, and do not change behavior intentionally.

## Ground Rules

- [ ] Treat this work as pure refactoring. Do not intentionally change public
      APIs, generated output, or validation semantics.
- [ ] Keep passes isolated. Finish one pass, run its verification, and only
      then start the next pass.
- [ ] Add or tighten characterization tests before restructuring code.
- [ ] Keep existing validation/error text stable unless a test proves exact text
      is irrelevant.
- [ ] Prefer small helper extraction over broad rewrites. Preserve exported
      entrypoints and package boundaries unless this checklist says otherwise.

## Pass 0: Baseline and Safety Harness

- [ ] Confirm the worktree is clean enough to review changes safely.
- [ ] Run the existing focused baseline before editing:
  - [ ] `go test ./expr/...`
  - [ ] `go test ./codegen/service/... ./http/codegen/... ./grpc/codegen/... ./jsonrpc/codegen/...`
- [ ] Record any failing baseline tests before changing code.
- [ ] Add missing characterization coverage for any branch that will be split
      but is not already directly exercised.

## Pass 1: Split HTTP Endpoint Validation

Target: [expr/http_endpoint.go](/Users/luca/code/loom-mono/loom/expr/http_endpoint.go#L411)

### Required coverage

- [ ] Duplicate response status codes.
- [ ] Request-body option conflicts:
  - [ ] `OptionalRequestBody` with `FormRequest`
  - [ ] `OptionalRequestBody` with `MultipartRequest`
  - [ ] skip-body flags with incompatible body options
- [ ] Array, map, and object payload/body shape conflicts.
- [ ] Streaming request/response restrictions.
- [ ] Tagged response validation rules.

### Refactor steps

- [ ] Keep `func (e *HTTPEndpointExpr) Validate() error` as the single exported
      entrypoint.
- [ ] Extract response validation into a dedicated private helper.
- [ ] Extract body/payload-shape validation into a dedicated private helper.
- [ ] Extract request-body option compatibility checks into a dedicated private
      helper.
- [ ] Keep existing `validateSkipBodyEncoding`, `validateStreamingSSE`,
      `validateJSONRPCTransport`, `validateRedirect`, `validateRoutes`,
      `validateParams`, and `validateHeadersAndCookies` entrypoints intact unless
      a pure helper split is needed.
- [ ] Replace the nested duplicate-response-status scan with a linear map-based
      check while preserving the existing error message count and text.
- [ ] Keep validation ordering stable where practical so existing tests do not
      churn on message order.

### Verification

- [ ] Run focused tests:
  - [ ] `go test ./expr/...`
- [ ] If any validation message changed, confirm the change is intentional and
      update tests only when exact text is not a supported contract.

## Pass 2: Decouple Generic Service Codegen From Transport Policy

Target: [codegen/service/service_data.go](/Users/luca/code/loom-mono/loom/codegen/service/service_data.go#L1420)

### Required coverage

- [ ] Existing `codegen/service` seam tests stay green.
- [ ] Add direct seam tests for:
  - [ ] JSON-RPC SSE classification
  - [ ] JSON-RPC WebSocket classification
  - [ ] HTTP skip-request-body propagation
  - [ ] HTTP skip-response-body propagation
  - [ ] mixed-results stream metadata assembly

### Refactor steps

- [ ] Keep `buildMethodData` responsible for transport-agnostic method assembly.
- [ ] Move JSON-RPC transport classification into a private helper that returns
      the transport mode for a method.
- [ ] Move HTTP skip-body flag extraction into a private helper that returns the
      two skip flags for a method.
- [ ] Keep `MethodData` fields unchanged unless a field is provably unused
      outside transport-specific templates and tests.
- [ ] Split stream metadata assembly into:
  - [ ] a transport-neutral base stream builder
  - [ ] a transport-specific adjustment helper for JSON-RPC SSE/WebSocket cases
- [ ] Preserve all current stream naming, descriptions, and behavior.

### Verification

- [ ] Run focused tests:
  - [ ] `go test ./codegen/service/...`
- [ ] Run dependent transport tests that consume `service.MethodData`:
  - [ ] `go test ./http/codegen/... ./grpc/codegen/... ./jsonrpc/codegen/...`

## Pass 3: Decompose gRPC Service Analysis

Target: [grpc/codegen/service_data.go](/Users/luca/code/loom-mono/loom/grpc/codegen/service_data.go#L444)

### Required coverage

- [ ] Existing gRPC generation tests stay green.
- [ ] Add direct seam tests for:
  - [ ] message/import collection deduplication
  - [ ] request CLI arg assembly
  - [ ] response metadata assembly
  - [ ] security scheme partitioning between message and metadata
  - [ ] streaming endpoint stream data attachment

### Refactor steps

- [ ] Keep `analyze` as the top-level coordinator for one service, but remove
      detailed per-endpoint assembly from its main loop.
- [ ] Split the current work into explicit phases:
  - [ ] protobuf message conversion
  - [ ] message and import collection
  - [ ] request assembly
  - [ ] response assembly
  - [ ] endpoint assembly
- [ ] Replace repeated slice scans with internal lookup maps for:
  - [ ] collected messages by name
  - [ ] collected proto imports by path
- [ ] Remove the side-effect-heavy local `collect` closure and replace it with a
      dedicated collector helper or collector type.
- [ ] Preserve the final ordering of `sd.Messages`, `sd.ProtoImports`, and
      `sd.Endpoints` so generated output remains stable.

### Verification

- [ ] Run focused tests:
  - [ ] `go test ./grpc/codegen/...`
- [ ] Re-run related client/server generation tests if golden output changes are
      suspected.

## Pass 4: Simplify Protobuf Transform Generation

Target: [grpc/codegen/protobuf_transform.go](/Users/luca/code/loom-mono/loom/grpc/codegen/protobuf_transform.go#L106)

### Required coverage

- [ ] Existing protobuf transform golden tests stay green.
- [ ] Add direct tests for:
  - [ ] wrapped scalar fields
  - [ ] optional pointer fields
  - [ ] alias/user-type conversions
  - [ ] union handling
  - [ ] `Any` handling
  - [ ] assignment-vs-new-variable behavior

### Refactor steps

- [ ] Keep `protoBufTransform` as the top-level entrypoint.
- [ ] Separate top-level type dispatch from final code emission.
- [ ] Extract shared scalar assignment generation used by the `Any` and default
      cases.
- [ ] Extract pointer-handling branches from `transformObject` into focused
      helpers.
- [ ] Keep all new helpers package-private.
- [ ] Preserve current generated code shape unless a test demonstrates that
      formatting-equivalent output is acceptable.

### Verification

- [ ] Run focused tests:
  - [ ] `go test ./grpc/codegen/...`
- [ ] Confirm protobuf transform goldens and direct seam tests still pass.

## Follow-On Cleanup: Split `dsl/http.go`

Target: [dsl/http.go](/Users/luca/code/loom-mono/loom/dsl/http.go)

This happens only after Passes 1-4 are complete and green.

### Required coverage

- [ ] Existing `expr` and DSL-facing tests remain green.
- [ ] Add tests only if a helper move exposes an uncovered branch.

### Refactor steps

- [ ] Split endpoint-body/form helpers into a dedicated file.
- [ ] Split mapped-attribute accessors (`headers`, `cookies`, `params`) into a
      dedicated file.
- [ ] Split simple endpoint flag setters (`MultipartRequest`, `FormRequest`,
      `OptionalRequestBody`, skip-body helpers) into a dedicated file.
- [ ] Centralize repeated `eval.Current()` access patterns behind small private
      helpers where this reduces duplication without obscuring behavior.
- [ ] Do not change exported DSL function names or contracts.

### Verification

- [ ] Run focused tests:
  - [ ] `go test ./dsl/... ./expr/...`

## Final Verification

- [ ] Run formatting:
  - [ ] `go fmt ./...`
- [ ] Run full test suite:
  - [ ] `make test`
- [ ] Run lint:
  - [ ] `make lint`
- [ ] Review `git diff` to confirm the work stayed behavior-preserving and did
      not accidentally edit generated `gen/` artifacts.

## Opportunistic Follow-Up Triggers

These are not active sweep work. Revisit them only when nearby code changes
make the cost and value clear.

### Shared Stream IR Across Transports

Current state:

- Partial sharing exists through `internal/ssecodegen` for SSE header emission
  and write-and-flush helpers used by HTTP and JSON-RPC.
- WebSocket codegen still duplicates frame-boundary logic, buffered reads, and
  cancellation handling across HTTP and JSON-RPC.

Trigger:

- If a concrete SSE or WebSocket bug forces touching both HTTP and JSON-RPC
  stream codegen in the same change, extract the shared fix into an internal
  stream IR/helper package instead of applying it twice.

Starting points:

- `http/codegen/stream_sections_websocket_send.go`
- `http/codegen/stream_sections_websocket_recv.go`
- `jsonrpc/codegen/client_stream_sections_websocket.go`
- `jsonrpc/codegen/stream_sections.go`

### Template Source Composition

Current state:

- The HTTP `template_sources*.go` files are still below the rough 1000-line
  ceiling.
- A preemptive template registry would risk composition-order churn without a
  concrete payoff.

Trigger:

- If a template source file crosses the ceiling, or request/response generation
  gains a non-trivial shared partial, split that specific file with
  `joinHTTPTemplateSource` instead of introducing a broad registry.

### Base Service-Data Ownership

Current state:

- `ProtoImports` moved out of base `service.Data` into
  `grpc/codegen.ServiceData.ProtoGoImports`.
- No clearly transport-specific fields remain on base `service.Data` today.

Trigger:

- If a second transport-specific field accumulates on `service.Data`, revisit
  whether transports should implement a smaller data interface rather than
  continuing to embed and extend one base struct.

### Codegen Helper Visibility

Current state:

- Some dead or in-package-only exported helpers have been downcased, but many
  exported identifiers remain across `http/codegen`, `grpc/codegen`, and
  `jsonrpc/codegen`.

Trigger:

- When touching an area of codegen for another reason, audit nearby exports and
  downcase the ones that are provably in-package only. Avoid a broad sweep
  until plugin/public API boundaries are explicitly classified.

### DSL Evaluation Context Isolation

Current state:

- DSL evaluation still uses process-global `eval.Context`.
- This is acceptable for the one-shot `loom gen` CLI flow, but it limits
  concurrent independent evaluation and cleaner embedded/programmatic use.

Trigger:

- If Loom needs concurrent DSL runs, long-running embedded generation, or
  stronger evaluator isolation in tests, introduce explicit evaluation context
  plumbing instead of adding more behavior to the global context.

Starting points:

- `eval/context.go`
- `eval/eval.go`
- callers of `eval.Register`, `eval.RunDSL`, and `eval.Reset`

### Import Alias Determinism

Current state:

- Import alias collision safety has direct regression coverage in
  `codegen/import_alias_safety_test.go`.
- Generated code still depends on readable deterministic aliases for trust and
  reviewability when package names overlap.

Trigger:

- When generated imports or external type conversion code changes, add or
  tighten cases that prove alias selection stays deterministic,
  collision-free, and readable. Prefer stable aliases over opaque hash-heavy
  names where a clear alias is available.

Starting points:

- `codegen/import_cleanup.go`
- `codegen/service/convert_paths.go`
- `codegen/service/convert_types.go`
- `codegen/import_alias_safety_test.go`

## Done Criteria

- [ ] Checklist document committed before substantive refactor commits.
- [ ] Each pass completed in order with its own verification.
- [ ] No intentional public API changes.
- [ ] No intentional generated-output changes.
- [ ] Full repo verification green at the end.
