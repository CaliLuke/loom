# Nullable selected-branch conversion

This bounded model supports the shared transform correction for #832. A union
branch carries a named object, array, map, nested union or primitive, and can be
ordinary, intrinsically Nullable or Nullable only at its occurrence. Nullable
values have absent/null/concrete states. These are conversion states, not wire
admission: a tagged JSON branch still requires its value property, and its
validator can reject an absent programmatically constructed carrier.

The current union branch path selects a named-object helper before dispatching
Nullable. `collectHelpers` strips the wrapper and declares a concrete helper;
`transformUnionCaseData` calls a wrapper-derived helper with the wrapper itself.
`old` models both mismatches. `names-only` models custom projection helper names,
which hide the name mismatch but still pass a wrapper to a pointer parameter.
`drop-null` rejects the tempting collapse of null into absence. `checked` routes
Nullable through the existing shared dispatcher before concrete helper admission.
Ordinary object helpers and other branch carriers are controls.

TLC checks call/declaration identity, physical carrier agreement, preservation
of the three presence states and completion across 35 independent cases. This is
a dispatch/ownership abstraction, not a proof of emitted Go. It assumes the
existing nullable dispatcher preserves state and concrete object conversion is
correct; direct tests verify emitted helper names and address-taking, and the
generated transport/projection acceptance verifies compile and runtime behavior.
It does not establish recursive transform termination, arbitrary custom codecs,
JSON admission, constraints, or view selection. Those remain separate owners.

## Run

Use Java 25.0.4.1 and TLC 1.7.4, JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
From this directory, use a fresh temporary directory for each configuration:

```sh
model_tmp=$(mktemp -d)
java -Djava.io.tmpdir="$model_tmp" -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
  tlc2.TLC -workers 1 -metadir "$model_tmp/states" \
  -config old.cfg UnionTransformCarrier.tla
```

Repeat with names-only.cfg, drop-null.cfg and checked.cfg. Expected evidence:
old fails CallDefinitionIdentity (36 generated / 36 distinct); names-only fails
CallCarrierAgreement (36 / 36); drop-null fails PresencePreserved (37 / 37).
Checked passes all invariants and termination (105 generated / 70 distinct).
Logs are temporary under `/tmp/loom-nullable-transform-model`; remove them after
validation and review. No checker/build output belongs in the repository.

Implementation correspondence: `codegen/go_transform_collections.go` owns union
branch admission and helper collection; `codegen/go_transform.go` owns attribute
dispatch; `codegen/go_transform_presence.go` owns nullable conversion and concrete
helper calls. `TestUnionTransformNullableBranchDispatch` covers all five shapes
under ordinary/intrinsic/occurrence layouts, while
`TestNullableObjectUnionTransformCarrier` preserves the parent HTTP compile red.

## Selected nil nested unions

`NestedNil.tla` models a non-nullable nested union extracted from a selected
outer branch. The source and target branch declarations independently choose
pointer or value storage. A source pointer can be nil even though the outer
union records that branch as selected; conversion must not call `Kind` on it.
The selected outer branch remains selected with an invalid zero target value
(nil for a pointer target, an unset union for a value target), so the existing
validator or wire replay can reject it. Conversion does not turn it into an
absent outer union or invent a concrete nested selection. Nullable wrappers
remain owned by the nullable dispatcher and the earlier model above.

`nested-old.cfg` reproduces the dereference before validation (`NoPanic`,
22 generated/distinct states). `nested-drop.cfg` rejects guarding the entire
outer selection (`OuterSelectionPreserved`, 22 states). `nested-checked.cfg`
passes all four safety invariants and termination over 42 generated / 28
distinct states. These runs use TLC 1.7.4 (same JAR above) and Java Corretto
25.0.5. Run the same command above with `-config nested-checked.cfg NestedNil.tla`
(and each negative control), keeping `-metadir` in a temporary directory.

The implementation seam is `transformUnionCaseData`: it uses
`AttributeContext.UnionBranchContext` for the extracted branch's declared
storage, declares the target branch value before a nil guard, and always lets
the outer renderer call `SetBranch`. The guard does not infer pointer storage
for general collection or Nullable values. `TestNestedUnionTransformPreservesSelectedNil`
compiles and executes all eight source/target pointer/value × declaration modes;
normal empty/nonempty strings and selected nil branches check correspondence.
`TestGoTransformOptionalUnionFieldsPreserveAbsence` and
`TestUnionTransformNullableBranchDispatch` keep their separate absence/null
contracts. The maintained HTTP `UntaggedFormDSL` ContentNested case verifies
that request conversion reaches ordinary rejection without a panic.

The model abstracts concrete inner conversions and their validation predicates;
it establishes selection retention and dereference safety, not JSON membership,
custom codec behavior, or arbitrary recursive transformation termination.

## Shared object helpers and required occurrences

`ObjectHelperPresence.tla` models the same named-object helper reached first from
either a required or optional occurrence. Both occurrences share its identity.
The source object uses a pointer even when primitive fields use value storage;
the primitive-pointer layout flag does not establish object presence. An invalid
programmatically supplied nil must remain nil through conversion so the existing
validation and wire admission owners can reject it. A supplied empty object must
remain supplied. Requiredness must not change the shared conversion or make it
depend on traversal order.

The old helper guard depended on the first occurrence's requiredness and the
primitive-pointer layout flag. `object-helper-old.cfg` reproduces nil becoming
an empty object (`PresencePreserved`, 37 generated/distinct states). A nonempty
object instead dereferences nil, reproduced by the compiled Go regression.
`object-helper-checked.cfg` passes presence, no-panic and requiredness-independence
invariants plus termination (72 generated / 48 distinct states).

Run the same explicit TLC command above with
`-config object-helper-checked.cfg ObjectHelperPresence.tla`, and repeat with
`object-helper-old.cfg` for the negative control. These runs use the same pinned
JAR and Java Corretto 25.0.5, with checker state kept outside the repository.

`generateHelper` in `codegen/go_transform_collections.go` derives the nil guard
from its resolved source parameter reference. Helper discovery no longer passes
occurrence requiredness. `TestObjectTransformHelpersPreserveNil` compiles and
executes helpers shared by required and optional fields in both traversal orders,
covering nil, supplied empty and populated objects. The form-root invalid-selection
regression covers a selected nil empty-object branch without fabricating a valid
empty form. The existing nested-union test still requires an unselected outer
union to convert to absence; this change does not alter that contract.

The model assumes named-object helper parameters/results are pointer carriers
and abstracts successful field conversion. It does not prove required-field
validation, custom codecs or transport absence policy. Those remain independent
implementation checks, and proof execution remains opt-in.
