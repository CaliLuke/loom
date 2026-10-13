# Named validator eligibility follows physical field layout

Issue #819 exposes a missing correspondence obligation beneath the shared
[recursive validator](../recursive_validation/README.md) and
[callable identity](../validation_identity/README.md) models. Definitions and
calls already share names and carriers, but eligibility still used one caller
context throughout a generic attribute walk. A nested union changes its branch
value mode; a named object's registered field layout must remain independent.

The model has root, peer and leaf declarations. Peer holds Leaf through a
value-stored union branch. Leaf owns a required primitive field whose physical
storage either admits absence (pointer) or cannot be nil (value). An optional
back-edge explores recursive eligibility with finite visited-state traversal.
Every declaration emission order is explored. The old eligibility walk inherits
the caller mode; the candidate consults the named physical owner. CallClosure,
PredicateCoverage and NoFalseRequirement require complete nonempty definitions,
reachable required checks, and no invented primitive presence requirement.
FiniteRegistry and Terminates check the bounded declaration worklist.

Use Corretto Java 25.0.4.1, TLA+ tools 1.7.4 (TLC2 2.19), JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -workers 1 -metadir /tmp/loom-819-old -config Old.cfg ValidationLayout.tla
java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -workers 1 -metadir /tmp/loom-819-checked -config Checked.cfg ValidationLayout.tla
```

Old must fail a closure/coverage invariant; Checked must pass all invariants and
termination. Parser errors are not negative controls. Keep output outside the
repository. This is not a proof of arbitrary Go traversal or physical-map
construction. Named aliases, nullable carriers, arrays/maps, anonymous branches
and real wire validation require direct/generated checks.

Correspondence: `hasValidations` owns named predicate eligibility;
`presenceValueContext`, used by `NamedValidationFunction`, owns registered
physical mode. `renderUnionSumValidationCases` selects occurrence value storage;
that choice must not erase a named descendant's field layout. The existing
function collector emits only nonempty bodies; emitting empty stubs would hide
PredicateCoverage failures. Direct tests in `validation_named_layout_test.go`
cover physical/caller disagreement, nested union rendering, arrays/maps and
recursive reachability. `generator.TestUnionInUnionRoundTrips` checks HTTP,
JSON-RPC and gRPC generated compilation, valid round trips and invalid nested
HTTP/JSON-RPC responses plus direct constructed-value validators.


## Integration with physical union presence (#823)

The original checked model assumed only authored primitive presence predicates
could make a named body nonempty. After #823, a sum-union pointer branch also
requires a structural nil check even when its named value has no predicates.
The original model's closure claim is retained for its authored-predicate-only
subset; its broader correspondence now includes this additional fact at eligibility
and rendering together.

`PresenceCases` independently varies a structural pointer-presence guard owned
by Peer. `HasBody` retains that guard regardless of the eligibility policy;
`StructuralCoverage` requires the parent's call to reach that body. Primitive
requiredness remains controlled solely by the registered `physical` flag.
`NoFalseRequirement` applies when neither structural nor authored presence exists.
This preserves the distinction between branch pointer storage and the primitive
field storage inside its named value.

On integration parent `73451e4dad96837246d90fad56e1e492198e14b4`, the new direct
`TestNamedValidationEligibilityIncludesUnionPointerPresence` shows a nonempty
rendered body but false named eligibility. The frozen #819 required-layout
regressions remain red. Its generated invalid-response harness now compiles,
then exposes missing required-leaf validation as a conversion panic, rather than
failing at the previously missing helper: #823 supplies a presence-only body but
does not restore the named descendant's required-field layout.

With the pinned toolchain above:

| Configuration | Result | Generated / distinct states |
| --- | --- | --- |
| Old | CallClosure counterexample, exit 12 | 41 / 29 |
| NoPresence | StructuralCoverage counterexample, exit 12 | 41 / 29 |
| Checked | all invariants and termination pass, exit 0 | 104 / 64 |

Run `NoPresence.cfg` with the same TLC command as Old/Checked. The implementation
shares the physical union branch context and pointer-presence fact between
eligibility and rendering, then uses the existing named layout owner for
reachable predicates. This is not permission to equate pointer branch storage
with pointer storage of its child's primitive fields. The model abstracts Go
naming, Optional/Nullable codecs and source extraction; direct tests remain
responsible for shared attributes visited under two layouts and for no-op
inline value branches.

Implementation checks on the integrated candidate preserve #823's pointer guards,
presence-only cases, nullable ownership and concrete-value reset at object fields.
Full shared codegen, unchanged optional-union view/transport runtime controls and
physical-carrier regressions pass. The nested-union harness also checks nil and
non-nil unconstrained named values reached through a parent helper. All 77 bounded
artifacts match an independent repeat; only six nested-union HTTP/JSON-RPC files
change, adding reachable named-field predicates and structural-presence calls.
The recursive and named-nullable specimens are byte-identical to the parent.
