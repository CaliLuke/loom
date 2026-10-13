# Explicit alias body source binding — #893

## Contract and ownership

An explicitly selected named HTTP body may retain an alias chain independent
of the service payload/result/error declaration. Finalized transport bindings
authorize the semantic source; declaration ancestry continues to own names,
annotations and constraints. Extend may flatten the service object's shape
without making a source-authored response Body invalid.

Current parent `066b68aab`: response finalization binds only the outer body.
`valuePlanAliasSource` supports an independent terminal object definition but
rejects intervening named aliases absent from the service's alias chain. The
existing importer base-only shared-error test therefore evaluates successfully
and then fails actual generation. Request and response binding must be examined
together; do not add importer flattening or remove explicit Body/OpenAPIBody.

## Sequence

- [x] Read the actual issue, body finalization, controlled copy/source ancestry,
  value-plan pairing and existing declaration/pairing model.
- [x] Reproduce the current generated failure and direct independent named-body
  alias cases; retain foreign/cyclic rejection and ordinary structural controls.
- [x] Establish the shared binding owner: detached transport occurrences must
  preserve declaration identity while authorizing their complete root alias
  representation. Confirm shared authored declarations are never rebound by
  another endpoint. Freeze the precise owner proposal before production.
- [x] Extend the existing alias-pairing model only for the missing explicit
  binding distinction; check root-only and indiscriminate ancestry alternatives.
- [x] Obtain independent readiness review before implementation.
  Root approval: `/tmp/loom-893-root-readiness-verdict.json`.
- [x] Implement the approved shared owner and remove superseded binding paths.
- [x] Run direct regressions plus the existing shared-error object/base/map/alias
  generated cases, bounded request/response reuse controls and spec checks.
  Compare parent/candidate artifacts and independent generation where changed;
  compile each distinct output once. Preserve rejection and child contracts.
- [ ] Freeze exact final diff for independent review, then root integrates.

## Boundaries and open investigation

The proposal must not turn copied foreign aliases into valid mappings merely
because their shape matches. Source binding must come from finalized transport
mapping, not inferred names, equal kinds or hashes. Alias cycles still reject.
Local constraints, declaration IDs, field ancestry and selected source examples
remain independently checked after pairing.

The recursive importer error test's pointer assertion is a real ownership bug,
not stale. `finalizeObjectBody` calls `initAttrFromDesign` on a shared named body
before copying. That overwrites `Node.child.Type` with the derived error's
separate Node copy. The effective-design model requires canonical structural
back edges; a direct before/after regression reproduces this exact mutation.
It belongs to this same explicit-body binding correction, and the existing
importer pointer assertion remains unchanged in acceptance.

The initial checkpoint above was readiness only. Production implementation began
after the independent readiness approval; final review remains required.

## Frozen owner proposal

1. Explicit response preparation must detach the body before any field/source
   initialization or original-name metadata mutation. Request preparation already
   copies before binding; keep that property. Copies retain authored provenance.
2. Retain private explicit-binding provenance set only by actual
   `bindValueSource`, and preserved by controlled copying and occurrence capture.
   A nonnil `valueSourceOrigin` is insufficient: ordinary copies populate that
   pointer too and must not acquire binding authority. Treat the finalized root
   source binding as occurrence-owned authority for
   its representation alias path. Preserve ordinary nearest-ancestor pairing
   first; only an unmatched named representation edge with that explicit
   authority may retain the current semantic source. At the structural terminal,
   descend to the source structure and retain exact kind/member/branch checks.
3. Do not rewrite every named declaration's semantic origin to the root. A
   recursive child may reuse that declaration with a different source occurrence.
   Binding authority must travel with the paired occurrence/alias traversal,
   reset at structural children, and participate in memo identity if it affects
   node construction. Independently bound child occurrences retain their own
   authority. Declaration IDs, local constraints and source associations remain
   independent from this permission.
4. Plain controlled copies do not grant arbitrary new alias ancestry. Keep the
   existing foreign/cyclic rejection tests and matched-alias source sequence.
   Do not use equal names/kinds or a global enum-mode bypass as authorization.

The implementation may adjust private representation details, but widening
unbound ancestry admission, mutating shared declarations, or flattening imported
schemas would contradict this proposal and require renewed review.

## Readiness evidence and bounded final gates

On `066b68aab`, the existing base-only error generator regression fails after
10.34s with the reported alias-ancestry diagnostic. Six new direct cases fail:
request/response alias depths one and two, response occurrence isolation, and
response recursive source preservation. Depth-zero request/response and request
isolation pass. Existing source-binding, matched alias sequence, independent
request field mapping, unrelated-field and foreign/cyclic controls pass.

`ExplicitBodyBinding.tla` next to the existing alias-pairing model contrasts
root-only mapping and binding before detachment with detached complete occurrence
mapping. It abstracts already-authorized transport bindings; `AliasPairing.tla`
and direct Go controls retain unrelated-ancestry rejection. The model does not
establish graph extraction or recursive copy correctness.

Final bounded acceptance uses these direct cases, unchanged foreign/cyclic,
source/declaration and streaming wrapper tests, existing importer shared-error
four-shape table, and both unchanged recursive error YAML/JSON cases. Actual
parent/candidate generation must preserve error fields, authored component names,
examples and valid specs; compare ordinary object/map controls and every intended
alias/recursive output change. Extend existing module capture, not a new corpus.

Focused authority acceptance also includes the same source/target shared subgraph
encountered with and without authority in both orders (memo isolation), a bound
root containing an unrelated nested alias (no structural-child privilege leak),
and the unchanged matched-source sequence. These direct checks establish private
provenance correspondence; no additional generated corpus is required.

## Implementation correspondence

`AttributeExpr.valueSourceBound` is set only by `bindValueSource`. Copying retains
it, occurrence capture snapshots it separately from local constraint contracts,
and manual transport/view wrappers propagate it alongside existing provenance.
`valuePlanAliasSource` prefers ordinary ancestry; an explicitly authorized,
acyclic unmatched alias reuses its current semantic source. The builder includes
that authority in memo keys, propagates it only down alias edges and resets it
at structural nodes. Child mappings independently acquire only their own binding.
`HTTPResponseExpr.finalizeBody` detaches before `initAttrFromDesign` or original
name writes, preserving canonical authored recursive graphs.

All six initial direct failures and the authority/memo/child/cycle controls pass.
Full Expr/DSL race checks pass (8.291s/1.445s). The existing four shared-error
shapes generate and unchanged recursive YAML/JSON assertions pass
(50.062s); both syntaxes render identical DSL, so the existing harness compiles
that recursive output once. Actual archived-parent/candidate byte comparison is recorded in the
final evidence packet. No importer production or recursive identity assertion
was changed.

Final actual generation capture passes in 61.126s (candidate) and 44.102s
(archived parent controls). Five candidate modules contain 80 generated artifacts;
every byte repeats across independent generator processes. The three parent
controls contain 48 artifacts: map and alias-map are identical, and ordinary
object changes only `gen/loom.json`'s design digest because transport initialization
no longer mutates the authored declaration's `name:original` metadata. All Go and
spec output for that control stays identical. Base-only and recursive outputs
become available after their retained parent failures; all 32 new artifact paths
are listed in `/tmp/loom-893-final-artifact-comparison.json`.

Each distinct generated module compiles; all five vet successfully. Ten rendered
JSON/YAML specs parse with libopenapi and pass the pinned Vacuum error rules.
The recursive fixture also asserts the complete component graph. Existing
checked-in HTTP/JSON-RPC ticktock and mixedtick designs contain no explicit Body
or Extend declarations; no checked-in fixture is an affected alias/body control,
so they are not regenerated for this repair. Scope is the existing importer
shared-error and recursive specimens, not all exported designs.

Private-authority tests additionally cover alias-local length constraints,
controlled copies versus explicit binding, local-constraint capture, streaming
wrappers, cyclic sources, structural-child isolation and both memo encounter
orders. Production sources are unchanged after the final full Expr/DSL race and
actual artifact capture; later edits only strengthen assertions and document
results. Independent final review is the remaining implementation gate.
