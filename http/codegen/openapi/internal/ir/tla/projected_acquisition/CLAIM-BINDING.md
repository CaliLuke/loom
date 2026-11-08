# Prepared binding on suffixed canonical claims

`ClaimBinding.tla` refines the selected-binding premise of the acquisition models. The first Go implementation treated canonical metadata on a fresh wire variant as an unrestricted claim association. `TestParameterPropertyNames` showed that equal-F query and form occurrences then acquired whichever prepared child ExampleSet was encountered first. Their individual prepared values were stable; the selection changed. A suffixed association must retain its captured prepared target and example-plan position. Actual public canonical-reference reuse and validated canonical Body selection remain separate contracts.

Two candidate encounter orders each check five invariants and weak-fair termination in seven distinct states, queue empty. Both retain three independent text bindings, reuse the same held binding on reentry, and select one established public binding for two canonical Body requests. `BroadFlag` and `BroadFlagReverse` fail BindingPreserved; `Fresh` fails PublicCanonicalOwner; `NoBindingReuse` fails SameOwnerReentry; `BroadPublic` fails BindingPreserved. These are actual TLC invariant failures, not setup errors.

The model's labels are opaque already-captured binding tokens, not proposed transport enums. Public-contract compatibility is independently supplied by the two-contract fixture; the model does not prove Go schema equality, pointer/node extraction, nil binding, all alias/noRef/async behavior, or arbitrary graph termination. It adds eligibility correspondence; earlier acquisition/completion runs do not prove this new distinction. Use the maintained parameter order check, canonical-method/encoding/conflict cases and recursive contributor controls for implementation correspondence.

Checked with TLC 2.19 from tla2tools-1.7.4.jar (SHA-256 936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88), Amazon Corretto 25.0.5, one worker and 256 MiB:

```sh
java -Xmx256m -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -config ClaimBindingOwned.cfg ClaimBinding.tla
```

The candidate uses ClaimBindingOwned.cfg so it does not replace the protocol model's existing Owned.cfg. Other configs retain their distinct names. Proof execution remains explicit and opt-in, never a build/test/CI prerequisite. Raw logs and checker state are task artifacts, not repository inputs.

## Concrete owner and allocation invariant

`schemaRef.binding` captures the exact prepared `ValueTarget` pointer and `ValuePlanNode` example position at canonical suffix registration. `findMatchingSchemaRef` requires both for suffix reuse; `compatibleSchemaName` retains the literal public/preferred family rule. A suffix without a nonnil target and valid node fails before registration/reservation. Public canonical references and comparison registrations may have empty binding. `TestComponentClaimPreparedBinding` varies both dimensions independently and preserves exact-pair/preferred/public reuse; `TestComponentClaimRejectsUnboundSuffix` checks failure and unchanged indexes. `TestParameterPropertyNames` retains order-independent query/form/header samples; `TestDeclaredBodyCanonicalAcrossMethods` preserves the shared component and distinct media examples. The guard and pointer extraction are direct Go obligations outside the finite token model.
