# Typed additional attributes (#1013)

`AdditionalAttributes` gives an object a typed policy for the members it does
not declare, as JSON Schema `additionalProperties` with a schema value does.
`Required` may name such members. These models decide three design questions:
how `Extend` composes typed policies, how the OpenAPI importer translates
`allOf: [$ref B, S]` into `Extend(B)`, and how the generated codec carries
extras.

## Models

`ObjectComposition.tla` enumerates one or two roots and an `Extend` chain of
one or two levels over the keys `id`, `note` and `x`. Each source has
declarations from a small schema palette, an open, closed or typed policy, and
required keys that it may not declare. `Spec` is `allOf`: every source
constrains every key it does not declare. `Compose` is Loom's physical layout:
semantic fields, a synthesized field for each required key that no source
declares, and an extras domain. Each field and the extras carry the conjunction
of the obligations of every source.

- ObjectRequired, ObjectConflict, ObjectChain (`Lowering = "physical-fields"`):
  required undeclared keys stay undeclared and become synthesized fields. Exit 0
  with Sound, UnsatisfiableJustified, UnsupportedSatisfiable and
  EmptyExtrasIsClosed: 677376/338688, 1350000/675000 and 128000/64000
  generated/distinct states.
- ObjectSemanticPromotion (rejected): rewrite a typed root's required undeclared
  keys into declarations before composition. Exits 12, Sound violated in the
  initial state: the promoted key stops being constrained by the root's policy
  at later levels, so a derived declaration of `note` admits values the root
  forbids.

`ImportAllOf.tla` translates a raw `allOf` of a base `B` and a sibling `S`.
A DSL `Extend` level semantically declares every key of `B`, but raw `S` still
constrains the keys of `B` that it does not declare.

- ImportChecked: keep the closed-sibling admission and, when `S` is typed,
  redeclare each `B`-only key with the type of `S`. Exits 0 with
  TranslationSound, 41472/20736 states.
- ImportUnchecked (rejected): copy `S` as the local level. Exits 12,
  TranslationSound violated in the initial state.

`AdditionalCodec.tla` checks a generated object with an optional declared `id`
and `Int` extras.

- CodecCorrected: extras decode into a carrier that rejects `null`, and
  encoding rejects an extra named like a physical member. Exits 0 with
  DecodeFaithful and RoundTrip.
- CodecPlainMap (rejected): a plain `map[string]int` decodes `null` as `0`.
  Exits 12, DecodeFaithful violated.
- CodecNoCollisionCheck (rejected): without the outbound check, an extra named
  `id` encodes as the declared member and decodes into the wrong field. Exits
  12, RoundTrip violated.

## Running

Use TLA+ tools 1.7.4, TLC 2.19 revision 5a47802, JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Validated with OpenJDK 25.0.5. From this directory:

```sh
for pair in ObjectRequired:ObjectComposition ObjectConflict:ObjectComposition \
  ObjectChain:ObjectComposition ObjectSemanticPromotion:ObjectComposition \
  ImportChecked:ImportAllOf ImportUnchecked:ImportAllOf \
  CodecCorrected:AdditionalCodec CodecPlainMap:AdditionalCodec \
  CodecNoCollisionCheck:AdditionalCodec; do
  cfg=${pair%%:*}; module=${pair##*:}
  java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers auto \
    -metadir "/tmp/loom-1013-$cfg" -config "$cfg.cfg" "$module.tla"
done
```

The rejected configurations must fail their invariants, not parsing. Keep logs
and checker state outside the repository and remove them after review.

## Design decisions

- Required undeclared members keep their semantic classification. They are
  physical fields typed by `AdditionalAttributes` but stay subject to every
  source's policy, so `Extend` cannot weaken them.
- Composition conjoins policies per source. A typed base becomes an extra
  refinement parent of each derived field it does not declare. Disjoint typed
  domains compose to a closed object. A conjunction that admits only `null`
  is rejected as unsupported, because Loom cannot carry it.
- A required member with an empty domain makes the design unsatisfiable, and
  validation rejects it.
- The importer redeclares base-only properties under a typed sibling; see
  `referenceObjectAdmission` in `internal/openapiimport/analyze_reference_object.go`.
- Generated extras use `json:",embed"` maps whose values reject `null`, and all
  members match names with `case:strict`. Transforms call
  `loom.CheckAdditionalMembers` (`pkg/additional_members.go`) so that an extra
  never reuses a physical wire name.

## Choices outside the models

- An unvalidated `AdditionalAttributes(Any)` admits every value, so it does
  not constrain a conjunction. When it is the only inherited policy, the
  derived object keeps it, so that the generated type still carries extras.
  An explicit open policy over any typed base is rejected.
- The models intersect value sets. Loom must also carry the conjunction in
  one Go type, so it rejects overlapping policy types that do not refine one
  another, such as `Int` and `Float64`, as it does for declared members. Only
  disjoint JSON kinds, contradictory bounds and empty enum intersections close
  the object. Other constraint errors are reported.
- A transport projection that binds a declared member to a path parameter, a
  header or a cookie records it as omitted on the typed body. The body does not
  require the member, and its extras cannot use the member's name, because the
  service type still declares it.

## Implementation correspondence and limits

`composeAdditionalAttributes` in `expr/attribute_composition.go` implements the
composition. `TestAdditionalAttributesComposition`,
`TestAdditionalAttributesPolicy`, `TestAdditionalAttributesValueAdmission` and
`TestAdditionalAttributesRejectedPlacements` cover it in `expr`.
`TestImportObjectsWithTypedAdditionalProperties` and
`TestImportTypedReferenceSibling` cover the importer.
`TestGeneratedAdditionalAttributesRuntime` decodes, validates, transforms and
encodes generated code.

The models abstract keys as wire names and values as small domains. They
exclude nested and recursive objects, views, case-insensitive matching (which
validation rejects), non-JSON transports and the OpenAPI output.
