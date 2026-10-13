# Nested union validator closure — #819

Parent: `73451e4dad96837246d90fad56e1e492198e14b4` (delivered #823).
Implementation: `nested-union-validator-integration/loom` managed worktree,
union_carrier_recovery agent; root independently reviews and integrates.
The original `1eba09fe5` candidate and evidence remain recorded below.

1. [x] Reproduce existing `TestUnionInUnionRoundTrips` on current main; both
   HTTP and JSON-RPC clients reference absent `ValidatePeerResponseBody`.
2. [x] Preserve generated witness and identify the shared declaration/call
   eligibility owner. Compare nearby no-op, constrained, nested union and
   recursive cases before choosing a repair.
3. [x] Add a direct red seam test and model declaration closure if traversal or
   eligibility interactions need a revised invariant. Reuse established physical
   identity and recursive-validation contracts; no per-transport missing-symbol
   exception or generated-code edit.
4. [x] Independent readiness review of root cause, repair and bounded acceptance.
5. [x] Implement at the shared owner; preserve wire validation and type identity.
6. [x] Existing cross-transport round trips, affected direct tests, representative
   parent/candidate/repeat exact bytes, generated compilation and explicit lint.
7. [ ] Fresh final review, atomic commit/push/closure, durable receipt and cleanup.

Original reproduction: `/tmp/loom-union-validator-current.log`, Go 1.27.2;
`LOOM_DIR="$PWD" go test ./codegen/generator -run '^TestUnionInUnionRoundTrips$'
-count=1` fails during generated vet on the original parent. The integrated implementation
is complete and awaiting final review.

## Readiness owner and proposed correction

`hasValidations` walks every attribute using one caller `AttributeContext`.
Required primitive eligibility depends on that context's Pointer/IgnoreRequired
flags. `renderUnionSumValidationCases` legitimately changes a union branch's
value-storage mode, but a named descendant has independently registered physical
fields. `NamedValidationFunction` already uses `presenceValueContext` to restore
that layout for definitions; eligibility does not. Thus a root call to Peer is
emitted while Peer's nested union loses its required Leaf predicate and its body
is empty. Empty helper stubs would fix compilation while preserving the omission.

The same walk also overstates requiredness for inline object union branches,
whose primitive fields are values. A direct control proves the generated body is
empty while eligibility incorrectly says true. This is the same layout-traversal
owner, not a separate transport issue.

Replace the generic context-free Walk in named eligibility with a private,
cycle-safe contextual traversal. At every named boundary derive layout through
`presenceValueContext`, the existing named-function owner. Share the sum-union
branch context adjustment with its renderer rather than copy its policy. Preserve
ordinary predicate detection and generatedRequiredValidation; do not add semantic
predicates or change function names, nullability, wire formats or caller storage.
Track visited attribute/layout pairs so recursive graphs terminate without
conflating distinct physical layouts. Retain the current nonempty-definition
collector contract. No consumer fallback or public API is needed.

## Red evidence and acceptance

- `/tmp/loom-819-direct-red.log`: ten failing physical-versus-caller cases across
  root leaf, nested union, array, map and recursive shapes; one empty nested-union
  render failure; one inline-union false eligibility. Existing standalone Leaf
  generation still checks `Name == nil`.
- `/tmp/loom-819-generated-red.log`: existing cross-transport specimen with new
  invalid-response and direct constructed-value checks still fails at undefined
  Peer helper before runtime. The new checks require missing Leaf.name rejection
  through named/nested/peer union HTTP and JSON-RPC responses, and require Peer
  validators to reject constructed invalid values then accept corrected values.
- Existing retained parent module `/tmp/loom-819-parent` confirms physical
  `LeafResponseBody.Name *string`, nonempty Leaf validator and absent Peer helper.
- Bounded model Old fails CallClosure (41 generated / 29 distinct); Checked
  passes closure, predicate coverage, no false requirement, finite registry and
  termination (52 / 32). Logs `/tmp/loom-819-model-{Old,Checked}.log`.
  The existing #709 and #810 models remain unchanged; this model adds the missing
  physical-eligibility premise beneath their callable identity assumptions.

Before final review: affected shared codegen tests and the existing cross-transport
round-trip harness, focused recursive/nullable identity controls, pinned lint,
parent/current/independent-repeat generated artifacts and compilation for this
specimen plus representative unaffected recursive and nullable specimens. Cover
inline anonymous branch, value-layout no-op and pointer-layout required cases at
the direct seam. Reuse existing generators and fixture helpers; no full corpus.
Only definitions/calls required by corrected physical-layout eligibility may
change; valid wire behavior and established diagnostic paths remain unchanged.
No production files were edited at the original readiness freeze.

## Candidate checkpoint before branch-carrier prerequisite

The contextual traversal and shared union branch context are implemented.
Visited keys include attribute identity plus Pointer, IgnoreRequired, JSONPresence
and UseDefault; repeated nodes in a different physical layout are revisited.
Direct both-order shared-attribute tests protect that obligation. Unsupported
attribute kinds retain the previous explicit failure rather than silently omit
validation. Root approved readiness before these production edits.

Candidate acceptance passes: full shared codegen (28.968s); nested-union generated
round trips and invalid response/constructed-value cases (5.540s); HTTP anonymous,
named nullable and selected recursive controls (38.537s); JSON-RPC named nullable
controls (16.407s); pinned codegen lint (zero findings). Logs are
`/tmp/loom-819-codegen.log`, `/tmp/loom-819-generated.log`,
`/tmp/loom-819-http-controls.log`, `/tmp/loom-819-jsonrpc-controls.log`, and
`/tmp/loom-819-final-lint.log`. New direct cases pass in
`/tmp/loom-819-final-direct.log`.

Bounded comparison `/tmp/loom-819-comparison/summary.json`: 77 artifacts across
nested union, recursive object and named nullable object specimens. Independent
current/repeat bytes match exactly. Only four HTTP/JSON-RPC client/server
validation files differ, restoring nested predicate calls and nonempty helper
bodies; recursive/nullable artifacts and gRPC are unchanged. Current modules
compile/vet/run; the exact parent union specimen fails at its known missing
helper while both parent controls pass. `changes.diff` retains every difference.

A further constructed-value check exposed a separate preexisting carrier bug.
With only `MinLength(1)` added to parent Leaf.name, the unchanged parent compiles
and passes its original round trips, then `NewChoiceOrTags2Choice(nil)` passed to
ValidateShowResponseBody panics at actual.Kind(). Wire null rejects safely.
The branch renderer infers RequiresValue from rendered text instead of physical
pointer storage. Logs `/tmp/loom-819-nil-parent-generation.log` and
`/tmp/loom-819-nil-parent.log`; reproduction module `/tmp/loom-819-nil-parent`.
Root requires an atomic prerequisite repair before #819 final review. This
candidate is frozen while that independent branch-carrier owner is repaired.
Do not weaken invalid-value checks or add empty helper stubs. Reuse unaffected
acceptance and rerun the combined changed boundary afterward.


## Recovery onto the delivered carrier prerequisite

The frozen 10-file manifest `/tmp/loom-819-candidate-manifest.json` verified
unchanged (SHA-256 `ab149eeefc8ee89de252c6e205c9f1d1878972d9237be01680ebce57a9493030`).
Its original worktree and outputs remain intact. Continued work uses the managed
`nested-union-validator-integration/loom` worktree at pushed
`73451e4dad96837246d90fad56e1e492198e14b4`, which includes #823.

Before transferring production code, the frozen direct/generated regressions
were rerun on that parent. Missing named-layout predicates remain reproducible;
the generated response now reaches a conversion panic because #823 supplies a
presence-only helper body, rather than an undefined helper. An added direct
case confirms that a union's nil-pointer check also needs named eligibility
when the branch value has no authored predicates.

The model now separates that structural branch-presence obligation from the
primitive's registered physical field layout; Old and NoPresence each fail
41/29, while Checked passes 104/64. The combined helper will return both branch
context and physical pointer presence to eligibility and rendering. The #823
presence-only case admission and break behavior remain intact. No semantic
policy changes, compatibility paths or transport-local exceptions are needed.


## Integrated candidate acceptance

Root independently approved the extended model and direct red evidence before
production integration. `hasValidations` now traverses attribute/layout pairs,
restores named physical mode through `presenceValueContext`, and shares
`unionValidationContext` with sum-union rendering. That owner returns branch
context and structural pointer presence independently. The #823 nil guard,
presence-only case admission, nullable/collection treatment and child-field
`validationValue` reset remain intact. No decoder or #824 changes are included.

All integrated checks use Go 1.27.2 and local source at the parent above:

- Full shared `codegen`: pass (26.415s). Direct physical/layout regressions pass,
  including both shared-layout visit orders, inline value no-op and unconstrained
  named branch presence.
- Unmodified `TestOptionalUnionValidation` and #823 generated carrier harness:
  pass (6.080s), including view validation and nil/non-nil branch controls.
- Existing nested-union design and valid round trips are unchanged. Extended
  invalid response and constructed-value checks pass (6.705s), including named
  required fields and nil/non-nil unconstrained values through a parent helper.
- Current and separate-process repeat generated/compiled/vetted/runnable bounded
  matrix: 53 nested-union, 12 recursive-object and 12 named-nullable artifacts.
  All 77 repeat hashes match exactly. Recursive and nullable parent bytes remain
  unchanged. Parent nested responses expose missing validation; candidate rejects
  them before conversion. The final parent-production overlay separately verifies
  the added unconstrained nil branch regression is red.
- Pinned golangci-lint v2.14.0: shared codegen and all three generated specimen
  modules pass with zero findings.

Exact generated changes are limited to six nested-union HTTP/JSON-RPC files:
client/server validation files restore Leaf calls under nested/peer unions and
retain safe guard breaks before those calls; client/server list validators reach
collection element presence checks; client show validators reach Peer checks.
The two client CLI files gain the same Peer/list validation and error return.
These last structural-presence calls are the approved #823 correspondence delta;
no gRPC, service types, wire codecs, recursive or nullable artifacts change.

Evidence: `/tmp/loom-819-integration-comparison/{summary.json,changes.diff}` and
`/tmp/loom-819-integration-*.log`. An initial combined harness output capture was
invalid because its temporary overlay reused one output directory across designs;
that attempt is excluded. The accepted matrix runs only the selected union test
in its capture and runs optional/carrier controls in their original temp dirs.
Review and downstream integration still need these outputs and the retained #823
artifacts; cleanup follows their completion.
