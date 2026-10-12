# WebSocket connection lifecycle repair

Status: implementation and scoped acceptance complete; frozen for root integration
and independent final-diff review. No commit or push has been made from this worktree.

## Scope and source

Parent: `46b627ef` (the delivered #716 message-admission repair). The owner is
#722; #711, #712 and #714 are witnesses of that same missing connection-lifetime
owner. #739 request timeout/response ordering and gRPC lifecycle work are
excluded. Go is 1.27.2, Gorilla is the root module's pinned version. TLC uses
`tla2tools-1.7.4.jar` (TLC 2.19), Java 25.0.4.1. The tool SHA-256 is
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.

The root agent approved close/cancel on overload from the user's explicit
choice. Root also approved normal handler completion code 1000, handler error
or panic code 1011 with a generic safe reason, separate 1009 individual-message
and 1013 aggregate-overload codes, and explicit finite queue message accounting.
These choices are settled and root approved production implementation.

## Sequential work

- [x] Copy only the 19 lifecycle readiness/model/regression files from the old
  worktree; preserve #716 production and terminal-cause fixes.
- [x] Reproduce runtime and generated witnesses on the actual parent.
- [x] Recheck the old omission and backpressure models on this parent contract.
- [x] Add the empty-message resource counterexample and finite count policy.
- [x] Refine shutdown into publication, bounded control attempt, physical close,
  worker exit and late attachment; expose the writer-lock deadlock alternative.
- [x] Root approval of the concrete API below, then implementation.
- [x] Direct runtime normal/adversarial tests and generated witnesses green.
- [x] Representative parent/current/independent-repeat byte comparison, generated
  compile/vet, relevant full package checks and pinned lint.
- [x] Freeze exact manifest for integration.
- [ ] Root integrates and obtains independent final-diff review.
- [ ] Root commits, pushes, reconciles owner/witness tickets and master tracker.
- [ ] Remove task scratch after validation/review no longer need it.

## Owning API and admission contract

`http.WebSocketStream` remains the sole physical connection owner. Generated
code retains typed conversion, validation, views, lazy upgrade, and protocol
half-close adapters; it does not own reader/watchdog goroutines or duplicate
terminal-close decisions. JSON-RPC's existing logical reader continues ID
routing and sequential request dispatch over this owner's inbox.

The implemented immutable configuration constructor is:

```go
type WebSocketLimits struct {
    MaxMessageBytes int64
    MaxBufferedBytes int64
    MaxBufferedMessages int
}

func NewWebSocketConfig(limits WebSocketLimits, configure ConnConfigureFunc) (WebSocketConfig, error)
```

Zero fields resolve independently: 32 MiB per expanded message; total raw budget
of twice the resolved message maximum; 256 queued/currently-decoding messages.
`WebSocketConfig{}` resolves to those same defaults. Negative inputs reject.
The message maximum must be at most MaxInt64/2; the total raw budget must be at
least twice the message maximum, checked without multiplication overflow. A
positive explicit message-count limit admits at least one buffered message.
`MaxMessageBytes`, `MaxBufferedBytes` and `MaxBufferedMessages` accessors expose
resolved values. No legacy constructor or compatibility mode is retained.

Let M be the message maximum, B the total raw budget and N the message count.
Queued messages plus the raw message currently decoding may occupy at most B-M
bytes and N messages. The sole reader may hold one additional expanded message
of at most M bytes. Admission checks both limits after that read. Thus raw
message payload bytes are at most B (apart from #716's one-byte overflow probe),
and the queue/current decode holds at most N records, plus one in flight.
Empty messages count as one; a bytes-only policy cannot bound their metadata.
Queue storage releases consumed references rather than retaining old capacity
with live payloads. Decoded Go allocations, allocator/compression overhead and
byte slices deliberately retained by `ReadMessage` callers are not a process
heap bound and are explicitly outside these accounting claims.

`ReadJSON` holds the byte/count reservation through the complete JSON decode,
including error paths. `ReadMessage` transfers ownership to its caller and
releases the reservation on return. JSON-RPC server request parsing should use
the runtime's decode-reservation seam rather than prematurely release a raw
message still being decoded. No silent drop-and-continue is allowed: byte or
count exhaustion is terminal, returns a matchable overload error, attempts close
1013 with a bounded deadline, closes the socket and cancels service lifetime.

## Attachment and shutdown APIs

```go
func NewWebSocketStream(config WebSocketConfig, policies ...StreamWritePolicy) *WebSocketStream
func (s *WebSocketStream) SetConn(ctx context.Context, conn *websocket.Conn, cancel context.CancelFunc) error
func (s *WebSocketStream) RunHandler(run func() error, reportCloseError func(error)) error
func (s *WebSocketStream) CloseWithFinalJSON(value any) error
```

Attachment configures before starting the sole physical reader. The supplied
context belongs to the physical connection, not an individual receive. A context
already canceled, canceled during the hook, or canceled during attachment cannot
leave a usable attached connection. The cancellation callback is registered only
when ownership attaches, and stopped by the terminal transition. Same-connection
reattachment is a no-op; hook replacement/nil behavior and terminal late-attach
behavior remain as #716 defines. Hook cancellation after return still works.

HTTP servers keep their derived request/service context and pass that same
context to attachment and the updated request. JSON-RPC servers do the same.
HTTP clients use the dial-call lifetime. Shared JSON-RPC clients use their
connection owner lifetime, not the first acquiring stream's context: existing
per-stream reference counting and cancellation retain that authority.

`RunHandler` calls the endpoint callback once, returns its error unchanged, and
owns deferred completion. Success attempts 1000; error or panic attempts 1011
with a generic reason. It never upgrades an unattached stream. A nonnil cleanup
reporter receives cleanup errors only, preserving existing HTTP observation and
error-handler routing. With no reporter, cleanup errors join the returned
handler error. While unwinding a panic, cleanup completes and the original panic
is rethrown using a defer that also survives a reporter panic. Cleanup must not
replace the service panic or leak service text to the peer.

`CloseWithFinalJSON` is terminal and is used by HTTP client `Close` to preserve
its existing JSON-null final frame. The first caller publishes termination and
owns a best-effort final JSON write with a one-second bound covering both write
lock wait and wire I/O, then unconditionally releases the socket. The deadline
applies even when the normal write policy is unbounded. A deadline-triggered
physical close may interrupt an existing writer so terminal completion cannot
wait indefinitely behind that writer. Ordinary `Close` releases immediately; handler finish uses the same owner with
its appropriate bounded close-control behavior.
Repeated callers observe the recorded completion and cannot replace its cause.

HTTP `CloseAndRecv` remains a nonterminal null input half-close followed by the
final result receive; it must not use the terminal helper. Normal decode errors
remain recoverable. A per-operation context already done remains nondestructive.
An active-operation cancellation retains #716 first-close cause semantics.
A peer read failure releases immediately and retains its original cause; frames
already admitted before a natural peer close remain available in FIFO order to
live-context readers before that cause. Local explicit close/overload is a
visible terminal rejection and does not promise queued work delivery.

## Reproductions and model evidence

The baseline commands both failed for the intended assertions, with no race
report or compilation failure:

```sh
LOOM_DIR="$PWD" go test ./http ./jsonrpc -run 'TestWebSocketStreamTerminalReadReleasesSocket|TestServeWebSocketDisconnectCancelsDispatch' -race -count=1
LOOM_DIR="$PWD" go test ./http/codegen -run '^TestWebSocketLifetimeGeneratedIntegration$' -count=1 -v
```

The first shows normal/abrupt/deadline read socket leaks (#712) and JSON-RPC
normal/abrupt peer disconnect not canceling synchronous dispatch (#722).
Generated real-socket tests show failed terminal-write release (#711), service
success/error return leaks (#714), and idle service disconnect (#722). Logs are
`/tmp/loom-ws-722-baseline-runtime.log` and
`/tmp/loom-ws-722-baseline-generated.log`; durable tests regenerate all inputs.

Model results are recorded in `http/tla/websocket_lifetime/README.md`. The
empty-message bytes-only candidate violates count safety in 15 distinct states;
explicit count admission passes 37. The staged shutdown candidate passes 252
states, including attach/hook/cancel ordering and read-versus-local terminal
causes. The alternative that waits behind an application writer violates
liveness in 180 states. Older omission and FIFO controls still fail as expected.
Temporary logs/states are under `/tmp/loom-ws-722-models`.

The finite shutdown model assumes fair worker progress after the OS socket is
closed and eventual hook return. An application hook that never returns cannot
be killed by Loom; no claim includes arbitrary user callback termination. A
one-second control/write deadline is modeled as eventual timeout, not wall-clock
scheduler proof. Concrete synchronized tests must cover implementation locks,
callback registration, reader count, reservation release and socket closure.

## Acceptance matrix and limits

1. Runtime: terminal write failure, peer close/abrupt/deadline, idle lifetime
   cancellation, canceled/late attachment, repeated Close, concurrent active
   writer, empty-frame and unequal-size byte overload, queued/current-decode
   accounting, FIFO before normal close, decode error then valid frame, and
   stable first terminal cause. Test hooks must set reader state before adoption.
2. HTTP generated: all four witnesses, before-upgrade error, success/error/panic
   after upgrade with expected safe close codes, final null half-close response,
   hook ordering, read limits, selected views and observation. Remove generated
   server-streaming watcher/done/closeOnce state without expanding interfaces.
3. JSON-RPC: synchronous dispatch cancels on disconnect/overload; malformed
   envelopes still emit parse error and permit subsequent valid requests;
   existing logical demux, response ID/order, reference release and timer code
   remain unchanged. No second physical reader is permitted.
4. Representative generation: HTTP StreamingResultDSL, StreamingPayloadDSL,
   BidirectionalStreamingDSL, BidirectionalStreamingResultWithViewsDSL, and
   JSON-RPC ViewedStreamResponseIDDSL. Compare actual parent/candidate bytes with
   identical source inputs, compile/vet both, and compare two isolated candidate
   generations. Update affected checked-in goldens by their normal generator.
   Non-WebSocket fixtures are controls only if their bytes actually change.
5. Reuse focused valid evidence. No exhaustive exported-design sweep is a
   prerequisite. Run applicable package tests and the pinned lint tier before
   freezing. Public docs and GoDoc explain admission, lifetime and migration;
   root owns shared skills, model index, master tracker and ticket closure.

## Implementation refinements

Root approved preserving raw `Close` as immediate physical release after full
JSON-RPC close-during-write controls showed a new graceful write would introduce
an unrelated timeout/error contract. New `CloseWithStatus(code, reason)` replaces
superseded `WriteClose` then `Close`; it validates sendable codes and UTF-8 reason
length, bounds its write and always releases. Handler completion still chooses
1000/1011 in `RunHandler`. Invalid status arguments also close the socket.

`DecodeMessage(ctx, callback)` is the protocol-decoder reservation seam.
JSON-RPC captures parse errors inside that callback, so transport failure and
parse-response behavior remain distinct without releasing raw bytes early.

The physical reader starts immediately, so the active-operation cancellation
test now synchronizes on operation preflight, not on a socket read that might
already be running. It exposed a race: an inbox receive can return ctx.Err before
its AfterFunc runs, and stopping that callback could leave the socket open.
`withContext` now publishes terminal close itself for a failed operation whose
context became canceled after the nondestructive preflight. Already-done calls
remain nondestructive. Natural read errors remain the exact error object when
physical cleanup succeeds, preserving Gorilla's direct close-error classifier.

## Verification ledger

- Full HTTP and JSON-RPC runtime suites under `-race -count=1 -timeout=40s`
  pass (8.260s and 2.063s). The final added JSON-RPC overload-dispatch case passes
  separately under `-race` (1.480s). Runtime coverage includes count release after
  successful/failed decode, ReadMessage ownership transfer and hook preservation.
- Full JSON-RPC generator suite passes (323.433s). The attempted full HTTP
  generator suite exceeded its default ten-minute package timeout while actively
  rendering the unrelated service-local-name corpus. It reported no assertion
  failure before timeout. This is not recorded as a full HTTP package pass.
- The complete affected HTTP selection covers WebSocket-named tests, server and
  client streaming sections, server handlers, converted stream render sections,
  mixed client endpoint sections and stream admission. The final selection passes
  (23.856s). Its first run identified
  a stale server-side #716 expectation: overflow now cancels the service context,
  so RecvWithContext(serviceCtx) returns context.Canceled. The test still checks
  wire close 1009, while live-context runtime and generated-client paths retain
  ErrReadLimit. Only that expected service-context result changed.
- Representative generation covers all five designs listed above: 114 generated
  files per revision, all parent/candidate examples build and vet, and two
  independent candidate processes emit identical bytes. Eighteen files differ:
  four HTTP adapters per design (client.go/client websocket.go/server.go/server
  websocket.go) and two JSON-RPC files (client.go/server websocket.go).
  Intended differences are context-aware attachment, removing the generated
  reader-lifetime watcher/done state, shared handler cleanup, and terminal-close
  delegation. Service types, views, schemas, example files and other artifacts
  are byte-identical. `CloseAndRecv` remains a nonterminal input half-close.
- Normal golden regeneration updates 24 HTTP WebSocket goldens and one JSON-RPC
  server golden. The generator-rendered section capture updates 87 HTTP section
  expectations. No generated `gen/` files were hand-edited or added.
- Pinned golangci-lint across HTTP/JSON-RPC runtime and generator packages passes
  with zero issues. `git diff --check` passes; new source responsibilities are
  split across connection attachment, reader/inbox, shutdown and configuration.
- Current terminal-cause model passes 17 states. Historical #716 revised control
  still passes 16; old-before/after controls still fail the intended invariant
  at 11/7 states. The refined shutdown and empty-message model results above
  remain unchanged.

Temporary evidence is `/tmp/loom-ws-722-*`; representative manifests and exact
intended diff are under `/tmp/loom-ws-722-comparison`. The comparison driver is
`/tmp/loom_ws_722_compare.py`. Durable tests, models, configuration, acceptance
contracts and this ledger remain after scratch cleanup. No older #716 scratch
or `/tmp/loom-ws-readiness-generated` directories are dependencies.

Root-owned guidance deltas: update the consumer skill to the WebSocketLimits
constructor and context attachment, configurable bounded inbox/1013 overload,
automatic handler cleanup and unchanged CloseAndRecv half-close; update the
framework skill with sole physical-reader ownership, decode-held reservations,
shared terminal publication and bounded release; add the lifecycle model to the
formal model index beside message admission and JSON-RPC demux ownership.

## Integrated #722 validation

The reviewed 148-path candidate was integrated onto pushed main parent
`433d2e741b43b35783d293d0de60cabeca0be6a2`. All 145 implementation paths retain
the frozen reviewed hashes; root merged only the three shared guidance files.

Using Go 1.27.2, five representative designs produced 114 files per generation:
HTTP StreamingResultDSL, StreamingPayloadDSL, BidirectionalStreamingDSL,
BidirectionalStreamingResultWithViewsDSL, and JSON-RPC ViewedStreamResponseIDDSL.
Every integrated output byte matches the reviewed candidate and a second fresh,
independent-process generation. There are no new generated differences; the
reviewed build/vet evidence remains valid and was reused. The integrated CLI was
built once. No old-parent regeneration, broad package rerun, or corpus audit was
needed.

All new temporary modules require the verified pushed version
`github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261012181439-651372ce6ff0`, with an
explicit local replace and `LOOM_DIR` pointing to the integrated main checkout.
`go mod download -json` verifies origin commit
`651372ce6ff05ed1db495369fd51d48df0c0ec75`. This corrects the earlier comparison
harness's placeholder module version without changing production source or the
reviewed output bytes.

Focused integration checks pass:

- `LOOM_DIR="$PWD" go test ./http ./jsonrpc -run 'WebSocket|ServeWebSocket|ReceiveWebSocket' -race -count=1 -timeout=90s`
  — HTTP 4.077s; JSON-RPC 2.204s.
- `LOOM_DIR="$PWD" go test ./http/codegen -run '^TestWebSocket(Lifetime|MessageLimit)GeneratedIntegration$' -count=1 -timeout=3m`
  — 6.478s; the generated applications execute their real-socket regressions
  under the race detector.

Temporary evidence: `/tmp/loom-ws-722-integrated-comparison/results.json`, empty
`reviewed-vs-integrated.diff`, per-generation logs, runtime/generated logs, and
`/tmp/loom-ws-722-integration-pin.json`. Driver:
`/tmp/loom_ws_722_integrated_compare.py`. No production or repository documentation
was edited by this integration-validation step. Root owns the final review,
commit, push, and ticket reconciliation.
