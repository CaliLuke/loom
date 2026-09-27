# Views on JSON-RPC streams

`StreamingViews.tla` models two result frames whose selected views may differ.
The body and `loom_view` marker come from the same projection. Decoding each
frame independently preserves the selection and the projected payload shape.
The old path omits the marker and decodes directly into the canonical Go type;
it neither identifies the selected view nor validates that view's fields.

`ViewDecoding.tla` checks default and fixed selection, old frames without a
marker, unknown names, non-string metadata (including null), fixed-view
mismatches, and missing required fields.
Only accepted frames must satisfy the selected view's field requirements.

From this directory, run TLC with Java and a TLA+ tools jar:

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config before.cfg -metadir /tmp/loom-view-before StreamingViews.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config after.cfg -metadir /tmp/loom-view-after StreamingViews.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config decoder-before.cfg -metadir /tmp/loom-view-decoder-before ViewDecoding.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config decoder-after.cfg -metadir /tmp/loom-view-decoder-after ViewDecoding.tla
```

Both `before` configurations must fail with counterexamples. Both `after`
configurations must pass. TLC 2.19 finds lost per-frame selection and acceptance
of invalid result bodies in the old configurations, and no invariant violation
in either corrected configuration.

These are finite-state models, not proofs of the Go generator. They abstract
views as sets of required fields after transport field mapping, and assume an
ordered transport and an atomic
snapshot of the selected view during projection. They do not model arbitrary
schemas, nil collection elements, network failures, or application mutation of a
result during a send.
The generated module regression tests cover actual scalar and collection view
projection, WebSocket and SSE framing, changing views between messages, fixed
views, absent markers, invalid metadata, required-field validation, and rejection
of null collection elements without panics. The raw
GET listener test covers view selection with ID-less terminal suppression.
A separate generated regression test preserves WebSocket response-ID fallback
before view validation; ID mapping is outside these models.
The models supplement those tests and exact generated-output comparisons.
