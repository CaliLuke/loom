# Client Debug response ownership

Run with Java and pinned TLC 1.7.4, using external state directories:

```sh
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC -metadir /tmp/loom-827-model-Checked -config Checked.cfg ClientDebugResponse.tla
```

Eager and Replace isolate current response prefix-read/error-replacement defects.
Duplicate rejects notifying on every EOF/Close. Latest rejects completion that
prints mutable latest-exchange state instead of the completing exchange. Checked
uses passive reads and per-exchange once-only completion. Finish abstracts EOF or
caller Close; explicit Fprint is a stuttering snapshot. Two exchanges and three
read/completion events are explored. Error tokens stand for exact original
byte/error observations; retention and concurrent Close safety remain covered by
the shared transport request-capture model and direct runtime tests. This model
does not establish redaction parsing, Go race freedom, HTTP framing or liveness
for an abandoned unread/unclosed body (which intentionally emits no automatic
summary).

`http/client_response_observation_test.go` checks passive Do/Fprint and source
Read/Close identity. `http/client_debug_completion_test.go` checks EOF/Close
once-only logging, overlapping exchanges, nil/NoBody, cap boundaries, structured
redaction/omission, streaming return, GetBody preservation and concurrent snapshots.
`debugExchange` owns completion; `debugCompletionBody` forwards through the shared
`transport.CaptureBodyReads` owner. Close can print an incomplete snapshot before
an in-flight read returns; the model does not claim otherwise or prove Go races.

Readiness runs: Eager, Replace, Duplicate and Latest exit12 at the intended
invariants (2/2, 6/5, 17/14, 41/31 generated/distinct states). Checked passes
2078/855 states. The same model sources remain valid through implementation;
model invariants were not weakened to accommodate code.
