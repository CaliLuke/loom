# Transport IR Convergence Plan

Chosen design: keep transport-specific IR packages, but converge them around a small shared analysis kernel in `codegen/service` so new capabilities stop requiring one HTTP path and one gRPC-only path. Do not build a mega-IR or generic plugin system first.

## Scope

In scope:

- shared service-layer descriptors for payloads, results, errors, streams, mixed results, and wrapper structs
- tiny transport-local adapters over HTTP/gRPC IR where shared helpers need common inputs
- gRPC and HTTP rewiring to consume shared descriptors
- seam tests in `codegen/service` plus transport regression tests where behavior is sensitive

Out of scope:

- merging HTTP and gRPC IR packages into one model
- generic metadata normalization across HTTP headers/query/cookies and gRPC metadata
- renderer/template sharing across HTTP and gRPC
- moving HTTP body/response-status shaping into shared code

## Current State

Already complete:

- shared security partitioning in [codegen/service/security_partition.go](/Users/luca/code/loom-mono/loom/codegen/service/security_partition.go)
- gRPC IR boundary in [grpc/codegen/internal/transportir](/Users/luca/code/loom-mono/loom/grpc/codegen/internal/transportir)
- first shared descriptor seam in [codegen/service/transport_descriptors.go](/Users/luca/code/loom-mono/loom/codegen/service/transport_descriptors.go)

Still duplicated or partially shared:

- error type/package resolution
- stream payload/result type selection
- viewed-result projection selection in transport-specific stream/render code
- mixed-result capability classification
- request/response wrapper-struct capability classification
- HTTP websocket/SSE stream type selection and gRPC stream send/recv selection

## Milestone 1: Freeze Shared Kernel Boundary

Goal: lock the exact boundary and prevent generic-abstraction drift.

Acceptance Criteria

- This document names the shared entrypoints under `codegen/service`.
- Non-goals are explicit and match the code changes.
- Owners are classified as `shared-kernel`, `transport-orchestrator`, or `transport-specific`.

Checklist

- [x] Keep [codegen/service/security_partition.go](/Users/luca/code/loom-mono/loom/codegen/service/security_partition.go) as the security baseline.
- [x] Keep [http/codegen/service_data_analysis.go](/Users/luca/code/loom-mono/loom/http/codegen/service_data_analysis.go) as a transport orchestrator.
- [x] Keep [grpc/codegen/service_data_analysis.go](/Users/luca/code/loom-mono/loom/grpc/codegen/service_data_analysis.go) as a transport orchestrator.
- [x] Keep [http/codegen/service_data_payload.go](/Users/luca/code/loom-mono/loom/http/codegen/service_data_payload.go) and [http/codegen/service_data_response.go](/Users/luca/code/loom-mono/loom/http/codegen/service_data_response.go) transport-specific.
- [x] Keep [grpc/codegen/service_data_helpers.go](/Users/luca/code/loom-mono/loom/grpc/codegen/service_data_helpers.go) `extractMetadata` transport-specific.
- [x] Treat package/type/view/stream/error capability selection as shared-kernel work.

## Milestone 2: Finish Service-Layer Descriptor Kernel

Goal: make `codegen/service` the single source of truth for shared transport capability classification.

Acceptance Criteria

- [codegen/service/transport_descriptors.go](/Users/luca/code/loom-mono/loom/codegen/service/transport_descriptors.go) covers payload, result, error, stream, mixed-result, and wrapper-struct capability descriptors.
- Shared helpers do not depend on HTTP or gRPC IR packages.
- Shared-helper tests prove package overrides, viewed results, streaming modes, mixed results, and wrapper struct flags.

Checklist

- [x] Add error descriptors covering error attribute, package, name, and ref resolution.
- [x] Add stream type descriptors covering streaming payload/result attributes, packages, names, and refs.
- [x] Add method capability descriptor covering:
  unary result, streaming result, streaming payload, viewed result, mixed results, request struct, response struct.
- [x] Keep `DefaultPackageName`, payload descriptor, and result descriptor as shared entrypoints.
- [x] Add seam tests for:
  package override, viewed result, server streaming, client streaming, bidirectional streaming, mixed results, error package override, wrapper struct capability.

## Milestone 3: Align Common HTTP/gRPC IR Inputs

Goal: make shared helpers consume common concepts without forcing one merged IR.

Acceptance Criteria

- HTTP and gRPC each expose tiny local adapters or use existing `service.MethodData` plus attrs to call shared helpers.
- No shared helper imports `http/codegen/internal/transportir` or `grpc/codegen/internal/transportir`.
- Common input inventory is recorded here.

Common subset

- service name
- endpoint name
- endpoint description
- request payload attribute
- response result attribute
- error name/type/attribute
- stream presence and direction
- transport security requirement list

Transport-only fields that stay local

- HTTP:
  routes, status variants, cookies, redirect, SSE envelope fields, body origin, file servers
- gRPC:
  protobuf message attrs, metadata/trailers maps, proto streaming message refs

Checklist

- [x] Confirm shared helpers use only `service.Data`, `service.MethodData`, and raw attrs.
- [x] Add tiny transport-local adapters only where stream/message attrs differ.
- [x] Keep protobuf and HTTP body/status details out of shared descriptors.

## Milestone 4: Rewire gRPC Fully Onto Shared Descriptors

Goal: remove remaining gRPC-local capability classification.

Acceptance Criteria

- gRPC stream send/recv type selection uses shared descriptors.
- gRPC viewed-result selection uses shared descriptors.
- gRPC error type/package resolution uses shared descriptors.
- gRPC analysis files only own protobuf preparation, gRPC metadata extraction, and renderer assembly.

Checklist

- [x] Rewire [grpc/codegen/service_data_analysis.go](/Users/luca/code/loom-mono/loom/grpc/codegen/service_data_analysis.go) to consume method capability and error descriptors.
- [x] Rewire [grpc/codegen/service_data_helpers.go](/Users/luca/code/loom-mono/loom/grpc/codegen/service_data_helpers.go) result-context logic to use shared result descriptors only.
- [x] Rewire [grpc/codegen/service_data_convert.go](/Users/luca/code/loom-mono/loom/grpc/codegen/service_data_convert.go) request/result/error package resolution to use shared descriptors.
- [x] Rewire [grpc/codegen/service_data_stream.go](/Users/luca/code/loom-mono/loom/grpc/codegen/service_data_stream.go) send/recv branching to use shared stream descriptors.
- [x] Keep metadata extraction and protobuf conversion local.

## Milestone 5: Rewire HTTP Fully Onto Shared Descriptors

Goal: remove remaining HTTP-local capability classification where overlap is real.

Acceptance Criteria

- HTTP package/type/view selection uses shared descriptors everywhere overlap exists.
- WebSocket and SSE type selection use shared result/stream descriptors where possible.
- HTTP body/path/header/cookie/status shaping remains local.

Checklist

- [x] Rewire [http/codegen/service_data_payload.go](/Users/luca/code/loom-mono/loom/http/codegen/service_data_payload.go) and [http/codegen/service_data_body_types.go](/Users/luca/code/loom-mono/loom/http/codegen/service_data_body_types.go) to use shared payload descriptors.
- [x] Rewire [http/codegen/service_data_response.go](/Users/luca/code/loom-mono/loom/http/codegen/service_data_response.go) to use shared result and error descriptors.
- [x] Rewire [http/codegen/service_data_routes.go](/Users/luca/code/loom-mono/loom/http/codegen/service_data_routes.go) wrapper-struct and payload-ref decisions to use shared capability descriptors.
- [x] Rewire [http/codegen/websocket.go](/Users/luca/code/loom-mono/loom/http/codegen/websocket.go) stream payload/result selection to use shared stream descriptors.
- [x] Rewire [http/codegen/sse.go](/Users/luca/code/loom-mono/loom/http/codegen/sse.go) mixed-result and event-type selection to use shared result/stream descriptors.
- [x] Keep body/status/SSE envelope rendering local.

## Milestone 6: Extract Shared Capability Builders

Goal: move one more layer up from raw descriptors to reusable capability builders.

Acceptance Criteria

- At least one concrete cross-transport capability builder exists in `codegen/service`.
- It is used by both HTTP and gRPC.
- It does not render transport output directly.

Candidate builders

- request/result capability builder
- stream direction and type builder
- error capability builder
- wrapper-struct capability builder

Checklist

- [x] Extract a concrete method capability builder consumed by both HTTP and gRPC.
- [x] Keep transport-specific rendering and metadata/body logic out of that builder.
- [x] Delete replaced ad hoc booleans and branching helpers.

## Milestone 7: Remove Dead Duplication

Goal: leave only transport-specific survivors.

Acceptance Criteria

- Searches for old duplicated package/type/view helpers come back empty.
- Remaining direct branching in HTTP/gRPC is transport-specific.
- No dead shared or local adapters remain.

Checklist

- [x] Remove replaced helper code after migration.
- [x] Run:
  `rg -n "PayloadLoc|ResultLoc|ViewedResult|StreamingPayload|StreamingResult|HasMixedResults|StreamKind|ErrorLocs" http/codegen grpc/codegen codegen/service`
- [x] Confirm survivors are transport-specific or service-data generation, not duplicated capability logic.

## Milestone 8: Raise Test Bar

Goal: prove convergence with seam tests first, then transport regressions.

Acceptance Criteria

- `codegen/service` has direct seam tests for every shared descriptor/capability builder.
- gRPC and HTTP have targeted regression tests for viewed results, streams, mixed results, and package overrides.
- Existing generator suites stay green.

Checklist

- [x] Extend [codegen/service/transport_descriptors_test.go](/Users/luca/code/loom-mono/loom/codegen/service/transport_descriptors_test.go) for errors, mixed results, and wrapper structs.
- [x] Add or update gRPC seam/regression tests for viewed-result streaming and error package resolution.
- [x] Add or update HTTP regression tests for websocket viewed results, SSE mixed results, and package override cases.
- [x] Keep direct seam tests paired with transport test coverage.

## Milestone 9: Verify and Document Final State

Goal: finish with concrete proof that future feature work starts in one shared place.

Acceptance Criteria

- This file names final shared entrypoints.
- [AI_REVIEW_ARCH_20260403.md](/Users/luca/code/loom-mono/loom/AI_REVIEW_ARCH_20260403.md) reflects the implemented state.
- Full verification passes.

Final shared entrypoints

- `service.DefaultPackageName`
- `service.BuildPayloadDescriptor`
- `service.BuildResultDescriptor`
- `service.BuildErrorDescriptor`
- `service.BuildStreamDescriptor`
- `service.DescribeStream`
- `service.DescribeMethodCapabilities`

Checklist

- [x] Update [AI_REVIEW_ARCH_20260403.md](/Users/luca/code/loom-mono/loom/AI_REVIEW_ARCH_20260403.md) to reference implemented shared entrypoints.
- [x] Update this file to mark completed milestones and any deliberate survivors.
- [ ] Run `go fmt ./...`.
- [ ] Run `go test ./codegen/service/...`.
- [ ] Run `go test ./grpc/codegen/...`.
- [ ] Run `go test ./http/codegen/...`.
- [ ] Run `go test ./jsonrpc/codegen/...`.
