# Error declaration admission (#699)

Initial parent: `555d13306e126da0df4d0cefb156aa027e5ccfa5`; current readiness
parent `d212ddc51a121549a18881b1ef4c603835b94986`. Relevant expr/DSL owner files
are unchanged between them. Independent readiness review approved the owner
before implementation. Implementation parent is pushed
`5c6535b6d0ad92305c2dd14fbf963ff950d79a56`. Worktree reuse receipt:
`/tmp/loom-699-parent-reuse.json`. Original issue: `/tmp/loom-699-issue.json`.

## Contract and owner

An exact wire error name identifies one declaration in each authored API,
service or method scope. Repeating that name in the same list is invalid even
when both types match. Separate scopes may reuse or override the name. Distinct
wire names that normalize to the same Go name remain legal (#734 owns their
generated factory identities).

`dsl.Error` appends API errors to `RootExpr.Errors`, and service/method errors
to their local `Errors` lists. `eval.RunDSL` validates the complete authored tree
before finalization. Root/service/method validation must call one private expr
error-list name admission helper. Preserve existing per-error validation and
effective type-discriminator validation; do not broaden API error type semantics
inside this ticket. `MethodExpr.effectiveErrors` continues to implement method
override and ordered service inheritance after local uniqueness is established.
Do not deduplicate invalid authored lists, reject legitimate cross-scope names,
or patch downstream generators.

API declarations are reusable definitions, not automatically inherited method
errors. Test lookup and explicit narrower declarations separately from service
inheritance. This repair preserves those existing selection rules.

## Sequence and acceptance

- [x] Read the issue, DSL declaration, root/service/method validation, evaluator
  phase ordering and effective-error construction.
- [x] Add direct and actual DSL regressions for repeated exact names at all
  three scopes; cover identical/different types, unused declarations, three
  repetitions and a duplicate hidden by a narrower override.
- [x] Prove valid controls: API/service/method reuse, sibling methods/services,
  effective method override, case-distinct names and Go-normalized collisions.
- [x] Model current validation then merge, rejected post-merge and global-name
  alternatives, and local admission. Require duplicate rejection, valid scope
  reuse and unchanged winning declaration before generation.
- [x] Freeze readiness tests/model, owner inventory, exact failing/valid results
  and bounded generator acceptance for independent review.
- [x] Implement one private local-list admission owner in expr; root, service
  and method validation derive uniqueness from it. Update DSL GoDoc/public error
  guide and contributor guidance. No transport or name allocation changes.
- [x] Run direct/package validation and a bounded generation matrix: original
  mapped duplicate method (reject before generation), different-type service
  duplicate (reject instead of losing type), API duplicate (reject), valid
  cross-scope override with HTTP/JSON-RPC/gRPC consumers, and existing #734
  collision control. Valid parent/candidate/fresh-process artifacts must be
  byte-identical and compile; invalid designs must stop with scoped diagnostics
  before any generated output. Use explicit local source for candidate and
  pushed parent for baseline. No full corpus sweep or runtime transport rewrite.
- [ ] Freeze exact final diff/evidence for fresh independent review; one atomic
  commit is integrated by root, then update closure and clean inactive outputs.

## Limits

This ticket addresses declaration identity, not empty names, transport response
mapping duplicates, error object discriminator rules, or factory name collisions.
Formal models abstract validation ordering and scope identity, not every DSL
type/transport rule. Implementation tests must exercise the actual evaluator.

## Readiness evidence and final owner decision

One private helper takes the authored `[]*ErrorExpr` and its owning
`eval.Expression`, tracks exact names, and reports every occurrence after the
first as `error "name" defined twice`. Call it from `RootExpr.Validate` (API
list, diagnostic scope `design`), `ServiceExpr.Validate`, and method
`validateErrors`, before existing per-error and effective-discriminator checks.
Do not mutate or deduplicate lists. Keeping name admission separate preserves
Root's existing narrower error-field validation instead of accidentally changing
unrelated API-level type semantics.

- Thirteen invalid direct/DSL cases fail on the unchanged production owner,
  including all scopes, identical/conflicting types, unused lists, hidden service
  duplicates and three repeated direct declarations. The original nil method
  fixture was corrected to call `Prepare` before `Validate`; no panic remains in
  the acceptance evidence.
- Direct valid scope reuse passes with native primitive type selection and exact
  effective order assertions. API Boolean, service String and method Int remain
  distinct authored contracts; sibling services and methods are independent.
- A representative generated module with six methods across HTTP/JSON-RPC/gRPC
  builds and vets using compatible API/service/method ErrorResult declarations.
  Separate error titles make authored scopes distinguishable. Existing #734
  collision fixture builds/vets and its mounted wire race controls pass unchanged.
  Ledger: `/tmp/loom-699-parent-artifacts.json`.
- The four models check current admission, effective-only and global-only rejected
  alternatives, and local admission. Negative counterexamples use exit12; local
  passes all30 states. README records assumptions and implementation seams.
- Exact original downstream panic text from #699 is historical evidence; readiness
  confirms the current admission defect, without assuming every downstream failure
  remains identical after unrelated fixes.

Bounded implementation comparison uses these two valid generated modules only,
plus three invalid admission designs (mapped method, conflicting service, API).
For invalid cases, assert evaluator failure occurs before invoking generators;
for valid cases compare all generated bytes at parent/candidate/fresh repeat and
compile. Direct tests additionally protect primitive cross-scope overrides and
unused lists. No transport code changes are warranted.

The original readiness checkout also contained separately reviewed #872 runtime
work. After delivery, exact #699 readiness inputs and #872 sources were archived,
the reviewed #872 patch was reversed, and the checkout advanced to pushed
`5c6535b6d`. The #699 inputs were verified unchanged. No #872 change is part of
this ticket.

## Implementation and bounded acceptance

`validateErrorNames` now owns exact-name admission in `expr/service.go`. Root,
service and method validation consume it before existing checks. It reports each
repeat against its authored owner and does not alter a list or the effective
method merge. Public DSL documentation, the error guide and skills explain the
same-scope rejection and preserved cross-scope behavior.

A new CLI regression covers the three invalid designs through actual generation.
On the implementation parent, API/service duplicates generated successfully;
the mapped method duplicate reached a later generator error. After repair, all
three stop at `eval.RunDSL`, carry a scoped duplicate diagnostic, have no panic,
and leave no `gen/` output. This confirms early rejection, not merely a changed
late error message.

- Complete `expr`, `dsl`, `eval` race suites pass (7.291s, 1.499s, 1.300s).
- Three CLI rejection cases pass (10.101s).
- Current-parent valid scope module builds/vets (2.284s); candidate builds/vets
  (2.412s). The existing collision module builds/vets and runs its mounted HTTP
  race checks at parent (3.185s) and candidate (4.157s).
- All 42 scope-module and 15 collision-module generated files are byte-identical
  at parent, candidate and process-isolated repeat. Intended generated differences:
  none. Ledger: `/tmp/loom-699-final-artifacts.json`.
- Scoped pinned golangci-lint reports zero issues. Models are unchanged from the
  independently checked readiness packet: current/effective/global counterexamples
  and local30-state success remain applicable to this exact owner.

Final independent review and integration remain outstanding. No commit or push
has been made by the implementer.
