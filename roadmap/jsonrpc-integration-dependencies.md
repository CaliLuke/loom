# JSON-RPC integration dependency adoption (#910)

Parent: `c80c5b3595c772018ad9ba3607e36dd985bdc398`.

The shared wire-name validator correctly uses the framework's existing pinned
`golang.org/x/net/http/httpguts`. The independent JSON-RPC integration module
imports generator code but lacks its indirect requirement and checksums.
Keep the shared validator and readonly CI policy; repair the consuming manifest.
No dependency version or public behavior changes.

## Sequence and acceptance

1. [x] Reproduce current-main readonly framework compilation failure and compare
   the exact CI failure. Inspect the other integration consumer and lint discovery.
2. [x] Run tidy with a disposable modfile and inspect the complete delta: one
   existing pinned x/net requirement and its two root-identical checksums.
3. [x] Apply that exact two-manifest delta. Compile all integration packages with
   readonly modules, run the existing module lint and confirm tidy has no drift.
4. [x] Independent review of the exact final diff, then one #910 commit and push;
   close the report and reconcile the master inventory.

## Evidence

CI run `38498585620`, job `115639900588`, fails during JSON-RPC integration lint
because `expr/http_wire_bindings.go` cannot load `httpguts` in readonly mode.
`GOEXPERIMENT=jsonv2 go test -mod=readonly ./framework -run '^$'` reproduces the
same missing-checksum failure on the parent. `go list` without `-deps` is not a
sufficient reproduction: it only lists the root package and returns success.
The HTTP integration module's `go list -mod=readonly -deps ./...` passes; it does
not import the generator path. Existing lint already visits all nonfixture
modules, so no new harness or weakened check is needed.

Frozen CI log: `/tmp/loom-ci-226-main-job.log`.
Disposable tidy proposal: `/tmp/loom-jsonrpc-integration-tidy.mod` and `.sum`.

Candidate validation: all three integration packages compile in readonly mode
(`/tmp/loom-910-readonly-green.log`); existing module lint reports zero issues
(`/tmp/loom-910-lint.log`); `go mod tidy -diff` exits zero with no output
(`/tmp/loom-910-tidy.log`). The complete delta adds three manifest lines. No
runtime/generator code, generated output, dependency version or tests change.

Delivered as `877d4135f`, pushed and report confirmed closed. Independent exact
review: `/tmp/loom-910-independent-final-verdict.json`; no unresolved findings.
