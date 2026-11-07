# JSON form-field policy

A field's resolved codec owns null admission. JSON form fields retain their
Nullable or intrinsic-null union contract. Text grammars cannot represent null
and reject nullable designs. Absence remains independent: optional fields omit
an unset carrier; required fields reject it; explicit null is a present value.

`blanket.cfg` reproduces the old admission rejection (31 distinct states before
failure). `admission.cfg` admits JSON fields but retains the runtime's blanket
null rejection (33 states before failure). Both violate `Agreement`.
`captured.cfg` passes: 72 generated, 48 distinct states, depth 2. The model
covers two codecs, required/optional and nullable/nonnullable fields, and
absent/null/concrete input. Malformed JSON, Go storage and encoding details are
outside it and belong to the implementation tests. The initial unparenthesized
transition was corrected after a deadlock check; no deadlocked run is evidence.

Implementation correspondence:

- `expr.HTTPValueShape` preserves shape and null admission independently;
  `httpencoding.Resolve` admits nullable values only through JSON codecs.
- Form adapters consume captured field codecs instead of inferring all fields
  through reflection. `DecodeJSONField` receives explicit null policy.
- The field's physical presence owner supplies outgoing presence guards;
  native null unions use empty Kind for absence. See the adjacent codegen
  [union-presence model](../../../codegen/tla/union_presence/README.md).
- Media variants apply their own recorded physical layout before preparing
  fields and value plans. Parent JSON-union layout cannot overwrite a native
  form branch. `TestMediaFormFieldsUseBranchLayout` and the maintained styled
  media module check that boundary.
- `TestJSONFieldNullPolicy` checks absent, empty, whitespace, malformed,
  truncated, concrete, null and trailing JSON. `TestUntaggedFormGeneratedHTTP`
  compiles and mounts required/optional nullable and native-null union fields,
  checks problem responses and invocation counts, and checks outgoing wire.

Run with Java25 and TLA+ tools1.7.4, explicitly and outside CI:

```sh
for mode in blanket admission captured; do
  java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
    tlc2.TLC -workers 1 -metadir "/tmp/loom-json-field-$mode" \
    -config "http/tla/json_field_policy/$mode.cfg" \
    http/tla/json_field_policy/JSONFieldPolicy.tla
done
```

Negative controls exit12; captured exits0. Delete checker states after review;
retain sources and summarized results. This is bounded design evidence, not a
proof of arbitrary codecs or generated Go programs.

`EncodeJSONField` treats native nil collections as absence, but nonnil empty
maps and slices as supplied values in whole-value application. Empty item arrays
still produce no field occurrences. Custom marshalers on supplied empty maps
run normally. `TestJSONFieldCollectionPresence` checks that distinction; the
finite model above abstracts supplied empty values as `value` and does not
model reflection or marshaler execution. A former length-zero map shortcut
discarded supplied `{}` and bypassed custom output; the nil check replaces it.
