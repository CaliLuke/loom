# WebSocket final input and physical-close ordering (#795)

Status: implemented and validated; frozen for independent review.
Parent `601a4338bd7601b8b5a6dec33a6e1fa0eea20b3e`. Runtime repair below; no generator changes.

CI at `56bc7eae` failed nested collection and union WebSocket round trips with
server `context canceled`. Current-main union reproduction fails in all three
runs, while the nested-collection test passes those runs (same ordering race):

```sh
LOOM_DIR="$PWD" go test ./http/codegen -run '^(TestNestedCollectionPayloadGeneratedIntegration|TestUnionRequestBodyGeneratedIntegration)$' -race -count=3 -timeout=3m
```

Evidence: `/tmp/loom-ws-close-order-baseline.log`. This is not a stale expected
error: generated client Close should let the server consume input and its JSON
null end marker in the ordinary successful-close exchange.

The shared client's `CloseWithFinalJSON(nil)` writes null then physically closes.
The server pump may queue data/null, detect physical EOF and cancel the service
context before the service consumes that queue. RecvWithContext correctly honors
cancellation; orderly application EOF is lost, and earlier inputs may be lost.
The original lifecycle model's FIFO property applied only while contexts were
live, and its half-close case did not compose null with immediate physical close.
This missing composition is the model-correspondence gap.

Candidate to model: a bounded final-JSON completion phase, using the same sole
reader, keeps the connection alive after the null write until peer termination
or the existing one-second terminal deadline. A cooperating server drains data,
observes null, completes its handler and closes; the client then releases. A
stalled server remains bounded by deadline/physical release. Immediate raw Close,
real abrupt-disconnect cancellation and CloseAndRecv's nonterminal half-close
remain separate accepted contracts. No second reader, arbitrary sleep, unbounded
queue, cancellation suppression or test-error relaxation is proposed.

Before implementation, specify behavior for pending inbound frames during this
terminal phase, natural/abnormal peer termination, timeout after a successful
final write, first-close ownership, and concurrent writer/context cancellation.
The shared owner must observe peer termination without confusing reader exit due
to its own terminal state with peer acknowledgement. The final deadline must
still cover lock acquisition, wire write and the completion wait together.

Sequence:

- [x] Preserve #792 readiness and switch to an isolated current-main worktree.
- [x] Reproduce generated failures and identify null/physical-close/context order.
- [x] Model old sequence and candidate with explicit liveness assumptions.
- [x] Root reviews diagnosis and concrete candidate contract.
- [x] Add synchronized direct/generated regression controls, implement shared
  owner correction, and rerun affected lifecycle/half-close/overload checks.
- [ ] Freeze exact source for independent review; root integrates and delivers.

## Concrete candidate for root review

Working tree: `/Users/luca/.codex/worktrees/websocket-close-order/loom`.
Model: `http/tla/websocket_lifetime/WebSocketInputFinish.tla`; configurations
`cfg/input_finish_{immediate,bounded,final,abrupt}.cfg` beside it.

Keep the existing public API. `CloseWithFinalJSON` publishes terminal ownership,
rejects further application operations, discards its inbound application queue,
and stops its lifetime watcher exactly as today. It acquires the writer and sends
the final value under the existing one-second deadline. After a successful write,
it waits for the sole reader pump to terminate or that same deadline to expire,
then unconditionally releases the socket. Failed writes skip the wait. Raw
`Close` still releases immediately when it wins terminal ownership; repeated
terminal operations share the first owner's result, as before.

The pump must continue reading while local terminal cleanup owns the connection,
discarding each bounded incoming frame instead of enqueuing it. This both keeps
memory bounded and prevents a late ordinary data frame from being mistaken for
peer termination. `pumpDone` then signifies a physical read failure/close or a
protocol-limit failure, never an early exit merely because local close started.
No extra goroutine, reader, queue or public state/API is needed. A pump limit
failure is a terminal reason to stop waiting; it does not become a successful
application acknowledgement.

The peer observation is an underlying WebSocket/TCP termination, **not a JSON
null acknowledgement or a guarantee of service processing**. A normal generated
server consumes data/null, returns from the handler and sends its bounded close
control. An abrupt peer release also ends the wait. If both peers invoke final
JSON close, neither null is a transport acknowledgement: their shared one-second
budgets expire and both release. This remains bounded without inventing a second
application protocol. CloseAndRecv continues its nonterminal null half-close so
it can receive a final service reply.

Error contract remains cleanup-only: final write/lock failures and physical
close failures are returned; peer read causes and expiration after a successful
write do not add new Close errors. This preserves existing Close semantics while
making the completion interval explicitly best effort. The timeout callback
still force-closes a blocked write; one sync.Once owns physical release. Active
operation cancellation and first terminal cause keep their existing precedence.
The candidate does not change the server's genuine-disconnect cancellation or
allow reads with canceled contexts to consume a queue.

Acceptance: a synchronized real-socket test holds the receiving service until
both data/null are queued and proves its context stays live until consumption;
then server normal close promptly releases the client. Controls cover incoming
frames during final wait, stalled peer, simultaneous final close, abrupt close,
blocked writer, first-close sharing, immediate raw close and nonterminal final
reply. Existing two failing generated tests must pass unchanged under race.
Runtime-only change leaves generated bytes unchanged; bounded existing generated
lifecycle, JSON-RPC demux and half-close checks verify shared consumers.

## Acceptance evidence

Source parent: `601a4338bd7601b8b5a6dec33a6e1fa0eea20b3e`; Go 1.27.2
on darwin/arm64. Scope is two shared runtime files, one direct regression file,
public guidance, this readiness record and the owning TLA module/configuration
records. No generated source changes, dependencies or new API. Existing generated
harnesses run with explicit `LOOM_DIR` pointing at this worktree.

- Current parent generated baseline: union case failed all three race runs with
  `server recv: context canceled`; nested collection passed locally but failed
  the same ordering contract in CI. The new direct peer-wait test also failed
  before implementation (normal close returned before peer termination; abrupt
  case observed a broken pipe while the still-active peer sent its ping).
- `LOOM_DIR="$PWD" go test ./http ./jsonrpc -race -count=1`: pass,
  11.128s / 2.162s. Covers immediate raw close, terminal cause, overload, attachment,
  active cancellation, writer timeout, handler cleanup and JSON-RPC ownership.
- `go test ./http -run '^TestWebSocket(FinalJSON|SimultaneousFinal)' -race -count=5`:
  pass, 16.482s, including the subsequently added synchronized queued-input test.
  The ping barrier proves queue admission while the service decode remains held;
  no sleep or relaxed cancellation assertion is used.
- `LOOM_DIR="$PWD" go test ./http/codegen -run '^(TestNestedCollectionPayloadGeneratedIntegration|TestUnionRequestBodyGeneratedIntegration|TestWebSocketLifetimeGeneratedIntegration)$' -race -count=3 -timeout=4m`:
  pass, 38.691s. Existing generated tests remain unchanged and their emitted
  applications are compiled and run, including final-reply half-close and actual
  idle disconnect cancellation.
- `LOOM_DIR="$PWD" go test ./jsonrpc/codegen -run '^TestJSONRPCWebSocket(StreamsShareConnection|RecvAfterResponse|StreamingPayload)GeneratedModule$' -count=1 -timeout=2m`:
  pass, 10.330s; shared demultiplexer, pending response ordering and payload paths.
- `golangci-lint run ./http ./jsonrpc`: zero issues. `go run ./scripts/docscheck`
  and `git diff --check`: pass. No generator input/output decision changed;
  another generated-byte comparison would duplicate unchanged generator evidence.
- New input-order model: old failure 17 generated / 12 distinct; bounded handler
  23/17, simultaneous final closers 9/7, abrupt peer 37/24 pass. Existing affected
  models rerun: close/read/handler legacy controls still fail terminal release;
  old disconnect still fails eventual cancellation; corrected close/read/handler,
  disconnect, half-close, late attach, traffic/count overload pass. Shutdown
  corrected passes 563/252; unbounded writer-lock control still fails 338/180.
  Tool: pinned tla2tools-1.7.4.jar (TLC 2.19), Java 25.0.4.1. Results establish
  the documented abstractions, not wall-clock scheduling or guaranteed delivery.

Transient logs use `/tmp/loom-ws-close-order-*`; the model inputs and all commands
needed to reproduce are durable here and in the owning model README. Root owns
integration, independent final review, the single #795 commit, push and closure.
