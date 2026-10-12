# Union branch validation context

Validation context model for #796. A scalar field in a projected object is stored as a
pointer regardless of requiredness or defaults. Ordinary service fields use
values when required/defaulted and pointers otherwise. Validation must match the
declared storage and guard every pointer read, then retain the same required and
content constraints.

`BranchContext.tla` enumerates ordinary/projected, required/optional,
default/no-default, supplied/absent and valid/invalid states. Value storage has no
absent state; those combinations are excluded. The predicate abstracts an already
decoded field, not default materialization or raw JSON schema selection.

- `legacy` chooses representation from required/default alone. It fails
  `DeclarationAgreement` in 41 states (including an optional defaulted view field).
- `dereference-only` follows physical storage but guards only optional fields.
  It fails `NoAbsentRead` in 45 states for an absent required projected field.
- `context` follows physical storage and guards pointer reads. It passes all
  three invariants in 52 states. Content validation and requiredness remain
  separate from the guard; an absent required field is rejected without a read.

Run with the repository's pinned TLA+ tools release 1.7.4, SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
This JAR reports TLC 2.19. Java 25 was used:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config legacy.cfg -metadir /tmp/loom-union-context-legacy BranchContext.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config dereference-only.cfg -metadir /tmp/loom-union-context-deref BranchContext.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config checked.cfg -metadir /tmp/loom-union-context-checked BranchContext.tla
```

The two negatives must fail their named invariants, not parsing/evaluation or
deadlock checks. The initial scratch model had an unparenthesized disjunction
which made assignment nondeterministic and failed `Contract` in all modes; that
run is invalid evidence. The corrected source explicitly groups the assignment
and the value-storage admission implication. Only the second runs are evidence.

## Correspondence and limits

`buildProjectedType` declares projected field storage;
`projectedTypeContext` and `buildValidations` already own its validation context.
`buildViewUnionTypeData` now passes that projected context and view mode to
`unionBranchValidationCode`; the ordinary builder passes its ordinary context.
The shared helper delegates every predicate to `codegen.ValidationCode`.
`TestProjectedUnionValidationContext` checks range/pattern/enum/length storage,
required guards and ordinary/optional controls. `TestProjectedUnionValidationGenerated`
checks actual declarations, compilation and decoding together, including defaults,
nested objects, arrays and selected result views. The repair adds no per-constraint
rendering and does not replace authored branch validation with a default-view
validator. The selected result branch test explicitly distinguishes these contracts.

The model does not prove Go compilation, field naming, recursive traversal,
schema membership, JSON matching, custom carriers, selected views, or runtime
default behavior. The existing `codegen/tla/projected_presence` model establishes
a different conversion boundary and is unchanged. It does not prove union branch
validation either. Generated compilation and runtime acceptance/rejection tests
must bind the implementation to this context model.

`optional_recursive_probe.go.txt` records the separate #709 witness. Copy it
temporarily to `codegen/optional_recursive_readiness_test.go`, then run
`go test ./codegen -run '^TestOptionalRecursiveValidationReadiness$' -count=1 -v`.
Both field orders pass with native pointer presence and fail with JSON Optional
presence: the latter emits partial inline recursion without `ValidateNode(actual)`.
This is traversal/recursion ownership, not the physical branch-context selection
modeled here. Do not include a #709 production change in #796 based on this probe.

Keep only model/config/test sources and concise findings in the repository.
Checker state, raw logs and generated probe modules remain temporary until review.
