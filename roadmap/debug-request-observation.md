# Transparent Debug request observation (#680)

Status: implementation and bounded acceptance complete; independent final
review pending. Root approved the exact readiness proposal before production.
Parent: `fa5897578` (shared bounded Debug response observation, #679).

## Contract and ownership

Debug must observe the request's actual consumer without becoming a consumer.
The original middleware eagerly read the complete body, replaced read failures
with successful diagnostic text, and discarded the original Close implementation.
This altered request-size policy and delayed handlers until uploads finished.

The accepted delegated Debug policy is a 64 KiB preview limit and explicit
omission marker, preserving all application bytes. Apply it consistently to
requests. Preserve the existing one-summary-at-handler-return log timing.
Log only bytes returned to the downstream reader, in request-before-response order. If EOF
has not been observed, label the preview `[request body capture stopped before
EOF]`; this reports observation state, not a claim that unread bytes exist.
If capture exceeds the limit, discard the whole preview and print
`[body omitted after 65536 bytes]`. Never drain to complete a log or close on the
handler's behalf. Nil and http.NoBody remain empty without an incomplete marker.
Read errors stay on the application path, with partial bytes intact; the log's
incomplete marker avoids falsely presenting those bytes as a complete body.

The shared transport observation package owns a passive request
ReadCloser decorator and snapshot state. Forward each Read and Close directly,
with exactly its returned count/error; do not hold the capture lock while the
underlying Read or Close blocks. Reuse the response owner's bounded retention
primitive through a private generic body-capture owner, while keeping response
accepted-write/ReadFrom semantics and request consumed-read semantics distinct. When Debug wraps a size policy,
the policy itself is a downstream reader and its one-byte oversize probe is
observed too; do not claim the preview only contains endpoint-visible bytes.
The Debug middleware renders snapshots and attaches the observer; it must not
implement its own replay, draining or retention policy.

The client debug owner was inspected. It eagerly captures a bounded prefix and
replays it, or reads GetBody. That algorithm cannot meet passive server timing
and error ownership, so it is not reused here. Its separate response-read-error
replacement defect is separately reproduced and filed as [#827](https://github.com/CaliLuke/loom/issues/827), outside this atomic server request ticket.

## Sequence

- [x] Inspect original #680, Debug, client capture, response capture and body policy.
- [x] Add original read-error and ownership/timing/body-policy counterexamples.
- [x] Model eager observation, replacement errors, lost Close and bounded preview;
      check current counterexamples and the passive design.
- [x] Independent readiness review: approve shared owner and log semantics.
- [x] Implement shared passive capture and replace eager Debug request path.
- [x] Verify direct seam and middleware race suites, body-policy order, blocked
      upload/Close behavior, and response/SSE controls. No generator changes.
- [x] Update owning/public guidance and freeze exact final diff for fresh review.
- [ ] Root integrates one atomic commit, closes #680, and records cleanup.

## Bounded acceptance

Original `(0, boom)` and `(partial, boom)` errors retain identity and bytes.
Each downstream Read matches one source Read with the same requested buffer size;
no source reads occur before handler entry or after its last read. Repeated
Close calls and their errors reach the original body. Close can unblock a
concurrent Read. Nil bodies and http.NoBody are preserved. Both middleware orders
with RequestBodyPolicy retain 413/request_too_large, content-length early
rejection, unknown-length probe and exact-limit success. Capture limit cases are
0, limit-1, limit, limit+1 and large bodies, with full application bytes/errors.
Partial/no-read bodies are explicitly incomplete and never drained. Streaming
response bytes can reach the peer before an upload completes. Existing response
capture and Debug SSE/status/interface controls remain green. Race tests cover
snapshot observation and concurrent Close, not unsupported concurrent body Reads.

No generated artifacts, OpenAPI or DSL behavior changes. The request body cap
bounds retained preview bytes, not arbitrary headers, the request itself or the
underlying reader's allocations. HTTP server ownership still closes the body.
EOF is observed explicitly; a successful exact-length Read without EOF remains
incomplete in the diagnostic even if Content-Length matches.

## Readiness evidence

At exact parent `fa5897578`, `go test ./http/middleware -run TestDebugRequest
-count=1` fails thirteen leaf cases: original zero/partial read errors, unread
and partial ownership, all four size-policy order/length combinations, two
oversize previews, Close error delegation, nil body, and delayed handler entry.
The exact-limit/under-limit/empty preview and complete-log timing controls pass.
The initially mistyped service error name was corrected to the existing
`request_too_large` before freezing the final red evidence.
Log: `/tmp/loom-680-readiness-red.log`.

Existing Debug response/status/SSE and basic output controls pass with race
detection (1.461s): `go test -race ./http/middleware -run
'^TestDebug($|EmptyBodies|GeneratesRequestID|Response)' -count=1`.
Log: `/tmp/loom-680-existing-controls.log`. Scoped middleware lint: 0 issues,
`/tmp/loom-680-readiness-lint.log`.

TLC 1.7.4 negatives Eager/Replace/Close/Unbounded each exit 12 on the intended
invariant, respectively 2/6/12/27 generated and distinct states. Checked exits 0,
2,866 generated / 1,845 distinct states. Logs: `/tmp/loom-680-model-*.log`;
compact results: `/tmp/loom-680-model-results.json`.

This readiness receipt covered tests/model/plan only; the implementation gates
that followed are recorded below.

## Final implementation and acceptance

`transport.CaptureRequestBody` now owns a passive ReadCloser and detached,
synchronized snapshots. Promoted Close preserves the original implementation;
Read calls the source before taking the state lock, records its returned prefix
and EOF observation atomically, and returns its original count/error. Request
and response capture use the private `bodyCapture` retention primitive; response
accepted-write, partial-write and ReadFrom lookahead policy remains unchanged.
Debug no longer eagerly reads or replaces requests. It snapshots consumed request
bytes after the handler, before rendering the existing response summary.

Direct source tests cover `(0,nil)`, partial success, zero/partial custom errors,
zero/partial EOF, repeated original Close errors, nil/NoBody, detached snapshots,
cap boundaries, nested observation, bounded state across reads, concurrent
snapshots, and Close unblocking an active Read. Middleware tests retain the
original thirteen red cases, add exact-size policy success in both orders and
known/unknown length, preserve complete-log ordering, and prove real-network SSE
headers/data arrive before an upload pipe closes. The network test waits for
handler/log completion before inspecting the summary, avoiding a client/server
completion race.

Final gates (Go 1.27.2, this worktree):

- `go test -race ./observability/transport ./http/middleware -count=1`: pass,
  1.172s/1.821s, `/tmp/loom-680-final-owner-race.log`. Includes all existing
  response writer, positive Hijack, partial I/O, cap, SSE and body-policy tests.
- `go test -race ./http ./clue/log -run
  'Test.*(Body|RequestDecoder|ParseFormWithLimit|SSE|Debug|HTTP)' -count=1`: pass,
  2.960s/1.289s, `/tmp/loom-680-runtime-controls.log`.
- Focused real streaming upload and four exact-policy cases: pass 1.513s,
  `/tmp/loom-680-streaming-policy.log`.
- Scoped `golangci-lint run ./observability/transport/... ./http/middleware/...`:
  0 issues, `/tmp/loom-680-final-lint.log`; `git diff --check`: clean.
- The readiness model inputs are unchanged; its checked and counterexample
  receipts remain valid. The model README now maps the concrete implementation
  and runtime checks. Response retention extraction is covered by its unchanged
  direct/race suite; no new response policy or model rule was introduced.

Public HTTP guidance, consumer skill, framework owning guidance and formal-model
index describe passive request observation and the adopted bound. No generated
artifacts changed; no generator corpus, dependency update or overall CI-green
claim is part of this runtime ticket. #827 client response ownership and #695
inherited-bound admission remain separate outstanding work.
