# WebSocket message admission

This bounded #716 model compares the previous unrestricted runtime, a framing-
only limit, and the shared framing plus expanded-byte policy. The abstract limit
is two bytes; wire sizes range from one to three and expanded sizes from the
wire size through four. Compression overhead, allocator overhead and scheduling
are outside this admission model. It establishes no process heap bound.

Run with TLA+ tools release 1.7.4 (TLC 2.19), jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
cd http/tla/websocket_message_limit
java -XX:+UseParallelGC -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar \
  tlc2.TLC -workers 1 -deadlock -metadir /tmp/loom-message-model \
  -config cfg/message_complete.cfg WebSocketMessageLimit.tla
```

Use a distinct temporary metadir for each run. `message_old.cfg` and
`message_framing.cfg` are expected failing controls: ExpandedBound fails in
12 distinct states. `message_complete.cfg` passes ExpandedBound,
OverflowReleased and ValidAccepted in 18 distinct states. These runs used Java
25.0.4.1 on 2026-10-12.

`Read` corresponds to `http.WebSocketStream.ReadMessage`: Gorilla's framing
limit rejects wire overflow; a max-plus-one expanded read detects compressed
expansion, sends a bounded 1009 close control and releases the socket.
`TestWebSocketMessageLimit` tests exact, exceeded, compressed, default, framing
hook and repeated-terminal-read controls on real sockets. Before implementation
the default and compressed-expansion controls accepted oversized messages,
and the framing-only rejection left the local socket open.

`TestWebSocketConfigValidation` and `TestWebSocketSetConn*` cover configuration
and attachment outside this abstract model. Lifetime and inbox ownership are
separate pending #722 work, outside the #716 correction.

## Terminal read cause

`WebSocketTerminalCause.tla` models the close/failed-read ordering exposed by
independent review of #716. The previous candidate recorded a network error
after context cancellation or explicit Close had unblocked the read. Subsequent
reads then returned that low-level error instead of the documented closed
sentinel. Explicit Close after a previously recorded, nonclosing read failure
had the same incorrect precedence.

Run the command above with `WebSocketTerminalCause.tla` and one of
`cfg/terminal_old_before.cfg`, `cfg/terminal_old_after.cfg`, or
`cfg/terminal_revised.cfg`, using distinct temporary metadirs. The two old
controls violate `LocalCloseOwnsCause` in 11 and 7 distinct states respectively.
The revised model passes all three invariants in 16 distinct states (20 states
generated). These runs used the same pinned TLC and Java versions on 2026-10-12.

The first physical-close transition owns the terminal read cause. An explicit
or context Close clears a prior nonclosing read failure and selects the closed
sentinel. Read-driven overflow cleanup selects the limit failure. Later closes
and read errors caused by closing cannot replace either cause. Cancellation may
still override the result of the active operation with its context error; it
does not override an already selected connection-terminal cause.

`LocalClose` and `OverflowCleanup` correspond to the shared
`WebSocketStream.close(readCause)` transition under `connLock`.
`RecordFailure` corresponds to `recordReadErr`, which records only while open.
The model explores local close before and after failure detection, before and
after overflow cleanup, and before and after read-error recording. The bounded
1009 control write precedes overflow cleanup; a concurrent local close can win
during that write. The model abstracts that interval as scheduling before
`OverflowCleanup`. Physical Close and error publication are atomic under the
lock; syscall failures, attachment and payload decoding remain outside this
model. The invariants concern subsequent reads, not the low-level error returned
to an operation already in progress, and make no liveness claim.

`TestWebSocketStreamLocalCloseOwnsTerminalCause` synchronizes on the underlying
socket read before canceling or closing, and separately closes after a completed
deadline read failure. All three cases failed before the correction.
`TestWebSocketStreamAlreadyCanceledReadPreservesConnection` checks that a context
already done neither reads nor closes the socket; a subsequent valid message
still succeeds. `TestWebSocketStreamOverflowOwnsTerminalCause` checks that a
later explicit Close preserves the limit cause. These tests check the modeled
ordering against real Gorilla sockets; race-detector runs check shared state.
