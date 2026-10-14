# External SSE test completion ownership (#933)

The callback records an expected event before signaling completion. The old
observer selected independently between an event, request cancellation and
connection completion. Callback cancellation made a successful event compete
with a false timeout or a `context canceled` failure.

`Completion.tla` models one expected event (or the final event of an expected
sequence), callback cancellation, a competing deadline, connection return and
observer completion. `Current` fails `ReceivedWins`: enqueue, cancel, observe
cancellation. `Owned` makes callbacks publish only; the observer cancels, joins
connection completion, then classifies the complete event snapshot. Received
completion wins a simultaneous cancellation. A missing event remains failure.

| Configuration | Result | Generated / distinct states |
| --- | --- | --- |
| Current | `ReceivedWins` counterexample | 14 / 11 |
| Owned | `ReceivedWins`, `Joined` and completion pass | 26 / 13 |

Checked with TLC 1.7.4, jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`,
Java 25. Set `TLA_JAR` to that installed checker and run from this directory:

```sh
model_tmp=$(mktemp -d)
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -metadir "$model_tmp/states" \
  -config Owned.cfg Completion.tla
```

Repeat with `Current.cfg`; expect exit 12 for the stated invariant, not a parser
or resource error. Remove temporary checker outputs after reviewing the concise
results.

The implementation seam is `collectExternalSSEEvents` in
`../../sse_external_collector_test.go`. `TestExternalSSECompletion` controls
completion-before-observation, EOF, cancellation, deadline error, extra events
and unexpected transport failure. Separate controls verify observer cancellation,
connection joining and incomplete cancellation. The mounted decode-error,
oversized-error and ordinary external-client tests retain their exact event and
JSON-RPC envelope assertions.

The model assumes connection return follows the last callback and cancellation
eventually ends the client, as the external client does. It does not prove socket
or parser correctness, exact event counts, error wrapping, callback concurrency,
or the external library's cancellation behavior; Go controls and mounted tests
cover those boundaries. Terminal errors other than cancellation/deadline remain
failures even when the expected events were received.

## Preserved parent witness

Parent `65a75387b619c99fcab3b365d1653d64c8910328`; CI run
38597036619/job 115966310525 reports a timeout after 1.57 seconds with a
three-second deadline. `callback_first.patch` is a scheduling control for that
parent only: it resumes the observer after callback cancellation, without
changing the event callback or select cases. Apply it with `git apply --unidiff-zero` to a disposable parent
source (never generated files), then run:

```sh
cd jsonrpc/integration_tests
GOEXPERIMENT=jsonv2 go test ./tests \
  -run '^TestJSONRPCSSEExternalClientReceivesDecodeErrorsOnMessageEvent$' \
  -count=10 -timeout=2m
```

That controlled run reproduced six failures in ten iterations: false timeout or
unexpected `context canceled`, in 18.451 seconds. Ten unmodified parent iterations
passed in 23.792 seconds; this distinguishes an ordering defect from missing
protocol delivery. The model exhaustively selects the failing ordering; the
controlled network repetition corroborates it without claiming every random
select must fail.
