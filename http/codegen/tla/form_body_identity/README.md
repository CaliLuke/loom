# Authored form body identity

The body type owns form layout. Objects retain named fields even if their sole
field is a tagged or untagged union; actual root unions own flat branch layout.
The rejected alternative inferred a synthetic wrapper from field count and
silently flattened an authored object. No synthetic wrapper marker exists.

`Shape.cfg` reproduces that violation of `AuthoredShape`. `Kind.cfg` passes with
24 generated / 16 distinct states, depth 2. The model enumerates one/two fields
and tagged/untagged unions; it abstracts concrete codecs and Go types. It does
not prove wire decoding or compilation.

Implementation seams: `transportir.formFieldCodecs`,
`representation.normalizeParameterTarget`, and `buildUserRequestBodyTypeDetails`.
The synthetic TypeData metadata and generated wrapper methods were removed.
`TestSingleUnionFormFieldKeepsNamedCodec` checks resolved field ownership;
`TestUntaggedFormGeneratedHTTP` exercises named scalar/list/map/object fields,
JSON codecs and rejection cases. `TestFormRequestUnionOAuthIntegration` checks
flat tagged root wire behavior and generated service/client compilation.

Run explicitly with Java 25 and TLA+ tools 1.7.4; these are design checks, not CI:

```sh
for mode in Shape Kind; do
  java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC \
    -workers 1 -metadir "/tmp/loom-form-body-$mode" \
    -config "http/codegen/tla/form_body_identity/$mode.cfg" \
    http/codegen/tla/form_body_identity/FormBodyIdentity.tla
done
```

Shape exits 12 with the invariant violation; Kind exits 0. Remove temporary
checker directories after inspection. The migration is to author flat forms as
root unions and regenerate, rather than retain implicit shape-based behavior.

## Mixed content field declarations

`ContentCarrier.tla` models the captured codec tree independently of authored
body shape. A JSON leaf needs JSON field/element presence whether it is the
entire field or a branch of a mixed Content union. A shallow Style-only check
skips those branch leaves. `ContentShallow.cfg` reproduces that mismatch and
fails `LeafOwnsLayout` in eight states; `ContentRecursive.cfg` passes eight
states. Run the same explicit TLC command with this module and either config.
An initial ungrouped transition excluded cases rather than assigning false;
that run is not evidence. The recorded results use the grouped assignment.

The model assumes separate private declarations for distinct physical roles;
the source-node identity model under `expr/tla/type_copy` and direct tests cover
that separate ownership obligation. It does not prove generated Go layouts or
schema matching. `walkFormFieldCarriers` consumes captured branches, allocates
private boundaries before descending, and records JSON layout only at JSON
leaves. A named JSON item array uses JSON element presence while its enclosing
form field or mixed union keeps native presence. No new collection-layout mode
is needed. `TestJSONFormContentBranchCarriers` checks source isolation, retained
item requirements and the recorded mixed/leaf layouts; generated Content-field
tests check missing item fields, nullable items and empty array presence.

## Multipart physical roles

The same `bodyCarrierPlan` projects captured URL-form codecs and multipart
`PartKind`/`StyleCodec`/ordered branches into JSON or native storage roles.
`Multipart` is not inferred from a media label. Both directions of ordinary
requests and media mappings consume this projection before naming declarations
and recording layout. Raw byte and text items remain native; JSON items retain
JSON presence. Mixed unions recursively own separate branch declarations.

`CarrierRole.tla` models the captured field role versus ambient JSON layout.
`RoleAmbient.cfg` is the rejected ambient-wins alternative; `RoleCaptured.cfg`
passes 20 generated / 16 distinct states, depth 3. Run it with the same opt-in
TLC command, choosing the corresponding module and configuration. The role
model abstracts MIME application; the resolver owns Value/Items and PartKind.
`TestJSONFormContentBranchCarriers` now exercises both URL forms and multipart,
and `TestMultipartFieldCarriersKeepNativeTextItems` checks ordinary native array
isolation. Generated multipart root/media and imported source-array fixtures
check compilation and decoding beyond the model.
