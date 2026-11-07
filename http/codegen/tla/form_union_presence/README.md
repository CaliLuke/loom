# Required members in form-union replay

The root form matcher counts schema membership and typed decoding independently.
Client transport structs can store a required scalar as a native value, so a
missing wire field decodes to the same zero as an explicitly supplied `count=0`.
Validation of that Go value cannot recover which input was supplied.

`native.cfg` models the initial generator: an empty required form matches the
Empty schema but both typed branches, rejecting a legal empty-body value.
`physical.cfg` uses the captured field's physical claims to enforce required
wire presence before typed branch decoding. It preserves `count=0`, admits an
empty required body through Empty, and retains optional zero-length absence.
Schema success never filters the typed domain. Body presence and field presence
remain distinct facts.

`form_union_match.go` applies this rule through `FormProjection.FieldClaims`.
`TestFormRootPresence`, `TestFormRootOutgoing`, and
`TestFormRootOptionalEmptyEncoding` in the maintained untagged-form fixture check
the generated implementation. The model assumes faithful physical claims and
scalar parsing. It does not prove raw parsing, JSON projection, code generation,
or arbitrary branch schemas. The runtime form-branch and untagged-presence models
cover the separate projection/closure and body-presence contracts.

Run explicitly; proof tools are not build or CI requirements:

```sh
for policy in native physical; do
  java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
    tlc2.TLC -workers 1 -metadir "/tmp/loom-form-union-presence-$policy" \
    -config "http/codegen/tla/form_union_presence/$policy.cfg" \
    http/codegen/tla/form_union_presence/FormUnionPresence.tla
done
```

The native control must fail `BodyAndMemberPresence`; physical must pass both
invariants. Remove temporary checker states after independent review.
