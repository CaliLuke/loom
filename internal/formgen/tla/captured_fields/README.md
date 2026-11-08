# Captured form-field lowering

The shared codec resolver already decides JSON value versus item application
from the complete source shape. Generated lowering must retain that decision,
explicit nullability and the physical field's presence predicate. Reflection
cannot recover sourceType or authored nullability from Go values.

`CapturedField.tla` holds resolved policy constant while varying root/member,
value/item application, nullability and absent/null/value input. The prior
root-only JSON correction loses item application (31 states, exit 12) and member
null admission (42 states, exit 12). The captured candidate checks all 48 states.
An initial syntax error in the scratch model was corrected before these runs.

```sh
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -config field-default.cfg -metadir /tmp/loom-field-default CapturedField.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -config member-default.cfg -metadir /tmp/loom-member-default CapturedField.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -config field-captured.cfg -metadir /tmp/loom-field-captured CapturedField.tla
```

`internal/formgen.Field` owns leaf calls; `Presence` consumes allocated physical
contexts. Ordinary JSON fields, ordinary Content alternatives and tagged union
members share this lowering. HTTP declaration registries pass their actual
layout maps to union builders; service/view builders pass their own contexts.
The old whole-object reflection delegation in generated tagged form methods is
removed. Generic runtime reflection defaults are not changed.

`TestCapturedFieldCalls`, `TestCapturedFieldPresence`,
`TestDiscriminatorHintGeneratedWire`, `TestFormContentUnionUsesCapturedItems`
and `TestUntaggedFormGeneratedHTTP` check correspondence: nullable complex
members, complex-array roots/members, inner arrays as individual JSON items,
source-type-sensitive rejection, ordinary Content matching/ambiguity and hint
neutrality. Selected root tags prove empty repeated-array presence; object
members still require their own key evidence. Nil and nonnil empty arrays share
zero emitted items and are materialized as nonnil empty after selection.

The model abstracts allocated Go types, parser/error behavior and Content
candidate matching. Those remain implementation-test obligations. Source policy
is modeled separately in `internal/httpencoding/tla/field_codecs/SourceApplication.tla`;
selected presence in `codegen/uniongen/tla/form_selection`. Existing tagged-root
admission rejects nested untagged unions; the maintained negative control stays
in `TestTaggedFormPreservesNestedUntaggedRejection`. This is not a runtime bug
reproduction or an exporter-created restriction. Models remain opt-in design aids.

## Package-owned binding

The first extraction incorrectly reused a descriptor captured from the first
HTTP package. `ResultBodyUnionCustomKeysMultiDSL` compiled client fields as
Optional but server fields as pointers; server form emission called Present on
those pointers. This was introduced by the extraction, not a prior runtime bug.

`PackageBinding.tla` separates allocated name identity from physical package
binding and checks both client/server orders. `package-shared.cfg` reproduces
the mismatch; `package-detached.cfg` checks all 6 states. Run the same TLC command
above with the respective config and `PackageBinding.tla`.
`bindHTTPUnionDeclaration` publishes detached field descriptors while retaining allocated
names/type references and immutable JSON predicates. Physical JSON validation
bindings are also detached; the [union validation model](../../../../http/codegen/tla/union_validation_binding/README.md)
covers that extension and root-storage adaptation. It never mutates the shared
registry or a previously published package. `TestHTTPUnionFormBindingsBelongToPhysicalPackage`
checks both orders, names, pointer isolation and presence; the maintained custom
keys multi-result specimen checks actual compilation and independent regeneration.
