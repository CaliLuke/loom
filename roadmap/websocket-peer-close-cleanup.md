# WebSocket peer-close cleanup ordering — #871

CI run `38401076601`, job `115308557245`, fails the generated
`TestCollidingBodyRoundTrips` at client Close with `websocket: close sent`.
Three focused repetitions on local main pass, so a deterministic ordering witness
is required before implementation. The shared shutdown correction is now ready for final review.

1. [x] Inspect generated fixture, shared shutdown owner and Gorilla close handler.
2. [x] Gate the configured close handler after it sends the normal acknowledgement
   but before the physical reader publishes terminal state; reproduce client Close.
3. [x] Model that ordering and compare shutdown-owner classification with treating
   every close-frame-already-sent result as a fresh write failure.
4. [x] Independently review the narrow contract and acceptance, then implement at
   the shared shutdown owner. Preserve ordinary data-write failure behavior and
   independent physical cleanup errors; do not patch generated callers.
5. [x] Run direct race/order controls and the existing generated round-trip test;
   document exact output invariance if no generator is changed.
6. [x] Obtain independent exact-diff review, commit, push and close the report.

The sole reader delegates close acknowledgement to the configured Gorilla handler.
That handler can mark the native connection close-sent before `NextReader` returns
and `readPump` publishes termination. Application Close can win this small window
and attempt final JSON, receiving `ErrCloseSent`. The shared shutdown owner already
promises idempotent closure; the proposed correction recognizes an already-sent
close frame during shutdown while preserving a real socket-release failure.
Normal application writes must still fail after native close framing.

The original deterministic final-JSON witness fails both normal and physical-error
cases under `-race` (0.826s); the same gate now also covers close-control shutdown.
Tracked as https://github.com/CaliLuke/loom/issues/871 .

## Reviewed-contract candidate and bounded acceptance

Treat Gorilla's already-sent close-frame sentinel as completed framing only in
`releaseConnection`'s shutdown result classification, after the selected terminal
write attempt. It must not trigger a new best-effort peer wait when the write
returned that state: a configured close handler may still be on the reader stack.
Join the separately recorded physical-close error unchanged. Do not alter reader
cause selection, inbox disposition, timeouts, successful final-write peer waits,
ordinary WriteJSON errors or generated templates.

The four deterministic cases (final JSON/control × normal/physical-error) fail
under the original owner (0.460s). Existing real generated round-trip test passed
three local repetitions but failed CI; its failure is supported by the deterministic
ordering witness, not asserted reproducible on demand. The new bounded model
reproduces spurious cleanup (13 states), rejects indiscriminate cleanup suppression
(8 states) and suppression in data writes (14 distinct states); owned classification
passes 44 generated/24 distinct states. It does not prove timeout/callback behavior.

After correction, run the gated cases and existing WebSocket lifetime, queued-input,
configured-handler, close failure, bounded wait and cancellation tests under race.
Add an explicit independent write-failure control if existing tests do not assert
that error's identity. Run the existing generated type-identity and lifecycle cases
without changing their assertions. No generator source or design changes are
planned; output byte invariance follows from identical generator inputs, and the
existing compiled generated harnesses supply the runtime boundary checks.

## Final implementation and evidence

Independent readiness review approved the shared owner. `releaseConnection`
recognizes only the raw native `ErrCloseSent` identity after the existing
successful-write wait branch. Wrapped application encoding failures remain
errors, and physical release remains unconditional. The wrapped-encoding test
checks both the application cause and independent physical-close cause, including
on repeated cleanup. Ordinary late writes still return `ErrCloseSent`.

All WebSocket runtime cases pass under race (5.061s), including the four gated
peer-acknowledgement cases, wrapped encoding, queued final input, physical errors,
configured handlers, attachment cancellation, bounded waits and terminal causes.
The unchanged generated type-identity and lifetime harnesses pass (7.469s),
including generated compilation and real client/server interactions. No generator
or design source belongs to this change; there is no generated output difference.
The model inputs match the independently reviewed readiness inputs, so the
44-generated/24-distinct corrected result remains applicable without another run.
Error wrapping is deliberately outside that model and covered by the Go test.

Delivered `4f569fad1`, independently reviewed, pushed and confirmed closed.
Final verdict: `/tmp/loom-871-independent-final-verdict.json`. Temporary output
cleanup is recorded in `/tmp/loom-871-cleanup-receipt.json`.
