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
