# Value pipeline follow-ups

These are the remaining source-level ownership and evidence gaps, rechecked
against the current code after the correctness backlog closed. They are not
claims of reproduced runtime failures. The accepted contract lives in
[internal/valuecontract/DESIGN.md](../internal/valuecontract/DESIGN.md), and the
[proof correspondence ledger](../expr/lean/value_projection/correspondence.md)
records the tested domain and historical delivery evidence.

## Remaining work

Keep source selection separate from location codecs. A raw compatibility API or
legacy helper name alone does not establish a defect or authorize its removal.
Each follow-up needs one bounded reproduction or evidence obligation before
implementation; the [execution plan](value-contract-plan.md) gives the sequence.

### R1 — JSON CLI body default projection

`cliBodyDefault` selects a payload default and canonicalizes it independently,
then `NewFlagData`/`flagDefault` turns it into the default accepted by a real JSON
body flag. This bypasses the retained body representation owner even though the
existing collection-default regressions pass.

Start with one direct table for bytes, union and mapped-key defaults, plus the
existing HTTP and JSON-RPC collection-default controls and a plain-byte default
control. Establish the target representation and preserve absent/default semantics
before selecting a repair. Keep this separate from ordinary location codecs.

### R2 — Material location and metadata example sources

HTTP request path/query/header/cookie/basic-auth, non-JSON/plain body, and gRPC
request metadata examples still sample attributes independently before emitting
CLI help/diagnostics.
Their text, byte, numeric and collection codecs are intentional; their source
selection has not migrated to the retained occurrence owner.

Verify authored root-versus-member precedence and unchanged flag spelling at one
HTTP and one gRPC shared seam, including a plain Bytes body example control.
Select one bounded transport/position follow-up
before implementation. Existing CLI renderer controls can be reused; a new
transport matrix is not required merely to establish selection ownership.

### R3 — Unconsumed example fields and sampling effects

Body/type-registry, response constructor, path initializer, WebSocket constructor
and gRPC converter example fields have no internal production reader. Their
sampling calls can still affect later samples through generator/cache state.
Exported carrier fields can also have external users.

For one carrier class, identify public compatibility requirements and compare the
later material examples with and without its sampling, including repeated and
reordered construction. Only then decide whether to remove the call or populate
the field from its retained owner. Reuse a relevant generated response/stream or
converter fixture if that follow-up changes emitted code. This is cleanup and
ordering evidence, not a demonstrated transport bug.

### R4 — Inline schema failure outcomes versus compatibility

`inlineSemanticJSONValue` and `inlineDeclaredEnumValues` can fall back to the raw
compatibility adapter after semantic resolution fails. The existing inline-schema
regression explicitly keeps an ambiguous union example unchanged, while the value
contract prohibits built-in consumers from bypassing failure outcomes.

Resolve this concrete contract discrepancy before changing public behavior:
the exported `InlineJSONSchema` API has no other production caller in this
repository. Identify which public compatibility guarantees require pass-through
and which schema-generation outcomes must honor semantic failures. Use the existing ambiguous-example control alongside
resolved bytes/union/default controls, then check the emitted example against its
schema. Preserve opaque-codec and enum projection boundaries. This audit records
the conflict; it does not authorize dropping a compatibility guarantee.

### R5 — Reference adapter boundaries

`productionGraph.capture` rejects duplicate source wire aliases and constrained
alias/key shapes that it cannot independently lower. `TestProductionConstraintGuardFailsClosed`
ensures those guards actually reject; `expr/value_alias_ownership_test.go` checks
production alias/visibility behavior. The specialized `named-key-contracts` and
`effective-alias-lengths` groups supply narrower independent comparisons.

Start with the duplicate-alias shape already exercised by the rejection control
and mapped-metadata tests: decide whether a separately scoped independent lowering
is needed for that supported graph. Retain every unsupported-shape guard until
its replacement has independent inputs and negative controls. Arbitrary Go
reflection, codecs, DSL extraction and generated compilation remain tested/external
boundaries; their exclusion is not evidence of a production bug or a mandate to
formalize all of Go.

### R7 — Retained branch identity for OpenAPI enum/default projection

OpenAPI defaults in `analyzer.go` and enum clauses in `analyzer_validation.go`
pass through `enumvalue.Normalize` into `projectOpenAPIExample` and
`normalizeOpenAPIExampleForAttribute`. Their untagged-union branches call
`matchingUntaggedOpenAPIBranch` on converted values instead of carrying the
resolved branch identity. A missing/ambiguous match returns the value without
branch-specific projection. This is a source-level duplicate interpretation
boundary; the audit has not reproduced a wrong emitted default or enum.

Use one default and one enum control whose semantically selected untagged branch
shares the declared/converted matcher input with another branch but has different
field visibility. Branch-specific visibility projection happens after matching.
Check retained identity, projected values and schema validity. Establish that
boundary before deciding the repair; preserve the intentional raw compatibility
adapter and the already migrated prepared-example path.

## Completed compiler cases

The three historical R6 compiler failures were repaired. Their direct/generated
regression anchors are retained in the
[comparison harness record](../internal/valuecontract/README.md#historical-compiler-repairs).
They are not outstanding roadmap work.
