# Durable workflow roles (#1024) acceptance record

Implementation parent: `f0c7a2b8c35374f042b96ca04d339e8cb1e6c729` on `main`.
All six capability designs/plans and their formal sources were already present
as uncommitted preparation; `.mcp.json` is pre-existing and remains untouched.
Only #1024-owned files and shared documentation hunks belong in this ticket.

## Workflow and scope

The implementation recital covers native prerequisite/pin, lifecycle and
registration, admission/lookup, then real process and SQLite/PostgreSQL checks.
Behavioral regressions precede implementation after compile-safe interfaces.
One exact-final-diff independent review authorizes one atomic ticket commit.
No generated output changes or dependency repository development belong here.

## Native prerequisite

Selvedge v0.8.1, source `5adb66608169ad8f28fa88281b514ca36e501cee`, module sum
`h1:LNcpZUDNAhSIPuJMxJMvR1rRT6beE/dyK2IqxA80Cdg=`; Go 1.27.3.
The release check records source correspondence and the isolated race-enabled
probe. Native shutdown retries only `ErrShutdownInterrupted`; unmarked results
are final, even if they wrap a context error. The native handle never escapes.

## Inventory dispositions

The plan's exact `rg` inventory was run before implementation. `execution`'s
Activity/Future/adapter contracts retain their behavior; the new Workflow
contract sits beside them. `execution/selvedge/activities.go` gains the optional
worker lifecycle gate while New(app) remains caller-owned. Existing registration,
retry, timeout and scope tests remain regression controls. `options.go` retains
native activity-option projection. The prior activity formal models remain
unchanged; workflow_roles models supply this ticket's design evidence.
`docs/execution.md` and the two Loom skills document the new ownership/use boundary.
No transport, generator or consuming-project implementation is in scope.

## Results

- Release-preparation consumer probe: `go test -race -v -count=1 .` in an
  isolated v0.8.1 module passed both producer and worker completion cases.

- Native prerequisite: `go test ./execution/selvedge -run '^TestWorkflowNativePrerequisite$' -count=1` passed on pinned v0.8.1.
- M2 compile-safe interface check: `go test ./execution/... -run '^$'` passed.
- M2 red tests: lifecycle/registration/start-shutdown race command failed against
  explicit not-implemented bodies, including bounded startup synchronization.

- M2 lifecycle, registration and native-regression race selection passed after
  implementing the shared gate. Retained activity registries seal through it.
- M3 admission/lookup behavioral tests failed against the compile-safe stubs
  before implementation; operation-drain also failed before admission existed.
- Source inspection identified a native-codec boundary omitted by the plan's
  lint inventory. Independent design review accepted one exact-file exception
  for `workflow_encoding.go`; native input bytes and marshaler semantics remain
  the accepted contract. All other JSON stays v2; no dependency bug/workaround.

- M3 focused admission/lookup race checks and ordinary `go test ./execution/...`
  passed. Real SQLite input-identity checks passed for Producer and Worker.
- Separate-process and single-process examples passed; the worker is killed at
  an acknowledged HTTP activity boundary, then restarted with the same database.
- `go test -race ./execution/... -count=1` passed (9.318s for the adapter),
  covering new roles and unchanged activity registration/options/scopes.
- `go test ./scripts/jsonv2check -count=1` passed. The exact native-codec exception
  rejects sibling, other-package and generator imports; v2 determinism remains
  enforced even at the exception path.
- `make fmt` and `make lint` passed. Formatter-only import rearrangements outside
  this ticket were removed; no semantic edits from other tickets are included.
- `make test-workflow-postgres` passed with race detection (7.276s), on official
  PostgreSQL 18.6 aarch64, Go 1.27.3 and Selvedge v0.8.1. The disposable image is
  `postgres:18.6@sha256:bd5dfa494a5a259704365813dbfa5fa6b3e69be7348d43b547f540c730478583`.
  [Upstream release metadata](https://www.postgresql.org/about/news/postgresql-186-1711-1615-1519-1424-and-19-beta-3-released-3365/)
  and official-image metadata established the latest stable pin. The helper
  provisions loopback-only storage and removes its own container and volume.
- The initial PostgreSQL race run exposed a test log collector's embedded
  `bytes.Buffer.ReadFrom` bypassing its lock. A private buffer field removed that
  path. Readiness now checks TCP to exclude PostgreSQL's temporary init server.
  The passing rerun includes both corrections.

- Exact-diff review identified subprocess ownership in the new process harness.
  Builds, producers and workers now use `internal/testprocess` execution methods
  and its process guardian, including explicit stdout capture for producer Run.
  Intentional worker SIGKILL remains followed by wrapper Wait; the shared owner
  also cleans descendants on test-process loss. The affected race-enabled
  SQLite/single-process checks passed (7.657s), and explicit PostgreSQL acceptance
  passed (5.766s). `make lint` passed again after this correction.

## Scope and correspondence

No generated code changes, default proof gates, native source patches or
consumer-specific policies are included. The native codec bridge preserves the
released serializer's semantics; it is not a dependency bug workaround. The
[model README](tla/workflow_roles/README.md) maps each Go owner and regression
surface to the bounded design evidence. Model assumptions/transitions did not
change during implementation, so the recorded opt-in runs remain applicable.
The completed design and historical native probes live beside those models;
active roadmap work contains only the remaining ticket plans.

Atomic acceptance requires the final staged diff against the parent above to
receive independent review with no unresolved findings before commit. Review
verdicts are recorded in the task's review messages; no post-review evidence edit
is necessary. Publication follows the repository's ordinary exact-source CI.
