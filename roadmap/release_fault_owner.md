# Release fault ownership (#947)

## Evidence

Required CI run 38611345499 failed TestSlowReleaseKeepsLease because the direct
stop caller returned nil. A controlled parent regression delays the successful
stop-publication reply until the asynchronous worker enters releaseReplyHook.
The same expected-error assertion fails: the asynchronous consumer receives the
fault and the direct consumer legitimately retries the retained release.

The shared hook also checks then stores its fired flag non-atomically, allowing
concurrent unrelated releases to enter one channel gate twice. Neither failure
establishes a production lease or release defect.

## Sequence and design

- [x] Reproduce the asynchronous-consumer-first order with an explicit reply
  barrier and file #947 only after the parent assertion fails.
- [x] Model current wrong-caller admission and non-atomic gate ownership, then
  check scoped atomic admission with deliberate direct-delivery ownership.
- [x] Bind releaseReplyHook to its explicit context identity, admitting one
  matching attempt with CompareAndSwap. For after-write faults, only claim after
  next succeeds so NOSCRIPT can fall back without consuming the injection.
- [x] Make immediate consumers choose the intended context; use the existing
  sinkless node fixture for the real-worker lease test so the direct operation
  owns delivery. Keep the complete lease advancement and cleanup assertions.
- [x] Test non-target calls and concurrent attempts at the hook seam; retain an
  explicit asynchronous-first public stop control that asserts valid success.
- [x] Run bounded affected miniredis/native Redis races, lint and model checks.
- [x] Independent exact-diff review and atomic delivery as
  `a0642853ce638c19d686fe3e5150fd83126dd66d`; pushed and #947 closed.

Context scoping alone is insufficient: a worker reader could complete the stop
before the tagged direct caller reaches release. The sinkless fixture separates
this test's deliberate direct-delivery ownership from independent worker lease
renewal; the public asynchronous-first control retains the valid competing path.
No runtime production code or protocol changes belong in this correction.

## Acceptance receipt

The controlled parent reproduction fails with the same expected-error/got-nil
assertion as CI. The corrected hook and immediate consumers pass focused races
against miniredis (10.472s) and native Redis at 127.0.0.1:32768 (12.463s), including
before/after replies, slow callback and release lease renewal, stop-intent
responsibility, asynchronous-first public delivery and exact hook admission.
Package golangci-lint reports zero issues. No production source changes.

TLC 2.19 from pinned tla2tools v1.7.4 reproduces both current safety violations
and verifies owned admission plus eventual direct failure. Tool SHA256:
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
The fault model abstracts gate release and a successful retained-release retry;
actual lease timestamps, Redis delivery and cleanup refusal are tested directly.

Delivery review left no unresolved findings; root repeated the focused race
checks three times (26.469s). Final required CI on this commit remains pending.
Exact inputs, review, reproduction/model sources and concise evidence are archived
at `/tmp/loom-947-durable-inputs.tar.gz`, SHA-256
`a76a6e03a2a21b33a92dc5ca260105b1ed53fcc0d5854c9781812834fcc04a7e`.
Activity-checked cleanup is recorded in `/tmp/loom-947-cleanup-receipt.json`.
