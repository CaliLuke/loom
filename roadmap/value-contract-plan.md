# Value-contract plan

Resolve values once; transports consume shared plans. Preserve untagged bodies and runtime behavior.

M1–M2 core pushed. Migration status and delivery order:

1. **#574 — complete:** Shared schema ownership preserves async policies and matches byte schemas to decoders.
2. **#571 — complete:** Centralize effective enums, defaults, numeric/length bounds, pattern/format clauses, and required fields. Derived enums refine ancestor enums: equal/subset declarations are valid and any outside member is a design error.
3. **#456 — complete:** Reject authored map-key collisions during design validation, preserving known witnesses across opaque or cyclic siblings without invoking custom codecs.
4. **#581 — complete:** Bind separately authored request-body members to their payload sources on each endpoint's own body copy. The model, direct regressions, 45-design comparison and repository gates pass; the named-body fixture now generates, builds and vets successfully.
5. **#572 — next:** Remove private OpenAPI synthesis; validate rendered examples.
6. **#565:** Migrate transports/CLI; pass generated runtime and SSE tests.
7. **#434 → #573:** Verify protobuf round trips; remove duplicate interpretation; complete inventory and CI.

Fix shared owners, test sibling cases, update proofs. Review, delivery and cleanup follow [AGENTS.md](../AGENTS.md).

## Required execution procedure

Apply this procedure to the active migration goal before implementation of each
remaining ticket. Keep the full ticket scope and delivery order above. Root owns
architectural decisions and proof interpretation; Sol
agents at high reasoning effort execute engineering and verification.

1. Build one acceptance checklist from the ticket, accepted design, consumer
   inventory, proof correspondence ledger and repository gates. Name the evidence
   required for each obligation, including generated behavior and delivery.
2. Before further fixes, review implementation and proof coverage against that
   checklist in one bounded pass. Collect all known gaps together; distinguish
   production defects, missing coverage, outstanding gates and later-ticket work.
3. Run independent diagnostic checks to completion and collect their failures
   before repairing them as a batch. Stop a run when safety, resource exhaustion
   or a failed prerequisite makes continuing invalid; preserve completed evidence.
4. Settle the combined findings and acceptance scope before implementation. Do
   not weaken the accepted contract or waive an obligation to obtain a passing run.
   Use models to examine design changes before implementing them.
5. Freeze a candidate for expensive validation. Record which source, inputs and
   obligations each result covers. Rerun only checks invalidated by a change or
   failure; review documentation-only and test-only deltas separately when they
   cannot affect previously verified generated output.
6. Reconcile the checklist, obtain independent review of the exact final diff,
   then commit and push the ticket atomically. Report delivered tickets separately
   from implementation or validation progress.

This procedure reduces serial discovery; it does not promise that every defect
can be found upfront. A new finding must identify its concrete evidence and the
existing obligation it affects, or the explicit decision that adds an obligation.

[Design](value-contract-design.md) · [Owners](value-contract-inventory.md) · [Comparisons](../internal/valuecontract/README.md) · [Proofs](../expr/lean/value_projection/correspondence.md) · [#574 evidence](../internal/valuecontract/BYTE_SCHEMA.md)
