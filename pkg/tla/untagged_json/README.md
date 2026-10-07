# Untagged JSON matching

`UntaggedJSON.tla` models the orchestration in `pkg/json_union.go` and the
transactional adapters emitted by `internal/unionjson`. It supplements the
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
