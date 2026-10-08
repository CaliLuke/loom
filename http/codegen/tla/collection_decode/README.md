# Reject null collection elements before conversion

This model records the shared representation repair for #617. JSON decoding
must enforce element nullability before a conversion can discard presence.
Previously both nullable and non-nullable elements used `loom.Nullable` on the
wire. Full body validation could reject nulls, but viewed responses bypassed
that validation so smaller views could omit required fields. Conversion then
turned null array elements into zero values and omitted null map entries.

## Decision and contract

Use the existing `loom.Optional` carrier for non-nullable decoded collection
elements. Its JSON decoder rejects null. Keep `loom.Nullable` for explicitly
nullable typed elements; unconstrained `Any` keeps its JSON-value representation.
Both presence carriers expose `Value`, so conversion and constraint validation
share their existing paths. No new runtime type or view-specific null scanner
is needed. The element cannot be absent in JSON; Optional's absent state remains
useful for zero-initialized Go storage and custom decoder validation.

Alternatives were preserving wrappers throughout service view representations,
or adding a second body validator for every view. Both spread a JSON decoding
obligation into semantic projection. The selected representation enforces the
same type rule for ordinary and viewed responses, HTTP/WebSocket, and requests.

Accepted examples include empty/populated collections and `[null]` when elements
are explicitly nullable. `[null]` for `ArrayOf(String)` and `{"x":null}` for
`MapOf(String, Int)` fail during decoding, even in a known wire-type field
excluded from the selected view, just as a wrong JSON kind does. Fields absent
from the wire type follow the codec's unknown-field policy. Missing fields
still belong to selected-view validation. Valid wire values are unchanged.

This intentionally classifies invalid null elements as decoding failures:
HTTP clients return `decoding_error` with the JSON semantic cause, WebSocket
receivers preserve that cause, and HTTP requests return 400 `decode_payload`
with the standard safe detail `invalid request body`. JSON-RPC params use
`-32602` with `decode_payload` in the error data. The previous full-body
validator reported `invalid_field_type`; viewed clients incorrectly succeeded.
No fallback accepts the formerly invalid values. Custom codecs must enforce the
same non-null contract; this model assumes the built-in JSON codec.

## Model and reproduction

The bounded model considers a known wire-type field. It has array/map shape, absent/empty/concrete/null-element input,
element nullability, selected/omitted fields, requiredness, and known/unknown
views. Decode, conversion and selected-view validation are separate steps.
`NoErasure` requires every successful conversion to preserve its wire value;
`Contract` checks acceptance and weak fairness checks termination.

Use TLC 2.19 revision `5a47802` (TLA+ tools 1.7.4), JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Validated with Corretto Java 25.0.4.1. From this directory:

```sh
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-elements-legacy -config Legacy.cfg ElementDecode.tla
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-elements-checked -config Checked.cfg ElementDecode.tla
```

Legacy must exit 12 with a `NoErasure` counterexample; Checked must exit 0
(512 distinct states). Parser/evaluation errors are not successful controls.
Keep checker state and traces outside the repository and remove task output
after validation and independent review.

## Implementation correspondence and limits

`http/codegen/typedef.go` owns array/map decoded element representations;
`pkg/optional.go` owns rejection of JSON null. `codegen/go_transform_collections.go`
extracts concrete carrier values only after successful decoding. The shared
representation also serves JSON-RPC generated HTTP body types.

`TestResultViewPresenceGeneratedIntegration` exercises compiled HTTP and
WebSocket clients, selected/omitted fields, missing values and invalid nulls.
`TestNullableCollectionResponsesGenerated` checks nested and explicitly nullable
collections. Request-body matrix tests retain required/optional and
empty/whitespace/malformed/truncated/valid cases plus status, problem code/detail,
service invocation and payload state checks. `TestJSONRPCOptionalValueParamsGeneratedModule`
checks non-null arrays and maps through the shared JSON-RPC params boundary.

The model abstracts concrete values and assumes successful scalar decoding and
presence-preserving conversion. It does not prove Go generation, recursive
schemas, arbitrary custom codecs, JSON error pointers, or view content
constraints. Runtime/generation tests establish those covered cases separately.
