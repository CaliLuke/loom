# Nullable union conversion carrier

Parent: `66c61745673d797d4d15c860210fc082afc80fed`.
Worktree: `nullable-union-transform-carrier/loom`.
Status: implementation complete; final independent review pending.

1. Preserve the ordinary HTTP parent reproduction and search for duplicate issues.
2. Add direct carrier/dispatcher controls across named object, array, map, nested
   union and primitive branches, with intrinsic and occurrence nullability.
3. Model helper admission and carrier agreement, comparing the current bypass
   with dispatch through the existing nullable transform owner.
4. Obtain independent readiness review before implementation.
5. Implement in shared transformation, compile/run the bounded generated HTTP
   reproduction, compare parent/current/isolated-repeat artifacts and preserve
   ordinary union controls. Resume #829's unchanged full projected matrix.
6. Fresh independent exact-diff review before one atomic root-owned commit.

#829's candidate remains frozen in its separate worktree. Its declaration fix
exposed this existing defect; a parent ordinary Type HTTP payload/result without
ResultType or views reproduces undefined nullable object transform helpers.
`transformUnionCaseData` selects its named-object helper shortcut before
`transformAttributeStmt` can dispatch Nullable. `collectHelpers` independently
unwraps the attributes and emits concrete object helpers. Thus the union call
and helper declaration disagree on both helper name and parameter carrier.
Projection's custom helper naming hides the missing-name failure but leaves the
same wrapper-to-pointer argument error and pointer-to-wrapper result cast.

Candidate owner: route each selected union branch through shared attribute
transformation, whose nullable dispatcher already preserves absent/null/value
states and calls concrete helpers with unwrapped values. Retain the ordinary
named-object helper path only at the shared conversion owner where its physical
preconditions are established. Do not add transport-specific wrappers or duplicate
Nullable state handling. Check whether the existing object dispatcher already
provides the ordinary helper decision before deciding whether the union shortcut
can be deleted entirely.

Readiness source inputs and exact parent failure are under
`/tmp/loom-829-carrier-probe`. Duplicate searches for `nullable union transform`
and `nullable helper` found #829 and unrelated closed collection/body issues;
no matching open conversion issue found. A focused issue draft will receive root
editorial review before publication.

## Readiness freeze — #832

Issue: https://github.com/CaliLuke/loom/issues/832, filed after root editorial
approval of the exact setup, pushed source pin and explicit generation commands.

The 15-context direct matrix fails only named-object intrinsic and occurrence
Nullable branches; ordinary objects and nullable array/map/nested-union/primitive
controls pass. Parent generated HTTP compilation fails for the independent
unprojected specimen. Both declaration/call identity and parameter agreement are
explained by the bypass; #829's declaration normalization does not cause it.

The bounded model distinguishes current, names-only and null-collapsing designs
before the checked shared-dispatch choice. Results and limits live beside the
model. Do not remove the ordinary helper shortcut blindly: the attribute object
path emits an inline conversion at its root, so deleting the shortcut altogether
would change ordinary output without helping the nullable contract. The narrow
shared-owner change is to admit a direct named-object helper only for concrete
branch carriers; nullable branches use the existing attribute dispatcher.

Before final review, expand the existing generated specimen with all direct
matrix shapes where representable and exercise actual client/server conversions,
including null/concrete values, missing-value/nonnullable rejection and one
ordinary helper control. Compare exact parent/current/process-repeat artifacts
for this specimen and one unchanged ordinary-union specimen; compile/vet all
successful outputs. Pin external modules to the verified pushed parent version
`v1.10.0-alpha.9.0.20261013032024-66c61745673d` with exact local replacements.
Re-run #829's frozen whole projected matrix with this prerequisite overlaid to
establish the dependency; keep declaration changes out of #832's atomic commit.

## Implementation and bounded acceptance

Root approved readiness before production. The shared union case admits its
concrete object-helper shortcut only when neither side is Nullable. All nullable
branches use the existing attribute dispatcher, including rejection when source
and target nullability disagree. Ordinary helper generation remains byte-identical.

Direct 15-context shape/presence controls and both mismatch directions pass; the
parent accepts mismatched nullability and fails both new rejection controls.
The full shared codegen package passes (21.941s). The generated HTTP specimen
runs 42 branch/mode/value cases through actual client and server conversions,
checks rejected missing/nonnullable-null inputs before service invocation, and
checks programmatic absent object carriers remain absent during conversion.

Parent/current/process-isolated repeat comparisons cover 42 complete generated
artifacts: 21 for the expanded specimen and 21 for the existing physical carrier
control. Only three specimen files change: client/server types_helpers.go and
client cli.go. Every change replaces a nullable-object direct helper call with
the existing nullable state dispatcher and concrete helper invocation. Ordinary
object cases and all control artifacts remain identical. Current and repeat are
byte-identical; successful modules compile/vet and run their harnesses. Parent
nullable object compilation fails as expected; its ordinary control passes.

The frozen #829 projected HTTP/JSON-RPC nullable matrix compiles/vets/runs with
only this prerequisite production file overlaid (5.903s), resolving the independent
conversion failure without changing #829's declaration candidate.

One generated case remains explicitly outside #832: an intrinsically nullable
named nested union fails ordinary service declaration formatting as
`type loom.Nullable[NullableNested] struct`. Root assigned this declaration-owner
finding to #829. Its exact test input and parent-class failure are preserved in
`/tmp/loom-832-intrinsic-union-identity-witness_test.go` and the adjacent log.
Direct intrinsic nested-union conversion coverage remains green; runtime coverage
uses the representable occurrence-nullable nested union. No case is recorded as
passing when it was excluded for that declaration defect.

Evidence and source pins: `/tmp/loom-832-comparison/summary.json` records every
artifact hash; `changes.diff` records exact intended output differences. External
comparison modules use pushed parent version
`v1.10.0-alpha.9.0.20261013032024-66c61745673d` and exact local replacements.
No generated artifacts or checker output are committed. Models abstract matching
nullable layouts; direct mismatch tests establish rejection outside that domain.
