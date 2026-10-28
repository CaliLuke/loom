# Untagged JSON matching

`UntaggedJSON.tla` models the orchestration in `pkg/json_union.go` and the
transactional adapters emitted by `codegen/unionjson`. It supplements the
[value-contract model](../../../expr/lean/value_projection/README.md): schema
membership and concrete Go decoding are independent predicates over one wire
value. Both must identify the same single branch; emission must retain the
selected branch, and failure must leave the destination unchanged.

Use TLC 1.7.4 (the repository's pinned TLA+ tools version), from this directory:

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config Legacy.cfg -metadir /tmp/loom-350-legacy UntaggedJSON.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config Strict.cfg -metadir /tmp/loom-350-strict UntaggedJSON.tla
```

The legacy control must exit 12 with an `EmissionRetainsIdentity` counterexample:
a selected decodable branch can emit without a unique matching schema. The strict
configuration must pass all three invariants. Checked with TLC 1.7.4 during #350:
legacy failed as intended (69 states generated); strict passed (128 distinct states).

The bounded model abstracts two branches as schema and decoder match sets and an
unexpected-failure flag. It includes zero/multiple matches, different unique
identities, and failures despite an otherwise valid match. Set membership makes
order irrelevant in the model; reversed candidate tests check implementation
ordering. The initial destination sentinel represents an arbitrary preexisting
value. Predicate extraction, JSON syntax, recursive constraints, numeric
representability and codec classification are assumptions, not proved here.

Correspondence checks:

- `pkg/json_union_test.go`: order reversal, independent signed/unsigned schema
  matches, unique-identity disagreement, and unexpected errors after success.
- `expr/value_plan_json_shape_test.go`: lowered matching agrees with the captured
  schema plan for inherited constraints, bytes, closed members and nulls.
- `http/codegen/untagged_arrays_test.go`: compiled encoding/decoding, nil/empty
  arrays, invalid nested elements, ambiguous arrays, unchanged destinations,
  both HTTP directions and required/optional request error contracts.
- `jsonrpc/codegen/untagged_arrays_test.go`: typed arrays and objects through the
  existing JSON-RPC envelope.

The model does not prove Go refinement, unlimited recursion, arbitrary codecs,
or full OpenAPI equivalence. Opaque codecs are rejected by DSL validation. Keep
checker state, traces and logs outside the repository and remove task-owned
outputs after validation and independent review.

## Nullable enum membership (#1037)

`NullableEnum.tla` models a record field's missing, null, enum-member and
non-member states. Requiredness and occurrence/inherited nullability vary
independently. `Nullable()` adds null to the value domain; enum predicates
apply to concrete values. This is distinct from Any's intrinsic admission of
null, which does not bypass authored enum membership.

Use the pinned TLC command above with `NullableEnum.tla` and the
`NullableLegacy.cfg`, `NullableLocalOnly.cfg` and `NullableChecked.cfg`
configurations, each with its own temporary state directory. Legacy reproduces
enum rejection of an admitted null. Local-only preserves direct nullability
but loses named-type nullability. Checked retains both and leaves required
absence and concrete non-member rejection intact.
With Java 25.0.5 and TLC 1.7.4, both negative controls fail `Agreement` after
42 distinct states; checked passes all 64 states. The implementation comparison
uses base `a2df44b82` and Go 1.27.3. The initial local-only repair failed the
inherited-null direct test before effective nullability was added to lowering.

`schemaJSON` uses the captured attribute's `IsNullable` fact after syntax/body
admission; `lowerJSONShape` puts that effective fact in the runtime descriptor.
`JSONShape.match` accepts admitted null before value predicates, while `Match`
still validates the full description first. `TestNullableEnumMatchingAgreement`
compares prepared and lowered matching with rendered schemas for primitive,
inherited, inherited-with-occurrence-refinement, conjoined and array enums.
Review exposed a separate rendering gap: an occurrence enum outside a nullable
named schema removed null. Inline JSON Schema and OpenAPI now wrap the complete
refinement in the nullable domain. Direct IR checks and the rendered 3.1/3.2
specimen verify null/open acceptance and closed/other/missing rejection.
The extended untagged-array fixtures check
ordinary required record validation, union marshal/unmarshal, HTTP and JSON-RPC
round trips, and invalid concrete values. Runtime tests retain ambiguous-null
rejection and fatal malformed-description errors.

The bounded before/after generation comparison uses the six designs in
`generationDeterminismDSLs` with identical fixture inputs on both revisions.
Intended differences are inherited `Nullable: true` descriptors in the service
and both HTTP union codecs, and null-preserving refinement wrappers in the
OpenAPI JSON/YAML specimen. The five control designs remain byte-identical.
The generated HTTP fixture compiles and runs; independent-process generation
checks exact bytes across five runs.

The model assumes the nullability query, JSON parser and enum evaluator; it does
not prove arbitrary alias graphs, Go codecs or JSON Schema rendering. Those
boundaries remain direct and generated-test obligations, and the original
union uniqueness/transaction model remains separate.
