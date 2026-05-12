# Generated Projection Parity Tests

Projection parity will be guarded in the framework by adding a shared test helper that compares canonical result attributes and generated projected attributes by field name and pointer presence. The first implementation targets service view projection generation because HTTP, gRPC, and JSON-RPC all consume `codegen/service` viewed result metadata; HTTP and gRPC transport tests assert that response metadata is wired to the same viewed-result projection boundary, while existing golden tests keep outward transport code generation stable.

## Status

- 2026-05-12 — Plan created from roadmap note.
- 2026-05-12 — Shared projection parity helper added and `go test ./codegen/testutil` passes.
- 2026-05-12 — Service view projection guardrail added and `go test ./codegen/service -run 'TestViews|TestProjectionParity'` passes.
- 2026-05-12 — HTTP and gRPC transport projection smoke guardrails added and focused transport tests pass.
- 2026-05-12 — Full projection parity package set passes with `go test ./codegen/testutil ./codegen/service ./http/codegen ./grpc/codegen`.
- 2026-05-12 — Required plan peer review was performed after implementation; plan language was reconciled with the implemented service parity and transport smoke checks.

## Milestones

### Milestone 1: Shared Projection Parity Helper

Toc: Helper

Goal: Add a reusable reflection-free helper for generator tests to compare Loom expression attributes by declared field path.

Acceptance Criteria

- `codegen/testutil` exposes helpers where `ProjectionParityDiffs` reports missing, extra, and required-presence drift for exact generated projection shapes, and `ProjectionViewParityDiffs` reports missing and required-presence drift for a single view while allowing extra fields in union projected types.
- `codegen/testutil/projection_parity_test.go` contains tests where a missing projected field and required-presence mismatch produce failure text naming the concrete field path.
- `go test ./codegen/testutil` passes from `/Users/luca/code/loom-mono/loom`.

Checklist

- [x] Add `codegen/testutil/projection_parity.go` with field-path collection for `expr.Object`, nested result types, arrays, and maps.
- [x] Add `codegen/testutil/projection_parity_test.go` cases for exact parity, missing field, extra field, and required-presence mismatch.
- [x] Run `go test ./codegen/testutil` from `/Users/luca/code/loom-mono/loom`.

### Milestone 2: Service View Projection Guardrail

Toc: Service Views

Goal: Make generated service view tests fail when projected result types drift from their view declarations.

Acceptance Criteria

- `codegen/service/views_test.go` asserts every `ProjectedTypeData` result type matches the full method result shape and contains every declared view projection.
- `codegen/service/testdata/views_dsls.go` includes a nested result-type view case that proves nested projected fields are compared against the nested view.
- `go test ./codegen/service -run 'TestViews|TestProjectionParity'` passes from `/Users/luca/code/loom-mono/loom`.

Checklist

- [x] Extend `codegen/service/views_test.go` to run the parity helper for every generated projected result type.
- [x] Add a nested projected-view fixture in `codegen/service/testdata/views_dsls.go` with fields that differ between default, tiny, and extended views.
- [x] Run `go test ./codegen/service -run 'TestViews|TestProjectionParity'` from `/Users/luca/code/loom-mono/loom`.

### Milestone 3: Transport Projection Wiring Smoke Checks

Toc: Transports

Goal: Add transport-level smoke assertions that generated response metadata consumes the same viewed-result projection boundary as service view metadata.

Acceptance Criteria

- `http/codegen/service_data_refactor_test.go` contains `TestHTTPProjectionParity`, which asserts `resp.ViewedResult.FullRef` equals `endpoint.Method.ViewedResult.FullRef` and `resp.ServerBody` fans out the same `default` and `tiny` views as `resp.ViewedResult.Views`.
- `grpc/codegen/server_test.go` contains `TestGRPCProjectionParity`, which asserts `endpoint.ViewedResultRef` equals `endpoint.Method.ViewedResult.FullRef` and `endpoint.Response.ServerConvert` consumes the projected `RTView` attribute from the viewed-result wrapper.
- `go test ./http/codegen -run TestHTTPProjectionParity` and `go test ./grpc/codegen -run TestGRPCProjectionParity` pass from `/Users/luca/code/loom-mono/loom`.

Checklist

- [x] Add `TestHTTPProjectionParity` in `http/codegen/service_data_refactor_test.go` using `ExplicitBodyUserResultObjectMultipleViewDSL` to assert `resp.ViewedResult.FullRef` and view fanout match `endpoint.Method.ViewedResult`.
- [x] Add `TestGRPCProjectionParity` in `grpc/codegen/server_test.go` using `MessageResultTypeWithViewsDSL` to assert `endpoint.ViewedResultRef` and `endpoint.Response.ServerConvert` point at the viewed-result projected attribute.
- [x] Run `go test ./http/codegen -run TestHTTPProjectionParity` from `/Users/luca/code/loom-mono/loom`.
- [x] Run `go test ./grpc/codegen -run TestGRPCProjectionParity` from `/Users/luca/code/loom-mono/loom`.

### Milestone 4: Peer Review Reconciliation

Toc: Review

Goal: Reconcile the execution plan with the required peer-review findings before final handoff.

Acceptance Criteria

- The plan states that the peer review happened after implementation and identifies the reconciliation in `## Status`.
- Milestone 3 no longer describes transport smoke checks as field-shape parity assertions.
- `roadmap/generated_projection_parity_tests.html` is regenerated after review-driven Markdown edits.

Checklist

- [x] Run the required critique-only peer review against the plan and changed code paths.
- [x] Update Milestone 1 to distinguish exact parity from view containment with extra generated fields allowed.
- [x] Update Milestone 2 to state that service tests check full generated projection shape plus every declared view projection.
- [x] Update Milestone 3 to name the exact HTTP and gRPC reference-wiring assertions instead of overclaiming transport field-shape parity.
- [x] Regenerate `roadmap/generated_projection_parity_tests.html` after peer-review reconciliation.

### Milestone 5: Roadmap Closure

Toc: Closure

Goal: Leave the implementation, tracker, and exact verification commands in a handoff-ready state.

Acceptance Criteria

- This Markdown plan marks all completed checklist items with `[x]` and the sibling HTML tracker is regenerated from this file.
- `go test ./codegen/testutil ./codegen/service ./http/codegen ./grpc/codegen` passes from `/Users/luca/code/loom-mono/loom`.
- `git diff -- roadmap/generated_projection_parity_tests.md codegen/testutil codegen/service http/codegen grpc/codegen` shows only projection parity helper, tests, fixtures, and tracker updates.

Checklist

- [x] Run `go test ./codegen/testutil ./codegen/service ./http/codegen ./grpc/codegen` from `/Users/luca/code/loom-mono/loom`.
- [x] Regenerate `roadmap/generated_projection_parity_tests.html` from the Markdown plan.
- [x] Inspect `git diff -- roadmap/generated_projection_parity_tests.md codegen/testutil codegen/service http/codegen grpc/codegen` from `/Users/luca/code/loom-mono/loom`.
