# Public error disclosure (#713)

The shared message owner distinguishes explicit public errors from diagnostic
wrappers and independent failures. A transparent wrapper does not own public
text; a join cannot select one branch's safe message. A direct public owner of a
join deliberately chooses its text. Unknown failures use `internal server error`.
Native cancellation retains its protocol code and no synthetic detail while
using canonical context text. Explicit protocol statuses remain protocol-owned.

`PublicDisclosure.tla` enumerates thirteen bounded error shapes, two message
surfaces and history entries. `Legacy.cfg` reproduces disclosure and branch-selected
ownership; `HistoryLegacy.cfg` reproduces a separate history remedy lookup that
misses an intermediate safe owner beneath a nil/empty/code-only outer remedy;
`Checked.cfg` checks shared owner selection, diagnostic exclusion, original
preservation, history masking and native termination. This is a classification model, not a proof
of arbitrary Go traversal, protobuf serialization, custom error bodies, cyclic
unwrap trees or network delivery. Runtime and generated tests check those
implementation boundaries relevant to this change.

Use TLA+ tools v1.7.4 (TLC 2.19), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
From this directory, run each configuration with a distinct task output path:

```sh
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir "$TASK_MODEL_OUTPUT" -config Legacy.cfg PublicDisclosure.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir "$TASK_MODEL_OUTPUT" -config HistoryLegacy.cfg PublicDisclosure.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir "$TASK_MODEL_OUTPUT" -config Checked.cfg PublicDisclosure.tla
```

Legacy and HistoryLegacy must violate a named invariant (exit 12); Checked must
pass (exit 0).
Keep checker output outside the repository and remove it after independent
review. The following evidence records the implementation correspondence.

## Evidence

At parent `4535a403048510f055fc59ec3dfed41e3769ad58`, the new direct
regressions failed on raw unknown diagnostics, wrapper messages and joined
branch ownership. Legacy TLC failed `PublicOwnership` (exit 12, 11 states);
Checked passed (exit 0, 20 states). Both ran before the production correction.
The review repair extends that model: Legacy still fails `PublicOwnership`
(14 states); HistoryLegacy fails `NoDiagnosticDisclosure` with safe status/detail
and private history (24 states); Checked passes all six invariants (26 states).
These extended model checks preceded the history repair.
The related gRPC model controls still behave as intended: Legacy fails
`CompleteFailure` (19 states), LegacyStatus fails `OrderIndependentStatus`
(25 states), Aggregate passes (36 states), and ContextChecked passes
(30 distinct / 40 generated states). Runs used Go 1.27.2 and Corretto
25.0.4.1 with the pinned TLA+ jar above.

Implementation correspondence:

- `pkg.ResolveErrorMessage` follows one cause, takes an explicit safe remedy or
  named public error's own text, and stops at independent joins. It never writes
  to the input graph. Direct tests cover nil, generated/public errors, wrappers,
  single-child and reordered joins, explicit outer ownership and retained causes.
  Its second result is true only when the selected owner supplies a nonempty
  `SafeMessage`, including when its text equals the generic or named message.
  `ErrorSafeMessage` is the message-only projection of this resolver.
- HTTP problem detail and gRPC default status/detail/history consume that owner.
  A merged validation history uses each entry's safe message; an enclosing safe
  remedy masks entries that have no explicit safe override. Aggregate and entry
  decisions use the same resolver result; no independent remedy traversal or
  text comparison decides masking. Protocol status
  selection and custom detail bodies remain with their existing owners.
- The exported `grpc.NewStatusError` remains an explicit protocol-message
  constructor. Its caller-provided text is unchanged; classified server encoding
  supplies the shared public message through a private constructor. Native
  cancellation/deadline classifications use canonical sentinel text and retain
  their codes without adding protobuf detail.
- Existing JSON-RPC unary, SSE and WebSocket emitters already call the shared
  helper. Generated tests exercise endpoint-returned failures and explicit
  stream `SendError`, checking envelope code/id/message, secret exclusion and
  the original service error/cause retained on the server.

Passed focused acceptance commands from the implementing worktree:

```sh
go test ./pkg ./http ./grpc ./jsonrpc -count=1
go test -race ./grpc -count=1
LOOM_DIR="$PWD" go test ./codegen/generator -run '^TestGRPC(JoinedServerErrors|ClientErrorBoundary|ClientErrorCause)$' -count=1
LOOM_DIR="$PWD" go test ./http/codegen -run '^TestGeneratedErrorDisclosure$' -count=1
LOOM_DIR="$PWD" go test ./jsonrpc/codegen -run '^TestGeneratedErrorDisclosure$|^TestJSONRPCGeneratedTypedAndRawErrorsRoundTrip$|^TestJSONRPCErrorProjectionUsesCoreHelpers$' -count=1
```

The scoped golangci-lint run over `pkg`, `http`, `grpc`, `codegen/generator`,
`http/codegen` and `jsonrpc/codegen` passes with zero findings.
`go run ./scripts/docscheck` and `git diff --check` also pass.

The bounded CLI comparison uses the two new disclosure DSL specimens and the
existing `grpcClientErrorBoundaryDSL`, all copied into identical
`example.com/disclosure` modules. The requirement is the pushed parent version
`v1.10.0-alpha.8.0.20261011180955-4535a4030485`; an explicit `replace` selects
the pristine parent checkout for baseline and the implementation checkout for
current/repeat. Each generation is a separate CLI process. Commands are:

```sh
LOOM_DIR="$SOURCE" GOFLAGS=-mod=mod go run -mod=mod github.com/CaliLuke/loom/cmd/loom gen example.com/disclosure/design
go mod tidy
go build ./...
go vet ./...
```

All nine generated modules compile and vet. Every `gen/` byte, including
`loom.json`, is identical between base/current and current/repeat for all three
specimens. There are **no intended generator output differences** and no
checked-in fixture output changes. Copying the same new wire harness into
base/current and running `go test -race ./...` fails against baseline and passes
against current for HTTP, JSON-RPC and gRPC. Baseline JSON-RPC exposes the
sentinel in all five unary/stream branches; baseline gRPC exposes it in status
messages and misses safe overrides; baseline HTTP exposes wrapper/join text.
The changed contract takes effect on runtime upgrade without regeneration.

Source-selection limitation: the existing JSON-RPC generated-test helper
`resolveJSONRPCTestLoomModulePath` resolves the current Git root and ignores
`LOOM_DIR`. Therefore its baseline replay with only that variable is not evidence.
The explicit persistent-module replacements above establish the baseline
comparison without changing the shared baseline or expanding this ticket into
harness infrastructure work.

## Review repair: shared history owner

The initial candidate's gRPC history encoder called `ExtractErrorRemedy`, which
stops at the first remedier even when it returns nil or only a code. Its public
message resolver continued through that wrapper to an intermediate safe owner.
`TestNestedHistoryErrorDisclosure` reproduced the divergence before the repair:
a merged pair of `Fault("private first")` / `Fault("private second")`, wrapped
in a safe remedier and then a nil, empty or code-only remedier, had a safe detail
message but private history. The original errors and causes remained intact.

The repair centralizes both the text and explicit-safe-override flag in
`ResolveErrorMessage`. gRPC resolves aggregate and entry messages through that
one owner. New controls preserve an entry's own safe override and prove that
identical public strings do not decide masking. The exported resolver adds no
new disclosure policy; `ErrorSafeMessage` retains its message-only contract.

After the repair, the runtime suites (`pkg`, `http`, `grpc`, `jsonrpc`), gRPC race
suite, scoped `pkg`/`grpc` golangci-lint and all three existing persistent wire
harnesses (`go test -race -count=1 ./...`) pass. The generator, design inputs and
toolchain did not change, so the exact-byte/determinism evidence above is reused.
No regeneration or exhaustive fixture audit was needed. Repair checker output
is retained under `/tmp/loom-error-disclosure-review-{Legacy,HistoryLegacy,Checked}`;
wire results are `current/*/review-repair-wire.log` under the comparison directory.

## Separate residual: joined typed-response ownership

This correction owns public **message** selection, not existing typed-body or
status selection. A live generated witness using
`jsonRPCTypedErrorRoundTripDSL` returns:

```go
errors.Join(&tools.ToolError{
    Name: "forbidden", Code: "TOOLS_FORBIDDEN",
    Message: "instance message for forbidden",
    RemedyCode: "tools.forbidden.change",
    RetryHint: "retry forbidden after changing the request",
}, errors.New("independent cleanup failure"))
```

POST `/tools` with method `typed`, id `joined`, and params
`{"id":"joined","kind":"forbidden"}` returns code 4403, message
`internal server error`, and the first branch's designed `ToolError` data,
including `code: "TOOLS_FORBIDDEN"` and its instance message. The generated
`errors.As` selection crosses the independent join. Preserve this as a separate
protocol/typed-body ownership issue; changing those mappings is outside #713.
The same class of tree-wide selection exists in the stream data switches.

Task evidence is retained temporarily under
`/tmp/loom-error-disclosure-comparison` (base/current/repeat modules, exact empty
diffs, command and wire logs, and `residual` live witness) and
`/tmp/loom-error-disclosure-*` model output directories. Root coordinates cleanup
after independent review; none of those generated files or raw logs is committed.
