# HTTP body projection (revised): composition identity per own contract

An opt-in design model for #1005's projected-body composition. It is not a
build, test, hook, CI or release requirement. It supersedes the first design
model under `source-v7-design`, which stays preserved.

## Placement

The model belongs to `expr`, beside the owner of body projection:
`objectHTTPBody`, `responseBodyObject`, `NewMappedAttributeExpr`/`Remap`,
`removeAttribute` and explicit Body cloning.

## Facts kept apart

- **Each declaration's own contract.** Parent `P`: `kind` (hint, required),
  `mode` (optional, values `{a,b}`), `id` (required). Child `C` narrows `mode`
  to `{a}`, requires it and adds `thumb`. Sibling `S` keeps `{a,b}` and adds
  `note`.
- **The occurrence's selection.** Authored members excluded by bindings, or an
  explicit inline Body subset.
- **The occurrence's wire facts per authored member.** Form names `kind`→`k`
  and `thumb`→`t`; an occurrence override makes `mode` JSON content.
- **Component identity.** Declaration plus complete occurrence context for
  provisional acquisition; completed graph equivalence owns canonical sharing.
  The first model abstracts direction-dependent policy away; the acquisition
  model below tests why JSON without exclusions is not sufficient for reuse.

## Settled invariants

| Invariant | Meaning |
| --- | --- |
| `RetainedComposition` | Every projected body of `C` (not an inline subset) references a `P` component. Partial and empty projections count. |
| `NoSubsetInheritance` | An explicit inline subset inherits no payload composition. |
| `NoReintroducedMembers` | Contributor and sibling components name no excluded member and require only selected wire names. |
| `WireConsistency` | Every component in the context uses the occurrence's wire name and codec for each authored member. |
| `ContributorOwnContract` | Projected `P` is `P`'s own members, values and requiredness, minus exclusions. There is no child leakage. |
| `SiblingAdmitsOwnValues` | A published sibling's effective values (own ∩ projected `P`) equal its own. |
| `HintRule` | A bound hint follows its member's wire name, or is suppressed when that member is excluded. The canonical hint is untouched. |
| `ChildRefinementRetained` | The body keeps `C`'s narrowing and requiredness. |
| `DeterministicIdentity` | One content per key; the canonical `P` is never rewritten. |

## Run explicitly

Use TLA+ tools **1.7.4** (jar sha256 `936a2620…`), one worker, `-Xmx256m`, a
fresh `-metadir` per configuration, and stop at 120 s or 1000 distinct states.

```sh
java -Xmx256m -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -config rhp-candidate.cfg -metadir /tmp/<fresh> HTTPBodyProjection.tla
```

| Configuration | Expected | Distinct states | Kind | Meaning |
| --- | --- | ---: | --- | --- |
| `rhp-current-full-loss.cfg` | Exit 12, `RetainedComposition` | 2 | **current code, Go-confirmed** (`form_json_v6/probe-root/run-1`) | Renamed full form body loses composition. |
| `rhp-current-header-loss.cfg` | Exit 12, `RetainedComposition` | 2 | **current code, Go-confirmed** (root `composed-form-header-diagnostic/run-1`) | `Header("kind")` form body is flattened. |
| `rhp-current-controls.cfg` | Exit 0 | 4 | current code | JSON identity reuse; explicit subset. |
| `rhp-naive-rename.cfg` | Exit 12, `NoReintroducedMembers` | 2 | implementation variant | Canonical JSON `P` beside form names. |
| `rhp-naive-exclusion.cfg` | Exit 12, `NoReintroducedMembers` | 2 | implementation variant | Canonical `P` reintroduces an excluded member and its requiredness. |
| `rhp-drop-partial.cfg` | Exit 12, `RetainedComposition` | 2 | prior candidate | Dropping partial bases loses authored composition. |
| `rhp-child-derived.cfg` | Exit 12, `ContributorOwnContract` | 2 | prior candidate (section 3.2) | Projected `P` built from the child body leaks `C`'s narrowing and requiredness. |
| `rhp-child-derived-sibling.cfg` | Exit 12, `SiblingAdmitsOwnValues` | 2 | prior candidate | The same leak forbids a sibling's own value `b`. |
| `rhp-subset-inherits.cfg` | Exit 12, `NoSubsetInheritance` | 2 | implementation variant | An explicit subset inherits the payload's bases. |
| `rhp-shared-key.cfg` | Exit 12, `DeterministicIdentity` | 2 | implementation variant | A projection written under the canonical key. |
| `rhp-declaration-wire.cfg` | Exit 12, `WireConsistency` | 2 | implementation variant | Contributors ignore the occurrence override. |
| `rhp-stale-hint.cfg` | Exit 12, `HintRule` | 2 | implementation variant | A renamed hint keeps the literal `kind`. |
| `rhp-no-suppress.cfg` | Exit 12, `HintRule` | 2 | implementation variant | An excluded bound hint is retained. |
| `rhp-candidate.cfg` | Exit 0, all invariants and termination | 20 | revised candidate | Ten scenarios: JSON full/excluded, form full/override/`id`/`kind`/all-`P` excluded, optional `kind`, explicit subset, mixed JSON+form. |

Countercontrols omit `Terminates`.

## Limitations

- **Abstract schemas.** Member schemas are abstracted to value domains, a wire
  name and a codec. Examples, defaults and openness follow the same projection
  as requiredness and are not modeled separately.
- **Unmodeled shapes.** Unbound or originally optional hints keep the existing
  3.2 rules; only optional `kind` is modeled. Explicit mapping targets,
  canonical-`P` publication, nested JSON fields, unions, tagged roots and
  ValuePlan construction are outside the model.
- **Body derivation is fixed.** It is the same in every design, so
  `ChildRefinementRetained` has no red countercontrol.
- **Abstraction.** The model checks the design abstraction, not the Go
  implementation.

Logs and checker state belong under task artifacts.

## Implementation correspondence

| Model fact | Implementation seam |
| --- | --- |
| Projection record (source identity, exclusions) | `expr/http_body_projection.go` `HTTPBodyProjection`; `projectHTTPBody`, `removeAttribute` and explicit Body cloning in `http_body_types.go` |
| `ContributorOwnContract`, `SiblingAdmitsOwnValues` | `expr.ProjectHTTPBodyContributor`: the base's own contract through the same exclusions |
| `WireConsistency` | `representation.ProjectContributor`: the shared normalization with the body's captured contract rebound by authored field identity. `captureOccurrenceJSONName` lets value capture keep the occurrence's effective JSON name for refined members |
| `RetainedComposition`, `NoSubsetInheritance` | IR `analyzeCompositionReferences` (every source base, captured occurrence context) |
| `HintRule` | IR `projectedDiscriminatorProperty` with `expr.DiscriminatorMember` |
| `DeterministicIdentity` | content-keyed contributor memo; the representation registry owns emitted components |

Projected publication, provisional acquisition and recursive reference reservation are covered by the additional bounded models below.

**Regressions:**

- `TestHTTPBodyCompositionProjection*` (v3, independently authored);
- `TestHTTPBodyProjectionRecordsAuthoredSource`, `TestProjectHTTPBodyContributorOwnContract`, `TestDiscriminatorMemberBindsWireProperty` and `TestValueCaptureKeepsOccurrenceJSONName` (expr);
- `TestProjectContributorUsesBodyWireContext` (representation);
- `TestBodyProjectionParentHint` (IR).

The original implementation preserved the initial model assumptions; subsequent witnessed context and ordering defects motivated the model extensions and explicit checker runs below.

## Complete selection and publication closure

A child-only excluded member must remain in the occurrence context while a
Parent is projected, even when Parent owns no such member. Physical deletion
intersects each declaration's members; forwarding and provisional identity
retain the complete original selection. `HTTPBodyProjection.Source` remains
the contributor's own declaration so discriminator binding and semantic
constraints are never borrowed from Child.

The added `excludeThumb` and `mixedChildExclusion` cases publish both Child and
Sibling. `NoReintroducedMembers` compares their contents with the original
occurrence exclusions. `PublicationOwnContract` checks the actual published
members, requiredness and value domains against each declaration's own expected
contract. `PublicationReachable` prevents dropping the family. Parent's local
schema can be equal across these contexts, so
`SameKeyImpliesSamePublicationContext` additionally checks its complete required
publication closure rather than merely its local fields.

| Configuration | Expected outcome | Design distinction |
| --- | --- | --- |
| `rhp-parent-filtered-members` | `NoReintroducedMembers` violation | Filtering forwarded context to Parent members loses Child.thumb exclusion. |
| `rhp-parent-filtered-identity` | `SameKeyImpliesSamePublicationContext` violation | The same filtering collapses the publication identity. |
| `rhp-parent-key-filtered` | `SameKeyImpliesSamePublicationContext` violation | Correct propagation alone is insufficient. |
| `rhp-parent-propagation-filtered` | `NoReintroducedMembers` violation | Correct identity alone is insufficient. |
| `rhp-publication-overfiltered` | `PublicationOwnContract` violation | Nonempty publication records with empty contents do not satisfy completeness. |
| `rhp-complete-context` | All invariants pass | Detached complete selection, local physical deletion, own semantic facts. |

The six configurations above were checked with TLC 1.7.4 during design;
negative modes exit 12 on the named invariant, and the positive mode completes.
The original candidate and current-controls configurations also still pass.
Use the same explicit command above with the selected configuration name.

## Memo, acquisition and prepared policy are separate

`ContributorAcquisition.tla` models two ordered caches: contributor memoization
and pre-analysis named-schema acquisition. A complete memo key cannot protect
publication if the second stage reuses Parent only by its locally equal shape.
The model has both full/removed form orders, both partial JSON request/response
orders, both full JSON request/response orders, and an equal full/full control.
It models the observable supplied-zero example of an authored `omitzero`
field: request presence retains zero while native response storage omits it.
It assumes those policies have already been resolved by the existing field
owner; it does not recreate Go storage or omission algorithms.

```sh
java -Xms32m -Xmx256m -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC \
  -workers 1 -config acquisition-complete.cfg -metadir /tmp/<fresh> \
  ContributorAcquisition.tla
```

The complete configuration passes all four invariants in 21 distinct states.
`acquisition-localAcquisition`, `acquisition-selectionAcquisitionOmitted`, and
`acquisition-policyAcquisitionOmitted` violate `AcquisitionOwnContext`;
`acquisition-policyMemoOmitted` violates `MemoOwnContext`;
`acquisition-hardcodedRequest` and `acquisition-canonicalShortcut` violate
`OwnMaterialization`. These negative controls all exit 12 on the intended
invariant. Parser failures and timeouts are not counterexamples.

JSON codec plus no exclusions is not sufficient evidence of canonical
identity: request and response policies can differ on that same full body.
Every HTTP body contributor now takes the captured-context path; only the
completed graph registry may merge equal schemas and publication dependencies.
The first model's canonical case abstracts away direction-dependent policy and
means equal completed observable graphs, not an implementation shortcut.

Implementation correspondence:

- `expr/http_body_projection.go` preserves detached full selection while
  deleting only locally owned members. `http_body_projection_context_test.go`
  checks ownership, detachment, own defaults and downstream Child projection.
- `representation/value_plans.go` captures the actual selected root layout and
  detached declaration registries. `body_contributor.go`/`body_context.go` retain
  resolved occurrence field wire name, visibility and omission by full authored
  member identity. Only the projected root receives common-member wire facts;
  Required/defaults and nested/sibling-only semantics remain their own.
- `transportir.ValueTarget.ProjectionContext` survives target copies; a rebuilt
  target replaces it. `TestBodyProjectionContextCapturesBoundLayout` checks an
  actual BuildValuePlan override, detachment and rebuilding.
- IR `componentAcquisitionFingerprint` combines semantic shape with complete
  context before lookup/reservation; semantic naming/sample identity stays
  separate. `completeSchemaFingerprints` retains final graph equivalence.
- The rendered Child-only exclusion test checks both orderings, including that
  full-context Child still has thumb. The partial and full JSON response-policy
  tests compare prepared ordinary projection with actual contributor examples
  in both method orders. `TestHTTPBodyContributorCapturedFieldPolicy` compares
  selected native-required field omission with Parent's still-optional semantic
  field, and checks a same-named nested field is not rebound from the root.

Bounds: finite representative declarations and seven ordered pairs; no proof
of arbitrary Go graph traversal, numerical constraints, codecs or performance.
Direct Go tests establish the captured field-policy/layout correspondence that
these models assume. Proof tools remain opt-in and outside ordinary gates.

## Recursive publication must observe the reserved reference

`ProjectionPublication.tla` models synchronous recursive reentry through Parent
publication and Child composition. The existing projection caches have distinct
roles: `projectedSchemas` owns a reference wrapper keyed by baseline node, plan
position and complete context; `projectedComponents` owns its component name.
Reserving only the component leaves the cached wrapper's copied baseline Ref
observable during recursive materialization. A publication that copies that
string is not repaired by updating the wrapper later.

`publication-current.cfg` reproduces `NoBaselinePublication` (exit 12).
`publication-reserve-first.cfg` completes all three invariants in six distinct
states: recursive observations see the projected identity, no baseline Ref is
published, and final publication reaches the completed component. Run with the
same bounded command using `ProjectionPublication.tla` and either config.
The first checker attempt required parentheses around an IF expression's RHS;
its evaluation error is excluded from counterexample evidence.

`projectBaselineComponent` now sets the caller's Ref on both new-reservation and
cache-hit paths before recursive schema filling. `projectBaselineSchema` and
`declarationRepresentationRef` use that one contract. The maintained rendered
`TestHTTPBodyContributorPublishesCompletedJSONChildren` checks that the actual
hinted JSON Parent publishes a Child with completed base64 constraints, never
the baseline binary byte schema. This is an ordering proof within a finite
synchronous cycle, not a proof of arbitrary concurrency or schema semantics.
