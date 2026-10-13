# Recursive validator graph (#709)

The pre-repair shared renderer stored an in-progress source buffer by named type
identity. Optional and Nullable unwrapping inlined that named body. A repeated
edge copied only the existing prefix; its `actual` value local binds to the
inner value while the copied diagnostic context still belongs to the outer
occurrence. Native pointers and array/map elements called named validators, giving
complete traversal but restarting each error path at the callee's root.

`ValidationGraph.tla` explores self and mutual recursion, both field orders for
both types, and a finite six-level input chain. `Legacy` recursively builds the
inline check records and returns the already-emitted prefix on an active edge.
It reproduces the field-order dependence: an early recursive field has no prefix;
a later recursive field copies the leaf predicate and its old diagnostic path.

The candidate separates a finite declaration worklist from evaluation on input
values. Each complete named declaration has its own predicate and a symbolic
edge to another declaration. `Evaluate` follows those edges and decrements the
remaining input depth; `named-static` resets the diagnostic context, while
`named-context` passes the occurrence path. No candidate copies another
renderer buffer or increases a generation-time recursion limit.

Run from this directory with Java 25 and TLA+ tools release 1.7.4, SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`
(the pinned JAR reports TLC 2.19):

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config legacy.cfg -metadir /tmp/loom-recursive-709-legacy ValidationGraph.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config legacy-path.cfg -metadir /tmp/loom-recursive-709-legacy-path ValidationGraph.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config named-static.cfg -metadir /tmp/loom-recursive-709-static ValidationGraph.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config checked.cfg -metadir /tmp/loom-recursive-709-checked ValidationGraph.tla
```

The negatives fail `Coverage` (17 states), `Paths` (19 states), and `Paths`
(17 states) respectively. The checked
configuration passes 28 states and requires finite/complete declarations, all six runtime occurrences
covered, occurrence-correct paths and eventual completion of the declaration
worklist under weak fairness. These are bounded claims, not a Go implementation
proof. The finite input bound is a model parameter choice, not a proposed runtime
limit. Arbitrary cyclic Go object values are outside the finite decoded-tree
contract and already recurse through native named calls; no new cyclic-value
acceptance or termination policy is proposed.

## Correspondence and production obligations

- The legacy interpreter models the removed mutable-prefix reuse. The shared
  `recurseValidationCode` now expands a complete root definition, and named
  occurrences call `validateNamedOccurrence` instead of returning cached text.
  The model gives the raw root object its predicate before following an edge.
- `validateOptionalAttribute` and `validateNullableAttribute` perform unwrapping;
  both use the same named-occurrence decision. `presenceValueContext` retains
  field-layout facts separately from `validationValue`, which identifies an
  unwrapped object value. The common `NamedValidationFunction` resolver retains
  the original named declaration carrier before unwrapping: intrinsic Nullable
  declarations receive the original wrapper; ordinary object declarations receive
  the concrete value address. Root nullable definitions expand their concrete
  value once.
- `validateNamedOccurrence` already partitions local predicates from intrinsic
  named predicates. Reuse this semantic owner, preserving occurrence constraints,
  validation prefixes, selected views and the effective-constraint provenance.
- Named declarations and calls must agree on actual physical representations.
  The model abstracts that agreement; typed Optional/Nullable, aliases, named
  nullable declarations and concrete carriers require Go compile/runtime tests.
- Model path depths abstract field suffixes. Real paths must preserve the existing
  `.field`, `[*]`, `[key]`, union and missing-field conventions, with no message
  string rewriting or substitution of Go variable names for authored names.
- `ValidationFunction` owns declarations, optional diagnostic path components,
  physical parameter references and established roots. `NamedValidationFunction`
  resolves the allocated name/prefix, parameter carrier and static mode for both
  declaration and occurrence owners. Its derived attribute
  context lowers authored field paths into runtime expressions. HTTP/JSON-RPC,
  service/views, authorization and both gRPC declaration collectors share it.
  Existing package registries deduplicate complete physical declarations;
  diagnostic paths never enter their keys.

`TestRecursiveValidationReadiness` checks the 30 self/mutual/named-indirection
and array/map combinations. `TestRecursiveValidationDiagnosticContext` checks
physical value calls, custom prefixes and runtime path propagation.
`TestRecursiveValidationGeneratedReadiness` compiles and executes eight HTTP
specimens with valid and invalid finite inputs at depths one through six, both
field orders, null/absence controls, safe problem responses and complete paths.
The gRPC specimen covers unary request/response, streaming receive, designed
error and viewed-result call forms under race detection. JSON-RPC preserves safe
public error messages and tests the internal decoder's paths. The authorization
specimen checks that deep invalid input reaches neither authorizer nor service.

The model abstracts Go typing and the concrete registry implementations. The
runtime/build matrix supplies that correspondence, including #796 selected-view
controls, aliases, named nullable collections and occurrence refinements. No
arbitrary cyclic Go-value termination policy or runtime depth limit is introduced.
Checker state, raw logs, parent archives and temporary modules stay outside the
repository and are removed after independent final review.

The first final review found that the initial implementation violated the model's
physical-call agreement assumption for a named object whose own declaration was
Nullable. Parent/candidate compile evidence reproduced that regression. The
correction retains intrinsic callable identity before unwrapping; occurrence-only
predicates still inspect the value and optional absence does not invoke a
required root validator. `TestNamedNullableValidationCallCarrier` checks carrier,
name/prefix, static layout and retained local requirements. Generated HTTP and
JSON-RPC controls compile and run named nullable object/array/map declarations at
required/optional root and field occurrences, including values, invalid values,
absence and null. These concrete typing obligations are not claimed as TLC proofs.

Follow-up #810 exposed a missing concrete obligation beneath the physical-call
agreement assumption: rebuilding a bare occurrence attribute discarded naming
metadata used by its declaration. The [physical identity model](../validation_identity/README.md)
reproduces that loss and distinguishes it from copying occurrence-only presence
into the intrinsic parameter carrier. The shared occurrence resolver now retains
physical metadata alongside the named type. Its focused shared/protobuf tests
and generated three-transport checks supply the concrete naming correspondence;
the recursion model above still does not prove metadata traversal or Go typing.
