# Next Refactor Plan - 2026-04-20

The next refactor work should stay incremental and behavior-preserving. The
best current targets are the conversion generator in
`codegen/service/convert.go`, the remaining websocket-heavy logic in
`http/codegen/stream_sections.go`, and the OpenAPI build pipeline split across
`http/codegen/openapi/internal/ir/analyzer.go` and
`http/codegen/openapi/v3/builder.go`.

## Milestones

### Milestone 1: Service Convert Pipeline Split

Completed on 2026-04-20. The service conversion pipeline is now split so
entrypoints live in `codegen/service/convert_entrypoints.go`, shared file
assembly and helper deduplication live in
`codegen/service/convert_builder.go`, and package/path resolution lives in
`codegen/service/convert_paths.go`. `codegen/service/convert_test.go` still
covers multi-package output, a direct helper-dedup seam test was added, and
`go test ./codegen/service` passed.

### Milestone 2: HTTP Stream Websocket Split

Completed on 2026-04-20. The remaining websocket-heavy logic was split so
shared websocket helpers now live in
`http/codegen/stream_sections_websocket_common.go`, struct/configurer assembly
lives in `http/codegen/stream_sections_websocket_structs.go`, send/view/body-init
logic lives in `http/codegen/stream_sections_websocket_send.go`, and receive
logic lives in `http/codegen/stream_sections_websocket_recv.go`. A direct seam
test was added in `http/codegen/stream_sections_websocket_send_test.go`,
`go test ./http/codegen -count=1` passed, and the slice was committed as
`refactor websocket stream sections` (`6016f604`) and pushed to `origin/main`.

### Milestone 3: OpenAPI IR And Builder Boundary Tightening

Completed on 2026-04-20. The OpenAPI IR analyzer boundary is now split so
body-type assembly lives in
`http/codegen/openapi/internal/ir/analyzer_body_types.go` and request/response
usage pruning lives in
`http/codegen/openapi/internal/ir/analyzer_usage_pruning.go`, leaving
`http/codegen/openapi/internal/ir/analyzer.go` focused on analyzer state and
schema construction. The v3 builder entrypoint now delegates top-level document
assembly to `http/codegen/openapi/v3/document_builder.go`, while component
schema cleanup and ref traversal live in
`http/codegen/openapi/v3/schema_cleanup.go` and
`http/codegen/openapi/v3/schema_refs.go`. Direct seam coverage was added to
`http/codegen/openapi/internal/ir/analyzer_test.go` and
`http/codegen/openapi/v3/builder_test.go`, and
`go test ./http/codegen/openapi/internal/ir ./http/codegen/openapi/v3` passed.

### Milestone 4: Delivery

Completed on 2026-04-20. The Milestone 1 slice was delivered on its own,
`go test ./codegen/service` was re-run after the final formatting pass, the
tracking plan was updated, and the work was committed as
`refactor service convert pipeline` and pushed to `origin/main`.
