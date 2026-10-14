# Canonical protobuf conversion-pair normalization (#937)

Base: `194ab48c693e41084d828aea794f97676a0bc03b`.

The shared representation fallibility query currently uses generic compatibility,
which admits object/union pairs. The protobuf emitter must instead unwrap its
message before pairing it with a service union. The resulting mismatched object
walk panics on existing union payloads and collection branches. Local exported
`GRPCEndpointWithUnionCollectionBranches` generation fails at this boundary;
CI also fails named union reuse, nested unions, map aliases and import-alias cases.

The correction belongs to canonical protobuf pair normalization. Preserve the
original child nodes and normalize only the protobuf side, using the same
`protoMessageCompatible` predicate as emission. Helper discovery and fallibility
must consume that one nonmutating operation. Remove the redundant helper variant.
The visited key is the canonical source/target attribute occurrence pair under
fixed contexts. Resolving physical names merely to form a key reserves names for
unwrapped unions and changes `Choice` to `Choice2`; the query must not perform
that allocation. Occurrence identity also retains sibling metadata/default facts.
Do not change generic object/union compatibility, wrapper wire shapes, defaults,
selected fields or the shared conversion capability contract.

1. [x] Preserve exact-parent generated failure and inspect all reported shapes.
2. [x] Add direct failing pair normalization and fallibility regressions, including
   both directions, unions containing collections, named wrappers, mapped names,
   recursive identity and incompatible pairs.
3. [x] Consolidate canonical normalization and remove duplicate helper policy.
4. [x] Run bounded exported-design generation/build/vet and existing custom String,
   recursive/default/disabled-default/error controls. Compare ordinary generated
   outputs against the parent byte for byte; record parent panics for repaired
   specimens and compare repeated candidate generation in separate processes.
5. [ ] Independent exact-diff review, atomic publication and CI follow-up.

The existing `codegen/tla/custom_representation/FallibilityGraph.tla` model assumes
correct graph extraction. This correction establishes the concrete protobuf
shape precondition; it changes no graph algorithm or modeled state transition.
Direct tests must preserve canonical identities instead of cloning recursive
nodes into fresh visited keys. The model remains a bounded classification proof,
not a proof of arbitrary expression normalization.

## Acceptance evidence

- Exact parent exported union-collection generation fails with the mapped-object
  panic (19.084s). Direct union pair tests reproduce shape mismatch; after shape
  correction, an independent name-allocation regression exposes `Choice2` churn.
- Candidate direct and generated gRPC race controls pass (57.844s), including
  custom String supplied/default failures, recursive values, scalar/collection
  wrappers, native controls, mapped metadata and union round trips. Existing
  protobuf and conversion goldens remain unchanged.
- All ten reported exported designs pass generation, example, module tidy, build
  and vet (55.834s): the expr union-collection specimen plus gRPC import aliases,
  message reservations, map aliases, direct/nested unions, named union reuse and
  CLI protojson.
- Five representative render specimens produce 60 artifacts: native defaults,
  recursive custom String, named union reuse, map union and nested union.
  All 24 ordinary native/custom artifacts are byte-identical to the parent.
  The three repaired specimens cannot finish generation on the parent; all
  candidate outputs compile, and all 60 match a second independent process.
- Targeted gRPC lint reports zero issues. No fixture regeneration or wire-format
  change is intended. Temporary capture code is excluded from the final source.

This ticket repairs the protobuf analysis owner only. Later main changes #936
(union physical identity) and #938 (HTTP fixture refresh) are independent and
must remain intact when integrating this patch. Independent review and delivery
are still pending.
