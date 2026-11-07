# Captured codecs for dynamic form-map entries

A map root owns a captured `HTTPFormEncoding.Element`, just as an object owns
captured field codecs. `FormMapFields.tla` compares the old reflection-derived
field grammar with consumption of that captured element across ordinary
requests, media branches, and untagged root alternatives. The old policy loses
JSON item application for arrays of records/arrays and branch-specific item
codecs. It fails `CodecAgreement` for an array of objects; the checked policy
passes 36 generated / 18 distinct states with TLC 1.7.4.

Run explicitly from this directory; these commands are not CI or build gates:

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 -config old.cfg -metadir "$TMPDIR/form-map-old" FormMapFields.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 -config checked.cfg -metadir "$TMPDIR/form-map-checked" FormMapFields.tla
```

`form_map.go` emits dynamic names from actual map keys, then passes each value
through the captured JSON application or existing text/content-union owner.
Names are sorted, empty and bracket-containing keys remain literal, and entries
with no wire spelling fail before publication. Union selection still performs
schema membership before typed decoding and outgoing replay. The private
`Element.wireNameExpr` supplies dynamic names to the same field emitters; ordinary
static fields retain their quoted names. `form_union_wiring.go` routes ordinary,
media and untagged map roots through this owner.

The model abstracts parsing, omission and physical storage. Normal tests cover
those boundaries: `TestFormMapFieldsUseCapturedCodec` checks captured application;
`TestFormMapFieldsGeneratedRoundTrip` compiles and exercises scalar/list/object,
JSON items, nested arrays, text/content unions, ordinary/untagged/media roots,
empty/literal keys, absence, malformed input, deterministic encoding and replay.
`TestFormRootMapEmptyKeyIntegration` retains the earlier scalar-map contract.
Source type absence without a usable default is rejected by the shared resolver,
not guessed from normalized shape (`TestFormRootMapElementAdmission`).

Map values store unions by value, unlike standalone sum branches. The physical
reference now lives on `AttributeContext.CollectionElementRef`, with its existing
implementation moved from the transform helper and used by transforms, defaults,
validation and map lowering. This is an ownership move, not a change to storage.
The declaration/reference matrix in `collection_declaration_identity_test.go`
and collection validation/transform regressions check the shared contract.

Bounded validation compared `FormMapFieldsDSL`, `UntaggedFormDSL`,
`UnionCollectionsDSL` and `PayloadFormBodyObjectDSL` at the exact pre-adapter
source and candidate, with all unrelated owners held constant. Of 140 raw and
rendered sections, eight changed: client/server encoder-decoder files for the
new map specimen and existing untagged form specimen. Differences replace only
map reflection paths with captured entry codecs and deterministic name order;
static fields, collection transforms, schemas and ordinary object controls are
byte-identical. All eight generated modules compile. Two independent candidate
processes produce identical complete output. Focused generated transport tests,
the collection reference/declaration/validation/transform matrix and scoped lint
pass. Imported map specimens additionally pass rendered libopenapi JSON/YAML and
the repository's pinned Vacuum contract rules.
