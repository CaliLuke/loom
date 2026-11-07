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

## Server object absence through construction

`ServerObjectPresence.tla` models body observation, member validation, separately
bound input validation, payload construction and invocation. An absent optional
native object stays absent; a supplied empty multipart envelope is a value and
must satisfy its own member requirements. The body and the enclosing payload
are separate: selected bodies and declared non-body fields require an envelope,
including optional bound fields whose input is absent. Body conversion/defaults
run only for supplied bodies; non-body assignments retain their own semantics.

The model consumes the codec owner's observation. Required form zero bytes are
supplied empty input; optional form raw zero bytes are absence. Built-in JSON
whitespace yields EOF/absence; `WithNonNullableBody` rejects literal null before
carrier mutation. Malformed/truncated inputs are decode errors. Nullable values,
raw/custom codecs, union selection and arbitrary transformation correctness are
outside this native-object abstraction. Concrete generated tests must establish
these assumptions, HTTP problem details and actual payload values.

The current value-carrier mode reproduces both false missing-member rejection
and invented empty payloads. Validation-only repair still invents a payload;
pointer-only repair dereferences nil; early return skips non-body validation or
assignment. The owned carrier/construction candidate checks all six invariants
across 512 distinct states. Six negative configurations each fail their named
invariant. Initial checker syntax errors are excluded from design evidence.

```sh
java -Xmx256m -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
  tlc2.TLC -workers 1 -config http/codegen/tla/body_presence/server-owned.cfg \
  -metadir /tmp/loom-server-object-presence \
  http/codegen/tla/body_presence/ServerObjectPresence.tla
```

Replace the configuration with `server-value-validation.cfg`,
`server-value-materialization.cfg`, `server-validate-only.cfg`,
`server-pointer-only.cfg`, `server-early-return-validation.cfg` or
`server-early-return-binding.cfg` for the respective counterexample. Models stay
opt-in and outside ordinary build/test gates.

Implementation correspondence: `isOptionalObjectBody` owns the nil-capable
server carrier independently of selected-member client identity. Its domain is an
actual native object body: the framework `expr.Empty` no-body sentinel is
excluded even though it is object-shaped. `TestOptionalObjectCarrierBoundaries`
checks that boundary; bodyless endpoints retain their ordinary bound-input
construction without an invented body argument.
`payloadBuilder` plans body-only conversion separately from outer-envelope
allocation; the shared constructor owner performs conversion only when supplied.
The existing JSON/form/multipart decoders retain the observation, and the common
request decoder still validates all separately bound inputs before construction.
`TestOptionalRootObjectBodyPresence` checks planning and rendered control flow.
In the module generated by `TestUntaggedMultipartGeneratedHTTP`,
`TestOptionalObjectSourcePresence` checks actual multipart, JSON and form
absence, supplied empty input, member constraints, bound inputs, credentials and
service invocation; in the module generated by `TestUntaggedFormGeneratedHTTP`,
`TestOrdinaryFormPresence` checks optional and selected form absence against
supplied empty and populated input.
