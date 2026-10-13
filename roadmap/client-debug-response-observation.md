# Passive client Debug response observation (#827)

Status: implementation and bounded acceptance complete; independent final review pending.
Parent: `66c61745673d797d4d15c860210fc082afc80fed`.

## Contract and owner

`NewDebugDoer` eagerly reads a response prefix before returning it. A read error
replaces the real response with successful diagnostic text and loses its Close.
Even successful capture delays an unread streaming response. The public HTTP
guide promises unchanged caller bytes. The delegated development-speed/safety
policy supports a passive 64 KiB preview with explicit omission.

Generalize the delivered `transport.CaptureRequestBody` read-observation owner
and type to body-read names, updating its server consumers without compatibility
aliases. Retain the separate accepted-write response observer. Shared bounded
retention, detached snapshots, exact Read and Close forwarding, and no lock
across underlying I/O remain authoritative. The client must not own a second
replay or capture buffer for response bodies.

Each exchange owns its captured request, response metadata, observation state
and once-only automatic print. Return the response without reading it. Print
automatically at observed EOF or caller Close (including Close failure); print
nil/NoBody immediately. A non-EOF error does not imply EOF: a caller can continue
reading, and its eventual Close prints the observed incomplete state. Never drain
or close on the caller's behalf. Repeated Close still reaches the source but
cannot duplicate automatic output. Sequential reuse while an earlier response
remains open must retain each exchange's identity. Concurrent Do support is not
introduced; concurrent Close can unblock Read without an observation lock.

Explicit `Fprint` prints the latest exchange's detached snapshot without reading
its body. This may be incomplete before EOF. For JSON and form content, omit an
incomplete preview rather than expose an unredacted fragment; retain existing
complete-body redaction and mark non-EOF state. Oversized capture discards its
preview and prints the existing omission marker. Unknown content types remain
raw diagnostic previews. Preserve current request capture/GetBody/replay and
request-capture-error behavior. Existing #718 sensitive-name policy is separate.

## Sequence

- [x] Inspect original failure, public diagnostics contract and delivered capture.
- [x] Add direct error/Close/read-count/Fprint and empty-body witnesses.
- [x] Model eager reads, replacement errors, duplicate completion and wrong-exchange
      ownership; compare passive exchange-owned completion.
- [x] Independent readiness review before production.
- [x] Generalize read-observation owner and adopt per-exchange passive response state.
- [x] Complete bounded runtime/race, timing, redaction and existing-request controls.
- [x] Update public/owning guidance and freeze final candidate.
- [ ] Independent exact final review.
- [ ] Root integrates atomic commit, closes #827 and preserves compact cleanup receipt.

## Acceptance

Parent direct tests must fail for eager read, partial/zero-byte read errors,
lost Close identity and nil-body handling; normal EOF and request-capture controls
remain. Candidate must preserve each requested Read length and original returned
bytes/error; Close counts/errors; nil/NoBody identity; empty bodies; early Close;
EOF with bytes; non-EOF errors followed by later reads; and unread streaming
responses returning before producer completion. Test automatic print once at
EOF/Close, repeated explicit snapshots without I/O, and overlapping exchanges.
Cover limit-1/limit/limit+1 and large payloads, complete nested JSON/form redaction,
partial structured omission, malformed structured omission, headers and URL
redaction, plus existing request capture failure and replay tests. Test Close
unblocking a blocked Read and snapshot concurrency under race. Run affected
`http`, `http/middleware`, and `observability/transport` packages, scoped lint and
the model. No generated code changes or generator corpus gate.

## Evidence

The exact parent fails six leaf cases: passive Do, two partial/zero-byte errors,
Close identity, nil panic, and NoBody identity. Existing request/error/redaction/
cap controls pass (0.163s). Logs: `/tmp/loom-827-readiness-red.log` and
`/tmp/loom-827-controls.log`. Scoped HTTP lint passes after closing empty test
responses. TLC Eager/Replace/Duplicate/Latest exit 12 with their intended
invariant failures (2/2, 6/5, 17/14, 41/31 generated/distinct states); Checked
passes 2078/855 states. `/tmp/loom-827-model-results.json` records each run. Models abstract byte/error
identity and two exchanges, not HTTP framing, redaction parsing or Go races;
corresponding direct and runtime tests establish those obligations.

## Implementation and final acceptance

`transport.CaptureBodyReads`/`BodyReadCapture` replace the request-specific names
without aliases; implementation semantics are unchanged. Server Debug adopts the
renamed owner. Client response capture deletes eager reading, replay and synthetic
error-body replacement. A `debugExchange` owns immutable captured request bytes,
redacted metadata, passive response snapshot and completion once. EOF/Close
callbacks retain that exchange instead of consulting the doer's latest state.
Fprint is a detached snapshot; it no longer reads or resets memory-backed bodies.
The original client request prefix/GetBody algorithm and failure behavior remain.

Readiness was independently approved by root before production. All original
parent failures now pass. New controls cover non-EOF errors followed by resumed
reads, bytes with EOF, early/repeated/error Close, nil/NoBody, empty response,
sequential overlapping exchanges, limit-1/limit/limit+1/large bodies, complete and
malformed JSON/forms, incomplete structured omission, snapshot-only diagnostics,
unread pipe return, and GetBody transport-body identity. Existing header/query/
request redaction and request-capture failure tests remain. The existing full
response redaction test now consumes the body before expecting its content in a
snapshot; its original transport-byte and secret-exclusion assertions remain.

Concurrent Close forwards before printing and need not wait for an in-flight
read. The deterministic gated test proves Close completes while the read is still
blocked, records an incomplete automatic summary, then permits late bytes/error
to reach the caller and a later explicit snapshot. Automatic output stays exactly
once. Another test snapshots during ongoing reads; shared owner tests retain
source-size/error/Close identity, detached snapshots and capture-lock ordering.

Initial affected packages passed HTTP3.987s/middleware.366s/transport.281s. Race
middleware1.333s/transport1.290s passed. HTTP race initially exposed a test-only
`require.Same` misuse for an io.NopCloser value; switching the identity assertion
to equality preserved its actual purpose. Full HTTP race then passed6.869s.
Scoped HTTP/middleware/transport golangci-lint reports zero issues. Evidence:
`/tmp/loom-827-{initial-green,completion,owner-race,http-race,final-lint}.log`.
Final focused race rerun (ten repetitions, 1.885s) covers subsequent test cleanup
changes: `/tmp/loom-827-final-focused-race.log`. The unchanged
readiness model and shared observer model remain the bounded proof evidence;
no generator/source-output changes or corpus checks are involved.

The public HTTP guide and both skills describe EOF/Close logging timing,
incomplete previews and unchanged source I/O. Direct users of the recently added
request-only observation API should use the generic `CaptureBodyReads` and
`BodyReadCapture` names. No public logging configuration or compatibility alias
is introduced. Sensitive-name expansion (#718) remains separate.
