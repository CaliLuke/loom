# Client body presence before conversion

A transport conversion may dereference a root service pointer or inspect a
union discriminator. Presence must be decided before conversion, independently
of the target media. The old convert-first candidate panics for an absent
optional pointer (the original two-carrier model explored 11 states). The checked candidate preserves absence,
rejects required absence, and retains explicit null (the original two-carrier model explored 20 states).

```sh
java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -workers 1 \
  -config http/codegen/tla/body_presence/checked.cfg \
  -metadir /tmp/loom-client-presence http/codegen/tla/body_presence/ClientPresence.tla
```

Use `convert_first.cfg` for the expected `SafePresence` violation. This finite
model assumes a resolved pointer or explicit-presence carrier and a whole-body
payload without separately mapped bindings. `requestPayloadPresence` records
that physical carrier and the request's optional-body contract; the request
encoder evaluates it before conversion or field access. An absent envelope with
separate parameters or credentials is rejected. A member selected by `Body`
retains its field-presence guard, and JSON-RPC retains its envelope owner.

`TestClientRootPresenceBeforeConversion` in the generated multipart specimen
covers JSON/form/multipart object and union roots, required absence, and nullable
absent versus supplied null. Existing selected nullable/Any body integration
tests cover member guards outside this model. The model does not establish
conversion correctness, custom codecs, HTTP parsing, authentication or arbitrary
Go representations. It is an opt-in design tool, not a build or CI requirement.

Raw `loom.JSONValue` uses nil versus nonnil bytes, so its absence expression is a
nil check, not `Present()`. Opaque custom value codecs imply neither wrapper
methods nor automatic absence. `TestRequestPayloadPresenceUsesResolvedCarrier`
checks those physical distinctions; the generated Any-root control preserves
supplied JSON null. `TestRequestPayloadPresenceDoesNotInventBody` excludes the
`expr.Empty` marker so query-only payloads do not acquire a body requirement.

## Native union selection

The model now includes a nonnil native union with an empty kind. Its absence is
semantic: a pointer-only test admits it, then conversion produces a nil body.
`physical_only.cfg` reproduces that failure (27 distinct states); `checked.cfg`
checks root and selected-member policy for native union absence, selected null
and concrete values alongside pointers and explicit presence (32 states).
Selected null is supplied and remains distinct from an unselected union.

`requestValueAbsence` derives the observation shared by root and member body
planning. `requestMemberAbsence` reads canonical service field storage before
transport normalization. Required bodies reject absence; optional members omit
only the body while preserving query/header bindings; nil outer envelopes
cannot supply those bindings and are rejected. Templates consume the planned
predicate before body conversion for JSON, form and multipart encodings.

Direct coverage: `TestRequestPayloadPresenceUsesResolvedCarrier`,
`TestRequestMemberAbsenceUsesServiceField`, and
`TestOptionalUnionRequestBodyClientEncoder`. Generated coverage extends
`TestClientRootPresenceBeforeConversion` across JSON/form/multipart,
`TestSelectedUnionPresence` across selected required/optional members, and the
zero-field form root incoming/outgoing/invalid-selection cases. Nullable, raw
JSON and native selected-null controls remain separate from union absence.

The required `Any` member role is outside the model's absent-state set: as
specified in `docs/http-guide.md`, it has no absent state and a nil value encodes
as JSON null. `requestMemberAbsence` preserves that role before consulting the
carrier observer. Optional `Any` members and whole-body raw JSON retain their
separate absence contract. Direct required/optional Any assertions and the
existing `TestRequiredAnyBody`/`TestOptionalAnyBody` wire matrix check that
boundary; the model does not authorize changing it.
