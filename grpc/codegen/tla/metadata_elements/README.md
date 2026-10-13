# Metadata element representation — #877

A bounded representation model compares named and native primitive array elements
at extraction, codec selection, source formatting, service construction,
validation and default storage. Five kinds represent Bytes, String, Boolean,
UInt32 and Float64; constraints and defaults are independent presence flags.

The current design leaves named elements in an unqualified nominal transport
variable and chooses unsupported codecs by their authored name. `codec-only`
changes codec selection without the variable representation. `storage-only`
normalizes the transport variable and reuses the existing constructor (checked by
`TestMetadataElementConstructorProjection`), but leaves validation/default plans
on the old occurrence and omits required source casts. `owned` derives all these
consumers from the same physical occurrence while preserving the nominal service
field. The model is a design obligation, not a proof of generated Go behavior.

Run Java25 with TLC1.7.4 (jar SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
java -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-877-state-owned \
  -config owned.cfg MetadataElements.tla
```

Replace both configuration names to check alternatives. Negative configurations
must exit12 with an invariant counterexample, not a parser error. Owned must
exit0. Completed resolution self-loops with deadlock checking enabled.

Current and codec-only fail ResolvableVariables. Storage-only fails an agreement
obligation (source conversion, constraints or defaults) while the existing shared
constructor remains correct. Owned passes120 generated/80 distinct states.
Exact counts and counterexamples remain in the temporary validation receipt until
independent review; do not commit checker state or raw logs.

## Correspondence and limits

`metadataStorage` resolves a detached physical occurrence and records source
element conversion; `extractMetadata` publishes that shared representation.
Request/header/trailer codecs, TypeName/TypeRef, default literals, validation and
constructor/CLI arguments must consume the same lowered occurrence. FieldType
continues describing the service declaration. NameScope and InitStructFields own
qualified target identities and casts; do not recreate them in transport code.
Named bool/string/float64 formatting needs an explicit conversion before builtin
formatters. Bytes and UInt32 already have a cast in their builtin formatter;
the model does not claim an additional cast is needed for those kinds.

`TestMetadataElementPhysicalStorage` reproduces the five nominal variables;
`TestMetadataElementGeneratedModules` compares complete native and named modules
across request/header/trailer. `TestMetadataElementConstructorProjection`
independently checks the reusable shared constructor boundary for all five kinds.
The preserved explicit-bin Bytes witness separates #877 from #705 key derivation.
The aggregate adds named outer arrays, a type in another generated package,
inherited length bounds and an inherited optional default. The generated CLI
compiles against those same physical argument types. Real TCP peers exercise
binary/empty elements, malformed input, required absence, validation failures and
optional absent/default/supplied values in all three metadata placements.

The model abstracts actual values, exact inherited predicates, nullability,
custom Go representations, nested collections, helper names, runtime framing and
arbitrary type graphs. Tests must check real constraint/default lowering, values,
malformed input and compiled source, including qualified nominal casts. The
selected owner does not extend unsupported metadata shapes or change key policy.
