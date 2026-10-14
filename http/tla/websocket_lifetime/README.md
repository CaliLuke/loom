# WebSocket lifetime and bounded inbox exploration

These models guided #722, with #711, #712 and #714 as witnesses of the same
missing lifecycle owner. Initial old-behavior evidence used source
`7d73607827eb2896e185b1438095b868a9d88df8`; the concrete red regressions were
reproduced on the actual implementation parent `46b627ef` after #716. The models
are deliberately staged: they reproduce old omissions, compare admission
policies, then refine shutdown ordering. Tests check the Go correspondence.

## Reproduce

Pinned tool: `tla2tools-1.7.4.jar`, reporting TLC 2.19. The runs below used Java
25.0.4.1 on macOS arm64 (the installed JVM reports x86_64).

```sh
cd http/tla/websocket_lifetime
java -XX:+UseParallelGC \
  -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -deadlock -metadir /tmp/loom-websocket-model \
  -config cfg/traffic_overload.cfg WebSocketLifetime.tla
```

Choose each configuration below separately. Use a distinct temporary `-metadir`
per live run. `-deadlock` disables deadlock checking because completed finite
scenarios may have no next action. Expected failing configurations are controls;
a parse/evaluation exception does not count as a reproduced failure.

| Configuration | Observed result | Distinct states |
| --- | --- | ---: |
| `close_old` | `TerminalReleases` fails after close-control write failure; caller's terminal operation returned but local socket is open. | 2 |
| `read_old` | `TerminalReleases` fails after terminal read failure; socket and watcher remain live. | 2 |
| `return_old` | `TerminalReleases` fails when service completes after upgrade without owning cleanup. | 2 |
| `disconnect_old` | `DisconnectCancels` fails: peer gone, dispatch running, no reader, infinite stutter. | 2 |
| `close_overload`, `read_overload`, `return_overload` | Atomic owner transition satisfies release and watcher/cancel safety. | 2 each |
| `disconnect_overload` | `DisconnectCancels` holds under fair reader progress. | 3 |
| `traffic_fifo` | Safety invariants hold; `TrafficEventuallyStops` fails with stalled consumer and pending frame unable to enter inbox. | 24 |
| `traffic_overload` | Safety and `TrafficEventuallyStops` hold without assuming consumer progress. | 29 |
| `half` | Input half-close retains socket and output capability. | 2 |
| `decode` | Payload decode failure does not terminate connection. | 2 |
| `late` | Attach after local terminal close does not revive socket. | 3 |

Results were obtained on 2026-10-12. The first traffic run exposed a model
representation error (comparing string close marker to numeric frame IDs); it
was corrected by a numeric close marker and both traffic configurations rerun.
No result above relies on those failed evaluation runs.

## Queue policy finding

The bounded wire sequence is three complete frames of sizes 1, 2 and 1 followed
by a peer close. Message cap and waiting/decode budget are each 2 abstract bytes.
At most one additional message can be in flight. `dispatch = TRUE` represents a
service that is executing synchronously and not consuming more input. Consumer
progress is possible but not assumed fair: waiting forever on `ctx.Done()` is a
real service shape behind #722.

The shortest FIFO counterexample is:

1. read frame 1;
2. queue frame 1 (one byte);
3. read frame 2 (two bytes);
4. it cannot join the two-byte inbox, so the only reader waits forever;
5. peer close remains behind frame 3; service context is not canceled.

The overloaded candidate visibly terminates at step 4. It does not discard the
frame and then continue receiving. This is a public admission-policy change:
valid frames may be rejected when a consumer does not keep up. Once terminal,
delivery of buffered frames is not promised by this policy model. The user accepted close-on-buffer-overload on 2026-10-12. The selected policy
terminates the connection with bounded close-control write (1013) and cancels
the service; individual-message overflow remains 1009. The model informed the
choice but did not authorize it.

`BufferedBytes` includes the queued frames and the raw frame being decoded.
`FlightBytes` includes the reader's one additional message. Moving a frame to
decode preserves its reservation; finishing decode releases it. The invariant
bounds their sum by `BufferMax + MessageMax`. It assumes #716 or another
accepted message-admission owner bounds **expanded** message bytes before
accumulation. Gorilla's compressed-frame read limit alone does not establish
that assumption.

Decoded Go object overhead and objects/byte slices retained by application code
are outside this raw-byte bound. The model does not prove a process heap bound.
Wire and inbox messages carry unique IDs so `FIFO` and `NoSilentLoss` check
identity/order, not merely lengths. Visible terminal rejection is explicitly
outside the pre-termination delivery guarantee.

## Correspondence and concrete regression evidence

| Model action | Current source seam | Regression |
| --- | --- | --- |
| `Fault`, close/read/return | Shared `stop` and `RunHandler` replace missing generated cleanup. | `TestWebSocketStreamTerminalReadReleasesSocket`; generated close-after-peer-close and success/error/panic return tests |
| `DetectDisconnect` | Sole `readPump` continues while typed HTTP services and JSON-RPC dispatch execute. | Generated `TestGeneratedDisconnectCancelsService`; `TestServeWebSocketDisconnectCancelsDispatch` |
| `ReadFrame`, `QueueFrame` | `readPump`, admission under `connLock`, expanded-message owner retained from #716. | `TestWebSocketInboxOverload`: unequal byte sizes and empty-message count |
| `StartDecode`, `FinishDecode` | `DecodeMessage` holds byte/count reservation through callback; `ReadJSON` and JSON-RPC parser use it. | `TestWebSocketDecodeRetainsReservation`, `TestWebSocketInboxReleasesConsumedReservation` |
| `FIFO`, `NoSilentLoss` | `nextMessage` preserves admitted queue for natural failure and live operation contexts; explicit rejection discards queue. | `TestWebSocketInboxNaturalClosePreservesFIFO` |
| `HalfClose` | Generated HTTP `CloseAndRecv` sends null without terminal shutdown, then receives final result. | Generated `TestGeneratedHalfCloseFinalReply` |
| `DecodeFailure` | Decode errors release reservation and remain nonterminal. | `TestWebSocketStreamReadJSONDecodeErrorIsNotTerminal`, `TestReceiveWebSocketRequestClassifiesReadFailures` |
| `LocalCancel`, `LateAttach` | Context-aware `SetConn`, callback registration after hook, single terminal owner. | Canceled-before/during-hook, reentrant hook close and late-attachment controls |
| Shutdown publication/control/release | `stop`, `releaseConnection`, one-second control/final-JSON deadline including writer-lock wait. | `TestWebSocketFinalJSONCloseBoundsWriterWait`, handler close-error/reporter/panic tests |

The direct normal/abrupt/deadline and generated regression tests failed on
parent `46b627ef` with intended lifecycle assertion failures, no compilation
failure and no race report. The repaired runtime passes the full HTTP/JSON-RPC
runtime packages under `-race`. The generated HTTP harness compiles and runs
real-socket tests under `-race`; it also checks no forced upgrade on pre-upgrade
error and safe close reasons on handler error/panic. Reproduce with:

```sh
LOOM_DIR="$PWD" go test ./http ./jsonrpc -race -count=1
LOOM_DIR="$PWD" go test ./http/codegen -run '^TestWebSocketLifetimeGeneratedIntegration$' -count=1
```

Tests regenerate their own inputs; historical scratch directories are not
reproduction dependencies. The physical reader
is singular by construction in this model; source/generator assertions and
race-detector tests check the implementation. The existing JSON-RPC logical
client demux remains the only ID-routing reader over the shared inbox. Existing
`WebSocketClientDemux`, `ClientClose`, `CloseErrors`, and `WebSocketPayloadNull`
models remain separate controls.

Keep model sources and summarized results; remove task-owned checker state,
trace and generated-module scratch after validation and independent review no
longer need them. Keep the shared installed toolchain.

## Current-parent refinement checkpoint

The old witness and policy configurations were rerun against the contracts of
parent `46b627ef` after #716. Runtime and generated witnesses still fail for the
same intended lifecycle assertions. The copied tests use that current runtime;
no old production source was copied. Logs are temporarily retained under
`/tmp/loom-ws-722-models`, with direct/generated witness logs beside that folder.

`MessageCountMax` now bounds queued plus currently decoding messages separately
from raw bytes. The empty wire sequence uses three zero-byte messages then peer
close. The bytes-only candidate violates `BoundedMessages` after 15 distinct
states (24 generated). The count-bounded overload candidate passes 37 distinct
states (53 generated) and terminates without consumer-progress assumptions.
`cfg/empty_bytes_only.cfg` is the expected failing control;
`cfg/empty_overload.cfg` is the implemented policy. This addresses record allocation
for empty messages, which byte accounting alone cannot constrain.

`WebSocketShutdown.tla` separates terminal publication, bounded control attempt,
physical release and reader/writer exit. Attachment may start before or after
close and return from its hook after cancellation. Terminal cause and cancellation
are selected once. `cfg/shutdown.cfg` passes 252 distinct states (563 generated).
`cfg/shutdown_writer_lock.cfg` models waiting behind an application writer before
the terminal attempt; its liveness check fails after 180 states (338 generated).
The corrected design must not require that writer's voluntary progress to close
the socket. The final JSON close path therefore requires a deadline-triggered
physical close while it is waiting for the application write lock, not merely a
write deadline installed after acquiring that lock.

Use the same TLC command with `WebSocketShutdown.tla` for the two shutdown
configurations. The original configuration results remain unchanged: traffic
FIFO fails at 24 states, overload passes 29; old close/read/return fail at two
states and old disconnect fails liveness at two states. Half-close, decode-error,
late-attachment and corrected omission controls still pass.

These finite results informed the implementation; they do not prove its Go
implementation or arbitrary network behavior. Shutdown assumes
fair scheduling of workers after socket close and eventual return of the
application configuration hook. It models bounded terminal I/O as timeout
availability; tests must establish the actual deadline and locking protocol.
The inherited terminal-cause model under `../websocket_message_limit` remains a
separate control and must continue to match the implementation.

## Final input and disconnect composition (#795)

`WebSocketInputFinish.tla` fills a correspondence gap exposed by generated
nested-collection and union tests after #722: `CloseWithFinalJSON(nil)` wrote the
null marker and immediately released TCP. The receiving pump could admit data
and null, observe EOF, and cancel the service before its reads consumed either.
The older FIFO invariant covered only live operation contexts; modeling input
half-close separately did not exercise this composition.

Run this module with `cfg/input_finish_<name>.cfg` using the command above.
The old `immediate` configuration fails `OrderlyInputPreserved` after 17 generated
states (12 distinct): data/null are queued, local release occurs, and disconnect
cancels before consumption. The corrected `bounded` configuration passes 23/17
states; `final` (both sides final-close) passes 9/7; `abrupt` passes 37/24. All
corrected configurations check FIFO and eventual close completion.

The candidate waits for peer termination after the final write, sharing the
existing one-second budget across writer acquisition, write and wait. Deadline
expiration permits abort; the invariant promises ordered consumption only for a
cooperating handler when the deadline did not expire. The model abstracts time
as a fair timeout action; it does not guarantee scheduling or completion within
one second. A peer termination is not a service-processing acknowledgement.
Simultaneous final closers do not acknowledge each other's JSON null: timeout
still guarantees release. Abrupt peer termination still cancels the service.

`releaseConnection` implements the wait on the existing sole reader's `pumpDone`.
`readPump` discards bounded frames during local terminal cleanup and continues
until a physical read/protocol-limit failure, so an ordinary late message cannot
falsely complete the wait. No additional inbox admission occurs after shutdown.
The existing aggregate byte/count model continues to own admission bounds;
this small ordering model does not model allocations, writer errors or attach
races. `WebSocketShutdown` still checks first-owner publication, forced release,
writer-lock timeout and attachment cancellation separately.

`TestWebSocketFinalJSONPreservesQueuedInput` gates service decoding and the final
write, then uses a following ping to prove null admission before releasing the
service. `TestWebSocketFinalJSONCloseWaitsForPeer` sends late data and a ping to
prove the reader does not exit on local closed state. Stalled-peer, simultaneous
close and blocked-writer tests check bounded physical release. The existing
generated nested-collection/union tests remain unchanged; generated lifecycle
coverage retains the nonterminal CloseAndRecv final reply and real disconnect
cancellation. Raw Close remains immediate when it owns the terminal transition.

## Native peer acknowledgement before terminal publication (#871)

`PeerCloseCleanup.tla` separates a native close acknowledgement from the reader's
later terminal publication. The application can call Close in between. The
current shutdown owner reports `ErrCloseSent` as a new cleanup failure even
though native close framing has already completed. Both final JSON and close
control paths share this classification boundary.

Run the same pinned TLC jar with `-config peer-owned.cfg PeerCloseCleanup.tla`
and an external `-metadir`. Replace `owned` with each negative configuration:

| Configuration | Result |
| --- | --- |
| current | NoSpuriousCloseError fails, 13 generated/distinct states |
| suppress-all-close-errors | KeepWriteError fails, 8 generated/distinct states |
| suppress-all-writes | DataWritesFail fails, 15 generated/14 distinct states |
| owned | All five invariants pass, 44 generated/24 distinct states |

These configurations keep deadlock checking enabled and stutter after terminal
release. The model abstracts acknowledgement as completed native framing, and
physical/write errors as independent booleans. It does not establish networking,
callback progress, timeout precision or error-wrapping behavior. The older
shutdown and input-finish models continue to own those lifetime obligations.

`TestWebSocketCloseAfterPeerAcknowledgement` gates the configured close handler
after its acknowledgement but before reader publication, reproducing both
final-JSON and close-control failures. It keeps ordinary late-data writes failing
and checks that a real physical-close error remains observable. The
implementation seam is `releaseConnection`'s shared cleanup result classification;
never suppress this state in ordinary WriteJSON or individual generated callers.

Only the raw native `ErrCloseSent` sentinel is classified as completed framing,
after the final-write wait branch; it never starts a new peer wait. The
`TestWebSocketFinalJSONPreservesWrappedEncodingError` control preserves a custom
marshaler error wrapping that sentinel plus an independent physical-close error.
This error-wrapping behavior is outside the finite model above.
