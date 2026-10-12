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

`Read` corresponds to `http.WebSocketStream.readPump`: Gorilla's framing
limit rejects wire overflow; a max-plus-one expanded read detects compressed
expansion, sends a bounded 1009 close control and releases the socket.
`TestWebSocketMessageLimit` tests exact, exceeded, compressed, default, framing
hook and repeated-terminal-read controls on real sockets. Before implementation
the default and compressed-expansion controls accepted oversized messages,
and the framing-only rejection left the local socket open.

`TestWebSocketConfigValidation` and `TestWebSocketSetConn*` cover configuration
and attachment outside this abstract model. Lifetime and inbox ownership are modeled separately in
[`../websocket_lifetime`](../websocket_lifetime/README.md).

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

The #716 revised model records first physical-close ownership. Local close
selects the closed sentinel; overflow selects the limit failure. In that earlier
runtime a non-limit network failure was recorded without closing, so a later
local close superseded it. `terminal_revised.cfg` preserves that historical
control; it is not the complete current lifecycle contract.

`cfg/terminal_lifecycle.cfg` adds automatic network-failure closure. It passes
`LocalCloseOwnsCause`, `OverflowOwnsCause`, `NetworkOwnsCause` and
`OwnerIsTerminal` in 17 distinct states (20 generated). Network failure,
expanded overflow and local close now all compete for the same first terminal
publication. A later transition cannot replace that cause. Cancellation may
still override the active operation's result with its context error; it cannot
replace an already selected connection cause. The original network error object
is preserved when physical cleanup succeeds, retaining Gorilla's close-error
classification behavior.

`LocalClose`, `OverflowCleanup` and lifecycle `RecordFailure` correspond to
`WebSocketStream.stop` publishing `closed` and `readErr` under `connLock`.
Publication precedes bounded terminal I/O and physical release. The separate
shutdown model explores that non-atomic interval and attachment races; this
small cause model abstracts it and makes no liveness claim. Syscall failures,
payload decoding and operation-context precedence are outside its state space.

`TestWebSocketStreamLocalCloseOwnsTerminalCause` synchronizes on operation
preflight before cancellation or explicit close. Synchronizing on the socket
read is insufficient now that the physical pump starts at attachment.
`TestWebSocketStreamAlreadyCanceledReadPreservesConnection` checks that an
already-done operation neither reads nor closes; a later valid message succeeds.
`TestWebSocketStreamOverflowOwnsTerminalCause` preserves overflow through later
close. `TestWebSocketInboxNaturalClosePreservesFIFO` checks admitted delivery
followed by the exact stable natural cause, including after explicit close.
`TestWebSocketStreamTerminalReadReleasesSocket` covers normal, abrupt and deadline
read failures. Real-socket race-detector tests establish implementation evidence
beyond the model's finite assumptions.
