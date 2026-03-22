# Refactor Checklist

This document tracks the ongoing complexity-reduction work so progress survives context compaction.

## Rules Of Engagement

- Refactor only. No feature behavior changes.
- Tests before structural edits for each hotspot.
- Small slices only. Keep the worktree green after each step.
- Update this document after each meaningful step.

## Current Status

- [x] Inventory completed.
- [x] Worktree baseline checked.
- [x] First refactor slice in `expr/http_endpoint.go` complete.
- [x] Second refactor slice in `expr/http_endpoint.go` complete.
- [x] `expr/http_endpoint.go` validation/finalization responsibilities split.
- [x] Shared transport metadata builders reduced in `http/codegen/service_data.go`.
- [x] Generic and protobuf transform engines deduplicated.
- [x] JSON-RPC streaming section builders decomposed.
- [x] `http/form.go` traversal and scalar conversion separated.
- [x] Security projection consolidated.
- [x] `codegen/service/service_data.go` long builders trimmed.

## Priority Inventory

### P1

- [x] Split `expr/http_endpoint.go` validation/finalization responsibilities.
  Target areas:
  - `validateBodyAndPayload`
  - `Finalize`
  - `validateParams`
  - `validateHeadersAndCookies`
  Goal:
  - separate request-body rules, missing-payload rules, transport mapping, and payload-shape-specific validation.

- [x] Reduce duplication between `codegen/go_transform.go` and `grpc/codegen/protobuf_transform.go`.
  Goal:
  - extract a shared transformation kernel for object/array/map walking and keep protocol-specific hooks local.

- [x] Collapse repeated transport metadata assembly in `http/codegen/service_data.go`.
  Target areas:
  - `extractPathParams`
  - `extractQueryParams`
  - `extractHeaders`
  - `cookieData`
  Goal:
  - one shared builder for transport elements with narrow strategy inputs.

### P2

- [x] Decompose `jsonrpc/codegen/stream_sections.go` raw section builders.
  Goal:
  - split SSE/WebSocket framing, error encoding, and response/notification message assembly.

- [x] Isolate traversal, scalar conversion, and key parsing in `http/form.go`.
  Goal:
  - separate reflection walking from scalar codec and form-key parsing.

- [x] Centralize security projection across DSL, service-data, and OpenAPI builders.
  Goal:
  - add one intermediate security model instead of re-deriving it in multiple packages.

### P3

- [x] Trim long coherent builders in `codegen/service/service_data.go`.
  Goal:
  - reduce function size after higher-risk hotspots are under control.

## Active Slice

### `codegen/service/service_data.go`

Completed seam:

- [x] Split the large method-data builder into payload/result/security helper phases and verify the service/codegen suites.

## Work Log

- 2026-03-22: Created tracking document and locked the initial hotspot order.
- 2026-03-22: Added a direct internal seam test for the empty-payload/request-option branch and extracted the first two validation helpers from `validateBodyAndPayload`.
- 2026-03-22: Extracted array/map/object payload transport validation helpers from `validateBodyAndPayload`.
- 2026-03-22: Split `Finalize` into explicit phases: JSON-RPC stream normalization, requirement finalization, transport body finalization, JSON-RPC body state, and response/error finalization.
- 2026-03-22: Reduced `validateParams` and `validateHeadersAndCookies` into smaller mapped-attribute and payload-compatibility helpers, backed by direct seam tests in `expr/http_endpoint_internal_test.go`.
- 2026-03-22: Verified the `expr` package with `go test ./expr/...` after each refactor slice.
- 2026-03-22: Collapsed repeated transport metadata assembly in `http/codegen/service_data.go` behind shared transport-element builders and re-verified with `go test ./http/codegen/...`.
- 2026-03-22: Extracted a shared primitive object-field assignment helper under `internal/transformassign` and rewired both Go transform engines before re-running `go test ./codegen/...` and `go test ./grpc/codegen/...`.
- 2026-03-22: Decomposed the JSON-RPC SSE stream section builders into named endpoint/service emitters while keeping `go test ./jsonrpc/codegen/...` green.
- 2026-03-22: Split `http/form.go` into dedicated traversal and key-parsing helpers, added direct parser coverage, and re-ran `go test ./http/...`.
- 2026-03-22: Centralized security requirement projection for service/http codegen and OpenAPI builders, then re-ran `go test ./codegen/service/... ./http/codegen/... ./jsonrpc/codegen/...` and `go test ./http/codegen/openapi/...`.
- 2026-03-22: Trimmed `codegen/service/service_data.go` by extracting payload/result/error projection helpers from `buildMethodData`, then re-ran `go test ./codegen/service/... ./http/codegen/...`.
