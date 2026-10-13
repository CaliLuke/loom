# gRPC presence admission (#804)

`Admission.tla` models the ordering boundary between service contract completion,
transport preparation, admission and finalization. Transport placeholders do not
contain inherited presence intent until finalization. Admission must inspect
service intent as well as explicit transport declarations. Inspecting only the
placeholder can admit a source contract that a protobuf pointer cannot represent.

The candidate inspects effective source and local intent. `anyMessage` abstracts
only a position with a dedicated protobuf Value representation that preserves
presence; it is not a blanket exception for string metadata, collection elements,
or union branches. Message-root Any is normalized into a message field by the
existing generator. The model assumes the shape/position classifier is correct;
it neither proves that classifier nor generated conversion behavior.

Use Java25 and TLA tools1.7.4 (TLC2.19), JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config legacy.cfg -metadir /tmp/loom-grpc-presence-legacy Admission.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config effective.cfg -metadir /tmp/loom-grpc-presence-effective Admission.tla
```

Legacy violates `NoLostPresence` after31 distinct states: a nullable source is
hidden by an explicit mapping with no local Nullable declaration. The effective
candidate passes all36 states and preserves native/Any acceptance. These results
are bounded ordering evidence, not proof of the position classifier or Go output.

Correspondence:

- `GRPCEndpointExpr.Prepare` allocates empty request/response attributes.
- `validateNullableTransport` previously saw those placeholders before
  `finalizeRequestMessageFromPayload`, `finalizeStreamingRequest`, and response
  finalization copy source fields. Explicit mappings inherit their source type
  later as well.
- `containsUnsupportedGRPCPresenceRecursive` already follows named types,
  collections and unions. The source-aware admission owner reuses that
  traversal, with the correct message-root versus nested position fact.
- `TestGRPCPresenceAdmissionUsesServiceContract` exercises complete DSL evaluation
  for unary/streaming request/response and five representation shapes. Existing
  `TestGeneratedNullableAnyRootPayloadRoundTrip` checks the supported Any
  exception with absent, null and concrete values through generated converters.

No model output belongs in Git. Remove task-owned checker state and raw output
when validation and review finish, preserving summarized results here.

The implementation lives in `expr/grpc_presence.go`. Its checks include complete
service occurrences, scope-correct inherited error sources and explicit local
presence. Local checks pair source-owned types with authored presence without
mutating either graph. Metadata resolves mapped source fields and applies the
non-JSON null restriction; protobuf Value is only a message exception. The
normalized root Any predicate now agrees with the existing generated converter.
New direct tests cover named recursion, graph immutability, metadata classification
and still-rejected nullable collection elements. Full DSL tests cover explicit
local declarations, mapped metadata, inherited errors and malformed mappings.
`TestGeneratedLocalNullableAnyPresence` exercises the previously false-rejected
Any mapping with absent, null and concrete values in generated code.


## Local traversal correction from independent review

The first implementation checked only each local root and its direct nullable
fields. Independent review found that nested object members, array elements and
scalar message wrappers could silently lose locally declared nullability. The
ordering model above assumes a complete local predicate; it does not establish
that completeness. Its passing result therefore did not rule out this defect.

`LocalPresence.tla` now separately explores root/field/descendant depth, scalar
wrappers, concrete versus Any source types, and field versus collection positions.
The shallow candidate violates `LocalIntent` after38 states. Treating every Any
position as supported also violates `LocalIntent` after36 states. Recursive
positional admission satisfies both `LocalIntent` and `SupportedValues` across
all48 states. It abstracts graph pairing and cannot prove arbitrary recursion or
Go generation; direct recursive graph tests and generated Any controls cover
those implementation seams.

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config shallow.cfg -metadir /tmp/loom-grpc-presence-local-shallow LocalPresence.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config all-any.cfg -metadir /tmp/loom-grpc-presence-local-all-any LocalPresence.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config recursive.cfg -metadir /tmp/loom-grpc-presence-local-recursive LocalPresence.tla
```

`unsupportedGRPCMappingPresence` visits every local descendant and pairs it with
its source-owned type without rewriting either graph. Active attribute pairs
terminate recursive named graphs; sibling occurrences are checked independently.
A scalar message's local object is a container: its wrapper field pairs with the
whole source occurrence. Object fields keep field-presence support; array/map
members and union branches do not acquire it. Metadata uses the same pairing but
rejects null at every location. `unsupportedGRPCPresenceOccurrence` owns the
message-position predicate shared with ordinary source graph checks.

Correspondence tests:

- `TestGRPCPresenceAdmissionChecksNestedLocalDeclarations`: nested object, array,
  map and scalar-wrapper declarations, with supported Any field/wrapper controls
  and rejected nullable Any collection members.
- `TestGRPCPresenceAdmissionTraversesRecursiveLocalGraphs`: active-cycle
  termination without dropping later local nullable members or mutating input.
- `TestGRPCPresenceAdmissionRetainsLocalUnionPosition`: a local branch does not
  gain the Any object-field exception.
- Existing generated root and local-field Any tests retain their absent/null/value
  roundtrips and build/vet obligations. Accepted metadata, streaming and Any output
  remains byte-identical to the reviewed artifacts across isolated generations.
