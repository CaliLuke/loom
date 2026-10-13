# Protobuf Go-name validator harness (#811)

Parent `baf64781d58eb02c3d4dfee80f94a112fa2b7fff`.
Worktree `/Users/luca/.codex/worktrees/grpc-validator-harness/loom`.

1. [x] Reproduce the existing `TestGeneratedProtoGoNames` type error at the parent.
2. [x] Update only embedded callback types in `checkNames` and `checkInvalid` to
   accept generated validators' optional diagnostic path arguments.
3. [x] Run the existing generated build/vet/runtime test and scoped pinned lint;
   freeze this test-only diff separately from production identity issue #810.

Acceptance: all protobuf field-name, valid/invalid value, both union branch,
request/response conversion and CLI round-trip assertions run unchanged. No
production code, generated output, validator API or model changes are needed.

The parent diagnostic is `func(message *namespb.EchoRequest,
validationContext ...string) (err error)` cannot be passed as `func(Q) error`.
#709's optional diagnostic path contract is accepted; retaining the stale
function type prevents the regression from exercising its original obligations.

Validation: Go 1.27.2 darwin/arm64, golangci-lint 2.14.0. The exact existing
`LOOM_DIR=<worktree> go test ./grpc/codegen -run
'^TestGeneratedProtoGoNames$' -count=1` passes (4.061 s), including generated
`go build ./...`, `go vet ./...` and the unchanged runtime checks.
`golangci-lint run ./grpc/codegen` reports zero issues; `git diff --check` passes.
No broader sweep or generated-output comparison is needed for callback types
inside the test harness: the production generator and design are unchanged.

Root integration against `7a61ec0e0` preserves the two-line callback correction
exactly. The existing generated test passes again with explicit local source;
receipt `/tmp/loom-811-integrated.log`. All original assertions remain unchanged.
Final exact integrated review approved patch
`693d3f301784214e0e89a8daabd7f2c65b96b3155f024f2d5615b0c5f35469e4`
with no findings. Delivered as `285da6b0e`, pushed and confirmed closed.
