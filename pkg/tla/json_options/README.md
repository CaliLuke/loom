# JSON options across nested codecs

`JSONOptions.tla` models the option needed to encode or decode canonical
boolean map keys after generated unions and presence wrappers start fresh JSON
operations. An outer operation starts with the option enabled, as the CLI did
before #559. Each nested `union`, `optional`, or `nullable` layer then starts its
own operation. The terminal map codec requires the option.

The three configurations compare the original behavior and two repairs:

- `outer.cfg`: only the outer operation has the option. TLC finds a rejected
  canonical map after a wrapper restarts JSON.
- `union.cfg`: only unions reapply the option. TLC still finds a rejected map
  inside an optional wrapper. Fixing one codec boundary is insufficient.
- `all.cfg`: every framework-owned restart reapplies the option. TLC checks
  1,332 states, all 120 wrapper chains of lengths one through four, and both
  encoding and decoding, with no safety or termination failure.

Run from this directory with a TLA+ tools jar available:

```sh
java -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir "$(mktemp -d)" -config outer.cfg JSONOptions.tla
java -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir "$(mktemp -d)" -config union.cfg JSONOptions.tla
java -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir "$(mktemp -d)" -config all.cfg JSONOptions.tla
```

The first two commands must fail the invariant; the last must succeed. This
bounded model checks option handling, not JSON parsing, type generation, or
custom codec precedence. The same invariant extends by induction to any finite
chain of these layers: each restart establishes the option independently of
its caller, so the terminal map always receives it. Generated and runtime Go
tests check that the real boundaries use these options, preserve custom JSON
and text codecs, reject malformed input and noncanonical keys, retain boolean
value syntax, and preserve deterministic marshaling in custom marshalers.

## Presence decoder options (#1057)

`PresenceDecode.tla` models chains of Optional and Nullable decoders with caller
strictness and custom codecs, plus Loom's boolean-key fallback. The existing
restart model above remains relevant to encoding and generated union codecs.
Presence decoding now uses the enclosing decoder instead of restarting JSON.

- `presence_restart.cfg` reproduces #1057: reapplying only framework defaults
  drops the caller's options (31 states before the counterexample).
- `presence_inherit.cfg` preserves caller options but loses the default
  boolean-key support when the caller did not supply it (31 states).
- `presence_compose.cfg` preserves caller options and adds the fallback at each
  boundary: all 30 chains of lengths one through four pass, with 128 states.

Run each configuration explicitly, using a temporary directory for checker data:

```sh
java -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir "$(mktemp -d)" -config presence_restart.cfg PresenceDecode.tla
java -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir "$(mktemp -d)" -config presence_inherit.cfg PresenceDecode.tla
java -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir "$(mktemp -d)" -config presence_compose.cfg PresenceDecode.tla
```

Validated with TLC 1.7.4. The first two configurations must violate an invariant;
the composed configuration must pass. Remove temporary checker data afterward.
This model abstracts options as capabilities: it does not prove codec ordering,
JSON parsing, error paths, or presence-state mutation. `pkg/presence_decode_test.go`
checks those implementation boundaries, including caller-codec precedence,
strictness, stringified numbers, null policy, and one-value stream consumption.
`http/codegen/presence_decoder_options_test.go` checks generated required and
optional HTTP request bodies, safe errors, service invocation and decoded state.
Implementation ownership is `Optional.UnmarshalJSONFrom`,
`Nullable.UnmarshalJSONFrom`, and `unmarshalPresenceJSON` in `pkg/json_options.go`.
Generated unions that restart byte-based decoding and custom application codecs
are outside the model's caller-option guarantee.
