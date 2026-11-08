# Status selector proofs (#976)

Run `./check.sh` with Lean 4.35.0 (`lean-toolchain`). The script compiles
`StatusSelection.lean` with warnings as errors, requires every theorem in
`required-theorems.txt`, and rejects any axiom other than `propext`,
`Quot.sound` and `Classical.choice` (so `sorryAx` and `Lean.ofReduceBool` fail).
It uses only Lean core; no Lake build or `.lake` output is produced.

Recorded 2026-11-05 at source `f0c7a2b8c`: all eight theorems check; six use
`propext` and two also use `Quot.sound`.

| Theorem | Statement (unbounded in keys and codes) |
| --- | --- |
| `select_membership_invariant`, `select_perm_invariant` | final selection depends only on key membership, so any declaration order or permutation selects the same key |
| `select_covers` | a selected key is declared and covers the status |
| `select_most_specific` | no declared covering key is more specific than the selected one |
| `domain_iff_select` | for a declared key, the server runtime-domain formula (exact; class minus exact keys; fallback minus exact and class keys; final statuses only) holds exactly when selection returns that key |
| `phases_disjoint` | no status is selected by both the final and the interim selector |
| `switching_protocols_never_interim` | 101 selects neither an interim nor a final key |
| `declaration_order_counterexample` | first-cover in declaration order selects `default` for 200 in `[default, 2XX]`; the design selects `2XX` |

Limits: keys are modeled as values without well-formedness (class digit range,
three-digit codes) because the theorems hold without it; bodiless-status
exclusion, StatusField constraints, ownership, media and the HTTP runtime are
not modeled here. The theorems establish properties of these Lean functions,
not of Go code. Go correspondence: `TestSelectFinalPrecedence`,
`TestSelectInformational` and `TestStatusSelectorMatchesExpr` check the shared
selector and its runtime copy; `TestHTTPStatusRuntimeDomain` checks the domain
formula; `TestFinalStatusSelectionRendering` and the generated
`TestFinalStatusRoundTrip` check decoders. These do not assert a new proof run. The bounded TLC
models in `../../tla/status_selection` cover ownership, decoding and phases.
