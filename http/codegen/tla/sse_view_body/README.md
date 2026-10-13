# Keep selected SSE views through wire-body conversion

Issue #727: dynamic HTTP SSE projection selected the right view, but analysis
retained only the first HTTP body constructor. A full-view event consequently
lost fields at wire conversion. Viewed collections had the same defect.
Fixed-view and discriminator-selected paths already chose the right body.

`SSEData.ResponseBodies` retains the complete eligible body family.
`writeServerStreamBodySelection` owns body selection for ordinary SSE,
discriminator SSE, and HTTP WebSocket sends. Dynamic selection uses the
validated result's `res.View`; fixed selection emits its one constructor.
Unary HTTP and JSON-RPC already use the validated viewed result's View.
There is no new protocol marker or client view negotiation.

A mapped SSE member can itself be omitted by the selected view. Its presence
comes from the selected body's finalized `Value.Plan`, via
`sseBodyHasDataField`. When absent, `MappedSSEEncoding` determines the existing
empty representation: raw text/bytes have empty data; JSON has null. No
constructor can resurrect a hidden required/defaulted member. When present,
the renderer extracts the mapped field before erasing the body's concrete Go
type. Ordinary primitive, unconverted collection, and mixed-result paths retain
their direct encoding.

## Bounded design model

`ViewBody.tla` explores two successful sends with any intervening rejected
attempts, in dynamic, fixed, or discriminator selection modes. Dynamic empty
selection means default; fixed selection ignores runtime requests;
discriminators map legacy/updated to default/full. Nil values and unknown views
reject before a write. The full view adds one field; tiny omits the mapped
member. Mapping is whole-event, raw text, or JSON. As in the DSL, discriminator
selection cannot combine with mapped data.

- `FirstBody.cfg` models the original loss of the body family in dynamic SSE.
- `MissingMappedField.cfg` models selecting the right body but still accessing
  a mapped Go member that the view removed. This exposed an implementation
  constraint before the selected-body repair was finalized.
- `SelectedBody.cfg` derives the body from the validated projection and handles
  mapped omission through the existing codec's empty representation.

`SelectedBody` requires transmitted whole-event fields to equal the selected
view. `WellFormedBodyAccess` prohibits reading an omitted Go member;
`MappedData` checks both included values and codec-specific empty data.
`NoRejectedWrite` checks rejected attempts write nothing. No liveness claim is
made, and SetView/sends are assumed sequential.

Use TLC 2.19 revision `5a47802` (TLA+ tools 1.7.4), JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Validated with Corretto Java 25.0.4.1. From this directory:

```sh
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-727-first -config FirstBody.cfg ViewBody.tla
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-727-mapped -config MissingMappedField.cfg ViewBody.tla
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-727-selected -config SelectedBody.cfg ViewBody.tla
```

FirstBody exits 12 with the expected full-view omission (277 generated,
276 distinct states). MissingMappedField exits 12 on `WellFormedBodyAccess`
(265/264 states). SelectedBody exits 0 (3976/923 states, depth 8).
Parser errors do not count as negative controls. Keep checker output outside
the repository; remove unused task artifacts after validation and review.

## Implementation correspondence and limits

- `Choose` corresponds to sequential SetView, fixed DSL view, or discriminator
  selection before a send.
- `Project` corresponds to `writeSSEResultSetup` and shared service
  `NewViewed…` validation/projection.
- `Encode` corresponds to the retained body family,
  `writeServerStreamBodySelection`, `sseBodyHasDataField`, and
  `writeSSEPayloadSetup` using `MappedSSEEncoding`.
- `TestSSEViewedResultBodySelection` checks object, collection, mapped-data and
  fixed-view rendered selection. `TestSSEViewedResultGeneratedIntegration`
  compiles/vets generated Go and checks changing views through raw frames and
  the generated client, collection full-view fields, mapped optional/required/
  defaulted omission, fixed full, and unknown/nil rejection before stream commit.
- `TestSSEProjectionGeneratedModuleCompiles` checks discriminator precedence over
  SetView, server wire fields, generated-client projection, and unknown
  discriminator rejection. `TestWriteServerWebSocketResponseBodySwitchesViews`
  checks the shared WebSocket renderer. Existing viewed-WebSocket integration and six
  streaming expected-output sections cover the other shared-renderer consumer.

The model abstracts values, constructors and serialization into field sets and
codec classes. It assumes projection and validation follow the service-view
contract; it does not prove Go compilation, recursive shapes, nullable carriers,
actual framing, concurrent SetView, mixed-result selection, or custom codecs.
Focused generated-Go, mapped-data, primitive/collection, mixed-result, and
checked-in SSE fixture tests cover relevant boundaries separately.
