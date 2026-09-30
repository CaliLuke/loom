# Request body analysis

For value source selection, representation phases and cache ownership, see the
child [value projection model](value_projection/README.md). Its bounded pipeline
checks complement the separate [Lean semantic foundation](../lean/value_projection/README.md);
neither establishes correctness of generated Go. The repository-wide
[formal model index](../../.agents/skills/loom-framework/references/formal-models.md)
maps other concerns to their local models and records maintenance/cleanup rules.

`RequestBodyAnalysis.tla` models two endpoints sharing an authored body shape.
The integer for each endpoint counts suffix applications to a nested user type.
Analysis may run zero, one, or two times before finalization; the checker explores
both endpoint orders. Finalization must apply exactly one transport suffix.
The old implementation stores the derived body during analysis, so one analysis
followed by finalization violates `RenamedOnce`. Pure analysis preserves the
source until finalization and satisfies the invariant in this bounded model.

Run with a downloaded TLA+ tools jar:

```sh
cd expr/tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -config RequestBodyAnalysis_before.cfg RequestBodyAnalysis.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -config RequestBodyAnalysis_after.cfg RequestBodyAnalysis.tla
```

The before configuration must report an invariant violation; after must pass.
This is an abstract lifecycle model, not a proof of Go graph cloning. The direct
`TestHTTPRequestBodyAnalysisPreservesSource` regression exercises actual array,
map, and inline-object graphs, repeated derivation, source pointer identity, and
names that already end in `RequestBody`. DSL and generated-module tests check
union branches, shared constructor allocation, presence validation, and compile.

## Explicit request-body member bindings

`BodyMemberBindings.tla` models two endpoints reusing one authored body whose
members map to distinct payload occurrences. Binding only the body root leaves
independently authored members unrelated to their payload members. Binding the
shared declaration instead makes finalization of the second endpoint change
the first endpoint's mapping. The checked rule binds each endpoint's owned
transport copy and leaves the authored declaration unchanged.

```sh
cd expr/tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -config BodyMemberBindings_before.cfg BodyMemberBindings.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -config BodyMemberBindings_shared.cfg BodyMemberBindings.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -config BodyMemberBindings_after.cfg BodyMemberBindings.tla
```

The before configuration violates `MemberCorrespondence` after one endpoint
finalizes. The shared-declaration control violates it after the second endpoint
finalizes. The after configuration passes all three invariants over four
distinct states and both finalization orders. The member mapping is an input
to this finite model; it does not prove Go graph copying or derive an HTTP
mapping from names or shapes.

`TestFinalizeExplicitHTTPRequestBodyBindsIndependentMembers` checks actual
endpoint copies, distinct payload bindings, element names, authored metadata,
constraints, examples and declaration identity. The selected-payload companion
checks the explicit source selection. Finalization owns these bindings;
`ValuePlan` must still reject unrelated targets. The checked-in
`EndpointBodyAsUserType` fixture covers the separately declared named body at
the generation boundary.

## Method type ownership

`MethodTypeOwnership.tla` models a named type used for both a payload and a
result, with one occurrence extended. Code generation emits one declaration per
Go type identity, in either traversal order. Reusing the original identity can
silently widen the plain occurrence or leave fields missing from the extended
occurrence. Giving the extended occurrence its own identity preserves both
shapes; description-only customization keeps the shared identity.

```sh
cd expr/tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -config MethodTypeOwnership_before.cfg MethodTypeOwnership.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -config MethodTypeOwnership_after.cfg MethodTypeOwnership.tla
```

The before configuration violates `EveryUseHasItsOwnShape`; the after
configuration checks all 16 states without error. The model abstracts the Go
identity and field set, not expression IDs or requiredness. The direct method
matrix additionally checks all four payload/result positions, both declaration
orders, direct types, alias chains, result types, inherited requiredness, and
description-only controls. Generated HTTP and WebSocket round trips check both
direct types and alias chains.

## Inheritance and default view finalization

`ResultViewFinalization.tla` checks when a result's automatic default view
captures its fields. Capturing before inheritance loses added fields during
projection even when the canonical Go type contains them. Merging inheritance
first preserves the automatic view's full shape and leaves explicit field
selection unchanged.

```sh
cd expr/tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -config ResultViewFinalization_before.cfg ResultViewFinalization.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -config ResultViewFinalization_after.cfg ResultViewFinalization.tla
```

The before configuration violates `ViewMatchesContract`; after checks all six
states without error. The model covers field selection and ordering, not Go
cloning or validation. Direct projection tests cover inherited and method-local
requiredness, while generated HTTP and WebSocket tests check the response value
survives the default view on the wire.
