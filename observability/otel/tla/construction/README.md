# Telemetry construction ownership (#763)

`observability/otel.New` constructs trace, metric and log providers in that order.
The parent implementation publishes each provider immediately and returns on a
later error without cleanup. The implemented owner stages providers and their
shutdown callbacks, publishes only after all enabled stages succeed, and uses
one reverse-order cleanup function for rollback and Runtime.Shutdown.

The model enumerates all eight enabled-stage subsets, four failure selections,
and success/error outcomes from each cleanup callback. A disabled failure stage
cannot fail. It checks that failed construction changes no globals, every
acquired callback is attempted, attempts follow reverse acquisition order, and
successful globals contain precisely the enabled providers. A cleanup error
must not stop later attempts. An attempted callback may fail to release its
resource; the model does not equate an attempt with successful reclamation.

Publication is abstracted as one final action. Actual SDK setters remain
sequential; concurrent bootstrap and atomic global snapshots are outside the
contract. The model assumes callbacks return; implementation uses an
uncanceled, bounded rollback context, which cannot forcibly stop a callback that
ignores cancellation. Actual error identity, context values/deadlines and SDK
constructor behavior are tested in Go rather than abstracted as proof claims.

Correspondence: Acquire represents each successful provider constructor and
registration; Fail represents its error branch; Cleanup is the shared reverse
shutdown loop; Commit is delayed SDK publication. Runtime success shutdown uses
the same loop and caller context, while failed construction creates a fresh
cleanup deadline. The test `TestFailedConstructionPreservesGlobals` uses actual
metric and log constructor failures. The pinned trace exporter accepts the same
invalid endpoint during construction, so no trace-failure repro is claimed.

Use TLC1.7.4 (JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`)
and Corretto25.0.4.1. From this directory:

```sh
java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -workers 1 -metadir /tmp/loom-763-state-checked -config checked.cfg Construction.tla
```

Use distinct output directories for the other configurations. Results:

- current: FailedGlobals counterexample,81 generated/distinct states.
- publication_only: AllCleanupAttempted counterexample,81 states.
- wrong_order: CleanupOrder counterexample,156 states.
- checked: all invariants pass,169 generated/distinct states.

Direct cleanup-loop tests cover reverse order, all callbacks after errors,
joined error identities, empty acquisition, retained context values and bounded
uncanceled rollback even when the initiating context is canceled. Existing
bootstrap and disabled-section tests cover successful publication. These checks
and review of callback registration connect the model to concrete SDK providers;
no test claims to prove garbage collection or force shutdown of uncooperative
external code.

Independent readiness review repeated all four model outcomes. The implementation
registers each successful SDK provider before any subsequent construction and
publishes globals after all enabled providers succeed. The full runtime and
HTTP/gRPC observability race suites pass; direct cleanup tests retain error
identities, caller context on success, and fresh rollback deadlines/values.
