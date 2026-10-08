# SSE frame completion

`FrameCompletion.tla` models the complete-event contract consumed by
`http.SSEStreamReader`. Framing and parsing are implemented by the SSE library,
not a Loom scanner. A stream
contains up to five tokens: a nonempty data line fragment, CR, or LF. Delivery
can split the input at any token. Scanning tracks the current line, pending
block and the LF following CR. A blank line completes a block; EOF discards a
pending block. This follows the [SSE interpretation contract](https://html.spec.whatwg.org/multipage/server-sent-events.html#event-stream-interpretation).

The legacy EOF action emits a pending block and violates `OnlyCompleteFrames`
for a single data token. The checked action preserves the ordered sequence of
complete blocks, emitting each exactly once, and clears the tail at EOF.
TLC checks 7,016 distinct states (16,173 generated) for the checked configuration.
The legacy configuration must fail with an invariant violation.

Use Java and TLA+ tools 1.7.4, with jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -metadir /tmp/sse-framing-checked -config checked.cfg FrameCompletion.tla
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -metadir /tmp/sse-framing-legacy -config legacy.cfg FrameCompletion.tla
```

`Scan` abstracts the library parser's line and block processing; `EOF` abstracts
its dispatch rule. The legacy case also reproduces the removed `readFrame`
behavior, which returned an incomplete tail. Complete-frame positions stand for
owned events. The model abstracts field parsing, memory and frame-size limits.
It does not prove Go refinement or unbounded streams.

`ReaderLifecycle.tla` checks the adapter's iterator ownership for up to two reads.
`Read` advances the iterator under `readLock`; `Close` closes the body first;
`Stop` waits for that lock before releasing the iterator. The unguarded candidate
violates `ExclusiveIterator` by stopping during an active read. The checked
candidate visits 13 distinct states (28 generated), without a violation.

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -metadir /tmp/sse-lifecycle-checked -config lifecycle-checked.cfg ReaderLifecycle.tla
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -metadir /tmp/sse-lifecycle-unguarded -config lifecycle-unguarded.cfg ReaderLifecycle.tla
```

The lifecycle model abstracts `sync.Once`, context callback scheduling, Go's
iterator implementation and close errors. It checks safety, not liveness. The
response body's Close must unblock Read, as required by `net/http`; the adapter
cannot guarantee cancellation for an arbitrary reader that violates this rule.
Generated fixture tests check concurrent close, canceled reads, context-error
precedence, and cancellation after a complete read. Direct lifecycle tests check
close before reading and between events, error identity, and close-once behavior.

`TestSSEStreamReaderDiscardsIncompleteTail` checks real bytes, every two-chunk
split and bytewise reads, all three line endings, repeated EOF, complete events
before a tail and returned-buffer ownership. The existing SSE reader tests cover
cancellation, close and frame limits. Both checked-in ticktock fixtures exercise
the generated clients through `TestGeneratedSSEClientDiscardsIncompleteTail`.
`ParseSSEStream` and `ParseSSEEvent` use the same library semantics; incomplete
final events are discarded. The single-event helper reports an error when no
complete event was present.
