# Captured form field identity

Form field identity comes from the same authored full-tag set that emits Go
struct tags. With no full tag set, the HTTP element name is the default. Any
full tag set replaces that default set: absent or empty form tags and `-` omit
the field. A JSON-name-only override does not replace the full tag set. Tag
options do not change outer field identity. Named body aliases keep this rule;
nested JSON-valued fields retain their own JSON names.

`encodingmeta.StructTags`, `FormFieldName` and `FormTagName` own those facts.
`expr.HTTPFormFieldCodecs` captures wire name, omission, codec and key claims
together. Transport IR retains the captured contract. Generated ordinary and
styled fields, sibling namespace reservations, representation target plans and
OpenAPI Encoding Objects consume it. Encoding metadata is authored using HTTP
element names and exported using effective wire names. Metadata remapping reads
a detached source so swapped names cannot overwrite each other's contract.

The finite model compares default, renamed, empty, skipped and JSON-only tag
sets against three sibling names. `element.cfg` reproduces the lost override;
`namespace-element.cfg` reproduces checking namespaces under obsolete element
names after runtime renaming. Both fail. `captured.cfg` passes wire and namespace
agreement: 45 generated states, 30 distinct, depth 2. It abstracts value codecs,
Go layouts, escaping and dynamic key languages; existing parameter namespace
models own those details.

Run explicitly with Java 25 and TLA+ 1.7.4, outside ordinary test/CI gates:

```sh
for mode in element namespace-element captured; do
  java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC \
    -workers 1 -metadir "/tmp/loom-form-tags-$mode" \
    -config "internal/encodingmeta/tla/form_tags/$mode.cfg" \
    internal/encodingmeta/tla/form_tags/FormTags.tla
done
```

The two controls exit 12; captured exits 0. Direct expression tests cover mapped
names, aliases, full/empty/skipped tags and collisions across JSON union, styled
union, ordinary sequence and deep-object codecs. Representation tests prove
form names do not leak into other media or nested JSON members. The maintained
`UntaggedFormDSL` exercises generated outgoing/incoming names, omission,
validation and renamed ordinary/styled/JSON-union fields. OpenAPI 3.1/3.2
rendering checks the same property and encoding names.
