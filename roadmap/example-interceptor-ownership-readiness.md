# Example interceptor ownership readiness

Status: independently approved by `/root/review_artifacts` on 2026-10-12;
implementation complete; integrated review pending. Owning ticket: #741.
Parent: `41b4a8a35d5effc908f3533da4a2df40e013b59f`, which delivers
#670. Worktree: `/Users/luca/.codex/worktrees/example-interceptor-ownership/loom`.
Root prepared this record from the live issue,
`codegen/service/example_interceptors.go`, its existing tests and fixtures, and
the shared artifact ownership evidence.

## Scope and acceptance

Example interceptor files belong to the application. The declaration must carry
`SkipExist: true` for both server and client files, and the shared writer must
decide whether the destination exists relative to its output directory. Delete
the two cwd-relative `os.Stat` guards and the resulting unused import. Do not
add another existence check or change rendered interceptor code.

An existing output file retains its exact bytes and identity. An absent output
file is generated even when the same relative name exists in the process's
working directory. Server-only, client-only, both, and no-interceptor designs
retain their expected declarations. Existing shared writer tests own creator
races, occupied symlinks/directories and transaction rollback; this ticket does
not reimplement or broaden those mechanisms. #751 is excluded.

## Files and test-first sequence

1. Extend `codegen/service/example_interceptors_test.go` with table-driven
   declaration and rendering regressions for server/client files. Cover distinct
   cwd/output directories, an existing output file, a cwd-only file, fresh
   output, and repeated rendering. Assert both file bytes and preservation of
   the existing file identity. Use existing server/client fixtures and
   `t.Chdir`; do not run cwd-changing tests in parallel.
2. Run `go test ./codegen/service -run '^TestExampleInterceptors' -count=1`
   before the implementation change. Record the failed preservation and cwd-only
   generation cases. A missing command/tool is not a reproduced defect.
3. Correct declarations in `codegen/service/example_interceptors.go`. Keep
   service/interceptor analysis independent of live filesystem state.
4. Rerun focused regressions and `go test ./codegen/service -count=1`.
   Existing rendered goldens must remain byte-identical; no golden update is
   expected. Run scoped formatting, vet and applicable repository lint checks.

## Generated evidence

Readiness reproduction on 2026-10-12: all four overlay cases failed as expected
with Go 1.27.2 on main parent `096a793b` plus the staged #670 candidate
(`790fbfeb9969ece2f9b0286828cdac9436331b4b534a5a1f1e53d2708f5e3a80`).
Both server/client existing output files were overwritten; both cwd-only files
suppressed the output declaration. The interceptor source is unchanged from the
parent, SHA-256 `48c3e6778cefa987670b997c017f1c1883368b9b2b354de34132b0de7eac3665`.
The retained probe is
`/var/folders/v7/x61rxqvx60q5w_p9jwgc9t_m0000gn/T/loom-741-readiness-izaek4i2/probe_test.go`
(SHA-256 `3bae95637fcee8ab8d7bb754fb3824db9da1d31b518f7b27b27f8b9cf0e54789`).
Command: `go test -overlay <probe-directory>/overlay.json ./codegen/service -run '^TestReadinessExampleInterceptorOwnership$' -count=1`.
This is a red declaration-seam probe, not final candidate or CLI acceptance.

Use `MultipleInterceptorsExampleDSL` for the real CLI probe: it exercises both
server and client declarations. Generate clean baseline and candidate modules
with `loom gen` and `loom example`, retain the outputs, and compare all generated
and example file bytes. Repeat candidate generation in an independent process
and compare bytes again. Build and vet each distinct baseline/candidate module.
Use explicit framework replacements so the CLI, imported fixture and runtime
all use the recorded source, rather than assuming an environment override is
honored. Fresh outputs should have no byte differences.

In the candidate CLI module, edit both interceptor files with a valid Go
comment, then invoke `loom example` from the design directory with the module
root as output. Both edits and file identities must survive. Also exercise a
cwd-only interceptor path with a separate fresh output directory. Record the
exact commands, source revisions, file comparisons and compile receipts here.
The existing shared artifact model already proves the selected write policy
within its assumptions; this change only corrects a declaration and removes a
competing filesystem decision. Link its implementation correspondence to these
regressions instead of adding a duplicate model.

## Review and handoff

The intended source diff changes two Go files, with this readiness/evidence
record and a concise correspondence addition to
`cmd/loom/tla/artifact_write_intent/README.md`. Existing public documentation
already promises preservation; update it only if the implementation reveals a
missing user-facing detail. No shared skill change is expected.

Freeze exact files and hashes after checks. A fresh reviewer must approve the
integrated diff against its intended parent before the single #741 commit.
Root updates the master tracker, pushes, then removes temporary comparison
outputs once no reviewer needs them. Delivery is pending.

## Implementation evidence

The source removes both cwd-relative existence checks and declares `SkipExist`
on server and client example files. No rendered sections, shared writer,
installation protocol, model state machine, or checked-in fixture changed.
The shared model README links the declaration and rendering regressions to the
existing ownership invariant.

With unchanged parent source, `go test ./codegen/service -run
'^TestExampleInterceptors' -count=1` failed (0.566s): server/client output cases
overwrote the original bytes, cwd cases returned zero declarations, and fresh
cases lost edits on repeat. The existing declaration matrix also failed its
new create-only assertion. After the fix, that command passed (0.545s), and
`go test ./codegen/service -count=1` passed (6.640s), including all existing
interceptor goldens. The ownership table also checks exact file identity and
omission of skipped files from render results.

The immediate-parent comparison used an exact `git archive` of the full SHA
above, candidate source in this worktree, and Go 1.27.2 on Darwin/arm64. Both
CLIs were built as `loom`; every child used `GOWORK=off`. Each isolated module
had module path `example.com/ownership` and an explicit
`replace github.com/CaliLuke/loom => <matching-source-root>`, so launcher,
fixture import, helper, and runtime resolved the recorded source. Its design
imported `codegen/service/testdata`, declared `API("ownership", func() {})`,
and invoked `testdata.MultipleInterceptorsExampleDSL()` from `init`.

The comparison command sequence for baseline, candidate, and an independent
candidate repeat was:

```sh
go build -o <source-specific-bin-dir>/loom ./cmd/loom
# In each isolated fixture module, with the matching source replacement:
go mod tidy
loom gen example.com/ownership/design -o .
loom example example.com/ownership/design -o .
```

All 26 generated and example files compared byte-for-byte equal between parent
and candidate and between independent candidate processes. Module and design
inputs were excluded from the output comparison. Baseline and candidate each
passed `go mod tidy`, `go build ./...`, and `go vet ./...`; the identical repeat
did not repeat compilation.

Two additional real CLI probes passed:

1. Append `// application customization` to both candidate interceptor files.
   From `<module>/design`, run `loom example example.com/ownership/design -o
   ..`. Both exact byte sequences and device/inode identities survive; neither
   filename appears in stdout. The edited module passes build and vet.
2. In a fresh module, place both interceptor names only under
   `<module>/design/interceptors`, containing a valid Go sentinel comment.
   From `<module>/design`, run `loom gen example.com/ownership/design -o ..`
   and `loom example example.com/ownership/design -o ..`. Both module-root
   interceptor files are created and reported in stdout. The cwd sentinels'
   bytes and identities survive. The complete module passes tidy, build and vet.

These preservation and cwd-independent declarations are the only intended
behavior differences. Fresh output bytes are unchanged. There is no new
rejected input; the inherited gen-tree admission and filesystem ownership
limits remain unchanged.

`go vet ./codegen/service`, `make lint-namescope lint-gofmt`, and
`git diff --check` pass. `make lint-json-v2` is blocked by the existing
`jsonrpc/codegen/recursive_projection_test.go` import of `encoding/json` at line
45 and use of `json.RawMessage` at line 99. `git grep` at the exact parent
confirms both, and candidate/archived-parent file SHA-256 is identical:
`4607ff91ef6508858594283b57cab88a9ca6246f0977752ced0f180afff2f766`.
That unrelated file is not part of #741.

Integration parent is `93f513be`. That separate maintenance commit corrects the
JSON v2 harness violation, and its lint and both-order wire checks pass. Between
the comparison parent and integration parent, only roadmap records and that
unrelated test harness changed; interceptor generator/runtime inputs remain
identical, so the bounded #741 output evidence remains applicable.

Review scratch is
`/var/folders/v7/x61rxqvx60q5w_p9jwgc9t_m0000gn/T/loom-741-compare.0s9puy_n`.
It contains `compare.py` (the exact commands and assertions), `receipts.json`
(every CLI/build/vet command, working directory, exit code and output), three
output hash manifests, `summary.json`, and `red.log`. These temporary inputs
and checker receipts are retained only until review and then removed; the
reproduction inputs and concise results above remain durable. No #670 scratch
or previous worktree was reused or changed.
