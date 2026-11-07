# Form map root admission

`MediaRootMaps.tla` models the importer decision after schema normalization. A
form map has the same native root contract whether it is the only representation,
a branch of a media union, or nested in untagged alternatives. Multipart maps have
an ordinary single-media custom whole-body codec, but cannot enter generated
multipart alternatives. `depth` counts surviving untagged alternatives: singleton
wrappers normalize away and have depth zero.

The old `mapsAllowed` argument selected `false` for every media union and was
replaced with `!multipart` during recursion. Thus a direct multi-media form map
failed while a genuine union containing the same map passed. The checked policy
derives native map support from media kind and carries only the separate custom
multipart permission; that permission stops at a surviving union boundary.

Run explicitly from this directory (never a normal build or CI gate):

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 -config old.cfg -metadir "$TMPDIR/media-map-old" MediaRootMaps.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 -config checked.cfg -metadir "$TMPDIR/media-map-checked" MediaRootMaps.tla
```

Observed with TLC 1.7.4: `old` violates `AdmissionAgreement` for a direct form map
with multiple media; `checked` passes all three invariants (24 generated states,
12 distinct). The model assumes a supported scalar map value; it does not model
field parsing, null, or the value codec grammar. Implementation regression
`TestMediaRootMapAdmission` additionally checks normalized singleton wrappers.
`TestFormRootMapElementAdmission` checks that values use the shared
`httpencoding.Resolve(FormField, ..., default)` contract, matching
`expr.HTTPFormBodyEncoding`, rather than the unrelated URL-parameter `ScalarMap`
shape. Generated round-trip coverage lives in `TestFormRootMapMediaRoundTrip`;
`TestRenderPlansDynamicFormMapRequestBodies` preserves the custom multipart seam.
