# Response capture (#608)

One response capture state owns final status and byte accounting for both
request observation and clue HTTP logging. httpsnoop v1.1.0 owns conditional
interface forwarding. HTTP failure routing reads state through the observer,
not through a concrete writer type. Unary handlers share HandlerLifecycle.

The model explores three sequential header/write/flush operations with 101,
103, 200 and 500. 103 remains informational, 101 commits an upgrade, and the
first final status wins. `Legacy` fails `Status` after 103; `Checked` passes
with 13 distinct states. Model assumptions are valid HTTP codes and serialized
writer calls. It does not model bytes, I/O failures, optional interfaces,
hijacked connection I/O or arbitrary middleware. Tests establish those seams;
atomic capture accessors do not make the underlying writer concurrent-safe.

Run with official TLA+ tools v1.7.4 (TLC 2.19 rev `5a47802`), JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Validated using Corretto 25.0.4.1. From this directory:

```sh
for cfg in Legacy Checked; do
  java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
    -metadir "/tmp/loom-capture-$cfg" -config "$cfg.cfg" ResponseCapture.tla
done
```

Legacy must fail the invariant (exit 12), not fail parsing. Checked must exit 0.
Delete checker artifacts after review.

`ResponseCapture.commit` implements the first-final transition; WriteHeader
filters informational responses. Write/WriteString and Flush/FlushError commit
200, while ReadFrom commits only after transferring bytes. An empty ReadFrom
need not commit, matching net/http. No-write state remains zero. Upgrade writes
directly to a hijacked connection are outside capture. The library metrics API
is callback-scoped; it cannot own independently started/ended observation and
its status policy does not cover flush commits and terminal 101.

`TestCaptureResponseWriterInformationalStatus` and the capability regression
failed before implementation. `TestCaptureResponseStatus`, `Capabilities` and
`FastPaths` cover transitions, nested capture, optional interface absence,
forwarded errors, controller operations and optimized I/O. HTTP lifecycle tests
cover committed write failures and failed WebSocket upgrades; compiled generated
HTTP/WebSocket and SSE fixtures cover transport consumers. No generator changes
or generated-output differences are intended.
