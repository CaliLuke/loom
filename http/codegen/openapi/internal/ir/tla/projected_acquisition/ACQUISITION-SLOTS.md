# Contributor result and mapping slot refinement

This opt-in finite design model supplements the unchanged ProjectedAcquisition model after focused N1–N4 review. The model inputs remain unchanged after implementation; README.md maps concrete owners and tests. No CI/default gate is implied.

## Exact scope

AcquisitionSlots separates a contributor preparation entry's reservation, component-handle binding, reentrant use, final reference-local result and slot installation. The supplied final result includes both a reference and an authored overlay/example token. It also follows a discriminator record through a nullable value-copy, then installs the mapping before a projection copy. Drain and post-scope root projections use the same exact request/key. These are source-correspondence abstractions, not concrete Go pointer/serializer proofs.

ReserveC records a visiting entry without a handle. BindC selects and records its stable component handle before Traverse can reenter. A composition consumer stores a deferred result-cell edge. FinishResult represents return through analyzePreparedSchemaPositions including its reference-local example; InstallComposition uses that complete result. UnboundSnapshot deliberately traverses before binding and snapshots zero. BareHandleSnapshot binds correctly but copies only the reference before the final overlay exists. They fail separately.

All actual contributor call sites are nested under an enclosing schemaAnalysisScope; their final result tails finish before its outer unwind. This is the source prerequisite for the modeled order, not evidence that a new top-level contributor call is safe. Future outermost acquisition would need the same scope to enclose its finalizer. No second drain is proposed.

CaptureMapping saves the walk-time discriminator record. Nullable wrapping moves that same record into the non-null assertion and leaves the annotation wrapper without it. InstallMapping targets the captured record, not the mutable outer Schema owner. OuterOwner updates the wrong wrapper record and loses the rendered mapping. ProjectBeforeInstall copies the non-null assertion before its mapping exists. Both fail MappingPreserved. The model explores both installation orders of the composition and mapping slots; source code must use construction relocation/stable result cells for schema edges because slice-copy setters are not shared mutable slots.

## Registration selection

All modeled registrations share the same supplied F and normalized hint. A preferred-first registration is supplied as having a suffixed public ref and no canonical claim association. A later canonical request must select a distinct recorded claim route and perform the supplied compatibility check. Canonical-first followed by preferred remains eligible for reuse. PreferredHintOnly chooses the preferred-only suffix for a canonical request and fails CanonicalAssociation. Compatible=false in the reverse order rejects before post-scope root projection; no final-name/publication proof is claimed.

This uses the same ordered F bucket plus selector/claim semantics as the combined API, not a new cache or universal canonical-boolean key. The original model's one-eligible-registration-per-F/hint fixture covered only canonical-first role order. The reverse case is an extension of that finite domain, not a claim that two legitimate selected associations violate the real reservation invariant. Actual uniqueness is per selected registration/recorded association. It cannot be inferred from F/hint alone in the extended case.

## Abstraction mapping and inherited evidence

For a fixed selected registration, ReserveC is a stuttering preparation step relative to the old graph; BindC corresponds to the old atomic Acquire component selection and contributors[ck] assignment. Traverse records/reenters an edge. FinishResult exposes the completed occurrence result formerly abstracted away. InstallComposition and InstallMapping refine the old Install phase; Materialize and Compare preserve its ordering. This is an explicit semantic mapping for the finite witness, not a machine-checked refinement theorem. The role-order extension has the separate scope described above.

The old 32 runs do not prove these new steps. They are reused unchanged only for their already accepted graph, context, count, rollback and completion properties. This new model does not repeat their arbitrary graph or body-completion reasoning. Both models assume exact captured contexts and selected binding facts; neither extracts actual ValuePlans, handles/name hashes or ExampleSets.

## Results and commands

Final inputs: AcquisitionSlots.tla and nine configs. Three positives (OwnedNullable, OwnedPlain, IncompatibleReverse) exit 0 with empty queues, eight invariants and weak-fair termination. OwnedNullable explores 32 distinct states. Six negatives exit 12 on their named invariant: UnboundSnapshot, BareHandleSnapshot, OuterOwner, ProjectBeforeInstall, RootDifferentOwner and PreferredHintOnly.

Toolchain remains TLC 2.19, tla2tools-1.7.4.jar SHA256 `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`, Amazon Corretto 25.0.5, one worker/256MiB. Run explicitly from this directory:

```sh
java -Xmx256m -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -config OwnedNullable.cfg AcquisitionSlots.tla
java -Xmx256m -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -config OuterOwner.cfg AcquisitionSlots.tla
```

Raw logs/temporary state are review artifacts, not repository additions. A parse error is not a semantic negative. Keep these commands out of CI, hooks, ordinary tests and release gates.

## Required Go correspondence

Add the maintained reentrant contributor case with distinct root/contributor F and final local overlay/example assertions; direct plus rendered nullable hinted Parent/explicit Child mapping in both encounter orders; drain/root same-request handle reuse; and preferred-only suffixed registration before canonical claim, with equivalent and conflicting candidates. Assert canonical association and concrete target/record ownership, not only names or counts. Preserve authored duplicate-mapping validation, source isolation, independent media examples and ordinary preferred reuse. Maintained tests mapped in README.md exercise these obligations; historical model receipts remain distinct from implementation evidence.
