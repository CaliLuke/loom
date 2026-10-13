# Problem-document error properties

This model compares current status inference, adding wire fields while
retaining inference, and explicit property transport. The source has any of the
eight combinations of Timeout, Temporary and Fault. Status is deliberately
independent of those properties: the DSL permits arbitrary response mappings.

The selected set models a full built-in body (all properties), an explicitly
pruned object body (some or no properties), or explicit header/cookie mappings.
The set is the union conveyed across body, headers and cookies. Scalar Body and
Body(Empty) map remaining members to headers, so neither implies an empty set.
Reconstruction supplies selected values to the client; omitted properties remain false.
The model represents true properties as a set. Runtime tests must separately
prove that a full document includes explicit false booleans and accepts the
generated schema. Status can still determine a response code when no mapping is
authored; it must never invent a source error property during decoding.

`current.cfg` models the missing fields and status-derived properties. The
`wire-only.cfg` rejected repair adds fields but leaves the lossy decoder. Both
must produce a PreserveSelected or NoInventedProperties counterexample.
`explicit.cfg` carries the original properties through projection and consumes
them without inference. Its passing check is bounded to this representation;
it does not establish error aggregation, status selection, safe-message policy,
Go pointer layouts, malformed input validation, or generated compilation.

Implementation seams are `http.NewProblemResponse`,
`expr.ErrorResult`, `http/codegen.buildProblemClientResult` and
`http.ProblemErrorFromBody`. The first two now carry explicit fields; the last consumes those fields without
status inference. `buildProblemClientResult` passes the reconstructed document.
`http/problem_properties_test.go` checks every value combination and explicit
false wire values; generated three-transport tests cover DSL helper construction
and malformed full booleans. The selected-body harness covers pruned, scalar,
empty and header/cookie projections; OpenAPI tests parse and lint the same shape.
HTTP and JSON-RPC share this boundary. gRPC already transmits explicit properties
and is an unchanged cross-transport control.

Use Corretto 25.0.4.1 and TLA+ tools 1.7.4 (TLC 2.19 revision 5a47802), JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -config explicit.cfg \
  -metadir /tmp/loom-problem-properties-explicit ProblemProperties.tla
```

Repeat with `current` and `wire-only` in separate temporary directories. Expected
exit statuses are 0 for explicit and 12 for each intended counterexample; parsing
failures do not count as negative controls. Keep these sources and concise
results; remove checker output after independent review and validation finish.
