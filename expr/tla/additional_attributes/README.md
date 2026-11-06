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
declared fields and one extras map for every undeclared key, including required
keys. Each field and the map values carry the conjunction
of the obligations of every source.

- ObjectRequired, ObjectConflict, ObjectChain (`Lowering = "map-keys"`):
  required undeclared keys stay undeclared and use the map. Exit 0
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
  map entries typed by `AdditionalAttributes` and stay subject to every
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

## Literal required-key storage decision

Required undeclared members now share the additional-members map with optional
undeclared members. The earlier separate-field representation could not express
all legal JSON keys: mapping syntax reinterpreted colons, JSON field tags parsed
commas as options, and unrelated keys could produce the same Go identifier.
Rejecting these keys would reject valid typed-object contracts; quoting tags is
not supported by the pinned JSON codec. Shared custom object codecs would add a
second representation and codec owner. One map preserves exact identity and
uses the existing additional-value validation and conversion paths.

`RequiredKeys.tla` reproduces the prior identity loss with `RequiredSynthetic.cfg`
and checks exact membership and value preservation with `RequiredMap.cfg`.
Its finite universe includes colon, comma, empty and ordinary names, and zero,
nonzero and null values. Null eligibility is intentionally delegated to the
existing additional-value model and runtime codec. It does not prove arbitrary
Go names or reflection behavior. Run explicitly from this directory:

```sh
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-required-synthetic -config RequiredSynthetic.cfg RequiredKeys.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-required-map -config RequiredMap.cfg RequiredKeys.tla
```

The first must fail `RequiredExact` or `Preserved` with exit 12; the second
passes both invariants (12,288 generated / 8,192 distinct states, depth 2).
These remain opt-in design tools, not build, test, commit, CI or release gates.
Delete temporary checker output after review.

Implementation owners: `RequiredAdditionalMembers` supplies exact keys;
`MappedAttributeExpr.Remap` and effective constraints preserve their identity;
shared codegen renders map membership checks and defaults, and transforms the
single existing carrier. `ObjectFields` describes declared fields only. Inline
JSON Schema and OpenAPI keep required keys without inventing properties. The
importer reserves only the additional map's own Go field name.

`TestRequiredAdditionalMembersStayInMap`,
`TestRequiredAdditionalMemberIdentityAcrossMappings`,
`TestRequiredAdditionalMemberDefaultAndValidation` and
`TestRequiredAdditionalLiteralKeysEmission` cover these seams. The maintained
HTTP/JSON-RPC composite specimen now checks required object/union/collection
extras, punctuation, empty names, Unicode, case collisions, zero versus absent,
and nullable null versus absent. The original HTTP specimen covers inheritance,
recursive values and declared/transport-omitted collision rejection.

The generated Go API changes: regenerate and move accesses to formerly
synthesized fields into `AdditionalAttributes["literal-key"]`. The accepted DSL
and JSON contracts stay the same. See `docs/dsl-reference.md` for migration.

### Acceptance evidence

With Go 1.27.3, comparison against `d1be7f0a0` used 26 representative maintained
designs, retaining the parent composite input on both sides. Of 895 raw/rendered
output sections, 871 were byte-identical and 24 changed, all in the typed
additional-attributes specimen. Intended differences remove synthetic fields,
move their examples/defaults into the map, check required map membership, and
stop reserving required extra names as field collisions. OpenAPI stayed equal.
The expanded composite input additionally compiles and runs through generated
HTTP and JSON-RPC servers and clients. Five isolated generations of 25 maintained
designs, including the expanded composite input, produced identical bytes.

The full `expr` and shared `codegen` packages, focused generated runtime and
OpenAPI tests, selected importer typed-object/reference-context tests, and
scoped Go lint passed. Rendered composite JSON/YAML for OpenAPI 3.1 and 3.2
passed libopenapi validation and pinned Vacuum 0.36.2 rules. A whole importer
package run exceeded its default ten-minute timeout; acceptance uses the bounded
owner tests rather than treating that incomplete run as a pass.

`TestMappedRequiredDeclaredAndAdditionalMembers` distinguishes a declared
mapped field from a literal undeclared key through named-type flattening.
`TestRequiredAdditionalMemberHashIdentity` ensures one required key `a-b` cannot
collide with two keys `a` and `b`; both structural hash paths encode key boundaries.
These implementation cases exceed the finite model's simplified wire-name
universe. The composition models still pass for required, conflicting and chained
policies; semantic promotion still gives its expected counterexample.
