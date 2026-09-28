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
