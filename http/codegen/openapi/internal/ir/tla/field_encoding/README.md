# Encoding Object export

`FieldEncoding.tla` explores two independent export decisions: keeping the
source type that chooses whole-value versus item content application, and
choosing one named Encoding Object across root alternatives. The bounded
scenarios include a source array normalized to an untagged union, a source-absent
singleton normalized to an array, heterogeneous implicit item defaults, and
incompatible authored text/JSON grammars sharing one physical name.

Run explicitly with Java and TLA+ 1.7.4; these commands are never CI gates:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config normalized.cfg FieldEncoding.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config global.cfg FieldEncoding.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config checked.cfg FieldEncoding.tla
```

The normalized-shape policy fails `ApplicationPreserved`. A globally forced
JSON label fails `DefaultsPreserved`. The checked policy preserves source type,
resolves each proposed label across all leaves, and rejects conflicting authored
contracts: 8 generated states, 4 distinct, no invariant failures.

`schema_encoding_source.go` retains source assertions, using a singleton oneOf
when source absence must hide an otherwise necessary concrete type. Its
construction slots preserve correspondence during byte projection. The schema
fingerprint includes the same structural fact. `url_field_encoding.go` groups
captured fields by physical name and compares candidate codecs using the shared
resolver; it does not infer new serialization policy. Tests in
`field_encoding_export_test.go`, `form_tags_test.go`, and the rendered
`EncodingSourceDSL` fixture check recursion, aliases, item defaults, explicit MIME
labels, false flags, name swaps, omission and conflicts. The shared expression
source-type model is under `expr/tla/source_encoding_type`.

The model assumes faithful captured codecs, physical name grouping and valid
source facts. It abstracts metadata syntax, constraints, MIME parsing and
nested schema membership. Its finite label set does not prove complete semantic
equivalence of arbitrary serializers. Direct tests and compiled imported DSL,
rendered JSON/YAML validation, Vacuum, and process-isolated byte comparisons
check those implementation boundaries. Generated transport carrier integration
is validated separately; the model makes no claim about Go conversions.

`OccurrenceOwnership.tla` checks the interaction between source wrappers,
canonical aliases, annotations and outer nullability. The old policy writes an
occurrence's source marker into the named declaration and leaves annotations
inside the source wrapper. The old policy fails outer annotation ownership;
checking declaration isolation separately also fails for a named occurrence. The
checked policy wraps the reference and moves annotations to the wrapper before
nullability is applied: 56 generated states, 40 distinct, no invariant failures.
Run explicitly with the same jar:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config occurrence-old.cfg OccurrenceOwnership.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config occurrence-declaration-old.cfg OccurrenceOwnership.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config occurrence-unwrapped.cfg OccurrenceOwnership.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config occurrence-checked.cfg OccurrenceOwnership.tla
```

The model abstracts schema constraints and actual names; it establishes ownership
only, not JSON Schema equivalence. `TestEncodingSourceTypeCanonicalAliasOccurrence`
and `TestEncodingSourceTypeWrapperAnnotations` cover the shared IR boundary.
Importer nullable-reference and nullable-annotation generation tests check real
DSL evaluation, canonical component allocation and rendered annotations.

The `unwrapped` policy isolates source metadata and annotations but omits the
nullable wrapper when a named target already allows null. A concrete source
assertion then removes inherited null; `NullPreserved` fails. The checked policy
wraps the complete source refinement, as it already does validation refinements.
`TestEncodingSourceTypeOverlayPreservesInheritedNull` checks this correspondence.
