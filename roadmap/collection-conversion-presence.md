# Collection conversion presence (#889)

Status: implemented against delivered `ee1d653cf`; final independent review pending.

1. [x] Preserve the exact #887 full nil roundtrip and parent constructor failure;
   archive delivered work and safely reuse its worktree.
2. [x] Reproduce root array/map conversion with new and existing destinations;
   retain supplied empty/populated, nested collection and named constructor cases.
3. [x] Independently review the shared owner and bounded acceptance.
4. [x] Implement at collection conversion, compare affected generated outputs,
   compile/run the existing mounted witness and focused presence/default controls.
5. [ ] Independent final review, atomic delivery and temporary-output cleanup.

## Established owner

`InitStructFields` calls `GoTransform` for an unnamed HTTP header argument going
into a named collection field. `renderTransformGoArray` and
`renderTransformGoMap` allocate unconditionally. A nil source becomes a nonnil
empty destination. The same operations serve root values, nested collection
elements and object fields; some callers guard nil independently, so correctness
currently depends on the call path. This is a shared conversion contract, not a
header decoder or projection template defect.

The declaration/binding repair #887 deliberately left this constructor unchanged.
An ordinary non-view alias fails on archived `7ed1bfc63`; a plain-array viewed
constructor directly assigns and preserves nil. Both actual repro inputs and the
full mounted named-view failure are retained at `/tmp/loom-889-parent-witness`.

## Correction and boundary

A native collection conversion preserves its source storage state: nil remains
nil, nonnil empty remains nonnil empty, and supplied contents are transformed.
This applies to both array and map owners. Declare a new target outside any guard;
for an existing target, nil must clear previous contents rather than leave stale
values. Allocate and copy only in the supplied branch. Preserve destination type
aliases, element presence/wrapping and current copy isolation. Do not add a new
allocator or nil check in `InitStructFields` or transport templates.

`FieldPresence`, Optional/Nullable wrapping, absent-only defaults and validation
remain separate owners. In particular, documented required-array encoding and
selected union nil-as-empty semantics are not redefined by this storage fix.
Review existing outer guards/default branches for redundancy or changed behavior;
remove only checks whose responsibility the shared collection conversion now
fully owns. Required admission and explicit default materialization must remain
visible at their owning boundaries.

## Bounded consumers and acceptance

- Shared root `GoTransform`: native and named arrays/maps, declaration and
  existing destination assignment; nil, empty and populated states; nested nil
  collection elements and nonnil empty element controls. Assignment clears a
  prepopulated destination; populated output remains detached.
- `InitStructFields`: reproduce the actual named collection header constructor.
  The existing full mounted #887 check covers both views, server bytes and client
  nil/empty/populated roundtrip without a workaround.
- Object field conversions, named helper calls and Optional/Nullable values all
  delegate to the collection transform. Reuse the existing projected presence,
  Bytes/default and nullable named collection controls rather than adding a new
  runtime harness. Keep ordinary primitive/object/union policies unchanged.
- HTTP/JSON-RPC generation and gRPC callers of shared `GoTransform` consume the
  correction; protobuf's separate collection transformer is not silently changed.
  Select a nested collection design and the named-view/ordinary-alias controls
  for actual parent/candidate/repeat output comparison and compilation. Reuse the
  three checked-in fixture capture only for changes affecting their outputs.

The existing `codegen/tla/projected_presence` model already rejects materializing
an absent collection before validation and accepts preservation. Reuse its
collection-presence invariant, state explicitly that it models fresh targets,
and use compiled tests for clearing prepopulated destinations and nested values.
No new concurrency model or full exported-design sweep is needed.

## Readiness evidence

Compiled shared-seam tests reproduce eight root nil failures (native/named
arrays/maps, new/existing target), two nested nil failures and the original two
mounted view failures. Supplied empty/populated cases and detached-copy controls
pass. Existing projected-presence, Bytes/default and nullable-union controls pass.
The unchanged model's objects-only alternative violates PresencePreserved in
105 distinct states; the preservation configuration passes all 192 states.
Independent readiness review approved this owner and bounded acceptance without
findings. The implementation follows that approved plan.

## Final implementation and evidence

`renderCollectionConversion` now owns declaration, allocation and nil assignment
for arrays and maps. Object-field conversion delegates to it, retaining only the
existing required-array encoding materialization branch. The mounted named-view
witness now exercises absent values instead of returning early for the tracked
limitation.

Focused compiled race tests pass for all eight root variants, nested collections,
projected presence, Bytes/default behavior and nullable unions (10.164 seconds).
The complete affected transform/constructor selection and regenerated owner
goldens pass (6.863 seconds). The original mounted HTTP witness passes both views
and all three storage states (4.978 seconds). Four optional/required nullable
collection module controls pass (12.413 seconds); gRPC metadata and JSON-RPC
selected-response controls pass (1.756 and 6.577 seconds). Configured codegen lint
reports no issues. An initial stale structural assertion rejected the new explicit
nil assignment; it was corrected and the final focused selection passes.

Actual CLI comparison uses an archived parent checkout, the candidate and a
separate repeat process. All four specimen modules build and vet at parent and
candidate. Of 66 generated artifacts per run, seven files change: named-view
client constructors and service conversions, plain-view service conversions,
ordinary-alias client constructors, and nested HTTP server/client/CLI conversion.
Each difference is collection declaration, guarded allocation or nil assignment;
type declarations, schema and wire framing remain unchanged. Candidate and repeat
are byte-identical. All 55 generated files across HTTP ticktock, JSON-RPC ticktock
and mixedtick are byte-identical to the parent and checked-in fixtures; parent
and candidate fixture modules build and vet.

The final packet records exact sources, inputs, artifact hashes and commands in
`/tmp/loom-889-final-manifest.json`; the seven-file output comparison is
`/tmp/loom-889-output-comparison.json`. Existing model results remain applicable:
its source and invariant are unchanged, while the README now maps collection
ownership to the shared renderer. The model does not distinguish an existing
populated destination; compiled runtime checks establish that obligation. No
full exported-design audit or framework-wide test pass is claimed.
