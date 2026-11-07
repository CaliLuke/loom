# Physical constraint facts and semantic default admission

Physical validator generation needs effective predicates and default presence.
A protobuf nested-union wrapper changes a branch into an object while its
containing occurrence retains an authored semantic default. Re-admitting that
raw default against the physical graph rejects a valid design.

`expr.EffectiveConstraintFactsFor` shares occurrence capture, inherited-layer
ordering and predicate derivation with strict semantic occurrences. It retains
physical required members, domain checks, enum declarations and local clauses.
Its default declaration/presence follows `effectiveDefaultLayer` without value
admission. `EffectiveConstraintsFor` and `ValueContext.NewOccurrence` retain
strict default admission, inherited-default conflict checks and materialization
contracts. Ordinary copies retain their own refinements; no origin fallback is
introduced. The facts result is never a semantic value occurrence.

Shared `codegen` validation consumes facts in both direct rendering and named
intrinsic/local predicate subtraction. `grpc/codegen.collectionContract` uses
the same query while retaining its raw default identity input. Scalar/metadata
lowering and the native default renderer retain their strict source queries.
Neither defaults nor physical wrapper predicates are removed.

The model separates authored admission, physical lowering and predicate query.
It varies local default validity, bounds and default presence. The current
policy re-admits raw defaults against changed physical shape; the split policy
uses the strict authored gate and derives physical predicates independently.

| Policy | Result | Distinct states |
| --- | --- | --- |
| current | `ValidDefaultsSurvive` counterexample | 32 |
| split | all four invariants pass | 32 |

Run explicitly with TLC1.7.4 (JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
java -Xms32m -Xmx128m -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-constraint-facts-current -config current.cfg ConstraintFactsSplit.tla
java -Xms32m -Xmx128m -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-constraint-facts-split -config split.cfg ConstraintFactsSplit.tla
```

The model assumes normal generation follows strict authored admission. It does
not prove graph capture, source precedence, predicate lowering or enum mapping.
Direct `TestEffectiveConstraintFacts*` controls cover aliases, local snapshots,
inherited conflicts, absent/false/zero/empty sources, detached identity,
refinement rejection and invalid predicates. The real
`TestPhysicalProjectionDefaultSemanticOwner` wraps an admitted union with the
actual protobuf helper and exercises shared validation; wrapper requiredness,
collection identity and named local predicates have dedicated controls.
The unchanged generated nested-union default specimen now compiles and passes
runtime checks for absent defaults, supplied zero values and invalid branches.
Ordinary HTTP/gRPC controls remain byte-identical and independent candidate
generations match byte for byte. These implementation checks establish the
correspondence that the bounded model assumes. A query of physical facts does not
certify arbitrary raw values against that physical representation.

Formal tools are opt-in design aids, never ordinary test or CI requirements.
Keep logs and checker state outside the repository; preserve concise results and
remove temporary state after validation and review.
