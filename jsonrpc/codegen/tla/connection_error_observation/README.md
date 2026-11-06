# Connection error observation

This model describes the synchronization obligation in the generated-module
`TestConnectionFailureEndsEveryStream` regression in
`../../websocket_demux_test.go`. `WebSocketClientConn.fail` in
`../../../websocket_client.go` publishes errors to waiting requests before
calling the user error handler. Receiving every request error therefore does
not establish that the handler has run.

The model has two streams, a connection failure, receipt of each request error,
a handler report and a test assertion. Each action occurs at most once. Weak
fairness of the finite, monotonic action set excludes indefinite scheduling
stalls. It abstracts socket I/O, Go memory visibility, timeout duration and
other shutdown paths. It establishes the observation ordering within these
assumptions, not correctness of the entire connection implementation.

`Current.cfg` models the old test: it asserts after both receipts, without
joining the report. TLC reproduces the race (13 generated states, 10 distinct,
depth 5): failure, two receipts, assertion, with no handler report.
`Joined.cfg` requires receipt of the report before asserting. TLC checks the
synchronization invariant and eventual assertion (15 generated states,
10 distinct, depth 6), with no errors.

Run explicitly with TLA+ tools 1.7.4 and Java (checked with Java 25):

```sh
cd jsonrpc/codegen/tla/connection_error_observation
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -metadir /tmp/loom-observation-current -config Current.cfg Observation.tla
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -metadir /tmp/loom-observation-joined -config Joined.cfg Observation.tla
```

The first command must report `ObservationIsSynchronized` violated; the second
must pass. Model checking is an optional design tool, never a test or CI gate.
Delete temporary checker directories after inspecting the results.

The implementation regression holds the error handler until both outstanding
requests and later operations have observed failure, then releases it and
receives its nonnil error through a channel within the existing test deadline.
This checks that a blocked callback cannot hold up requests and avoids assuming
that callback completion precedes request completion. A deferred release also
unblocks the callback when an earlier assertion fails.

Validation against parent `4799149d7e9ce4a361419327d95570454b15b8d2`:

- With the handler deliberately held, the old assertion failed in all three
  generated race-test runs. This deterministically reproduced the intermittent
  failure first seen in the push check.
- The corrected generated-module test passes three independent outer runs;
  each generates and compiles the fixture and runs its tests with `-race -count=3`.
- The runtime connection failure and close tests pass with the race detector.

Only the test and this model change; runtime behavior, generator templates and
fixture output are unchanged.
