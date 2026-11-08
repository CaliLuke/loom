# Imported boolean annotation presence

The normalized importer keeps `deprecated`, `readOnly`, and `writeOnly` values
separate from their authored-presence bits. False with presence is an annotation;
false without presence is absence. Programmatically constructed true values
remain annotations even without a parsed presence bit.

The same rule must govern both creation of a DSL block and emission of metadata
inside that block. An emitter that preserves false is insufficient when the
caller skips the block. Contributor and occurrence declarations keep separate
metadata: a local false must not disappear and expose an inherited true.

`SchemaAnnotationPresence.tla` models one boolean annotation independently at
each of those two owners. Its four states per owner include absent false,
authored false, authored true and programmatic true. It compares rendered owner
metadata and the effective local-or-contributor value with the intended facts.

| Configuration | Policy | Result |
| --- | --- | --- |
| `truthy` | Blocks and emission both depend only on value | `OccurrencePreserved` fails, 19 distinct states visited |
| `emitter-only` | Emission preserves presence, block selection still tests value | Same failure, 19 distinct states visited |
| `presence` | Both use presence or a true value | All three invariants pass, 32 distinct states |

Implementation correspondence:

- `retainSchemaPresence` captures the authored keyword presence.
- `Schema.hasBooleanAnnotation` supplies the shared presence-or-true rule to
  `renderer.hasSchemaBlock` and `renderer.schemaAnnotations`.
- The emitter uses the stored boolean value, preserving false as `Meta(...,
  "false")` without changing contributor facts or constructor selection.
- `TestSchemaBooleanAnnotationBlocks` checks all three states for all three
  keywords across attributes, declarations, bodies and collection members.
- `TestPrimitiveBooleanAnnotationOwners` analyzes every contributor/local
  absent/false/true combination for each keyword, compiles the rendered design
  and checks both native declaration owners independently. Existing annotation
  tests cover programmatic true values and nullable composition semantics.

This model assumes normalized values and ownership are already correct. It does
not model parsing, general schema normalization, reference resolution, Go graph
copying or OpenAPI export. It does not change nullable-wrapper annotation
combination policy. Those remain implementation and separate model obligations.

Run from the repository root with Java 25 and the existing TLA+ tools 1.7.4:

```sh
for policy in truthy emitter-only presence; do
  java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC \
    -workers 1 -metadir "/tmp/loom-schema-annotation-$policy" \
    -config "internal/openapiimport/tla/schema_annotation_presence/$policy.cfg" \
    internal/openapiimport/tla/schema_annotation_presence/SchemaAnnotationPresence.tla
done
```

The first two configurations exit 12 with the expected counterexample; the
presence policy exits 0. These checks are opt-in design tools, never build,
hook, CI or release gates. Remove temporary checker directories after review.
