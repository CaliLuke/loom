# Next Refactor Plan - 2026-04-20

The next refactor work should stay incremental and behavior-preserving. The
best current targets are the conversion generator in
`codegen/service/convert.go`, the remaining websocket-heavy logic in
`http/codegen/stream_sections.go`, and the OpenAPI build pipeline split across
`http/codegen/openapi/internal/ir/analyzer.go` and
`http/codegen/openapi/v3/builder.go`.

## Milestones

### Milestone 1: Service Convert Pipeline Split

Status: Completed on 2026-04-20.

Goal: Separate conversion file assembly, path/import resolution, and legacy single-file flow so conversion changes stop requiring whole-file reasoning.

Acceptance Criteria

- `codegen/service/convert.go` no longer owns both multi-file and legacy single-file orchestration plus path/import utility logic in one file; those concerns live in focused files with concrete names.
- `codegen/service/convert_test.go` still proves multi-package conversion output, and `go test ./codegen/service` passes from the repo root.
- Public behavior stays unchanged: `ConvertFiles` still emits one `convert.go` per grouped target path, and `ConvertFile` still emits the legacy single-file output for the default path.

Checklist

- [x] Split `codegen/service/convert.go` into focused files for orchestration, package/path resolution, and section rendering without changing exported entrypoints.
- [x] Keep `ConvertFiles` as the multi-file entrypoint and `ConvertFile` as the legacy single-file entrypoint, but route both through smaller shared builders.
- [x] Move `commonPath`, `getPkgImport`, and `getExternalTypeInfo` out of the main orchestration file if they still serve package/path resolution only.
- [x] Add direct tests for any newly extracted builder or deduplication helper that currently hides behind `ConvertFiles`.
- [x] Run `go test ./codegen/service`.

### Milestone 2: HTTP Stream Websocket Split

Goal: Reduce the remaining branch density in `http/codegen/stream_sections.go` by splitting websocket send/recv/view/body-init behavior into narrower emitters.

Acceptance Criteria

- `http/codegen/stream_sections.go` no longer mixes SSE entrypoints, websocket struct generation, websocket send/recv generation, and websocket view/body-init helpers in one file.
- Existing websocket regression coverage in `http/codegen/streaming_test.go` and `http/codegen/websocket_golden_test.go` stays green.
- The recently extracted SSE client helpers remain in their own file and are not pulled back into the websocket refactor.

Checklist

- [ ] Split websocket struct/configurer code from websocket send/recv/body-init code into dedicated files under `http/codegen/`.
- [ ] Extract the server websocket result/view/body-init path around `writeServerWebSocketSend`, `writeServerWebSocketSendResult`, and `writeServerWebSocketBodyInit` into a focused helper file.
- [ ] Extract the websocket receive path around `writeServerWebsocketRecvBody` and its validation/return helpers into a focused helper file.
- [ ] Add at least one direct seam test for an extracted websocket helper where the current coverage is only through large golden matrices.
- [ ] Run `go test ./http/codegen`.

### Milestone 3: OpenAPI IR And Builder Boundary Tightening

Goal: Make the OpenAPI pipeline easier to change by separating IR analysis responsibilities from v3 rendering cleanup and component-pruning logic.

Acceptance Criteria

- `http/codegen/openapi/internal/ir/analyzer.go` has a clearer boundary between schema analysis, request/response usage pruning, and body-type assembly.
- `http/codegen/openapi/v3/builder.go` no longer concentrates alias collapse, component pruning, path rewriting, and top-level document assembly in one file.
- `go test ./http/codegen/openapi/internal/ir ./http/codegen/openapi/v3` passes from the repo root.

Checklist

- [ ] Split `http/codegen/openapi/internal/ir/analyzer.go` so request/response usage pruning helpers no longer live in the same file as top-level analyzer construction and `BuildBodyTypes`.
- [ ] Split `http/codegen/openapi/v3/builder.go` so schema-alias collapse and unused-component pruning move behind focused helpers or files with narrow ownership.
- [ ] Preserve the current public entrypoint `openapiv3.New` while reducing the amount of inline orchestration it owns.
- [ ] Extend the existing OpenAPI tests in `http/codegen/openapi/internal/ir/analyzer_test.go`, `http/codegen/openapi/v3/builder_test.go`, or adjacent files instead of creating parallel redundant suites.
- [ ] Run `go test ./http/codegen/openapi/internal/ir ./http/codegen/openapi/v3`.

### Milestone 4: Delivery

Goal: Land the chosen refactor slice with clean tests and an updated tracking note.

Acceptance Criteria

- The selected milestone’s target package tests pass from the repo root with the exact command named in that milestone.
- `git status --short` shows only the intended files for the slice before commit.
- The commit message names the specific refactor surface that was changed.

Checklist

- [x] Execute one milestone at a time instead of mixing conversion, stream, and OpenAPI refactors in one changeset.
- [x] Re-run the exact package test command from the completed milestone after the final formatting pass.
- [ ] Update this plan or replace it with a narrower follow-on plan once the completed milestone is no longer current.
- [ ] Commit the slice with a surface-specific message.
- [ ] Push `main` to `origin`.
