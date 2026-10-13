# Debug response transparency (#679)

Base: `00fc9ba8bdebc8b77c42bd30f724707bbbea86a0`.
Status: implementation and bounded acceptance complete; independent final
review pending. User delegated policy choice; root adopted a 64 KiB response
capture cap with an explicit omission marker.

## Contract and owner

Debug must observe an HTTP response without changing supported optional writer
interfaces, ResponseController operations, accepted bytes, write errors, or
stream timing. The current responseDupper loses Flush and Unwrap, advertises
Hijacker unconditionally, logs the last requested status rather than the first
committed final status, and captures attempted rather than accepted bytes.
These are one response-observation ownership defect. Use the existing
observability/transport.CaptureResponse owner and httpsnoop's conditional
interface preservation rather than adding optional methods to responseDupper.

Request body handling is separate ticket #680 and remains unchanged here.
Generator output and full fixture-corpus audits are outside this runtime change.

## Accepted public policy

Debug's GoDoc promises all bodies; it buffers complete responses until the
handler returns. Streaming can therefore grow memory indefinitely. Client debug
has a separately documented 64 KiB capture/oversize-omission policy, which does
not currently define server behavior. Alternatives: preserve complete capture;
limit capture with a visible omission marker; or omit streaming bodies entirely.
Recommendation: use a bounded 64 KiB capture and explicit omission marker while
forwarding the complete response unchanged, consistent with client diagnostics.
The user delegated the choice to prioritize development speed and safety; root
adopted this recommendation. At or below 64 KiB retain the complete accepted
body; above it discard the preview and print `[body omitted after 65536 bytes]`.
All wire bytes still flow, including SSE; capture does not impose a response
size limit or change stream timing. Request capture remains #680.

## Sequential tasks

- [x] Read original #679/#680, public Debug contract and shared response owner.
- [x] Create isolated managed worktree and this recovery plan.
- [x] Add original Flush reproduction and controller/interface/status/accepted-byte
  cases; establish red baseline and unchanged ordinary Debug controls.
- [x] Check meaningful SSE writer and body-policy interaction without generation.
- [x] Model status/stream operations before implementation; reuse existing capture
  status proof and extend only the newly relevant Debug observation seams.
- [x] Resolve body-capture memory policy and record accepted/rejected examples.
- [x] Freeze readiness inputs and obtain independent architectural review.
- [x] Implement shared capture integration after release, removing responseDupper.
- [x] Run direct and affected package/race tests, meaningful runtime SSE controls;
  no generated changes expected.
- [ ] Update public/internal guidance, obtain exact final diff review, then root
  integrates one atomic commit, pushes and closes #679.
- [ ] Preserve evidence and remove temporary models/build outputs after review.

## Acceptance obligations

Original Flush succeeds with a capable underlying writer. Unsupported interfaces
remain absent. Controller deadlines/full duplex and flush errors forward exactly.
First-final status, implicit success, informational responses and 101 agree with
the shared owner. Observation sees only accepted bytes through Write, WriteString
and ReadFrom, preserving partial writes/errors and fast-path behavior. SSE can
open, emit and flush before the handler returns, and failures retain existing
terminal semantics. Ordinary request/debug formatting controls remain stable
except explicitly corrected status/accepted-byte output and accepted capture
policy. No request read/error/Close changes are included.

## Proposed implementation seam

Keep conditional writer interfaces and the status/byte observer in
`observability/transport.CaptureResponse`. Add opt-in bounded body observation to
that owner under the accepted policy; ordinary metrics consumers
continue retaining no body. A small capture option can carry the body limit,
with read-only body/omission accessors on the capture state. Debug consumes the
state after the handler returns and removes responseDupper entirely. It must
not independently implement response commit rules or attach unconditional
optional methods. Avoid adding public Debug configuration until a use case
requires it; a shared capture option is for internal framework consumers.

Write and WriteString observation must use the returned count. ReadFrom may
read ahead before a partial downstream failure: bounded temporary observation
must be trimmed to its returned accepted count, preserve the original reader
and error semantics, and not treat bytes merely read as bytes sent. Keep the
optimized underlying ReadFrom call and test with a reader that avoids WriterTo.
Ordinary capture remains callback-free and body-free. Atomic counters do not
promise that ResponseWriter is concurrent-safe; sequential
writer ownership follows net/http and SSE's existing lock.

## Readiness evidence

The new tests fail against the untouched production owner: controller forwarding,
capability presence/absence, implicit/first-final/upgrade status, all three
partial write paths, and SSE Open. Existing ordinary Debug output and shared
CaptureResponse controls are run separately. Original issue Flush is exercised
through the real SSEStreamWriter, not just a replacement fake implementation.
Logs: `/tmp/loom-679-readiness-red.log`, `/tmp/loom-679-controls.log`.

The bounded Debug model keeps the existing response-capture status model and
adds three isolated current-behavior negatives plus a checked owner. Checker
logs live at `/tmp/loom-679-model-Debug{Status,Bytes,Flush,Checked}.log`; no checker
state or compiled artifacts belong in the commit.

Results: 11 failing leaf cases reproduce the response defects (0.429s).
Existing shared capture and ordinary Debug controls pass (0.404s/0.158s).
Scoped golangci-lint reports zero issues. DebugStatus/Bytes/Flush exit 12 on
the intended invariants (3/3, 6/6, 9/9 generated/distinct states); DebugChecked
passes with 293/75 states. Earlier malformed-model attempts were corrected;
only the final invariant counterexamples count as evidence.

Positive Hijacker forwarding is covered by the existing shared owner test
`TestCaptureResponseWriterForwardsHijack` (included in the passing controls).
Retain it in final acceptance alongside Debug's negative capability assertion.

## Implementation and final validation

Debug now uses CaptureResponse with WithResponseBodyCapture(65536); responseDupper
is deleted. The shared capture state provides detached synchronized Body snapshots
and preserves its existing default body-free behavior. Write/String retain only
accepted prefixes. ReadFrom retains bounded reader lookahead and commits only its
returned accepted count. Once oversized, preview storage is discarded and further
ReadFrom calls pass the original reader directly. This adds no Debug configuration
mode and changes no request-body handling or generated output.

The cap tests were red before implementation (`/tmp/loom-679-cap-red.log`).
Full observability/transport and http/middleware race tests pass, including exact
limit/overflow, all I/O paths, ReadFrom lookahead on partial failures, original
flush/controller/interface/status defects, SSE terminal failure and real network
SSE visibility before handler return. Existing positive Hijacker forwarding remains
in that suite. Scoped lint reports zero issues. The extended model retains its
three original negatives and adds DebugCap (exit 12, 56/27 states); checked remains
0 (293/75 states). No generator outputs are affected or regenerated.

Final logs: `/tmp/loom-679-owner-race.log` (transport 1.483s, middleware 1.497s),
`/tmp/loom-679-runtime-controls.log` (clue/log 1.611s, HTTP runtime 1.655s),
`/tmp/loom-679-final-lint.log` (zero issues). Runtime controls include shared
HTTP logging, SSE lifecycle/stream write policy and request-body policy tests.
Public behavior is recorded in docs/http-guide.md, the consumer skill, and
Debug's GoDoc. Root owns framework-guide/model-index integration and delivery.
