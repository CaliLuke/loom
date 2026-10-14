# Scalar binding presence

One mapped scalar has independent transport presence and string contents. Required
checks use presence, optional defaults fill absence, and supplied values reach
conversion even when empty. An empty string is valid without a rejecting DSL
constraint; an empty integer or UUID reaches its codec and fails there.

The bounded model explores all 32 combinations of supplied/empty, required,
defaulted and whether the codec accepts empty. Legacy.cfg reproduces the existing
content-based absence check: a supplied empty optional value violates Presence.
Checked.cfg verifies presence, default and conversion admission for every state
(64 generated states, 32 distinct). It abstracts actual HTTP parsing, custom
codec implementations, generated Go types and validation diagnostics; generated
transport tests exercise those boundaries. It does not change collection lists,
framed Authorization, explicit tags, or body codecs.

Use the pinned TLA+ 1.7.4 tools (see the neighboring basic_auth README for its
checksum), from this directory:

```sh
java -cp "$TLA_JAR" tlc2.TLC -config Legacy.cfg -metadir /tmp/scalar-binding-old ScalarBindingPresence.tla
java -cp "$TLA_JAR" tlc2.TLC -config Checked.cfg -metadir /tmp/scalar-binding-new ScalarBindingPresence.tla
```

The old run must fail Presence, not fail parsing. The checked run must succeed.
Generated checker outputs belong outside the source tree and are removed after
review. The `scalar_element` partial in `template_sources_request_decoder_partials.go`
owns requiredness, defaults, conversion and validation; request/response element
renderers supply raw contents and independently extracted presence.
`TestScalarBindingDecodeOwner` checks the rendered plan.
`TestScalarBindingPresenceGenerated` mounts HTTP requests and decodes responses;
`TestJSONRPCScalarBindingPresence` checks the same request owner through mounted
JSON-RPC. Existing list and Authorization tests constrain neighboring protocols.
