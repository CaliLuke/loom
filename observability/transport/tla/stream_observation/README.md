# Stream observation ownership — #749

This bounded design model separates actual runtime transitions from observer
callback delivery. `current.cfg` reproduces the missing SSE opening event from
`http.SSEStreamWriter.initHeaders`. The parent-source Go probes also
reproduce missing close, bare failure/suppression identity and a callback invoked
under the SSE writer mutex. The other negative configurations explore rejected
repairs; they do not claim those hypothetical designs exist in production.

| Configuration | Result | Distinct states |
| --- | --- | ---: |
| current | LifecycleRecorded fails | 5 |
| identity-shared | IdentityStable fails after A binding and B dispatch | 16 |
| split | OpenBeforeClose fails after independent close publication | 11 |
| locked | CallbackOutsideLock fails | 12 |
| general | BoundedPending fails when recoverable failures join a blocked queue | 36 |
| owned | All six safety invariants pass | 528 |

Tools: TLA+ tools 1.7.4 (TLC 2.19), Java 25. Run each configuration separately:

```sh
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-stream-observation-owned \
  -config owned.cfg StreamObservation.tla
```

Negative configurations must exit 12 with their named invariant, not a parser or
semantic error. Owned exits 0. Raw checker outputs are temporary, not sources.

## Correspondence and limitations

- `Commit` is successful SSE header commitment or server WebSocket Upgrade.
  Opening does not require a successful flush/configuration callback. The runtime
  owner already decides this; observation does not alter its decision.
- `Release` corresponds to SSE's existing terminal transition or WebSocket native
  connection release, not merely receipt of a close frame or logical suppression.
  Reuse `http/tla/websocket_lifetime/WebSocketShutdown.tla` and its existing
  `shutdown.cfg` evidence for native release, cancellation and attachment races.
  This model does not re-prove I/O ownership or network completion.
- `Drain` claims one of at most two lifecycle records after releasing the runtime
  mutex. Only one drainer is active; Close can reenter while its callback is active.
  `CallbackReturns` and `CallbackPanics` both release that claim. The abstraction
  counts an entered callback as delivered, and tests must establish defer cleanup
  and the original Go panic propagation/precedence. No panic swallowing is allowed.
- `Failure` repeats up to three times, enough to expose a generic queue exceeding
  its two-record bound. In the selected design failures use direct synchronous
  callbacks and never add retained records. The model does not guarantee a total
  order for concurrent failure callbacks; the existing Observer contract permits
  concurrency. A blocked observer can block its caller, not grow a failure queue.
- `Bind`, `NextMethod`, `LateSend` represent a retained method-A wrapper after
  method B has been dispatched. Identity is immutable and independent of the
  synchronous request span's finished flag. Per-call cancellation is still owned
  by the supplied operation context; only identity is retained by the wrapper.
- `Suppress` is the existing logical StreamClose event with the explicit
  ReasonStreamFinalResponseSuppressed reason. It changes neither physical lifecycle
  records nor the socket state. Existing normal/suppressed protocol controls and
  a same-socket continuation check establish implementation correspondence.

Background WebSocket shutdown records finite terminal metadata without running a
new arbitrary observer callback on the read-pump stack. Foreground operations and
handler cleanup drain it. A handler that ignores cancellation can delay notification;
this is not a claim that the socket stays open. Callback liveness, application
retention, network I/O, goroutine scheduling and unbounded reentrant callbacks are
outside this finite safety model. Direct and mounted tests in the owning plan cover
those practical boundaries relevant to this repair.

The `terminal-*` configurations cover the enclosing generated JSON-RPC SSE
mutex discovered during implementation. Two final callers and one ordinary sender
share admission and writing. `terminal-wrapper` fails callback-outside-lock;
`terminal-split` fails terminal exclusivity after dropping ownership between
admission and write; `terminal-owned` passes all three invariants (65 distinct
states). Preparation failure occurs after serialized admission while the caller owns the
writer; it releases ownership without publishing terminal state, allowing another
caller to complete. The negative wrapper and split variants fail over 14 and 35
distinct states respectively. This mode is independent
of the lifecycle delivery graph above. Run with `-deadlock`: finite completed
executions are expected and are not a liveness claim. Implementation correspondence
is the shared SSE writer terminal operation replacing generated `complete` state;
ordinary/final race and generated callback-lock tests check the boundary.
