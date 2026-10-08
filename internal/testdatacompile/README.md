# Exported design compile corpus

Run `make test-testdata-compile` from the repository root. The harness discovers
exported `func()` design variables and functions in `codegen/service/testdata`,
`http/codegen/testdata`, `grpc/codegen/testdata`, `jsonrpc/codegen/testdata`, and
`expr/testdata`. Discovery uses Go types, so code embedded in golden strings is
not mistaken for a design, and new designs require no catalog edits.

Each design gets an isolated temporary module. The harness runs:

1. `loom gen`, including protoc and both Go plugins for emitted gRPC schemas.
2. `loom example`.
3. `go mod tidy`.
4. `go build ./...` and `go vet ./...`, including the example application.

The wrapper clears imported package initialization state before invoking the
selected design, matching the fresh DSL context used by the direct unit tests.
Shared fixture expressions remain available through their package variables.

`loomsource.Resolve` selects the framework source once for the suite. Use
`LOOM_DIR=/absolute/path/to/loom` for an unpushed change or the shared
`make loom-local` / `make loom-remote` selector. Remote mode fetches the current
pushed commit. The CLI and designs come from that same resolved source.

The suite shares the caller's module and build caches. At most
`TESTDATA_PARALLEL` temporary modules run at once (default 2); each module is
removed at the end of its subtest. Allow several GB of free disk for Go's
compiled artifacts. A command timeout or disk exhaustion aborts the suite and
fails the gate; infrastructure failures must never become design expectations.

Commands use `internal/testprocess`, the shared subprocess lifecycle owner.
On Unix, its guardian ends compiler and generator descendants on cancellation
or parent termination; other platforms retain standard `os/exec` cancellation.
The [process ownership model](../testprocess/tla/README.md) records the protocol
and its assumptions. The harness timeout regression checks this integration
without running the full design corpus.

Examples:

```sh
make test-testdata-compile TESTDATA_RUN='TestDesigns/grpc/codegen/'
make test-testdata-compile TESTDATA_RUN='TestDesigns/http/codegen/PayloadQueryBoolDSL$'
LOOM_TESTDATA_RESULTS=/tmp/loom-corpus make test-testdata-compile
```

The optional results directory records one JSON observation per executed case,
including the first failing phase and command diagnostics. It does not change
the tracked expectations. Remove old observations before comparing run counts,
or use a fresh directory. Ordinary unit tests run catalog/manifest checks;
Make, `ci-local`, and the dedicated CI matrix run the complete compile tier.

## Expected failures

`expectations.json` lists exact design IDs, failing phase, required diagnostic,
and reason. Intentionally invalid DSL fixtures use the reason
`intentional validation failure` and must fail during `gen`. Every other known
failure requires a linked Loom GitHub issue with a reproduction. Do not add
blanket domain exclusions, silently skip designs, or automatically accept a
failed run as a new baseline.

Expected failures still execute and verify their phase and diagnostic. A pass
fails the test until its stale expectation is removed. Missing catalog entries,
unknown fields, empty diagnostics, and missing issue links also fail validation.
Newly exposed framework failures belong in separate issues and atomic commits,
not inline fixes to this harness.
