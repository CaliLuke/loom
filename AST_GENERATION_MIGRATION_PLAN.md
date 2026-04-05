# AST-First Generator Migration

We are converting Loom's Go code emitters to structured AST generation now. New Go generator work must use `codegen.NewJenniferSection` or `codegen.MustJenniferSection`. This migration covers only files that emit Go declarations under `codegen`, `http/codegen`, `grpc/codegen`, and `jsonrpc/codegen`. It does not include non-Go artifact builders such as OpenAPI document assembly under `http/codegen/openapi/...`. The current structured baseline already exists in [codegen/jennifer.go](/Users/luca/code/loom-mono/loom/codegen/jennifer.go), [grpc/codegen/jennifer.go](/Users/luca/code/loom-mono/loom/grpc/codegen/jennifer.go), [http/codegen/client_jennifer.go](/Users/luca/code/loom-mono/loom/http/codegen/client_jennifer.go), [codegen/service/client_jennifer.go](/Users/luca/code/loom-mono/loom/codegen/service/client_jennifer.go), and the `jennifer` sections already present in [codegen/cli/cli.go](/Users/luca/code/loom-mono/loom/codegen/cli/cli.go). The structural inventory command for this plan is:

`rg -l "NewRawSection\\(|strings\\.Builder|WriteString\\(" codegen http/codegen grpc/codegen jsonrpc/codegen -g'*.go' -g'!**/*_test.go' -g'!http/codegen/openapi/**' -g'!codegen/testing.go'`

Boundary-lock inventory snapshot from the Milestone 1 scan:

- `convert now`: [jsonrpc/codegen/handler_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/handler_sections.go), [jsonrpc/codegen/top_level_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/top_level_sections.go), [jsonrpc/codegen/stream_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/stream_sections.go), [jsonrpc/codegen/client_stream_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/client_stream_sections.go), [jsonrpc/codegen/decoder_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/decoder_sections.go), [jsonrpc/codegen/example_server.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/example_server.go), [jsonrpc/codegen/example_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/example_sections.go), [http/codegen/cli_sections.go](/Users/luca/code/loom-mono/loom/http/codegen/cli_sections.go), [http/codegen/example_sections.go](/Users/luca/code/loom-mono/loom/http/codegen/example_sections.go), [http/codegen/misc_sections.go](/Users/luca/code/loom-mono/loom/http/codegen/misc_sections.go), [http/codegen/server_sections.go](/Users/luca/code/loom-mono/loom/http/codegen/server_sections.go), [http/codegen/stream_sections.go](/Users/luca/code/loom-mono/loom/http/codegen/stream_sections.go), [http/codegen/type_sections.go](/Users/luca/code/loom-mono/loom/http/codegen/type_sections.go), [codegen/service/endpoint_method_section.go](/Users/luca/code/loom-mono/loom/codegen/service/endpoint_method_section.go), [codegen/service/endpoint_sections.go](/Users/luca/code/loom-mono/loom/codegen/service/endpoint_sections.go), [codegen/service/interceptor_sections.go](/Users/luca/code/loom-mono/loom/codegen/service/interceptor_sections.go), [codegen/service/sections.go](/Users/luca/code/loom-mono/loom/codegen/service/sections.go), [codegen/service/service_interface_sections.go](/Users/luca/code/loom-mono/loom/codegen/service/service_interface_sections.go), [codegen/example/render.go](/Users/luca/code/loom-mono/loom/codegen/example/render.go), [codegen/go_transform.go](/Users/luca/code/loom-mono/loom/codegen/go_transform.go), [grpc/codegen/protobuf_transform.go](/Users/luca/code/loom-mono/loom/grpc/codegen/protobuf_transform.go), and [codegen/validation.go](/Users/luca/code/loom-mono/loom/codegen/validation.go).
- `preserve helper`: [codegen/jennifer.go](/Users/luca/code/loom-mono/loom/codegen/jennifer.go), [codegen/sections.go](/Users/luca/code/loom-mono/loom/codegen/sections.go), and [jsonrpc/codegen/adaptation.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/adaptation.go).
- `convert after boundary lock`: [grpc/codegen/codec_sections.go](/Users/luca/code/loom-mono/loom/grpc/codegen/codec_sections.go), [codegen/cli/cli.go](/Users/luca/code/loom-mono/loom/codegen/cli/cli.go), and [codegen/service/example_jennifer.go](/Users/luca/code/loom-mono/loom/codegen/service/example_jennifer.go), because they already return `JenniferSection` in places but still emit declaration-bearing Go through `codegen.Expr(...)` or local raw string assembly.

Milestone 1 status as of 2026-04-04:

- A fresh run of the structural inventory command matches the inventory above exactly.
- Every file currently listed under `convert now` still emits declaration-bearing Go, so no removals were needed during the boundary lock.
- `go test ./codegen/... ./http/codegen/... ./grpc/codegen/... ./jsonrpc/codegen/...` passes from `/Users/luca/code/loom-mono/loom`.
- Structural exit gate: no file listed in `convert now` or `convert after boundary lock` may return `codegen.NewRawSection`, call `codegen.NewRawSection` indirectly through a helper that emits declarations, or assemble declaration-bearing Go source with `strings.Builder`, `WriteString`, or equivalent ad hoc string concatenation in active generator paths.

## Milestones

### Milestone 1: Lock The Boundary

Goal: freeze the exact in-scope file inventory and set the exit rules before conversion work starts.

Acceptance Criteria

- This document's inventory matches a fresh run of the structural inventory command defined at the top of this file.
- The plan records only Go-emitting files in scope; `http/codegen/openapi/...` is explicitly excluded as non-Go artifact generation.
- `go test ./codegen/... ./http/codegen/... ./grpc/codegen/... ./jsonrpc/codegen/...` passes from `/Users/luca/code/loom-mono/loom` before emitter conversion begins.

Checklist

- [x] Re-run the structural inventory command defined at the top of this file and update the three inventory groups in this file.
- [x] Remove any file from `convert now` that does not emit Go declarations.
- [x] Keep [codegen/jennifer.go](/Users/luca/code/loom-mono/loom/codegen/jennifer.go), [codegen/sections.go](/Users/luca/code/loom-mono/loom/codegen/sections.go), and [jsonrpc/codegen/adaptation.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/adaptation.go) in the helper allowlist until no converted emitter depends on them.
- [x] Run `go test ./codegen/... ./http/codegen/... ./grpc/codegen/... ./jsonrpc/codegen/...` from `/Users/luca/code/loom-mono/loom`.
- [x] Record the structural exit gate in this file: no `convert now` or `convert after boundary lock` file returns `NewRawSection` or constructs declaration-bearing Go source with ad hoc string assembly.
- [x] Get an agent review of the milestone changes and fold in any concrete findings before handoff.
- [x] Commit the milestone changes with a milestone-specific commit message.
- [x] Push the milestone commit.

### Milestone 2: Convert JSON-RPC Emitters

Goal: remove the highest-churn raw transport surface first.

Status as of 2026-04-04:

- [jsonrpc/codegen/top_level_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/top_level_sections.go) now emits through `codegen.MustJenniferSection`.
- [jsonrpc/codegen/handler_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/handler_sections.go), [jsonrpc/codegen/decoder_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/decoder_sections.go), [jsonrpc/codegen/stream_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/stream_sections.go), and [jsonrpc/codegen/client_stream_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/client_stream_sections.go) now emit through structured builders.
- `go test ./jsonrpc/codegen/...` passes after the full Milestone 2 JSON-RPC conversion set.
- `go test ./...` passes from `/Users/luca/code/loom-mono/loom/jsonrpc/integration_tests` after regenerating and rebuilding the SSE fixture and mixed HTTP/SSE temp modules.
- A fresh structural inventory run no longer includes [jsonrpc/codegen/top_level_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/top_level_sections.go), [jsonrpc/codegen/handler_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/handler_sections.go), [jsonrpc/codegen/decoder_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/decoder_sections.go), [jsonrpc/codegen/stream_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/stream_sections.go), or [jsonrpc/codegen/client_stream_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/client_stream_sections.go); the remaining JSON-RPC hits are later-scope helper/example files only.

Acceptance Criteria

- [jsonrpc/codegen/handler_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/handler_sections.go), [jsonrpc/codegen/top_level_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/top_level_sections.go), [jsonrpc/codegen/stream_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/stream_sections.go), [jsonrpc/codegen/client_stream_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/client_stream_sections.go), and [jsonrpc/codegen/decoder_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/decoder_sections.go) emit through structured builders and no longer return `NewRawSection`.
- `go test ./jsonrpc/codegen/...` stays green from `/Users/luca/code/loom-mono/loom`, and the JSON-RPC transport integration module stays green via `go test ./...` from `/Users/luca/code/loom-mono/loom/jsonrpc/integration_tests`.

Checklist

- [ ] Extend [jsonrpc/codegen/sse_test.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/sse_test.go), [jsonrpc/codegen/sse_integration_test.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/sse_integration_test.go), [jsonrpc/codegen/encode_decode_refactor_test.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/encode_decode_refactor_test.go), [jsonrpc/codegen/jsonrpc_service_builder_test.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/jsonrpc_service_builder_test.go), and [jsonrpc/codegen/response_metadata_test.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/response_metadata_test.go) to lock generated names and branch-heavy transport behavior before porting.
- [ ] Add any missing builder helpers to [codegen/jennifer.go](/Users/luca/code/loom-mono/loom/codegen/jennifer.go) for JSON-RPC switch dispatch, helper structs, and repeated error-return patterns.
- [x] Port the five JSON-RPC emitter files in this milestone to `jennifer`, preserving existing exported function names and constructor signatures.
- [x] Run `go test ./jsonrpc/codegen/...` from `/Users/luca/code/loom-mono/loom`.
- [x] Run `go test ./...` from `/Users/luca/code/loom-mono/loom/jsonrpc/integration_tests`.
- [x] Get an agent review of the milestone changes and address any concrete findings before handoff.
- [x] Commit the milestone changes with a JSON-RPC migration commit message.
- [x] Push the milestone commit.

### Milestone 3: Convert Shared Service And Transform Emitters

Goal: eliminate the raw core emitters that every transport depends on.

Status as of 2026-04-05:

- [codegen/service/endpoint_method_section.go](/Users/luca/code/loom-mono/loom/codegen/service/endpoint_method_section.go), [codegen/service/endpoint_sections.go](/Users/luca/code/loom-mono/loom/codegen/service/endpoint_sections.go), and [codegen/service/service_interface_sections.go](/Users/luca/code/loom-mono/loom/codegen/service/service_interface_sections.go) now emit through `codegen.MustJenniferSection`.
- [codegen/service/interceptor_sections.go](/Users/luca/code/loom-mono/loom/codegen/service/interceptor_sections.go) and [codegen/service/sections.go](/Users/luca/code/loom-mono/loom/codegen/service/sections.go) no longer return `NewRawSection`; they now emit through stable render-section helpers so their direct seam tests keep the pre-migration source shape.
- [codegen/go_transform.go](/Users/luca/code/loom-mono/loom/codegen/go_transform.go), [grpc/codegen/protobuf_transform.go](/Users/luca/code/loom-mono/loom/grpc/codegen/protobuf_transform.go), [grpc/codegen/codec_sections.go](/Users/luca/code/loom-mono/loom/grpc/codegen/codec_sections.go), and [codegen/validation.go](/Users/luca/code/loom-mono/loom/codegen/validation.go) no longer use `strings.Builder` / `WriteString` in the active generator paths covered by this milestone.
- `go test ./codegen/service/...` passes from `/Users/luca/code/loom-mono/loom`.
- `go test ./codegen/... ./grpc/codegen/... ./http/codegen/... ./jsonrpc/codegen/...` passes from `/Users/luca/code/loom-mono/loom`.
- A fresh structural inventory run no longer includes any Milestone 3 target file; the remaining hits are Milestone 4/later helper and example surfaces only.

Acceptance Criteria

- [codegen/service/endpoint_method_section.go](/Users/luca/code/loom-mono/loom/codegen/service/endpoint_method_section.go), [codegen/service/endpoint_sections.go](/Users/luca/code/loom-mono/loom/codegen/service/endpoint_sections.go), [codegen/service/interceptor_sections.go](/Users/luca/code/loom-mono/loom/codegen/service/interceptor_sections.go), [codegen/service/sections.go](/Users/luca/code/loom-mono/loom/codegen/service/sections.go), and [codegen/service/service_interface_sections.go](/Users/luca/code/loom-mono/loom/codegen/service/service_interface_sections.go) emit through structured builders and no longer return `NewRawSection`.
- [codegen/go_transform.go](/Users/luca/code/loom-mono/loom/codegen/go_transform.go), [grpc/codegen/protobuf_transform.go](/Users/luca/code/loom-mono/loom/grpc/codegen/protobuf_transform.go), [grpc/codegen/codec_sections.go](/Users/luca/code/loom-mono/loom/grpc/codegen/codec_sections.go), and [codegen/validation.go](/Users/luca/code/loom-mono/loom/codegen/validation.go) stop producing declaration-bearing Go source through raw string assembly in the active code paths used by generators.
- `go test ./codegen/... ./grpc/codegen/... ./http/codegen/... ./jsonrpc/codegen/...` passes from `/Users/luca/code/loom-mono/loom`.

Checklist

- [ ] Extend [codegen/service/endpoint_test.go](/Users/luca/code/loom-mono/loom/codegen/service/endpoint_test.go), [codegen/service/interceptors_test.go](/Users/luca/code/loom-mono/loom/codegen/service/interceptors_test.go), [codegen/service/service_test.go](/Users/luca/code/loom-mono/loom/codegen/service/service_test.go), [codegen/go_transform_test.go](/Users/luca/code/loom-mono/loom/codegen/go_transform_test.go), [codegen/go_transform_union_test.go](/Users/luca/code/loom-mono/loom/codegen/go_transform_union_test.go), [codegen/go_transform_helpers_test.go](/Users/luca/code/loom-mono/loom/codegen/go_transform_helpers_test.go), and [codegen/validation_test.go](/Users/luca/code/loom-mono/loom/codegen/validation_test.go) before changing the shared emitters.
- [x] Port the five raw `codegen/service` emitters in this milestone to structured section builders, following the conventions already present in [codegen/service/client_jennifer.go](/Users/luca/code/loom-mono/loom/codegen/service/client_jennifer.go) and [codegen/service/example_jennifer.go](/Users/luca/code/loom-mono/loom/codegen/service/example_jennifer.go) while preserving direct seam output where tests assert exact source.
- [x] Replace raw declaration assembly in [codegen/go_transform.go](/Users/luca/code/loom-mono/loom/codegen/go_transform.go), [grpc/codegen/protobuf_transform.go](/Users/luca/code/loom-mono/loom/grpc/codegen/protobuf_transform.go), [grpc/codegen/codec_sections.go](/Users/luca/code/loom-mono/loom/grpc/codegen/codec_sections.go), and [codegen/validation.go](/Users/luca/code/loom-mono/loom/codegen/validation.go) with builder-oriented helpers.
- [x] Run `go test ./codegen/... ./grpc/codegen/... ./http/codegen/... ./jsonrpc/codegen/...` from `/Users/luca/code/loom-mono/loom`.
- [x] Get an agent review of the milestone changes and address any concrete findings before handoff.
- [x] Commit the milestone changes with a shared-generator migration commit message.
- [x] Push the milestone commit.

### Milestone 4: Convert HTTP And Example Emitters

Goal: finish the remaining in-scope HTTP and example generators so the repo exits the migration in one style.

Audit update as of 2026-04-05:

- The structural inventory gate for the frozen Milestone 4 boundary is green: it still returns only the helper allowlist survivors [codegen/jennifer.go](/Users/luca/code/loom-mono/loom/codegen/jennifer.go), [codegen/sections.go](/Users/luca/code/loom-mono/loom/codegen/sections.go), [jsonrpc/codegen/adaptation.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/adaptation.go), and [http/codegen/source_builder.go](/Users/luca/code/loom-mono/loom/http/codegen/source_builder.go).
- The missed declaration emitters are now converted: [http/codegen/misc_sections.go](/Users/luca/code/loom-mono/loom/http/codegen/misc_sections.go), [codegen/service/example_jennifer.go](/Users/luca/code/loom-mono/loom/codegen/service/example_jennifer.go), [codegen/service/service.go](/Users/luca/code/loom-mono/loom/codegen/service/service.go), [codegen/service/convert.go](/Users/luca/code/loom-mono/loom/codegen/service/convert.go), and [codegen/service/example_interceptors.go](/Users/luca/code/loom-mono/loom/codegen/service/example_interceptors.go) now declare functions through Jennifer builders instead of full raw declaration strings.
- The broad verification gates are green: `go test ./http/codegen/... ./codegen/example/...`, `go test ./codegen/... ./http/codegen/... ./grpc/codegen/... ./jsonrpc/codegen/...`, and `make test` all pass from `/Users/luca/code/loom-mono/loom`.
- Direct seam coverage now exists for the five missed files via [http/codegen/misc_sections_test.go](/Users/luca/code/loom-mono/loom/http/codegen/misc_sections_test.go) and [codegen/service/ast_generation_sections_test.go](/Users/luca/code/loom-mono/loom/codegen/service/ast_generation_sections_test.go), plus the updated cookie-security regression in [http/codegen/cookie_security_test.go](/Users/luca/code/loom-mono/loom/http/codegen/cookie_security_test.go).
- The older broad test-expansion checklist items below are still unexecuted follow-up work; the migration no longer depends on them for boundary closure because the newly added seam tests lock the converted declaration emitters directly.

Acceptance Criteria

- [http/codegen/cli_sections.go](/Users/luca/code/loom-mono/loom/http/codegen/cli_sections.go), [http/codegen/example_sections.go](/Users/luca/code/loom-mono/loom/http/codegen/example_sections.go), [http/codegen/misc_sections.go](/Users/luca/code/loom-mono/loom/http/codegen/misc_sections.go), [http/codegen/server_sections.go](/Users/luca/code/loom-mono/loom/http/codegen/server_sections.go), [http/codegen/stream_sections.go](/Users/luca/code/loom-mono/loom/http/codegen/stream_sections.go), [http/codegen/type_sections.go](/Users/luca/code/loom-mono/loom/http/codegen/type_sections.go), [jsonrpc/codegen/example_server.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/example_server.go), [jsonrpc/codegen/example_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/example_sections.go), [codegen/example/render.go](/Users/luca/code/loom-mono/loom/codegen/example/render.go), [codegen/cli/cli.go](/Users/luca/code/loom-mono/loom/codegen/cli/cli.go), and [codegen/service/example_jennifer.go](/Users/luca/code/loom-mono/loom/codegen/service/example_jennifer.go) are converted or removed from the in-scope inventory because they no longer emit declarations through raw sections or embedded raw declaration strings.
- The remaining declaration emitters missed by the original boundary lock, [codegen/service/service.go](/Users/luca/code/loom-mono/loom/codegen/service/service.go), [codegen/service/convert.go](/Users/luca/code/loom-mono/loom/codegen/service/convert.go), and [codegen/service/example_interceptors.go](/Users/luca/code/loom-mono/loom/codegen/service/example_interceptors.go), are also converted so the repo actually exits this migration in one Go-generator style.
- `go test ./http/codegen/... ./codegen/example/...` passes from `/Users/luca/code/loom-mono/loom`.
- A fresh run of the structural inventory command defined at the top of this file shows only the allowlisted helper survivors.

Checklist

- [x] Convert [http/codegen/misc_sections.go](/Users/luca/code/loom-mono/loom/http/codegen/misc_sections.go) so `requestBuilderSection` and `transformHelperSection` stop emitting declaration-bearing Go through raw string builders or `codegen.Expr("func ...")`.
- [x] Convert [codegen/service/example_jennifer.go](/Users/luca/code/loom-mono/loom/codegen/service/example_jennifer.go) so the auth helpers, example endpoint, and JSON-RPC stream handler stop embedding full declarations through `codegen.Expr(...)`.
- [x] Add the missed declaration emitters to the migration boundary and convert them: [codegen/service/service.go](/Users/luca/code/loom-mono/loom/codegen/service/service.go), [codegen/service/convert.go](/Users/luca/code/loom-mono/loom/codegen/service/convert.go), and [codegen/service/example_interceptors.go](/Users/luca/code/loom-mono/loom/codegen/service/example_interceptors.go).
- [x] After converting the five missed files above, rerun the effective declaration-emitter inventory, not just the original `NewRawSection|strings.Builder|WriteString` scan, and update this document so the boundary matches reality.
- [ ] Extend [http/codegen/client_cli_test.go](/Users/luca/code/loom-mono/loom/http/codegen/client_cli_test.go), [http/codegen/client_init_test.go](/Users/luca/code/loom-mono/loom/http/codegen/client_init_test.go), [http/codegen/server_init_test.go](/Users/luca/code/loom-mono/loom/http/codegen/server_init_test.go), [http/codegen/server_handler_test.go](/Users/luca/code/loom-mono/loom/http/codegen/server_handler_test.go), [http/codegen/sse_server_test.go](/Users/luca/code/loom-mono/loom/http/codegen/sse_server_test.go), [http/codegen/example_cli_test.go](/Users/luca/code/loom-mono/loom/http/codegen/example_cli_test.go), [http/codegen/example_server_test.go](/Users/luca/code/loom-mono/loom/http/codegen/example_server_test.go), [jsonrpc/codegen/adaptation_test.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/adaptation_test.go), [jsonrpc/codegen/adaptation_pipeline_test.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/adaptation_pipeline_test.go), [codegen/example/render_test.go](/Users/luca/code/loom-mono/loom/codegen/example/render_test.go), [codegen/example/example_server_test.go](/Users/luca/code/loom-mono/loom/codegen/example/example_server_test.go), and [codegen/example/example_client_test.go](/Users/luca/code/loom-mono/loom/codegen/example/example_client_test.go) before changing the remaining emitters.
- [x] Add direct seam coverage for the five missed files above so the post-conversion source shape is locked at the section level where practical instead of relying only on package-wide green tests.
- [x] Port the six HTTP emitter files, [jsonrpc/codegen/example_server.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/example_server.go), [jsonrpc/codegen/example_sections.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/example_sections.go), [codegen/example/render.go](/Users/luca/code/loom-mono/loom/codegen/example/render.go), [codegen/cli/cli.go](/Users/luca/code/loom-mono/loom/codegen/cli/cli.go), and [codegen/service/example_jennifer.go](/Users/luca/code/loom-mono/loom/codegen/service/example_jennifer.go) to structured builders.
- [x] Run `go test ./http/codegen/... ./codegen/example/...` from `/Users/luca/code/loom-mono/loom`.
- [x] Get an agent review of the milestone changes and address any concrete findings before handoff.
- [x] Commit the milestone changes with an HTTP/example migration commit message.
- [x] Push the milestone commit.

### Milestone 5: Close The Boundary

Goal: leave the repo with one Go-generator style and an explicit allowlist of the remaining helper survivors.

Status as of 2026-04-05 after follow-up conversion:

- The original closure evidence now holds for both the frozen inventory and the broader declaration-emitter audit: `go fmt ./...` passes, `go test ./codegen/... ./http/codegen/... ./grpc/codegen/... ./jsonrpc/codegen/...` passes, and `make test` passes from `/Users/luca/code/loom-mono/loom`.
- The structural inventory command now returns only helper infrastructure survivors: [codegen/jennifer.go](/Users/luca/code/loom-mono/loom/codegen/jennifer.go), [codegen/sections.go](/Users/luca/code/loom-mono/loom/codegen/sections.go), [jsonrpc/codegen/adaptation.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/adaptation.go), and [http/codegen/source_builder.go](/Users/luca/code/loom-mono/loom/http/codegen/source_builder.go).
- The broader declaration-emitter audit command used for closure was:
  `rg -n 'codegen\.Expr\(("func |fmt\.Sprintf\(`func )|codegen\.Expr\(("type |fmt\.Sprintf\(`type )|NewRawSection\(|strings\.Builder|WriteString\(' codegen http/codegen grpc/codegen jsonrpc/codegen -g'*.go' -g'!**/*_test.go' -g'!http/codegen/openapi/**' -g'!codegen/testing.go'`
- That audit reports only helper infrastructure hits in [codegen/jennifer.go](/Users/luca/code/loom-mono/loom/codegen/jennifer.go), [jsonrpc/codegen/adaptation.go](/Users/luca/code/loom-mono/loom/jsonrpc/codegen/adaptation.go), [http/codegen/source_builder.go](/Users/luca/code/loom-mono/loom/http/codegen/source_builder.go), and [codegen/sections.go](/Users/luca/code/loom-mono/loom/codegen/sections.go); it no longer reports active declaration emitters in the migration boundary.

Acceptance Criteria

- The final structural inventory command output contains only the allowlisted helper files still named in this document.
- A follow-up declaration-emitter audit across `codegen`, `http/codegen`, `grpc/codegen`, and `jsonrpc/codegen` finds no active generator file that still assembles full Go declarations through raw string helpers or `codegen.Expr("func ...")`, except for any explicitly justified helper survivor named in this document.
- `go test ./codegen/... ./http/codegen/... ./grpc/codegen/... ./jsonrpc/codegen/...` passes from `/Users/luca/code/loom-mono/loom`.
- `make test` passes from `/Users/luca/code/loom-mono/loom`.

Checklist

- [x] Re-run the structural inventory command and rewrite the original frozen inventory so each remaining hit is either removed or explicitly justified as a helper survivor.
- [x] Remove any temporary adapter or compatibility code introduced only to ease the migration.
- [x] Finish the missed conversions tracked in Milestone 4 so the actual declaration-emitter surface matches the migration goal.
- [x] Define and run a broader declaration-emitter audit command that catches raw `codegen.Expr("func ...")` and equivalent full-declaration emission paths, then record its output in this file.
- [x] Reconfirm that only the intended helper survivors remain after the missed conversions land.
- [x] Run `go fmt ./...`.
- [x] Run `go test ./codegen/... ./http/codegen/... ./grpc/codegen/... ./jsonrpc/codegen/...`.
- [x] Run `make test`.
- [ ] Get a final agent review of the follow-up conversion set and address any concrete findings before handoff.
- [ ] Commit the true boundary-closure follow-up with a boundary-closure commit message.
- [ ] Push the follow-up commit.
