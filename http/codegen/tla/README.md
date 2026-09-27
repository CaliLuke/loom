# WebSocket payload nullability

`WebSocketPayloadNull.tla` checks the framing boundary behind #464. HTTP
WebSocket clients send JSON `null` to finish their input stream. The server
therefore cannot distinguish a nullable root payload from end-of-stream.
Nested nulls are inside a non-null frame and do not have this ambiguity.
Responses end with a WebSocket close frame, and JSON-RPC payloads have an
envelope; neither uses the HTTP request's root-null sentinel.

The model enumerates both directions, HTTP and JSON-RPC framing, nullable and
non-nullable roots, four type shapes, and concrete, root-null, nested-null,
and end messages. Shapes abstract away Go representations: this model proves
message distinguishability, not generated Go compilation or JSON validation.
Compile and generated transport tests cover those implementation boundaries.

Run with a Java runtime and a TLA+ tools jar, keeping TLC state outside the repo:

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config payload_null_before.cfg -metadir /tmp/loom-payload-null-before WebSocketPayloadNull.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config payload_null_after.cfg -metadir /tmp/loom-payload-null-after WebSocketPayloadNull.tla
```

The before configuration must fail `AcceptedMessagesPreserved`: an accepted
nullable HTTP request sends `null` and receives end-of-stream. The after
configuration rejects only nullable HTTP request roots and must satisfy both
invariants (416 reachable states). The generated WebSocket decoding regression
also checks that a root-null frame remains end-of-stream and nested nulls remain
payload values.
