# Sum-union branch validation carrier (#823)

The selected branch of a generated sum union has a physical Go representation.
A nil object or nested-union pointer is a missing branch value. A nil slice or
map is a concrete empty collection: its length predicates must still run.
An explicit `loom.Nullable[T]` carrier owns absence, null and concrete values;
it is not a pointer merely because its concrete type is an object or union.

At parent `00fc9ba8bdebc8b77c42bd30f724707bbbea86a0`,
`renderUnionSumValidationCases` infers `RequiresValue` from a rendered
`if actual != nil` prefix. A nested union instead starts with a discriminator
switch, so its nil pointer reaches `Kind()` and panics. Named collection
validators start with the prefix too, causing valid nil collections to be
rejected as missing. Changing only that prefix decision would then let the
named-call nil guard skip collection predicates.

## Candidate and assumptions

`BranchCarrier.tla` compares the old text-derived decision, a pointer-guard-only
repair, and a carrier-sensitive invocation. The candidate guards only physical
pointer storage and runs collection predicates even on nil. Pointer presence
alone admits a validation case even when its value has no predicates. Nullable absence
fails, explicit null passes without value predicates, and present concrete
values retain their predicates. The discriminator is assumed to have selected
one known branch. Invalid/unselected discriminators, JSON decoding, allocator
correctness, custom Go codecs, recursive graph traversal and emitted Go typing
are outside this finite model.

The `old` mode abstracts the generated prefixes of the constrained named
object, nested union, list and map witnesses. It is not a claim about every
possible validator body. `guard-only` rejects an incomplete repair that derives
the right nil guard but retains the old named-call skip. The physical candidate
must resolve representation anew at each branch; inherited extracted-value
context cannot describe the selected branch. The finite model starts at that
resolved carrier. `predicates` independently varies pointer branches with and
without value predicates. Predicate-free branches have nil and valid values;
there is no invented invalid value predicate when none exists. The direct context matrix tests this extraction obligation.

## Run and observed readiness results

Use Corretto Java 25.0.4.1 and TLA+ tools release 1.7.4 (TLC 2.19), JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
From this directory, set `TLA_JAR` to that installed JAR and run each command:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config OldPointer.cfg -metadir /tmp/loom-823-model-OldPointer BranchCarrier.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config OldCollection.cfg -metadir /tmp/loom-823-model-OldCollection BranchCarrier.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config GuardOnly.cfg -metadir /tmp/loom-823-model-GuardOnly BranchCarrier.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config SkipEmpty.cfg -metadir /tmp/loom-823-model-SkipEmpty BranchCarrier.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config Physical.cfg -metadir /tmp/loom-823-model-Physical BranchCarrier.tla
```

| Configuration | Result | Generated / distinct states |
| --- | --- | --- |
| OldPointer | `NoPanic` violated, exit 12: nested nil reaches invocation | 66 / 66 |
| OldCollection | `Contract` violated, exit 12: nil list with minimum 0 rejected | 27 / 27 |
| GuardOnly | `Contract` violated, exit 12: nil list with minimum 1 accepted | 52 / 52 |
| SkipEmpty | `Contract` violated, exit 12: predicate-free nil pointer accepted | 70 / 70 |
| Physical | `NoPanic` and `Contract` pass, exit 0 | 84 / 84 |

The original three negative controls were reproduced during readiness. Final
review exposed missing case admission for predicate-free pointer branches; the
extended model reproduced that failure and all configurations were rerun before
the admission correction. The toolchain remains Go 1.27.2 and the pinned TLC
above; production source starts from the recorded parent. No syntax/tool failure substitutes for an
expected invariant counterexample. A bounded passing model does not establish
production implementation or generated-Go correctness.

## Correspondence and remaining obligations

- `NameScope.GoTypeRef` / `GoFullTypeRef` already determine storage for service,
  projected-service and HTTP union declarations. `AttributeScope.Ref` delegates
  to that same physical name owner. Branch validation should consume this
  representation, not parse emitted validation statements or reclassify storage
  separately for HTTP and JSON-RPC.
- `renderUnionSumValidationCases` owns the selected branch's validation context
  and `RequiresValue`. Admit a case when either pointer presence or value
  predicates need validation. Set the concrete-value fact independently from child
  field layout, retaining existing JSON presence and projected field modes.
- `validateNamedOccurrence` uses `validationValue` to distinguish a value call
  from a nil-guarded occurrence. A branch collection is a value even when nil;
  a branch object pointer must not inherit an outer extracted-value fact.
- `renderUnionSumValidation` owns the missing-value error before invocation.
  The existing nullable validation owner keeps its three-state semantics.
- `TestUnionValidationUsesPhysicalBranchCarrier` covers object, nested union,
  nullable nested union, named list/map and primitive in service, forced-pointer,
  JSON-presence and extracted-value contexts. At the parent, 11 of 24 cases fail:
  all nested contexts, object/extracted and list/map except extracted.
- `TestUnionBranchCarrierValidation` generates and vets actual service and HTTP
  code, then invokes the public response validator. At the parent, nested nil
  panics, nil zero-minimum list/map fail, and nil positive-minimum list/map report
  `missing_field` instead of `invalid_length`. Valid and invalid nonnil objects,
  nested and nullable branches, absent/null nullable values, nil/empty collection
  JSON round trips and non-nullable wire-null rejections remain controls.

The candidate now derives `RequiresValue` from `Scope.Ref` and resets
`validationValue` for each selected branch. All 28 direct contexts and the
generated carrier runtime pass. The empty-allowed collection controls use
`MaxLength(2)` rather than the readiness specimen's redundant `MinLength(0)`;
oversized collections are also rejected. This refines runtime coverage without
changing the model's lower-bound abstraction.

A bounded parent/candidate/independent-repeat comparison covers 78 artifacts
from three designs. Ten validator/CLI/view files differ only in pointer guards
and removal of collection nil guards; candidate/repeat bytes agree exactly.
All nine generated modules compile/vet. Candidate/repeat runtime controls pass;
the parent carrier runtime retains its intended failures. Generated-source lint
passes for all three candidate designs. The repair was delivered in
`73451e4dad96837246d90fad56e1e492198e14b4`. The related #819
contextual eligibility repair and #824 nullable-branch JSON decoder defect are
separate: this model establishes neither. Keep #824 wire-null admission out of
this ticket's success claims.

Keep these sources, configs and concise evidence. Retain task logs and generated
modules only while review or dependent #819 validation needs them; remove exact
task-owned output paths after the final active consumer finishes.


## Final-review admission correction

The earlier 72-state candidate checked only branches with value predicates.
Its production-correspondence claim was incomplete: `val == ""` still dropped
pointer presence checks. Independent review supplied a named empty-object
witness. Permanent direct tests reproduced zero emitted cases in all four
contexts; the generated public validator accepted nil while a nonnil empty
object passed. The broader admission claim was pending until this reproduction,
model refinement and correction were checked.

`SkipEmpty` models that incomplete candidate, not the original text-inference
bug. Its predicate-free pointer reaches `ok` instead of the required error.
The corrected case admission is `val != "" || requiresValue`. All 28 direct
cases and the 26 constructed-value cases pass after repair, including nil and
nonnil empty objects. Nullable and collection rules remain independently tested.
The enlarged physical model passes 84 states; its former narrower result is not
used as evidence for the new cases.
