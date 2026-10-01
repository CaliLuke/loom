# Value-contract plan

Resolve values once; transports consume shared plans. Preserve untagged bodies and runtime behavior.

M1–M2 core pushed. Delivered changes and reconciliation status:

1. **#574 — complete:** Shared schema ownership preserves async policies and matches byte schemas to decoders.
2. **#571 — complete:** Centralize effective enums, defaults, numeric/length bounds, pattern/format clauses, and required fields. Derived enums refine ancestor enums: equal/subset declarations are valid and any outside member is a design error.
3. **#456 — complete:** Reject authored map-key collisions during design validation, preserving known witnesses across opaque or cyclic siblings without invoking custom codecs.
4. **#581 — complete:** Bind separately authored request-body members to their payload sources on each endpoint's own body copy. The model, direct regressions, 45-design comparison and repository gates pass; the named-body fixture now generates, builds and vets successfully.
5. **#572 — complete:** Built-in OpenAPI generation consumes retained shared examples; private synthesis is removed and rendered examples are validated. Option 1 retains each occurrence's examples and shares complete automatic request-body, response, parameter and header definitions only when their contents agree. Explicitly named components and schema/Go names remain protected. The 88 updated JSON/YAML fixtures cover 39 example-only cases and five approved automatic component/reference changes. The named-singleton renderer guard and importer shared-error ownership defects are repaired. The repository batch passed except for the importer failures; the full importer suite passes after repair, alongside affected suite rechecks, vet, the rendered example matrix and final full `make lint`. Final AutoK and drum-meoh checks pass; Drum-meoh loses eight unused automatic exports while retaining identical operation APIs. The full-catalog audit was stopped after 656 verified cases and is not a ticket gate. Focused acceptance evidence and independent exact-diff review support this atomic delivery. See [the evidence record](../internal/valuecontract/OPENAPI_EXAMPLES.md). The subsequent CLI deliveries and documentation reconciliation are recorded below.
6. **#565 — complete:** Route advertised HTTP/JSON-RPC JSON body examples through the client body's retained runtime plan. Preserve bytes, branch identity, explicit null and numeric precision. Propagate unavailable examples through both diagnostic constructors and individual/aggregate help without removing commands. Generated HTTP and JSON-RPC builders reproduce the old bytes failure and now round-trip the retained branch and bytes. The affected CLI, example, HTTP and JSON-RPC suites pass, including shared omission rules and existing optional-body and collection-default controls; focused lint passes. Plain/location codecs, defaults and response/WebSocket data carriers remain unchanged here; their current dispositions and remaining work are recorded in inventory R1–R3.
7. **#434 — complete:** gRPC CLI message examples project the retained service value through protobuf field and oneof allocation instead of synthesizing again. Type and payload authored examples, omitted unions, selected branches, bytes and exact Int64 values are preserved. Unusable hints produce diagnostics and disappear from help without removing commands. Focused direct, generated CLI, determinism and lint checks pass. All four existing authored-value probes fail at parent `4857a3f3` and pass with the fix; the bounded output comparison changes only CLI example/help strings. The historical nullable Any root-converter failure was subsequently reproduced and repaired under inventory R6; generated build/vet and absent/null/value converter checks now pass. Explicit-null CLI hints remain omitted under the existing policy.
8. **#573 — documentation reconciliation:** The [current inventory](value-contract-inventory.md) classifies the remaining consumers and records finite implementation and evidence follow-ups. The [correspondence ledger](../expr/lean/value_projection/correspondence.md) separates delivered gates from historical checkpoints and excluded domains. This ticket changes no production code and does not declare the whole migration complete.

## Remaining work

Use the inventory's [remaining-work list](value-contract-inventory.md#remaining-work)
to select one bounded follow-up at a time. Each entry distinguishes a source-level
ownership gap, missing evidence, or a recorded compiler limitation and names the
smallest check needed before implementation. Intentional compatibility APIs and
location codecs stay protected. A legacy helper call alone does not justify a
replacement.

Production repairs, expansion of independent reference adapters, and validation
of historical compiler failures require their own scoped tickets. No full-corpus
audit, blanket consumer rewrite, or repeat of every completed proof gate is a
prerequisite for closing the #573 documentation reconciliation.

Fix shared owners, test sibling cases, update proofs. Review, delivery and cleanup follow [AGENTS.md](../AGENTS.md).

## Required execution procedure

Apply this procedure before implementation of a selected production follow-up.
Follow that ticket's scope and dependencies; the inventory does not authorize
executing every follow-up together. Root owns
architectural decisions and proof interpretation; Sol
agents at high reasoning effort execute engineering and verification.

1. Build one bounded acceptance checklist from the ticket, accepted design,
   consumer inventory and proof correspondence ledger. Select relevant repository
   gates and representative cases, naming the distinct obligation each proves.
   Use direct regressions, affected packages, generated behavior and delivery
   checks. The full exported-design corpus is an occasional audit after major
   changes, never a ticket acceptance, review or commit requirement.
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
   cannot affect previously verified generated output. Reuse valid parent
   evidence. Repeat generation where determinism is required; do not repeat
   build/vet for identical output under identical source, toolchain and dependency
   inputs. A separate broad audit needs an explicit purpose and cost estimate;
   do not silently promote it into the ticket's critical path.
6. Reconcile the checklist, obtain independent review of the exact final diff,
   then commit and push the ticket atomically. Report delivered tickets separately
   from implementation or validation progress.

This procedure reduces serial discovery; it does not promise that every defect
can be found upfront. A new finding must identify its concrete evidence and the
existing obligation it affects, or the explicit decision that adds an obligation.

[Design](value-contract-design.md) · [Owners](value-contract-inventory.md) · [Comparisons](../internal/valuecontract/README.md) · [Proofs](../expr/lean/value_projection/correspondence.md) · [#574 evidence](../internal/valuecontract/BYTE_SCHEMA.md)
