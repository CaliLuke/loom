# gRPC tracing test completion observation (#828)

Status: test-only implementation and bounded acceptance complete; final
independent review pending. Production instrumentation remains unchanged.
Parent: `7b28984a7e312ef2ea5d2981766ce2790a6a747b`.

## Shared ordering defect and accepted scope

The existing test snapshots SpanRecorder immediately after the client Check RPC
returns. That return does not order server OnEnd publication. CI race job
115094496569, run 38335487607, at 063198a2f observed one span instead of two.
Root's local count-100 race run passed, but the gated server publication witness
reproduces the permitted failing schedule deterministically.

This ticket owns test completion observation only. Production gRPC middleware,
OpenTelemetry instrumentation and transport generation remain unchanged. Exactly
two spans, client/server kinds, and both health Check names remain mandatory.

A test-local SpanProcessor should publish to the existing SpanRecorder before
sending a nonblocking completion notification. A bounded context waits for two
published spans using notifications before taking the final snapshot. Do not
sleep, poll on intervals, infer completion from RPC return, weaken the count or
block the processor if extra spans arrive. The same assertion path should run
with ordinary scheduling and a server OnEnd gate released after RPC return.
Gate release must precede shutdown even if a test assertion fails. Collect the
Serve goroutine's shutdown error instead of discarding it; bound RPC and
completion waits and preserve clean client/server/provider cleanup.

## Sequence and acceptance

- [x] Read issue, CI evidence, original test and production wrapper.
- [x] Preserve deterministic parent gate witness and small ordering model.
- [x] Independently review readiness and release the test-only implementation.
- [x] Implement completion observation and normal/gated regression modes.
- [x] Check gate schedule, notification-after-recording and exact assertion
      preservation; exercise a missing-completion deadline without sleeping.
- [x] Run affected package race tests, repeated normal/gated cases and scoped
      lint; prove production/generated files unchanged.
- [ ] Fresh exact-diff review, one atomic root commit and issue closure.

Durable parent witness: `grpc/middleware/otel/tla/span_completion/testdata/parent_test.go.txt`.
Overlay it onto `grpc/middleware/otel/middleware_test.go` without modifying the
checkout. It preserves the exact immediate assertion but gates server OnEnd;
its Serve error is collected to avoid introducing another ignored error. The
witness must fail because the snapshot has one span, not because of cleanup or
compilation. The corrected test must pass that schedule without weakening any
span assertion. No generated-output corpus or dependency change is applicable.

## Readiness receipts

The retained overlay witness fails at the intended exact-count assertion with
one client span (0.485s), `/tmp/loom-828-parent-red.log`. Command:
`go test -race -overlay /tmp/loom-828-parent-overlay.json
./grpc/middleware/otel -run '^TestServerAndClientOptionsCreateTracingSpans$'
-count=1`. The unchanged package control passes with race detection (1.477s),
`/tmp/loom-828-baseline-control.log`; its passing schedule does not invalidate
the deterministic counterexample.

TLC: Old exits 12 on CompleteSnapshot, 7 generated / 6 distinct states;
NotifyFirst exits 12 on the same invariant, 15 / 10; Checked exits 0, 9 / 7.
Logs `/tmp/loom-828-model-*.log`, compact receipt
`/tmp/loom-828-model-results.json`. The model sources and gate witness are
committed inputs only; no raw output is part of the candidate.

That readiness freeze modified no existing test or production file. Root
independently approved its exact seven-file manifest before implementation.

## Final implementation receipt

The original test now runs ordinary and deliberately late server-publication
schedules through one assertion path. Its test-local processor invokes
SpanRecorder.OnEnd before a nonblocking buffered notification. The wait checks
published snapshots after notifications until the expected count is reached or
its context expires. It returns the entire snapshot; extra spans are retained
and would fail the original exact-two assertion. Both original kind and name
assertions remain unchanged. A separate deadline control uses an already-expired
context, with no sleeps. An extra-span control publishes three spans without
waiting between them and verifies none are filtered out.

The gated test confirms exactly one span after RPC return, releases the server
and waits through the same completion owner. A deferred idempotent release runs
before cleanup even on assertion failure. RPC/completion waits are bounded;
Serve reports through a buffered result channel and cleanup checks its result
after Stop with a bounded wait. The previous ignored Serve error is removed.

Acceptance: full package race pass 1.497s, `/tmp/loom-828-owner-race.log`;
100 repeated race runs (ordinary and gated plus deadline/extra-span controls)
pass, `/tmp/loom-828-repeat-race.log`. Scoped golangci-lint reports 0 issues,
`/tmp/loom-828-lint.log`; diff check passes. Model inputs are unchanged from the
approved readiness receipt, so its three results remain applicable.

Only the existing gRPC tracing test, its owning model/reproduction/plan and the
formal-model index change. Production wrappers, dependency pins, generated
artifacts and span contracts are untouched. No generator or broad framework
suite is needed for this observation-only repair. Root owns integration, final
review, the atomic commit and issue closure; no overall CI-green claim is made.
