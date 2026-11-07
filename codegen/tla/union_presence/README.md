# Union field presence

This opt-in model explores the field storage decision owned by
`expr.ServiceFieldLayout` and `expr.HasNativeJSONPresence`. It distinguishes an
absent field, an explicitly selected null branch, and an outer `Nullable` null.
Requiredness constrains absence independently of null acceptance. An untagged
union accepts null only when exactly one branch accepts it; zero or multiple
matching branches reject it. Explicit outer `Nullable` owns null before branch
selection.

The model abstracts concrete values, codecs, transport framing and protobuf
branch tags. A successful value match is assumed for `value`; JSON ambiguity,
validation and actual generated conversions are covered by implementation tests.
The model does not prove arbitrary branch predicates disjoint.

Run from this directory with the repository's pinned TLA+ tools (1.7.4):

```sh
java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -config native.cfg -metadir /tmp/loom-union-presence-native UnionPresence.tla
```

Substitute `wrapper.cfg`, `pointer.cfg`, or `optional.cfg` to reproduce the
rejected alternatives. Each violates `Contract`: a wrapper changes selected-null
into outer-null; a pointer changes optional null into absence; `Optional`
rejects an otherwise valid selected-null branch. The native policy checks all
72 distinct states without an invariant violation (108 states generated).
These are design aids, not default build, test, CI or release requirements.

Implementation correspondence:

- `expr/field_layout.go` selects native storage for null-admitting untagged
  unions and retains explicit outer nullable storage.
- `codegen/transformer.go` uses the same physical presence decision.
- Shared and protobuf transforms derive pointer decisions from field layout.
- `codegen/validation_recurse.go` observes an unset union kind as optional
  absence, preserving a selected-null branch as present.
- `codegen/union_presence_test.go` covers declarations and physical presence.
- HTTP, JSON-RPC and gRPC `untagged_null_fields_test.go` compile generated
  modules and exercise actual client/server exchanges, including required and
  optional fields, roots, collections, explicit outer nulls, and rejected
  ambiguous nulls where each transport supports them. Protobuf still rejects
  nullable branches and requires object wrappers around collection unions.

The former inferred `WrapNullable` path has been removed: semantic null
membership is not a request to add a physical outer wrapper. Regenerate service
and transport code together after this correction. No proof state or raw traces
belong in the repository.
