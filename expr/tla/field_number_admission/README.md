# Emitted protobuf field-number admission (#703)

The model explores selection, union-number derivation, admission and publication ordering.
Fourteen designs use reduced field bounds 1..7, a reserved gap 4..5 and a 16-value
machine word. These reductions expose the same boundaries as protobuf's real
bounds and uint64 addition; Go tests use the actual values. A field records its
message owner, raw spelling, explicit number or inherited seed/offset, and whether
it belongs to the emitted message. Invalid syntax is represented by a negative
number; this model does not implement a decimal parser.

The current strategy compares raw identities and derives with machine wrap.
Bounds-only admission still misses equivalent spellings. Numeric admission after
wrapped arithmetic can accept an illegal mathematical slot as a small number.
Admission before selection rejects harmless metadata tags. Global identity
rejects reuse in distinct messages. The owned candidate selects emitted fields,
derives without wrap, then validates domain and numeric identity per message.
Explicit branch tags take precedence; unused seeds are not emitted fields.
The container strategy admits legal emitted fields but still parses an unused
malformed seed during publication, reproducing review finding 703-R1.

## Run

Use the pinned TLA+ 1.7.4 jar (TLC 2.19), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`,
with Corretto 25. From this directory:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-field-numbers-owned \
  -config owned.cfg FieldNumbers.tla
```

Repeat with each other configuration and a distinct temporary metadir.

| Strategy | Generated / distinct | Result |
| --- | --- | --- |
| current | 43 / 43 | AdmissionSafe: invalid field reaches generators |
| bounds-only | 50 / 50 | AdmissionSafe: equivalent numeric spellings collide |
| wrapped | 56 / 56 | AdmissionSafe: overflow can appear legal |
| early | 55 / 55 | ValidReuse: excluded metadata is incorrectly rejected |
| global | 49 / 49 | ValidReuse: independent messages cannot reuse numbers |
| container | 67 / 67 | PublishedWhenAccepted: unused malformed seed still fails in generation |
| owned | 70 / 70 | All four invariants pass for the bounded designs |

## Correspondence and limits

`GRPCEndpointExpr.validateMessageFieldTags` and `rpcMessageAttribute` select emitted
request, streaming, result and mapped error fields. `validateRPCMessageTags`
traverses reachable named/object/array/map/union shapes with cycle protection.
`AttributeExpr.UnionFieldTags` owns explicit branch precedence and inherited
consecutive numbers; protobuf rendering and generator field checks consume it.
`validateRPCObjectTags` is the shared per-message admission owner.
`writeProtoFieldShapes` is also a consumer: the original object traversal parsed
the non-emitted union container tag before visiting emitted branch numbers.
It now derives identity from emitted numbers, matching protobuf rendering and
field validation, without inspecting an unused seed as an emitted field.

The implementation admits numeric domain/identity at that owner and prevents
wrapping in shared union derivation. Message shape analysis records no container
number for oneofs; it records their emitted branch numbers instead. It does not rewrite authored
metadata, change field selection, rename fields or assign replacement numbers.
`TestRPCFieldNumberAdmission*` and `TestRPCUnionFieldNumberAdmission` check real
bounds, malformed text, equivalent spellings, explicit overrides and wrap.
`TestGRPCFieldNumberMessageRoles` checks actual evaluation across message roles,
recursive types and collection nesting. Selection controls preserve metadata,
opaque mappings, HTTP-only designs, unused types and separate message scopes.
`TestGenerateRejectsInvalidRPCFieldNumbers` requires actual CLI evaluation errors
with source locations before rendering or protoc.
`TestProtoFieldNumberIdentityIgnoresUnusedUnionSeed` checks equivalent shape
identity across unused numeric seeds and malformed/overflowing seeds, while
retaining distinct identity for different emitted numbers.
`TestGenerateRPCUnusedUnionSeeds` checks actual generation, protoc, Go build and
vet for fully explicit branches; numeric seeds are the parent control. Shape hash
equality is checked by Go tests, not by the model publication boolean.

The finite model checks ordering and identity within its stated assumptions.
It does not establish arbitrary graph traversal, integer parsing, generated-Go
compilation or protoc acceptance. Bounded generated artifact comparison and
compilation remain implementation gates. No concurrency or runtime codec changes
are proposed.
