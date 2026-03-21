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

Target: [expr/http_endpoint.go](/Users/luca/code/goa-light/expr/http_endpoint.go#L411)

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

Target: [codegen/service/service_data.go](/Users/luca/code/goa-light/codegen/service/service_data.go#L1420)

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

Target: [grpc/codegen/service_data.go](/Users/luca/code/goa-light/grpc/codegen/service_data.go#L444)

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

Target: [grpc/codegen/protobuf_transform.go](/Users/luca/code/goa-light/grpc/codegen/protobuf_transform.go#L106)

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

Target: [dsl/http.go](/Users/luca/code/goa-light/dsl/http.go)

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

## Done Criteria

- [ ] Checklist document committed before substantive refactor commits.
- [ ] Each pass completed in order with its own verification.
- [ ] No intentional public API changes.
- [ ] No intentional generated-output changes.
- [ ] Full repo verification green at the end.
