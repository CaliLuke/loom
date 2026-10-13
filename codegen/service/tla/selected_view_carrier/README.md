# Selected-view carrier operations (#834)

Selected fields and collection/root positions distinguish optional absence,
required presence, nullable null and concrete values. Ordinary pointers,
occurrence wrappers and intrinsically wrapped named declarations have distinct
physical call targets. Intrinsic named helpers accept their wrapper and skip
absent/null. An occurrence wrapper unwraps before calling an ordinary named
helper. A viewed record holds selection metadata independently of its payload.

| Configuration | Violated invariant or result | Generated/distinct states |
| --- | --- | --- |
| old | LegalOperations | 277/277 |
| guard-only | CallAgreement | 285/285 |
| null-as-missing | AdmissionAndValidation | 303/303 |
| unwrap-only | AdmissionAndValidation | 223/223 |
| projection-nil | ProjectionEntrySafe | 337/337 |
| envelope-wrap | EnvelopeIndependent | 417/417 |
| occurrence-drop | OccurrencePreserved | 337/337 |
| envelope-reuse | OccurrencePreserved | 266/266 |
| occurrence-call | ConversionCallAgreement | 341/341 |
| null-erasure | RoundTripPreserved | 349/349 |
| checked | All nine invariants pass | 432/432 |

Every rejected alternative exits 12; checked exits 0. The final model adds a
prior envelope carrier to expose reuse across ordinary/nullable method occurrences,
and records projected payload identity, canonical conversion arguments and
round-trip state. Both possible prior carrier orders are checked. The earlier
216-state readiness result covered admission, validation and envelope independence;
this extension checks 432 states. Required nullable root absence rejects both
at construction and at envelope validation; explicit null passes without child
predicates. Canonical helpers may still skip absent/null values.

Assumptions: named helper identity is resolved (#829), semantic projection retains
nullability (#837), and selected views exist. Ordinary root nil is rejected before
validation. Root selection is mandatory; nested fields may be excluded. A
collection position cannot be omitted: constructed absent elements reject,
explicit nullable null is allowed. The prior carrier abstracts cache reuse, not
concurrent operations. Round-trip state models invocation/adaptation, not codecs.
The model abstracts object-pointer versus collection-value argument layout;
direct tests and generated compilation check those representations. Unknown-view
selection is checked directly and through runtime acceptance.

Implementation correspondence:

- `viewValidationCarrier` binds the canonical named validator, presence tests,
  concrete access and call arguments. Fields and collection elements derive
  admission and invocation from it; intrinsic root bodies unwrap before selectors.
- `buildServiceMethods` retains method occurrence Nullable in the projected attribute
  and keys viewed metadata by declaration, view and physical nullability.
  `wrapProjected` preserves that occurrence in the envelope payload. Nullable
  occurrences use a separate envelope declaration, while projected declarations
  and canonical conversion helpers remain shared. `TestSelectedViewRootOccurrence`
  checks ordinary/nullable methods in both visit orders.
- `viewedResultProjectionCall` adapts occurrence wrappers around canonical helpers
  in both directions, preserving null and concrete states for objects and
  collections. Root constructors reject absence. Envelope validation independently
  rejects nullable root absence. NameScope derives all physical references.
  HTTP, WebSocket and JSON-RPC validation consumers use the envelope's recorded
  `Validate.Name`; SSE already did so.
- Projection entry nil guards apply only to physical pointers. Shared nullable
  conversion retains intrinsic absence/null. Diagnostic paths remain the caller's
  path instead of exposing temporary variables.

`TestSelectedViewNullableFieldCarrier`, `TestSelectedViewIntrinsicNullableCarrier`,
`TestSelectedViewRootCarrierBoundary` and `TestSelectedViewRootOccurrence` check
these seams. The four `TestSelectedViewCarrierCompile` fixtures run 620 service/
HTTP/JSON-RPC runtime cases and 28 helper/envelope boundary cases. They cover
ordinary, field-occurrence, intrinsic and method-root occurrence carriers;
default/tiny/nested selection; optional/required absence/null/concrete validity;
collection elements and nullable collection roots; unknown views; selected wire
values; client reconstruction and fault status/code. Full service tests and
existing optional/nested/physical-union runtime controls preserve adjacent owners.

Parent 0bc139d5e generates and compiles the ordinary fixture, emits compile errors
for field-occurrence/intrinsic carriers, and panics while generating method-root
occurrence. An ordinary Type control at the same remote-only pushed parent
compiles; adding views exposes the envelope loss. Current/repeat match all 120
artifacts exactly. All 30 ordinary artifacts are unchanged from parent. The
other parent-generatable specimens change only nine carrier-owning artifacts.
No parent artifacts are claimed for the root-occurrence generation panic; its
30 candidate artifacts are compared with the ordinary control and the independent
repeat. This establishes bounded correspondence, not a proof of all generated Go.

Run with Corretto 25.0.4.1 and TLC 1.7.4 JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
mkdir -p /tmp/loom-834-checked-java
java -Djava.io.tmpdir=/tmp/loom-834-checked-java -XX:+UseParallelGC \
  -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC \
  -config checked.cfg -metadir /tmp/loom-834-checked SelectedViewCarrier.tla
```

Repeat each negative configuration with separate Java temporary and metadata
directories. Concurrent TLC processes can otherwise race extracting library
modules. Syntax or tool errors are not accepted counterexamples. Preserve sources
and concise findings; remove checker/generated output after active dependencies
and independent reviews finish.
