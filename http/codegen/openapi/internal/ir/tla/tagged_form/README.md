# Tagged form schema projection

Native tagged form object branches flatten onto the root; nested object/map
fields use their captured JSON field codecs. They do not acquire an implicit
nested-bracket grammar. JSON occurrences of the same union keep their envelope.
The schema projection is the pullback of each payload constraint through adding
the discriminator property, on a detached occurrence graph.

`TaggedFormProjection.tla` covers required fields, closure and property-count
bounds. `envelope.cfg` reproduces the wrong root shape in 97 states;
`unshifted.cfg` reproduces unchanged payload counts in 97 states; `captured.cfg`
checks 192 states. Its precondition excludes an existing discriminator key from
the payload. `HTTPFormBodyEncoding` establishes this at admission: captured
field names (including inherited and full form-tag renames) cannot collide;
typed additional members are rejected by the shared form-body owner.

`TaggedFormAdmission.tla` records another representability precondition: a
flattened nullable object has no distinct null spelling. The admitted control
fails `Representable`; rejecting that shape checks four distinct states. JSON
value-key map branches retain a null spelling. Repeated nullable scalar arrays
remain rejected by their existing shared codec, independently of this check.

```sh
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config envelope.cfg -metadir /tmp/loom-tagged-envelope TaggedFormProjection.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config unshifted.cfg -metadir /tmp/loom-tagged-count TaggedFormProjection.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config captured.cfg -metadir /tmp/loom-tagged-flat TaggedFormProjection.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config nullable-admitted.cfg -metadir /tmp/loom-tagged-null-old TaggedFormAdmission.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config nullable-rejected.cfg -metadir /tmp/loom-tagged-null-new TaggedFormAdmission.tla
```

`analyzeFormOccurrence` consumes captured TaggedBranches and registers detached
schema components with the existing representation finalizer. The object
projection transforms same-value references/compositions and preserves nested
property/item contracts. `TestTaggedFormObjectConstraintProjection` uses the
independent schema validator for closed referenced allOf objects, enum, required
and property-count constraints; `TestTaggedFormSchemaAcceptance` checks actual
maintained fixture roots and unchanged JSON occurrences. Expr and DSL tests
check admission, renamed/inherited collisions and source isolation. Generated
wire tests cover actual nested JSON, arrays, maps and conversions. The bounded
models do not prove arbitrary JSON Schema equivalence, recursive graph naming,
example conversion or Go codec behavior; those remain implementation tests.
