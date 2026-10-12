# Safe public error disclosure readiness — #713

Status: implementation and bounded acceptance complete; frozen candidate awaits
root integration and fresh independent exact-diff review. No commit authorized yet.
Owner: `/root/plan_review`, acting as the error-disclosure implementer after its
independent reviews of the separate authentication/projection plans.
Worktree: `/Users/luca/.codex/worktrees/safe-error-disclosure/loom`.
Parent: `4535a403048510f055fc59ec3dfed41e3769ad58`.
Owning ticket: #713. No other ticket is included implicitly.

## Contract and limits

Unknown service errors expose `internal server error`, retaining the original
error and cause on the server. One shared public-message decision belongs in
`pkg`; HTTP, gRPC and JSON-RPC must consume it. Explicit generated errors and
`ServiceError` values own their public message; an explicit `SafeMessage`
overrides it. Transparent wrappers contribute diagnostic context, not public
message text. Independent joins without an explicit outer owner have a generic
public message; selecting the first branch's message or remedy is forbidden.

Protocol classification remains in its current transport owner. Preserve gRPC
status-code consensus for joins, explicit designed mappings, direct/native
`GRPCStatus` contracts and context-termination codes/no fabricated detail/cause.
Native context termination uses its canonical underlying cancellation/deadline
message instead of private wrapper text; existing wrapped-context test messages
change accordingly. Preserve designed typed
error bodies. This task does not invent new public status mappings, add generic
JSON-RPC instance IDs, change client-side remote-cause retention, or redesign
HTTP/JSON-RPC selection of typed bodies from joined errors. Root confirmed this
message-boundary scope and independently reviewed/released this readiness record.

No mutation of a service error, its remedy, history or underlying cause is
allowed during encoding. Safe-message overrides must govern the default gRPC
status, detail message and message-bearing history that would otherwise expose
the overridden diagnostic; designed custom detail schemas remain author-owned.

The concrete risk is raw database/host/path diagnostics in gRPC and JSON-RPC.
The existing HTTP default contract and documented `SafeMessage` behavior in
`docs/error-handling.md` establish the consumer need. Applications still decide
the content of explicitly authored public errors and protocol statuses.

## Inventory and owner

Run and reconcile:

```sh
rg -n 'ErrorSafeMessage|newErrorResponse|NewStatusError|EncodeServerError|err.Error' pkg http/error.go http/codegen/service_data_response_errors.go grpc jsonrpc/codegen/handler_sections.go jsonrpc/codegen/stream_sections_websocket.go jsonrpc/server.go
```

- `pkg/error_remedy.go:ErrorSafeMessage` currently uses tree-wide remedy lookup
  and falls back to raw `err.Error()`. Replace that message decision with direct
  owner checks and transparent single-cause traversal, stopping at independent
  joins. Existing `LoomErrorNamer` identifies generated/public service errors;
  direct `LoomErrorRemedier` with nonempty safe text explicitly owns the message.
- `http/error.go:NewProblemResponse` already hides plain errors, but discards
  their explicit safe remedy and can expose wrapper text around a service error.
  Route detail text through the shared decision while retaining HTTP status,
  problem type, code and instance handling.
- `grpc/error.go:newErrorResponse` synthesizes fault details from raw text;
  `NewStatusError` independently exposes raw status text. `grpc/error_contract.go`
  owns classification and its `encodeError` must make both message surfaces use
  the same disclosure decision. Do not use tree-wide `errors.As` to classify joins.
- `grpc/error.go` history emission currently copies `h.Message` directly. Cover
  safe override and safe validation history without mutating the source history.
- JSON-RPC unary handler, SSE and WebSocket emitters already call
  `loom.ErrorSafeMessage(err)`. Prove the new runtime owner reaches all of them;
  change generated policy only when a concrete remaining call bypasses it.
- `http/codegen/service_data_response_errors.go` already emits the shared helper
  for default designed error bodies. Preserve authored typed body encoding.
- `jsonrpc/error_data.go` contains machine metadata/remedies, not a plain-error
  message fallback. Its current protocol classification is outside this task.

The repository-wide non-generated `ErrorSafeMessage` caller scan also finds
`jsonrpc/server.go:envelopeDecodeError` (RequestBodyTooLarge public validation
error) and tests. All production callers are wire-message construction; no
server-side diagnostic logging calls this helper. Generated JSON-RPC handlers
retain the original error for error-handler/observation paths. Preserve this
invariant in generated behavior tests.

Residual to hand to root with a concrete generated witness:
`streamErrorSwitch` and `writeStreamErrorDataSwitch` in
`jsonrpc/codegen/stream_sections_websocket.go` use tree-wide `errors.As` to pick
a mapped `LoomErrorNamer` and typed body. An independent join of one mapped
custom error and an unrelated failure can therefore publish that branch's
designed data. This is separate typed-response ownership, not authority to
expose unknown error messages in this task.

## Ordered tasks and evidence

- [x] Recite the root recovery workflow before implementation: failing direct
  tests, failing legacy model, shared fix, focused runtime/generated checks,
  exact independent diff review, one atomic #713 commit, root push/integration,
  then task-owned temporary-output cleanup. Preserve all unrelated files; no
  forbidden Git commands or direct edits of checked-in generated files.
- [x] Add table-driven failing tests in `pkg/error_disclosure_test.go`,
  `http/error_disclosure_test.go`, and `grpc/error_disclosure_test.go`. Cover
  plain/wrapped secret, nil, service/generated owner, safe remedier, wrapper
  around owner, single-error join, reordered independent joins, outer owned join,
  merged validation history, explicit status and native cancellation/deadline.
  Assert original error text, `errors.Is` cause and owned history remain intact.
  Run `go test ./pkg ./http ./grpc -run 'Test.*Disclosure' -count=1` and record
  actual baseline failures before changing production code.
- [x] Add `pkg/tla/error_disclosure/PublicDisclosure.tla`, `Legacy.cfg`,
  `Checked.cfg` and README with owner traversal, independent-join stop and wire
  surface invariants. Legacy must expose a secret; checked must not. Maintain
  `grpc/tla/error_contract/ErrorContract.tla` and README correspondence: the
  previous full-message aggregate detail guarantee becomes safe aggregate
  disclosure while status consensus and explicit-owner invariants remain.
- [x] Implement shared public-message ownership in `pkg/error_remedy.go` with
  focused private extraction in `pkg/error_disclosure.go` only if needed for
  clarity. Integrate HTTP detail and gRPC default status/detail/history through
  it. Preserve protocol-owned context/status/typed mapping branches.
- [x] Re-run direct regressions, then
  `go test ./pkg ./http ./grpc ./jsonrpc -count=1` and
  `go test -race ./grpc -count=1`. Update old expectations that explicitly
  asserted raw diagnostics in `pkg/error_test.go`, `grpc/error_contract_test.go`
  and `grpc/context_error_test.go` to the accepted message contract, retaining
  every prior status, detail type, cause and classification assertion.
- [x] Extend `codegen/generator/grpc_joined_error_test.go` and its existing
  `grpcClientErrorBoundaryDSL`/`runDesignHarness` with plain and wrapped sentinel
  secrets and safe overrides. Check real unary and stream-opening failures over
  bufconn for both status message and decoded protobuf detail/history. Run
  `LOOM_DIR="$PWD" go test ./codegen/generator -run '^TestGRPC(JoinedServerErrors|ClientErrorBoundary|ClientErrorCause)$' -count=1`.
- [x] Add `http/codegen/error_disclosure_test.go` using `renderHTTPModule` for
  generated default error dispatch; assert HTTP status/body, service call and
  original error retained by the endpoint/observer seam. Add
  `jsonrpc/codegen/error_disclosure_test.go` using `renderJSONRPCModule` for
  unary, POST SSE and WebSocket error envelopes. Reuse existing stream harness
  patterns in `viewed_stream_result_test.go`; do not create a new harness engine.
  Assert generic message, existing code/id behavior, safe override, and original
  server error, including endpoint failure and explicit stream SendError paths.
- [x] Run with `LOOM_DIR` bound to this worktree:
  `go test ./http/codegen -run '^Test.*ErrorDisclosure$' -count=1` and
  `go test ./jsonrpc/codegen -run '^Test.*ErrorDisclosure$|^TestJSONRPCGeneratedTypedAndRawErrorsRoundTrip$|^TestJSONRPCErrorProjectionUsesCoreHelpers$' -count=1`.
  Each new generated-module test must tidy, build/vet and execute its actual
  transport behavior. Keep typed public body/remedy roundtrip controls intact.
- [x] Document generic versus explicit public message ownership and regeneration
  impact in `docs/error-handling.md`; send framework/consumer skill wording to
  root for shared-file coordination. Record exact proof commands/results in the
  owning model README. Independent exact-diff review and commit remain pending below.

## Models and generated evidence

Use `/Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar`, SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
From each model directory:

```sh
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir "$TASK_MODEL_OUTPUT" -config Legacy.cfg PublicDisclosure.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir "$TASK_MODEL_OUTPUT" -config Checked.cfg PublicDisclosure.tla
```

Use distinct output directories per run. In `grpc/tla/error_contract`, run
`Legacy.cfg`, `LegacyStatus.cfg`, `Aggregate.cfg` against `ErrorContract.tla`,
and `ContextChecked.cfg` against `ContextTermination.tla` with the same command
shape. Negative controls must violate their named invariants (exit 12), not fail
parsing; checked configurations must pass (exit 0). These finite models do not
prove arbitrary Go unwrap graphs, protobuf encoding or network delivery.

Expected generator output differences: none, because existing default encoders
already call the runtime boundaries. Confirm by generating the existing gRPC
client-boundary design and the bounded new HTTP/JSON-RPC disclosure designs into
task-owned persistent `/tmp/loom-error-disclosure-comparison/{base,current,repeat}`
modules with identical module paths/design source/toolchain. Baseline replacement
is `/Users/luca/.codex/worktrees/correctness-baseline/loom`, used read-only;
current/repeat replace the implementing worktree. Pin the module requirement to
the pushed parent SHA. For each module run the existing CLI in a fresh process:

```sh
LOOM_DIR="$SOURCE" GOFLAGS=-mod=mod go run -mod=mod github.com/CaliLuke/loom/cmd/loom gen example.com/disclosure/design
go mod tidy
go build ./...
go vet ./...
```

Compare all `gen/` bytes with `diff -ru` for base/current and current/repeat.
Record source-version manifest differences explicitly. Runtime regression tests
must fail with baseline replacement and pass with current replacement, without
mutating the shared baseline. No checked-in fixture regeneration is needed when
all generated bytes remain identical; any actual generator change requires a
revised intended-difference record, representative fixture regeneration and
root re-review of validation scope before delivery.

Implementation and focused acceptance are complete. See
[`pkg/tla/error_disclosure/README.md`](../pkg/tla/error_disclosure/README.md)
for exact model results, generated base/current/determinism comparison, baseline
wire failures, source-selection limitation, intended differences and the separate
joined typed-body witness. Runtime `grpc.NewStatusError` remains an explicit
constructor with unchanged caller-owned message text.

- [ ] Root transfers the frozen scoped files and coordinated skill/index updates.
- [ ] A fresh non-implementing reviewer authorizes the exact integrated diff
  against its actual parent; address findings and rerun invalidated gates.
- [ ] Root makes one atomic #713 commit, pushes and updates the master tracker.
- [ ] Preserve durable evidence, then remove task temporary output after review.

The first independent final review found a high-severity history-owner divergence.
The repair adds `ResolveErrorMessage` as the sole traversal returning message and
explicit-safe-override flag; `ErrorSafeMessage` retains the message-only contract.
Nil, empty and code-only outer-remedy regressions and a `HistoryLegacy.cfg`
counterexample preceded the correction. The owning README records the affected
gates and reused unchanged generated-output evidence. Re-review remains pending.

Root owns integration. Do not commit this worktree candidate independently.
