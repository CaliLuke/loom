# Query/form serialization ownership

Model for the **user-adopted uniform JSON contract**. The earlier
bracket-preserving checked model and its blanket-deepObject candidate are
superseded, not evidence for the new runtime.

Each finite state is one named OpenAPI 3.2 query parameter or form property.
Shapes distinguish scalar, scalar array, scalar map, fixed object and nested
map. Named maps and form objects use JSON at every depth. Fixed object query
admission stays unsupported. Default form encoding for composites normalizes to
JSON; an explicitly authored form style instead normalizes to flattened data.
Explicit JSON media is equivalent for supported composites. Scalar and repeated
array spelling remains unchanged. Bracket sources are rejected as mismatches.

`Target` is the accepted contract. `LegacyRuntime` is the parent bracket codec.
`Plan` models runtime client/server selection and OpenAPI/import choices:

| Configuration | Purpose and observed result |
| --- | --- |
| legacy | Parent codecs violate adopted UniformCompositeMode; 113 states. |
| legacy-import | Independent admission accepts incompatible default query maps; ImportPreservesWire fails, 113 states. |
| legacy-export | Export metadata disagrees with runtime; ExportPreservesWire fails, 113 states. |
| metadata-only | Updating docs/import while keeping brackets fails ExportPreservesWire, 113 states. |
| client-only | Updating one codec side fails ClientServerAgree, 113 states. |
| native-json | JSON mode with native physical carriers violates JSONCarrierRejectsNull; 113 states. |
| checked | One selection and JSON physical carriers drive all consumers; six invariants pass, 160 states (80 initial). |

Run from this directory using Java25, tla2tools release1.7.4 (TLC2.19), SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 \
  -config checked.cfg -metadir /tmp/loom-uniform-json-checked Serialization.tla
```

Repeat with each named config and a separate metadir. CHECK_DEADLOCK is false
because a planned value is terminal. Temporary logs use
`/tmp/loom-uniform-json-model-*.log`. Remove task-owned logs/states after final
validation and review; retain this source and summarized results.

The finite model excludes JSON parsing/escaping, actual code generation,
recursive schemas, root-form layout, MapParams, aliases, unions, byte/custom
codecs, earlier specification versions and M7 presence/default semantics. A
single nested category checks uniform mode selection, not arbitrary recursive
JSON correctness. No liveness/concurrency or arbitrary Go refinement is proved.
Concrete runtime/generated/module/OpenAPI tests own those boundaries.

Implementation correspondence: query mode selection and templates,
transportir.Parameter/Request, http/form.go named-versus-root boundary, OpenAPI
IR parameter/media builders, and importer parameter/request-content admission.
All derive their mode from the dependency-neutral `internal/httpencoding`
`Select`/`Match` owner. ValueTarget and existing JSON codecs retain actual
physical value, presence and custom serialization ownership. Do not turn this
small selection owner into a second value graph.

Implementation checks: `http/uniform_json_readiness_test.go` checks reflected
form and JSON-field codecs, custom JSON methods, JSON versus form tags, escaped
keys, numeric keys and malformed values.
`http/codegen/deep_object_readiness_test.go` compiles generated transports and
checks independent client/server wire values plus mounted-handler status, safe
problem details and non-invocation on malformed input.
`http/codegen/json_field_alias_test.go` covers named numeric maps and validation.
The OpenAPI and importer `deep_object_readiness_test.go` files check supported
3.1/3.2 contracts, schema references, admission, and imported generated wire.
The representation codec regression in `prepare_test.go` checks that named JSON
query values consume JSON value projections, while scalar bytes retain text.

URL attributes remain non-nullable under expression validation. JSON-field
decoding reuses `decodeNonNullableJSON`; malformed, trailing and null values
cannot become absence. Empty outer maps retain the existing omission rule;
M7 presence/default semantics remain outside this model and repair.

Final review exposed a missing correspondence obligation: selecting JSON text
does not itself select a null-rejecting physical carrier. The native-json
configuration preserves client/server/export mode agreement but fails the new
JSONCarrierRejectsNull invariant. The checked configuration explicitly requires
the existing JSON presence carrier for builtin composite boundaries. This is a
finite carrier-selection obligation, not a proof of JSON parser correctness.
Opaque custom codecs own their representation and are outside that obligation.

`http/codegen/json_field_carriers.go` names builtin form JSON subtree declarations
and registers their JSON layouts while the enclosing form stays native; query
fields decode through the existing JSON type renderer. `codegen.GoTransform`
consumes registered layouts for named collections as well as objects.
`codegen/json_boundary_transform_test.go` is red on the parent and green on the
implementation for map/array fields beneath a native root. Generated mounted
checks cover null map values, nested map leaves, nested array elements and form
object fields; the root-form child control covers its conversion and empty key.
