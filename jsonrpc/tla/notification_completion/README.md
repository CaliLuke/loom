# JSON-RPC WebSocket request completion (#682)

This finite model distinguishes failure of one decoded request from failure of
its connection. It informed the shared typed-dispatch completion design before
implementation. The historical streaming decode branch returned its business error
for an ID-less frame, causing the receive loop to terminate. The owned implementation
completes that request without a reply and allows the following valid request.

Run with the pinned TLA+ 1.7.4 jar (TLC 2.19), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`,
and Corretto 25:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-notification-owned \
  -config owned.cfg NotificationCompletion.tla
```

| Configuration | Expected result | Generated / distinct |
| --- | --- | --- |
| current | RecoverableConnection counterexample, exit 12 | 131 / 131 |
| blanket | TransportFailureEscapes counterexample, exit 12 | 65 / 65 |
| owned | All invariants pass, exit 0 | 240 / 180 |

An absent ID is a notification; string and explicit null IDs both request a
response. Params and service failures are request-local unless an attempted
error-response write fails. Read/context/write failures must remain terminal.
The follow-up is a valid request acting as a protocol barrier. The model rejects
both spurious termination and suppression of a genuine transport failure.

`writeWebSocketRequestCase` in
`../../codegen/stream_sections_websocket_request.go` is the typed dispatch seam.
Its generated unary/streaming decode and endpoint error branches must consume
one request-failure completion renderer. That renderer suppresses a notification's
error response, or sends the typed request error and returns only a send failure.
`ReceiveWebSocketRequest` in `../../stream_server.go` keeps connection errors
unchanged. An error value alone carries insufficient origin information to put
blanket suppression at that runtime boundary.

`TestJSONRPCNotificationCompletionGenerated` generates and mounts unary,
server-stream and bidirectional methods; it sends a bad first message then a
valid barrier request. It checks correlation, response code/detail, invocation
counts, and the existing invalid-params observer reason. Required-ID notification
admission exercises #790 without confusing service data with envelope HasID.
`TestNotificationCompletionPreservesTransportFailures` injects dispatch and
protocol-error sender failures. Existing receive tests cover real close codes,
timeouts, cancellation, malformed frames, and repeated reads after termination.

The model explores one failed frame and one following valid request, three ID
presence classes, two adapter families and independent error-write failure.
It does not model goroutines, networking, payload codecs, nullable service data,
batch envelopes, or callback panics. The boolean terminal outcome means a failure
escapes to the receive loop; it does not prove socket cleanup latency. Tests check
actual generated framing, observations and transport behavior. No liveness or
unbounded fairness claim is made. Keep raw model output outside the repository.
