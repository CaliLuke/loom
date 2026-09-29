# Recursive representation equivalence

The #574 mixed-representation comparison found two identical completed recursive
schemas under different component names. The old fingerprint records active DFS
depth using original component identities. A self-cycle and an equivalent graph
with an extra unfolding before that cycle therefore receive different hashes.
The direct registry regression reproduces this before the production repair.

RepresentationEquivalence.tla models the proposed shared-owner correction:
partition the finite complete component graph by exact non-reference data,
refine classes using ordered reference edges without merging prior classes,
stop when the class count stops increasing, and fingerprint the rooted quotient
graph. Representative component identities do not appear in the fingerprint.
The observation oracle unfolds the bounded graph independently; it does not use
partition classes or DFS cycle markers.

Run each configuration separately, with checker output outside the repository:

    java -cp /path/to/tla2tools.jar tlc2.TLC \
      -metadir /tmp/loom-representation-checked \
      -config checked.cfg RepresentationEquivalence.tla

Replace the configuration and temporary directory for the negative controls.
The legacy configuration must fail EquivalentContractsShare; overmerge must
fail DifferentContractsStaySeparate. The latter deliberately drops reference
edges. The checked configuration must pass refinement, convergence, observation
agreement, and fingerprint sharing/separation.
Both checked configurations also require AcyclicFingerprintsPreserved: for
acyclic roots, the quotient and legacy fingerprints have the same abstract
structure. This assumes both use the same fingerprint serialization.

TLC 2.19 checked the following configurations. A failed negative control must
name its expected invariant; a parser, tool, or resource failure is not evidence.

| Configuration | Result |
| --- | --- |
| legacy.cfg | Exit 12: EquivalentContractsShare fails for a self-cycle and an equivalent prefix into that cycle. |
| overmerge.cfg | Exit 12: DifferentContractsStaySeparate fails when one root has a child and another does not. |
| checked.cfg | Exit 0: 41,888 distinct states, including 32,768 initial graphs; search depth 2. |
| checked-chain.cfg | Exit 0: 16,000 distinct states, including 10,000 initial graphs; search depth 3 exercises successive refinements. |

The branching configuration has three component nodes and two ordered reference
slots per node. The chain configuration has four nodes and one slot per node.
Both have two abstract non-reference labels and absent-reference slots. They include
self-cycles, mutual recursion, prefixes into cycles, and branching references.
Labels stand for exact serialized schema keywords, annotations and external
references; extracting those labels and ordered local-reference paths remains a
Go test obligation. Partition equality compares exact signatures; the model
assumes collision-free hashing only for the final fingerprint.

This is a bounded algorithm check, not a proof of arbitrary schema equivalence
or Go refinement. It does not model public-name reservations, source-declaration
and legacy-shape collision guards, actual codecs, example sampling, or JSON
Schema instance semantics. Keep the registry's direct negative controls and
parent/candidate artifact comparisons for those boundaries.

A production compatibility control found that changing fingerprint serialization
could change which synthetic envelope receives the canonical name, even for
acyclic graphs. The model's structural tuples did not represent that serialization
change. Retaining the old serialized schema shape and substituting only recorded
reference slots is a separate Go obligation; the acyclic invariant alone does
not prove exact JSON bytes, hashes or allocated names. Keep a direct two-envelope
collision regression and the complete parent/candidate comparison for it.

Production ownership remains in the shared IR representation registry and its
complete-graph equivalence helper. Do not deduplicate rendered documents or
special-case fixture component names. The direct controls in
[representation_registry_test.go](../../representation_registry_test.go) are
TestRepresentationRegistrySharesEquivalentRecursiveUnfoldings,
TestRepresentationRegistryKeepsRecursiveContractDifferences and
TestRepresentationRegistryReservesPublicNames. Pair them with the mixed
byte-representation-ownership probe in the revision comparison.

Retain only this source, configurations
and concise findings; remove task-owned checker output after validation and
independent review under the repository's cleanup process.


## Response public identities

`ResponseAllocation.tla` separates complete response equality from the historical
serialized key used for public names. Retaining reference-sibling examples
changed a non-Bytes response suffix even though its emitted schema was unchanged.
The naming-only projection cannot authorize semantic sharing.

The complete parent ordered pass reserves every historical slot before semantic
eligibility filtering. Each eligible original representative keeps its slot and
per-use references, including multiple public names for one complete contract.
Retired slots and authored-only names remain reserved. Split classes use checked
fallback names; genuinely unassigned equal uses reuse the earliest binding.
`reusable_components{,_helpers,_naming}.go` owns this separation.

Run `ResponseAllocation.tla` using `response-checked.cfg` or one of the six
`response-*.cfg` negative configurations, with task-owned `-metadir` output.
Checked passed **887,040 distinct states** from 82,944 initial assignments.
`current` fails `OriginalRepresentative`, `old-dedup` fails `NonMerge`,
`unreserved` fails `PublicSlotOwner`, `drop-ineligible` and `steal-owner` fail
`RetiredReservation`, and `steal-authored` fails `AuthoredOnlyReserved` (exit 12).

The domain has four ordered uses, three semantic classes, independently assigned
historical keys, all forcing subsets, and reserved-name collisions. Allocation
order is fixed and canonical; subsequent emission explores every order. Keys
abstract serialization and complete-contract equality; the model does not prove
recursive hash extraction, hash collision resistance, parent suffix collisions,
or arbitrary allocation-order invariance. Integer fallback candidates abstract
hex-prefix extension and checked numeric suffixes. Direct response allocation
tests cover three historical contexts, exact producer preservation and occupied
prefixes. `TestResponsePublicIdentityPreservesReferenceAnnotations` uses the
literal parent-generated suffix; `TestResponseRetainsHistoricalSlotAfterSemanticSplit`
checks an independently recorded later name. Full isolated artifact comparisons
remain required. Keep checker output outside the repository and remove owned
state after review; retain sources, configs and concise findings here.
